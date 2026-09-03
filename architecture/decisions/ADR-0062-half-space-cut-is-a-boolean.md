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

## Consequences

**ADR-0061's stage order was wrong, and this corrects it.** Stage 3 — the sphere and torus charts — is
stage 2's GATE, not the stage after it: the half-space cut is the only reason those two surfaces need a
split pipeline of their own, and until the mixed boolean charts them the old pipeline cannot be
deleted. The corrected order is: close the cone's axis-parallel hyperbola, chart the sphere and the
torus, then land stage 2's deletions in one commit, because one equivalence unlocks all of them.

**The gate is the table above, kept as a corpus test.** `TestHalfSpaceCutEqualsABoundedDifference`
compares the two paths on every row and records which still differ. It is a ratchet: a row that moves
from "differs" to "equal" is a stage landing, and no row may move the other way.

**The hyperbola gap is a real defect, not a missing recogniser.** A planar tool face whose section with
a cone is a hyperbola arm is dropped, so the cut is not closed. It is the same class as the shared-plane
classification ADR-0061 stage 1 fixed — a face the pipeline builds but does not keep — and it is fixed
in the general path, not by a branch for hyperbolae.
