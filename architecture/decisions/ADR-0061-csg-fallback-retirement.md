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
