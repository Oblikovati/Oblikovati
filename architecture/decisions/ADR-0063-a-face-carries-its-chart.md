# ADR-0063 — A trimmed face carries its parametric trim; it is not re-derived from its loops

**Status:** Accepted — on `m48/kernel-ground-rules`. · **Builds on**
[ADR-0062](ADR-0062-half-space-cut-is-a-boolean.md) (the half-space cut is a difference) and
[ADR-0060](ADR-0060-loop-framed-ruled-chart.md) (the loop-framed chart). · **Deletes:** every rule that
recovered a face's region from open, period-turning polylines — `trimRegion.ringsEnclose` and its four
branches, `betweenPeriodicRims`, `rimMaterialBehind`, `ringLevelAt`, `ringCrossAt`,
`shiftAlongIntoBranch`, `acrossOf`, `ringNet`, `shiftIntoRingBranch`, `upwardRayCrossings`,
`periodicWindowHoldingMaterial`, `windowHoldsMaterial`, `faceIsRingComplement`, `regionSignedArea`,
`ringSpansAPeriod`, `ringClosesByAWholeTurn`, and `trimRegion.complement`. · **Touches:** `kernel/brep`,
`kernel/topo`.

## Context

ADR-0062 closed twenty-eight of the thirty-three failures that stood between the half-space pipeline and
its deletion, and named the two that remained as one defect with one cause. This is the fix for that
cause.

**The defect.** A trimmed face on a periodic surface is not determined by its 3D loops. A cylinder
band's two rim circles bound the strip between them AND the strip the other way round the seam. A
torus's two spiric ovals bound both of the bands they separate. A closed-surface face's single ring
bounds the little cap and the big one equally. Every reader that asks "is this `(u, v)` point on this
face" therefore has a choice to make that its input does not settle, and each of ours made it by its own
rule:

| reader | rules it carried |
| --- | --- |
| `trim_region.go` | an azimuth-turning rim read by an upward ray; a tube-turning ring read by the nearest rim above and that rim's material side; a choice between the two windows two rim levels bound on a periodic axis; the `outerless` complement flag |
| `classify_trim_uv.go` | leave a period-turning loop OPEN in the crossing count; fall back to the loop's handedness where the rings bound nothing |
| `orient_consistent.go` | reassemble a band's rims into one circuit before taking the shoelace |

Seven rules for one question, in three places, disagreeing. That disagreement is not incidental — it is
the whole run of defects ADR-0062 collected, each one a place where a rule that assumes a closed contour
met one that is not: the sampler that unwrapped only the azimuth (#3453, #3429), the two rims on a
periodic axis, the cap whose contour was missing its pole, the ring that turns the tube, the bore wall
that reported the handedness of its own opposite (#3506).

**Grounded in OCCT, this is not a family of special cases; it is the bill for a representation choice.**
OCCT stores a face's SEAM edge explicitly: twice in the wire, with two pcurves. Every wire is therefore
a closed contour in `(u, v)`, and one uniform pair of rules covers every face it can build —
`BOPAlgo_WireSplitter::Path` assembles edges into wires by minimum angle at each vertex, and
`PerformAreas` then classifies each wire by the signed area of its `(u, v)` polygon, with
`IntTools_FClass2d::Init` falling back to a point classification exactly where that area degenerates. No
rule anywhere in it asks whether a loop wraps, because no wire of OCCT's ever does.

Our charts drop the seam — `dropArtificialLoops` removes it precisely because it bounds nothing real in
3D, which is true and is why the 3D loops are right to be what they are.

**The cheap version does not work, and the reason is the decision.** `regionSignedArea` already
reassembles the closed contour by concatenating a face's period-turning rings, so the obvious economy is
to classify a POINT against that same concatenation. Measured: it takes the oblique figure-eight from 84
wrong sample points to 10, and takes the ordinary perpendicular torus band from 0 to 115 of 120 — it
INVERTS a case that works. `loopToUV` unwraps each loop onto whichever turn it started on, so two rims
of one band routinely land on different branches, one walking 0→2π and the next 2π→4π; a SHOELACE
survives that (the closing chords supply the seams, which is the whole trick `regionSignedArea` relies
on) but even-odd does not, and the concatenation reads as a zigzag. Aligning the branches first does not
rescue it. And where two lobes TOUCH — the oblique figure-eight — there is no seam-free circuit to
reassemble at all.

A contour cannot be reassembled after the fact. It has to be carried.

## Decision

**A face's parametric trim is part of its definition, recorded by the producer that wound it.**
`topo.Face` carries a `chart`: closed contours in the covering space of its surface's own `(u, v)`,
outer first, material on the left. `brep` records it on every face the `(u, v)` arrangement builds and
reads it wherever a face's region is asked for.

Three things follow.

**Producing it costs one bit.** `keptBoundaryEdges` welds `u=2π` onto `u=0` (and `v` likewise on a
torus), which turns a wrapping region's two seam traversals into reverse twins that cancel. Run the same
trace with that fold OFF and they survive: the walk that yields two open rim polylines yields the one
closed contour that runs rim → seam → rim → seam. That is the chart, and it is traced from the same
cells as the loops, so the two cannot disagree.

**The unit is the kept COMPONENT, not the emitted loop group.** A component carries contours its loops do
not: the genus-1 complement's outer contour is the whole parameter rectangle, all seam, which
`dropArtificialLoops` rightly removes from the 3D loops and which in the chart is exactly what bounds
the face. Claiming contours by the loops that survived left that face with its hole and no outer, and
read every point of it inverted — measured, 217 of 217 samples.

**Handedness is read against the region, not from a shoelace.** The chart is orientation-free — it winds
material-on-left whatever the face's stored sense — so `loopHandedness` cannot come from its area. It
now asks the question the definition is made of: a boundary is traversed with material on its LEFT as
seen from outside, so step a hair to the left of a sampled boundary edge and ask the region whether that
step landed in the face. Every edge votes. That deletes the rim-circuit reassembly and the complement
negation together.

**A face with no carried chart derives one, or declines.** Deriving is legitimate exactly where it is not
a guess:

- rings that all CLOSE in the covering space already bound one region there, and are the chart;
- an `outerless` face is framed by its surface's own parameter rectangle — a full turn on a periodic
  axis, the surface's domain on a bounded one (a sphere's latitude, which ends at its poles);
- two rings that each turn a period, running OPPOSITE ways, on a surface whose OTHER axis is not itself
  periodic, bound exactly one strip, and the seam that closes them is determined: re-cut both at the
  same crossing and join them.

Anything else — two turning rings on a doubly-periodic surface, a non-monotone turning ring, more than
one band — is refused. `fluxDomain` then returns `ok=false` and the shell is left uncertified, which is
the existing named decline; nothing picks a side. That is the whole difference from the deleted rules,
which always picked, and picked the same one every time.

## Consequences

**Measured, against an oracle that reads no chart.** A point on the operand's surface belongs to the
result's face exactly when the boolean's own keep rule says so against the tool's half-space. Over eight
cases and ~1700 samples the carried chart is right at every one, including both oblique figure-eight
rows, where the readers it replaces are wrong at 108 of 218 and 110 of 222. That is
`TestFaceChartCoversTheKeptSide`, kept as the gate.

**Net delta.** Seventeen functions and two struct fields deleted; two added (`chartContains` and the
derivation that closes a band). Six of the seven rules for "which side of its rings is the face" are
gone, and the seventh — the derivation — refuses where it does not know.

**What this does not do.** It does not put a seam edge in the B-rep. The 3D loops stay the true
geometric boundary: two rim circles, not a keyhole. OCCT pays for its uniform rule with a wire that
mentions an edge bounding nothing, and with the edge-use counts, tessellation and naming that follow
from it; this records the same fact beside the loops instead of inside them.

**Producers that record no chart.** The primitives and the planar constructors do not, and derive. STEP
carries pcurves and should hand them over rather than be re-derived; that is follow-up work, not a gap
this opens — those faces derive today exactly as they did before.
