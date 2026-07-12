# B1 — Closed-seam edge orientation in the fillet weld (design)

**Status:** approved 2026-07-12. Branch `feat/occt-blend-parity-corpus`.
**Scope:** the B1 sub-cluster of the inconsistent-orientation investigation
(`docs/superpowers/specs/2026-07-12-inconsistent-orientation-rootcause.md`): 13 corpus
cases `simple/{R9,S1,S3,S4,S6,S7,S9,T1,T3,T4,T9,X3,Y1}` rejected by the fillet as
"inconsistent orientation at edge N". **Out of scope:** B2 (fillet-face winding, K/L
cases) and the 5 STEP-import-defect + 1 IsSolid cases — each its own increment.

## Problem

The fillet rebuild re-welds surviving B-rep faces into a new body via `assembleBody`
(`kernel/ops/assemble_curved.go:48`). For each directed loop segment `a→b` it calls
`edgeCatalog.use(a, b, curve, srcE)` (`:189`), which mints one shared edge per undirected
welded vertex-pair on first request and, on the second face's request, returns it reversed:

```go
return topo.Use{Edge: rec.edge, Reversed: rec.from != a}   // :192
```

This is exact for an **open** edge (`a != b`): the welded vertex order encodes the traversal
sense. It **fails for a closed seam edge** — a full-circle edge whose start-vertex ==
end-vertex, so both endpoints weld to the *same* index `a`. Then `rec.from != a` is `false`
for **both** uses; both get `Reversed=false`; `ops.Validate` (`validate.go:47`) requires the
two uses of every manifold edge to have *opposite* `Reversed`, so the body is rejected.

**Confirmed empirically (Case A, unanimous).** A telemetry probe measured the 3D arc length
of every `start==end` edge on all 13 B1 imported bodies: **every one is a real full circle**
(arc length = 2πR, clean integer radii 8, 20, 25, 30, 35 …), **zero** zero-length edges. The
seams arrive from STEP as `geom.NewArc3d(center, normal, ref, radius, 0, 2π)`
(`kernel/exchange/step/topomap/build_edge.go:96-118`) — legitimate closed edges, not
degeneracies. **Poles are a different animal and are NOT in scope:** a sphere pole / cone
apex arrives as a STEP `VERTEX_LOOP` and is *dropped* on import (`topomap/loop.go:12-63`,
`face.go:85-88`) — it never becomes an edge. So B1 is exclusively **real-arc closed circles**.

The imported body passes `Validate` because STEP sets the two seam reversed-flags *explicitly*
(from `ORIENTED_EDGE.orientation`). The fillet weld **re-derives** them from vertex order, and
that is where the invariant is lost.

## Root cause (one line)

`edgeCatalog.use` recovers coedge sense from the welded *start-vertex index*
(`rec.from != a`). A closed edge welds both endpoints to one index, erasing that datum — the
welder is topologically blind to a single-vertex loop.

## Solution — Method A: topological flip, gated on geometric closure

Chosen over Method B (geometric derivation of sense from surface `u/v` + tangent) and
Method C (split the circle at its far point into two half-edges). Rationale below.

### The change (single choke point)

In `edgeCatalog.use` (`assemble_curved.go:189-200`), when a segment is `a == b` **and** its
curve proves geometric closure, mark the minted `edgeRec` as closed. On the **second** use of
that `seamEdgeKey`, a closed record returns `Reversed: true`; open records keep
`rec.from != a` unchanged.

```
use(a, b, curve, srcE):
  key := seamEdgeKey{canon2(a,b), edgeClassOf(a,b,srcE,classes)}   // UNCHANGED — per identity class
  if rec, ok := edges[key]; ok:
      if rec.closed:  return Use{rec.edge, Reversed: true}          // closed seam: 2nd coedge is antiparallel
      return Use{rec.edge, Reversed: rec.from != a}                 // open edge: UNCHANGED
  ... create edge e ...
  edges[key] = edgeRec{edge: e, from: a, to: b, closed: isClosedSeam(a, b, curve, weld)}
  return Use{e, Reversed: false}
```

### The closure gate (why Option 1, not index-only)

`isClosedSeam(a, b, curve, weld)` returns true iff **`a == b`** AND the curve returns to its
start over its full domain:

```
a == b  &&  curve != nil  &&  ‖curve.PointAt(lo) − curve.PointAt(hi)‖ < weld
```

where `[lo,hi] = curve.Domain()` and `weld` is the model-relative weld tolerance already
computed in `assembleBody` (`ResolutionForPoints(pts).Weld()`, `geom/resolution.go:122` —
`weldCoef·epsRel·size`, ADR-0042). This is threaded into `edgeCatalog` as a new field.

Index equality alone (`a == b`) is insufficient: an upstream point-welder tolerance defect can
collapse a *micro-arc* to one vertex index, and an index-only flip would launder that
geometric micro-crack into a `Validate`-clean topological ghost that crashes a later boolean.
Requiring the curve to prove its own closure means an `a==b` edge that *fails* the norm is left
with both uses `Reversed=false` → `Validate` rejects it **loud and early** — the fail-fast we
want for a genuine weld defect. A straight-line closed segment (`curve == nil`) never passes
(it is a true zero-length degeneracy) and is correctly left to fail.

Note: the gate is domain-agnostic (norm of endpoint closure), **not** a `span≈2π` test — so a
future closed periodic B-spline normalized to `[0,1]` would also qualify, without a circle-
specific constant.

### Why parity is correct, unconditionally

`Validate` (`validate.go:47`) is a pure combinatorial-orientability check —
`uses[0].Reversed() != uses[1].Reversed()`, never consulting `Face.reversed`. First-store-
`false` / second-`true` satisfies it for **both** closed-edge topologies: (1) a periodic face
using its seam twice, and (2) a closed edge shared once by each of two faces. Both present to
`use()` as "same key requested twice," so one rule covers both.

### Why the flip is tessellation-safe (the load-bearing proof — mesh correctness is priority #1)

`Validate` passing is *necessary but not sufficient*; a wrong seam orientation must not corrupt
the mesh. It cannot, on the analytic path, because the seam use flag is **read then discarded**:

```
edge_discretize.go:50   loopBoundary reverses the sampled ring when u.Reversed()
  → tessellate_trim.go:388-414  periodicBandGrid maps ring pts via ParamAt, keeps only
                                bracketPeriod(uu) + sortUnique(vv)      ← ORDER-INDEPENDENT
  → closed_surface_mesh.go:18   closedDomainMesh REBUILDS the mesh from the surface (u,v) domain
  → emitCellOutward / emitClosedTri   wind every triangle to the SURFACE NORMAL, not the loop
  → tessellate.go:145           outward sense applied from Face.Reversed() ALONE, post-mesh
```

Reversing a closed seam edge's ring order only permutes `(u,v)` samples that are immediately
`sortUnique`-d away — it cannot move a vertex, drop a cell, or flip a normal. **Mass properties
are immune too:** `consistentOutwardFlips` 2-colors the *triangle* mesh, never reading
`EdgeUse.Reversed` (`massprops.go`). (Geometry-math consult, 2026-07-12.)

### The one escape hatch (B-spline seams → Method B)

The *only* consumer that reads the seam use flag **order-sensitively** is the NURBS pcurve
mesher: `concatLoopPcurve` reverses the pcurve when `u.Reversed()`
(`nurbs_pcurve_mesh.go:140-147`) and feeds a CDT where order matters. It fires only for
B-spline faces with populated pcurves — nil on the analytic weld (`topology.go:171-178`), so it
is out of scope today. But some B1 cases (S/T grid) contain B-spline faces. **Contingency:** if
any B-spline B1 case *regresses* under the flip, isolate it and derive that use's sense
geometrically (Method B: at the seam, sense = sign of `(T × d_into)·N` with `Face.reversed`
folded into `N`) instead of the topological flip. The corpus scoreboard is the gate that
surfaces this; we do not pre-build Method B.

## Guards / pitfalls (all from the geometry-math consult)

1. **Key on `seamEdgeKey`, never the raw `(a,b)` pair** — keeps the flip orthogonal to the
   #1600 tangent-seam splitter. A genuine periodic seam is one source-edge id →
   `edgeClassOf→0` → not split (`assemble_curved.go:160`); two coincident circles get two keys
   and each flips independently. The `closed` decision lives on the *keyed* `edgeRec`.
2. **Fail-fast on spurious weld** — the closure norm (above) turns an `a==b` that is *not* a
   real circle into a loud `Validate` rejection, not a silent pass.
3. **Faces with opposite `Face.reversed`** (a Difference cut wall meeting a normal face at a
   closed edge) are irrelevant to Method A — parity is on the two edge-use flags only; normals
   are applied later from `Face.reversed`.
4. **Do not touch `curvedSolid`/count logic** — it already counts a closed edge to 2 via its
   `seamEdgeKey` (`:114`); the fix changes only the *sense* returned, never edge creation.

## Tests

**Unit (hand-built, `kernel/ops`):** two closed-seam bodies assembled through
`assembleBody`, asserting the result passes `Validate` (opposite parity on the seam edge) and
tessellates to a positive, correct area/volume:
- **T-A full-cylinder side** — one periodic face using its seam edge twice (exercises
  `periodicBandGrid` → `closedDomainMesh`).
- **T-B equator shared by two caps** — a closed edge used once by each of two faces (exercises
  the sphere-cap path).
Plus a **fail-fast unit**: an `a==b` edge whose curve does *not* close (endpoints apart >
weld) must leave both uses `Reversed=false` and be rejected by `Validate` (the closure gate).
Plus an **open-edge no-op unit**: an ordinary `a→b` edge still orients via `rec.from != a`.

**Corpus (regression gate):** promote the 13 B1 cases; the scoreboard must green all 13
(`TestOCCTBlendScoreboard`) with the baseline **PASS=31 unchanged** (no regression to any
currently-passing case, trihedral corpus unmoved).

**Live (final gate, CLAUDE.md Live-tests):** render one filleted body with a surviving periodic
seam via the MCP bridge + screenshot API; confirm **no inside-out band, no missing-wedge or
doubled column at u=0, no z-fighting** along the seam ruling — the exact failure tells the
consult named.

## Files

- **Modify:** `kernel/ops/assemble_curved.go` — add `weld float64` + `closed bool` fields to
  `edgeCatalog`/`edgeRec`; add `isClosedSeam`; branch in `use()`; pass the weld tol from
  `assembleBody`. (Keep the file < 500 lines; extract `isClosedSeam` as its own 4-8 line func.)
- **Add:** `kernel/ops/assemble_curved_seam_test.go` — the four unit tests above.
- **Modify:** `model/feature/occtparity/` — promote the 13 B1 cases into the passing gate
  (mirrors the G5 V5/V1 promotion pattern in `fillet_g5_runout_test.go`).
- **Modify:** `docs/superpowers/specs/2026-07-12-inconsistent-orientation-rootcause.md` +
  the SDD ledger — correct the stale "zero-length self-loop" wording to "real-arc closed
  circle (Case A confirmed)".

## Non-goals

B2 fillet-face winding (K/L); the 5 STEP-import-defect cases; the D5 IsSolid one-off; poles
(already correctly dropped on import); Method B (built only if the escape hatch triggers). No
PR until the whole corpus is green — this increment greens 13 and holds the line at 31+13.
