# ADR-0061 — The faceted fallbacks retire in stages, against a debt that can only fall

**Status:** Accepted — on `m48/face-sense-invariant`. · **Scopes**
[Oblikovati#2251](https://github.com/Oblikovati/Oblikovati/issues/2251) (delete the `booleanCSG`
triangle-soup BSP engine) and the stages below. · **Builds on**
[ADR-0052](ADR-0052-planar-boolean-corefined-seam-classification.md) (the co-refined seam the mesh
arrangement rescues), [ADR-0056](ADR-0056-analytic-face-reconstruction-boolean.md) (the reconstruction that was to
replace it), [ADR-0058](ADR-0058-tolerant-analytic-boolean.md) (the mixed per-face boolean) and
[ADR-0060](ADR-0060-loop-framed-ruled-chart.md) (the loop-framed ruled chart that closed the last road
to `ops.Facet`). · **Adds:** the `fallbackDebt` ratchet, the shared-plane
rule in the boolean's membership classifier, and the certificate on the public curved-boolean entry. ·
**Deletes:** stage by stage, every call site into a faceted engine, then the engines. · **Touches:**
`archguard`, `kernel/brep` (classification, the coplanar predicate), `kernel/ops/boolean` (the public
entry).

## Context

Four call sites still hand a boolean to an engine that returns a triangle soup: two to `booleanCSG`,
one to the exact mesh arrangement, one to the mesh reconstruction. Each exists because some
configuration declines out of the analytic pipeline, and each produces a body whose curved faces are
gone for good — every downstream fillet, thread, mass property and export then reads facets.

They are also the reason the suite is slow. On the multipoint disk of the Inventor corpus, one feature
falling to the faceted path on a 400-face analytic body took the rebuild from two minutes to
thirty-five. The cost is not the fallback itself: it is that faceting is permanent, so every LATER
feature meets a 500-face polyhedron instead of a dozen analytic faces, and meets it at the grazing
angles a 24-gon makes and a cylinder never would.

The ground rules already forbid the shape — "a new engine shipped beside the old one as a fallback is
not complete", "a strangler migration carries the ticket that deletes the old system and the corpus
gate that unlocks it". What was missing is a NUMBER. Nothing counted how much of the boolean still
leaves the exact pipeline, so "we are retiring the fallback" was an intention, not a measurement, and
each stage's progress was invisible until the whole thing was done.

## Decision

**The retirement is measured before it is attempted.** `archguard.TestCSGFallbackDebt` pins three
counts that all reach zero exactly when the retirement is complete, and fails on ANY move, up or down,
after the pattern of `kernelNetDeltaPin`:

| number | what it counts | reaches zero at |
| --- | --- | --- |
| `faceted-entry-sites` | calls to `booleanCSG`, `booleanViaMeshbool`, `reconstructedCurvedBoolean` | stage 6 |
| `mixed-decline-returns` | returns of `ErrUnsupportedMixedBoolean` — the mixed boolean's named declines | stage 5 |
| `faceted-engine-files` | the non-test sources of `kernel/meshbool` and the `csg`/`meshbool`/`mesh_brep` files of `kernel/ops/boolean` | stage 7 |

The first two are the doors, the third is the room. Pinning the room as well is what stops the
retirement becoming a strangler that never strangles: closing every door while the engines stay is not
done. A FALL is a stage landing and the pin comes down in the same commit; a RISE is a new door and
needs a reason in the PR. Baseline taken 2026-09-03: 5, 3, 38.

**Stages run in dependency order, each with the corpus gate that unlocks the next.** They are, in
order: (0) count it; (1) the shared-plane classification below; (2) one ruled chart with partial
conic arcs, deleting the band frame, `cutCylinderUV` and the `splitFaceByPlane` ladder — measured
2026-09-03 at 22 non-test files and 3 637 lines under `curved_halfspace_*`, which is the stage's real
size
([#3509](https://github.com/Oblikovati/Oblikovati/issues/3509),
[#3508](https://github.com/Oblikovati/Oblikovati/issues/3508),
[#3255](https://github.com/Oblikovati/Oblikovati/issues/3255)); (3) sphere and torus charts, deleting
the ball-and-rod recognizers; (4) every ruled crossing through one general pipeline, deleting the 26
recognizers of `curvedExactPaths` ([#2246](https://github.com/Oblikovati/Oblikovati/issues/2246),
[#2153](https://github.com/Oblikovati/Oblikovati/issues/2153)); (5) a chart for freeform faces, which
ends the pass bucket; (6) failure becomes local, so one bad face no longer discards a whole analytic
body; (7) delete the engines ([#2251](https://github.com/Oblikovati/Oblikovati/issues/2251)).

**Stage 1 — a sub-face point on a plane the other solid shares is classified from both sides.** This
is the first stage to land, and it is the defect the multipoint disk was left on.

A boolean classifies each fragment of a face by asking whether a point ON that fragment lies inside the
other solid. `coplanarCover` resolves the point that lies on a FACE of the other solid, through the
ON/ON table. The case nobody had resolved is the point that lies on that face's PLANE and on no face of
it: it is not on the other solid's boundary, so the ON/ON table does not apply, yet neither evaluator
of the membership oracle can answer it.

- The ray-parity classifier pierces every face of that plane at t≈0, so every candidate direction
  grazes and `firstCleanDirection` finds none. Direction re-selection is the mechanism built for a
  grazing CROSSING; here the degeneracy is at the ray ORIGIN, and no direction can move it.
- The winding-number fallback zeroes exactly those faces by design (`faceSolidAngle`'s on-plane rule,
  which is right — a flat polygon subtends no solid angle at a coplanar point), and where the point
  also lies within the on-plane band of a thin feature's walls it zeroes those too, leaving a sum that
  reads "outside" for a point in the solid's interior.

`coplanarCover` now reports `onPlane` alongside `covered`, and `insidePlaneSafe` answers the on-plane
uncovered case by probing a resolution-derived step to EACH side of the plane. Requiring the two probes
to agree is what makes it a certificate rather than a guess: the point is provably not on the other
solid's boundary, so the material to either side is the same material, and a disagreement means the
point is on the boundary after all and the caller keeps its own verdict. No output coordinate moves —
this displaces a classification query, never geometry.

**One operation, one certification.** `CurvedBoolean` and `CurvedBooleanWithDiagnostics` — the entries
the model layer calls — went straight to `curvedExactBoolean`, while the identical call inside
`booleanGeneralExact` went through `curvedExactGuarded`. So a recognizer that over-matched to a valid
body of materially wrong shape was certified for the kernel's own caller and uncertified for the
feature layer's; only the feature layer's face-count gate stood between it and the model. Both public
entries now take the guarded path, which is what the guard's own doc always claimed.

**A predicate about planes answers "no" about a cylinder.** `coplanar` read both faces' planes before
checking their kind. A cylinder's `NormalAt(0, 0)` is a valid unit vector, so a cylinder whose radial
normal aligned with the plane's passed the parallel test and the type assertion panicked. The mixed
boolean's coplanar cover screens every face of the other operand, cylinders included, so this fired on
the slotted screw's cross-hole — where `safeRecompute` turned the panic into a sick feature and the cut
silently did nothing. `planeOf` reports the kind; `coplanar` is total.

**What did NOT land: widening the exact path's gate to a classification.** The feature layer gates the
exact curved boolean on a face COUNT — the tool must be a bare analytic primitive, or the target one
against an all-planar tool. That is a special case where the rules ask for a classification: the planar
path cannot consume a curved face at all, so "either operand carries one" is the honest gate. Measured,
it removes twenty `CodeBooleanAnalyticFaceted` defects from the multipoint disk and takes its rebuild
from 226 s to 39 s. It is held back because it also drives a fine-pitch coil join into the mesh
reconstruction, which does not terminate on that body: an unconditioned gate trades a faceting defect
for a hang. The widening waits on the cost gate stages 2 and 4 give the analytic paths, and
[#2254](https://github.com/Oblikovati/Oblikovati/issues/2254) carries it with the coil as its corpus.
Recording it here rather than landing it is the point: the gate is a known special case with a measured
replacement and a named blocker, not an open question.

## Consequences

The multipoint disk rebuilds as one closed solid, which it had not done since the part entered the
corpus. Two more defects fell out of the same trace: an uncertified public entry and a predicate that
panicked on a cylinder, both of which had been masked by the very gate this ADR wants to delete — which
is the argument for deleting it, not for keeping it.

The corpus keeps a reduced fixture for the classification defect. `wedgeStepBlock` is eighteen faces
carrying the three properties that reproduce it — a face in the shared plane, a razor wedge standing on
it whose converging walls fall inside the on-plane band, and a bore so the body takes the ray-parity
classifier — where the original input was a 523-face body. `TestUncoveredPointOnASharedPlaneIsInside`
asserts the fix AND that the plain query still gets it wrong, so it can never pass vacuously.

Stage 6 changes user-visible behaviour and is called out here before it is written: after it, an
unmodelled configuration is a sick feature naming the faulty face, not a faceted body that looks solid.
That is what "failure is local" asks for, and it is a deliberate trade — some parts that render today
will render as a quarantined feature instead. If a transition period is wanted, it is a gate inside
stage 6, and this ADR is superseded rather than edited to add one.

Until a stage lands, the tests that assert a fallback FIRES (`boolean_partial_rim_test` ×3,
`partial_rim_decline_test`, `diag_integration_test`) stay as they are: they assert the decline code,
never the faceted body, so each converts to a positive corpus case when its configuration lands rather
than being deleted to make a number move.

## Stage 2: what the rewire still costs, measured 2026-09-04

The half-space rewire (`HalfSpaceCut` → `Boolean(Difference, body, BoundedHalfSpace(plane, box))`) now
costs a handful of failures, down from thirty-three. (The figure "three" originally written here was
measured on a pathological state and is corrected in the section below; the clean count is six leaf
cases.) ADR-0063's carried chart and the solved section touch
took the rest. What is left is recorded here so the next attempt starts from the measurements rather
than from the symptoms.

**1. The oblique figure-eight: tangency is coincidence, not proximity.** The two spiric lobes arrive as
separate closed ovals that meet at the pinch. Their endpoints are evaluated independently and land
1.03e-07 apart, because the spiric's `u(v) = Φ ± arccos w` is ill-conditioned there — `d(arccos)/dw`
diverges as `w → ±1`. Carrying the SOLVED meeting on the arc ends (`imprintArc.meet0/meet1`) fixes that
much: measured, the arrangement goes from **one** kept cell to the **two** lobes it should have, each
25.266.

It is still one traced loop, and the reason is not a tolerance. Near a tangential contact the two curves
separate QUADRATICALLY, so over a stretch either side of the pinch they are indistinguishable at the
arrangement's own `arrTol` of 1e-09 — the two cells share EDGES there, not a point, and
`keptComponents` reports one component. Grouping on the finer grid was tried and does not help
(measured, reverted); neither does splitting a traced loop wherever it approaches itself within
`tjTol` (measured, does not fire). What this needs is to COLLAPSE a tangential coincidence to its solved
contact — to recognise that a stretch of two curves lying within tolerance of each other is one point,
not a shared edge. That is a distinct piece of work and it is the whole of this row.

Note the shipping path does NOT take that route: through the mixed boolean the same body is exact
(0.000000 against OCC on both axis-parallel rows, 0.015 and 0.012 on the oblique pair).

**2. `TestLoopedSplitHalvesACapBySymmetry`: one corner placed three ways.** A sphere cap halved by a
symmetry plane leaves six open edges. The corner at (0, ±4, −3) — where the cap's rim circle meets the
cutting plane — is emitted three times over:

| placed by | value |
| --- | --- |
| the sphere patch's rim arc | (0, 4, −3) exactly |
| two `LineSegment`s | y = 3.9996767761694647 |
| an `Arc3d` | (0, 3.999882919988047, −3.000156100336764) |

The disagreement is 3.2e-04, which is the chord SAGITTA at the sampling density — some path resolves
that incidence on a sampled polyline instead of solving it. Two candidates were checked and are NOT the
source: the lid's own frame×imprint crossings ARE solved exactly (measured, at ±4 to the printed
precision), and the prism face's straight imprint does not reach the section circle at all
(`geom.CurveTouches` returns no hit), so an island×straight incidence is not it either. The producer of
the 3.9996767 point is not yet identified; find it before writing any code.

**A third candidate was checked and REFUTED — it is not a gap at all.** `sphereFaceUV` never
populates `loopFrame.crossings`, while `ruledFaceUV` does through `admits` — so the sphere chart samples
straight past its own frame×imprint incidences where the ruled chart solves them. Adding
`c.crossings, _ = c.solveFrameCrossings(imprint)` to `sphereFaceUV.assembleSegments` DOES find them
(measured: two crossings on this body, where there were none), and the body still does not close, so it
was not kept for want of a case that turns green.

  **Measured 2026-09-05 and dropped for good.** A direct test — a sphere cap whose rim is cut by a
  section, asserting the assembled boundary carries a vertex at the exact crossing — passes at
  **9.2e-16 WITHOUT the call** and 6.7e-16 with it. The sphere chart already places that incidence
  exactly; `solveSeamCrossings` and the frame sampling between them cover it. The call is neutral on
  the shipping path and on the rewire count, and now neutral on the property it was supposed to fix, so
  there is nothing to keep. Do not propose it a fourth time: the asymmetry with `ruledFaceUV` is real
  in the source and empty in effect.

The sphere face's own emission is where to look next. Its loop still carries an arc ending at
(0, 3.999882919988047, −3.000156100336764) — a point ON the sphere but 1.56e-04 off the rim — while a
sibling arc ends at (0, 4, −3) exactly. One of the two section arcs is emitted to a sampled vertex and
the other to the solved one, which is the asymmetry to chase.

**3. `torus − box (figure-eight pinch)` still declines to CSG under the rewire**, though the same case
is exact on the shipping path.

The `fallbackDebt` ratchet is unmoved at 5 / 3 / 38: no door has closed yet. Stage 2 lands when these
three are green, and not before.

## Stage 2, continued: what the tangent pinch actually is (2026-09-04, later)

Recording the measurements, and the approaches that did NOT work, so neither is re-derived.

**First, a correction to the section above.** "Three failures" was measured on a state that was
PATHOLOGICAL, not clean: at that point one island was being split into 130 000 arcs (see below), the
figure-eight rows were reaching CSG, and CSG clears the faceted budget — so they "passed" for the wrong
reason. It is not a baseline and nothing should be compared against it. The two clean like-for-like
measurements, both from complete runs of `go test ./kernel/...` under the rewire with 34 packages
reporting, are:

| | leaf failures | `TorusFigureEight` | `StayExact` | `VolumesMatchOCC` |
| --- | --- | --- | --- | --- |
| before the section fix (`05055ffd`) | 6 | 38.84 s | 85.37 s | 107.41 s |
| after it, with the welder fix (`1bb39022`) | 6 | **0.07 s** | **15.68 s** | **23.26 s** |

The count is UNCHANGED and the runtimes are 5–6× better. Which rows fail shuffled — `StayExact`'s
axis-parallel pinch now passes and a `VolumesMatchOCC` row now fails — but nothing regressed in
aggregate, and the shipping path (rewire parked) is green throughout with those same rows EXACT.

**The looped split is fixed, by solving an incidence instead of refusing it.** `geom.CurveTouches`
could bracket a meeting but never refine into it: an alternating coordinate descent stalled 3.1 mm
short of a circle meeting a chord and a box-shrinking grid stalled 1.2 mm short, both reporting no
meeting at all, because a TRANSVERSAL crossing makes a V-shaped valley and anything stepping both
parameters together drifts along its wall. Nested golden section — an outer 1-D search whose objective
is an inner closest-point search — lands on it at 4.6e-15 and is safe for the tangential shape too.
With that working, `islandStraightHits` places an island's meeting with a straight imprint and splits
both sides there, and the clearance half of `islandContactOK` is deleted: a decline replaced by a
solve. The sphere cap now closes, every vertex exact.

**`TorusPlaneSection` was emitting two DEGENERATE arcs at the tangency**, and they were most of what
made the figure-eight look hard. Where the plane grazes the tube the two roots of w(v)=±1 coincide, so
one span has zero width and both its branch arcs are the tangency point repeated — four curves where
there are two lobes. Fed in as imprints they are closed curves of zero extent: the meeting solver
reported 4225 meetings between them at separation zero, and the caller split on every one. Dropping an
arc that spans no length took the cut from **30.89 s to 0.05 s** and `kernel/brep` to 31.9 s, under its
33 s baseline.

**What is left is one pairing decision.** With clean input the two lobes' pinch is evaluated 1.07e-07
apart by the two branches — `u(v) = Φ ± arccos w` is ill-conditioned there — against a 1e-07 weld grid,
so they round to adjacent cells and stay two vertices. The two cells then share the sliver edge between
them, the shared-edge dissolve cancels it, and the boundary trace joins the lobes into one circuit: one
lid where there are two. There are NO ties in the angular walk; it never gets a choice. The decision is
at the weld.

Two attempts at it, both measured, neither kept:

- **Give `seamWelder` the 8-neighbour search `welder3` has had since #879.** It is a real defect that it
  lacks one — a cell-exact lookup leaves coincident points unmerged whenever they straddle a cell
  boundary, and which side they fall is an accident of where the grid lies. It welds the pinch into one
  vertex, and then the TORUS chart declines instead (`closedSurfaceSplitFaces`), because a 4-valent
  tangential vertex is exactly what the angular rule cannot resolve. Necessary, not sufficient.
- **Split a vertex shared by cells that share no edge**, the chart analogue of ADR-0047's per-disk
  duplicates. Implemented as a union-find over edge-adjacent cells and a per-region vertex id: **58
  failures** in `kernel/brep`. Cells touching at a vertex are not always two regions — `nextByAngle`
  was built for the Steinmetz pinch, where the boundary genuinely passes through — so forcing the split
  is wrong. If this is revisited it must be conditioned on the contact being TANGENTIAL, which
  `geom.CurveTouches` can now report and does not yet.

The nesting half of `islandContactOK` was also removed and reverted: it fixes nothing measurable on its
own, so it is not carried.

## A note on measuring this gate

Three claims about the rewire's cost were made during this work and two were wrong. Both errors are
easy to repeat, so they are recorded as method rather than as history.

**Count leaf cases, never `--- FAIL` lines.** Go prints a line for the parent test AND for each
subtest, so a table-driven test with two bad rows reads as three failures. "Nine" was that.

**A measurement of a broken state is not a baseline.** The "three" above was taken while one island was
being split into 130 000 arcs; the rows that appeared to pass were reaching CSG, which clears the
faceted budget. Comparing a later, correct state against it manufactured a regression that did not
exist — and then two bisects were spent hunting a cause, both coming back negative because there was
nothing to find.

**The gate to compare is the SHIPPING path.** The rewire is an instrument for sizing a deletion, not
what anyone runs. A change can be right and green on the shipping path while the rewire count stays
put, which is exactly what happened here: the shipping suite is green with the figure-eight rows exact,
and the rewire still has its six.

## The looped split is not an axis quirk: it is the pole seam, half the time

`TestLoopedSplitHalvesACapBySymmetry/halve_by_y=0` reads like one bad orientation. It is not. A sphere
cap cut by a plane THROUGH ITS POLE leaves two open edges for about half of all cutting orientations,
and which half flips with the side kept:

| cut normal | open edges |
| --- | --- |
| `(1,0,0)` | 0 |
| `(-1,0,0)` | **2** |
| `(0,1,0)` | **2** |
| `(0,-1,0)` | 0 |
| `(1,1,0)` | **2** |
| `(2,1,0)` | 0 |

So `x=0` passing and `y=0` failing is a coin toss, not a property of either axis — and the test
happens to sample one of each.

**What the two open edges are.** Both faces split their SHARED section one sampling step from the pole,
but on DIFFERENT meridians. Measured on `(0,1,0)`:

- the cut plane's loop carries a sliver from `(0.0613577, 0, -4.9996235)` to the pole `(0,0,-5)`;
- the sphere patch's loop carries a sliver from the pole to `(0, 0.0613577, -4.9996235)` — a quarter
  turn away, on the chart's artificial SEAM.

The sphere chart emits a run along its own seam adjacent to the pole as a real meridian arc
(`emitSeamRun` builds one whenever the run's ends differ), and at the pole they differ by one sampling
step. The receiving plane knows nothing of that seam and splits the section on its own sampling
instead, so the two slivers never pair.

**Why the pole makes it unavoidable as currently placed.** `placeSeams` puts the longitude seam in the
widest gap of the imprint's longitudes, which works everywhere except at a pole — where every longitude
meets, so the seam ALWAYS touches an imprint that passes through it. The fix is therefore not a better
seam placement: it is that a run along the seam ENDING AT A POLE bounds nothing and must not be emitted
as an edge, exactly as `poleSegments` already says of the pole segment itself ("it bounds no geometry
and welds to nothing"). That reasoning is in the code and the emission does not follow it.

Not attempted here — recorded so the next attempt starts from the table rather than from one subtest
name.

## The pole seam, solved — and the paragraph above corrected (2026-09-05)

The hypothesis that closes the previous section is **wrong**, and usefully so: the seam run at the pole
is not something the emission should suppress, it is something the sampling should never have created.
Instrumenting the sphere chart's segment set for the `(0,1,0)` cut shows the imprint arriving at the
pole and then doing this:

```
seg kind=imprint (1.570796, -1.546253) -> (1.570796, -1.570796)   # down the u = π/2 meridian, onto the pole
seg kind=imprint (7.853982, -1.570796) -> (4.712389, -1.546253)   # a HALF-TURN leap in u, at v = -π/2
```

The section curve passes exactly through the pole, where longitude names no direction. `sampledPolyline`
nevertheless carried the pole sample's azimuth forward by continuity (`unwrapAzimuthNear`), so the two
samples flanking the pole became **one segment spanning π in u along `v = −π/2`**. That segment crosses
the placed seam. The boundary walk then follows it out to the seam and up it — which is the seam run
`emitSeamRun` was faithfully turning into a meridian arc. The emission was reporting the defect, not
causing it; suppressing it there would have hidden a wrong arrangement behind a right-looking loop.

**The rule.** At a parametric pole the surface collapses to a point, so `u` is free and continuity may
not choose it. A pole sample takes the azimuth of its NEIGHBOUR — separately on each side — so each
half of the imprint reaches the pole ON ITS OWN MERIDIAN and stops there, and the chart's own pole
segment bridges the two. That is exactly what `poleSegments` exists for. The 3-D geometry is untouched:
`point3(u, ±π/2)` is the same point for every `u`, so this is a re-parameterisation, not a nudge.

Implemented in `loopFrame` (`sampleChartPoints` + `anchorPoleEnds`), so it holds for every loop-framed
chart with a singular point — a sphere's poles and a cone's apex alike, not a sphere special case. The
degeneracy test is the existing scale-free `sampleOnPole`, so no recognizer and no tolerance constant is
added.

**Cost.** Shipping path `./kernel/... ./archguard/`: green. Under the rewire the leaf failures go
**6 → 5**; `TestLoopedSplitHalvesACapBySymmetry` is gone and nothing else moved. The remaining five are
all the torus figure-eight rows (the tangent-pinch pairing of the previous section).

Corpus: `TestSphereCapCutThroughItsPoleClosesAtEveryOrientation` takes all four axis-aligned removals,
because the defect took half of all orientations. Without the fix it fails 2 of 4 under the rewire and
1 of 4 on the shipping path — the coin toss, pinned.

## The figure-eight: the solve was moving a point that was already exact (2026-09-05)

The torus figure-eight was the rewire's last blocker, and the previous section's reading of it — an
ill-conditioned formula the two lobes evaluate differently — named the right mechanism in the wrong
place. Three defects, all of them one rule broken three ways: **decide an incidence once, and never with
a formula where the exact answer is already in hand.**

**1. The solve moved an exact point.** `TorusPlaneSection` at offset `R−r` returns the two lobes as two
`SpiricArc`s that both begin and end at exactly `(0, 3, 0)`: `5 + 2·cos(π)` is 3 to the last bit, so
`w = 1` exactly, `spiricCosineAtLimit` fires, `arccos(1) = 0`. The section is exact. `arcPairTouches`
then solved the meeting numerically, converged to within `arcEndFraction` (1e-6 of a span) of each end —
which at a pinch is the worst place to evaluate `u(v) = Φ ± arccos w` — and replaced the two exactly
equal ends with

```
meet0 = (0, 3.0000000000000009,  5.2768006608014108e-08)
meet1 = (0, 3.0000000000000009, -5.4156397588656084e-08)
```

**1.07e-07 apart**, which is the number the previous section attributed to the section itself.

**2. The welder and the boundary-edge filter used two tolerances.** `seamWelder` merged within
`seamWeldGrid` (1e-7); `keptBoundaryEdges` dropped a degenerate edge only within `arrTol` (1e-9). A step
between the two is an edge from a vertex to ITSELF. It then either cancelled against its twin and merged
two regions into one self-touching loop, or chained as a one-edge loop and emitted a phantom face — one
failure per kept side, which is why the two normals failed differently. `weldsToOneVertex` asks the
question once, separating the genuine full-wrap edge (an uncut rim circle, ends a whole period apart) by
its (u,v) span rather than by the welded indices alone.

**3. Neither candidate is always right, so certify against the geometry.** Snapping to the ends alone
fixed the axis-parallel pinch and broke the OBLIQUE figure-eight, where the arcs' own ends come from a
numeric root of `|w| = 1` and the two meetings straddle the true tangency by ±5.2e-08. `bestTouchParams`
takes whichever pair — the solved parameters or the arcs' own ends — puts the two arcs closer together.
It is a runtime certification, not a case split, which is what the ground rules ask of a branch choice.

**Cost, measured on the four torus figure-eight rows** (OCC as the oracle):

| row | before | now |
| --- | --- | --- |
| `torus ∩ box (pinch)` | 275.28 vs 114.89 — exact, WRONG (the complement) | 112.53, 3 faces, exact ✓ |
| `torus − box (pinch)` | 112.53 vs 279.90 — exact, WRONG | 271.27 ✓ volume, faceted (CSG) |
| `torus ∩ box (oblique)` | 239.84 vs 151.90 — exact, WRONG | 149.80 ✓ volume, faceted (CSG) |
| `torus − box (oblique)` | 239.93 ✓ volume, faceted (CSG) | 239.93 ✓ volume, faceted (CSG) |

**Every wrong-shape body is gone**; what is left is the fallback doing its job. Leaf failures under the
rewire go 5 → 3, and all three are now `TestCurvedBooleansStayExact` — a demotion, not a wrong answer.
`TestHalfSpaceCutTorusFigureEight` and both figure-eight volume-oracle rows pass.

**What this exposed next, fixed in the same pass.** The three demotions came out of
`CurvedBooleanWithDiagnostics` returning `ok=false` with **no diagnostic recorded**. The guarded entry
has four exits and three of them reported; "no exact path claims this configuration" returned silently,
so a boolean with a curved operand could fall to triangle soup with nothing downstream able to say why.
`declineCurvedExact` names it (`CodeBooleanNoExactCurvedPath`), and stays silent for an ALL-PLANAR pair,
where the planar B-rep path is exact and the decline costs nothing. `fallback-sites` rises 29 → 30, the
same shape of rise as `CodeBooleanAnalyticInvalid` before it: a degradation that was already happening,
now reported.

Corpus: `TestIslandTouchKeepsTheExactPinchOfAFigureEight` (the solved meeting equals the arcs' own shared
endpoint to a few ulps of the torus radius), `TestBestTouchParamsIsNeverWorseThanEitherCandidate` (the
certification's whole contract, on both sections), `TestKeptBoundaryDropsAnEdgeFromAVertexToItself` and
`TestKeptBoundaryKeepsAFullWrapEdge` (the two halves of the tolerance rule),
`TestABooleanWithNoExactCurvedPathDeclinesByName` and `TestAnAllPlanarBooleanDeclinesSilently` (the
decline and its exemption). Each fails without its fix.

## The pinch vertex, and the two defects behind the oblique rows (2026-09-05, later)

**The radial fan's connector is the loop, not the face.** With the figure-eight solved, both axis-parallel
rows still declined — and not for want of geometry. The body the general path built was closed, manifold
and correctly oriented, and `Validate` refused it anyway:

```
Euler characteristic V−E+2F−L = 1 is inadmissible for a closed solid of 1 shell(s)
```

It is right to refuse it. A torus cut by a plane tangent to its inner equator is genuinely PINCHED: the
section is two lobes meeting at one point, so the boundary has a non-manifold vertex there. ADR-0047's
radial sew exists precisely to resolve that — at a vertex it partitions the incident edge-groups into
radial disks and mints one vertex per disk. It did not fire, because `groupFans` joined two groups when
some FACE used both. Here the two lobes bound one torus face through TWO of its loops, so the face
identity welded the pinch onto a single vertex and the Euler count came out odd.

The connector is the loop: a face whose boundary passes through one vertex on two of its loops is
pinched there, and the two loops are two separate fans on that face — exactly as two faces kissing at a
point are two fans on the body. Keyed by `(face, ring)`, the partition only ever REFINES: two loops of
one face are still unioned transitively wherever another face genuinely joins their groups, which
`TestGroupFansStillUnionsThroughAnotherFace` pins alongside the manifold corner.

Both axis-parallel rows then land **exact**:

| row | ours | OCC |
| --- | --- | --- |
| `torus ∩ box (figure-eight pinch)` | 114.886320 | 114.886326 |
| `torus − box (figure-eight pinch)` | 279.897856 | 279.897854 |

Leaf failures under the rewire go 3 → 2, both of them the OBLIQUE figure-eight.

**What the oblique rows are waiting on — two defects, both reduced to a one-liner.** Neither is the
figure-eight; both are general and both are worth their own fix.

*1. `Face.RangeBox` under-reports a trimmed curved face.* `computeRangeBox` bounds a body by its
vertices, its edges and its BOUNDARYLESS faces — so a trimmed curved face contributes only through its
boundary edges, and a face that bulges past them is invisible:

```
sphere      box = {-5 -5 -5}..{5 5 5}
hemisphere  box = {-5 -5  0}..{5 5  0}     <- the cap reaches z = -5
cylinder    box = {-3 -3  0}..{3 3 10}     (its rims bound it)
torus       box = {-7 -7 -2}..{7 7  2}     (boundaryless)
```

This is what breaks the oblique composition. `curvedConvexIntersect` composes a half-space cut per box
face; the first cut is correct (151.898715 against OCC's 151.898735), and the NEXT plane — the box's far
wall at z=20, which touches nothing — builds its bounded half-space from a range box that is flat in z,
so the tool it subtracts is not the tool it should be. OCCT answers the same question with
`BRepBndLib::Add`: the surface's bound over the face's UV window, enlarged by the face tolerance.

*2. A no-op difference flips one lid's sense on the oblique band.* Reduced to this, with the tool a block
far outside the body:

| body | before | after |
| --- | --- | --- |
| torus, sphere, hemisphere, pinch band, two-oval band | unchanged | unchanged |
| **oblique band** | 151.898715 | **239.841783** (the complement) |

The rebuilt body is topologically identical — same faces, same edges, same loop counts — and one planar
lid's `Reversed` flag differs. The material-side votes locate it. Read straight back off the STORED
body, before any reorientation runs:

```
lid A   votes -31
lid B   votes +31      <- the two lids wind oppositely
torus   votes  +0, +0
```

Both lids are marked `Reversed`, and their loops wind against each other. The body is internally
inconsistent as built — `BodyGeometryProperties` integrates from the face flags and the surface normals,
so it still reported the right 151.898715, but every reader that takes a face's region from its
TRAVERSAL (`senseFromLoopWinding`, the flux classifier) reads one lid inverted, and the rebuild then
stores the flag it read.

Two candidate causes were tested and refuted, so the next pass need not repeat them:

- *Not the two-colouring's free bit.* `curvedOrientationFlips` flips whole FACES, which cannot produce a
  one-loop asymmetry: flipping the torus face would move both of its loops together.
- *Not the lid filing.* The oblique section's two lobes are side by side, not nested — sampled into the
  cutting plane they share the whole u range and split v at the tangency, `[-4.66,2.00]×[-6.87,0]` and
  `[-4.66,2.00]×[0,6.87]` — so two lid faces is the right answer and a containment rule changes nothing.
  (Implemented as `lidLoopGroups` and reverted: it never fired, and an unexercised rule is not a fix.)

What is left is the emission itself: one of the two lobe loops is emitted with the material on its
RIGHT. `materialSideVotes` already answers that question per loop; the producer does not ask it.

Recorded rather than fixed, so the next pass starts from the measurement.

## The oblique figure-eight was one arccos away, and the rewire is now GREEN (2026-09-05, later still)

The previous section's reading — "the two lids wind oppositely as built" — was a symptom, and chasing it
through the emission was the wrong direction. The cause is one line upstream, and it is the same defect
as the pinch's, in the other root solver.

`spiricTubeSpans` finds the tube angles the plane reaches by solving `|w(v)| = 1`, which
`harmonicRoots` turns into `cos(v − atan2(B, A)) = D / amp`. At a TANGENCY `|D| = amp`, the two roots
coincide, and arccos is infinitely steep there: a ratio short of 1 by half an ulp put the double root's
two halves **1.5·10⁻⁸ apart in v**, so each lobe of the section came back as an arc that misses closing
on itself by **1.03·10⁻⁷** — the number that has been turning up all evening.

A hair in the section is not a hair downstream. `stitchKeyFor` welds a loop edge's two ends and calls
the edge CLOSED when they weld together; 10⁻⁷ apart they do not, so a near-closed lobe was stored as an
OPEN edge whose direction is recovered on read-back by inverting the curve at two endpoints 10⁻⁷ apart.
That does not round-trip, and one lobe's loop came back wound against its own material — measured, the
lids read −31/−31 during construction and −31/**+31** when read straight off the stored body.

The fix is `spiricCosineAtLimit`, which already exists for exactly this and was already applied to `w`:
where the exact value is in hand, do not feed the ill-conditioned formula a near-value. Routing the
harmonic root's cosine through it makes the two roots coincide EXACTLY, so each lobe closes to 2·10⁻¹⁶
and both lobes meet at one point. No new tolerance, no new recognizer, one shared helper.

**Every torus figure-eight row is now exact against OCC:**

| row | ours | OCC |
| --- | --- | --- |
| `torus ∩ box (figure-eight pinch)` | 114.886320 | 114.886326 |
| `torus − box (figure-eight pinch)` | 279.897856 | 279.897854 |
| `torus ∩ box (oblique figure-eight)` | 151.898715 | 151.898735 |
| `torus − box (oblique figure-eight)` | 242.885461 | 242.885450 |

and the no-op difference that rebuilt the oblique band as its complement returns it unchanged.

**The rewire is green.** `go test ./kernel/...` with `HalfSpaceCut` rewired to
`Boolean(Difference, body, BoundedHalfSpace(plane, box))`: 36 packages, **zero failures**. The shipping
path with the rewire parked, plus `./archguard/` and `./model/...`, is green too. The instrument that
has measured this stage since 2026-09-03 — 33 failures at the start, 6 after ADR-0063, then 5, 3, 2 —
now measures nothing, which is the gate stage 2 was waiting on.

## The range box did not bound the body (2026-09-05, last of the pair)

`Body.computeRangeBox` swept the vertices, the edges and the BOUNDARYLESS faces. A trimmed curved face
contributed only through its boundary edges — and a face can reach past those:

```
sphere      box = {-5 -5 -5}..{5 5 5}
hemisphere  box = {-5 -5  0}..{5 5  0}     <- the cap reaches z = -5
cylinder    box = {-3 -3  0}..{3 3 10}     (its rims span its whole azimuth)
torus       box = {-7 -7 -2}..{7 7  2}     (boundaryless)
```

A hemisphere's only edge is its equator, so the body reported a box of ZERO height. That is what broke
the oblique composition before the tangency fix landed: `curvedConvexIntersect` composes a half-space
cut per box face, and the box's far wall — a plane that touches nothing — built its bounded tool from a
range box flat in one axis.

No rule about the EDGES can fix this: the equator bounds the upper hemisphere and the lower one alike.
The face's CHART says which side it is on, which is what ADR-0063 put on the face, so the sweep is over
the chart's (u, v) window. The window rather than the trim itself, because the surface over the window
encloses the surface over the trim — a non-rectangular patch is bounded generously rather than missed —
and a face carrying a chart no longer needs the boundaryless sweep at all, which is the same answer for
a whole surface and a far tighter one for a patch.

Measured: the hemisphere becomes `{-5,-5,-5}..{5,5,0}`, and a small cap above z = 4 still bounds to
`{-3,-3,4}..{3,3,5}` rather than ballooning to the whole ball.

**Two honest limits.** This stays a SAMPLED bound, of the same kind the edge sweep already produces — a
torus band's `y` came out 6.982 against a true 7 on the shared 8×8 grid. The certified-tight box, which
reads each surface's interior extrema in closed form through `geom.SurfaceAxisCriticalPoints`, is
`query.PreciseRangeBox`; the doc now points at it. And the fix bites where a face carries a chart, which
is the general (u, v) path — the analytic half-space pipeline sets none, and that pipeline is what stage
2 deletes rather than something to retrofit.

## Stage 2 lands: the half-space cut IS a difference (2026-09-05)

`HalfSpaceCut` no longer has a pipeline of its own. It builds the plane's positive side as an ordinary
solid bounded to the target's box and hands it to the general boolean:

```go
cut, err := Boolean(Difference, body, BoundedHalfSpace(plane, body.RangeBox()))
```

which is what OCCT's `BRepPrimAPI_MakeHalfSpace` does and what ADR-0062 said this should become. The
per-primitive dispatch above it — a cylinder fast path, a cone fast path, a torus fast path with its own
three-way spiric switch, then `generalHalfSpace`'s `splitFaceByPlane` ladder — is **deleted**, along with
the looped split, the lid chainer, the two-oval band builder and the axis-parallel figure-eight
recognizer.

**Measured:** 1 008 lines removed against 117 added, across 19 files; two files deleted outright and a
third (`curved_halfspace_general.go`) reduced to three shared helpers that now live under a name that
says what they are (`curved_same_point.go`). Three ratchets FELL and were lowered in the same commit:

| ratchet | before | after |
| --- | --- | --- |
| `geomSwitchDebt["kernel/brep"]` | 93 | 82 |
| `literalLineageTags["kernel/brep"]` | 61 | 54 |
| `kernelNetDeltaPin["type-assertions"]` | 765 | 754 |

**What did NOT move, and should not have.** `fallbackDebt` stays at 5/3/38. Its three numbers count the
CSG and mesh ENGINES and the doors into them, which stages 5, 6 and 7 close; stage 2 deletes an analytic
pipeline, not a faceted one. Reporting it as progress would be reporting the wrong number.

**Two obsolete tests retired, one rewritten.** `TestTorusAxisParallelFigureEightGuards` guarded a
recognizer that no longer exists. `TestClipParamsMultiArmHyperbola` tests `ruledUV.clipParams`, which is
alive through the general boolean's `newConeUVSolid`, and only reached it through the plane-based
constructor; it now goes through the surviving one, so it no longer holds a dead path up. The looped-split
acceptance test keeps its geometry and loses a comment naming a deleted function.

**Follow-up, named here so it is not forgotten.** Twenty files still carry the `curved_halfspace_` prefix
while holding the general (u, v) chart machinery the boolean uses — the arrangement's five phases, the
ruled and torus (u, v) models, the side interface. The names are now wrong. That is a mechanical rename
and it belongs in its own commit, not buried in this one. (Done below.)

### One defect the deletion exposed: a parity test answering on a boundary

`./model/...` — which the rewire instrument never ran, because the instrument was
`go test ./kernel/...` — caught `TestNativeRevolveTorusHalfSpaceCutsAreExact`. The figure-eight tangent
cut fell to CSG for a torus about the **Y** axis and passed for one about **Z**, on geometry identical up
to a rotation. Reduced:

| torus axis | tangent on | exact? |
| --- | --- | --- |
| z | +y, +x | ✓ ✓ |
| y | +z | ✓ |
| y | +x | **✗** |
| x | +y, +z | **✗ ✗** |

The section is identical in every row — two lobes, each closing on itself to 4.9e-16, meeting at one
point. What differed was `islandsWalkNestedOrApart`, the gate that refuses two islands "not nested and
not apart". It asks `pointInRing2D` of every sample of one lobe against the other, and the lobes' shared
TANGENCY is a sample of both. An even-odd parity test has no answer ON a boundary: it returns whichever
side the ray happened to fall, so the shared vertex read "inside" for some rotations and "outside" for
others, and the gate refused the geometry it was rotated from.

`ringStraddles` now skips a point coinciding with one of the other island's arc ENDS — which is exactly
where `splitIslandsAtTouches` solved the two to meet. One incidence, decided once, not re-decided by a
parity test that cannot see it. A genuine crossing puts many points inside the other ring, so the gate
keeps its purpose, and `TestRingStraddlesIgnoresASolvedMeetingPoint` pins both halves.

The corpus takes all six axis/normal pairs, because the defect was a coin toss.

### The residue the deletion left, and the names it left wrong (2026-09-05, same day)

Two things stage 2 left behind, both now done.

**Production code kept alive only by tests of the path that was deleted.** `unused` cannot see it — a
test counts as a use. Running it with the tests excluded (`golangci-lint run --tests=false
--enable=unused`) names it exactly, and three whole files fell out: `curved_halfspace_ruled_face.go`
(the plane-based ruled wall split), `curved_halfspace_looped.go` (the loop-by-plane splitter it was the
only caller of) and `curved_halfspace_torus_oblique_general.go` (the oblique spiric span helpers),
together with `torusSpiricSection`/`spiricBranches` and two helpers of the old general stage. Their unit
tests went with them; the three BEHAVIOUR tests that happened to live in the oblique file — they cut a
tilted torus and check the result — stayed, because they test `HalfSpaceCut`, not its old innards.

**The names.** Twenty files carried a `curved_halfspace_` prefix while holding the general (u, v) chart
machinery, so they are renamed to what they are: the arrangement's five phases to `curved_uv_*`, the
ruled and torus models to `curved_ruled_uv*` / `curved_torus_uv*`, the cut-cylinder chart to
`curved_cut_cylinder_*`, and the primitive recognizers to `curved_*_solid_params` / `curved_*_side_band`.
`curved_halfspace.go` keeps its name: it IS the half-space cut. So do the behaviour tests named after
the cuts they drive. `toleranceDebt`'s keys are file paths and were renamed with them — same budgets,
no ratchet moved.

**Named follow-up: a second, unused entry to the radial sew.** `radialSew` and its three helpers
(`extractEdgeGroups`, `indexUsesByGroup`, `sortedPairKeys`) plus `sewPlan.useGroup` are reachable from
tests alone: `buildCurvedStitchPlan` inlines the sew instead of calling it. They are NOT
interchangeable — `extractEdgeGroups` walks `sortedPairKeys`, the stitch walks first-encounter order,
and the group index is what edge lineage ordinals are built from — so this is a delete, not a merge, and
it wants its own commit and its own reading of what the tests were proving. The same sweep lists
`provenanceOf`, `allEdgesPaired`, `curvedImprint`, `interiorPointOf`, `planeUVContactOK`,
`chartContains` and `eccCap`; each needs the same judgement and none of it is stage-2 residue.

## Stage 3 measured: its charts have landed, its DELETION belongs to stage 4 (2026-09-05)

The stage list reads "(3) sphere and torus charts, deleting the ball-and-rod recognizers". Measured, that
is two things with different readiness, and pairing them was a mistake in the plan.

**The charts are in and load-bearing.** `sphereFaceUV` and `torusFaceUV` are wired into the mixed
boolean as its `sphere` and `torus` buckets, and stage 2 depends on them: the hemisphere and the torus
band a half-space cut now returns are built by those charts, carry their own charts through the stitch
(ADR-0063), and are what `Body.RangeBox` reads. Stage 3's capability shipped as the thing that made
stage 2's rewire pass.

**The deletion is gated on stage 4, and the code already says so.** Removing
`curvedBallRod{Intersect,Cut,Join}` from `curvedExactPaths` costs **24 corpus rows** —
`TestCurvedBooleansStayExact` × 20, `TestCurvedBooleanVolumesMatchOCC` × 2 and four dedicated
ball-and-rod tests — and takes `kernel/ops/boolean` from ~35 s to **938 s**. Two of those rows do not
merely go faceted, they come out WRONG: `coaxial shoulder rod − ball` and `coaxial bi-shoulder rod −
ball` miss the OCC volume.

The decline is one gate, `closedSurfaceUncovered`, and its own comment names the stage:

> A wall, another sphere or a pass face still declines on box overlap: curved-versus-curved contact
> stays with the bespoke recognisers until the crossings are charted (ADR-0061 stage 4).

A ball and a coaxial rod meet along a circle on a sphere and a cylinder — the simplest curved-versus-
curved crossing there is. It is not a sphere-chart gap; it is the crossing bucket, which stage 4 opens.
So the ball-and-rod recognizers move to stage 4's deletion list, beside the 26 of `curvedExactPaths`
they belong with, and stage 3 is complete as a capability.

Recorded rather than forced: widening `closedSurfaceUncovered` to admit a wall would trade twenty exact
rows for faceted ones and two for wrong ones, which is the opposite of a gate.

### The second entry to the radial sew, deleted (2026-09-05)

Named as a follow-up above and now done. `radialSew` was the documented entry to ADR-0047's radial-edge
core, and nothing called it: `buildCurvedStitchPlan` assembles the sew itself as it walks the geometric
edges. The two are not interchangeable and merging them would be a defect — `extractEdgeGroups` walks
`sortedPairKeys`, a SORTED order, while the stitch walks first-encounter order, and the group index is
what edge lineage ordinals are built from. So the duplicate goes and the walk's order becomes what the
`sewPlan` doc now says it is: part of the contract.

Deleted with it: `extractEdgeGroups`, `indexUsesByGroup`, `sortedPairKeys` and the `sewPlan.useGroup`
field that only the dead path filled. `TestRadialSewSurfaceAgnostic` proves a real property — that a
>2-use tangent edge is paired from injected per-face normals, the OCCT `GetFaceDir` contract — and it
was reaching it through the dead wrapper; it now calls `resolveEdgeUses` directly, which is the function
that does the work.

`mapOrderDebt["brep/boolean_radial_edge.go"]` falls 2 → 1: `sortedPairKeys` ranged a map.

## Stage 4 begins: the first curved-versus-curved crossing (2026-09-05)

Sized the way the ground rules ask — one representative case driven to a valid solid before anything is
generalised. The case is the coaxial **plug**: a ball of radius 5 and a rod of radius 3 whose axis
passes through the centre, intersected.

`closedSurfaceUncovered` declined it on box overlap alone. It now pairs the buckets the way the
plane×wall pairing already does — solve the crossing ONCE, in closed form, and write the same curves
into both sides' imprint lists, so the two charts split on identical coordinates and their fragments
weld. The general intersector already answers the pair: `IntersectSurfacesAnalytic(sphere, cylinder)`
takes the parametric×implicit bucket and returns two closed curves, each closing to ~10⁻¹⁵.

Measured, through `brep.Boolean` — BELOW the recognizer list, so it measures the general pipeline and
not the recognizer that still claims this shape first: **3 faces, valid, volume 127.7581 against an
analytic 127.7581** (a cylinder to the crossing at y = 4 plus the cap above it).

**The scope is narrow and named**, because a slice that quietly did more would be the try-ladder this
retirement exists to delete:

- the closed surface must be BOUNDARY-LESS, so every crossing is inside its trim by construction;
- every crossing must come back CLOSED, so it is an island on both charts and each splits by even-odd
  containment alone;
- every crossing must lie strictly inside the wall's band or strictly clear of it. **A crossing with the
  INFINITE ruled surface is not a crossing with the wall** — a rod starting at the ball's centre crosses
  the sphere in two circles and only one is on the rod — and imprinting the other cuts the ball where
  nothing touches it.

`bandPlacement` is now one rule with two span sources: a conic's centre and amplitude in closed form,
or a general crossing walked. `spansOverlap` lost its `pad` parameter, which every caller passed the
same constant for.

**What this slice does NOT do, measured.** The same pair's CUT and JOIN still decline, and the reason is
one step further in: the ball minus the rod is the sphere MINUS a cap, a kept region that is the
complement of its own loop, and `sphereFaceUV.orientLoops` files every kept region as an outer loop —
it never reports `outerless`, which the torus chart does. So the sphere face comes back unbounded-wrong
and the stitch drops it, leaving a two-face body the guard refuses. That is the next slice, and it is a
sphere-chart gap rather than a crossing one. (Done below.)

### The slice completed: the whole coaxial family, through the general pipeline (2026-09-05)

The previous slice left the plug exact and the same pair's CUT and JOIN declining, and named the reason:
the ball minus the rod is the sphere MINUS a cap, a kept region that is the complement of its own loop.
Two fixes, both of them a rule stated for one chart being applied to the property it is actually about.

**`sphereFaceUV.orientLoops` never reported `outerless`.** The torus chart has said for a while that on a
CLOSED surface a single CW loop bounds a dropped island, so the face is the complement of its rings. That
is a property of a closed surface, not of a torus. A cap does not reach this code — its boundary wraps
the longitude, which `wrappingSolidFaces` takes — so the rule applies unchanged.

**`dropArtificialLoops` was gated on v-periodicity, which names the torus rather than the property.** A
loop made entirely of artificial seam edges bounds nothing, because the surface is closed or degenerate
across every edge of it. A torus's complement has the whole parameter rectangle as such a loop; a
SPHERE's complement has the two POLE segments, which `poleSegments` already describes as bounding no
geometry and welding to nothing. Gated out, the ball came back as two boundary-less faces that
`boundedTrims` then dropped, so the difference lost its sphere entirely. The gate is gone; for a ruled
side the seam edges still cancel pairwise, so it stays the no-op it always was, and `dropArtificialLoops`
no longer needs the chart at all.

Measured through `brep.Boolean`, all three ways round, three analytic faces each:

| operation | ours | exact |
| --- | --- | --- |
| ball ∩ rod (the plug) | 127.7581 | 127.7581 |
| ball − rod (a blind bore) | 395.8407 | 395.8407 |
| ball ∪ rod (the stud) | 819.9557 | 819.9557 |

**A method note, because it nearly cost a false result.** The first version of this corpus test was
extended from one case to three by a scripted replacement that silently did not match, so the run that
"passed all three" had run one — and would have reported the two new rows green without executing them.
Every scripted edit that must match existing text now asserts the match, and a test extended to new rows
is read back for those rows in the output before it is believed.

### Contact is contact, whether or not the section stays inside (2026-09-05)

Working toward stage 4's next family turned up a defect worth more than the family: **a sphere
intersected with a box came back WHOLE.**

The closed-surface pairing asked whether a plane section sat wholly inside the receiving face's trim.
A section that crosses the receiver's own edge — which is what happens whenever a box clips a ball
across a corner — answered "no", so no imprint was planned, no gate declined, and the sphere passed
through the boolean untouched. A valid solid of entirely the wrong shape is worse than any decline, and
this one was reachable from `brep.Boolean`, the kernel's own entry.

Two causes, both now fixed:

- **`Face.RangeBox` was empty for a boundary-less face.** A range box is built from vertices and edge
  curves, and a bare ball has neither, so every pairing that screens two faces on their boxes found no
  contact at all. It now bounds such a face by its surface. (The chart-window sweep the BODY's box also
  takes is deliberately NOT applied per face: a window encloses a non-rectangular trim generously, which
  costs a body nothing and would pull spurious pairs into a per-face screen — measured, it took a slot's
  breach off its exact ruling.) The `unboxed` list the boundary index kept for exactly this case is
  deleted with it.
- **The contact test asked the wrong question.** `sphereSectionEnters` now asks whether the section
  MEETS the trim, which is what "does this pair touch" means. The pairing that CARRIES a crossing — clip
  it to the trim once, for both sides, as `wallSectionIsland` already does for a wall's conic — was
  implemented and then withdrawn: it made the sphere∩box and box−sphere cases exact but left box∪sphere
  filing its two kept regions as one face with a hole, integrating to nothing. Trading a decline for a
  wrong answer is the opposite of the point, so what ships is the decline, and the crossing pairing
  waits for the union case to be understood.

Also generalised while here: `closedSurfaceOuterless` — a closed-surface face has no outer loop when
EVERY one of its rings bounds a dropped island, not only when there is exactly one such ring. And
`widestCylinderFace` in the corpus now fails cleanly instead of returning nil, which nil-dereferenced
inside the mass-props query and killed a whole package's run mid-measurement.

### Ruled versus ruled: the crossing-cylinder family, through the general pipeline (2026-09-06)

The same pairing again, one bucket over: two WALLS that cross. `wallOverlapsUncovered` declined every
overlapping wall pair, which is what sent the whole crossing-cylinder, Steinmetz and cone-crossing family
to its bespoke recognizers.

`pairWallWallImprints` solves the crossing once and writes it into both walls' imprint lists, under the
scope the closed-surface pairing already uses: the crossing must come back CLOSED and must lie strictly
inside BOTH bands or strictly clear of them. Measured through `brep.Boolean`, with every recognizer
switched off:

| operation | general pipeline | mesh engine |
| --- | --- | --- |
| crossing cylinders ∩ | 41.0411 (3 faces, exact) | 40.6465 |
| crossing cylinders − | 298.2509 (4 faces, exact) | 296.2500 |
| crossing cylinders ∪ | invalid | 380.7486 |

The mesh figures sit ~1% low, which is the facet deficit on a convex body — the analytic answers are the
tighter ones. The JOIN is not carried and still declines.

**Two gates had to keep declining, and both taught something.** Opening `wallOverlapsUncovered`
wholesale broke an emboss pad riding a chamfer cone and a grazing partial-rim cut. The pad is a pair
`geom.SurfacesApart` already settles — the pairing has to honour that proof exactly as the gate does, or
it declines a boolean over a crossing that does not exist. The grazing rod is the sharper lesson: an
analytic solver that finds NOTHING between two walls whose boxes overlap and which no separation proof
settles has not proved they are clear, so an EMPTY crossing keeps the decline. "Carried" means an
imprint was actually produced.

`TestPartialRimGrazingCutDeclinesObservably` converts to `...TakesTheGeneralPath`: the pairing carries
that cut now, and it comes back a valid closed manifold solid of five analytic faces. That is the
conversion this ADR promised for the decline-asserting tests — the decline was the observation, not the
goal.

**Where stage 4 stands, measured with all 26 recognizers off** (`kernel/ops/boolean`, ~600 s against 31 s
healthy): **38 → 33 failing tests, 49 → 46 leaves.** Closed so far: the coaxial ball-and-rod family, the
crossing-cylinder ∩ and −, the drilled wall, the elliptic-section oracle, the cone-cap crossing. Open:
Steinmetz (all three ops), cone∩cone, cone∩cylinder, partial penetration, the partial-rim corner
junction, near-pinch continuity, the cap and rim crossings, the shoulder ball-rod variants, and every
JOIN of a ruled crossing. Each is a slice of the same shape as these two, and each will surface its own
defects on the way — the ones this session found were all of that kind.

### A zero-length edge bounds nothing, whatever put it there (2026-09-06)

Chasing the crossing-cylinder JOIN — the one operation of that family the wall pairing does not carry —
found two charts dropping degenerate edges only in the case they were written for.

`ruledFaceUV.finalizeLoops` dropped them only when the face was `boundedByApex`, and `ruledUV` dropped
an apex LOOP but no degenerate edge at all. A zero-length straight edge bounds nothing whatever put it
there: left in, it has a single use and the body reads as open — which is what the comment already said
about a cone's apex. A crossing that wraps a rod's azimuth leaves the same thing at a seam. Both charts
now drop them, as `sphereFaceUV` does at a pole and as OCCT's degenerate edges do in a face's wire.

Measured with all 26 recognizers off, `TestTwoCapCrossingCutMembershipMatchesCSG` recovers: **33 → 32**
failing tests (the other name that left the list is the grazing test, renamed rather than fixed).

**The join itself is still open, and it is NOT the degenerate edges.** Reduced: the rod's far rim comes
back as ONE circle on its cap and as TWO arcs on its wall, so five edges never pair. The two sides of a
shared RIM must traverse the same edges, exactly as the two sides of a shared crossing must — the wall's
emission splits its rim at the chart's seam, and the cap does not. The other end of the same rod welds
its rim whole, so the split is not a property of the configuration but of where the seam lands. That is
the next thing to reduce.
