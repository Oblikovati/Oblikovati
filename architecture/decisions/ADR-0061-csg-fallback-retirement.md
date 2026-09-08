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
[#2153](https://github.com/Oblikovati/Oblikovati/issues/2153)) — **DONE 2026-09-07**, see "Stage 4's
deletion" below; (5) planned as a chart for freeform faces ending the pass bucket — **the premise did
not survive measurement**: the pass bucket is already empty and the refusals come from the INTERSECTOR,
see "Stage 5" below, where the folded ruled∩quadric window landed and the torus family is what is left;
(6) failure becomes local, so one bad face no longer discards a whole analytic body — **DONE**;
(7) delete the engines ([#2251](https://github.com/Oblikovati/Oblikovati/issues/2251)) — **DONE**.

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

### One drop, one place (2026-09-06)

The degenerate-edge drop had grown three copies — `sphereFaceUV.finalizeLoops` was one, `ruledFaceUV`'s
was another (gated on an apex), and `ruledUV` had none. It is one rule about the FACES, not about any
chart, so it now runs once where the faces are built: after `finalizeLoops`, in `trimByImprint` and in
the wrapping emission. Three call sites become one, and every chart gets it — including the planar ones,
which never had it. `TestApexChartDropsTheDegenerateApexEdge` now exercises the shared rule instead of a
copy that no longer exists.

`rejoinAcrossDrop` goes with it: dropping the edge is not enough when the vertex it stood on had already
split a rim into two arcs, because two arcs are not one circle and the cap on the other side traverses
it whole. Only a pair the DROP made adjacent is merged, so a rim genuinely divided by an imprint keeps
its vertex.

**The crossing-cylinder JOIN is still open and the cause is not here.** Its five unpaired edges are the
rod's far rim, whole on the cap and split in two on the wall, plus two zero-length segments that survive
this drop — so they are not produced through `trimByImprint` at all. Which builder emits them is the
next thing to find; the other end of the same rod welds its rim whole, so it is not a property of the
configuration.

### The crossing-cylinder join closes, and it was the degenerate edges after all (2026-09-06)

The previous entry said the join's five unpaired edges were "not the degenerate drop", because they
survived it. They survived it because they were never offered to it: `ruledFaceUV` has its OWN wrapping
emission (`componentFaces`/`faceOf`, which assemble bands), and neither it nor `loopFrame`'s goes
through `finalizeLoops` at all. Two copies of the drop, and the faces that needed it went through
neither.

`dropDegenerateLoops` now runs on every face a trim returns, whichever emission built it — one place,
after both the wrapping return and the contractible loop. With the zero-length edges gone the rod's far
rim welds whole on both sides, and the cap's two-arc split goes with them: the split was the vertex the
degenerate edge had injected, not a seam placement at all. My earlier reading had the two faces the
wrong way round — it is the CAP that had the arcs and the WALL that had the circle — which is what a
face-by-face dump settles and a description does not.

Crossing cylinders are now exact all three ways round, through `brep.Boolean` with every recognizer off:

| operation | ours | exact |
| --- | --- | --- |
| ∩ (the shared lens) | 41.0411 | — |
| − (the drilled cylinder) | 298.2509 | 339.292 − 41.041 |
| ∪ (the cross) | 383.0739 | 339.292 + 84.823 − 41.041 |

The union is inclusion-exclusion to six figures, which is what makes the three one statement rather than
three numbers. **29 failing tests left of the original 38**, 45 leaves of 49.

### The Steinmetz degeneracy, solved where it belongs (2026-09-06)

Two cylinders of EQUAL radius are the one pair the ruled∩quadric closed form declines for a reason that
is not ill-conditioning: its quadratic's two roots COINCIDE at a fold, and the section is not two
azimuth wraps but two planar ELLIPSES crossing at the folds. Subtracting the two implicit forms leaves a
difference of squares, so the section lies in the two planes through the axes' bisectors, and in each it
is an ellipse of semi-axes r and r/sin(half-angle).

`equalCylinderSection` supplies it INSIDE `IntersectSurfacesAnalytic`, which is where the ground rules
put a surface-pair closed form — a Steinmetz solid is a boolean of two cylinders like any other, and the
only thing special about it is that its section has a closed form the generic one cannot express. It
fires on the degeneracy and nowhere else: an unequal pair, or one whose axes are parallel or skew, falls
through to the general form, which `TestUnequalCylindersKeepTheGeneralForm` pins.

All three ways round, through `brep.Boolean` with every recognizer off:

| operation | ours | exact |
| --- | --- | --- |
| ∩ (the bicylinder) | 144.0000 | 16r³/3 = 144 |
| − | 195.2920 | 339.292 − 144 |
| ∪ | 534.5840 | 2 × 339.292 − 144 |

**And four OCCT blend-parity cases came with it.** `simple/K7`, `L1`, `L7` and `N5` were on the pending
list as "result self-intersects: Cylinder×Cylinder blend-flank crossings ~0.03 deep" — two blend flanks
of equal radius are exactly this pair, and the shallow interpenetration was the fold those declined arcs
left behind. `pendingCapabilityCount` falls 109 → 105 and the parity scoreboard's green count rises
127 → 131 simple, 143 → 147 overall. The fifth of that group, `J5`, is a torus through a plane and
stands.

**26 failing tests left of the original 38**, 44 leaves of 49.

### Where the cone crossings stand, and one rule withdrawn (2026-09-06)

Cone crossings mostly carry already. Through `brep.Boolean` with every recognizer off:

| pair | ∩ | − | ∪ |
| --- | --- | --- | --- |
| cone × cylinder | 18.5838 ✓ | 19.1153 ✓ | 370.9736 ✓ |
| cone × cone | **open** | 26.7997 ✓ | 378.6580 ✓ |

Each row is inclusion-exclusion consistent with the operands' own volumes, so the only gap in the family
is the cone∩cone INTERSECT, and it is a weld: the two walls trace the SAME closed crossing curve and
emit it as two different edges, one starting at `(3.2230, 0, 1.3380)` — the curve's own domain start —
and the other at `(2.6732, 1.0401, −0.7897)`.

Reduced: on one chart the loop is one imprint run spanning the curve's whole domain, which
`emitImprintRun` canonicalises to `[lo, hi]`; on the other it is an imprint run covering 92 % of it plus
a SEAM run for the rest, which `closedRunCoversCurve` rightly refuses to call a full traversal, so it
emits the partial arc it actually walked.

A rule for that — a loop whose only real boundary is one closed imprint curve IS that curve, the seam
runs beside it being the chart's bookkeeping — was written, fired on the right loop, and **changed
nothing**: 26 failing tests before and after. Something further on still separates the two edges. An
unproven change does not ship, so it is withdrawn rather than kept for looking right; what survives is
the reduction, which is where the next attempt should start.

### A section that touches nothing is not a decline (2026-09-06)

Partial penetration — a stub whose cap ends INSIDE the other cylinder, on its very axis — declined all
three ways round, and not in the crossing pairing at all. The stub's CAP is a planar face, and a plane
through a cylinder's axis sections that wall in two straight RULINGS. `wallSectionIsland` asks
`geom.AsConic` first and refuses anything that is not a conic, so a section with no contact in it
refused the whole boolean: those rulings sit at radius 3 on a cap of radius 1.5, three units clear of
its rim.

The rule is the one the closed-surface pairing learned two entries ago, the other way about: a section
CLEAR of the pair contributes nothing, whatever kind of curve it is. **The order matters, and the first
attempt had it wrong.** Asking "is it clear?" FIRST cost an emboss pad on a chamfer cone its exact path
— `conicEntersTrimInBand` brackets a branch's in-band window and answers conservatively, and a section
the island rule does carry can still fail that test. Asked SECOND — only where `wallSectionIsland` has
already declined — it turns a decline into a skip and never the other way round.

All three ways round, through `brep.Boolean` with every recognizer off:

| operation | ours | exact |
| --- | --- | --- |
| ∩ (the plug) | 20.5205 | — |
| − (the blind hole) | 318.7715 | 339.292 − 20.521 |
| ∪ (the stub joined on) | 361.1830 | 339.292 + 42.412 − 20.521 |

**22 failing tests left of the original 38**, 43 leaves of 49. The four partial-penetration rows are
gone in one change.

### A boundary loop is never dropped for being unplaceable (2026-09-06)

`groupLoopFaces` files each hole on the smallest outer loop that contains it — and, when none did,
dropped it. Silently. That is the degradation the ground rules forbid, and it costs real boundary: a
component with a loop that WRAPS the azimuth has no meaningful (u, v) area, so an oblique tunnel's entry
crossing read as an unplaceable hole and vanished, leaving the tunnel face with one boundary instead of
two.

A hole the containment test cannot place now goes on the component's largest face. Measured on the
cap-crossing certification fixture — a Ø1.8 tool at 45° through a Ø6 cylinder, exiting the top cap —
the volume goes 269.984087 → **266.671894 against OCC's 266.6720995**, a match to six figures where
before it was 1.2 % out. One corpus leaf recovers with it.

**The face still does not weld, and the reduction is worth recording.** The entry crossing comes back as
ONE closed curve on the wall it pierces and as an arc plus a straight edge on the tunnel's own chart, so
three edges stay unpaired. That straight edge is tagged `segPolygon` — a FRAME edge, the face's own
boundary — not `segSeam`, which is why the rule written for the seam case (a loop whose only real
boundary is one closed imprint curve IS that curve) does not fire here and, tried twice now, changes
nothing. Whatever puts a frame run in that loop is the next thing to find; the volume says the geometry
is already right.

### The seam split left the far piece measured from the wrong side (2026-09-06)

The frame run in the previous entry was real, and it was a symptom. What put it there is one line in
`splitPeriodicSeam`, the fallback that cuts an imprint segment straddling the chart seam when the
crossing solver did not see the incidence.

The split is right about where the seam is crossed: it interpolates the other coordinate and the curve
parameter to the crossing and emits `a → 2π` and `0 → b`. But `b` still holds the value it arrived
with, measured from the side the run LEFT. A run that climbs a hair past the seam ends at u = 6.2955,
so the second piece ran `0 → 6.2955` — one segment across the WHOLE chart at the crossing's v, instead
of the 0.0123-long stub it is.

That spurious near-horizontal cut sliced the kept region into slivers. On the certification fixture the
tunnel's entry crossing then bounded two thin cells rather than the band, one kept and one dropped, and
the boundary walk closed the survivor along the face's own ruling — the `segPolygon` run. Every
symptom above it (the unplaceable hole, the three unpaired edges) hangs off this.

The far end is now re-based onto its own end of the strip: `cu - seam + other`.
`TestSeamSplitKeepsBothPiecesInTheStrip` asserts the invariant directly — neither piece leaves [0, 2π], neither spans more than half the chart — and `TestObliqueTunnelThroughWallAndCapWelds` is the corpus case: the Ø1.8 tool at
45° through a Ø6 cylinder now returns a **watertight** 4-face solid at 266.671894 against OCC's
266.6720995.

The reason this sat undiscovered is worth keeping: the fallback almost never runs. When
`solveSeamCrossings` finds the incidence, `sampleChartPoints` snaps that sample exactly onto the seam
and no segment straddles it at all. The exit ellipse in the same fixture takes that path and is
correct; only the entry crossing, whose incidence the solver missed, reached the split — and the split
had never been exercised by a case that checked its output.

**Stage 4 standing, all 26 recognizers off (2026-09-06): 17 failing tests, down from 22.** The whole
cap-crossing family is gone from the list — single cap, two caps, cone cap — and
`TestCurvedBooleansStayExact` is down to ONE failing row (the rim crossing). What remains, by cluster:

| Cluster | Tests | What is still missing |
| --- | ---: | --- |
| near-pinch (snap mesh, recovered band, cut/join, intersect continuity) | 4 | the tangential-contact family |
| partial rim (corner junction ×3, chained decline) | 4 | corner and second-cut chaining |
| rim crossing (moments, watertight, corpus row) | 3 | the exit crossing straddles the top RIM |
| torus tangent about every axis | 1 | convex compose |
| coaxial ball-and-rod with a SHOULDER (analytic + 4 OCC rows) | 2 | shoulder variants |
| sphere ∩ box | 1 | the curved cap must survive |
| second-bore rim provenance | 1 | naming, not geometry |
| face-interior-point oracle | 1 | a cylinder face yields no interior point |

### The rim crossing, through the general pipeline (2026-09-06)

A tool that leaves a cylinder through its top RIM — so the exit crossing straddles the rim rather than
landing inside a cap — was the hand-written `curvedRimCrossCut`'s case (#1724 slice 2). That recognizer
assembles an "exit chain" by hand: a wall arc plus a rim arc, with a mixed-arc cap bite to match. The
general pipeline reaches the same body once two refusals are lifted, and neither is a special case.

**A crossing that leaves the wall through a rim is CLIPPED to the band, not refused.** The stretch
between the rims is real imprint, and the rim circle is already in the face's own frame, so the
arrangement closes the region the clipped arc opens — the exit chain the recognizer built by hand is
what the frame gives for free. `clipCrossingToBand` brackets each rim crossing on a walk of the curve
and bisects it with the same `bisectRoot` the corner solver uses, so the clip introduces no tolerance of
its own. Closedness moves off `keepCrossingsOnTheWall` and onto the intersector's raw output
(`crossingsClose`), where it belongs: an OPEN curve out of an intersector is a partial answer and is
still refused, while the open arcs the clip itself produces end on rims this pipeline knows about.

**An open imprint may end on a CURVED frame edge.** `openFrameCrossings` skipped every non-straight
frame edge, so a section ending on a disc's own rim circle was invisible: it entered the arrangement as
a chord dangling inside the face, bounded nothing, and the cap came back whole. It is solved by the
conic-against-conic substitution the island rule already uses on such an edge — `conicEdgeCrossings`
now has a sibling, `conicEdgeCrossingPoints`, that returns WHERE rather than how many. Counting and
locating are one solve; a clip that needed the parameters used to re-derive them by a second, polyline
route, which is also why `sectionFaceCuts` found nothing on a disc: `planarRings` gives one point per
edge, and a face bounded by ONE closed circle has a ring of one point and no segments at all.

With both, `brep.Boolean` returns the rim crossing as a watertight four-face solid — notched holed wall,
bitten top cap, whole bottom cap, tunnel — and with all 26 recognizers off `TestRimCrossingCutMomentsMatchOCC`,
`TestRimCrossingCutIsWatertightAndFoldFree` and the `TestCurvedBooleansStayExact` rim-crossing row all pass.

### One incidence solver for every crossing on a face (2026-09-06)

The corner junction — a rod drilled through an ALREADY-NOTCHED cylinder, its exit bite crossing the
notch (EPIC #1738, ADR-0048) — is the case where two imprints on one face bound the kept region
between them. The rod's exit follows the wall crossing over part of the turn and the notch plane's
section over the rest, and the two meet at two triple points.

Three things were wrong, and they compound.

**Two imprints on one face never met.** The arrangement WELDS coincident vertices; it does not split a
segment where another segment crosses it. With no vertex at the triple points the rod's chart kept the
whole wall crossing and ignored the notch entirely, so the tunnel ran on past the notch plane.
`solveImprintCrossings` now co-refines every imprint pair, which is what `planeFaceUV` already does
between its islands.

**The section-plane solver reported none of those meetings.** It works from two curves' section PLANES,
and a ruled crossing is not planar. The general question is different and simpler: both curves lie on
ONE surface, so substituting one curve's parameterisation into the other's incidence makes the meeting a
scalar root. `geom.CurveIncidence` names that condition — a section's plane, a ruled crossing's quadric
— and `curveRootsOnOther` brackets its sign changes on a walk and bisects each.

A crossing names BOTH of its surfaces, which matters more than it looks. Which one is the chart's own
host carries no information at all — every point of the host satisfies it — and which is which depends
on the side asking. Naming only the carried quadric made the crossing's meeting with a section on its own
base unsolvable: the condition was identically zero along the walk, its sign flipped on float noise, and
the roots came back as seven pieces of noise instead of the two triple points.

**The frame solver and the imprint solver disagreed about where the triple point IS.** Frame crossings
went through the section-plane route, which locates the shared point only to the accuracy of its
candidate and then INVERTS it on each curve. On this fixture the wall named the triple point 5e-5 away
from where the notch cap and the tunnel named it. The stitch does not weld across that, and does
something worse than fail: it splits the wall's neighbouring notch edge at the stray vertex, leaving a
zero-length remnant in the loop. Both solvers are now one — `curvePairMeets`, each side's parameter
solved in its OWN walk, paired by the point they evaluate to — and the section route survives only as
the fallback for a TANGENTIAL contact, where nothing changes sign.

`brep.Boolean` now returns the corner junction as a watertight five-face solid. With all 26 recognizers
off, `TestPartialRimCornerJunctionTakesAnalyticPath` and `TestPartialRimChainedCutDeclinesObservably`
pass; the two corner certifications remain, and what they now measure is a TESSELLATION deficit (88 free
edges at default quality, volume 0.98 % low) on a face whose boundary mixes a clipped ruled crossing with
an elliptical arc — downstream of the modelling, and the next thing to take up in this family.

**Stage 4 standing, all 26 recognizers off (2026-09-06, after the corner junction): 12 failing tests,
down from 22.** Gone since the last count: the whole rim-crossing family, the partial-rim analytic-path
and chained-decline gates, and `TestCurvedBooleansStayExact` in its entirety. What remains:

| Cluster | Tests | What is still missing |
| --- | ---: | --- |
| near-pinch (snap mesh, recovered band, cut/join, intersect continuity) | 4 | the tangential-contact family |
| corner junction (moments, tessellation) | 2 | TESSELLATION only — the solid is watertight and 5-faced |
| coaxial ball-and-rod with a SHOULDER (analytic + 4 OCC rows) | 2 | shoulder variants |
| torus tangent about every axis | 1 | convex compose |
| sphere ∩ box | 1 | the curved cap must survive |
| second-bore rim provenance | 1 | naming, not geometry |
| face-interior-point oracle | 1 | a cylinder face yields no interior point |

### An edge's curve spans exactly that edge (2026-09-06)

With the corner junction modelled correctly the certification still failed, and what it measured was
the MESH: 88 free edges at default quality on a body that is watertight and five-faced, the volume 0.98 %
low as a consequence.

The wall's crossing edge — the ruled crossing clipped between the two triple points — stored the WHOLE
closed crossing as its curve. `edgeCurveFor` restricts a circle, an arc, a line, every conic and a
spiric branch to its loop sub-range, and falls through to "stored whole" for anything else; a ruled
quadric arc was among the anything else. So the tessellator, which reads the curve's own domain, meshed
the entire closed crossing for an edge covering a third of it, and the neighbouring face's mesh met it
nowhere.

`RuledQuadricArc.SubArc` restricts natively (it carries its own azimuth range, so the sub-arc is exact
and keeps its kind, which the incidence solver and the analytic integrator both read). The default is no
longer "stored whole" either: a kind with no restriction of its own is wrapped in `TrimmedCurve3`, which
exists precisely to re-present a sub-range over its own domain.

The invariant is now a test — `TestEveryEdgeCurveSpansItsOwnEdge`, over both the corner junction and the
rim crossing — and it immediately found a second instance: an OPEN curve whose loop walks its FULL domain
BACKWARDS was stored forward while `edgeEnds` anchored the start vertex to the loop's first point, so the
curve began at the END vertex. A closed curve is not that case (its edge carries one vertex and the use's
reversed flag orients it), which is what `storedWhole` now distinguishes.

Both corner certifications pass with all 26 recognizers off. The hand-written recognizer, for comparison,
avoids the whole question by storing that crossing as a 9-point `geom.Polyline` — a faceted edge on an
otherwise analytic solid, which is exactly the kind of silent degradation stage 4 exists to remove.

**Stage 4 standing, all 26 recognizers off: 10 failing tests, down from 22.**

### Recognising the section, and probing a wrapping band (2026-09-06)

Two gaps that are not modelling defects at all — the bodies were right — but that kept a per-face gate
from certifying them.

**A section that IS a circle must come back as one.** The one general intersector builds every
ruled∩quadric section as a `RuledQuadricArc`, and a cylinder through a sphere's centre cuts it in two
CIRCLES. Delivered as general ruled arcs they lose everything downstream that reads the curve's kind:
the per-face oracle's band walk, the tessellator's conformal rim stations, a bore rim's provenance name,
every conic clip. `canonicalSection` asks the CURVE what it is after the general path has built it —
sample the full sweep, certify one circle through every sample — so this is a recognition, not a
type-pair fast path beside the intersector. The circle's normal takes a sign fixed by a total order on
its components, so the two operands, which build the same section from opposite bases, derive the SAME
circle, seam included.

**`FaceInteriorPoint` declined every seam-wrapping face,** and that decline was right while the probe
was a guess. What makes the probe safe is not the probe but the certification under it: whatever uv the
band or cap rule proposes, the point is returned only when `brep.PointInFaceTrim` — an independent
classifier, not these loops' own polygon — agrees it is on the face. A probe landing in the band the
operation discards fails that and still declines. So the gate never gains a probe it cannot stand
behind, and it stops skipping every ordinary bore wall and rod tunnel the general pipeline builds.

**Stage 4 standing, all 26 recognizers off: 9 failing tests, down from 22.**

### A rim's name must not assert which path built it (2026-09-06)

`TestSecondBoreRimIsProvenanceNamed` asserted that a bore rim's key contains `drillwall` — the token the
DRILL RECOGNIZER mints for a bore wall. The general pipeline inherits the tool cylinder's own face key
instead and names the rim `cylinder:f#2/curvedbool:x#0/slab:face#0`, which is the same thing said with
the parents it actually has. The property under test is the SHAPE of the name — build-order-independent,
derived from the two generating faces — and it now asserts that, so it will still hold when stage 7
deletes the recognizer whose token it named.

Two real naming gaps surfaced while checking this, and neither belongs to the retirement:

- **The mixed stitch leaves a CURVED rim on a build-order ordinal.** It runs the planar imprint's naming
  (`planarStitchNaming`) with the curved relineage OFF, which names every planar intersection edge well
  and leaves a rim no planar imprint generated with `brep:edge#N`. Turning the relineage on is not the
  fix: it also renames the originals the planar path deliberately keeps on ordinals, and re-derives the
  planar names from the BUILT faces rather than the imprint's own parents — measured, six sew-golden
  signatures and two naming tests move. The fix is to name the fallback groups from their bordering
  faces at the hook, where the ordinal is minted, and it wants its own change.
- **Two bores of one plate give their walls the SAME key** — `brep:drillwall#0` twice on the recognizer
  path, `cylinder:f#2` twice on the general one. Ambiguous key resolution is an error by the ground
  rules. This is pre-existing and identical on both paths.

### A closed-surface section is clipped to its receiver (2026-09-06)

A sphere intersected with a box is the most ordinary curved boolean there is, and it declined. The
closed-surface pairing kept a section only when it lay WHOLLY inside the receiving planar face's trim,
and every section circle a box face cuts from a sphere leaves through that face's own edge. The earlier
entry recorded the honest decline; this is the capability.

Three changes, and the third is the one that mattered.

**The receiver promotes on MEETS, not on inside.** `closedSurfaceSectionEntersFace` asked the island
question, so a crossing section never moved its receiver to the exact-frame bucket, and the pairing then
declined because a section entered a face it had left in the polygonal bucket. `wallConicEntersFace`
already takes the meets verdict; this now does too.

**The section is clipped, once, for both sides.** `closedSurfaceUVImprint` keeps a section whole when it
is an island, CLIPS it to the trim when it crosses (`clipSectionToFace`, the same rule
`wallSectionIsland` follows), and drops it when it is clear.

**A component's loops are ordered ends-first.** With the clip in, `brep.Boolean` returned box ∪ sphere
as a VALID solid whose volume was the box exactly — 64.0000 where 67.02 is right. A sphere poking out of
a box through three of its faces leaves an ANNULUS on the sphere: an outer three-arc loop that wraps the
azimuth, and an inner three-arc loop around the box's corner (around the sphere point nearest that
corner, which is inside the box) that does not. `wrappingComponents` takes `loops[0]` as the face's outer
boundary and had been handing it whatever the trace produced first — inner-first here, so the face read
as "the little corner patch minus everything else". `periodTurningFirst` puts the loops that turn a
period ahead of the ones that do not, which is `splitEndsAndHoles`' rule applied where the component is
assembled. The order is otherwise stable, so a band with two wrapping ends and no holes is untouched.

Worth recording separately: the per-face certificate did NOT catch that wrong body. `FaceInteriorPoint`
could not certify a probe on the inverted annulus, and a face whose probe is unavailable is SKIPPED
rather than failed — by design, so the gate never rejects a correct result over a missing probe. The
consequence is that the gate is not a backstop for an emission that inverts a face; only a post-condition
on the emission itself would be. That is a named follow-up, not a change made here.

**Stage 4 standing, all 26 recognizers off: 8 failing tests, down from 22** — the near-pinch family (4),
the coaxial ball-and-rod with a SHOULDER (2 tests, 4 rows), and the torus tangent about every axis.

### The edge sub-range switch moves to geom (2026-09-06)

Restricting a run's curve to its own [t0, t1] was a ten-case switch over curve kinds in the stitch, and
adding the ruled crossing to it tripped the geom-switch ratchet — rightly, because the switch was in the
wrong place to begin with. `geom.SubCurve` now owns it: the stitch asks for the piece and gets back
whatever kind owns it, a circle's sub-range as an `Arc3d`, a ruled crossing's as a shorter ruled arc,
anything else as a `TrimmedCurve3`. The ratchets fall with it — `geomSwitchDebt["kernel/brep"]` 82 → 74,
`kernelNetDeltaPin["type-assertions"]` 754 → 746.

### Contact on a DISC is decided by the disc, not by a one-point polygon (2026-09-06)

A rod stopping PART WAY through a ball's shoulder — its end cap neither inside the ball nor outside it,
the ball's own surface crossing it — left the ball UNCUT. The cap's section with the ball is a circle
strictly inside the cap's disc, and the contact test asked `pointInFace2D`, which reads ONE point per
boundary edge. A face bounded by one closed circle therefore has a "polygon" of a single point, which
contains nothing at all, so the section read as no contact, no imprint was planned, and the sphere passed
through whole.

`sectionMeetsFace` and `sectionInsideFace`'s walk now go through `faceContainsExact`, which meets an arc
boundary by exact ray intervals — the containment test that already existed for exactly this reason. The
ball ∪ rod join comes out at the closed-form volume and area to six figures.

The same one-point-ring shape had already cost `sectionFaceCuts` its crossings on a disc (see the rim
crossing above). `planarRings` is right for an all-straight face and wrong for every other, and the
remaining call sites are worth an audit of their own.

**Stage 4 standing, all 26 recognizers off: 7 failing tests, down from 22.** The near-pinch family (4),
the torus tangent about every axis, and two rows of the shoulder rod — its CUT still keeps the wrong side
of the ball, and its sphere comes back as ONE face with two loops where the surviving surface is two
disconnected caps.

### A disconnected kept region's components are faces (2026-09-06)

After the shoulder rod's contact was seen at all, the ball ∪ rod came out exact but the ball − rod did
not: it kept the wrong material, and its census showed ONE sphere face where the surviving surface is two
disconnected caps — one below the seam circle, one beyond the rod's end.

The emission grouped a trim's loops by (u,v) containment over the WHOLE kept set. Containment is the
right question inside a component and meaningless across components: two loops in different components
need not contain one another at all, and the two caps' rim circles are horizontal lines across the chart,
so neither contained the other and both were filed on one face. That face carried two rings around a
region it did not bound, and the analytic integrator read it as a third of the volume.

`trimByImprint` now walks `keptComponents` first — each component is a face (or a set of patches) —
and `groupLoopFaces` files the holes within it. `keptComponents` was already computed for the charts, so
this asks the same question once instead of two different ones.

The whole coaxial ball-and-rod family passes with all 26 recognizers off, `TestShoulderRodBooleansStayAnalytic`
and the four `TestCurvedBooleanVolumesMatchOCC` rows together.

**Stage 4 standing, all 26 recognizers off: 5 failing tests, down from 22** — the near-pinch family (4)
and the torus tangent about every axis.

### What the torus tangent cut is now failing on (2026-09-06)

The last non-near-pinch row is worth stating precisely, because it is no longer a modelling defect.

A plane tangent to the R=5 r=2 torus's inner equator cuts a figure-eight whose two lobes TOUCH, and the
intersect keeps a band on the torus that pinches once. All six axis placements now return the SAME body —
three faces, two planar lobes and one pinched torus band, through the exact path — so the coin-toss the
corpus was written for is gone. What fails is the MEASUREMENT:

| measure | value | truth |
| --- | ---: | ---: |
| closed form (∫ over the (ρ,z) disc of ρ·(π − 2 arcsin 3/ρ)) | — | 114.886320 |
| OCC | 114.886326 | matches to 8 figures |
| ours, tessellated at DefaultQuality | 112.529001 | 2.05 % low (the gate's budget is 2 %) |
| ours, `AnalyticShellVolume` | 106.438834 | **7.4 % low, and it reports ok** |

The tessellated figure is a chord deficit on a torus and converges. The analytic one does not: it is
wrong by 7.4 % on a face whose boundary touches itself, and its vector-area closure post-condition passed
it. That is a mass-properties defect (M48/C3), not a boolean one, and it is the thing to take up — an
oracle that gates a result must be more exact than the result it gates, and here it is an order of
magnitude worse than the tessellation it replaced.

### What the near-pinch family needs, and why it is not more of this sweep (2026-09-06)

The four remaining near-pinch tests are one capability, not four defects. Two cylinders of NEARLY equal
radius crossing at 90° (|Δr|/R from 3e-3 down to 7e-6) decline at
`geom.IntersectSurfacesAnalytic` — `ruledQuadricConditioning`'s branch-separation gate, `minGap ≥ 0.05 ·
maxGap`, which is doing its job: near the pinch the two roots of the ruling quadratic nearly coincide and
`(−b ± √(b²−4ac))/2a` is solved by cancellation. Measured at R=3, the general path returns the exact
three-face solid down to |Δr| = 0.01 and declines from 0.003; the corpus asks for 2e-5.

So the gate is right and there is nothing below it. The ground rule says a fast path demotes to the
general path when it is ill-conditioned, and for this pair the general path does not exist yet: it is a
near-tangential surface-surface intersector — a marcher, with its own conditioning story and its own
corpus. The hand-written recognizers cover the band with a snap and a per-loop fat-wall trim
(#1781 and #1818), which is why they still hold this family.

That is the next ADR, and widening the conditioning gate to reach it would be exactly the threshold
tuning the ground rules forbid: the gate measures a real loss of precision, not a policy.

**Stage 4 result: 38 → 5 failing tests with all 26 recognizers off.** What is left is two efforts, each
with its own scope: a near-tangential SSI (4 tests) and the analytic mass-properties defect on a
self-touching trim (1 test).

### The torus tangent cut was a modelling defect, and the certificate that finds it (2026-09-06, later)

The entry above called the last non-near-pinch row a measurement defect. It is corrected here, from the
measurement it did not make: the same six bodies TESSELLATE to 275.28 on the recognizer path and 112.53 on
the general path, against a truth of 114.886. The tessellator was reading two different bodies, and one
of them was the torus's complement.

Walked in 3D, the general body's torus face carried one loop through the touch point twice, and walked
its second lobe the wrong way round the tube; the planar lobe sharing that edge was inverted with it.
Every edge was used twice in opposite directions, so `Validate` admitted the shell. The cause was in the
stitch, at a CLOSED loop two faces share while carrying DIFFERENT curve objects for it: the lid takes its
section from the plane cut and the wall its clipped copy from the torus chart, parametrised the other way
round. `stitchUseReversed` read each run's direction against its own parameter, `geom.SubCurve`
re-presented a whole closed spiric run walked backwards as a reversed curve, and the minter's "the stored
closed curve runs forward" then held for one face and not the other. Three rules replace them: a whole
closed run is stored unchanged whatever its kind (`storedWhole`, now first in `SubCurve`); a closed run's
sense is read where it is exact, from the traversal tangent at the loop's one vertex
(`closedRunsOppose`); and a closed edge's stored-curve flip is simply "the representative walked it
backwards", the same rule an open edge already used. The general body now integrates to 114.886320.

**What the integrator did with the inverted body is the finding worth keeping.** `enclosedTerms` signs
every loop from its own boundary integral and normalises a single-cycle band, so a torus face wound
against its normal measured RIGHT on the recognizer path (its two loops each read as a rim) and WRONG on
the general one (its single loop read as a band with twice the travel). The tessellator's band loft
reads no winding at all. So an inverted face shipped as a valid solid, with a correct volume, and only
its mesh — the derived view — showed it. That is the emission post-condition the previous slice left
named: `brep.FaceWindingConsistent` reads every closed loop against its nesting, a chartless band
against its rims' alternation, and a period-turning loop against the face's chart, and the boolean now
adopts nothing that fails it. A recognizer whose body fails DEMOTES to the general pipeline — the thing
it is a shortcut for — rather than past it to the faceted engines; a general-pipeline body that fails
declines with a `boolean.winding-reject` defect.

Measured with the certificate in place, with all recognizers ON:

| recognizer | inverted face | what shipped before |
| --- | --- | --- |
| drill through-hole | the top cap's hole loop | a valid plate, measured right, meshed right |
| two-cap crossing | both caps' holes and the wall's chart | the same |
| ruled crossing cut/intersect | the stub's crossing loop on the rod | the same, with the band loft hiding it |
| torus tangent | one lobe and its torus edge | a valid solid meshing as its complement |

Every one of them shipped valid, integrated right, and meshed as something that does not read the
winding. All of them demote to the general pipeline now, and the corpus holds — after two more things
that demotion exposed.

**A plate with one bore could not take a second.** `planarReceivesConic` refused to move a planar face
carrying a detached curved hole to the exact-frame chart, and the polygonal bucket it stayed in then
declined the second bore's circle entering it. The chart frames a holed face already (the demotion route
does exactly that when an imprint MEETS a hole); the refusal is gone. This also corrects the stage-4
count above: the second bore of `TestSecondBoreRimIsProvenanceNamed` had been served by the
`reconstructedCurvedBoolean` engine, which the recognizers-off measurement never switched off. The honest
measurement is with the recognizers AND the faceted engines off; it is taken at the end of the next slice.

**The mixed stitch's curved rim is named.** The follow-up recorded two entries up is done at the hook:
an unparented CURVED edge group is named by the two faces that border it (`curvedRimLineages`), two rims
of identical parents ranked by the total order on their midpoints, and every planar name the goldens pin
stays as minted — a straight unparented edge is the planar path's split-original fragment and keeps its
convention. A double-bored plate's four rims read `cylinder:f#2/brep:x#0/slab:face#0[/brep:seg#1]`.

**The near-pinch entry above is corrected too.** With the branch-separation margin relaxed for the
measurement, the ruled∩quadric closed form returns the exact section at EVERY radius gap — the two loops
are never closer than 2√(2R·Δr), 0.022 at the corpus's smallest — and the general pipeline builds the
three-face solid down to |Δr| = 3.2e-4 and comes back with three open edges below it. The gate was not
measuring a loss of precision; the roots are exact to 1e-13 there. What fails below is the chart: the
fat wall's seam is placed by the widest gap between the imprint's SAMPLES, which the lens tips' sparse
azimuth cover misses once the corridor between the lenses is narrower than a sample step; the seam then
runs through a lens, and `seamHit` returned ONE crossing per curve, while a seam through a window loop has
two; and a seam ruling had no incidence form at all, so its crossing with a ruled-quadric arc was never
solved and the section-plane route it fell to answers only for planar sections. OCCT's
`IntPatch_ImpImpIntersection` parametrises this section by U1 with no separation margin at all
(`CyCyNoGeometric`); its only guard is the arccos argument's rounding near ±1, the fold this intersector
refuses by base role. Those three are the next slice, and the margin goes with them.

Named and not done here: the tessellator's spiric band loft meshes the corrected lens at 138.92 against
an analytic 111.68 — it lofts a self-touching single loop as a band between two ovals it does not have.

### The near-pinch family through the general pipeline (2026-09-06, later still)

The three defects the previous entry named are gone, and the branch-separation margin with them.

**A seam ruling carries an incidence.** `geom.CurveIncidence` gives a straight curve its two conditions —
the distances to two perpendicular planes through it — so the chart's artificial seam goes to the same
incidence solver the frame and the imprint use for one another (`curvePairMeets`). `seamHit`, which took
the section-plane candidates and stopped at the first inside both spans, is deleted: a window loop the
seam enters and leaves reports both crossings, and a ruled∩quadric arc, which has no section plane, is
met through its own incidence. One point is decided once: a curve carrying two conditions has both
vanish at a crossing, the walk brackets it once per condition, and `meetKnown` keeps the first — two
vertices a rounding apart on one boundary read as an open edge, which the cone's clipped-rim ellipse
showed at once.

**The seam is placed exactly.** `curved_seam_place.go` reads every imprint curve's azimuth EXTENT from its
turning points — the sampled azimuth's extrema, each refined to rounding by `geom.ExtremumOnBracket` —
and puts the seam in the middle of the widest stretch free of every extent, every turning point, every
curve end and every frame coordinate. A curve that winds the azimuth leaves nothing free, and the seam
then goes as far from every turning point and curve end as it can, where its crossing is transversal and
bracketed. The sampled rule had found the corridor between two near-pinching lenses only while it was
wider than a sample step; the exact one finds it at every gap the corpus asks for. The sphere's and the
torus's seams take the same rule, the torus's in both coordinates.

**The tube seam is solved, not folded.** A torus chart's second seam crossed the imprint only where the
sampling happened to straddle it. It is now a curve on the host (`tubeSeamCurve`), its crossings are
solved with the azimuth seam's, snapped onto the seam in v as the azimuth seam's are in u, and the
parameter rectangle's four sides are split at every incidence solved on them. Two exact-boundary cases
surfaced on the way and are fixed where they belong: a run that ends EXACTLY on the seam is written on
its own side (`splitPeriodicSeam` re-bases the end to the branch continuous with the start, rather than
leaving a segment that jumps the period), and a root that falls exactly on a sampling station counts as a
root (`curveRootsOnOther`; a strict sign change saw none). An injected incidence now also wins over the
sampling station it coincides with (`preferInjected`): the seam placed at π/2 on a rim sampled at quarter
turns merged into the station's value and lost its mark.

**The gate reads the minimum, not a margin.** `ruledQuadricConditioning` refines every bracketed local
minimum of the branch gap and requires the least of them to exceed the stitch resolution — the
certificate that the two section branches are two curves the stitch can tell apart. The twentieth-of-
the-largest-gap margin is deleted (`kernelNetDeltaPin` tolerance-constants 232 → 231). Measured with
the recognizers demoting:

| corpus | before | now |
| --- | --- | --- |
| `TestNearPinchRecoveredBandWatertight` (12 rows, R=3 and R=30, Δr/R down to 7e-6) | recognizer | general pipeline, exact, watertight |
| `TestNearPinchCutJoinWatertight` cut (8 rows) | recognizer | general pipeline, exact, watertight |
| `TestNearPinchCutJoinWatertight` join (8 rows) | recognizer | general pipeline, valid solid, MESH open |
| `TestBooleanIntersectNearPinchContinuity` | recognizer, snapped | general pipeline |

The join rows fail on the tessellator, not the boolean: the fat wall keeps its two rims and the two
lens holes, and `tessellate` records `wall-wrap-unmeshed` for a full-wrap wall carrying three holes and
meshes only part of it; the rod stubs' band loft discretises the shared rim on its own stations, so
the caps' rims do not weld to it. Both are named tessellation follow-ups with the spiric band loft
above, and the three of them are one shape: a tessellator that reads a face by recognising its edge
pattern instead of by its chart.

Recorded and not done: `TestBooleanIntersectNearPinchContinuity` pins the SNAP the retired recognizer
performed — four faces for a radius gap under the stitch resolution — which the general pipeline does
not perform and the ground rules forbid (a nudge to make the operation succeed); that row's premise is
the recognizer's and moves with its deletion in stage 7.

### What the demotion exposed, and the hole feature joins the boolean (2026-09-06, evening)

With the recognizers demoting, three corpus families changed hands, and each showed a defect that
had been sitting behind the recognizer.

**The tessellator read two faces by their own rule.** The spiric band loft meshed the band between two
spiric ovals "the long way round the tube" whatever the face was — right for a cut through the hole,
wrong for the lens an intersect keeps — and the corrected torus tangent body measured 138.9 against an
analytic 111.7 for it. It now asks the face's chart (ADR-0063) which side of the ovals the trim is on
(`spiricBandSpan`). The two-rim holed band bridged its rims at the widest gap between the lens holes'
SAMPLES, anchored on existing rim vertices: two lenses that nearly pinch leave a corridor narrower than
a rim step, so a straight slit between the nearest vertices crossed a lens however it was placed, the
mesher declined, and the wall fell to a flat patch with a `wall-wrap-unmeshed` defect and a hundred open
mesh edges. The seam now runs at the chart's seam — placed by the boolean in that corridor, exactly — as
a POLYLINE that keeps to the seam azimuth wherever a lens could be (`bentSeamOnSurface`); its interior
points are its own, so both copies still weld. Every near-pinch join row meshes watertight.

**A frame crossing on the face's own seam ruling is not a vertex.** `splitAtFrameCrossings` cut a
re-emitted rim wherever a frame edge crossed it, on the grounds that the crossing is a vertex on the
neighbour. Where the frame edge is the face's own seam ruling — the edge a primitive's wall carries
twice — the neighbour across it is this face, the ruling dissolves inside the kept band, and nothing
keeps the point; an extruded circle's bore came back as two half-circles on the tool's seam azimuth
while the cap across the rim held one circle, and the stitch split the cap's circle to match. A seam
ruling's crossings are skipped (`frameEdgeIsSeam`), and `TestCircularCutKeepsCircleEdge` holds through
the general pipeline.

**Parallel cylinders are provably apart.** `geom.SurfacesApart` proved only COAXIAL cylinders apart, so
a bore inside a disc's rim "overlapped an uncovered wall" and the mixed pipeline declined every second
bore of a patterned disc to the reconstruction engine. The least separation of two parallel cylinders
is closed-form — side by side, or nested — and the predicate reads it (`parallelCylindersApart`).

**The hole feature cuts with the boolean.** `HoleFeature.cutCylinder` called `brep.CutCylindricalHole`
and `CutBlindCylindricalHole` directly: a drill recognizer invoked from the model layer with no boolean
around it, whose bodies bypassed every post-condition the boolean applies — the top cap's hole loop
came out wound against the cap and only the winding certificate ever saw it — and whose wall was minted
`brep:drillwall#0` whatever the feature, so two holes in one part were two faces with one key and a
pick of either was ambiguous. A pattern replaying the hole's recorded tool then cut a target the
certificate had never certified and fell to the faceted engines. The hole now cuts with its analytic
cylinder tool, named for the feature instance (`brep.SolidCylinderNamed`), through
`ops.BooleanWithDiagnostics` — the same operation an extruded circle cuts with — and the recorded
replay tool and the cut are the same solid. `TestTwoHolesAreTwoNamespaces` pins the keys. A from-to
bore, which the exact blind drill declined and the faceted prism served, is an ordinary cylinder to
the boolean, and its corpus row now asks for πr²·h rather than the 32-gon's area.

Two builders stop being called from the model layer by this: `CutCylindricalHole` and
`CutBlindCylindricalHole`, which stage 7 deletes with the recognizers that wrap them. The counterbore and
countersink builders are still called directly and are the same shape of defect; they follow.

Two gaps the hole feature's move surfaced, named and not done here. **A bore tangent to a face** — a
Ø2 bore a radius in from a block's side, the corner bores of the sketch-placement corpus row — is a line
contact between the tool's wall and a target face; no path claims it, the CSG engine tears it, and the
feature then shipped that torn body as healthy: **the feature engine does not `Validate` a result it
gets without an error** (`classify`), which is the post-condition every public operation owes and the
model layer does not yet ask for. The inscribed 32-gon prism the feature used to cut with was never
tangent, which is why the row passed; it now drills Ø1 bores clear of everything and asks for πr²·h.

### The honest count, and what it is made of (2026-09-06, night)

The stage-4 count was taken with the recognizers off and the faceted engines ON, so a configuration the
general pipeline declined and the reconstruction engine rebuilt from faceted provenance counted as a
pass. Taken honestly — recognizers off AND `reconstructedCurvedBoolean`, the CSG engine and the
mesh-arrangement rescue all declining — the kernel suite fails 20 tests. Eight of them assert that a
fallback FIRES (`TestBooleanMeshArrangementFallbackRescue`, `TestBooleanRecordsCSGFallbackDiagnostic`,
`TestABooleanWithNoExactCurvedPathDeclinesByName`, the reconstruction and volume-reject tests) and
convert with stage 7, as the Consequences section already says. The rest are the general pipeline's
remaining scope, each now measured to its cause:

| configuration | rows | what the general pipeline says |
| --- | --- | --- |
| two-cap crossing: a steep rod exits both caps of a cylinder | 5 | the mixed pipeline declines (target 3 faces, tool 3) |
| coaxial cylinders, overlapping or abutting, joined; a sphere joined to a rod on its axis | 3 | declines — the degenerate-overlap family, coincident surfaces |
| the near-pinch continuity row's SNAP band | 1 | builds three exact faces; the row asks for the retired recognizer's four |
| the drilled plate at 10 µm and 100 µm | 1 | drops a cap (see below) |
| the corner junction's tessellation and membership audit | 2 | a regression of the night's own tessellation commit, found by bisecting it and fixed |

**The drilled plate at a tenth of a millimetre is the resolution floor.** `minModelSize` floors every
`Resolution` at one database centimetre, so a sub-centimetre model is measured with a centimetre's
tolerances: a 10 µm plate's bottom cap, a tenth of the plate's thickness above the bore's rim, is
"coplanar" with that rim at the floored sew gap, `sectionOnWallEdge` files the cap's section as a
boundary contact, and the cap is dropped. Lowering the floor to a degeneracy floor (ten nanometres) mends
the plate at 100 µm and fails 26 kernel tests at unit scale: the fixtures the corpus is built from are
sub-centimetre rods and balls, and thirty call sites build a resolution from a radius, a height, or two
origins rather than from the operands' extent, all of them leaning on the floor. That is ADR-0042's
remaining debt — a resolution derives from the model's extent, once — and it is its own slice; the floor
stays until it lands, and the row is served by the mesh-arrangement rescue, reported.

**The corner junction cracked along the tessellator's bridge.** The chart-seam bridge of the two-rim
holed band ignored the NOTCH in a notched rim — the chart's seam is clear of the imprint, not of the
face's own boundary — and anchored in it. A notched rim keeps the sampled placement, which was written
around the notch; the chart seam serves an intact rim, which is the near-pinch case it was made for.

**Every hole type cuts with one revolved tool.** Counterbore, countersink and the drilled point each
called a bespoke drill builder; each is now a meridian — stepped, chamfered, coned — revolved about the
bore axis into one analytic solid (`revolvedTool`), named for the feature instance, and taken out by the
general boolean in one operation. `CutCounterboreHole`, `CutCountersinkHole` and `CutBlindConicalHole`
have no caller in the model layer left.

**A feature that yields an invalid solid should be sick, and is not yet.** `classify` marks a feature
healthy whenever its Recompute returns no error, whatever body it returned; a torn CSG body a tangent
bore produced shipped as a healthy hole. Sickening a feature whose solid fails `Validate` was written
and measured: eleven model tests fail, because their fixtures — the stand-in boxes the engine, suppression,
reorder and pattern tests are built on — are bodies that call themselves solid and are not. The
post-condition is right and the fixtures are the debt; it is withdrawn here and named for the slice that
rebuilds those fixtures on real solids.

The planar stitch's weld grid also derives from the operands now (`stitchInputBox`), as the curved
stitch's did; it was an absolute grid a millimetre-scale plate could not survive.

### The degenerate overlap, and an empty answer that was a proof (2026-09-07)

Two of the four configurations the honest count named are done, and both were the same mistake in
different clothes: the pipeline read an ABSENCE as an inability.

**A tool that never touches the wall.** A steep rod enters a cylinder through one cap and leaves
through the other, staying inside the wall the whole way. `overlapsUncarriedWall` asked the wall pair
for a crossing and read "no curves" as "not decided", so the whole two-cap family declined —
five corpus rows, one of them the OCC-certified `TestTwoCapCrossingCutMomentsMatchOCC`.
`wallWallImprint` already says which it means: ok=false is undecided, ok=true with no curves is the
decided answer, and it has two roots — the two infinite surfaces are known not to cross, or every
crossing they have lies clear of one of the two bands. Either way the walls do not touch. The guard
the old reading protected, a grazing partial-rim cut, declines at the intersector's conditioning gate,
which is ok=false, and its certifications hold on both paths.

**Two walls on ONE surface.** Two coaxial cylinders of equal radius overlap in a REGION, not along a
curve — the degenerate-overlap class ADR-0045 names, which the planar boolean has always had and the
curved side never did. Asked for their crossing, the intersector correctly answers that it cannot: the
ruling quadratic's leading coefficient vanishes identically along a cylinder's own axis. The pipeline
read that as an unsupported pair.

It is three rules, and none of them is about cylinders:

- **Identity.** `geom.SurfacesCoincide` is the sibling of `geom.SurfacesApart`, and answers with the
  same discipline — exact and one-sided, true only where a finite set of parameters proves it. Planes,
  cylinders, cones, spheres and tori; anything else is false. The type switch lives in `kernel/geom`,
  where the ground rules put it.
- **The imprint.** Two coincident walls' contact is bounded by where one band ENDS inside the other, so
  the imprint is each band's rim circles taken in the other's frame (`coincidentWallImprint`). Each
  side's own rim lands on its own frame and `admits` drops it there, so the pair is still solved once
  and written to both.
- **The classification.** A point covered by a face of the other operand ON THE SAME SURFACE follows the
  ON/ON table, never the membership oracle — which is meaningless there, the point being on the
  boundary the two solids share. That rule existed for planes as `uvKeepAt`/`coplanarCoverExact`; both
  are deleted, and `coincidentKeepAt` serves the exact-frame chart, the ruled wall, and the demoted
  planar face alike. The wall path had no such test at all, so it now takes the other operand's faces.

One more thing had to give way. The section a coaxial cylinder's wall cuts from the plane of the cap
closing the other IS that cap's own rim, and no face is split by its own boundary. That is the
receiving face's half of the rule `sectionOnWallEdge` already stated for the wall, and it is stated by
walking the section: every sample must lie on ONE edge's own span, so a section that merely touches the
boundary or runs past its end stays a genuine imprint.

**Honest count: 20 → 12.** Of the twelve, nine assert that a fallback fires or exercise the
reconstruction engine and convert at stage 7; one is the near-pinch continuity row's SNAP premise,
which belongs to the recognizer being deleted; one is the drilled plate below a millimetre, which is
the resolution floor named above. The general pipeline's remaining scope is the tangent contact — an
off-axis rod on a ball (a quartic seam), and a bore tangent to the face it cuts.

### Two spheres, and the certificate's own false positive (2026-09-07, later)

**The simplest curved crossing there is was not in the intersector.** Two spheres meet in a circle —
subtracting their implicit forms cancels the quadratic terms and leaves the radical plane — and neither
bucket of `IntersectSurfacesAnalytic` reached it: a sphere has no straight ruling to substitute, so the
parametric×implicit form declined both role assignments and the pair went to the marcher. The closed
form is `sphereSphereSection`, gated like the others on conditioning: apart or nested is DECIDED with no
curves, a tangent touch and a concentric pair decline, because a point and a coincident surface are not
crossings a section curve can carry.

The mixed pipeline had no pairing for two closed surfaces either, so a ball meeting a ball declined on
box overlap alone. `pairClosedSurfaceImprints` takes the scope its sibling already takes — both faces
boundary-less, every crossing closed, written to both sides — and `closedSurfaceUncovered` now reads a
decided crossing as covered, which is the same correction the wall pairing needed. All three sphere-pair
booleans come out as two spherical caps with exact volumes: the lens (π/12d)(2r−d)²(d²+4dr), and the
union and difference that follow from it.

**The winding certificate was condemning a correct body.** The band a rod keeps where it crosses a
fatter cylinder is bounded by two loops that each wrap the rod's azimuth, and the chart carrying it is
that band cut open at a seam. The certificate read a rim's direction against the nearest CONTOUR
SEGMENT, the rim's middle sample sits exactly on the seam, and the seam runs across the rim — so the
crossing-cylinder INTERSECT, whose volume matches OCC to six figures, was rejected and demoted to the
faceted engines. A chart contour carries the artificial boundary along with the real one; a point list
cannot tell them apart.

The face's own trim can. One rule replaces three: at each station of a boundary ring, step a short way
to the side the winding claims the material is on, and to the opposite side, and ask the trim about
both. A station where they land on different sides has resolved the boundary and votes; one where they
agree measured nothing and is passed over. That is the contract stated directly — no shoelace, no
nesting among the other loops, no special case for a loop that wraps a period — and the drill and
two-cap recognizers still ship the inverted faces the certificate was built to catch.

**A converted test, by its own rule.** `TestBooleanRecordsCSGFallbackDiagnostic` declined on the sphere
pair, and said in its own comment that it would convert when that configuration landed. It now asserts
the wiring on a ball joined to a TORUS, whose crossing is a genuine quartic, and
`TestSpherePairVolumesAreExact` is the sphere pair's positive form.

### The resolution floor, measured (2026-09-07, later)

The drilled plate below a millimetre is the last corpus row the general pipeline does not build, and it
is not a boolean defect. `geom.minModelSize` floors every `Resolution` at one database centimetre, so a
part smaller than that is measured with a centimetre's tolerances. One circular derivation fell out of
looking at it and is fixed here — `SectionCrossingCandidates` judged "are these two section planes one
plane" at a tolerance derived from the gap between those very planes, which the floor then turned into
an absolute 1e-4 — but the floor itself is what stops the plate.

Lowering it to a genuine degeneracy floor (ten nanometres) mends the plate at a millimetre and fails 22
kernel tests, and they do not share one cause. Measured, they are:

| shape | example |
| --- | --- |
| a test that PINS the floor's value for a degenerate operand | `ResolutionForBody(nil).Size()` |
| an angular classification reading a floored length tolerance | the fillet's cone∧plane arm calls a perpendicular plane "oblique" |
| the boolean's own gates on fixtures a few units across | the general crossing, the half-space cuts, the Steinmetz |

Each is a site deriving its resolution from a LOCAL quantity — a radius, a height, two origins — rather
than from the model's extent, which is ADR-0042's rule and its remaining debt. The floor was masking
all of them at once. Fixing them is its own slice with its own corpus (a part swept across six decades
of scale), and it is not ADR-0061's: the retirement needs it only for this one row, which the
mesh-arrangement rescue serves, reported.

### Stage 4's gate is met: the model corpus needs no recognizer and no engine (2026-09-07, night)

Measured with the 26 `curvedExactPaths` recognizers emptied AND the three faceted engines refusing —
`reconstructedCurvedBoolean`, the triangle CSG, and the mesh-arrangement rescue:

| corpus | failing tests |
| --- | ---: |
| `./model/...` — every feature, and the OCC parity suite | **0** |
| `./kernel/...` | 9 |

Nothing in the model layer needs a recognizer or a fallback any more. Every hole, boss, chamfer,
fillet, emboss, pattern, revolve, loft and sheet-metal case, and every OCC-certified volume, is built
by the general per-face pipeline alone.

The nine kernel rows are not general-pipeline scope, and each is named:

- **seven test the fallbacks themselves** — the mesh-arrangement rescue, the CSG-fallback diagnostic,
  the named decline, the observable chained decline, the two reconstruction rebuilds, and the off-axis
  rod whose valid faceted solid is the assertion. They convert or go with the engines at stage 7,
  exactly as the Consequences section said they would.
- **one is the near-pinch continuity row's SNAP**: four faces where the general pipeline builds three,
  because the retired recognizer snapped two near-equal radii together. That is the recognizer's own
  convention and it moves with it.
- **one is the drilled plate below a millimetre**, which is the resolution floor — ADR-0042's debt,
  measured and named above.

The last three defects that stood between the pipeline and this measurement were all one mistake in
different clothes, and all of them shipped a WRONG BODY rather than a decline wherever the rescue was
not there to catch it:

- an imprint that runs along a frame edge is a CONTACT, whatever kind of curve carries it. The guard
  said "two sections in one plane are one conic on the surface", which cannot see a crossing delivered
  as a ruled arc — and a chamfer wedge's cone crosses the shaft's wall at exactly the wedge's own rim.
  The arrangement carried that rim twice, a rounding apart, and zig-zagged between the copies: ninety
  alternating fragments where one circle belonged.
- the same rule from the receiving face's side, for a coplanar pair: two coaxial cylinders abutting cap
  to cap hand each disc a copy of its own boundary.
- and the merge that follows from both — two faces on one surface whose common boundary dissolves are
  one face.

### The resolution floor is lowered: the whole kernel builds at a nanometre (2026-09-08)

The measurement above said the floor's 22 failures "do not share one cause". That reading was wrong,
and the correction is the point of this section. Lowering `minModelSize` further, from a centimetre to
a nanometre, raised the count to 55 — and 53 of those had ONE cause in two places, plus one more in a
third. Each is a quantity read from too small a sample, which the centimetre floor had been silently
replacing with a centimetre.

**A face's box must bound its edges, not its vertices.** `faceLoopBox` and `curvedFaceBox` both took a
face's extent from the points where its loop edges meet. A face bounded by ONE closed circle — every
cylinder cap, every sphere lid, every disc — has both ends of that circle at the same seam point, so
its box was a POINT and the resolution derived from it was the floor. Nineteen call sites read
`faceLoopBox`; the stitch's weld grid reads `curvedFaceBox`. At a centimetre the point-box handed every
such face a centimetre's tolerances and the answers happened to be right. At a nanometre the cap's own
rim stopped crossing its own probe line, and the hemisphere's two seam vertices (1.2e-15 apart) stopped
welding: `V−E+2F−L = 3`, an inadmissible Euler characteristic on a two-face solid.

The fix is `geom.CurveBox(c, t0, t1)`, the exact box a curve reaches over its span: `AxialExtent` along
each world axis, so a conic's box carries its interior stationary points and not merely its two ends.
Both face boxes are built from it, and `curvedFaceBox` is now `faceLoopBox` over the set rather than a
second copy of the same walk.

**A window's pad must be relative when the roots inside it are.** `lineWindowOf` padded the face's
extent along a probe line by an absolute `facePairCullPad` (1e-5), while the conic solver that consumes
the window rejects a root within `tjTol` (1e-7) of either end measured in the segment's NORMALISED
parameter. The two do not compose: at a 3 m cap the pad is 1.7e-9 of the window, so the cap's own rim
read as outside its own window and the face reported no interior point. The pad is now a fraction of
the window's own span (`lineWindowSlack`), which is scale-invariant by construction. This is what
`TestCornerJunctionScaleInvariantAngle` was already asserting one level up.

**The empty box was not union's identity.** `math.EmptyBox` documents itself as "the identity box for
union", and `Box.Union` extended by the other box's corners — taking the ±Inf sentinels literally and
returning an INFINITE box. A seamless sphere face has no loop edges, so its face box is empty, so a
stitch set containing one had an infinite box and an infinite model size. That is the same defect as
the point-box with the sign reversed: a disjoint union of a block and a ball came back as seven shells
with twelve unpaired edges, because the weld was as coarse as the point-box's was fine.

With those three fixed, `./kernel/...`, `./model/...` and `./math/...` are green at
`minModelSize = 1e-9`. Two premise tests moved, and both were pinning the floor rather than testing
behaviour: `kernel/ops`'s constructors now read the floor from `geom.ResolutionForSize(0)` instead of
repeating the literal, and the fillet's cone-arm tests now hand `coneArmEdge` the BODY their fixture
builds instead of `nil` — an angular band read off a nil body is a band read off the degeneracy floor,
which is what made a perpendicular cap plane read as "oblique".

The floor is now what its name says: a guard against a degenerate operand, not a smallest part. The
drilled plate below a millimetre builds through the general pipeline, and the last of stage 4's nine
kernel rows that was not a fallback's own test is closed. `geom/resolution.go` owes the tolerance
ratchet one literal instead of two — `epsRel` and `volCoef` are relative fractions and now say so,
leaving `minModelSize` as the single absolute anchor the relative system stands on.

### The feature layer's face-count gate becomes a classification (2026-09-08)

The Decision section above recorded a widening that did NOT land: `curvedBooleanWorthTrying` admitted a
tool with exactly ONE curved face and refused one with two, where the rule asks for a classification —
the planar path cannot consume a curved face at all, so "either operand carries one" is the whole of
it. It was held back because the widening drove a fine-pitch coil join into the mesh reconstruction,
which did not terminate on that body.

**That blocker is gone.** With stage 4's charts in, `TestCoilJoinFinePitchWatertight` passes through
the widened gate at all four pitches, and the whole model suite is green and no slower for it:
`model/feature` 911 s → 892 s, the OCC parity suite 509 s → 456 s. The gate is now the classification,
and `curvedFaceCount` is deleted in favour of the `hasCurvedFace` predicate that was already there.

**A correction to the record.** The stage-4 gate section above reports `./model/...` failing 0 tests
with the recognizers and engines off. That was wrong when it was written: the wrapped emboss failed
then too, and the run that produced the 0 did not include it. The honest model count at that moment was
2, both rows the wrapped emboss, and this section is what takes it to 0.

The count was the whole of the model corpus's honest failing count, and the row it cost is the wrapped
emboss. Its pad is a watertight cage of TWO cylindrical caps and a ring of planar side walls, so
`oc == 2` and the count refused it; `planarized` then turned the shaft's analytic cylinder into a
24-gon prism and handed a 26-face polyhedron and a 36-face cage to the planar boolean, which declined,
and the triangle-soup CSG built the body. No recognizer was ever involved — the row was the FEATURE
layer's, and it was the one place a shipping feature still needed an engine stage 7 deletes.

**The model corpus now needs no recognizer and no faceted engine, measured, with the resolution floor
at a nanometre:**

| corpus | failing tests, recognizers AND engines off |
| --- | ---: |
| `./model/...` — every feature, and the OCC parity suite | **0** |
| `./kernel/...` | 9 |

Of the nine kernel rows, eight are the fallbacks' own tests (seven) plus the near-pinch continuity
SNAP premise, all of which convert at stage 7. The ninth is
`TestTessellationWatertightAcrossScales` at the 10 µm and 100 µm plates: the boolean builds them, and
the MESH is not watertight there. That is the tessellator's own scale debt — `mesh.Quality` carries an
absolute `ChordTolerance` — and it is downstream of this ADR.

### A wall band's cull margin was an absolute length (2026-09-08)

The ninth kernel row above — `TestTessellationWatertightAcrossScales` at the 100 µm plate — was not
the tessellator after all. The BOOLEAN declined it, and the reason is one more absolute length used as
a model-relative margin.

`bandPlacement` asks whether a section curve sits strictly inside a wall's axial band, clear of both
rims. Its margin was `facePairCullPad`, which is ten planar stitch grids: right for a part about one
database unit across, and nothing at any other scale. The plate is 6e-5 tall and its bore runs from
−1e-5 to 7e-5, so the top cap's section sits EXACTLY one pad below the bore's rim. It read as a rim
contact rather than an interior island, `wallSectionIsland` fell through to `clipSectionToWall`, and
the simplest drill there is — a block cut by a cylinder, six faces against three — declined.

The margin is now the band's own: ten stitch welds of a `Resolution` built from the band's extent
(`bandCullPad`), which reproduces the absolute constant exactly at the historical ~1-unit part and
scales with the part everywhere else. `spansOverlap` becomes `spanMeetsBand` — all four callers were
comparing a span against one band, so the band is the argument and the margin is read off it — and
`ruledSide.size` is now the same `bandSize` the margin uses.

**What is left below the corpus, named.** At 10 µm — a decade below the sweep's smallest case — the
same drill returns an EMPTY body with no error. Every face classifies as removed, so nothing reaches
the stitch. That is a wrong body rather than a decline, which this ADR's rules forbid, and it is not
the band margin (scaling `facePairCullPad` down by four decades does not move it). It is named here
rather than fixed: `facePairCullPad` is still an absolute length at nine other sites — the AABB cull,
`geom.SurfacesApart`, the interval inflation, the band-window test — and relativising them is one
slice with a scale sweep as its corpus, not a change to make one row at a time.

### Stages 6 and 7: the doors close and the room is deleted (2026-09-08)

**The doors.** `booleanGeneralExact` no longer reaches a faceted engine at any exit. A pair no exact
path models is refused by name — `ErrUnmodelledBoolean`, carrying the operation, both operands' face
counts and the underlying cause — and a result that fails its own acceptance gate is refused the same
way rather than replaced by a triangle-soup stand-in. `meshArrangementFallback` is gone from
`booleanGeneral`, `reconstructedCurvedBoolean` from `curvedExactBoolean`, and `booleanCSG` with them.
`faceted-entry-sites` reaches 0.

**The room.** Deleted: `kernel/meshbool` entire, and `csg*`, `meshbool_*` and `mesh_brep.go` from
`kernel/ops/boolean` — 4 429 + 1 173 lines of engine, plus their tests.
`faceted-engine-files` reaches 0.

`MeshToBRep` came out of that deletion alive, in a new package. It is not an engine and nothing falls
back to it: it converts a welded triangle mesh into a faceted B-rep, which is the mesh-solid IMPORT
path a shipping feature uses. `kernel/ops/meshbrep` carries it and the welder it needs; the CSG's own
triangle production (`bodyTriangles`, `booleanInputQuality`) went with the engine.

**The tests converted, none deleted to move a number.** Every fallback-asserting test now asserts the
REFUSAL — the error, the diagnostic, and that no body comes back — on the same fixture:
`TestBooleanRefusesAnUnmodelledConfigurationByName` (a ball joined to a torus),
`TestPartialRimChainedCutRefusesByName`, `TestABooleanWithNoExactCurvedPathRefusesByName`, and the
off-axis rod on a ball. The five reconstruction fixtures became positive corpus rows driven through
the PUBLIC boolean in `boolean_reconstruction_corpus_test.go`: the stepped shaft keeps both analytic
walls, two boxes union exactly, the oblique bore keeps its elliptical rims, the oblique stub keeps its
elliptical seam, and the cocylindrical cap stays analytic. The welder's own tests moved with it.

**Three defects the engines had been masking, found by closing the doors.**

- **A ruling imprint was clipped to the tool's trim but not to the wall's own band.** A ruling is an
  infinite line; the tool face can cover a stretch of it that lies entirely outside the wall being
  imprinted. A D-profile prism seated on a cylinder meets that cylinder along two rulings through its
  chord's corners, and the tool's chord face covers them four units above the wall receiving them. The
  wall split on a segment that never touched it and closed the fragment with a spurious full-turn arc
  at each corner. `rulingSegments` intersects both trims.
- **`pointOnFaceBoundary` chorded the ring and read a database centimetre.** It walked the loop
  vertices, which is exact only while every edge is straight, at an absolute 1e-7. It now walks the
  EDGES at the face's own on-plane tolerance. Judging the distance on the on-plane class rather than
  the sew gap is what the sliver intersect needed: a sew gap is a tenth of a millimetre on a centimetre
  part, and it read a 1e-4-thin slab's own interior imprint as lying on the boundary, dropping the four
  side faces of the slab and leaving two faces and eight open edges.
- **A hole ring that returns to a vertex is TWO loops.** Two glyphs of an embossed word whose outlines
  meet trace as one walk through the shared contact; welded, that walk pinches on one vertex and the
  body comes back closed and edge-manifold with an ODD Euler characteristic. The CSG used to split such
  vertices apart. `splitPinchedRing` splits the walk into the loops it is and drops the slit's remnant.

**What the general pipeline still refuses, named.** `mixed-decline-returns` is 3 and does not reach
zero here: it counted configurations that routed to a faceted engine, and now counts the ones the
pipeline refuses by name — which is what the ground rules ask for. Stage 5, a chart for freeform faces
that ends the pass bucket, is what takes it to zero, and it blocks no corpus case today.

**One quality gap, named rather than approximated.** A D-profile prism seated on a cylinder of the same
radius builds a valid solid of the right volume with both walls analytic and on ONE surface — which is
what #2167 was about, the faceted seam — but as TWO faces where a correct B-rep has one. Their common
boundary is part of the cylinder's rim, not a whole edge of it, and `mergeCoincidentFaces` merges only
a whole shared boundary. Splicing a partial one needs more than cutting the edge at the run's ends: the
two faces' chart SEAMS meet inside the run being dissolved, so the merged loop has to fuse those too.
Both tests pin the count at 2 rather than relaxing it, so landing that merge trips them and converts
them.

### What closing the doors surfaced in the host, and its fixes (2026-09-08)

Closing the doors turned two host configurations from silently faceted into refused. Both were
configurations that ONLY ever worked through a faceted rescue — the trade this ADR's Consequences
section named before stage 6 was written — and neither was a regression in anything this stage changed:
reverting each of the stage's four geometry fixes in turn left both failing, and the same operands
produced the same invalid body on the parent commit, where the rescue replaced it. Both are fixed here,
and `./kernel/...`, `./model/...`, `./app/...`, `./math/...`, `./addin/...` and archguard are green with
the engines deleted.

- **A drilled hole whose depth exactly equals the plate's thickness — FIXED.** The app's default hole
  on a 4 × 4 × 2 block is Ø1 × 2 from the top face, so the tool's cylinder ends at z = 2 flush with the
  top AND its drill point's shoulder sits exactly at z = 0, the plate's bottom plane. The section there
  is the cylinder wall's rim AND the cone's base, so the rule "a section on the wall's own edge is a
  contact" skipped it from both faces. The underside then took the whole-face classification, its
  interior point — the bore's centre — read as removed, and the face was DROPPED: six faces where seven
  belong, with the four bottom edges and the bore's rim unpaired.

  What tells a contact from a crossing when the structural rule cannot is MEMBERSHIP, and asking it is
  the fix. `sectionEnclosesOtherMaterial` steps to each side of the receiving plane at the section's
  centre and asks the other solid's own oracle: material on BOTH sides is a crossing, so the face is
  split; material on one is a contact, so nothing is. Never the plane itself — a through hole whose
  tool ends exactly at the face has its centre ON the tool's boundary there, and that must read as a
  contact. Two more rules fell out of the same fixture: a wall is never split by its OWN rim even when
  the face it imprints is (each side takes only what is an imprint for it), and two walls of one solid
  that meet AT the receiving plane section it in the SAME curve, which the arrangement cannot split a
  face by twice (`appendDistinctSection`).

- **An assembly revolve machining a participant — FIXED.** A rectangle revolved a full turn is an
  annular ring — two coaxial cylinders and two annular caps — and cutting it from a box left eight open
  edges, because the ring's INNER wall contributed no fragment at all.

  That wall is REVERSED, so it frames itself from its far rim and its band reports [−1, 0] where the
  outer wall's reports [0, 1]. `bandBase` — the point a band's axial coordinate is measured FROM — read
  `−bandV(bottom) + vMin`, which cancels to zero and returns the bottom rim itself; but bandV AT the
  bottom rim is vMin, not zero. So the [vMin, vMax] window was offset by a whole band height and the
  inner wall's imprints came back clipped over the band BELOW it: measured, its two rulings spanned
  y ∈ [0, 0.5] where the wall lives at y ∈ [0.5, 1.5]. Nothing crossed the wall, the arrangement made
  two cells instead of four, and neither was inside the box. The outer wall, whose band begins at zero,
  was unaffected — which is why exactly half the tool went missing.

  The error was invisible for as long as every band began at vMin = 0 and a faceted engine stood behind
  the result. `bandBase` is now the bottom rim offset BACK by vMin, so `origin.VectorTo(p)·axis` IS
  bandV(p) and a window means what it says. `TestAnnularRingCutClosesTheBox` pins the topology and
  `TestAnnularRingCutRemovesExactlyAQuarter` the exact volume: the ring lies wholly within the box's y
  span and reaches ±1.5 inside a 2-unit box, so the cut removes exactly a quarter of the annulus.

## Stage 4's deletion — the 26 recognizers are gone (2026-09-07)

The stage-4 GATE was met on 2026-09-07: with `curvedExactPaths` emptied and the three faceted engines
refusing, `./model/...` failed nothing. Stages 6 and 7 then landed, deleting the engines. What remained
was the deletion the stage exists for, and this is it.

**What went.** `curvedExactPaths` and its 26 entries; the four `kernel/ops/boolean` files that declared
them (`boolean_crossing_cylinder.go`, `boolean_curved_convex.go`, `boolean_curved_subtract.go`,
`boolean_curved_flat_subtract.go`) with the `gatedCurved` / `withoutRecorder` adapters; and the 25 brep
files behind them — the ruled-crossing and partial-penetration drivers, the equal-radius Steinmetz family
and its snap ceiling, the four cap-crossing slices and their rim-corner solver, the partial-rim cut and
its corner-junction builder, the drill through-hole recipe, the cylinder boss and the straddling boss,
the edge scallop, the coaxial cylinder union, and the coaxial sphere-and-rod builders with their span,
winding and membership machinery. **45 files, 5 061 production lines and 2 069 test lines net.**

`curvedExactBoolean` is now four guards around one call to `brep.BooleanDiag`. The near-pinch decline
went with them (`CodeImprintNearPinchDeclined` and the gate that recorded it): it existed to hand a
narrow-neck crossing to the Steinmetz constructor below the snap ceiling and to the faceted route above
it, and neither destination exists. So did `curvedSolidMembership` and `newConeUVSolid`, whose only
callers were the drivers; the two closed-form oracles they wrapped stay, reached from `ClassifyPoint`.

**The ratchets.** `recognizers` **37 → 11** — the whole fall is the 26, and the 11 left are the
tessellator's `specialCurvedMeshers`, now the only first-fit ladder in the kernel. `fallback-sites`
26 → 24, `tolerance-constants` 226 → 216 (the drivers' own calibrated welds and snaps),
`type-assertions` 727 → 692 (a per-pair recognizer recognises by asserting its operands' surface kinds;
35 of those went). `dispatchLadders` loses its `boolean_curved.go` entry outright.

**What the corpus says.** Every row the drivers' own tests carried was re-pointed at the general entry
rather than deleted, and each came back with the SAME answer:

| family | census through the driver | census through `Boolean` |
| --- | --- | --- |
| cone ∩ cone | 3 cones | 3 cones |
| cone ∩ cylinder | 1 cone + 2 cylinders | same |
| cylinder ∩ cylinder | 3 cylinders | same |
| crossing cut / join | 2 cyl + 2 planes / 3 cyl + 4 planes | same |
| partial ∩ / − / ∪ | 2+1 / 2+3 / 2+3 | same |
| coaxial ball ∪/−/∩ rod, all 8 extents | sphere/cylinder/plane tallies | same |
| shoulder extents, all 8 | sphere/cylinder/plane tallies | same |

Three rows came back BETTER, and each is worth naming because each was a defect the driver carried:

- **Six of the nine arrangement-golden cases tightened.** E fell 5 → 4 and the free-edge count 1 → 0:
  the drivers emitted a seam edge used twice by ONE face, which `structSig` counts as free. Every Euler
  characteristic is unchanged, so it is the same body with one fewer artificial edge. The golden is
  rebaselined against `Boolean`, which is what the kernel now ships.
- **The rim-crossing cut became exact.** Its driver clipped the section loop open at the rim, the
  closed-form path refuses a clipped chain by contract, and the imprint therefore MARCHED — that pair
  was the corpus's only body with a non-zero `AchievedBoundaryTolerance`. The per-face pipeline meets
  each face's own section in closed form (the wall's ruled∩quadric arc, the cap's ellipse, the rim's
  circle), so every edge is analytic and the body reports 0.
- **The near-pinch band lost its ceiling.** Below the stitch resolution the Steinmetz recognizer SNAPPED
  the radii to their mean and emitted the four-lobe bicylinder; the general pipeline does not move
  geometry, so at δ = 0.4·ceiling it emits the honest three-face answer (the thin cylinder's full-wrap
  band plus the fat one's two lens caps) and reserves four lobes for radii that are exactly equal. The
  whole sweep now tracks the analytic crossing-intersection volume to 1e-4, where the old test had to
  allow 4% for its faceted half. `TestBooleanIntersectNearPinchContinuity` carries the new premise.

Nothing regressed: `./kernel/...`, `./model/...`, `./app/...`, `./math/...`, `./addin/...`, archguard
and golangci-lint are green.

**What stage 4 does NOT close.** Several pairs are still refused by name — a torus cut by a drill, a
torus meeting a sphere, a sphere crossing a non-coaxial cylinder — exactly as they were before this
deletion, because no recognizer covered them either. They are stage 5's chart for freeform faces, which
takes `mixed-decline-returns` from 3 to 0.

## Stage 5 — the folded ruled∩quadric window (2026-09-07)

Stage 5 was planned as "a chart for freeform faces, which ends the pass bucket", gated on
`mixed-decline-returns` reaching zero. **Both halves of that were wrong, and measuring first is what
showed it.** Driving every pair the boolean refuses through the per-face pipeline and reading which gate
declined gives:

| pair | pass bucket | gate that declined |
| --- | --- | --- |
| torus − axial drill | empty | closed-surface × wall imprint |
| torus − radial rod | empty | closed-surface × wall imprint |
| torus − sphere | empty | closed-surface × closed-surface imprint |
| sphere ∩ crossing cylinder | empty | closed-surface × wall imprint |
| sphere − cone | empty | closed-surface × wall imprint |

The **pass bucket is empty in every case**: stages 2 and 3 gave the sphere and the torus their charts, so
nothing in the corpus passes through un-charted any more, and the stage's stated content was already
done. And the gate that declines is the IMPRINT — `geom.IntersectSurfacesAnalytic` answering
`handled=false` — not a chart.

The gate is wrong too. `mixed-decline-returns` counts the three sites where the pipeline refuses a pair
BY NAME, and the ground rules require exactly that: "an unsupported configuration is refused at
classification with a named decline". A kernel has a boundary; a count of zero would delete the place
that states where it is. Stage 5 measures itself by which surface-pair families the sections cover.

### What the intersector was missing: the fold

`RuledQuadricArc` follows one ordered root of the ruling quadratic across the base's whole azimuth. That
is the section's shape only while the ruling meets the quadric at EVERY azimuth — a rod driven right
through a wall. A ball sitting off a cylinder's axis is the other shape: the ruling meets it over an
azimuth WINDOW and misses it outside, so the two roots meet where the discriminant vanishes and the
section is ONE closed loop that runs out along the upper root and back along the lower. Those meeting
points are FOLDS, and `dv/du` is infinite at them, so the conditioning gate — which demands the two
branches stay apart across the whole sweep — refused the whole family.

A fold is a defect of the (u, v) GRAPH, not of the curve: the section is smooth in space, tangent to the
ruling as it turns. `RuledQuadricLoop` is that curve, and the whole trick is its parameter. The azimuth
runs u(s) = m − w·cos s over one turn, whose speed vanishes at each fold exactly as fast as dv/du
diverges — both like the square root of the distance to the fold — so dP/ds stays finite. Measured
against a central difference at 65 stations including both folds, the analytic tangent agrees to
cos = 1.000000, and every point of the loop sits within 1e-15 of both surfaces.

Three details earned their comments the hard way:

- **The window's ends are found on the NEGATIVE side of the bisection**, never at the bracket's midpoint.
  At an azimuth where the discriminant is a rounding ABOVE zero the two roots differ by 2√Δ/|a|, and √Δ
  of a number bisected to 1e-20 is still 1e-10 — the loop closed with a 1.55e-7 gap. Taken from the side
  where the roots have already merged, `foldRoot` answers the double root −b/2a for both halves and the
  closure is exact.
- **The window form must refuse a full wrap rather than dress it up.** A discriminant with no sign change
  is the arc form's case, and a "loop" whose two folds are the same azimuth is not a curve.
- **A tangency, a clear pair and an enclosed pair are ANSWERS, not refusals.** Reporting them as
  unhandled would send a pair with no section to the marcher.

### What it unlocked

A ball crossing a rod off its axis — the smallest folded pair — now booleans exactly in all three
operations. Certified against an independent oracle (the two primitives' analytic membership integrated
by Monte Carlo, deliberately not the kernel's own classifier): join 13.0779 against 13.0779, cut 12.5542
against 12.5559, intersect 0.012195 against 0.012187, each inside the integral's own error. The face
census is asserted with the volume, because a body that measures right can still be the wrong shape.
The seam itself is exact — the result reports `AchievedBoundaryTolerance` 0 and every point of the seam
lies within 1e-9 of both surfaces.

That fixture has now been three things in a row, which is the retirement's story on one pair: the
mesh-arrangement rescue's case, then the corpus row for the named refusal once the engines went, and now
an exact result.

### What the first slice did NOT close

The torus pairs — torus × cylinder, torus × cone, torus × sphere, torus × torus. A torus is quartic, so
it has no implicit quadric to substitute a ruling into and neither ruled form applies. The marcher would
trace them (measured: two closed loops each, deviation ~1e-3, no diagnostics) — but wiring it back into
the boolean would re-open a door stages 6 and 7 closed. The boolean is exact-or-refuse today, and that
is the better property; the torus family wants a closed form, not an approximation.

It got one: the second slice below reduces the torus against an axis-invariant quadric to a single
harmonic. What that leaves refused is the pairs where the quadric is NOT axis-invariant — a rod across
the ring, a tilted drill, another torus. Deleting the last recognizer left `coneCylinderImprint` — a one-cone-one-cylinder
type guard in front of the general trace — with no caller, and it went the same way as the 26.

## Stage 5, second slice — the torus reduction (2026-09-07)

The folded window closed the ruled × quadric family; what stayed refused was every pair with a torus in
it. A torus is quartic, so it has no implicit quadric of its own and the ruled bucket cannot reach it —
which is where the reasoning stopped last time, and it stopped one step early.

**The substitution runs the other way.** A point on a torus is

```text
P(u, v) = C + ρ(v)·e(u) + r·sin v·â,   ρ(v) = R + r·cos v,   e(u) = cos u·ê₁ + sin u·ê₂
```

which is AFFINE in e(u) — exactly the shape a ruled surface has in its ruling parameter, with the tube
angle as the station and the azimuth as the unknown. Substituting it into a quadric leaves

```text
A(v) + ρ(v)·(T(v)·e(u)) + (e·Me)·ρ(v)² = 0
```

and when the quadric's quadratic form M is INVARIANT about the torus axis, e·Me is a constant and the
whole azimuth dependence collapses to one harmonic |T⊥|·cos(u − ψ). One harmonic has a closed form:
u = ψ ± arccos(−A/(ρ|T⊥|)), two ordered azimuths wherever |A| ≤ ρ|T⊥|.

The family that reaches is every quadric whose M commutes with rotation about the torus axis — a SPHERE
anywhere, and a cylinder or cone whose axis is PARALLEL to the torus's. In CAD terms: a ball meeting a
ring, and an axial hole or boss through one. The gate is a test on the TENSOR, not on the surface's
type, so the same cone passes coaxial and fails tilted.

The topology is then the question the ruled bucket already asks, and it is now asked once for both:
`periodicRootWindows` finds the spans where the two roots exist and refines the ends to the folds. Three
shapes come out of it, and the third is the one a type-driven dispatch would have had to special-case:

- the quadric reaches the tube at EVERY station → two full-period `TorusQuadricArc` branches;
- it reaches part of the turn → one folded `TorusQuadricLoop` per window;
- it is COAXIAL, so the constraint has no azimuth dependence at all → whole CIRCLES at the stations that
  satisfy it, which is a shaft standing in the ring's hole.

Every point of every one of them sits within 1e-12 of both surfaces, the loops close exactly, and the
tangents agree with a central difference to cos 1.000000 through the folds.

### One trap worth recording

`kernel/geom/periodic_root_windows.go` compiled, formatted and was silently EXCLUDED from every build:
Go reads a `_windows.go` suffix as a GOOS constraint. The symptom is "undefined" errors for functions
that plainly exist in the same package. The file is `periodic_root_spans.go`.

The other was the azimuth's branch cut. An azimuth read from an arctangent carries an arbitrary whole
turn, so a difference quotient across the cut reported a 2π jump as an infinite derivative — tangents of
1e7 on a curve whose real speed is 5. Differences of two such readings are only meaningful modulo a
turn (`shortestTurnDelta`).

### What it unlocked, and the gap it exposed

A ball meeting a ring booleans exactly in all three operations. The three results are checked against
EACH OTHER by Requicha's identity — V(∪) + V(∩) = V(ring) + V(ball) and V(−) + V(∩) = V(ring) — which
three bodies built by three separate trims of the same section have no reason to satisfy unless the
section is right; they agree to a part in a million, and the ring's own integral matches 2π²Rr². A
coaxial shaft bored through a ring is exact too.

An AXIAL DRILL through a ring — a flange's bolt hole, and the commonest thing anyone does to a torus —
was a NAMED REFUSAL for one commit, and the cause is worth writing down because the symptom pointed
nowhere near it.

The section was exact from the start: two closed seams, each wrapping the drill's azimuth once, both
inside its band. But the ruled chart's trim kept only HALF the bore wall, splitting each seam at the two
azimuths where it reaches its extreme height and bridging with two rulings. Two shells, open boundary,
refused by the acceptance gate.

The probe that mattered compared the failing case against a WORKING analogue with the same shape — a rod
crossing a fat cylinder, whose wall also carries two wrapping imprints. Same seam azimuth, same band,
same bucket. The difference was one number: the rod's wall reported FOUR seam crossings and the drill's
TWO. The two missing ones were the imprints' own crossings with the chart's seam.

**`geom.CurveIncidence` knew a `RuledQuadricArc`'s two implicit conditions and none of the three section
curves added after it** — the folded ruled loop and both torus forms. `curvePairMeets` needs roots on
BOTH curves and pairs them by distance, so a curve that reports no incidence yields no crossing at all,
silently. The fix is three dispatch cases: a ruled section is on its quadric and on its base's implicit
form, a torus section is on its quadric and on the torus's own signed distance.

An intermediate probe ruled out everything else first: sampling the same seams as plain POLYLINES failed
identically, so it was never the new curve type or its parameterisation — a polyline has no incidence
either.

Every axial drill through a ring now comes out as two faces in one closed shell, and the three
operations satisfy Requicha against the operands' own analytic volumes.
`TestEverySectionCurveReportsItsIncidence` walks every section form the intersector can return and fails
when the next one arrives without its conditions.

## What stage 5 handed downstream: the mesh (2026-09-08)

The bodies stage 5 made buildable are exact B-reps, certified against independent oracles. Their MESHES
are not. Measured at default quality, against each body's analytic volume:

| body | free mesh edges | mesh volume | analytic | error |
| --- | --- | --- | --- | --- |
| bare torus (control) | 0 | 219.23 | 222.07 | 1.3% (chord) |
| drilled plate (control) | 0 | 175.03 | 174.87 | 0.1% |
| ring − axial drill | 64 | 215.61 | 216.26 | 0.3% |
| ring ∪ ball | 0 | 252.20 | 232.34 | 8.6% |
| ring − ball | 64 | 211.12 | 198.20 | 6.5% |
| **ring ∩ ball** | 64 | **227.33** | **23.87** | **852%** |
| rod ∪ ball (folded window) | 28 | 11.87 | 13.08 | 9.2% |
| ring − coaxial shaft | 64 | 144.78 | 203.59 | 29% |

The lens says what is happening: the curved-face router ends at the surface's WHOLE parametric domain
for a boundary no wrapping mesher recognised, and the ring∩ball lens is a torus band wrapping the tube
between two section curves that none of them charts. So the mesh is the entire torus — 227 mm³ where the
face carries 24 — and the face's own boundary is absent, which is where the free edges come from.

This is not a regression: no boolean could build these pairs before, and the tessellator's limitation
for torus trims is long-standing (its meshers are an eleven-entry first-fit ladder keyed on the shape a
face's boundary makes, which is the last such ladder in the kernel). But it shipped SILENTLY, and the
ground rules do not allow that: "a fallback, approximation, or dropped element is a `diag.Defect` that
reaches feature health, the API and the UI". `CodeTrimIgnoredFullDomain` now says it, on exactly the
faces whose trim was discarded and on no untrimmed one — a bare torus IS its whole domain and the same
grid is right there. `fallback-sites` rises 24 → 25 for it, which is what the ratchet exists to allow.

**A chart-driven mesher is the fix, and it has two constraints that a first attempt taught.** Meshing
each face from the `(u, v)` contours the boolean already records on it (ADR-0063) gets the REGION right
— the lens came back at 23.15 against 23.87, a 3% chord deficit like any other face. But it broke
watertightness everywhere it touched, because a face's boundary must be discretised identically on both
sides of every shared edge, and the chart's own sampling is not the tessellated edge's. So the mesher
has to take the REGION from the chart and the POINTS from the shared edges.

### The tube-wrapping band, generalised

One family already had a mesher built that way, and generalising it takes the lens with it. The spiric
band loft meshes the strip a plane parallel to the axis leaves swept around the tube: each boundary row
is the EXACT discretisation of its edge (so it welds to whatever meets it there) and only the interior
rows are lofted. What it would not take was any boundary that is not a `SpiricArc` — it asked whether
two spiric arcs were the opposite roots of ONE plane's section, which is a question about curve KIND.

It now asks about the SHAPE: a torus face with exactly two edges that each go the whole way round the
tube. That test also subsumes the guard the old one needed for its own reason — an arc fillet run out on
a side plane at each end carries one QUARTER-tube section per end, and lofting between those sweeps the
whole tube (measured on simple/W2, whose 0.418 band read 4.9146). Neither wraps, so neither reaches it.

| body | before | after |
| --- | --- | --- |
| ring ∩ ball | 852% high, 64 free edges | **1.3% chord deficit, watertight** |
| ring − ball | 6.5% high, 64 free edges | **1.1% chord deficit, watertight** |
| torus − box (two-oval band) | 1.1% low | unchanged |
| torus − box (figure-eight pinch) | 4.4% low | unchanged |

Three things had to be right, and each was wrong first:

- **Which of the two bands.** A pair of tube-wrapping boundaries bounds two, and they are
  complementary — so a loft that always takes the same one meshes a cut and its intersect IDENTICALLY,
  which is what the spiric-only version did (both came back 246.6967 on a torus of 394.78). The chart
  decides it, by AREA rather than by containment: the chart's own (u,v) area over the tube period is the
  region's mean azimuth width, and the two candidates' widths sum to a period. Containment cannot answer
  it, because the complement band wraps the azimuth and the chart records it as two contours split at
  its own seam; an even-odd test folds the query onto each contour's branch separately and a point can
  land inside BOTH, which reads as outside.
- **The width must be folded onto one period at every station.** An azimuth read from a boundary's own
  samples carries an arbitrary whole turn, so a raw difference is a period out at one station and not
  the next; the loft then varied its width by 2π and covered the tube more than once — 636 mm³ on a
  torus of 395.
- **The backward travel is the REST of the period, not the folded reverse difference.** They differ by
  exactly one case, and it is the one that matters: where the two boundaries MEET, the forward gap is
  zero and the backward travel is a whole period. Folding the reversed difference answers zero there and
  the band collapses at precisely the station where it is widest — the figure-eight's tangency, which
  came back with 67 free edges.

It also reaches two bodies that already had a mesher: J3 and A4, the spiric closed-rim canal hosts, whose
torus faces the loft now claims before the denser triangulation downstream of it does. Same geometry to
five decimal places — J3 7 395 243.913 against 7 395 592.452, A4 15 408 786.198 against 15 409 136.953,
both still watertight — at **a third of the triangles** (1 115 132 → 340 988 and 1 180 684 → 406 540). A
loft that carries each boundary's exact edge discretisation and fills between them needs far fewer than a
triangulation that re-covers the whole trim, and the volumes say it loses nothing. Their byte-identity
fingerprints are rebaselined with that measurement beside them.

What is left on the full-domain path is a band wrapping the ring's AZIMUTH rather than its tube (a
coaxial shaft bored through a ring), and the sphere and cylinder faces of the folded-window family.
`CodeTrimIgnoredFullDomain` reports each of them.

### The chart-driven mesher: region from the chart, points from the shared edges (2026-09-08)

The two constraints the first attempt taught are met by keeping them in different places. The REGION is
the face's carried chart (ADR-0063) — the closed `(u, v)` contours the boolean recorded, which are the
only record of which of the two regions a pair of wrapping rims bounds. The mesh vertices ON the
boundary are the SHARED EDGE discretisations and nothing else, so a face's boundary is identical on both
sides of every edge it shares. `chartFaceMesh` (`kernel/ops/tessellate/chart_face_*.go`) is that mesher,
and it takes any charted face whose surface wraps in one direction or both.

It works in the COVERING space rather than in a cut branch, and that is the whole of why it is general:
the chart's contours already close there, so no seam is cut, no keyhole is assembled and no boundary
point is invented. The construction is the one the periodic B-spline cover already used (#1510): lift
each boundary loop onto the chart's branch, replicate the boundary and an interior grid one period
either way, triangulate the lot ONCE with the boundary segments as constraints, and keep each triangle
whose centroid lies in the chart's own half-open window and on its material side. Period-shifted copies
of a boundary point are the SAME 3D point, so welding closes every seam; a sphere pole's row welds to
one vertex and its degenerate triangles drop out.

| body | before | after |
| --- | --- | --- |
| RS− ring − coaxial shaft | 144.78 vs 203.59 (29% low), 64 free edges, trim reported | **201.18 (1.2%), watertight, nothing reported** |
| RD− ring − axial drill | whole torus grid, SILENTLY, 64 free edges | **213.47 vs 216.26 (1.3%), watertight** |
| RODB∪ rod ∪ ball (sphere face) | whole ball, 28 free edges, trim reported | **watertight, nothing reported** (11.94 vs 13.08) |
| torus − axis-parallel half space | 201.07 vs 203.90 (1.4%), by the window mesher | **201.63 (1.1%), by the chart** |
| RING / bare sphere / drilled plate | unchanged controls | unchanged |

**Deleted.** `torus_complement_mesh.go` — the genus-1 torus complement's window-and-patch construction,
which charted the torus on a window centred on ONE oval, filled the window minus the oval with a local
patch, and fell to the full torus grid, silently, for a second window. It is exactly a charted outerless
face, and the measurement above is the proof the general mesher reproduces it. The router's
`s.(geom.Torus)` went with it: an outerless face on ANY periodic surface is meshed from what the face
RECORDS, not from what its surface is. `type-assertions` 692 → 691, `geomSwitchDebt[kernel/ops/tessellate]`
53 → 52; `fallback-sites` unmoved, because a face that carries no chart still reaches the reporter.

**Three things had to be right, and each was wrong first.**

- **A wrapping region's chart arrives SPLIT at the arrangement's own seam.** The bored ring records its
  torus face as two disjoint POSITIVE rectangles, `v ∈ [3.98, 2π]` and `v ∈ [0, 2.30]`, which are one
  band through the v seam. Even-odd over all contours together reads it correctly; reading the first as
  an outer and the second as its hole inverts the face.
- **An even-odd count ON a border answers by which side the ray was cast from.** A sphere's `v = +π/2`
  pole row read OUTSIDE while `v = −π/2` read inside, and the cap around the north pole came back
  missing — 32 free edges on the rod ∪ ball sphere. A station at a bounded axis's end asks half a grid
  gap inward instead, which is the cell it bounds and not a tolerance.
- **Between the chart's fine sampling of a boundary curve and the mesh's own coarse chord lies a band
  where the chart does not describe the mesh's boundary at all.** The chart calls a sliver outside the
  chord "inside the hole", the triangles there are dropped, and the rim detours around the gap through
  interior nodes the neighbour face has never heard of — 32 rim edges where the shared oval has 28.
  Keeping every interior node half a chord clear of the boundary puts that band inside the first
  triangle off the rim, whose centroid is then a third of a chord away.

A fourth was a limitation rather than a defect: a STRAIGHT axis (a cylinder or cone's height) needs no
chord subdivision, so the adaptive breakpoints are its two ends and the covering gets no interior row at
all. The triangulation then reaches right across the face for its diagonals — measured on a windowed rod
wall, triangles whose planes passed within 0.1 of the axis and a volume integral of 1.00 where 8.15 is
right. The station count is floored at the package's own `minInteriorCells`, which is the floor
`adaptiveStep` already applies to every step for the same reason.

**What it does not reach yet, measured.** The folded-window family's rod WALL is claimed by
`specialCurvedMeshers` before the router ever gets to the chart (`twoRimHoledBandMesh`, entry 9 of the
last first-fit ladder in the kernel). It meshes 24.47 mm² of wall area with triangles whose planes pass
as close as 0.5 to the axis, so the wall integrates 7.19 where 8.26 is right, and rod ∪ ball and
rod − ball stay 8.7% and 9.0% low. Driving that face through the chart mesher instead measures **1.44%
and 1.43%** — but it needs `twoRimHoledBandMesh`, `HoledConicWallMesh` and `saddleBandLoftMesh` retired
together, which is a slice of its own and the one that finally deletes the ladder.

### 2026-09-08 — the tessellator's last first-fit ladder becomes a classification

`specialCurvedMeshers` (`kernel/ops/tessellate/tessellate_trim_special.go`) was an eleven-entry ordered
try-list: each surface-specific mesher declined so the next could claim the face. It was the last
first-fit ladder in the kernel, and its order was load-bearing — the belt fan had to precede the
gnomonic patch, the notched-rim loft had to precede the holed unroll — with nothing anywhere saying so
except the sequence itself. `classifyCurvedTrim` (`curved_trim_classify.go`) replaces it. It reads the
face's trim once, names the one kind it is, and `specialCurvedMesh` switches on that name; a mesher that
declines on its own conditioning demotes the face to the general path, never to a second special case.

**The proof is a test a ladder cannot pass.** `TestCurvedTrimKindsAreMutuallyExclusive` evaluates EVERY
predicate on every curved face of seventeen corpus bodies and fails if two answer for one face, and it
also asserts that the kind the classification selects IS the one predicate that holds.
`TestTheClassificationCorpusReachesEveryArm` keeps that from going vacuous: each arm the package can
build must appear. For a ladder, two rungs claiming one face is the mechanism, not the defect — which is
why no such test could ever have been written against it.

**Eleven entries, eight arms.** Three entries were never separate recognizers:

| deleted | why it was not a recognizer |
| --- | --- |
| `sphereZoneCapFan`, `sphereSeamedCapFan` | three entries read three RIM FORMS into the same `buildSphereCap`; they are one `kindSphereCapFan` arm whose `sphereCapRim` names the form, and the forms exclude each other by the boundary's own shape (coplanar samples / a lone pole vertex / a doubled seam edge) |
| `notchedRimBandMesh`, `twoClosedRimBandMesh` | both delegated to `saddleBandLoftMesh`; they are one `kindRuledBandLoft` arm |
| `coneApexFan`, `coneSectorFan`, `coneApexSectorMesh` | the two fans differed only in the wrap-around triangle; one `apexFan(cone, rim, closed)` |

`recognizers` 11 → 8. `archguard`'s `dispatchLadders` loses its last entry and `countRecognizers` counts
classification arms alongside ladder entries — a ladder's rungs and a classification's arms measure the
same thing, so the number survives the shape change. Asking each surface family ONCE also collapsed six
geometry-kind assertions: `geomSwitchDebt[kernel/ops/tessellate]` 52 → 46, `type-assertions` 691 → 685.
`fallback-sites` and `tolerance-constants` are unmoved — no `diag.Code` and no tolerance changed hands.

**`kindChart` is now an arm, not a postscript.** A face that records its own parametric trim (ADR-0063)
and that no special shape claims is meshed from that trim directly, instead of travelling the whole
generic (u,v) path to reach the same conclusion at its end. `kindUncharted` takes the identical call and
declines on the spot, because there is no region to build from, so the general path is reached by the
faces that have nothing better — which is what a `default` arm is for.

| row | before | after |
| --- | --- | --- |
| RODB∩ rod ∩ ball | 0.00851 vs 0.012187 (30.20%) | **0.008847 (27.41%)**, 60 triangles, watertight |
| RS− / RD− / RODB∪ / RODB− | 200.903 / 213.473 / 11.937 / 11.424 | unchanged to the digit |
| occtparity fingerprint pins | — | unmoved (byte-identical) |

The rod WALL's lens patch carries a chart and used to chord flat across a lens 0.1 mm deep; it is meshed
from its own region now. What is left of that 27% is the ball's arc-bounded sphere patch.

**Every arm was measured against the chart mesher, and none can be deleted yet.** Each arm was driven
through `chartFaceMesh` on every face it claims across `kernel/ops/tessellate`, `kernel/ops/boolean`,
`kernel/ops/blend` and `kernel/brep` — 573 faces — comparing face area, triangle count and rim edges.
The keeps are numbered, not assumed:

| arm | faces | chart declines | verdict |
| --- | --- | --- | --- |
| `kindConeApexFan` | 9 | 8 | KEEP — identical area (24.55015 both) for 3× the triangles (96 vs 32); the fan is exact on a developable |
| `kindSphereCapFan` | 6 | 5 | KEEP — +0.31% area for **13×** the triangles (130560 vs 9728) |
| `kindSphereZoneBand` | 4 | 4 | KEEP — the chart mesher takes none of them |
| `kindSpherePatch` | 291 | 279 | KEEP — 96% of its faces record no chart |
| `kindRuledBandLoft` | 162 | 105 | KEEP — three faces LOSE area, −7.52% (35.31 → 32.65) and −5.19%/−5.25% (41.18 → 39.05); the ruled loft is exact rim-to-rim and needs no interior row |
| `kindSpiricBand` | 10 | 0 | KEEP — three faces match to ±0.17%, the fourth loses **9.39%** (310.80 → 281.62) |
| `kindTwoRimHoledBand` | 37 | 18 | KEEP on a body-level gap, see below |
| `kindWedgeBand` | 54 | 54 | KEEP — the chart mesher takes none of them |

The seven rungs of `meshSeamCrossingFace` measure the same way and more sharply: with `kindChart` an arm,
every face still reaching `unequalRimBandMesh`, `closedDomainMesh`, `HoledConicWallMesh`,
`saddleBandLoftMesh` or `closedBandLoftMesh` has the chart mesher DECLINE — 133 of 133 over the same
four packages, and 182 of 182 over the whole of `./kernel/...`. Those rungs are, by construction, the
paths for a face that records no region.

**The named gap: `kindTwoRimHoledBand`.** This is the arm the last slice's measurement pointed at, and it
is worth stating exactly where it stops. Per FACE the chart mesher wins: over the 19 charted two-rim
holed bands it matches the unroll to ±0.2% of area on 18 and betters the rod wall by **+1.48%** (24.475 →
24.838 mm², the analytic wall being 24.87) with 40–85% fewer triangles and the same rim count, and
routing them to it moves RODB∪ and RODB− from 8.72%/9.02% to **1.44%/1.43%**. Per BODY it loses, and a
face-local rim count is exactly the certificate that cannot see why: at `PropertyQuality` the corner
junction's wall (#1738) comes back with **870 rim edges against its neighbours' 864**, cracking the body
with 6 free edges, and its area FALLS from 160.93 to 158.65 as the chord tolerance tightens — refinement
is meant to raise it. `TestRimCrossingCutMembershipMatchesCSG` then moves 222 interior points off the
analytic predicate. Until the chart mesher's fine-quality boundary agrees with the neighbour's, the
unroll stays; RODB∪ and RODB− stay pinned at 8.72% and 9.02%, and their pins say so.

### 2026-09-08 — corrections to the curved-trim classification section above

Review found the section above claimed more than it had done, and one of its claims was measurably
wrong. This corrects it; the earlier text stands as written, since this ADR is append-only.

**The sphere family's ladder had been relocated, not eliminated, and the exclusivity proof could not
see it.** The first pass folded five of the eleven rungs into two arms, but underneath, the cap's three
rim readings were still tried in order, and the three "predicates" the exclusivity test evaluated were
`classifySphereTrim(...) == X` — three comparisons against one answer, which cannot both be true, so
the test was a tautology for exactly the part that still contained a ladder. The rim forms are now read
as an INVENTORY: all three evaluated, the number that held counted, and a boundary two forms claim
refused rather than resolved by position. `TestSphereCapRimFormsAreDisjoint` evaluates the three
independently on every corpus sphere face, and `TestTheTwoSeamedRimFormsAreDisjoint` holds them to the
one shape that used to collide.

**Two of the three rim forms were NOT disjoint, and the ladder had been hiding it.** A loop of
`[seam, ONE full circle, seam-reversed]` reads as a pole-seamed rim (one full-circle edge plus a lone
pole vertex) AND as a seamed multi-arc rim (a lone doubled edge plus a coplanar rim ring). It is OCCT
blend/simple J2's own sphere face. Refusing the ambiguity moved J2's byte-identity fingerprint from
165886 triangles to 33991 (and A3's from 305166 to 544658) — the fingerprint pins caught what no
reasoning had. The discriminator is the rim EDGE COUNT, which is what the multi-arc form's name has
always claimed: one rim edge is the pole-seamed form, two or more the multi-arc form. With it both
fingerprints are byte-identical again, and the ladder's old outcome is reproduced exactly, because the
rung it tried first is the form that now holds alone.

**`recognizers` 11 → 8 was a change of counting horizon, not a deletion.** The number is re-derived and
the pin is **12**, a RISE that is a correction of the measurement rather than new code. A ladder ENTRY
was never one recognizer: entry 0 recognized two cone shapes, and three entries read three rim forms
into one builder. Counting the bespoke SHAPE recognizers behind the arms (`curvedTrimRecognizers` in
`archguard/kernel_net_delta_test.go`, whose names must all be functions of `kernel/ops/tessellate` and
whose keys must all be cases of the classification's switch) gives **12 before this slice and 12
after**. Nothing bespoke was deleted. What was deleted is builder duplication — `coneApexFan` and
`coneSectorFan` became one `apexFan`; `sphereZoneCapFan`, `sphereSeamedCapFan`, `notchedRimBandMesh`,
`twoClosedRimBandMesh` and `SphereCapFan` became arms over shared builders. The number will fall when a
SHAPE goes, which is what it is for.

**Every recognizer was running twice per curved face.** The classification computed the cone rim, the
cap rim, the tube-wrapping edges and the wedge end chains and threw the results away; the selected
mesher computed them again. `classifyCurvedTrim` now returns a `curvedTrim` carrying the recognition
alongside the kind, and the switch hands it to the builder — "decide each incidence once and reuse the
result". `tubeWrappingEdges` and `wedgeBandEndChains` return the torus and the cylinder they decided,
so `spiricBandMesh`'s `t, _ := s.(geom.Torus)` — a second assertion of a kind already decided, with the
verdict discarded — is gone. `geomSwitchDebt[kernel/ops/tessellate]` 52 → **45**, `type-assertions`
691 → **684**.

**The merged ruled-band arm had narrowed one branch.** The old `twoClosedRimBandMesh` carried no lens
guard; the merge ANDed `!faceHasLensHole` onto both forms, so a developable side whose two closed edges
include a lens (a drilled cone apex cap) would have stopped reaching the loft. The guard belongs to the
NOTCHED form alone — the saddle loft pools all open edges into one rim, so only a notched band can fold
a lens into its base rim (#1591) — and it is scoped back to that form. No corpus row argued for the
narrowing, so behaviour is preserved rather than "fixed".

**A ladder's second shape is now guarded.** `TestNoFirstFitDispatchLadders` matches a range loop over a
table of funcs; writing the same rungs out as consecutive `if x, ok := recognise(...); ok { return x }`
statements is the identical mechanism and was invisible to it — which is how the sphere ladder survived
the first pass. `TestNoUnprovenPayloadGatedChains` (`archguard/payload_gated_chain_test.go`) detects
that shape. It is a REGISTRY rather than a ban, because the shape alone cannot distinguish a ladder
from a classification whose recognizers are proved disjoint, and `classifyCurvedTrim` is the latter. It
is scoped to `kernel/ops/tessellate`: kernel-wide the shape has **16** instances, 12 of them in
`kernel/brep` and `kernel/ops/blend`, and registering those from here would make the guard a source of
merge conflicts with work that owns those packages rather than a ratchet. Widening it is the natural
follow-up once those packages get their own disjointness proofs.

**The `kindSphereCapFan` KEEP rests on the wrong reason above.** It stands on the chart mesher
declining 5 of its 6 corpus faces, not on triangle count: where the chart mesher DID take a cap it read
+0.31% of area, which on a convex cap is CLOSER to the closed form, not further. Triangle count is a
cost, not a correctness argument, and the section above used it as one.

### Stage 5, third slice — the torus's SECOND harmonic (2026-09-08)

The second slice above reduced a torus against a quadric whose quadratic form `M` is INVARIANT about
the torus axis, and said plainly what it could not reach: "a skew rod is not in it: its M is not
axis-invariant, the u-dependence is a second harmonic, and the roots are a quartic in tan(u/2) rather
than an arccos". That is now the general case, and the arccos is the sub-family it collapses to.

**The dependence the first slice dropped.** Writing the torus point as `W₀ + ρ·e(u)` and substituting
into `Q(X) = W·MW + 2G·W + K`,

```text
Q = (W₀·MW₀ + 2 G·W₀ + K) + ρ·(e·T) + ρ²·(e·Me),   T = 2(M W₀ + G)
```

The term LINEAR in `e` is `x·cos u + y·sin u` — one harmonic, and the only one the first slice kept.
The term QUADRATIC in `e` is

```text
e·Me = m₁₁·cos²u + 2 m₁₂·cos u·sin u + m₂₂·sin²u
     = (m₁₁+m₂₂)/2 + ((m₁₁−m₂₂)/2)·cos 2u + m₁₂·sin 2u
```

with `m₁₁ = ê₁·Mê₁`, `m₂₂ = ê₂·Mê₂`, `m₁₂ = ê₁·Mê₂` — a constant plus a SECOND harmonic that vanishes
exactly when `m₁₁ = m₂₂` and `m₁₂ = 0`, which is `quadricIsAxisInvariant`. So the reduction the first
slice found was the general one with two coefficients set to zero, and the general one is

```text
f(u) = Level + Cos1·cos u + Sin1·sin u + Cos2·cos 2u + Sin2·sin 2u = 0
Cos2 = ρ²(m₁₁−m₂₂)/2   Sin2 = ρ²·m₁₂   Cos1 = ρ·x   Sin1 = ρ·y
Level = W₀·MW₀ + 2 G·W₀ + K + ρ²(m₁₁+m₂₂)/2
```

verified against `q.ValueAt(t.PointAt(u,v))` at random `(u, v)` to 1e-12 relative for a sphere, an axial
drill, a rod across the ring, a tilted drill and a tilted cone. The azimuths are the same Weierstrass
quartic the conic×conic bucket already solves (`trigQuadraticRoots`); this slice added no second solver.

**The real question was not the roots, it was which two of them are one branch.** The one-harmonic
station has exactly two azimuths, ordered by construction. A second-harmonic station has up to FOUR: an
infinite rod driven across a ring pierces the tube on BOTH of the ring's flanks, so every tube angle it
reaches carries two disjoint pairs. A pair is named by the EXTREMUM it straddles — `df/du` is a station
polynomial of the same shape, so the same solver finds the extrema — and the pair MERGES onto that
extremum at its fold, which is exactly what `phase ± arccos` does as the arccos falls to zero. The
lane's discriminant

```text
min(−f(c)·f(c⁻), −f(c)·f(c⁺))
```

with `c` the lane's extremum and `c⁻, c⁺` its neighbours, IS `reach² − level²` for a one-harmonic
station (whose only extrema are its peak and its trough). So `periodicRootWindows` answers the
window/fold topology for both buckets unchanged, and both build the same `TorusQuadricLoop`; the loop
gained one field, the lane label, and its azimuth reader classifies the station once and takes exactly
one reduction.

Three things are certified rather than assumed, because each of them is a place a guess would ship a
wrong body:

- **every azimuth by its residual** on the station polynomial, after a Newton polish in the ANGLE (the
  `tan(u/2)` chart is ill-conditioned near the half-turn, where a root sits at a nearly infinite `t`);
- **that a lane is a lane**: when four azimuths are present EVERY extremum has flanks of the opposite
  sign, so the COMPLEMENTARY pairing reports the same windows. It is rejected because its own extremum
  does not vanish at the fold — a comparison of computed values against each other, with no floor;
- **that the extremum tracks stay separable** across the tube's whole turn. They are seeded at one
  station and followed by nearest extremum; a count that changes, a drift past half the seeds' own
  spacing, or two seeds claiming one extremum is a pairing this form cannot name, and the pair demotes
  to the general marcher rather than being named wrongly.

A lane whose pair exists at EVERY station is the full-turn topology — four independent branches, not a
folded pair — and declines by name too. It is not in the corpus and no row needs it.

#### What the second harmonic unlocked

| body | analytic volume | oracle | census |
| --- | --- | --- | --- |
| ring ∪ rod across the ring | 241.4230 | Requicha 1e-9, membership 4e-4 | 1 torus + 2 cyl + 2 plane |
| ring − rod across the ring | 213.1487 | Requicha 1e-9, membership 3e-4 | 1 torus + 1 cyl |
| ring ∩ rod across the ring | 8.9174 | membership 6e-4 | 2 torus + 1 cyl |
| ring ∪ tilted drill | 236.3615 | Requicha 1e-9, membership 5e-4 | 1 torus + 2 cyl + 2 plane |
| ring − tilted drill | 216.2553 | Requicha 1e-9, membership 3e-4 | 1 torus + 1 cyl |
| ring ∩ tilted drill | 5.8108 | membership 8e-4 | 2 torus + 1 cyl |
| ring ∪ off-centre countersink | 247.3034 | Requicha 1e-9, membership 2e-4 | 1 torus + 1 cone + 1 plane |
| ring − off-centre countersink | 216.5158 | Requicha 1e-9, membership 3e-4 | 1 torus + 1 cone + 1 plane |
| ring ∩ off-centre countersink | 5.5503 | 1-D quadrature 5.5502832 | 1 torus + 1 cone + 1 plane |

Every one reports `AchievedBoundaryTolerance() == 0`: the section is a closed form and says so. The
countersink row is the REPRODUCTION row — its axis is parallel to the ring's, so its tensor is
axis-invariant however far off-centre it sits, and it must still take the arccos. Its level is proven
equal to the one-harmonic form's bit for bit, and its two azimuths to 1e-9.

`TestSkewRodThroughARingIsRefusedByName` is gone, and with it the two other fixtures that pinned the
same refusal. All three moved to the pair that genuinely has no closed form — two interlocked TORI,
where NEITHER side supplies an implicit quadric to substitute a chart into. `mixed-decline-returns`
stays at 3: the refusal SITE is unchanged, only what reaches it.

#### Two traps, both of which returned an empty answer rather than a wrong one

**A quartic whose leading coefficient vanishes divided by zero.** `RealQuarticRoots` needs `c4 ≠ 0`,
and the Weierstrass substitution makes `c4` the equation's value at the half-turn — zero whenever a
root sits there. `trigQuadraticRoots` already recovered that one root by hand, so the degeneracy was
KNOWN; what nobody had checked is what the rest of the solve does with it. It produced `±Inf`, every
other root came back `NaN`, and a `NaN` fails every "is this root real" comparison it is put through,
so the station returned no roots at all. The rod across a ring hits it at every tube station: its
derivative is `−42.25·sin 2u`, whose quartic is a cubic. `realRootsUpToQuartic` deflates to the cubic,
the quadratic or the line, both conic callers take it, and the regression row asserts no returned root
is non-finite.

**The membership oracle had a bias its own error bars denied.** Sampling triples from one `math/rand`
stream, the countersink lens read 5.5305, 5.5535 and 5.5536 from three seeds — a spread of 1e-3 that
did NOT shrink with the sample count, against an exact 5.5502832 obtained by a one-dimensional
quadrature of the disc-against-annulus area per height. That is the generator's 3D lattice structure,
not sampling noise, and at the tolerance this slice needed it would have read as a kernel error. The
oracle now samples a JITTERED LATTICE — one point per cell of a 160³ stratification — which lands
within 2e-5 on the same row. An oracle that gates a result must be more exact than the result it gates.

#### The mesh, measured — the handoff to the tessellation slice

The B-reps are exact and certified above. Their MESHES are the next slice's problem, and this is where
they stand at default quality (the bare torus control is 1.3% low from chord inscription):

| body | free mesh edges | mesh volume | analytic | error |
| --- | --- | --- | --- | --- |
| bare torus (control) | 0 | 219.2269 | 222.0661 | 1.28% |
| ring ∪ rod across the ring | 64 | 241.6298 | 241.4230 | 0.09% |
| ring − rod across the ring | 64 | 213.6862 | 213.1487 | 0.25% |
| ring ∩ rod across the ring | 0 | 8.7249 | 8.9174 | 2.16% |
| ring ∪ tilted drill | 64 | 235.5702 | 236.3615 | 0.33% |
| ring − tilted drill | 64 | 215.6019 | 216.2553 | 0.30% |
| ring ∩ tilted drill | 0 | 5.7069 | 5.8108 | 1.79% |
| ring ∪ off-centre countersink | 0 | 244.2205 | 247.3034 | 1.25% |
| ring − off-centre countersink | 0 | 213.6195 | 216.5158 | 1.34% |
| ring ∩ off-centre countersink | 0 | 5.3972 | 5.5503 | 2.76% |

The volumes are right everywhere — no row is out by a factor, which is what a wrong REGION looks like.
What is not right is watertightness on the six rows that leave the ring's own surface holed by a
through-bore: 64 free edges each, the same count and the same cause the existing `ring − axial drill`
row carries (measured here unchanged at 64, 215.6087 against 216.2569). The ring's surface with two
bore seams as holes is not a tube-wrapping band, so the loft above does not claim it and the trim ends
on the full parametric domain. The three countersink rows and the three lens rows are watertight,
because their torus faces are a single-hole trim and a wrapping band respectively.

#### A pre-existing defect this slice surfaced, and did not cause

Driving the tilted drill through a range of lengths, ~20% of the JOINs are refused by name: the mixed
per-face boolean returns a body with two shells and open edges, the acceptance gate refuses it, and
the operation declines. It is NOT this slice's: the identical failure reproduces on the ONE-harmonic
path, which this slice does not touch, and it reproduces on the wave base `c1e8f2a8` in a clean tree.
The smallest reproducer is an AXIS-INVARIANT one:

```text
ring  = brep.SolidTorus(P3(0,0,0), V3(0,0,1), 5, 1.5)
drill = brep.SolidCylinder(P3(5.3,0,-h/2), V3(0,0,1), 0.8, h)   // 21 of 21 heights refuse the JOIN
drill = brep.SolidCylinder(P3(5.5,0,-h/2), V3(0,0,1), 0.8, h)   // 21 of 21 refuse
drill = brep.SolidCylinder(P3(5.0,0,-h/2), V3(0,0,1), 0.8, h)   // 21 of 21 succeed
drill = brep.SolidCylinder(P3(5.4,0,-h/2), V3(0,0,1), 0.8, h)   // 21 of 21 succeed
```

The seam solve is not the cause — the wall's seam crossings and its split faces are identical between a
succeeding and a failing case (4 crossings, 2 faces each) — so it is downstream of the wall, in the
stitch. The corpus's tilted drill is 10 long, which is clear of it; the defect needs its own slice.

#### The measurement

| | before | after |
| --- | --- | --- |
| torus × quadric families solved | axis-invariant M only | every quadric M |
| azimuths a station may carry | 2 | 4, paired into lanes |
| curve forms | `TorusQuadricArc`, `TorusQuadricLoop` | unchanged — the loop carries both reductions |
| `CurveKind` values | 16 | 16 |
| section solvers | 1 (arccos) + the shared window finder | 1 (arccos) + 1 (quartic) + the SAME window finder |
| polynomial root solvers in `kernel/geom` | 1 | 1 |
| `mixed-decline-returns` | 3 | 3 |
| `tolerance-constants` / `type-assertions` | 214 / 692 | 214 / 692 |

Nothing was deleted, because nothing was replaced: the arccos is a sub-family of the new reduction and
stays as its fast path, gated on the tensor rather than on a type, and proven to reproduce bit for bit.
What was deleted is the REFUSAL — three fixtures that pinned "a skew rod through a ring cannot be
built", now three fixtures that pin the exact result.

#### Review round 1: four things the slice above got wrong (2026-09-08, later)

**A degenerate station fabricated an azimuth.** `torusLaneAt` answered a station with no azimuth
dependence by returning the ANCHOR as both branches. The anchor is a seed extremum from tube angle 0 —
not a root of this station — so a loop evaluated there would put a point on the torus that is not on the
quadric, silently, while the doc comment claimed every azimuth it reports is certified. It now returns
`ok=false`; the section declines by name, the branch-gap reader answers a zero gap (which fails the
conditioning gate), and the per-point reader answers NaN, which is what `torusHarmonic.root` already
does for the same reason. A plausible-looking wrong azimuth is worse than none.

**The demotion was not reported.** All the refusals collapsed into one anonymous `ok=false`, which the
ADR paragraph above described and nothing else did. But the ground rule is not "write it down": a
fallback is a `diag.Defect` that reaches feature health, the API and the UI. The problem is that the
intersector refuses two DIFFERENT things with that one bool — "no bucket claims this pair", which is the
ordinary case and no loss at all, and a CONDITIONING demotion, where the closed form applies and cannot
name its answer at these numbers. `geom.SectionDecline` now names the reason,
`IntersectSurfacesAnalyticDeclining` returns it, and brep's closed-surface pairings record it as
`CodeSectionConditioningDemotion` (a `diag.Defect`) on the boolean's own recorder. The ordinary refusal
records nothing, because a diagnostic that fires on every marched boolean in the system is noise.
`fallback-sites` rises 25 → 26 for it — a RISE that names a degradation nothing reported before.

**A window was dropped on an unverified premise.** The builder SKIPS a window whose branch pair merges
at a flanking extremum, on the theory that a neighbouring lane carries that pair itself. Nothing checked
that it does, and the test that drives the skip compares three quantities that are all small and
comparable just inside a fold: a mis-fire deletes an entire section loop — a hole in a solid that simply
is not there — with no error and no diagnostic. The finished loop SET is now counted against the
stations themselves: at every probe tube angle, the loops whose window covers it must account for
exactly the azimuths that station carries, two each. A dropped loop leaves two azimuths belonging to
nothing and a doubled one two too many, and `TestADroppedSectionLoopIsCaughtByTheAzimuthCount` removes
each of the rod's four loops in turn and requires the count to catch every one.

**A dead assertion.** `TestALaneStraddlesItsOwnExtremum` compared
`turnBetween(centre, upper, true) + turnBetween(centre, lower, false)` against `separation()`, which is
that expression verbatim — so the branch could never fire and the row never tested straddling at all.
It now asserts the property itself: each azimuth strictly between the lane's extremum and the flanking
one on ITS OWN side. Nothing weaker separates a correct pairing from one that took both roots from the
same side, which would still have two certified roots and a positive discriminant. Proven live by
swapping the two readers, which fails the row.

### The chart mesher at fine quality: boundary parity, a balanced covering, and two conditioning declines (2026-09-08)

Task 2 left the chart-driven mesher measurably better per face than the wrapping arms it was meant to
replace, and measurably worse per BODY at `PropertyQuality`. This slice is the diagnosis of why, and the
four fixes it took. Every number below is `tessellate.TessellateBody` / `TessellateFace` at
`ops.DefaultQuality()` (chord 0.05) and `ops.PropertyQuality()` (chord 0.001).

#### 1. The replication pad was measured on the wrong axis

The covering replicates the branch window a little way either side so the point set is periodic out past
any triangle's circumcircle; that is what makes both sides of the window triangulate the same rim the
same way, and it is what the canonical (half-open) window relies on to keep exactly one of each
seam-spanning triangle. The pad was `chartCoverPadStations` gaps of THAT AXIS'S OWN stations. A straight
axis gets no chord subdivision, so a cylinder wall's covering is three ROWS tall against 256 columns and
its triangles reach the whole height, while the pad is three COLUMN gaps — 0.22 mm on the #1738 corner
junction against circumcircles of millimetres.

Measured there, at `PropertyQuality`, on the wall the classification hands the chart mesher:

| | before | after |
| --- | --- | --- |
| unpaired edges vs its rim | 871 vs 868 | 868 vs 868 |
| rim segments carrying TWO triangles | 3 | 0 |
| free edges that are no rim segment | 4 | 0 |
| accepted by its own rim gate | no (fell back to the arm) | yes |
| face area, Default → Property | 160.933 → declined | 160.933 → 161.155 |

The pad is now `chartCoverPadStations` of the covering's COARSEST cell, taken as a 3D length across both
axes and carried back through each axis's metric, capped at the window. Replicating the WHOLE period is
also correct and was measured: 23.16 s against 1.69 s for the same 97460 triangles on the figure-eight
torus band at `PropertyQuality`, because a doubly-periodic covering has nine shifts. The pad exists to
avoid exactly that, so it stays — measured correctly.

Task 2 also recorded the same face's area FALLING from 160.93 to 158.65 under refinement. That does not
reproduce on this branch: it measures 160.933 → 161.155 both before and after the pad fix. The fall
belonged to an intermediate state of the sweep, not to the code that landed.

#### 2. A straight covering axis had no cell size of its own

With the pad fixed, the corner junction's body was watertight at both facetings and its wall's area
right to 0.07 % — and 2617 interior points of a 60³ membership audit read OUTSIDE the solid they are
inside. The covering's cells were 0.52 mm wide and 5 mm tall, so its triangles spanned the wall's whole
height and their planes cut 0.16 mm INTO a solid of radius 3. No area, volume or watertightness gate can
see that; only a membership oracle can.

`balancedCoverGrid` gives a FLOORED axis — one whose breakpoints are the package's minimum-cell floor,
because it has no chord to resolve — the cell size the other axis's chord asks for, capped at
`maxInteriorCells`. An axis the chord already subdivided is left alone: re-balancing a torus's tube
against its ring drove the figure-eight band from 110.947 mm² to the whole torus, so the rule is scoped
to the axis that has no density of its own.

With it, `TestRimCrossingCutMembershipMatchesCSG`, `TestCapCrossingCutMembershipMatchesCSG`,
`TestConeCapCrossingCutMembershipMatchesCSG`, `TestPartialRimDisjointCutMembershipMatchesCSG` and
`TestPartialRimCornerCutMembershipMatchesCSG` are all green with the wall on the chart mesher.

#### 3. The two-rim holed band is a CONDITIONING arm now, not a shape arm

`nearPinchCorridorChords` is swept, not chosen: failures over
`./kernel/ops/tessellate/ ./kernel/ops/boolean/` by ratio are 0.5→8, 1→5, 2→2, **3→0, 4→0, 6→0, 8→0**,
12→1, 20→1, 40→1. The plateau is 3 … 8 and 4 sits inside it.

`twoRimHoledTrimOf` recognised a shape. It now recognises a shape the general path cannot serve, which
is the ground rules' own test for a fast path. Two configurations qualify, and nothing else:

- a face that records no chart at all — there is no region to mesh from;
- a band whose two lens windows pass within `nearPinchCorridorChords` of the boundary's own chord. The
  chart mesher lays its constraints at the shared edges' discretisation; where two windows are closer
  than a few of those chords the two chord polygons no longer separate the corridor and the constrained
  triangulation loses it (measured on the #1818 near-pinch crossings: 126–2359 unpaired edges against
  rims of 128–2304, and the region as much as 4 % out). The unroll's BENT seam is built for exactly that
  corridor (stage 4).

Corpus rows that move, per BODY:

| row | before | after | analytic |
| --- | --- | --- | --- |
| RODB∪ rod ∪ ball | 11.93691 (8.72 % low) | 12.88969 (**1.44 %**) | 13.077910 |
| RODB− rod − ball | 11.42387 (9.02 % low) | 12.37665 (**1.43 %**) | 12.555898 |
| RODB∩ rod ∩ ball | 0.008847 (27.41 %) | 0.00876 (28.10 %) | 0.012187 |

RODB∪ and RODB− were PINNED two-sided at 8.72 % and 9.02 % with 1.44 %/1.43 % written down as the number
that would move them; it did, and they are bounded rows again at 2 %. RODB∩'s pin is re-measured, not
widened: its lens patch is refined by the balanced grid and a finer faceting of a lens 0.1 mm deep takes
a little more volume out of a body of 0.012 mm³.

The arm is NOT deleted. It keeps 8 corpus faces (16 face-tessellations across the two facetings), all of
them near-pinch crossing joins, and `TestTheTwoRimArmKeepsOnlyTheNearPinchBands` asserts that split in
both directions over the classification corpus.

#### 4. A spiric band that pinches is refused

`spiricBandMesh` sweeps ONE direction round the tube for the whole band, which describes the region only
while the strip between its boundaries has a width everywhere. Against an independent analytic oracle —
the figure-eight (torus R=5 r=2 cut by y=3, tangent to its inner equator), whose two pieces partition the
torus's 394.78418 mm² by integrating r(R+r·cos v) du dv over (5+2cos v)·sin u ≷ 3:

| piece | analytic | the loft | the chart mesher |
| --- | --- | --- | --- |
| below y=3 | 283.09969 | 310.80041 | 281.61993 |
| above y=3 | 111.68448 | 215.17746 | 110.94725 |
| sum | 394.78418 | **525.97787** | 392.56718 |

The loft covered the tangency twice, and its two halves summed to a third more than the whole torus —
which no body volume on either piece could show. `bandPinches` refuses that configuration before any
geometry is built, on an ARC LENGTH against the rims' own weld tolerance, never a bare angle. The
unpinched spiric bands are untouched: the three other corpus rows and `occtparity`'s J3/A4 fingerprints
are byte-identical, so the loft keeps the bodies it reads right — and their third of the triangles.

#### 5. A charted face on a singly-periodic surface never reached the chart

`meshSeamCrossingFace` ended a cylinder/cone/sphere face that no wrapping mesher reduced at the
best-fit-plane CDT, which flattens a band that wraps the seam. Task 5 measured 61 free edges there on
the merged cocylindrical wall a D-prism leaves on a cylinder of its own radius, and read it as "the
chart mesher does not take a singly-periodic surface". It does; it was never asked. `chartFaceMesh` now
takes such a face when it records a region, and the best-fit-plane CDT keeps the sphere cap straddling
the pole, which records none.

The corpus row is built on the FACE shape rather than on that body, because the merge itself is not on
this branch: a cylinder band whose second rim steps axially at ONE azimuth (the chord edges a D-prism
leaves), carrying the chart that region determines. Through the router, at both facetings:

| | before | after | analytic |
| --- | --- | --- | --- |
| free edges vs its rims (Default) | 26 vs 64 | 64 vs 64 | — |
| free edges vs its rims (Property) | 320 vs 584 | 584 vs 584 | — |
| area (Default) | 47.510 | 212.348 | 212.695 |
| area (Property) | 42.601 | 212.689 | 212.695 |

#### What was measured and NOT shipped

Task 5's other recommendation — that `orderedRing` rotate a ring rather than sort it — is not taken, and
the measurement is why. Under the classification a notched band no longer reaches `twoRimHoledBandMesh`
at all (it is `kindRuledBandLoft`, the ruled loft declines a rim that is not a v(u) graph, and it lands
on the router exit fixed above), so `orderedRing`'s only remaining callers are the two band LOFTS, which
stitch their rows BY angle and refuse a rim that is not single-valued in it — for them the sort is the
ordering. Driving the notched-rim fixture through `bridgeRimsAtSeam` both ways measures 78 unpaired edges
with the sort and 81 with a rotation, against a rim of 74: neither is right, and the residual belongs to
the bridge, not to the ordering. `orderedRing`'s doc now says which reader it is for.

#### One pre-existing defect this slice's new gate exposes

`TestEveryCorpusBodyIsWatertightUnderRefinement` meshes every classification-corpus body at both
facetings. One row is not watertight at `PropertyQuality` and is pinned at its measured count with the
diagnosis beside it: "ring − half space", the torus cut by the plane x = R, whose section is the
LEMNISCATE — the two spiric branches MEET at (R, 0, ±r), so the face's boundary passes through the same
3D point twice. The chart mesher returns 276 unpaired edges against a rim of 272, four extra, two per
node; it is declined by its own rim gate and the face falls to the surface's whole domain (296.062 mm²
against the 264.830 it had built), cracking the planar cap with it. PRE-EXISTING, proved by removing
this slice's spiric conditioning gate and re-measuring: identical, 272.

Body VOLUME is deliberately not asserted monotone. A body with a concave curved feature LOSES volume as
it refines, because the faceted bore is inscribed and grows into the solid (the drilled plate
325.419 → 325.262, the conical drill point 345.318 → 345.288). The invariant that does hold is per-FACE
area, and `TestEveryChartedFaceGainsAreaUnderRefinement` asserts it on every charted corpus face.

#### Ratchets

| pin | move | why |
| --- | --- | --- |
| `tolerance-constants` | 214, unmoved | no tolerance changed hands; the pad, the balance and the corridor are mesh-density quantities |
| `type-assertions` | 684, unmoved | nothing added or removed a geometry-kind assertion |
| `recognizers` | 12, unmoved | `lensCorridorOutrunsTheSampling` is a CONDITIONING gate inside `twoRimHoledTrimOf`, not a bespoke shape; the registry counts shapes |
| `fallback-sites` | 26, unmoved | no `diag.Code` added or removed |
| `geomSwitchDebt["kernel/ops/tessellate"]` | 45, unmoved | same |

### Stage 5, fourth slice: the cocylindrical wall merge (2026-09-08)

Two kept faces of one boolean result that lie on ONE surface and share a boundary are ONE face. The
splice that shipped with stage 4 merged them only across a whole shared LOOP or exactly ONE whole
shared edge, and the section above recorded the case it left behind — a D-profile prism seated on a
cylinder of its own radius — with this reading:

> Their common boundary is part of the cylinder's rim, not a whole edge of it, and
> `mergeCoincidentFaces` merges only a whole shared boundary. Splicing a partial one needs more than
> cutting the edge at the run's ends: the two faces' chart SEAMS meet inside the run being dissolved,
> so the merged loop has to fuse those too.

**That reading was wrong, and it is why the earlier attempt was reverted.** Measured on the pair the
merge actually sees — before the stitch, where `mergeCoincidentFaces` runs — the host's wall is a FULL
band whose top rim is ONE closed circle, and the boss's wall is a bare arc band. So the run is a whole
edge on the boss's side and part of an edge on the host's, which no edge-to-edge pairing can match.
Cutting each side at the OTHER's vertices — which the stitch does downstream anyway, and which is the
whole of the "partial boundary" problem — makes it whole on both. Cut first, and the run is **two**
whole edges, because the host's own seam ruling splits it. The thing to generalise was never a partial
splice; it was the assumption that the shared boundary is ONE edge.

**What the merge is now.** Dispatch is unchanged: the pair is decided by `geom.SurfacesCoincide` — the
same "one surface" answer the radial sew makes (ADR-0058) — plus equal sense. Then:

1. `splitAtSharedRunEnds` cuts each face's boundary at the other's vertices that fall inside a run the
   two WALK together, so a vertex that merely lands on the other's curve splits nothing.
2. `sharedEdgeTwins` pairs the shared edges one to one; an edge that would take two partners is a
   boundary subdivided differently on the two sides, and declines.
3. The merged boundary follows each loop's own order and CROSSES at every dissolved edge into the edge
   after its twin. No coordinate decides a successor. That matters: the point where two cocylindrical
   walls' seams meet carries four seam ends, and a "turn left at the vertex" rule would have to pick
   among them.
4. `dropSeamSlits` removes what the crossing orphans — the artificial seam, walked up and straight back
   down, dangling into the merged face's interior. A loop that is nothing but that slit disappears, and
   what is left is the two-rim band the face is.
5. The merged face keeps a's lineage and carries every reference key either parent resolved, and its
   chart is the one its FUSED loops determine (`faceChart`, ADR-0063) — the union of the two trims in
   the covering space, on the branch `loopToUV` unwraps the first loop onto. A pair whose fused loops
   determine no chart is a named decline, `CodeCocylindricalMergeUndecided`, not a chart nobody
   verified. `fallback-sites` rises 26 → 27 for it: a degradation nothing reported before.

**A defect this needed, fixed at its source.** `curveParamWithin` inverted a point and compared the
parameter with the span directly. `CurveParamAt` answers inside the curve's own domain, so a span
running UP TO that domain's end — exactly what a closed rim split at another face's vertices gives —
never matched at its own end: the point comes back as the domain's START, a whole period away.
`paramOnSpanBranch` places the parameter on the branch the span lives on, for a closed curve only. It
is the whole-turn rule this ADR already states for an azimuth, applied where the same inversion is read.

**Before and after**, on the pair the section above named, plus the two rows the merge must not change
and the negative row it must refuse:

| row | before | after |
| --- | --- | --- |
| D-prism on a cylinder of its own radius (JOIN) | 6 faces, 2 cylinder faces | 5 faces, **1** cylinder face |
| rod on rod, abutting cap to cap (JOIN) | 3 faces, 1 cylinder face | unchanged, now carrying its chart |
| rod on rod, overlapping bands (JOIN) | 3 faces, 1 cylinder face | unchanged, now carrying its chart |
| a bore continuing a bore (two CUTs) | 1 cylinder wall | unchanged |
| two bores separated by material (two CUTs) | 2 cylinder walls | **2**, unmerged — they share no edge |

Volume, `Validate` and `AchievedBoundaryTolerance` are unchanged on every row: the merge is
combinatorial, and no coordinate moves.

**Deleted:** `joinedLoops`, `spliceOnSharedEdge`, `rotatedChain`, `sharedEdgePair`, `sharedLoopPair`,
`loopsRunTogether`, `everyEdgeRunsAlong` and brep's `loopsExcept`. The whole-loop form and the
single-edge form are the general dissolve's ordinary cases, so they go with it rather than standing
beside it.

**One gap, named and measured rather than left to be found.** The merged wall's MESH has 4 free edges
where the two-face body had 0, and the cause is downstream of this B-rep. The merged face is a band
whose second rim is NOTCHED — along the host's rim at v = 6 across the boss's flat, along the boss's own
top rim at v = 10 elsewhere, joined by two runs at ONE azimuth each. The tessellation router hands such
a face to `twoRimHoledBandMesh`, whose `bridgeRimsAtSeam` orders each rim with `orderedRing`, a STABLE
SORT BY AZIMUTH. A stable sort keeps a tie's input order, and the rim approaches its two same-azimuth
runs from opposite sides, so one comes out reversed: the ring jumps rim to rim at that corner and the
four triangles around it do not pair. Disabling that mesher is worse — the router then short-circuits at
`IsPeriodic(u) != IsPeriodic(v)` to the flat-patch CDT, 61 free edges with the defect reported — so the
chart-driven mesher this face wants is not reachable for a singly-periodic surface at all. Both belong
to the chart mesher's own router. The count is pinned in
`TestCocylindricalCapOnWallIsOneAnalyticFace` with that diagnosis beside it, so landing the router's
fix trips the row and converts it, exactly as the face count was pinned before this slice.

#### Review round 1: the merge's silent exits, and the body's own mesh post-condition (2026-09-08, later)

Five things the slice above got wrong. Four are the same mistake in different clothes — a degradation
that was measured and written down here instead of being REPORTED where it happens.

**The mesh tear shipped unreported.** The section above pinned the merged band's 4 free edges in a
corpus row and named the router defect that causes them, and stopped there. But the ground rule is not
"write it down": a fallback, approximation or dropped element is a `diag.Defect` that reaches the
result. This is the #2167 piston head, so what shipped was one rendering defect traded for another
while the B-rep row went green. `TessellateBody` now carries its own post-condition —
`recordMeshTear`: when the B-REP says the body is a closed solid and the welded mesh has free edges,
`CodeMeshNotWatertight` records the count and the reference keys of the faces the tear touches. The
closure test reads the B-rep, never the mesh, so it cannot fire on a body that is genuinely open, and
a watertight body records nothing (both are rows). `mergedBodyMesh` keeps each face's triangle span so
the defect can name faces at all; `WeldedFreeEdgeCount` is now the SIZE of what `tornMeshEdges` finds,
so the count and the post-condition can never disagree about what a free edge is.
`fallback-sites` rises 27 → 28 for it — a RISE that names a degradation nothing reported before.

**Three merge exits shipped a silent two-face body.** `sharedEdgeTwins`' ambiguity case, and the walk
that does not close, both returned a bare `false` and fell out of `mergeOnSharedBoundary` with nothing
said — the same shape as the mesh tear, one layer up. Every exit now carries a `mergeDecline` naming
its reason, and `recordMergeDecline` reports all but ONE: two faces that share no boundary are simply
two faces, and a diagnostic that fires on the ordinary case is noise. The reasons are the ambiguous
pairing, the open walk, the undecided chart and the mixed complement below.

**The decline code was a promise, not a guard.** Its test recorded the constant into a throwaway
recorder and asserted it came back, which proves nothing about the merge. Three rows now drive real
pairs to real refusals: a cylinder wall against a face carrying one edge on that wall's own seam (the
wall walks its seam twice, so both traversals run with that edge and the pairing cannot choose), a
torus band whose fused loops bound two regions, and a complement merged with a patch.

**`outerless` was inherited, not decided.** The merged face took a's complement flag. That flag is
precisely the datum ADR-0063 says a face's rings do NOT determine, so inheriting it decides the merged
face's outer loop by which operand happened to be indexed first. A pair that disagrees on it is now
refused by name (`declineMixedComplement`).

**A test with a false premise.** `TestWeldedCutsDropsAStationNamedTwice` offset its second station by
`1e-16`, which is bit-identical to the first in float64, so it exercised the exact duplicate test and
never the weld it was written for. It now uses a station half a weld away — asserting first that the
two differ bitwise — and a third eight welds away that must survive, so the dedup cannot be a blanket
collapse.

#### Review round 2: the post-condition reached nobody (2026-09-08, later still)

Round 1 above claims the mesh tear "reaches feature health, the API and the UI". It did not, and this
corrects it. `recordMeshTear` recorded onto the mesh `TessellateBody` returns, and **every** caller of
that function discards it — the renderer's draw list, mass properties, inertia, the identical-bodies
compare, hidden-line removal, the mesh-format writers, the feature preview all take `mesh, _ :=`.
Worse, the two paths that DO harvest mesh diagnostics never ran the check at all: both
`query.BodyMeshDiagnostics` (the #2058 harvest that feeds a feature reply) and
`model/facetstore` → `CalculateBodyFacets` take the FACES route, `TessellateBodyFaces`. So the piston
head's crack was recorded onto a value nobody read. A defect that reaches no user is the same silence
the round-1 finding was about, one layer further out.

The check now runs in `TessellateBodyFaces` — the one point every route passes through — and records
on the FACE mesh of the lowest-indexed face the tear touches. That single call site reaches all three
consumers: `MergeMesh` carries face diagnostics onto the whole-body mesh, the facet store keeps the
face meshes, and the harvest reads them directly. `TessellateBody` goes back to a plain merge, and the
per-face triangle spans round 1 added to name faces are deleted with it: the tear now knows which mesh
it came from because the walk carries it.

`tornAcrossMeshes` welds the whole GROUP of face meshes at once, which is what makes a body's faces
meet — two faces' copies of one boundary point are separate vertices until they weld — and
`tornMeshEdges` for a single mesh is that same function with one member, so `WeldedFreeEdgeCount` and
the body post-condition still cannot disagree.

Rows that pin it where it has to be true: `query.BodyMeshDiagnostics` carries the code for a torn
closed body and stays silent for a plain cylinder, and `model/feature`'s
`TestPistonHeadMeshTearReachesTheFeaturesDiagnostics` reads the #2167 boss feature's OWN report — the
list a feature reply, the API and the UI show. The measured cost is the group weld on every
tessellation: no change on the bodies that dominate (a filleted box 8.38 → 8.37 ms, a torus 3.31 →
3.43 ms), and tens of microseconds on the smallest (a chamfered box 35 → 63 µs).

### Corrections and a second slice for the chart mesher at fine quality (2026-09-08, review round 1)

Five findings against the section above. Three of them are corrections to what that section CLAIMS; two
are defects it left standing. All numbers are `tessellate.TessellateBody` / `TessellateFace` at
`ops.DefaultQuality()` (chord 0.05) and `ops.PropertyQuality()` (chord 0.001).

#### Correction 1 — the net-delta report was wrong

The ratchets table above says `fallback-sites` "26, unmoved". The pin on this branch is **28**: the
cocylindrical-merge slice raised it 26 → 27 for `CodeCocylindricalMergeUndecided` and 27 → 28 for
`CodeMeshNotWatertight`, both before this work rebased onto it. This slice moves none of the four:

| pin | value at this HEAD | moved by this slice |
| --- | --- | --- |
| `tolerance-constants` | 214 | no |
| `type-assertions` | 684 | no |
| `recognizers` | 12 | no |
| `fallback-sites` | **28** | no |
| `geomSwitchDebt["kernel/ops/tessellate"]` | 45 | no |

`bandPinches` and `tubeSweepRadius` are DELETED this round (below) and were never recognizers;
`lensCorridorOutrunsTheSampling` and `onContour` are conditioning gates inside existing recognizers, not
bespoke shapes, and the registry counts shapes.

#### Correction 2 — the two-rim arm's survivor set and its test's name

The section above names `TestTheTwoRimArmKeepsOnlyTheNearPinchBands`; the test that shipped is
`TestTheTwoRimArmKeepsOnlyWhatTheChartCannotServe`, and it asserts a wider rule than the sentence beside
it. The arm keeps TWO configurations, not one:

- a band whose two lens windows pass within `nearPinchCorridorChords` of the boundary's own chord — the
  8 near-pinch crossing joins (16 face-tessellations across the two facetings);
- a band that records NO chart, which the general path cannot serve at all — one face in the
  classification corpus (the saddle band of `rod − crossing rod`).

#### The spiric arm is a CHART gate now, and `bandPinches` is deleted

The section above kept `kindSpiricBand` for every band and refused only the PINCHED ones, and argued
from a measurement of DELETING the arm (occtparity's J3 and A4 drifting 3×). That measurement tested a
different change from the one the brief asked for, and the section says as much in the same sentence:
those two hosts' faces record no chart, so the brief's rule would have left them on the loft.

Measured, per face, over every spiric band the kernel corpus builds — ten of them, at both facetings:

| face (boundary anchor) | tol | chart | loft area / tris | chart-mesher area / tris |
| --- | --- | --- | --- | --- |
| (6.325, 1.498, 0) | 0.05 | 1 | 24.808532 / 256 | 24.778946 / 206 |
| (6.325, 1.498, 0) | 0.001 | 1 | 24.866088 / 18944 | 24.865105 / 11612 |
| (2.236, 2, 0) | 0.05 | 2 | 248.799622 / 1620 | 248.387145 / 1364 |
| (6.325, 1.498, −0) | 0.05 | 2 | 269.934972 / 2176 | 269.700621 / 1920 |
| (6.325, 1.498, −0) | 0.001 | 2 | 271.205553 / 171008 | 271.197115 / 120184 |
| figure-eight, below y=3 | 0.05 | 2 | 310.800413 (analytic 283.09969) | 281.619928 |
| figure-eight, below y=3 | 0.001 | 2 | — | 283.075286 |
| figure-eight, above y=3 | 0.05 | 2 | 215.177464 (analytic 111.68448) | 110.947245 |
| figure-eight, above y=3 | 0.001 | 2 | — | 111.674711 |

Every one is charted, the chart mesher accepts every one, it reads the same area to within 0.004–0.17 %
on the eight that do not pinch — with a THIRD fewer triangles at PropertyQuality — and it is RIGHT where
the loft was wrong. And the sweep over `model/feature/occtparity` settles what keeps the arm: J3's and
A4's host tori record **chart = 0** and `chartFaceMesh` declines them outright.

So `spiricTubeTrimOf` declines a face that carries a chart, and `bandPinches`/`tubeSweepRadius` — which
existed only to refuse the two charted faces the loft read wrong — are DELETED with their tests. The
fingerprints do not move, because the faces that keep the loft are exactly the uncharted ones.
`kindSpiricBand` is now a named exception in `TestTheClassificationCorpusReachesEveryArm`, beside
`kindWedgeBand`, because no primitive boolean in this package builds an uncharted torus band.

#### The merged cocylindrical band, on the real body

`TestCocylindricalCapOnWallIsOneAnalyticFace` was pinned at 4 free edges by the merge slice and measured
61 under the classification. Three things were wrong, all of them in `kernel/ops/tessellate`:

1. **The router never offered the face its chart.** Fixed in the section above, and it is what takes the
   body from 61 free edges to a mesh at all.
2. **A band's artificial seam can be SLANTED, and the membership test folded onto one branch.** This
   face's bottom rim runs u ∈ [0, 2π] and its notched top rim u ∈ [−0.1963, 6.0868] — two seam edges an
   exact period apart but a fifth of a radian out of plumb — so the contour spans 6.4795 of a 6.2832
   period. Folding a query onto [uLo, uLo+2π) put the sliver between the two seam edges outside every
   contour: the region measured **171.141 mm² against an analytic 174.096**, exactly the seam triangle,
   and the covering tore along it (88 unpaired edges against a rim of 54). `covers` now counts even-odd
   at every period SHIFT; the region reads 174.086 and the face meshes 173.762 with free == rim.
3. **A centroid can land exactly ON that seam.** The slant is eight u-stations over the whole v range,
   so grid-built centroids sit on it to 1e-11 — measured, (−0.008181231, 0.416666667) against a seam at
   −0.008181231 — and an even-odd count there answers by which side the ray was cast from. Forty such
   holes tore the wall at PropertyQuality (615 against 578). `triangleIsMaterial` retries a NO by the
   majority of three points pulled toward the triangle's own vertices, and only when the centroid is
   within `chartContourIncidence` of a contour edge: an ungated retry regressed the cap-crossing,
   rim-crossing and cone-cap certifications (216, 6 and 435 free edges at PropertyQuality).

A fourth was on the FEATURE-built body, whose merged wall arrives as ONE wrapping loop rather than two:
its boundary walks the artificial SLIT twice, so `chainSegmentCount` read 56 where a correct patch bounds
54, declined a mesh that was right, and the wall fell to the flat-patch CDT (57.913 mm² where 173.811 is
its region's own area, and the body reported a 32-edge tear). The count is over segments used an ODD
number of times now.

| row | before | after |
| --- | --- | --- |
| `TestCocylindricalCapOnWallIsOneAnalyticFace` @Default | 61 free edges | **0** |
| the same @Property | — | 10, pinned (below) |
| `TestPistonHeadMeshTearReachesTheFeaturesDiagnostics` | asserted the tear | converted to `…MeshIsWatertightAndSilent`: 0 free edges at BOTH facetings, no diagnostic at all |

The ten at PropertyQuality are a `kernel/brep` fact and are pinned with it: the merged face's own edges
put the notch corners at u = 4.112388980 and 5.312388980 (`ParamAt` of the D-prism's chord vertices,
exactly ∓0.6 − π/2), while the chart it carries records them at 4.092588062 and 5.292588062 — the whole
notch rotated by **−0.019800918 rad**, 0.059 mm at radius 3. Region and boundary then disagree in a strip
0.06 mm wide and 4 mm tall along the boss's chord edges. Correcting the chart is the merge's `faceChart`,
which this task does not touch.

#### Correction 3 — "ring − half space" was NOT pre-existing, and it is fixed

The section above called that body's PropertyQuality tear pre-existing on the strength of ablating one of
its own changes. The mandated bisect says otherwise. At the wave base `c1e8f2a8`, in a clean worktree,
the body meshes **watertight at PropertyQuality, 203.865665 mm³** against an analytic 203.905.
`git bisect run` over `c1e8f2a8..2d1a996f` with a focused row names **6f8f5125** — the slice that deleted
`torusComplementMesh` and sent the genus-1 complement to the chart-driven mesher, measuring only
DefaultQuality.

The cause is the boundary clearance. That torus's section under the plane x = R is the LEMNISCATE: it
passes through the same 3D point twice, at (u,v) = (3π/2, π/2) and (3π/2, 3π/2), and its rim is sampled
coarsely right there — 0.17 rad of u in one chord against the covering's own 0.0245 stations. Interior
nodes landed INSIDE those chords and split them, four rim segments ended up carrying no triangle at all,
the face was declined by its own rim gate and fell to the surface's whole domain.

`chartBoundaryClearance` is 1.0 chords, not 0.5. A whole chord puts the chart-versus-chord band inside
the first triangle off the boundary, whose centroid is then two thirds of a chord away.

| | before | after | wave base |
| --- | --- | --- | --- |
| body free edges @Property | 272 | **0** | 0 |
| body volume @Property | 186.156188 | **203.869265** | 203.865665 |
| torus face @Property | 296.062 (whole domain), declined | 264.871, free == rim == 272 | 264.872 |
| torus face @Default | 263.730 | 263.423 | 263.596 |

`knownFreeEdgesAtFineQuality` is now EMPTY: every classification-corpus body is watertight at both
facetings.

#### Review round 3: the merged band's chart was 0.05 rad off its own edges (2026-09-08)

Task 7 measured what the round-2 section did not: the merged cocylindrical band's chart does not agree
with the face that carries it. On the D-prism body the merged face's own edges put the notch corners at
`u = 4.112388980` and `5.312388980` — `ParamAt` of the D-prism's chord vertices, exactly ∓0.6 − π/2 —
while the chart recorded `4.092588062` and `5.292588062`: the whole notch rotated by −0.019800918 rad,
0.059 mm at radius 3. Region and boundary then disagree in a strip 0.06 mm wide along the boss's chord
edges. At `DefaultQuality` the boundary clearance swallows it; at `PropertyQuality` it left ten
unpaired edges, pinned as `mergedBandFineFreeEdges`.

**Where the rotation entered.** `bandCircuit` closes two period-turning rims into one contour by
re-cutting each at the same crossing of the period, and `recutRing` did that by *snapping to the
nearest SAMPLE and translating the whole ring onto it*:

```go
cut  := nearestRingSample(ring, at, coord)
base := at - coord(ring[cut])            // an arbitrary sub-period shift
```

`base` is the gap between the wanted crossing and whichever sample happened to be closest — up to half
a sampling step, 0.098 rad at 32 samples — and every one of the ring's vertices was moved by it. The
first rim escapes (its cut is its own first sample, so `base` is 0); the second does not. `at` itself
was short too: `bandCircuit` passed `seam + netA`, and `netA` is the travel the SAMPLES report, which
`loopToUV` leaves one step short of a period because the closing step is the edge a sample list leaves
implicit. So the two seam traversals were a step apart as well.

**The rule this broke.** A face's chart is the face's OWN (u, v) boundary (ADR-0063). The only
re-basing it admits is by a WHOLE period, which maps back to the same 3-D point; anything else is a
different face. `recutRing` now cuts at the interpolated crossing on the ring's own chord — the one
point the construction may invent — and shifts by `sign·2π·floor(...)`, a whole number of turns and
nothing else. `bandCircuit` cuts the second rim where the first ENDS, `seam ± 2π`, not at the sampled
travel. `nearestRingSample` is deleted with the rule it served.

**Guarded as a class, not as an instance.** `TestEveryChartCarriesItsFacesOwnVertices` walks every
charted face of five cheap boolean bodies — the cocylindrical merge, both coaxial unions, a bore
continuing a bore, a drilled block — and requires every loop vertex the face carries to be a vertex of
its chart. It compares in 3-D, through `PointAt`, which is what makes the rule period-blind and
rotation-sensitive at once: a vertex a whole turn away is the same point, a rotated one is not. Run
against the previous code it fails with eight such vertices on the cocylindrical row.

**What it moved.** The merged band is watertight at BOTH facetings and `mergedBandFineFreeEdges` is
deleted. The piston head was already watertight and silent after Task 7 and stays so. One row lost its
subject: `query.TestATornClosedBodyReportsThroughTheHarvest` needed a body that tears, and nothing in
the corpus tears any more. It is replaced by the identity it was really guarding — the harvest's codes
are exactly the codes the face meshes carry — driven on the near-pinch crossing rods, the one corpus
body whose faces still record anything at `PropertyQuality`, and it refuses to pass on an empty set.

### Stage 6, the two silent exits and the engine's missing post-condition (2026-09-08)

Two places still let a wrong or unbuilt answer through without a word, and they are at opposite ends
of the same pipeline: the boolean's SMALLEST inputs, and the engine that stores whatever the boolean
returns.

#### The boolean's bottom end

`ops.Boolean` had no size classification at all. Measured on the RING corpus row (`brep.SolidTorus`
major 5, minor 1.5) cut by an axial drill at (5,0,-4) along +Z:

| drill radius | tool thickness | seam resolution | what shipped before |
| --- | --- | --- | --- |
| 0.8 (the RD− row) | 1.6 | 2.005e-5 | exact: 2 faces, 1 shell, volume 216.2569 |
| 1e-3 | 2e-3 | 2.005e-5 | refused — the built body's volume missed the Requicha bracket |
| 1e-5 … 1e-8 | 2e-5 … 2e-8 | 2.005e-5 | refused — but only AFTER the full intersect → imprint → classify → stitch, and reported as `boolean.no-exact-curved-path`, whose text says "the result will be faceted" (no longer true since stage 7) |
| 1e-10, 1e-12 | 2e-10, 2e-12 | 2.005e-5 | **the ring UNCHANGED**: `err=nil`, a valid closed 1-face solid, volume 222.066099 — every face certified against the operands, and inside the Cut bracket because `[V(A)−V(B), V(A)]` admits "removed nothing". Nothing recorded. |

`CurvedBoolean`, the public curved entry, did the same on its own account: it CERTIFIED the
unchanged ring as a cut result, because "removed nothing" passes the per-face membership rule and
the volume bracket alike.

Both rows are one configuration: a solid operand whose material is thinner than the tolerance at
which the boolean merges seam points, so the seam cannot separate the two sides of the tool. The
ground rule says an unsupported configuration is refused AT CLASSIFICATION with a named decline,
before any geometry is built. `declineSubResolutionOperand` is that classification — the thinnest
bounding-box extent of each SOLID operand against the pair's `geom.Resolution.Stitch()`, refused as
`ErrSubResolutionOperand` with a `boolean.sub-resolution-tool` Defect carrying the thickness, the
floor and the modelling remedy.

Three things it deliberately is not. It is not absolute: the same drill is refused in a big model and
built in a small one, because what fails is the ratio (a millimetre-scale ring with a proportional
drill classifies as modellable). It measures MATERIAL, so a sheet body — whose zero thickness is its
representation, not a part too thin to build — is not measured at all, and the split/replace-face
features that cut with one are untouched. And it is not a new tolerance: `Stitch()` is the weld the
boolean's own seam merge already uses, so `tolerance-constants` does not move.

#### The engine's end

`Validate` is a post-condition of every public kernel operation. The engine that STORES what those
operations return had none. Measured with a fake feature returning a body declared solid whose single
face leaves four boundary edges: `health = ok`, `diagnostics = []`, one body in `fs.Result()`, and the
viewport meshes it.

The post-condition now runs at ONE site — `evaluateBody`, between the recompute and the health
classification — so all 200-odd feature kinds and any added later inherit it without opting in. An
invalid body becomes an ordinary recompute error, which the existing mechanism already handles
correctly: the feature goes Sick with a reason naming the broken invariant, its dependents are
quarantined, and the body is DROPPED rather than carried forward.

One exemption, and it is the one the ground rules already name. A feature whose bodies come from
OUTSIDE the engine — a non-parametric base wrapping an imported STEP/STL body, a derived component
pulling another document's — can only be as valid as the file it came from, and refusing every
imperfect import is a product decision rather than a kernel one (the same reasoning that leaves
`meshbrep.MeshToBRep` on the validate-debt ledger). Such a feature declares
`AdoptedBodiesFeature`, and the post-condition REPORTS its invalid body as a Defect on feature
health instead of sickening it. That exemption is not hypothetical: two of the OCCT blend-parity
corpus's own STEP fixtures, `simple/H3` and `simple/H5`, import with three boundary edges each — a
fact the corpus recorded as a fillet decline ("is not a supported blend") while the real fault was
one layer upstream, in the input.

#### The measurement, stage 6

| | before | after |
| --- | --- | --- |
| `ops.Boolean(Cut, RING, drill r=1e-10)` | the ring unchanged, err=nil, 0 diagnostics | `ErrSubResolutionOperand`, 1 Defect |
| `ops.Boolean(Cut, RING, drill r=1e-6)` | refused after the full pipeline, mis-named | refused at classification, named |
| `CurvedBoolean(Cut, RING, drill r=1e-10)` | ok=true, the unchanged ring | ok=false, 1 Defect |
| feature engine post-condition sites | 0 | 1 |
| features exempt from it | — | 2, both `AdoptedBodiesFeature` (imported base, derived component) |
| a feature returning an invalid body | health ok, body stored | health sick, dependents quarantined, body dropped |
| `model/feature` suite wall time | 900.4 s | 908.3 s (+0.9 %, and the after run carried MORE background load) |
| `fallback-sites` | 26 | 28 (the code, plus its re-export through the `ops` facade — the counter counts both declarations) |
| `tolerance-constants` / `type-assertions` / `recognizers` | 214 / 691 / 11 | unchanged |

The suite cost is inside the run-to-run noise — the same code measured 900 s to 1014 s on this
machine depending on what else was running, an order of magnitude more spread than any difference the
post-condition could make — so no gating was needed. `ops.Validate` is already the
CHEAPEST of the ordered validity levels — topology and Euler over the body's edge list, reading no
geometry and no tessellation — and the post-condition measures only the bodies a feature BUILT: one a
feature passed through is the same pointer (the identity `producerOf` already relies on) and was
validated at the exit of the feature that built it, so re-checking it would cost O(features × bodies)
for an answer that cannot have changed.

What was deleted: nothing in the kernel — this stage adds the two refusals the pipeline was missing.
What it corrected is a FIXTURE: `model/feature`'s `makeBody` built a body declared solid with one
face and one loop edge, four boundary edges short of closed and a shape no operation could return.
Its invalidity was invisible while the engine stored whatever it was handed; with the post-condition
it sickens every test that uses it. It is `brep.SolidBlock` now — a fixture has to be something the
modeller could actually produce.

### Stage 6, review round 1 — the floor is measured, and the two policies become one (2026-09-08)

The stage-6 section above set the boolean's size floor by argument. This section replaces that
argument with a measurement, and the measurement moved the floor by three orders of magnitude —
downward, because most of what the first floor claimed was not a resolution limit at all.

#### The sweep

`TestTheAxialDrillSweepPinsTheResolutionFloor` and `TestNoDrillRadiusIsAnsweredSilently` drive the
RD− family — the RING corpus body (`brep.SolidTorus(P3(0,0,0), V3(0,0,1), 5, 1.5)`) cut by an axial
drill at (5,0,−4) along +Z — over twelve decades of tool radius, five points per decade. The pair's
extent is 20.0499, so `Weld` = 2.005e−8 and `Stitch` = 2.005e−5. Outcomes are classified against an
INDEPENDENT polar-quadrature oracle for the removed volume (`boreRemovalOracle`), which agrees with
the shipped exact row at r=0.8 to 3.5e−7 relative.

| tool thickness (2r) | thickness / Weld | thickness / extent | outcome |
| --- | --- | --- | --- |
| 2e−12 … 2e−9 | 1e−4 … 0.0998 | 1e−13 … 1e−10 | **SILENT**: the ring comes back untouched — err=nil, one face, removed=0, nothing recorded |
| 3.17e−9 … 1.262e−7 | 0.158 … 6.3 | 1.6e−10 … 6.3e−9 | refused by name (`boolean.no-exact-curved-path`) |
| 2e−7 … 1.262e−3 | 10 … 63 000 | 1e−8 … 6.3e−5 | refused by name (`boolean.no-exact-curved-path`) |
| 1.262e−3 … 1.262e−1 | 63 000 … 6.3e6 | 6.3e−5 … 6.3e−3 | refused by the Requicha bracket (`boolean.analytic-volume-reject`) — a VALID body of materially wrong volume, caught only there |
| 2e−1 … 1.6 | 1e7 … 8e7 | 1e−2 … 8e−2 | **exact**: torus + cylinder, 2 loops each, removed volume within 3.6e−6 of the oracle |

Three regimes, and only the first is a resolution limit.

#### Why the floor is Weld and not the plateau

The largest ratio below which the pipeline is not exact is ~6.3e−3 of the extent — a 1 mm bore in a
100 mm ring. Setting the floor there would refuse ordinary geometry on "resolution" grounds and, worse,
would relabel a real capability gap — the torus∧cylinder section at small radius, which builds a body
removing 2.84 where the true bore is 9.4e−6 — as a size policy, hiding it. The ground rule is the
opposite: find the invariant the pipeline breaks and fix it there; the failing input becomes a corpus
row. So the gap keeps its own loud refusal and gains a row that pins it,
`TestASmallBoreIsRefusedNotShippedWrong`, which asserts BOTH that no wrong body ships AND that the
refusal is not the size one. The day the section is fixed, that row says so.

What the size classification may claim is the SILENT band, whose top edge the sweep puts at
0.0998 × Weld. The floor is `Weld` itself: the smallest round, already-defined quantity that covers
the whole silent band, with ~6× margin, and nothing above it that used to work stops working — every
radius between the silent edge and Weld was already refused by name; only the name changes to the
truer one.

| | first cut | measured |
| --- | --- | --- |
| floor | `Stitch()` = 1e−6 × extent | `Weld()` = 1e−9 × extent |
| basis | asserted from the seam-merge argument | the sweep table above |
| radii claimed as sub-resolution | r ≤ 1e−5 | r ≤ 1e−9 |
| radii whose refusal keeps its own honest name | — | 1.585e−9 … 6.31e−2 |

#### One predicate, one sentence

`geom.FeatureResolvable` / `geom.SpanCeilingWarning` — surfaced to the user by
`PartComponentDefinition.FeatureScaleWarning` — answered the same question at `Weld`, while the first
cut of the decline refused at `Stitch`. Across the whole band between them the UI said "resolvable"
and the modeller refused. There is now ONE predicate, `geom.Resolution.Resolves`, and both callers
read it; the remedy sentence, previously typed out in both places, is `geom.ScaleRemedy`.

Making them share the predicate was not enough on its own: they were feeding it different extents.
`ResolutionForBodies` takes the LARGEST operand (right for a weld tolerance on a multi-body op),
while the UI measures the part's whole range box. On the RING pair those differ — 18.6 against
20.0499 — so a 2e−8-thick drill sat above one floor and below the other, and
`TestTheBooleanFloorAgreesWithTheFeatureScaleWarning` caught it. The decline now measures the union
of the operands' boxes (`pairExtentResolution`), which is the extent the user sees.

#### The exemption set, surveyed

The first cut exempted the base feature and the part derive, sampled rather than surveyed, so the
same imperfect STEP body was REPORTED through a part derive and SICKENED the feature through an
assembly derive. The complete set is the non-parametric base plus the three derive-family features
already grouped by `DeriveStatus`:

| feature | why it adopts |
| --- | --- |
| `NonParametricBaseFeature` | wraps bodies a translator produced (a STEP/STL import). |
| `DerivedPartComponent` | pulls a source PART's bodies, placed by a transform. |
| `DerivedAssemblyComponent` | pulls a source ASSEMBLY's placed bodies and merges the included ones. |
| `ShrinkwrapComponent` | simplifies a source assembly's bodies; a simplification cannot be more valid than what it simplifies. |

Each also falls back to `frozen` bodies captured at BreakLink, which are adopted twice over.
`AssemblyProxyCutFeature` is deliberately NOT in the set and is pinned as the counter-example: it
reads another occurrence's bodies as a TOOL and BUILDS a boolean, so its result is this engine's work
and carries the full post-condition.

#### An adopted defect now reaches health

Reporting the adopted body as a `diag.Defect` and returning nil left `pf.health` Healthy with an
empty Reason — a green tick over a torn body, which is the silence the stage exists to end. The
engine already owns a non-fatal channel, the one `ErrDeferred` and reference-heal drift use, so an
adopted invalid body is now `health.Warning` carrying the Validate reason, the body is kept, and
dependents are NOT quarantined.

#### The level the post-condition runs, and what it costs

The recorded justification was wrong: `ops.Validate` is not "topology and Euler only". It also runs
`checkHoleContainment`, which projects every multi-loop planar face's loops into the face plane —
and whose verdict, `HolesContained`, is not folded into `Valid`, so the post-condition was paying for
a result it discarded. `kernel/ops/validate` now exposes the ordered levels: `ValidateTopology` is
level 1, `Validate` is level 1 plus containment, and a new `HoleContainmentChecked` field stops a
level-1 report from reading as "no protruding hole found" when it never looked.

Measured on a drilled plate (a block with two bores, whose top and bottom faces each carry an outer
loop and two holes — exactly the shape containment works on), `go test ./kernel/ops/validate/
-bench Level -benchtime 300x`:

| level | ns/op | ratio |
| --- | --- | --- |
| `ValidateTopology` (level 1) | 888 | 1x |
| `Validate` (level 1 + containment) | 118 152 | **133x** |

The feature engine's post-condition runs level 1. That is a benchmark rather than a suite wall time
on purpose: it is not affected by what else the machine is doing, and it names the mechanism instead
of hiding it inside a 900-second number.

### Stage 6, review round 2 — the hang, the import family, and three corrections (2026-09-08)

#### The third outcome: a boolean that did not return

Round 1 recorded, as a concern, that a drill of radius ≈1.585e−7 through the RING body did not
terminate. That was the wrong disposition: the ground rules admit an operation that refuses and an
operation that answers, and nothing else. A hang is neither, and it is the worst of the three for a
UI, which cannot even report it.

The loop is `brep.splitTJunctions` (`arrange2d.go`), the planar arrangement's T-junction pass, which
subdivides "until stable". Its termination argument silently depends on scale: `tjTol` is an
ABSOLUTE 1e−7 and is read twice over — as a perpendicular DISTANCE from the edge, and as a
dimensionless bound on the parameter `t` along it. On geometry whose own features sit near 1e−7
those two readings stop agreeing, the pass keeps finding "interior" vertices on edges it has just
made, and it never converges. The stack at the hang:

```
BooleanDiag → booleanMixed → mixedPassFaces → closedSurfaceSplitFaces →
closedSurfaceSplitOne → trimByImprint → arrangeBand → Arrange → planarize → splitTJunctions
```

The bound is PROVABLE rather than tuned. Each split replaces one edge with two whose endpoints are
existing welded vertices, so it strictly grows a SET keyed by canonical index pairs; a set of
undirected pairs over n vertices holds at most n(n−1)/2 members, so at most that many splits can ever
add anything. `tjSplitBudget(n) = n(n−1)/2`, and a run that exceeds it is revisiting pairs it already
has. Exceeding it is NOT a silent break: `planarize` reports non-convergence, the new
`ArrangeChecked` returns `ok=false` (plain `Arrange` keeps its signature for the callers that cannot
act on the answer), `trimByImprint` returns `ErrUnconvergedArrangement`, and the mixed boolean's face
pass records `arrangement.unconverged` (a `diag.Defect`) on the recorder it already carries before
refusing. The cells are never used: an unstable cell complex would make every face traced from it a
guess.

| | before | after |
| --- | --- | --- |
| r = 1.585e−7 through the RING | did not return in any budget the suite could give it | refused in 0.07 s: `ErrUnmodelledBoolean`, with `arrangement.unconverged` recorded |
| `splitTJunctions` termination | "until stable", unbounded | bounded by `tjSplitBudget`, reported when hit |
| sweep rows | could wedge the suite | each runs under a deadline derived from `t.Deadline()` |

`fallback-sites` rises by one for `CodeArrangementUnconverged` — a RISE that names a degradation
nothing reported before, and in this case one that could not be reported at all, because the
operation never got far enough to report anything.

#### The import family

Round 1's exemption survey anchored on the `DeriveStatus` group and so missed the IMPORT family's
second member. `ImportedBodyFeature` (`imported_body.go`) wraps one body of a foreign MESH file
(STL/OBJ/3MF) or a STEP body and injects it verbatim — "the mesh-exchange counterpart of
`NonParametricBaseFeature`" — and declared nothing, so a torn STL sickened its feature and
quarantined everything downstream. An STL is very often not a valid closed solid; this is the single
most likely way a user meets the post-condition.

The list is now surveyed BY SHAPE — every `Recompute` that emits a `*topo.Body` it did not construct,
i.e. appends a stored field rather than a value it built this call — because two anchors in a row
each missed a member. The complete set is five: `NonParametricBaseFeature`, `ImportedBodyFeature`,
`DerivedPartComponent`, `DerivedAssemblyComponent`, `ShrinkwrapComponent`. Two neighbours are pinned
as counter-examples: `AssemblyProxyCutFeature` booleans another occurrence's bodies as a tool, and
`MeshSolidFeature` CONSTRUCTS a faceted solid through `ops.MeshToBRep` — both outputs are this
engine's own work and carry the full post-condition.

#### Three corrections to the round-1 record

The stage-6 round-1 section above stands as written (this ADR is append-only); these correct it.

1. **The floor's reach.** The round-1 table says the classification claims "r ≤ 1e−9". It does not:
   the floor is a THICKNESS of `Weld` = 2.00499e−8 at the RING pair, so it claims **radius <
   1.0025e−8** — an order of magnitude more than recorded. The measured silent band still ends far
   below that (≈0.0998 × Weld), which is the margin the floor was chosen for; only the description
   was wrong.
2. **The fourth row's mechanism.** The round-1 sweep table calls the 6.3e−5 … 6.3e−3 band "refused",
   next to rows that are refused BY NAME at classification. It is not the same thing: in that band
   the pipeline builds a complete, valid, correctly-wound solid and only the post-hoc Requicha volume
   bracket rejects it. The distinction matters because a volume bracket is a smoke test, not a proof
   — a wrong body of coincidentally right volume would pass it.
3. **The benchmark invocation.** The recorded command said `-benchtime 200x`; the numbers were taken
   at `300x`.

### Stage 6, review round 3 — the bound's own silent exits (2026-09-08)

Bounding the T-junction pass in round 2 introduced two new silent exits of exactly the kind this
stage exists to close, and the round-2 text above is wrong where it says the cap is "NOT a silent
break". It was not, at the one call site round 2 looked at. It was at the other two.

`ArrangeChecked` returns `(nil, false)` when the pass hits its budget, and round 2 kept an unchecked
`Arrange` that discarded the flag "for the callers that cannot act on the answer". Both production
callers COULD act on it:

| caller | what it did with `nil` cells |
| --- | --- |
| `brep.splitFace` (the PLANAR boolean face split) | read it as "this face has no material sub-faces" and dropped the face — no error, no `diag.Defect`. `ops.Validate` then reported an open body, with the cause erased. |
| `tessellate.unionTris` (overlapping-hole faces) | dropped every cell and meshed nothing, silently. |

**The unchecked entry is deleted** (delete-first: there is no unchecked sibling, and the doc comment
now records why, so it cannot be reintroduced as a convenience). `splitFace` returns
`([]subFace, bool)`; the planar chain — `selectFragments` → `selectFaces` → `booleanOnce` — carries
the flag to `BooleanDiag`'s recorder and the boolean refuses by name. `rebuildImprinted` (the public
imprint entry) returns the named error. The tessellator records `tessellate.arrangement-dropped-cells`
on the face mesh, the same result-carried shape as `CodeTrimIgnoredFullDomain`: the mesh still ships,
because a partial covering beats a missing face in a viewport, but it no longer ships silently.

Not folding the flag into "no sub-faces" is the load-bearing part. A non-converged split and a face
whose every region is outside the material both produce an empty slice, and the first is a defect
while the second is the ordinary answer; reading them as the same thing is what erased the cause.

#### The decline now names its site, and a guard keeps every site reporting

Round 2 threaded the recorder into ONE of the three `trimByImprint` call sites. The other two
(`wallSplitFaces`, `uvSplitFaces`) returned a bare `ok=false`, so an unconverged arrangement there
surfaced only as the generic `ErrUnmodelledBoolean`, naming nothing. All four arranging splits — the
closed-surface trim, the ruled-wall trim, the uv-plane trim and the planar face split — now record,
and the diagnostic NAMES which one declined: they fail for the same reason, so a report that did not
distinguish them would send the reader to the wrong one.

The coverage is a source guard rather than four corpus rows, and deliberately so. Only ONE of the four
is reachable geometrically with the fixtures the sweep can build: the RING drilled at r ≈ 1.585e−7
takes the closed-surface trim, and a rod, a block, an off-centreline drill and a thin planar slab all
refuse earlier for other reasons before any of the other three arranges anything (swept, 1e−9 … 1e−4).
`TestEveryArrangingSplitReportsANonConvergentArrangement` therefore scans the package for every
`trimByImprint`/`splitFace` call and requires a decline beside it, which covers the three unreachable
sites AND the split nobody has written yet. It was proven live by deleting one decline: the guard
names the file, the line and the call.

#### A correction to round 2's budget argument

The round-2 comment claimed the budget is bounded because "each split strictly grows a SET keyed by
canonical index pairs", while the code decremented on EVERY split. Those are not the same statement:
a split whose two halves are both already present adds nothing and only removes an edge. The code now
decrements only on a split that adds at least one pair, which is the quantity the n(n−1)/2 argument
actually bounds; a split that adds neither half strictly shrinks the set, so it cannot run away on its
own account and is not counted.

| | round 2 | round 3 |
| --- | --- | --- |
| unchecked `Arrange` | present, flag discarded at 2 production sites | DELETED |
| arranging splits that report | 1 of 4 | 4 of 4, each naming its site |
| coverage | 1 corpus row | 1 corpus row + a source guard over every site, proven by deletion |
| budget counts | every split (argument bounded only pair-adding ones) | pair-adding splits only |
| `fallback-sites` | 31 | 32 (`CodeArrangementDroppedCells`) |

#### Review round 4 (2026-09-08): the guard reads the AST, the flag has a test, the argument is stated exactly

Three things round 3 left short, none of them a behaviour change.

The source guard was a regex over call LINES plus an 8-line substring scan for the decline. It
matched `faces, _, err := trimByImprint(` and nothing else — not `return trimByImprint(...)`, not a
selector target, not a nested call — and a COMMENT naming `recordArrangementDecline` satisfied it.
`TestEveryArrangingSplitReportsANonConvergentArrangement` now parses each production file with
`go/parser`: a call is any `*ast.CallExpr` whose callee is the identifier `trimByImprint` or
`splitFace`, and the decline must be a `recordArrangementDecline` call expression (or, for a function
that carries no recorder, an `unconvergedArrangement` call in a `return`) in the SAME function body;
the parser drops comments before either is looked for. The scanner is proven on in-memory sources in
every call form, guarded, unguarded and comment-only (`arrange_decline_guard_test.go`), and again live
by deleting one production decline.

`CodeArrangementDroppedCells` had no test. `TestRecordDroppedCellsFlagsOnlyANonConvergedArrangement`
pins both senses geometry-free, and the two `unionTris` rows assert convergence on the ordinary path
instead of discarding the flag.

The budget argument, as round 3 stated it ("the quantity the n(n−1)/2 argument actually bounds"),
overstates: pairs can be deleted and re-added, so it is not "each pair is added at most once". What
holds is weaker and sufficient — the budget is an ENFORCED cap on pair-adding splits, set to the size
of the pair universe so that a run needing more has re-added a pair it already removed; a
non-adding split strictly shrinks a finite set; hence the pass terminates. `tjSplitBudget`'s comment
now says exactly that. `booleanOnce`'s refusal reported a FACE count in a message that says
"segments"; it now counts the segments.

### The chart mesher at fine quality, round 2: a set, a refusal and a swept clearance (2026-09-08)

Five findings against the round before. One of them (the merged band's residual ten free edges) was
closed upstream by `7a89841e`, which fixed the chart the merge records; the other four are here.

#### The rim gate counted where it had to compare

`chartMeshIsBoundedByItsRim` held the mesh's unpaired-edge COUNT against the boundary's segment count.
A count cancels: on the merged cocylindrical wall at PropertyQuality it read 578 against a rim of 578
while FIVE of those free edges were no rim segment at all and five rim segments carried the wrong number
of triangles. The face passed a gate it should have failed and the body shipped ten unpaired edges.
Neighbouring a wrong edge with a missing one is exactly the shape a chart that disagrees with its own
boundary produces, so the cancelling pair is the case worth catching, not a coincidence. It compares the
SETS now, keyed the way the mesh's own welded edges are. It costs no corpus row — `7a89841e` corrected
that chart — which is the point: the gate is what stops the next such chart passing.

#### `chartChain.longest` was dead, and wiring it is measurably worse

The field and `longestChainChord` were added and documented as "the only bound a cheap box rejection may
use", and `chainIsNear` never read it — so the previous round's claim that the clearance is read per
SEGMENT was false of the shipped code. Implementing it properly was measured and rejected: a boundary is
not sampled uniformly (the merged wall's notched rim carries 320 chords of 0.074 mm and TWO of 4 mm),
but a clearance scaled to those two clears a 4 mm disc of interior nodes off a face 4 mm tall and
starves the region — the merged band went from 10 unpaired edges at PropertyQuality to **469** and the
figure-eight band overshot its analytic area. The field is deleted and `chainIsNear` records why the
mean is the right scale.

#### The loft refuses a pinched band again, for the uncharted case

Routing a CHARTED spiric band to the chart mesher replaced `bandPinches` for every band the corpus
builds, and the round before deleted it on that basis. One step too far: the routing gates on a chart, a
property of the PRODUCER, where `bandPinches` gated on the loft's own failure mode, and "no primitive
boolean in the corpus builds an uncharted torus band" is an observation about today's corpus, not an
invariant. An uncharted pinching band would loft silently and wrong (310.800 mm² where the analytic
region is 283.100, the two halves summing to 525.98 against a torus of 394.78).

`bandPinches` is back, scoped in its doc to the uncharted case, and the refusal is NAMED downstream: the
face falls to the general path, which for a band that records no region ends at `recordIgnoredTrim`'s
`CodeTrimIgnoredFullDomain`. `TestAnUnchartedPinchedBandIsRefusedAndSaidSo` drives the figure-eight
piece's own torus face with `SetChart(nil)` through the router, for both operations, and asserts the
decline AND that the mesh is the reported fallback rather than the loft's answer. With the gate disabled
it fires twice: "meshed with no named decline: []" and 138.925 mm² where the fallback is 394.784 less a
chord deficit.

#### `chartBoundaryClearance` is swept, not re-tuned

It went 0.5 → 1.0 on one failing case, with a derivation that does not predict it. The band is bounded
by the discretisation's sagitta, chord²/8ρ, and k chords of clearance put the first triangle's centroid
(2/3)·k·chord out, so the sagitta argument alone is satisfied by any k > 3·chord/(16ρ) — about 0.03 for
these faces. It does not settle the constant because the band is not always a sagitta: where a boundary
TOUCHES itself the rim is sampled coarsely right at the touch (the lemniscate complement carries 0.17 rad
of u in one chord against the covering's own 0.0245 stations) and an interior node lands INSIDE the
chord rather than beside it. What bounds that is the chord, and its size is measured.

Swept on the three bodies whose charted faces the clearance governs — the genus-1 lemniscate complement,
RS− and RD− — at both facetings, counting failures over `./kernel/ops/tessellate/ ./kernel/ops/boolean/`:

| k | 0.125 | 0.25 | 0.5 | 0.75 | 0.875 | 1.0 | 1.1 | 1.25 | 1.5 | 2.0 | 3.0 | 4.0 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| failures | 5 | 4 | 2 | **0** | **0** | **0** | **0** | 1 | 3 | 4 | 7 | 16 |

A plateau of 0.75 … 1.1, pinned at its middle, **0.875**, with a `// tol:` naming the sweep. The
collateral the previous round only admitted in prose is asserted: the complement's torus FACE carries a
two-sided area pin, 263.610 ± 0.5 mm², and its row's doc records the re-measured 1.30 % (201.258 against
an analytic 203.905) rather than the 1.11 % it read before the constant moved.

#### The bisect the previous round asserted but did not show

`git bisect run` over `c1e8f2a8..2d1a996f`, with a focused row asserting `ring − half space` watertight
at PropertyQuality, in a clean worktree:

```text
Bisecting: 16 revisions left to test after this (roughly 4 steps)
…
95cdd21ec18cccf06840ef62c48602902aa89c0e is the first bad commit
    kernel/tessellate: a charted trim is meshed, not discarded, (ADR-0061)
bisect found first bad commit
```

The previous round named `6f8f5125` from a coarser run. The first bad commit is **`95cdd21e`** — the
same slice, one commit earlier: the wiring that replaced `torusComplementMesh`'s exits with the
chart-driven mesher, measured only at DefaultQuality.

### The chart mesher at fine quality, round 3: one rim key, and the clearance measured per body (2026-09-08)

Two findings against round 2, and the corrections they force to that round's own numbers.

#### One rim key, for the gate and for the tests

Round 2 made the acceptance gate compare SETS and left `chainSegmentCount` behind as production code with
only test callers. Worse, the two had DIVERGED: the gate keys a rim once per face on the MESH's weld
grid, `chainSegmentCount` keyed it per chain on per-chain grids, and `geom.ResolutionForPoints` scales
with the points it is given — so on a multi-loop face the corpus row and the shipped gate were answering
different questions, and `chart_face_cover_test.go` had stopped asserting the gate.

`chainSegmentCount` is deleted. `chartRimMismatch` is the one comparison, returning the two numbers the
gate decides on (unpaired edges that are no rim segment; rim segments the mesh does not bound), and the
tests read the gate through it. `TestTheRimIsKeyedOnceForTheWholeFace` drives the three-loop windowed
wall and requires the per-chain reading to agree with the gate's, so nothing can go back to counting its
own way without failing.

#### A refusal that names the shape it refused

`CodeTrimIgnoredFullDomain` said "no mesher recognised its boundary on this surface" even when a mesher
HAD recognised it and given it up — a pinched spiric band, which round 2 made the loft refuse. The two
are different repairs, so `specialCurvedMesh` now returns WHY it declined and the reporter says it: "the
mesher that recognised it refused the shape — its two tube-wrapping boundaries MEET, so the band is two
lobes joined at a point and no single sweep round the tube describes it". Asserted end-to-end by
`TestAnUnchartedPinchedBandIsRefusedAndSaidSo`.

#### The clearance, measured per body and per faceting

Round 2's sweep was one aggregate failure count, which cannot show a plateau's flatness and hid that its
own quoted areas came from different code states. Re-measured on this HEAD, per body, per faceting, as
free edges and the torus FACE's own area:

| k | complement D | complement P | RS− D | RS− P | RD− D | RD− P |
| --- | --- | --- | --- | --- | --- | --- |
| 0.50 | 0 / 263.72994 | **272 / 296.06212** | 0 / 236.36 | 0 / 237.87 | 0 / 290.29 | 0 / 291.88 |
| 0.55 | 0 / 263.72994 | **272 / 296.06212** | 0 / 236.36 | 0 / 237.87 | 0 / 290.29 | 0 / 291.88 |
| 0.60 | 0 / 263.68219 | 0 / 264.87124 | 0 / 236.36 | 0 / 237.87 | 0 / 290.29 | 0 / 291.88 |
| 0.70 | 0 / 263.60871 | 0 / 264.87119 | 0 / 236.10 | 0 / 237.87 | 0 / 290.29 | 0 / 291.88 |
| **0.875** | 0 / **263.55487** | 0 / 264.87111 | 0 / 236.10 | 0 / 237.87 | 0 / 290.29 | 0 / 291.88 |
| 1.00 | 0 / 263.42317 | 0 / 264.87104 | 0 / 236.10 | 0 / 237.87 | 0 / 290.29 | 0 / 291.88 |
| 1.10 | 0 / 263.33288 | 0 / 264.87097 | 0 / 235.44 | 0 / 237.87 | 0 / 290.28 | 0 / 291.88 |
| 1.50 | 0 / 263.15360 | 0 / 264.87045 | — | — | — | — |
| 3.00 | 0 / 260.51898 | 0 / 264.86644 | — | — | — | — |

Three corrections to round 2 fall out of it:

- **The failure edge is 0.55, not 0.5–0.75.** Only ONE body and ONE faceting ever fails — the complement
  at PropertyQuality, where the face is declined and falls to the whole domain (296.062 against the
  264.871 it builds). Above 0.6 nothing fails at any k measured, so the upper end is not a plateau edge
  at all: it is a monotone COST, the clearance removing interior nodes next to a coarse boundary.
- **The areas ARE monotone in k.** Round 2 reported them non-monotone (263.596 @0.5, 263.610 @0.875,
  263.423 @1.0); those three readings came from three different code states, not from one sweep. Measured
  in one pass on one HEAD they fall with every step, which is what a rule that only ever REMOVES interior
  nodes must do.
- **0.875 is chosen, not centred**, and round 2 should not have called it a middle. It is 1.6× the
  largest k that fails and 1.46× the smallest that passes, and it costs 0.13 mm² of 263.7 — 0.05 % —
  against sitting at 0.6.

#### The complement's pin, which round 2 did not actually pin

`263.610 ± 0.5` admitted both readings the clearance moved between (263.42317 at k = 1.0 and 263.72994 at
k = 0.5), so a row meant to catch the constant drifting would have sat green through exactly that, and
its centre was a value from an intermediate state rather than the measurement at HEAD. It is
**263.55487 ± 0.05**: tessellation is byte-identical run to run by ground rule, so the reading is exact
and the window absorbs only the five decimals the literal is written to plus the last-place spread an
FMA-contracting toolchain gives an area sum — four decades above that, and a factor of 2.6 below the
nearest reading it must exclude. Proven to fire: at k = 1.0 it reports "the complement's torus face
meshes 263.42317 mm², off its pin of 263.55487 ± 0.05".

### The retirement, measured at close (2026-09-08)

ADR-0061 set out to retire the CSG fallback: the triangle-soup BSP and the exact mesh arrangement that
stood behind the analytic boolean, and everything that existed only to keep them fed. The doors are
shut (`faceted-entry-sites` 0), the room is gone (`faceted-engine-files` 0), and the ladder that chose
between them is a classification. This section is the measurement at close: every corpus row rebuilt
through `ops.Boolean` on this HEAD, the archguard pins' trajectory from the wave base `c1e8f2a8`, and
the list of what the kernel still refuses or still serves from a bespoke arm.

Nothing below is a new claim. It is what the shipped code does today, measured in one pass by a
throwaway harness that was deleted before this commit — every number is reproducible from the corpus
tests named beside it.

#### The corpus at close

Built with `ops.BooleanWithDiagnostics`, validated with `ops.Validate`, meshed with
`tessellate.TessellateBody` at `ops.DefaultQuality()` and `ops.PropertyQuality()`, integrated with
`query.AnalyticGeometryProperties` (the analytic B-rep, not the mesh) and
`tessellate.MeshGeometryProperties`. "Oracle" is the value the row was certified against by the task
that landed it; where a row has no independent closed form the analytic integrator's own value is shown
and the certification is the Requicha identity plus the jittered membership integral in
`boolean_torus_skew_test.go`.

| row | outcome | Validate v/c/m/solid | analytic V | oracle | rel | free @D | free @P | mesh V @D | rel | diag codes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| RING ring | built | ✓/✓/✓/✓ | 222.06610 | 222.06610 | 0.0000 % | 0 | 0 | 219.22695 | 1.28 % | — |
| RB∩ ring ∩ ball | built | ✓/✓/✓/✓ | 23.86935 | 23.86935 | 0.0000 % | 0 | 0 | 23.53639 | 1.39 % | — |
| RB− ring − ball | built | ✓/✓/✓/✓ | 198.19675 | 198.19675 | 0.0000 % | 0 | 0 | 195.76723 | 1.23 % | — |
| RS− ring − coaxial shaft | built | ✓/✓/✓/✓ | 203.59254 | 203.59 | 0.0012 % | 0 | 0 | 199.83714 | 1.84 % | — |
| RD− ring − axial drill | built | ✓/✓/✓/✓ | 216.25687 | 216.26 | 0.0014 % | 0 | 0 | 213.47272 | 1.29 % | — |
| RODB∪ rod ∪ ball | built | ✓/✓/✓/✓ | 13.07777 | 13.077910 | 0.0010 % | 0 | 0 | 12.99201 | 0.66 % | — |
| RODB− rod − ball | built | ✓/✓/✓/✓ | 12.55418 | 12.555898 | 0.0137 % | 0 | 0 | 12.47920 | 0.61 % | — |
| RODB∩ rod ∩ ball | built | ✓/✓/✓/✓ | 0.01219 | 0.012187 | 0.0647 % | 0 | 0 | 0.00876 | 28.10 % | — |
| SKEW rod across ring ∪ | built | ✓/✓/✓/✓ | 241.42303 | 241.42303 | Requicha 1e-9 | 0 | 0 | 238.36236 | 1.27 % | — |
| SKEW rod across ring − | built | ✓/✓/✓/✓ | 213.14870 | 213.14870 | Requicha 1e-9 | 0 | 0 | 210.41873 | 1.28 % | — |
| SKEW rod across ring ∩ | built | ✓/✓/✓/✓ | 8.91740 | 8.91740 | membership 2e-3 | 0 | 0 | 8.80899 | 1.22 % | — |
| TILT ring − tilted drill | built | ✓/✓/✓/✓ | 216.25531 | 216.25531 | Requicha 1e-9 | 0 | 0 | 213.47580 | 1.29 % | — |
| TILT ring ∩ tilted drill | built | ✓/✓/✓/✓ | 5.81079 | 5.81079 | membership 2e-3 | 0 | 0 | 5.75447 | 0.97 % | — |
| FAT ring − fat rod r=2 | **refused by name** | — | — | — | — | — | — | — | — | `section.conditioning-demotion`, `boolean.no-exact-curved-path` |
| DPRISM cyl ∪ D-prism | built | ✓/✓/✓/✓ | 277.92004 | 277.92004 | 0.0000 % | 0 | 0 | 275.88981 | 0.73 % | — |
| PISTON #2167 feature body | built | ✓/✓/✓/✓ | 277.92004 | 277.92004 | 0.0000 % | 0 | 0 | 275.53673 | 0.86 % | — |
| HALF ring − half space | built | ✓/✓/✓/✓ | 203.90487 | 203.90487 | — | 0 | 0 | 201.25781 | 1.30 % | — |
| FIG8− torus − y>3 | built | ✓/✓/✓/✓ | declined | 279.89786 (= torus − ∩) | — | **2** | 0 | 276.07755 | 1.36 % | `tessellate.mesh-not-watertight` |
| FIG8∩ torus ∩ y>3 | built | ✓/✓/✓/✓ | 114.88632 | 114.88632 | — | 0 | 0 | 112.52207 | 2.06 % | — |
| NPX near-pinch crossing rods ∪ | built | ✓/✓/✓/✓ | 534.59058 | 534.59058 | — | 0 | 0 | 527.08218 | 1.40 % | `tessellate.cap-saturated` |
| SUBRES ring − drill r=1e-10 | **refused by name** | — | — | — | — | — | — | — | — | `boolean.sub-resolution-tool` |
| NONTERM ring − drill r=1.585e-7 | **refused by name**, terminates | — | — | — | — | — | — | — | — | `arrangement.unconverged`, `boolean.no-exact-curved-path` |

Reading the table:

- **Every built row is a valid, closed, manifold solid**, and every one whose analytic integrator claims
  it lands on its oracle to at worst 6.5e-4 relative. The B-rep is exact; the mesh error is a chord
  deficit, 0.6–2.1 % at `DefaultQuality` on every row but one.
- **SKEW is no longer a refusal.** The constraints table this wave started from recorded "rod ACROSS ring"
  as refused by name; the torus's second harmonic (stage 5, third slice) builds all three operations, and
  the named-refusal fixtures moved to the pair that genuinely has no closed form — two interlocked TORI.
- **RODB∩ is the one row whose mesh is not a chord deficit** (28.10 %, pinned two-sided at 0.2810 ± 0.005
  in `chart_face_mesh_test.go`). Its two faces are lens patches 0.1 mm deep on a body of 0.012 mm³; the
  rod's wall is charted, the ball's face classifies as `kindSpherePatch` and chords flat across the lens.
  It is a faceting bound on the sphere-patch arm, not a wrong body: the analytic volume is 0.065 % off.
- **FIG8− is the one row that tears, and it says so.** At `DefaultQuality` the cut piece meshes with 2
  free edges between a planar lid and the torus face, and `CodeMeshNotWatertight` names both faces. At
  `PropertyQuality` it is watertight (279.846 against 279.898, 0.019 %). This is NEW in this wave: at
  `c1e8f2a8` the same body meshed 0 / 0 free edges — but it meshed the WRONG surface, its torus face
  covering the pinch twice at 317.639 mm² where the analytic share is 283.100, so that the two figure-
  eight pieces summed to 527.88 mm² of a torus whose whole area is 394.78. At HEAD they sum to 394.75.
  A visible, named 2-edge crack at the coarse faceting replaced an invisible 133 mm² of doubled surface
  across the pair (+34.5 on the cut piece, +98.6 on the intersect piece).
  It is a defect and it is listed below, not written off.
- **The analytic integrator declines the FIG8 cut piece** — pre-existing, measured identically at
  `c1e8f2a8` — so that row's oracle is the complement identity rather than a direct integral.
- **The three refusals are refusals**, not hangs and not wrong bodies. The sub-resolution row names the
  offending thickness and the floor ("cut tool is 2e-10 thick, below this model's resolution 2.005e-8");
  the non-terminating row returns instead of hanging, naming `arrangement.unconverged` and which of the
  four arranging splits declined; the fat rod names the conditioning demotion that lost the closed form.

#### The archguard pins from `c1e8f2a8` to HEAD

`TestKernelNetDelta` fails on ANY move, up or down, so every line below is a re-pin with a dated
reason in the pin file. These are the moves of THIS wave only; the earlier stages' moves are recorded
in the sections above.

| ratchet | `c1e8f2a8` | HEAD | net |
| --- | --- | --- | --- |
| tolerance-constants | 214 | **214** | 0 |
| type-assertions | 692 | **684** | −8 |
| recognizers | 11 | **12** | +1 |
| fallback-sites | 25 | **32** | +7 |
| `dispatchLadders` (registry size) | 1 | **0** | −1 |
| `faceted-entry-sites` | 0 | **0** | 0 |
| `faceted-engine-files` | 0 | **0** | 0 |
| `mixed-decline-returns` | 3 | **3** | 0 |

The moves, one line each:

- **tolerance-constants 214 → 214.** No move, and the `toleranceDebt` map is byte-identical to the
  base's — not one file's count changed. The constants this wave DID add all carry a `// tol:`
  annotation, which is what clears a line out of `toleranceDebt`, so none of them is debt — but an
  annotation is not a proof. Only two are swept: `chartBoundaryClearance` (twelve values, per body, per
  faceting) and `nearPinchCorridorChords` (a measured plateau with a two-directional corpus proof).
  `chartContourIncidence` has an annotation and a resolution argument but no sweep, and
  `chartNodeClearance` (0.3) has neither. Both are open items below (G16, G17).
- **type-assertions 692 → 691** (stage 5, chart-driven mesher): a FALL — the curved-face router's
  `s.(geom.Torus)` went with `torusComplementMesh`. An outerless face on any periodic surface is now
  meshed from the chart it carries, so the router asks what the FACE records, not what its surface is.
- **type-assertions 691 → 684** (stage 5, the classification): a FALL of 7 — the seven geometry-kind
  assertions the curved-trim classification collapsed.
- **recognizers 11 → 12** (stage 5, the classification): a RISE that is a CORRECTION OF THE MEASUREMENT,
  not new code. The old 11 counted ladder ENTRIES, and an entry was never one recognizer: entry 0
  recognized TWO cone shapes, and three entries read three RIM FORMS into one `buildSphereCap`. Counting
  the shape recognizers behind the arms (`curvedTrimRecognizers`) gives 12 before this slice and 12
  after. `specialCurvedMeshers` is gone and its eleven entries are eight classification arms, but no
  bespoke SHAPE was deleted — what was deleted is builder duplication.
- **fallback-sites 25 → 26** (stage 5, third slice): `CodeSectionConditioningDemotion`. The analytic
  intersector refused two different things with one anonymous `ok=false` — "no bucket claims this pair",
  which is the ordinary case, and a CONDITIONING demotion, which is a fallback. Only the second is a
  degradation and it was indistinguishable from the first.
- **fallback-sites 26 → 27** (stage 5, cocylindrical wall merge): `CodeCocylindricalMergeUndecided`.
  Where the fused loops do not determine the merged face's trim, ADR-0063 refuses to guess a side — and
  the pair was then left as two faces with nothing said.
- **fallback-sites 27 → 28** (same slice, review round 1): `CodeMeshNotWatertight`. Every per-face mesher
  certified its own patch and nothing certified the BODY, so a crack between two correctly-meshed faces
  was invisible until somebody counted free edges.
- **fallback-sites 28 → 30** (stage 6): `CodeBooleanSubResolutionTool`, +2 not +1 — the counter counts
  every `diag.Code` ValueSpec under `kernel/`, and the ops facade re-exports the name. One new
  degradation, two declarations of it: an operand thinner than the model's seam weld used to come back
  as the target UNCHANGED with `err=nil` and nothing recorded.
- **fallback-sites 30 → 31** (stage 6, review round 2): `CodeArrangementUnconverged`. A degradation that
  COULD not be reported before: the planar T-junction pass subdivided "until stable" and, at the scale of
  its absolute 1e-7 tolerance, never became stable — the operation hung, which is neither a refusal nor a
  wrong body.
- **fallback-sites 31 → 32** (stage 6, review round 3): `CodeArrangementDroppedCells`. Bounding the
  T-junction pass made `brep.Arrange` able to return NO cells, and both production callers read that as
  an ordinary empty answer. The unchecked entry is deleted and both routes now say so.
- **`dispatchLadders` 1 → 0.** The tessellator's 11-entry `specialCurvedMeshers` (#3409) is replaced by
  `classifyCurvedTrim`, a classification whose predicates are proved mutually exclusive. Its registry
  entry is REMOVED rather than kept at zero — the registry may only shrink — and there is no first-fit
  ladder left in the kernel to register.
- **A new guard was added, not a pin moved**: `TestNoUnprovenPayloadGatedChains` (`archguard/`), which
  registers every consecutive payload-gated recogniser chain in `kernel/ops/tessellate` with what makes
  it not a ladder. It exists because the first attempt at retiring the tessellator's ladder turned the
  sphere family into exactly that shape and nothing saw it. Three entries today: one proved disjoint,
  two registered as DEBT (#3410, #3411).
- **The three `fallbackDebt` rows did not move**, and two of them are floors, not targets:
  `faceted-entry-sites` and `faceted-engine-files` were driven to 0 by stages 6 and 7 before this wave
  began and the guard now keeps an engine from coming back unnoticed. `mixed-decline-returns` stays at 3
  and is not going to zero — it counts the three sites at which the general pipeline REFUSES by name,
  which is where the kernel states its boundary.

#### What is still refused, and what is still bespoke

Each line names the guard that keeps it visible. Nothing here is silent.

Refused by name, correctly — the kernel's stated boundary:

- **G1.** **Torus × torus** — neither side supplies an implicit quadric, so no closed-form section exists.
  Guard: `TestATorusPairIsRefusedByName`, `TestCurvedImprintTorusPairDefers`.
- **G2.** **A conditioning demotion on the skew reduction** (the fat rod, r > tube radius: four independent
  full-period branches the second-harmonic reduction does not carry). Guard:
  `TestALaneConditioningDemotionIsReported` + `CodeSectionConditioningDemotion`.
- **G3.** **A sub-resolution operand** (material thinner than the model's seam weld). Guard:
  `TestASubResolutionDrillIsRefusedByName`, `CodeBooleanSubResolutionTool`.
- **G4.** **A non-convergent planar T-junction subdivision** (r ≈ 1.585e-7 through the RING). Guard:
  `TestTheNonConvergentDrillTerminatesAndIsNamed`, `CodeArrangementUnconverged`,
  `TestEveryArrangingSplitReportsANonConvergentArrangement`.
- **G5.** **An arrangement that returns no cells.** Guard: `CodeArrangementDroppedCells`.
- **G6.** **A cocylindrical merge whose fused loops do not determine the merged trim** — ADR-0063 refuses to
  guess a side. Guard: `CodeCocylindricalMergeUndecided`.
- **G7.** **An uncharted pinching band offered to the loft.** Guard: `bandPinches` +
  `TestAnUnchartedPinchedBandIsRefusedAndSaidSo`.

Capability gaps — refused rather than shipped wrong, but a real hole:

- **G8.** **The torus bore from ~1.6e-10 to ~6.3e-3 of the pair's extent cannot be built.** Measured on the RING
  pair: 1.6e-10…6.3e-5 of extent is refused by name (`no-exact-curved-path`); 6.3e-5…6.3e-3 builds a
  VALID body of materially wrong volume and is caught only by the post-hoc Requicha bracket
  (`analytic-volume-reject`) — a smoke test, not a proof. A 1 mm bore in a 100 mm ring is in that band.
  It was deliberately NOT relabelled a resolution policy, which would have hidden it. Guard:
  `TestASmallBoreIsRefusedNotShippedWrong` (4 rows), `boolean_drill_sweep_test.go`.
- **G9.** **`tjTol` is an absolute 1e-7 read as BOTH a length and a parameter.** It is why the row above exists
  and why the T-junction pass had to be bounded rather than fixed; a model-relative `geom.Resolution`
  there would likely shrink G8. Guard: the bound (`brep.tjSplitBudget`) and its named decline. Note what
  the tolerance ratchet does NOT do here: `tjTol` carries `// tol:calibrated`, which is exactly what
  clears a line out of `toleranceDebt`, so a length tolerance annotated as dimensionless is invisible to
  the guard — the one move `toleranceDebt`'s own doc comment tells you not to make.
- **G10.** **The FIG8 cut piece meshes with 2 free edges at `DefaultQuality`** (0 at `PropertyQuality`), between
  a planar lid and the torus face at the lemniscate pinch. New in this wave — the base meshed it closed
  but wrong. Guard: `CodeMeshNotWatertight` fires on it, and
  `TestTheFigureEightTorusBandsPartitionTheTorus` holds both pieces' faces to their analytic share.
- **G11.** **The near-pinch crossing rods record `tessellate.cap-saturated` at `PropertyQuality`** — the corridor
  between two lens windows is narrower than the boundary's own chords. Guard:
  `TestTheHarvestCarriesEveryCodeTheFaceMeshesDo` (which requires the code to REACH the body harvest).

Still bespoke — arms the general chart mesher has not absorbed:

- **G12.** **`kindTwoRimHoledBand` keeps 8 corpus faces (near-pinch) and one uncharted-band configuration.**
  Guard: `TestTheTwoRimArmKeepsOnlyWhatTheChartCannotServe`, which fails in BOTH directions (ratio 40:
  the saddle band goes to the arm; ratio 0.001: nothing does).
- **G13.** **`kindSpiricBand` keeps the `occtparity` J3 and A4 hosts**, whose tori record chart = 0 so
  `chartFaceMesh` declines them outright. Deleting the arm was implemented and measured: J3 and A4 drift.
  Guard: the `occtparity` cluster itself.
- **G14.** **Twelve shape recognizers survive** behind eight classification arms (`curvedTrimRecognizers`).
  The classification is proved disjoint; the recognizers are not deleted. Guard: `recognizers` = 12,
  `TestCurvedTrimKindsAreMutuallyExclusive`, `TestTheClassificationCorpusReachesEveryArm`.
- **G15.** **Two payload-gated chains remain in `kernel/ops/tessellate`** as registered DEBT: `splineFaceMesh`
  (#3410) and `meshSeamCrossingFace` (#3411). Guard: `TestNoUnprovenPayloadGatedChains` — the registry
  may only shrink.

Unswept constants and ungated code:

- **G16.** **`chartContourIncidence` has no sweep.** It has a resolution argument (the stored contour is
  `math.Point2`) and three certifications an ungated retry regressed, but not the per-body table
  `chartBoundaryClearance` now carries.
- **G17.** **`chartNodeClearance` (0.3) has neither a `// tol:` annotation nor a sweep.**
- **G18.** **`chartBoundaryClearance`'s failure edge rests on ONE face** — the complement's lemniscate self-touch
  at `PropertyQuality`. A second self-touching boundary in the corpus would be worth more than another
  value of k. Guard: the two-sided pin `263.55487 ± 0.05`, proven to fire at k = 1.0.
- **G19.** **The `head/` module is outside every gate** in this wave: `go test ./...` at the repo root does not
  reach it, and neither does the lint run. Nothing in this wave touched it, but that is an assertion
  about the diff, not a measurement.

Accepted product decisions, recorded so they are not mistaken for oversights:

- **G20.** **An imported or adopted invalid body is REPORTED, not sickened.** `H3` and `H5` in the blend-parity
  corpus import from STEP with 3 boundary edges each — valid=false, closed=false — and refusing every
  imperfect import would block the user. The feature engine's `Validate` post-condition exempts the
  adopt/derive family and warns instead. Guard: `TestAnAdoptedInvalidBodyIsReportedNotRefused`,
  `TestAnAdoptedInvalidBodyDoesNotQuarantineDependents`.

#### The known pre-existing failure, re-proven

`model/exchange/translators/inventor` `TestMultipointDiskRebuildsAsAClosedSolid` fails at the wave base
and at HEAD with the same number. Re-proven for this close-out in a clean worktree at `c1e8f2a8`
(`GOWORK=off go test -count=1 ./...`, 4 m 58 s):

```
--- FAIL: TestMultipointDiskRebuildsAsAClosedSolid (140.70s)
  multipoint_disk_test.go:70: disk volume = 9111 mm³, want 7679 ±5% (Inventor 2027)
```

It is unrelated to this ADR — a translator rebuild, not a boolean — and it is the only red row in the
whole local gate at either end of the wave.

### Final fix wave, finding 1 — the rim-only ear (2026-09-08)

The close-out table above records FIG8− at `DefaultQuality` as "2 free edges between a planar lid and
the torus face" and G10 calls it a crack. Re-measured on the same HEAD with `tornAcrossMeshes`: the two
edges are of degree FOUR, `on=[lid lid torus torus]`, the edge (0.968, 3, +0.765)–(0.968, 3, −0.765) and
its mirror. No edge has degree 1. The torus chart mesh emitted the triangle (P_lo, pinch, P_hi) — the
first three points of a lobe chain — whose three vertices all lie on y = 3: it is the lid's own tip
triangle with the opposite normal, a doubled surface and not a crack.

The cause is the boundary clearance meeting a corner. At the pinch the material corner is ~102° wide
and both boundary chords are 1.23 mm; `chainIsNear` culls every interior node within
`chartBoundaryClearance · chord ≈ 0.68 mm` of a boundary segment, which is the whole corner, so the
constrained triangulation closes it with a boundary-only ear. At `PropertyQuality` the chords are ~8×
shorter and the body is watertight. Lowering the clearance is not the fix: k ≤ 0.55 loses the genus-1
complement at `PropertyQuality` (the sweep in `chart_face_clearance.go`).

The fix (`chart_rim_ear.go`): after the kept selection, every kept triangle whose three vertices are
boundary vertices AND which the weld will emit is split at the surface point under its own (u,v)
centroid — `s.PointAt(r.fold(centroid))`, a point the region already certified material, evaluated on
the surface, nothing moved — and the ONE constrained triangulation runs again over the enlarged point
set, at most `chartRimEarRounds` = 8 times; an ear still standing after that declines the face and the
router reports the discarded trim. No deviation threshold decides it. A covering triangle two of whose
rim vertices are one 3D point (the intersect piece's single loop passes its pinch twice) collapses at
the weld and is not an ear.

Three tests: `TestTheFigureEightPiecesMeshClosedAtBothQualities` (body level, both pieces, both
facetings: 0 free edges and no `CodeMeshNotWatertight`); both figure-eight pieces are rows of
`classificationCorpus()`, so every corpus gate meshes them each run; and
`TestNoChartedFaceEmitsARimOnlyTriangle`, the mesher-level invariant over every charted corpus face at
both facetings. All three were proven to trip with the split disabled — and the invariant found a
SECOND instance nobody had seen: the #1738 corner-junction wall at `PropertyQuality` emitted two
rim-only triangles that happened to be watertight (a chord plane cutting into the wall at the notch's
corner, not a neighbour's plane). Both are split now.

The chart corpus re-measured at both facetings, before → after (body free edges / charted face area, mm²;
`TessellateBody` and `chartFaceMesh` directly; "=" means byte-identical):

| row | face | D free | D area | P free | P area |
| --- | --- | --- | --- | --- | --- |
| RS− ring − coaxial shaft | torus | 0 = | 236.10153 = | 0 = | 237.87007 = |
| RS− | cylinder | 0 = | 56.10829 = | 0 = | 56.19711 = |
| RD− ring − axial drill | torus | 0 = | 290.28594 = | 0 = | 291.88065 = |
| RD− | cylinder | 0 = | 13.92081 = | 0 = | 13.94356 = |
| RODB∪ rod ∪ ball | sphere | 0 = | 2.86078 = | 0 = | 2.88193 = |
| RODB∪ | cylinder | 0 = | 24.85688 = | 0 = | 24.89270 = |
| RODB− rod − ball | sphere | 0 = | 0.25358 = | 0 = | 0.25928 = |
| RODB− | cylinder | 0 = | 24.85688 = | 0 = | 24.89270 = |
| RODB∩ rod ∩ ball | sphere | 0 = | 0.25358 = | 0 = | 0.25928 = |
| RODB∩ | cylinder | 0 = | 0.23655 = | 0 = | 0.23942 = |
| HALF ring − half space (the complement) | torus | 0 = | 263.55487 = | 0 = | 264.87111 = |
| FIG8− torus − y>3 | torus | **2 → 0** | **281.61313 → 281.51437** | 0 = | 283.07523 = |
| FIG8∩ torus ∩ y>3 | torus | 0 = | 110.87045 = | 0 = | 111.67495 = |
| DPRISM cyl ∪ D-prism (merged band) | cylinder | 0 = | 173.75394 = | 0 = | 174.09117 = |

FIG8−'s body volume at `DefaultQuality` moves 276.07755 → 276.15601 (analytic 279.898): the doubled
surface is gone. Its torus face area FALLS by 0.099 mm² because the re-triangulation over one more point
chooses different diagonals in the corner's neighbourhood — still a chord deficit under the analytic
283.100 (−0.56 %), inside the per-face gate's (−1 %, 0]. No pin moved: the complement's two-sided
263.55487 ± 0.05 reads 263.55487, RODB∩'s 0.2810 ± 0.005 is unchanged, and every other row is
byte-identical at both facetings. The ratchets are untouched (no new tolerance, recognizer, assertion or
code); `chart_face_mesh.go` reached its size limit and its clearance section moved verbatim to
`chart_face_clearance.go`.

### Final fix wave, finding 7 — the T-junction pass walks one order (2026-09-08)

`brep.splitTJunctions` iterated the live `edges` map while mutating it, and `tjSplitBudget` counted the
pair-adding splits in that order. Whether a given split adds a pair depends on which splits came before
it, so on a converging input near the budget, decline-versus-converge was a run-to-run coin toss — the
one outcome a refusal may not have. Each pass now walks a SORTED snapshot of the set through the same
total order `planarize` already sorted its output by (`sortedEdgePairs`, one function for both walks);
halves added during a pass are not in its snapshot and the next pass takes them.

The bound itself is unchanged, and the alternative the review named — counting DISTINCT pairs ever
added, which would make n(n−1)/2 a theorem — was implemented in thought and rejected: such a count is
bounded by n(n−1)/2 by construction, so it can never exceed the budget and the pass would never decline.
The runaway the bound exists for IS re-adding: at the `tjTol` scale a vertex that did not qualify on an
edge qualifies on the shorter half that replaces it (G9), and the r = 1.585e-7 drill would hang again.
The budget stays a budget; what is new is that its verdict is a function of the input alone.

Two determinism rows: `TestAConvergingArrangementArrangesIdenticallyEveryRun` (a comb of nine teeth
standing on one spine, twenty runs, cells printed in full and compared) and
`TestTheNonConvergentDrillRefusesIdenticallyEveryRun` (the r = 1.585e-7 drill, twenty runs, error and
every diagnostic record byte for byte). The drill row still refuses by name in 0.07 s.

### Final fix wave, finding 3 — a slit is one EDGE walked both ways, not one curve (2026-09-08)

`isReverseTwin` compared `a.curve == b.curve` on two `geom.Curve3` interfaces. That is a run-time
panic the moment both hold the same uncomparable dynamic type — reproduced with a value `geom.Polyline`
("comparing uncomparable type geom.Polyline") — and a marched section leaves exactly that on the edge
that carries it, so a boolean chained on a boolean's result could reach the cocylindrical merge with two
of them. It was also the wrong question: two loop edges can carry equal curves and be two edges.

A `loopEdge` now carries its source `*topo.Edge`, set by `orientedLoopEdge` and kept through every cut
of it (`splitEdgeAtPoints`, `reverseEdge`); `isReverseTwin` compares that identity and the swapped span,
exactly, and a synthesized edge (no source) is nobody's twin. A cut of the seam cuts both traversals at
the same parameters, so the pieces pair as the whole did. The stage-5 unit rows were rebuilt on real
edges through the topo builder (`seamWalkedWall`), and two Polyline rows were added: a value-polyline
seam walked both ways still drops, two different polyline edges beside one another stay, and neither
panics — the old comparison is proven to panic on the first.

The other three `==` sites on curve interfaces (`continuesCurve`, `sameRun`'s imprint/polygon arm,
`frameEdgeIsSeam`) compare curves an edge can carry as a marched value polyline too. They read identity
through `geom.SameCurveObject`, which guards with `reflect.Value.Comparable` on the VALUE (a struct type
with a `Curve3` field is comparable as a type while a value of it holding a Polyline is not — `SubCurve`
over a marched section) and answers false for a value that has no identity: such a frame edge re-emits
per recovered piece rather than panicking, and a polyline is never a seam. `TestSameCurveObjectNeverPanics`
covers each kind. Net delta: no new tolerance, recognizer, geometry-kind assertion or diag code; one
field on a private struct.

### Final fix wave, finding 6 — the recognizer count is derived from the classification's source (2026-09-08)

`curvedTrimRecognizers` was a hand-written registry checked only against the switch's case names and
"each name is still declared". A fourth sphere rim form, a second cone topology inside `coneApexTrimOf`,
or a new recogniser gate inside an existing arm moved `recognizers` by zero — the three events the
ratchet exists to catch.

The count is now derived (`archguard/recognizer_derivation_test.go`). Starting at `classifyCurvedTrim`,
a recogniser is a package function the classification READS for a shape verdict, found by syntactic
shape — a payload gate `if v, ok := f(…); ok`, a boolean gate `if f(…)`, an inventory read `v, ok := f(…)`
whose `ok` is never `!ok`-guarded, the positive operands of a returned `||`/`&&` (a negated call is a
guard), and a `case …: return f(…)` — and followed only when the callee is a VERDICT function, declared to
return `bool` or `(<…Trim payload>, bool)`. That contract is what every arm already keeps, and it is what
stops the walk at a geometric helper: `capAxis` returns a vector, `chooseSphereChart` a chart,
`splitWrappingHoles` two slices, and the first cut of the derivation followed all three into the mesher.
A function that reads nothing is a leaf and counts once; one that reads others counts itself too only
when it owns a verdict — a returned bool built from a call that is no read, `coneApexTrimOf`'s
`len(rim) != len(outer3D)` — so `classifySphereTrim`, `sphereCapTrimOf`, `sphereCapRimOfForm` and
`ruledTwoRimBandHolds` are dispatchers and count nothing. The derivation yields the twelve names the
registry holds, and the registry is now asserted equal to it NAME for name
(`TestTheRecognizerRegistryEqualsItsDerivation`); the sphere rim forms are asserted as one inventory
across the `sphereCapRimForm` constants, `sphereCapRimOfForm`'s cases and the registry
(`TestTheSphereRimFormsAreOneInventory`); and `countRecognizers` reads the derivation, not the table.

Proven to trip, in a throwaway that was reverted: a fourth rim form (`rimFormFake` + its case + its
function), a second cone topology (`if faceIsConeStub(f)` inside `coneApexTrimOf`) and a gate inside
`wedgeBandTrimOf` (`if w, ok := fakeWedgeTrimOf(f); ok`) all appear in the derived set, the inventory
reads 4 constants / 4 cases / 3 registered, and the pin reports `recognizers: 12 → 14`. The name-set
equality is the guard, not the count: the wedge gate turned `wedgeBandTrimOf` into a dispatcher (its own
verdict is a literal after guards), so that event alone would leave the COUNT at 12 while the names
differ. `recognizers` stays 12; nothing in the kernel changed.

### Final fix wave, finding 2 — a tear is worded by its degree; G10 corrected (2026-09-08)

`CodeMeshNotWatertight` reported every edge not shared by exactly two triangles as "free edge(s) … a
pair of neighbouring faces did not discretise the boundary they share the same way", and `meshTear`
carried each edge's degree without the report ever reading it. A degree-1 edge IS that (a crack); a
degree-3-or-more edge is the opposite defect — a doubled surface — and its only live firing, FIG8− at
`DefaultQuality`, was exactly that misdescribed. The report now partitions the tears by degree
(`partitionTears`) and words each class on its own (`tearDetail`): cracks name the shared-boundary
disagreement, over-merges name a coincident or twice-meshed triangle, and a mesh carrying both says
"torn AND doubled". `TestADoubledMeshOfAClosedSolidIsReportedAsOverMerged` duplicates one triangle of a
box face and asserts the detail names an over-merge and not a crack; the crack row asserts the
converse.

**Correction to G10.** G10 reads "The FIG8 cut piece meshes with 2 free edges at `DefaultQuality` …
between a planar lid and the torus face at the lemniscate pinch" and the close-out table calls it a
crack. The two edges were of degree FOUR, `on=[lid lid torus torus]`: the chart mesher's boundary-only
corner triangle at the pinch coincided with the lid's tip triangle. It was a doubled surface, not a
crack, and it is gone (finding 1). G10 is closed.

### Final fix wave, findings 4 and 5 — the complement is an oval, and the node clearance is measured (2026-09-08)

**Finding 4, a false geometric claim.** Three earlier sections (the pre-existing-defect note, the bisect
correction, and G18) and the code comments beside `chartBoundaryClearance` say the ring − half-space
complement's section is "the LEMNISCATE", that "the boundary passes through the same 3D point twice", at
(u,v) = (3π/2, π/2) and (3π/2, 3π/2). Measured on HEAD: the torus R=5 r=1.5 cut by the plane x = R keeps
ONE loop of two `SpiricArc` edges, (5, 0, −1.5) → (5, 0, +1.5), a single smooth oval; the face's chart
is the whole domain minus one 512-point hole; and the two cited (u,v) points are the oval's top and
bottom, 3.0000 mm apart. The lemniscate is the figure-eight fixture (d = R − r = 3 on the R=5 r=2
torus), a different body. The clearance's failure mechanism is unchanged and correctly described
otherwise — the rim's chord is coarsest in u at the oval's APEX (0.17 rad in one chord against the
covering's 0.0245 stations) and an interior node lands inside that chord — only the shape was wrong.
Corrected in `chart_face_clearance.go`, `chart_face_mesh_test.go` and `chart_monotone_refinement_test.go`;
the earlier ADR text stands as written, superseded here.

**G18 re-stated.** `chartBoundaryClearance`'s failure edge rests on ONE face: the complement's torus at
`PropertyQuality`, at the oval's apex, where the rim's chord is coarsest in u. A second boundary sampled
that coarsely at a turn would be worth more than another value of k. Guard unchanged: the two-sided pin
263.55487 ± 0.05, proven to fire at k = 1.0.

**Finding 5, `chartNodeClearance`.** It was a bare `0.3`; it now carries `// tol:mesh-density` and the
sweep G17 asked for. It is a FLOOR under the chord clearance (`chainIsNear` takes `max(gridMargin,
0.875·chord)`), and over the whole chart corpus — RS−, RD−, the three RODB rows, the complement, both
figure-eight pieces and the merged band, both facetings, free edges and every charted face's area:

| k | not watertight | faces whose area moved from k = 0 | chains where the floor wins the max |
| --- | --- | --- | --- |
| 0.0 | none | — | 0 |
| 0.1 | none | none | 0 |
| 0.2 | none | none | 0 |
| 0.3 | none | none | 3 (the RODB∪/RODB−/RODB∩ rod walls' lens windows at `PropertyQuality`) |
| 0.4 | none | RODB∪/RODB− rod wall @D 24.85688 → 24.85677 | 5 |
| 0.5 | none | RODB∪/RODB− rod wall @D 24.85688 → 24.85680 | 8 |
| 0.75 | none | RD− torus @D 290.28594 → 290.28632; rod walls @D → 24.85662; lens patches @D 0.25358 → 0.25330 | 12 |

Which branch wins: at `DefaultQuality` the chord clearance wins on every chain of every face; at
`PropertyQuality` the grid floor wins on exactly three chains — the lens windows on the rod walls, whose
chords are shorter than 0.3 of a grid gap — and where it wins it culls no node the chord clearance had
not already culled: every face is byte-identical from k = 0 to k = 0.3. The first node goes at 0.4. So
the floor decides no corpus mesh today; it stays because the failure it guards (a node on a constraint
derails segment recovery) is real and its cost at 0.3 is nothing. The annotation on
`chartBoundaryClearance` said "swept 0.125…4" against a table of 0.50…3.00; it now says what the table
shows. G17 is closed; no pin moved.

### Final fix wave, finding 8 — the seam-crossing router re-measured (2026-09-08)

The `payloadGatedChains` entry for `meshSeamCrossingFace` said "every face still reaching it has the
chart mesher decline (measured 182 of 182), so it is the chartless-face path". That stopped being true
when `singlyPeriodicWrapMesh` began routing charted singly-periodic bands through it and the chart
mesher accepting them. Re-measured by instrumenting the router's exits over `go test -count=1
./kernel/...` (2 m 07 s, every package green; the instrumentation was reverted): 196 faces reach it.

| exit | faces | chart mesher's verdict |
| --- | --- | --- |
| `closedDomainMesh` (cylinder 117, cone 8, offset 2) | 127 | declines (no chart) |
| `closedBandLoftMesh` (torus) | 39 | declines (no chart) |
| `HoledConicWallMesh` | 9 | declines (no chart) |
| `saddleBandLoftMesh` | 8 | declines (no chart) |
| `unequalRimBandMesh` | 2 | declines (no chart) |
| `chartedTrimMesh` (torus, uncharted) | 7 | declines → reported full domain |
| `singlyPeriodicWrapMesh` (cylinder, charted) | 4 | **accepts** |

So the honest statement is: every face taking one of the five bespoke rungs (185 of 185) has the chart
mesher decline and records no chart, while the router also carries four charted bands the chart mesher
serves and seven uncharted tori that fall to the reported full domain. The entry is reworded to that;
it stays DEBT under #3411 and the registry may still only shrink. The earlier "133 of 133 / 182 of 182"
figures in the classification section above were true when measured and are superseded by this table.

### Platform stability, measured (2026-09-09)

CI run 34280554924 was green on every job but `test (macos-latest)`, where twelve rows failed across
`kernel/geom`, `kernel/brep`, `kernel/ops/boolean` and `model/feature`. macOS runners are arm64, and
the Go compiler FUSES `x*y+z` into a single-rounding FMA there and never on amd64. So the branch's
own arithmetic differed by an ulp on one platform, and four decisions that should not have been able
to see an ulp saw it. The rule each broke is the ground rule that **output is byte-identical across
runs and platforms**, and, under it, that a tolerance is classified by the origin of its operands.

Reproduction, without a macOS machine — cross-compile natively and run the test binary under
`binfmt` emulation, which is ~40× faster than compiling inside the container:

```
GOARCH=arm64 go test -c -o /tmp/pkg.test ./kernel/ops/boolean
docker run --rm --platform linux/arm64 -v <workspace>:/ws -w /ws/<worktree>/kernel/ops/boolean \
  golang:1.27 /ws/<worktree>/.armbin/pkg.test -test.run '<Row>$' -test.count=1 -test.v
```

To prove FMA is the whole cause, and then to find WHICH fusion: `-gcflags=all=-d=fmahash=<bits>`
enables FMA only where the position hash matches, so a 40-bit pattern disables it everywhere (the row
goes green ⇒ FMA is the cause) and prefix bisection over `0`/`1` narrows to the sites; add a leading
`v` to have the compiler print each matched `file:line`.

The four causes, none of them "a formula that needed rounding":

1. **Two spellings of one quantity** (`torus_quadric_harmonic2.go`). The axis-invariant station's
   level was written twice — `constant + m11·ρ²` and `constant + ρ²(m11+m22)/2`. arm64 fused the
   first and not the second (a division blocks fusion), so the reproduction proof that the arccos path
   IS the general path written out failed by one ulp. `harmonic()` now READS the general form's level;
   that one expression rounds its product explicitly.
2. **A weld grid finer than the coordinates it compares** (`geom.CurveSpanBox`, `brep.faceLoopBox`).
   `CurveBox` declines a curve kind with no closed-form axial extent and documents that the caller must
   then bound it by sampling; `faceLoopBox` never did. A face bounded by ONE closed spiric — each lobe
   of the oblique torus figure-eight — therefore measured its own scale as a POINT, took the 1e-9
   model-size floor, and welded its stitch on a **1e-15** grid. The lobes' shared pinch point is read
   twice; on amd64 the two readings were bit-identical and on arm64 1.3e-15 apart, so the seam tore
   open into unpaired edges and the exact result failed its own acceptance gate.
3. **One incidence, two windows** (`brep.spanIsRimContact`). `bandPlacement` pads both its verdicts, so
   a crossing sitting ON a wall's rim is neither "inside" nor "clear" and falls to the clip — which
   measures against the UNPADDED band, finds nothing between the rims, and refuses the boolean. Which
   window a rim crossing landed in was decided by the last bit of its axial coordinate: the chamfer
   wedge's cone meets its shaft wall exactly at the wedge's own rim, inside the band on amd64 and
   outside it on arm64. A crossing that IS a rim now classifies as such and is dropped — it imprints
   the edge the face already carries — so only a true straddle reaches the clip.
4. **A branch root read at a fold** (`geom.torusFoldAzimuth`). A folded section loop's window ends are
   bisected roots of the discriminant, so the discriminant there is zero only to rounding; where it
   rounds POSITIVE the branch pair still separates, by half a square root of that rounding — ~1e-8 in
   azimuth, ~1e-7 in position. `TorusQuadricLoop` now reads the MERGED azimuth at its folds (the lane's
   own extremum, or the one-harmonic form's fold phase) instead of a branch root, which is what the
   type always documented. Before it, a tilted drill's section closed 1e-7 off its host wall's own seam
   ruling on arm64 and on it on amd64, and the wall's chart arranged into a different set of cells.

Causes 2, 3 and 4 are latent defects, not arm64 defects: each is a decision taken at a precision
finer than the quantity deciding it, and the FMA difference only chose which side of it this corpus
landed on. Each carries a regression test that fails on amd64 too (the fold tangency and the
degenerate face box), or a unit test of the predicate that closes the gap (the rim contact).
