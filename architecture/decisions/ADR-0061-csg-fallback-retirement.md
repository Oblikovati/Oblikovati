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
