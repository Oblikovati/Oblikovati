# ADR-0049 — Partial curved-on-planar boolean via a `planeUV` operand

**Status:** Proposed (2026-07-04) — design for
[Oblikovati#1591](https://github.com/Oblikovati/Oblikovati/issues/1591), the partial
curved-on-planar contact deliberately deferred as a new feature by
[ADR-0045](ADR-0045-curved-boolean-kind-taxonomy.md) §Consequences. · **Builds on**
[ADR-0045](ADR-0045-curved-boolean-kind-taxonomy.md) (the T/P/D KIND taxonomy — this adds the
partial case under the curved-on-planar `[P]` KIND), [ADR-0046](ADR-0046-curved-boolean-cap-crossing.md)
(the shared `(u,v)` arrangement + OCCT-certification protocol), [ADR-0048](ADR-0048-corner-junction-coupled-overlay.md)
(the shared-exact-point weld discipline), [ADR-0042](ADR-0042-model-scale-and-relative-tolerances.md)
(model-relative tolerances) and [ADR-0043](ADR-0043-generalized-provenance-naming.md) (curved-stitch
provenance). · **Touches (when built):** new `kernel/brep/curved_plane_uv*.go` +
`curved_plane_partial_{drill,boss}.go`; two **additive** widenings of the shared
`kernel/brep/curved_halfspace_uv_arrangement.go` + `curved_halfspace_uv_side.go`; the interior-path
goldens; the OCCT oracle harness (`experiments/occ-boolean-oracle`, `kernel/ops/*_certification_test.go`).

## Context

The curved-on-planar KIND (a cylinder/cone tool crossing a **planar** face) is exact in one sub-case
only: the imprint conic (circle for a perpendicular axis, ellipse for oblique) lies **strictly inside
one face**, where the trim is a trivial "add the conic as an inner loop" (`drill.go`/`drill_multi.go`
`capWithHole`, `curved_cylinder_boss.go` `JoinCylindricalBoss`). The gates
(`circleVsCap`→`circleInsideFace`, `bossSeatFace`) return `ok=false` the moment the conic **clips a
face boundary**, dropping the operation to triangle-soup CSG (`CodeBooleanCSGFallback`, #1407):

- a drilled hole whose entry circle runs off a plate edge;
- a boss/spigot whose base ellipse straddles or overhangs the seat-face edge.

ADR-0045 asserted a `planeUV` operand was a poor fit — the shared `(u,v)` trimmer's vocabulary
(`placeSeams`, `wrapsAllU`, `vPeriodic`, rim/seam segment tags) is steeped in **periodicity**, and a
bounded, non-periodic, polygon-framed plane has no seam, no rims and no wrap. This ADR re-opens that
for the **partial** case (only) now that a code-grounded audit quantifies the mismatch as **two
additive core touches**, not a fatal leak — and now that `cutCylinderUV` has already proven the
`uvSide` seam tolerates a non-rim frame (it substitutes a prior-trim loop for the rim circles).

## Decision

**Implement a `planeUV` operand that satisfies the existing `uvSide` interface, route partial
curved-on-planar contacts through the one shared `trimByImprint` pipeline, and inject EXACT
conic∩polygon crossings into the sampled arrangement.** Three load-bearing sub-decisions:

### D-a — `planeUV` implements `uvSide` (not a parallel planar pipeline)

The plane's `(u,v) = to2D(plane, ·)` arrangement *is* the planar boolean's own `Arrange`. `planeUV`
implements `uvSide` with the periodic concepts made **inert**: `placeSeams` a no-op, `vPeriodic` /
`uPeriodic` / `wrapsAllU` false, `wrappingSolidFaces` `(nil,false)`, and the **face polygon**
(the seat face's outer + hole loops) tagged `segPolygon` as the arrangement frame in place of the
ruled rim+seam frame. The DCEL walk (`chainLoops`/`nextByAngle`), cell classifier
(`keptCells`/`interiorOfCell` pole-of-inaccessibility), loop grouping (`groupLoopFaces`) and arc
re-emission (`emitImprintRun`) are reused **verbatim** — they are already surface-agnostic and, being
pure planar combinatorics, are better suited to a plane than to a periodic band.

### D-b — Keep the strictly-interior fast-path; add the partial path beside it (do NOT unify)

The interior drill/boss handlers are fast because they **skip** imprint intersection + arrangement.
Gate on the **existing** classification: `clean` (strictly interior) → the unchanged interior path,
provably byte-identical; `pierced && !clean` (partial) → the new `planeUV` path, tried **before**
returning `ok=false` to CSG. Only genuinely-partial booleans pay the arrangement cost — and those
otherwise pay far more as CSG triangle soup.

### D-c — The face polygon replaces rim+seam framing; two additive core widenings

The ghost-periodic audit (below) found exactly **one** genuine leak in the shared core plus one
tag-collision risk. Fix both additively, so every existing `uvSide` implementer stays byte-identical:

1. **`segPolygon`** — a new `segKind` value. Its `sameRun` compares **curve identity** (like
   `segImprint`), not v-level, so two distinct polygon edges at the same `v` do not wrongly merge
   (the `segRim` constant-v assumption).
2. **`uPeriodic`** — a new `bool` on `seamWelder` and a new `uvSide.uPeriodic()` method (ruled/torus
   return `true`, plane `false`). The welder's currently-**unconditional** u-fold (folds any vertex at
   `u≈2π` to `u=0`) is gated on it — the exact mirror of the v-fold already gated on `vPeriodic`.

### D-d — Crossings EXACT, arrangement sampled (the numerics strategy)

Keep the 256-sample imprint polyline seeding the arrangement's **non-critical interior**, but replace
every topologically-decisive **conic∩polygon-edge crossing** with the exact algebraic root, injected
as a split vertex carrying its exact conic parameter — the same `clipEndToBand`/`refineCurveV`
rim-snap discipline the ruled side already uses, applied at polygon edges instead of rims. This is
what averts the blind-straddle failure mode (a *sampled* crossing that two adjacent faces compute
differently, leaving sliver free-edges).

- **Circle ∩ edge:** the stable quadratic `qq = −½(b + sign(b)·√Δ)`, roots `t₁=qq/a`, `t₂=c₂/qq`
  (Kahan; kills the cancellation that afflicts one root when `b²≫4ac₂`), `Δ = 4L²(r²−h²)`.
- **Ellipse ∩ edge:** affine-normalize the ellipse to the **unit circle** (an isometry-preserving
  linear map whose condition number is exactly `A/B = 1/cos φ`) and solve the same well-conditioned
  quadratic — never the general `Ax²+Bxy+Cy²+…` conic quadratic.
- **Conic parameter by closed-form inversion** (`t = atan2(...)/2π`, no iteration) is the **weld
  currency**: the seat-face arc endpoint and the tool wall's base-rim vertex are **both** generated by
  one shared `C.PointAt(t)`, so they are byte-identical → `freeEdgeCount == 0` (the ADR-0048 lesson:
  shared exact points, never independently re-derived endpoints).
- **Decline-biased gate** (mirrors `bossSeatFace`/`circleVsCap`): tangency (double root within a weld:
  `|r²−h²| ≤ (Weld()/2)²`, a **curvature-scaled** band, not a magic constant), eccentric ellipse
  (`B/A < 1e-3`, grazing-oblique), odd crossing parity, or a residual sub-tolerance sliver
  (`area < Area()` or shortest edge `< Weld()` after a merge attempt) → return `ok=false` → CSG. Merge
  slivers, never cull a kept one (culling opens a leak).

## Consequences

- Partial holes and straddling bosses become **exact analytic** solids (analytic `geom.Plane` seat +
  `geom.Cylinder`/`Cone` wall preserved), no `CodeBooleanCSGFallback`.
- Zero duplication of the hardest combinatorial code (the self-touch-correct DCEL walk, the cell
  classifier, loop grouping) — one arrangement pipeline serves ruled, torus **and** plane.
- The two core edits are additive: existing ruled/torus/cutCylinder goldens stay byte-identical (the
  new flags default to current behavior).
- Cost: the arrangement (`O((N+E)log(N+E))`, grid-hashed) is paid **only** on genuinely-partial
  contacts; the common fully-interior bore is untouched (D-b).
- The tolerance classifications (tangency band, sliver cull, vertex snap) are **tolerance-bounded**
  and model-scaled; the crossings, conic parameters and the shared weld point are **EXACT**. This
  ledger is explicit so the class of "looked exact, shipped a sliver" bug cannot recur.

## Rejected alternatives

- **A parallel planar imprint pipeline** (ADR-0045's implied path if `planeUV` were ever built):
  re-implements `chainLoops`/`nextByAngle`/`groupLoopFaces` — a second self-touch-correct DCEL
  walker — for no invariant gain. The exact bespoke-growth ADR-0045 fought.
- **ADR-0045's "never build `planeUV`"**: superseded **only for partial contact**. Its reasoning
  ("the cell subdivision buys nothing") stays law for the strictly-interior circle; for a
  boundary-*clipping* conic the subdivision is precisely what is needed.
- **Unify all drill/boss onto `planeUV`** (retire the interior fast-path): regresses the hot path and
  re-opens tested, already-exact code for no correctness gain (D-b).
- **General polygon clipping (Greiner–Hormann / Vatti)** carrying the conic as an edge attribute:
  imports a clipper we do not need, whose known fragility is exactly on the vertex-on-edge / tangency
  degeneracies this problem is dominated by.
- **Sampled-only crossings** (trust the arrangement to find the crossing on the polyline): the
  blind-straddle failure — a decisive vertex at a *sampled* point that adjacent faces disagree on.
- **Reuse `segRim` for polygon edges** / **offset `planeUV`'s `(u,v)` to dodge `u≈2π`**: the first
  trips `sameRun`'s constant-v merge; the second hides the real u-fold leak instead of gating it (D-c).

## Ghost-periodic-assumptions audit (the load-bearing finding)

Tracing every `uvSide` method **and** every shared free-function consumer reached from
`trimByImprint`, a non-periodic polygon plane trips exactly **one** genuine assumption:

| Shared consumer | Periodic assumption | Trips a plane? | Resolution |
|---|---|---|---|
| `seamWelder` u-fold (`uv_arrangement.go:517`) | folds **any** `u≈2π` vertex to `u=0`, unconditionally | **YES** — a seat vertex at ~6.283 db-units is silently welded to `u=0` | gate on `uPeriodic` (D-c), mirroring the gated v-fold |
| `sameRun` `segRim` branch (`:1082`) | rim is constant-v ⇒ equal-v ⇒ same run | YES **if** polygon edges are tagged `segRim` | new `segPolygon` kind, `sameRun` by curve identity (D-c) |
| `placeSeams` / `chooseSeamU` / `clipParams` / azimuth-unwrap | seam/band/wrap | No — `planeUV` overrides `assembleSegments`, makes `placeSeams` a no-op, never calls them | none |
| `dropArtificialLoops` / `wrappingSolidFaces` / `wrapsAllU` / `segSeam` | vPeriodic / wrap band | No — early-return / `false` / no-op; branches never fire | none |
| `chainLoops` / `nextByAngle` / `interiorOfCell` / `groupLoopFaces` / `dedgeLoopArea` | none — pure planar combinatorics | No — **better** suited to a plane (all loops contractible) | none |

**Conclusion: one additive core gate + one additive segment tag — not a periodicity-neutral rewrite.**

## Handoff — file tree, the math seam, and the slice plan

New files under `kernel/brep/` (each <500 LOC, SRP, SPDX `GPL-2.0-only`):
`curved_plane_uv.go` (the `planeUV` type + its `uvSide` methods, most one-liners) ·
`curved_plane_uv_frame.go` (polygon framing + the conic∩polygon clip) ·
`curved_plane_partial_drill.go` (`DrillPartialHole`) · `curved_plane_partial_boss.go`
(`JoinPartialBoss`). No `Oblikovati.API` surface change — the boolean dispatch is already wired.

The numerics seam the implementer calls (certified against OCCT `getMass`/section as oracle;
tolerances are model-scaled expressions, never bare literals — `Weld()=1e-9·size`,
`Plane()=1e-7·size`, `Stitch()=1e-6·size`, `Area()=1e-9·size²`):

```go
type planeConic struct{ c math.Point2; maj, min math.Vector2; A, B float64 } // circle: A==B==r
type hitClass int
const ( hitTransversal hitClass = iota; hitTangent; hitVertex; hitMiss )

// conicEdgeHits solves C ∩ (a→b) EXACTLY: stable quadratic (ellipse affine-normalized to the unit
// circle), s the edge param in (tjTol,1−tjTol), t the conic param in [0,1) (feeds uvSeg.tA/tB and
// emitImprintRun). Returns hitTangent when |r²−h²| ≤ (Weld()/2)² (curvature-scaled).
func conicEdgeHits(C planeConic, a, b math.Point2, res geom.Resolution) (hits []conicHit, cls hitClass)

// planeUVContactOK generalizes bossSeatFace/circleVsCap, decline-biased: eccentric ellipse (B/A<1e-3),
// any tangency, odd crossing parity, or a residual sub-tolerance sliver → false → CSG.
func planeUVContactOK(C planeConic, f planarFace, res geom.Resolution) bool
```

Vertical slices, each independently shippable and green, gate = **volume-vs-OCC `getMass`
(ADR-0042 relative) · `freeEdgeCount==0` · `Validate.Valid && Closed && Manifold`**:

- **Slice 0 (prep):** the two additive core touches (`segPolygon` + `uPeriodic`). Gate: **all existing
  brep/ops boolean goldens byte-identical** (flags default to current behavior). De-risks the core edit
  alone, no feature yet.
- **Slice A:** partial hole clipping one face edge — `planeUV` + frame + `DrillPartialHole`, off the
  `pierced && !clean` branch. OCC oracle `box − cylinder`; interior golden still byte-identical.
- **Slice B:** straddling/overhanging boss — `JoinPartialBoss`, reusing `planeUV` unchanged; only the
  material closure + wall/cap assembler differ. OCC oracle `plate ∪ cylinder`.
- **Slice C:** pin the interior drill/boss/counterbore output byte-identical (characterization golden),
  document D-b in-code, optionally DRY the shared `interior|partial|CSG` gate. Coverage >80%,
  duplication <3%.
- **Slice D:** this ADR → Accepted; extend ADR-0045's taxonomy table with a "curved-on-planar
  (partial)" row; live MCP visual test (plate + edge-clipping hole and a straddling boss; screenshot;
  confirm crack-free tessellation, `freeEdgeCount==0`).
