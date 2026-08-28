# ADR-0056 — Analytic-face reconstruction from the mesh-arrangement boolean

**Status:** Accepted — building on `feat/analytic-boolean-reconstruction`. · **Scopes**
[Oblikovati#2153](https://github.com/Oblikovati/Oblikovati/issues/2153) (the arrangement emits faceted
bodies) and closes [Oblikovati#2167](https://github.com/Oblikovati/Oblikovati/issues/2167) (visible
seam between two cocylindrical extrude walls). · **Builds on**
[ADR-0052](ADR-0052-planar-boolean-corefined-seam-classification.md) (the exact mesh-arrangement
boolean core `kernel/meshbool`) and [ADR-0043](ADR-0043-generalized-provenance-naming.md) (edge/vertex
provenance). · **Touches:** `kernel/meshbool/*`, `kernel/ops/meshbool_body.go`,
`kernel/ops/meshbool_soup.go`, a new `kernel/ops/reconstruct_*.go`.

## Context

The ADR-0052 core (`kernel/meshbool`) is a complete, exact, watertight mesh-arrangement boolean: it
co-refines the two operands on exact predicates, classifies by exact ray-cast, and emits a closed
2-manifold triangle soup with exact volume. It is wired as a robustness **fallback** in
`ops.booleanGeneral`, never the default.

It cannot become the default because the OUT adapter (`soupToBody` → `meshbool.MergeFaces`) rebuilds
the body by fitting **one `geom.Plane` per coplanar facet region**. Every curved boolean output is
therefore faceted: a crossing-cylinder union becomes ~108 planar facets instead of 7 analytic faces.
Defaulting the mesh engine would faceted-ize every curved boolean, breaking:

- **fillet/chamfer on a boolean edge** (needs an analytic edge curve, not a facet chain),
- **exact mass properties** (a facet body under-reports curved volume),
- **STEP/native export** (surfaces, not facets),
- **chained booleans and TNP provenance** (a facet fan has no face identity),
- and it produces the visible **#2167 seam**: two cocylindrical walls faceted **independently** leave
  mismatched facet grids at their z-interface. Each wall is smooth; the boundary between two
  independently-faceted copies of the *same* cylinder is not.

## Key insight — provenance, not fitting

Co-refinement (`kernel/meshbool/mesh.go` `refineAgainst` → `RefineFace`) only ever **splits** an input
facet against the other operand's crossing segments. It never fuses two input facets into one output
triangle, and it never creates a triangle off an original surface. **Every output triangle descends
from exactly one input face of one operand, whose exact analytic `geom.Surface` is already known.**

So reconstruction must NOT fit surfaces from a facet point cloud (fragile, tolerance-bound, and unable
to recover the exact radius/axis). It must **carry the originating surface as an exact provenance tag**
through the arrangement and group the output by tag. The face's surface is then the *original* surface,
bit-for-bit — no fitting, no error. This also restores ADR-0043 face identity for free (a reconstructed
face knows it came from operand A's face 3).

## Decision

Thread an exact per-triangle provenance tag through `kernel/meshbool`, then reconstruct the body from
tag-grouped faces on their original analytic surfaces.

### Layer 1 — provenance tagging (foundation)

- `meshbool` gains a `TaggedSoup{Tris [][3]Point; Tags []int}` and `BooleanTagged(a, b TaggedSoup, op)`.
  A tag is a caller-defined surface id; `refineAgainst` copies the parent face's tag onto every child,
  `CoRefine`/`Boolean` propagate it through keep/coplanar/orient. The untagged `Boolean` stays for
  callers that do not care (it delegates with a single-tag soup).
- The IN adapter (`ops/meshbool_soup.go`) builds the tagged soup from `tessellateBodyFaces`, which
  already meshes **one `*Mesh` per face**: operand A's faces get ids `0..na-1`, operand B's `na..na+nb-1`,
  and a side table maps each id → `(operand, *topo.Face, geom.Surface)`.

### Layer 2 — tag-grouped analytic faces

- Group the result soup by tag; each group is a 2-manifold patch on one original surface. Trace its
  boundary loops (the existing `boundaryEdges`/`traceLoops`, keyed on tag not coplanarity) and build a
  `topo.Face` with the **exact original `geom.Surface`** + traced loops.
- Adjacent groups on the *same* surface (cocylindrical walls, #2167) merge into one face.

### Layer 3 — analytic edge curves

Each boundary loop is a polyline of arrangement vertices. Reconstruct analytic edges:

- a run of vertices lying on an **original operand edge** reuses that edge's exact `geom.Curve`
  (circle, line, arc, ...) — identified by provenance (both endpoints original, from that edge);
- a **new intersection run** (operand A's surface ∩ operand B's surface) becomes the true
  surface-surface intersection curve (Layer 4), interim a faithful spline through the arrangement
  vertices that lie on both surfaces.

### Layer 4 — exact surface-surface intersection (OCCT parity)

Per surface-pair family produce the exact `geom.Curve`: plane∩plane = line; plane∩cylinder = line-pair
/ ellipse; sphere∩plane = circle; cylinder∩cylinder, cone∩plane, torus∩plane, etc. General/untractable
pairs keep the faithful spline. This is what raises boolean-edge geometry to OCCT parity.

### Layer 5 — cutover

Shadow-validate the reconstructed body against the existing analytic boolean AND the OCCT oracle
(`boolean_occ_oracle_test.go`): strict-superset gate — valid, closed, exact volume/area, and analytic
face/edge counts matching the analytic engine. Flip `ops.booleanGeneral` to prefer the reconstructed
mesh engine only when green on the full `kernel/brep` + `kernel/ops` corpus. Flip-back is the valve.

## Consequences

- The mesh engine can finally become the **default** boolean (ADR-0052 Phase 2 unblocks): watertight by
  construction AND analytic-faced, so downstream fillet/mass-props/export keep working.
- **Provenance is exact, not fitted.** The reconstructed surface is the original surface; the only
  approximation is the trim boundary where a general SSI curve stays a spline (Layer 4 removes it for
  the tractable families).
- Corpus safety is the gating invariant, as in ADR-0052: no cutover until the reconstructed engine is a
  strict superset of the analytic engine on the full boolean corpus and the OCCT oracle.

## Status (findings from the first build)

Layers 1–2c and the (gated-off) recovery hook are landed and green. Reconstruction is
proven on gluing unions (a stepped coaxial shaft → two analytic walls, exact volume),
planar booleans (box union, exact volume), and a curved-bar ∪ tool. It is **not yet the
default** — `reconstructionCutover = false` — because two frontiers remain:

1. **Cocylindrical cap-on-wall (#2167) — CLOSED at the kernel level.** Three cooperating
   fixes take it from a visible faceted seam to a valid analytic solid:
   - **Conforming tessellation** (`kernel/geom/canonical_sampling.go`): a circle/arc
     discretizes to the SAME points whenever it *is* the same circle — canonical
     absolute-angle sampling from (centre, axis line, radius), independent of RefDir, normal
     sign, or edge identity. Conformance is a property of the geometry, not of intra-body
     pointer sharing. Same power-of-two segment count as the adaptive path (density/volume
     error unchanged); closed circles seam-anchored to stay monotone. Applied to the BOOLEAN
     INPUT ONLY — `applyBooleanConformance` (`kernel/ops/meshbool_imprint.go`) installs it as
     a temporary snapped polyline on each operand's circle/arc edges (consulted first by
     `discretizeEdge`/`tessellateEdgeWithParams`) and restores it, so DISPLAY meshing keeps
     the adaptive bisection. The global variant closed #2167 too but folded an `occtparity`
     display mesh (`complex/F2`) and drifted golden facet counts; unifying on one global
     canonical discretization is deferred to #2168. This removes the rim-sliver membrane.
   - **Cross-operand vertex-on-edge imprint** (`kernel/ops/meshbool_imprint.go`): the mesh
     co-refinement runs per-operand and misses a VERTEX of one operand on an EDGE of the
     other. A D-profile chord corner sits on the cylinder rim at a non-canonical angle —
     OUTSIDE the inscribed rim polygon — so it cannot be recovered by splitting a soup edge.
     Before tessellating each operand for the boolean, imprint the other's on-edge vertices
     into the edge's discretization (installed as the edge's snapped polyline, both faces
     share it, restored after). Restricted to full-circle rims, where the faceting gap
     exists and `matchSubArc` can rebuild the sub-arcs; straight/other edges are exact
     already or a follow-up.
   - **Same-surface merge** (`meshbool_reconstruct_merge.go`): relabel coincident-surface
     tags before the arrangement trace, so the false seam between the lower cylinder wall and
     the upper cocylindrical arc wall becomes interior — the two rebuild as ONE analytic
     cylinder (the correct B-rep, as Inventor/OCCT merge cocylindrical faces), with the
     exposed cap trimmed to the minor segment via `matchSubArc`.

   `reconstructBoolean(cylinder, D-prism, Union)` now yields a closed, manifold, solid B-rep
   of five analytic faces at the exact stacked volume — asserted by the (now un-skipped)
   `ops.TestReconstructCocylindricalCapOnWall`.

2. **Layer-5 cutover — LANDED.** Reconstruction is wired as the LAST path in
   `curvedExactBoolean` (`reconstructedCurvedBoolean`, gated by `reconstructionCutover = true`),
   so it fires only where every analytic recognizer declined, and BOTH the direct `ops.Boolean`
   path and the feature `combine` path (which tries `CurvedBoolean` on the still-analytic
   operands before faceting) pick it up. It self-validates: a valid closed solid whose volume is
   inside the Requicha bracket, else it declines and the operation falls back to faceting — so it
   is never adopted wrong. The cutover is now the active default (`reconstructionCutover = true`,
   kept as an alpha kill-switch) with all three robustness layers below CLOSED. Codified by
   `ops.TestReconstructionCutoverShadow` (correct where it fires — D-join, coaxial cylinders, box
   union all match the analytic volume; declines safely on crossing cylinders), the
   `curved_operand_boolean_test.go` trio (a filleted bar ∪/−/∩ box now KEEPS its fillet cylinder
   and holds the exact analytic volume instead of faceting), and the feature-level
   `TestPistonHeadCocylindricalJoinKeepsAnalyticWalls` (#2167). The old `recoverFacetedCurvedJoin`
   recovery in `booleanGeneral` is subsumed and removed.

3. **Reconstruction robustness on extrude-built operands — the remaining blocker for the
   FEATURE-level #2167.** The kernel fixture reconstructs to a valid solid; the EXTRUDE-built
   operands hit two further layers, one now closed:
   - **CLOSED — near-degenerate cross-operand rim slivers.** The exact mesh boolean welds each
     operand internally but never the two to EACH OTHER, so where they share a boundary reached by
     two sampling paths (a canonical cap edge vs a cylinder surface evaluation) it left one vertex
     a sub-tolerance step from the other — a near-degenerate sliver that fragments the arrangement,
     and the reason the interface cap was not dropped. `weldResultSoup` (`meshbool_reconstruct.go`)
     welds the combined output with the size-relative tolerance (the tolerance lives in the ops
     layer; the meshbool core stays exact). With it, hand-cyl ∪ extrude-D reconstructs to a valid
     5-face solid.
   - **CLOSED — seam-wrap edge match (real cause: planar-cap pass-through).** For a fully
     extrude-built join the result was manifold with the correct faces but not CLOSED — open edges
     on the top. The cause was NOT a curved seam-wrap but the PLANAR cap pass-through: a face was
     passed through untouched when its result facet count equalled its input count ("kept whole"),
     and `weldResultSoup` broke that signal for planar faces — it drops the co-refinement's
     degenerate slivers, so the cylinder's top cap, cut to a minor segment by the D-prism, collapsed
     back to the full-disc facet count and passed through as the UNTRIMMED disc. Fix
     (`reconstructFace`, commit 690010ef): only a periodic CURVED wall is ever passed through; a
     planar face is always rebuilt from its arrangement boundary, where the trim is explicit. The
     extrude-built join now reconstructs to a valid, closed, manifold 5-face analytic solid
     (verified: Layer-5 gate on → the #2167 feature test PASSES; the fix reverted, gate on → FAILS).
   - **CLOSED — disjoint-region / oblique-conic cut (real cause: unweldable ellipse SSI).** A
     tool cylinder whose material intersection is two disjoint pieces separated by a void
     reconstructed to a valid-but-WRONG volume that slips the Requicha bracket (a cut's result can
     sit anywhere in `[tv-wv, tv]`, so an under-removal is inside it). The cause was NOT
     disjointness — a perpendicular cylinder crossing a slot void reconstructs correctly (circular
     exits). It was the OBLIQUE CONIC SSI: where the bore exits a slanted face, `intersectionRunEdge`
     synthesized a `geom.EllipseFull` (cylinder∩tilted plane). The ellipse is exact as a curve but
     each incident face samples it independently, so the two copies do not weld and the closed solid
     encloses the wrong region (SlottedScrew's cross-hole removed ~5 mm³ too little). Isolated by
     provenance, not thresholds: across the corpus the faithful reconstructions synthesize only
     `geom.Line` and `geom.Circle` SSI (exact, identical from both faces → weld); only the failing
     ones synthesize `geom.EllipseFull`. Fix (`weldableSSICurve`, commit 01ed8196): confine SSI
     reuse to lines and circles; an oblique conic DECLINES, so the caller falls back to the exact
     faceted boolean — no wrong geometry, no regression, until the numeric-SSI weld layer (Layer 4)
     lands. Regressions: `ops.TestReconstructDeclinesObliqueConicCut` (oblique declines) and
     `TestReconstructPerpendicularDisjointCutRebuilds` (perpendicular disjoint rebuilds, 2 bore
     walls, exact volume).
   - **READY — the Layer-5 cutover has no remaining correctness blocker.** With the guard, a full
     corpus run at `reconstructionCutover=true` leaves ONLY `csg_fallback_test`'s Join/Cut failing —
     and those assert the OLD faceted fallback where reconstruction now yields the more accurate
     analytic result (kept fillet cylinder, exact volume). SlottedScrew and the #2167 piston both
     pass. Flipping the gate is now a test-premise update, not a correctness fix: set
     `reconstructionCutover=true`, rewrite `csg_fallback_test`'s Join/Cut to assert the analytic
     result, and drop the piston test's gate-aware skip. The gate stays off in this commit pending
     that decision.

4. **Curved SSI-edge welding — LANDED for the ellipse (Layer 4).** A clean oblique plane∩cylinder
   cut (a cylinder bored through a box at an angle, cyl ∪/− box) now reconstructs analytically. Two
   findings closed it, and neither was the welding premise this ADR originally assumed:
   - The ellipse ALREADY welds. `geom.IntersectSurfacesAnalytic` canonicalises the plane∩cylinder
     ellipse (the plane is always the first argument), so both incident faces synthesize the
     bit-identical `EllipseFull`, and the endpoints+midpoint `edgeKey` fuses their copies into one
     `topo.Edge`. `weldableSSICurve` simply widens to admit `EllipseFull`/`EllipticalArc`.
   - The real defect was ARC-EXTENT, not welding. A closed conic restricted to two endpoints is
     ambiguous between the direct sub-arc and its wrapping complement; the raw `[t0,t1]` stored the
     wrong half (SlottedScrew's slanted bore removed ~5 mm³ too little). `orientRunEdge` now picks
     the arc the run's INTERIOR vertex lies on (the analogue of `matchSubArc`'s three-point arc).
   `reconstructBoolean` self-validates Euler-admissibility, so the one case still left faceted — an
   oblique exit that crosses an operand CORNER, splitting its ellipse across two faces into an
   inadmissible χ — declines safely. Cone∩plane parabola/hyperbola remain future work.

## Alternatives considered and rejected

- **Fit surfaces from the facet cloud (RANSAC / least-squares).** Rejected: tolerance-bound, cannot
  recover the exact radius/axis, and needless — the exact surface is already known via provenance.
- **Re-attribute output facets to surfaces by geometric on-surface test.** Workable but tolerance-based
  for curved surfaces (facet vertices are chord points, not on the true surface); exact tag propagation
  is strictly better and also feeds TNP identity.
- **Keep the mesh engine faceted and only special-case #2167.** Rejected: leaves every other curved
  boolean faceted; #2167 is a symptom of the general faceting, not a special case.
