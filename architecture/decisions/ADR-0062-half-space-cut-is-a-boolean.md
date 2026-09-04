# ADR-0062 — The half-space cut is not an operation; it is a difference against a bounded plane

**Status:** Accepted — on `m48/face-sense-invariant`. · **Scopes**
[Oblikovati#3509](https://github.com/Oblikovati/Oblikovati/issues/3509) (migrate the half-space and
ruled∩ruled paths to the loop-framed chart; delete the band frame and `cutCylinderUV`) and
[#3255](https://github.com/Oblikovati/Oblikovati/issues/3255) (replace `splitFaceByPlane`'s five-tier
ladder). · **Supersedes the stage ORDER of**
[ADR-0061](ADR-0061-csg-fallback-retirement.md) — stage 3 is stage 2's gate, not its successor. ·
**Builds on** [ADR-0060](ADR-0060-loop-framed-ruled-chart.md) (the loop-framed chart) and
[ADR-0058](ADR-0058-tolerant-analytic-boolean.md) (the mixed per-face boolean). · **Deletes:** the
whole half-space split pipeline — `splitFaceByPlane` and its ladder, `cylinderSideUVSplit`,
`coneSideBandSplit`, `coneApexSideSplit`, `loopedSplit`, `capSplit`, the perpendicular
cylinder/cone/torus fast paths, `fullCylinderSideBand`, `fullConeSideBand`, `fullConeApexSideBand`,
the band frame (`bandFrameSegments`, `keyholeOuter`, `topRimReversed`), `cutCylinderUV` and its gate,
and the synthesised lid. · **Touches:** `kernel/brep`.

## Context

ADR-0061 stage 2 was written as "migrate the band-framed handlers to the loop-framed chart". Twice that
migration was attempted and twice it turned about twenty corpus tests red. The failures were not a
deficiency in the chart — the chart trims the walls correctly, as its own probes showed — they were a
CONTRACT mismatch, and reading how two independent kernels do the same job names it exactly.

**OCCT.** `BOPAlgo_BuilderFace::Perform` splits a face in four phases: drop the edges that cannot bound
an area, assemble the remaining edges into wires (`BOPAlgo_WireSplitter`, whose `Path` walks the face's
own `(u,v)` graph choosing the minimum clockwise angle at each vertex), classify those wires into
growths and holes, then put the internal edges back. There is NO per-surface frame anywhere in it: the
face's own edges — its original ones, split at the intersections, plus the new section edges — are the
entire input, whatever the surface is. Outer-versus-hole is decided in `IntTools_FClass2d::Init` from
the SIGNED AREA of the wire's `(u,v)` polygon: positive is a growth, negative a hole. When that area
degenerates (`|aS| < Precision::SquareConfusion`) the wire is flagged `BadWire`, orientation is
abandoned for the whole face, and a POINT classification decides instead — and for a periodic surface
the 2π `u`-window is centred on the wire's own extent rather than fixed. That degenerate case is
precisely a seam-wrapping band rim, whose `(u,v)` polygon shoelaces to zero: the same degeneracy #3506
found here, met with a point classification rather than a synthetic frame.

**solvespace.** `SShell::MakeFromBoolean` computes every surface-surface intersection curve first
(`MakeIntersectionCurvesAgainst`, into one shared curve list), then calls `CopySurfacesTrimAgainst`
SYMMETRICALLY on both shells: every surface of BOTH operands is copied and re-trimmed against the other,
by the same shared curves. Nothing is synthesised for one side.

Both kernels therefore say the same two things. A face is framed by its own loops, whatever its
surface — which is ADR-0060, independently confirmed. And **both operands are split by the same shared
curves**; neither builds one side's boundary out of the other side's leftovers.

Oblikovati's half-space cut does the opposite. It splits only the target, asks each wall to hand back
the section arcs it trimmed on, and SYNTHESISES a lid from them. That contract is why the migration
kept failing: the loop-framed chart returns faces, not leftovers, and the pipeline it was being ported
into is shaped around leftovers.

## Decision

**A half-space cut is a difference against the plane's positive side, bounded to the target's box.** It
is what OCCT does — `BRepPrimAPI_MakeHalfSpace` builds exactly that solid and hands it to the ordinary
BOP — and it makes the lid what it always was: the tool's own face, split by the same section curves
that split the wall, welded because both sides were cut by the same edge.

That deletes the entire parallel pipeline named above. `HalfSpaceCut` keeps its signature and becomes
the bounded prism plus `Boolean(Difference, …)`.

**Measured before deciding**, on the surface families the cut serves:

| configuration | `HalfSpaceCut` | `Difference` against the bounded plane |
| --- | --- | --- |
| cylinder, axis-parallel off-centre | 4 faces, closed | 4 faces, closed |
| cylinder, oblique | 3 faces, closed | 3 faces, closed |
| cylinder, perpendicular | 3 faces, closed | 3 faces, closed |
| cone, oblique ellipse | 3 faces, closed | 3 faces, closed |
| cone, through the apex | 3 faces, closed | 3 faces, closed |
| cone, axis-parallel hyperbola | 4 faces, closed | 3 faces, OPEN — the tool's lid face is dropped |
| sphere, cap ×2 | 2 faces, closed | declines: no sphere chart |
| torus, perpendicular and spiric | 2–3 faces, closed | declines: no torus chart |

The equivalence already holds for the whole RULED family. Two things stand between it and the
deletions, and both are already on the retirement's list.

**Correction, measured 2026-09-03.** Every row above is now `equal` — and the deletion is still NOT
unlocked, because the table was too small. Deleting the pipeline against a green ten-row table turned
22 tests red in `kernel/brep` alone. The rows did not cover a cone bounded by its own APEX (four
configurations), a spiric cut that leaves ONE oval (two), or the notched-cylinder fixtures a second cut
composes on. So "one equivalence unlocks all of them" was wrong as stated: one equivalence unlocks
nothing on its own, and the GATE is the table only to the extent the table covers what the pipeline
serves. The table now carries sixteen rows, six of them `differs`, each naming its own gap. The
deletions land when every row is `equal` — and a row is added for every configuration the deletion
would otherwise silently drop, BEFORE the deletion, not after.

The apex gap is one defect, not four: the loop-framed chart takes its v-window from the face's own
loops (ADR-0060), and a cone bounded by its apex has no loop there — the apex is a singular POINT. It
is the same thing a sphere's POLE is, which `sphereFaceUV` already frames. The chart needs the ruled
apex for the same reason and in the same way.

**All sixteen rows are `equal`.** Three defects closed them, each in the general path:

1. **The recogniser refused a cone bounded by its apex.** It need not: such a face carries a RULING out
   to the apex and back, and that ruling IS a frame edge, so the face's own loops already reach it. What
   the chart does need is to CLOSE its parameter rectangle there, with a degenerate segment spanning the
   azimuth at the apex — the same thing `sphereFaceUV.poleSegments` does at a pole, and the same thing
   OCCT's degenerate edges are in a face's wire. Refusing instead sent every apex cone to the pass
   bucket, whose gate cannot prove a cone clear of a crossing plane, and declined the whole boolean.
   Four rows.

2. **An imprint was classified before it was assembled.** A section arrives as however many curves its
   own construction makes: a plane between a torus's tube radii sections it in ONE oval delivered as TWO
   spiric branches. Neither branch closes, so neither was an island; neither crossed the receiving
   face's boundary, so neither was an open crossing; both fell to the STRAIGHT bucket, where a curved
   arc becomes the chord between its ends. The oval collapsed to a sliver, the face kept everything, and
   the tool's lid passed through untrimmed. Open arcs are now chained into cycles BEFORE they are
   classified — `BOPAlgo_BuilderFace` does exactly this, `PerformLoops` before `PerformAreas` — and the
   open/island test now reads curvature rather than conic-ness, so nothing in it knows a spiric from an
   ellipse.

3. **The two branches of one oval did not agree on the point they share.** They meet where |w| = 1, the
   one place `arccos` is infinitely steep: half an ulp of error in `w` becomes 3·10⁻⁸ of azimuth. Each
   branch then sampled the shared point somewhere else, the arcs never welded into a loop, and the
   arrangement saw an open chain that divided nothing — so the SAME cut kept the whole face or trimmed
   it correctly depending only on which branch came first out of the intersector. `w` is now resolved to
   ±1 where it is within rounding of them, before the `arccos`, so both branches evaluate one azimuth.
   The corpus for it is a sweep of near-degenerate planes, because a single tidy plane passes.

## Consequences

**ADR-0061's stage order was wrong, and this corrects it.** Stage 3 — the sphere and torus charts — is
stage 2's GATE, not the stage after it: the half-space cut is the only reason those two surfaces need a
split pipeline of their own, and until the mixed boolean charts them the old pipeline cannot be
deleted. The corrected order is: close the cone's axis-parallel hyperbola, chart the sphere and the
torus, then land stage 2's deletions in one commit, because one equivalence unlocks all of them.

**Measured progress, 2026-09-04.** Rewiring `HalfSpaceCut` to the bounded
difference costs **2** failing tests in `kernel/`, down from 33 — plus
`TestLoopedSplitHalvesACapBySymmetry`, which is a unit test of `loopedSplit` and
goes with the deletion.

Nine defects account for the 30 closed, each fixed in the general path, each with
its own corpus, and each proved by disabling it: a shared section clipped to BOTH
trims; a slit, which bounds nothing, no longer cutting one; the parabola admitted
as the third conic; a cone bounded by its own apex; an imprint assembled into
cycles before it is classified; two rims on a periodic axis bounding two bands; a
cap's contour completed by the line at its pole; a closed-surface face anchoring
the orientation two-colouring; and a ring that turns the TUBE read by the same
winding rule as one that turns the azimuth.

Three of those nine were defects in MEASUREMENT, not in the boolean. A hemisphere's
area could not be integrated at all — `AnalyticFaceArea` measured a whole sphere
exactly and declined its half — so every sphere-capped body was gated by a mesh
rather than by the analytic B-rep the ground rules require. They now measure
exactly: 157.0796, 261.7994, 205.2507.

**What is left is the exact tangency**, and it is localised. Both remaining rows
are the oblique FIGURE-EIGHT: a plane grazing a tilted torus's inner equator, where
the two spiric lobes merge into one self-touching loop. Each lobe turns the TUBE —
netU = 0, netV = ±2π, one each way — so the two of them cut the torus into two
bands that touch at the pinch, and the difference emits the WRONG one. It builds a
clean two-face analytic body, one torus face and one lid, no CSG soup; it simply
bounds the other band.

Measured, so the search is over:

- the material predicate is RIGHT. The arrangement's four cells classify correctly
  against the tool — the two at z = 1.106 and 4.537 kept for `z ≥ 1`, the two at
  z = −1.513 dropped — so nothing is wrong with the keep table or the membership.
- the emitted FACE is wrong. `brep.PointInFaceTrim` on it puts z = 3.81 and 1.59
  outside the trim and z = −3.20 inside.
- it is not the `outerless` shoelace. A self-touching loop's signed area is the
  DIFFERENCE of its lobes' and names no region — but the figure-eight takes the
  wrapping path (`wrappingComponents`), where that flag is never read.

- the emitted BOUNDARY is right too. Stepping to the left of the longest boundary
  edge lands in a kept cell, so `keptBoundaryEdges` did put the material on the
  left, and reversing the traversal changes nothing downstream.

So the boolean is not what is wrong here. The face it emits is bounded correctly
and consistently with the cells; what disagrees is how a face bounded by two
TOUCHING tube-wrapping rings is READ — by `pointInCurvedFace`, which takes
inwardness from a nearest foot and the loop's local direction, and by the
tessellator, which picks a band from the same winding. Both name the other band,
and both are downstream of the operation.

**The sharpest lead, and where to start.** The two paths emit the torus face
differently, and it is one bit:

| | lid faces | torus loops |
| --- | --- | --- |
| retired pipeline | 2, one per lobe | 2, emitted `t=[1,0]` and `t=[0,1]` |
| the difference | 1, a self-touching loop | 2, both emitted `t=[0,1]` |

Each lobe is a closed spiric wrapping the tube, and the two wrap it OPPOSITE ways
(netV of +2π and −2π), so "both forward" runs them the same way round the region
and "one reversed" runs them opposite. The retired pipeline reverses one; the
difference reverses neither, and `brep.PointInFaceTrim` then misplaces 69 of 144
sample points on that face, boundary points excluded.

Two things were tried and REVERTED as no-ops, so they need not be tried again: a
certification of the boundary traversal against the arrangement's kept cells (both
loops pass — they are correctly wound in the CHART), and a certification against
the membership predicate by stepping along `inwardAt` (both pass at a single
sample). The second is not merely unlucky: for a TUBE-wrapping loop the surface
normal rotates the whole way round, so `inwardAt` sweeps through every direction
along one loop and no single sample of it says anything. A criterion for this case
has to be combinatorial — the relative sense of the two wrapping loops — not a
sampled one.

**Grounded in OCCT, the last case is not a special case at all — it is the bill
for a representation choice.** OCCT stores a face's SEAM edge explicitly: twice in
the wire, with two pcurves. Every wire is therefore a closed contour in (u,v), and
one uniform pair of rules covers every face it can build —
`BOPAlgo_WireSplitter::Path` assembles edges into wires by minimum angle at each
vertex, which resolves a degree-4 pinch on its own, and `PerformAreas` then
classifies each wire by the signed area of its (u,v) polygon, with
`IntTools_FClass2d::Init` falling back to a POINT classification exactly where that
area degenerates. No rule anywhere in it asks whether a loop wraps.

Our charts drop the seam: `dropArtificialLoops` removes it precisely because it
bounds nothing real. What that buys in simplicity it pays for in every wrapping
loop being an OPEN polyline in the covering space, with no shoelace and no
interior — which is why this ADR has collected a run of defects that are all one
defect: the sampler that unwrapped only the azimuth, the two rims on a periodic
axis, the cap whose contour was missing its pole, the ring that turns the tube.
Each was a place where a rule that assumes a closed contour met one that is not.

The figure-eight is the end of that run, and the one the reassembly cannot reach:
its two lobes touch, so there is no seam-free circuit to reassemble. `nextByAngle`
is already OCCT's angular rule and already resolves the pinch; what is missing is
the closed contour to apply the area rule to afterwards.

So the fix is to carry the seam through the emission, as OCCT does, rather than to
add a fifth rule for loops that wrap. That is an architectural change and it wants
its own ADR — it would DELETE the wrapping-loop special cases rather than join
them, which is the shape this repository's rules ask for.

**And the cheap version of it does not work, which is worth knowing before anyone
tries.** `regionSignedArea` already reassembles the closed contour by concatenating
the wrapping rings, and the obvious economy is to classify a POINT against that
same concatenation rather than build the seam. Measured: it takes the figure-eight
from 84 wrong sample points to 10, and takes the ordinary perpendicular torus band
from 0 to 115 of 120 — it INVERTS a case that works today.

The reason is exact. `loopToUV` unwraps each loop onto whichever turn it started
on, so two rims of one band routinely land on different branches, one walking
0→2π and the next 2π→4π. A SHOELACE survives that — the closing chords still
supply the seams, which is the whole trick regionSignedArea relies on — but
even-odd does not: the concatenation of two independently-unwrapped rings is a
zigzag, not a contour, and a ray cast through it counts nothing meaningful.
Aligning the branches first does not rescue it either (tried, measured, no change).

A contour cannot be reassembled after the fact; it has to be carried. That is what
OCCT's explicit seam edge IS, and why it is load-bearing rather than a convenience.

The earlier count of 5 was measured before the last three defects landed. Six defects in the
general path account for the 28 closed: a shared section clipped to BOTH trims (not
just the planar one); a slit, which bounds nothing, no longer cutting a section; the
parabola admitted as the third conic; a cone bounded by its own apex; an imprint
assembled into cycles before it is classified; and two rims on a periodic axis
bounding two bands, with the winding saying which.

**The gate is the table above, kept as a corpus test.** `TestHalfSpaceCutEqualsABoundedDifference`
compares the two paths on every row and records which still differ. It is a ratchet: a row that moves
from "differs" to "equal" is a stage landing, and no row may move the other way.

**The hyperbola gap is a real defect, not a missing recogniser.** A planar tool face whose section with
a cone is a hyperbola arm is dropped, so the cut is not closed. It is the same class as the shared-plane
classification ADR-0061 stage 1 fixed — a face the pipeline builds but does not keep — and it is fixed
in the general path, not by a branch for hyperbolae.

## The last case, after ADR-0063: it is the LID, and the section that touches itself

ADR-0063 carried the chart, and that settled what this ADR named as the cause. Measured against an
oracle that reads no chart — a point on the operand belongs to the result exactly when the boolean's
keep rule says so against the tool's half-space — the oblique figure-eight's torus face is now right at
**every one of 218 samples**, where the readers it replaced were wrong at 108. `brep.PointInFaceTrim` on
that face no longer misplaces anything.

**The volumes did not move**: ∩ 239.84 against OCC's 151.90, − 149.61 against 242.89, byte for byte. So
the region was never the whole story, and the remaining defect is in a different place. It is localised
and measured:

| | value |
| --- | --- |
| result faces | 2 — one torus, one planar lid |
| torus face, analytic area | 146.756 |
| **planar lid, analytic area** | **1.41e-06** |
| lid chart contour | one circuit, 512 samples |
| lid lobes, by chart shoelace | +25.2667 and +25.2667 (sum 50.533) |

The lid's CHART is right: its two lobes each wind counter-clockwise and add to 50.53. What integrates to
zero is the lid's 3D LOOP — a single closed spiric traversed once, which is a figure-eight in its own
plane, so its two lobes contribute opposite boundary integrals and `∮` cancels. The retired pipeline
emitted **two lid faces, one per lobe**; the difference emits one self-touching loop. That was recorded
above as "the sharpest lead" and it is exactly right — it is the lid, not the band.

**Why the wire is never split.** OCCT's `WireSplitter::Path` resolves a self-touch by choosing the
minimum angle AT THE VERTEX, and `nextByAngle` already does the same — for a TRANSVERSAL crossing,
where the arrangement has a vertex because the two branches cross. This touch is TANGENTIAL: the two
lobes meet with equal tangents, nothing crosses, and no vertex is created. The trace's nearest
non-adjacent approach to itself measures **1.03e-07** against a weld grid of **1e-07**, so the two
passes do not even weld — the circuit walks straight through its own touch. Splitting the traced loop
wherever it approaches itself within `tjTol` was tried and does not fire, for that reason (measured;
the lid stayed one contour at 1.41e-06).

**So the fix is to SOLVE the touch, not to sample it.** An imprint curve meeting itself is an incidence
of exactly the kind `planeFaceUV` already solves — its own comment says "One imprint meeting another is
an incidence too, and no frame crossing covers it" — with `i == j`. Solved and injected as a shared
vertex, the touch becomes a degree-4 vertex, `nextByAngle` splits the circuit into its two lobes with no
new rule, and `groupLoopFaces` emits the two lid faces the retired pipeline emitted. Widening the weld
grid instead would be an epsilon standing in for a solve, and 1.03e-07 against 1e-07 shows how little
margin such a grid has.

That is the whole of what remains for these two rows.
