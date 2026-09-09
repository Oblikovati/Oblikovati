# ADR-0066 — A torus is an implicit form for the torus section, and the pair is solved

**Status:** Accepted — on `m48/torus-torus-section` (Oblikovati#3514). · **Supersedes**
[ADR-0061](ADR-0061-csg-fallback-retirement.md) §"What is still refused, and what is still bespoke",
row **G1**, which records the torus pair as a standing refusal on a premise this ADR shows to be
false. G1 is retired here; the rest of ADR-0061 stands, and ADR-0065's retirement of G2 is untouched.
· **Deletes:** `geom.torusAgainstQuadric` and the intersector's two-role try-list for a torus pair;
the coaxial family's 720-probe level scan and its `bisectLevelRoot`; `geom.coaxialTorusQuadric`'s
720-probe reach sweep; the `torusStation`'s tensor fields, which no longer describe every station it
carries. ·
**Touches:** `kernel/geom` (the torus bucket), and the six guards in `kernel/geom`, `kernel/brep` and
`kernel/ops/boolean` that named the torus pair as the pair nothing claims.

## Context

ADR-0061 stage 5 gave the torus a bucket of its own. A torus is quartic and has no implicit quadric,
so it cannot be the *implicit* side of the intersector's parametric×implicit substitution; but its own
chart is affine in the azimuth, so it can be the *parametric* side, and any quadric substituted into it
reduces — station by station — to a five-coefficient trigonometric polynomial of degree two.

The close-out table recorded torus × torus as G1, refused by name:

> Neither torus supplies an implicit quadric, so the second-harmonic reduction (quartic in tan(u/2))
> does not apply.

**That premise is false.** The second harmonic does not come from the other side being a quadric. It
comes from the tube circle the chart torus sweeps, and it is there whatever the other side is.

### The algebra

A torus's implicit form, relative to its centre, with `W = X − C` and `â` its axis:

```
F(X) = (|W|² + R² − r²)² − 4R²(|W|² − (W·â)²)
```

Restrict it to ONE tube circle of the chart torus, `P(u) = O(v) + ρ(v)·e(u)`. Two terms carry the
azimuth, and **both drop a degree on a circle**:

```
|W|²   = |W₀|² + 2ρ(W₀·e) + ρ²|e|²  = (|W₀|² + ρ²) + 2ρ(b₁cos u + b₂sin u)
(W·â)² = (a₀ + ρ(n₁cos u + n₂sin u))²
```

The first is a FIRST harmonic exactly — the quadratic term is the constant ρ², because `|e| = 1`. The
second is a first harmonic squared, so a second. `F` is `(first)² − 4R²(first − second)`, which is a
SECOND harmonic: **the same five coefficients a quadric produces**, read by the same station solver,
paired into lanes by the same extremum structure, folded by the same window finder.

The geometry says the same thing independently. Bézout puts a conic against a quartic at eight points;
a CIRCLE passes through the two circular points at infinity, each of which lies on the absolute conic
that every torus quartic contains doubly, so four of the eight are spent there and **four finite
intersections remain — a quadric's count.** Measured over 4000 random stations before any code was
written: the maximum number of sign changes per station is 4, never more.

## Decision

### 1. The bucket's requirement is a DEGREE, not a type

`TorusCoForm` (`kernel/geom/torus_section_form.go`) is what the torus reduction asks of the other
surface: an implicit form whose restriction to a circle is a degree-two trigonometric polynomial. It is
sealed — its methods are unexported — because implementing it is a claim about that degree, and the
whole lane structure rests on the claim being true. `Quadric` and `Torus` are its two members, and
there are exactly two for the reason above rather than because the list stopped growing.

Measured: the coefficients reproduce `F(P(u,v))` to **4.56e-14** of the polynomial's own coefficient
scale over 20 000 random `(chart, other, u, v)` — against the **1e-9** a certified root is allowed —
and every certified station root lands on **both** surfaces to **9.23e-14** (284 roots over 400 random
pairs). Both are corpus rows.

### 2. The quadric family reproduces bit for bit

`torusStation` now carries the five coefficients, the one-harmonic reading and the classification,
rather than the quadric's tensor entries. `Quadric.stationOn` fills them with the expressions that
stood before, character for character — including the `float64()` on `rr*(m11+m22)/2`, which ADR-0064
identified as a live fusion site, and the LEVEL spelling that `harmonic()` reads rather than respelling
(ADR-0061's arm64 divergence). Nothing about the quadric family's arithmetic changed.

The one classification that could have changed is which family a pair takes, and each form answers it
from the representation that carries the property: a quadric from its TENSOR (`quadricIsAxisInvariant`,
untouched, a statement about every tube angle at once), a torus from its STATIONS. They are the same
predicate on the two representations, and neither reads the other's.

**One deliberate exception, and it is the subject of §4:** a coaxial quadric's section CIRCLES are not
bit-identical, because their tube angles moved from a 60-step bisection of a 720-station scan to an
arccos. That output is exact where it was approximate. Every statement above about bit-for-bit
reproduction is about the station COEFFICIENTS and the lane family that reads them.

### 3. The role assignment is a classification, not a try-list

The intersector used to try `torusAgainstQuadric(a, b)` and then `(b, a)`. For a torus/quadric pair at
most one applies, so that was a role assignment; for a torus PAIR both apply, and it would have been a
first-fit ladder whose answer depended on the order the arrangement handed the faces over — two
parametrisations of one curve, two sets of bytes downstream.

`torusSectionRoles` makes ONE assignment. For a torus pair `torusChartPrecedes` settles it with a total
order: **the fatter tube takes the chart**, ties broken lexicographically on the radii, the centre and
the axis, compared exactly. `TestTheChartAssignmentDoesNotDependOnTheCallerOrder` drives 2000 random
pairs in both argument orders.

The rule is a conditioning judgement and it was **written backwards first**, on the reasoning that the
thinner chart moves its stations least. Measuring is what said so. What decides the lane structure is
the CO-FORM's tube: a thin one is nearly a wire and cuts a simple, stable root structure on a circle,
while a fat one cuts a structure whose extremum COUNT changes over the turn. Over 1800 / 1775 / 1797
meeting random pairs at three seeds, the fat-chart assignment builds 608 / 678 / 651; the thin-chart
one loses on every seed and wins on none. An assignment by absolute minor radius is within noise
(607 / 663 / 646); the dimensionless ASPECT is kept so the same pair in metres and in millimetres takes
the same chart.

### 4. The coaxial family is SOLVED, not sampled

How many circles a coaxial section has, and where they sit, is a topological question. It used to be
answered by scanning the level at 720 tube angles and bisecting each sign change 60 times — and a grid
answers a topological question wrongly whenever the feature is narrower than the grid, with no amount of
bisection afterwards able to recover a crossing the scan stepped over. That is the shape ADR-0065's
review found fatal one bucket over.

There is nothing to sample. For **a quadric**, `Level(v) = constant(v) + ρ(v)²(m₁₁+m₂₂)/2` with
`ρ = R + r·cos v` and `W₀(v) = w + r·sin v·â`, which expands to a degree-two trigonometric polynomial
in v — the same shape `torusSecondHarmonic` carries in the azimuth, read by the same exact quartic
solver. For **a torus** the level FACTORS, and each factor is a first harmonic:

```
Level = ((ρ − R_b)² + a₀² − r_b²)·((ρ + R_b)² + a₀² − r_b²)
F±(v) = (R ± R_b)² + d² + r² − r_b² + 2(R ± R_b)·r·cos v + 2d·r·sin v
```

because the `r²cos²v` and `r²sin²v` collapse to the constant `r²`. Each is `level + reach·cos(v − phase)`:
an arccos, exact, two roots or none. Solving BOTH factors is what makes it right for a spindle torus as
well as a ring — they are the two halves of the co-form's meridian circle.

The CLASSIFICATION is exact too, and also read once rather than probed:

- a quadric's reach components are `A + B·sin v`, so they vanish at every v exactly when `A` and `B` do
  — four scalars. A function of that shape with 720 zeros is identically zero, so this agrees with the
  sweep it replaces wherever the sweep was right, and cannot be stepped over.
- a torus co-form is one-harmonic exactly when it is COAXIAL (§ the traceless-part argument in
  `sectionFamily`), which is two geometric statements: the axes parallel, and the co-form's centre on
  the chart's axis. One dimensionless direction test and one length against the chart's own weld.

**This is not cosmetic, and the corpus proves it is not.**
`TestACoaxialSectionNarrowerThanTheOldGridIsStillFound` derives a coaxial pair whose window falls
strictly between two of the old scan's samples — centre half a grid step off a probe, half-width an
eighth of a step, which fixes `d` and `r_b` from the algebra above. The closed form returns its two
circles at **1.3e-15** from both surfaces; the 720-probe scan of the same level finds **zero** sign
changes and would have returned the honest-looking empty section. The row asserts both halves, so it
cannot pass vacuously.

### 5. A new post-condition: the section's own points, measured as a length

`torusSectionSatisfiesItsForm` reads 257 points of every section curve and requires each to lie on the
co-form's surface, within the modelling weld at the chart torus's own reach.

It exists because every certificate before it certifies a PART — a root where it is solved, a fold
azimuth that is the station's own extremum rather than a root, a lane labelled by an anchor carried
from another station, a census that counts branches rather than placing them. Their composition can
still put a point off the surface, and two co-centred PERPENDICULAR rings do: their branch pair is
tangent at v = 0 and v = π, the turn splits into two windows whose folds are that tangency, and the
fold reads a lane extremum that is not the merged root. Measured: the section came back `ok=true
why=none` with points **1.353e-5** off the surface they claimed, past the anchor gate, the ownership
gate, the separation gate and the azimuth census alike.

Three things about it are deliberate and each was measured:

- **A distance, not the polynomial's residual.** At a fold `df/du` is zero by definition, so `f` falls
  off quadratically in the azimuth error: that same 1.353e-5 displacement reads as a residual far under
  what a certified root is allowed. A residual gate is blindest exactly where this failure lives.
- **257 samples.** The excursion peaks at t ≈ 0.9875 and has fallen to 5.6e-10 by t = 0.999. A grid of
  65 steps clean over it (worst 3.6e-15); 97 does not. 257 catches it with a 500-fold margin and costs
  a third more than the section it certifies (193 ms → 257 ms for twenty skew-rod sections). A spike
  narrower than one part in 256 of a curve's parameter is still stepped over: that is a bound this gate
  has, and refining it belongs with the fold refinement ADR-0065 already owes.
- **The weld comes from the CHART, not from the caller's `Resolution`.** The closed-surface pairings
  hand in `ResolutionForBox(faceLoopBox(f))`, and a boundary-less face — which is every torus before
  anything is imprinted on it — has no loops, so that box is empty and the weld collapses to ~1e-18.
  `kernel/brep`'s `declineOpenSection` records the same defect and works around it by loosening its
  class to `Sew()`; loosening is not available here, because `Sew()` at these sizes is 2e-3 and the
  excursion is 1e-5.

**What the post-condition costs the quadric family: nothing.** Over 2000 random cylinders driven
against the corpus ring, **871 sections built before it and 871 after**. Over 2000 random torus pairs,
412 before and 411 after — the one it removes is a genuine excursion.

### What is still decided by sampling, and whose it is

The LANE family's window finding is `periodicRootWindows`, at 720 stations. It is shared with the ruled
bucket, it is not introduced here, and this ADR does not make it exact: doing so means finding the
discriminant's roots in v exactly, which is a resultant over the station's extrema — its own change.
Its gate here is the sampled post-condition of §5, whose blind spot is stated with the number.

So the honest summary of this bucket after this ADR: the reduction is exact, the per-station root solve
is exact, the coaxial family is exact end to end, and the lane family's window topology is sampled and
inherited.

## What is certified

- **The reduction** — `TestTheTorusReductionIsTheOtherTorusQuartic`, against the quartic written from
  its definition in the test.
- **The roots** — `TestEveryTorusPairStationRootLiesOnBothTori`, against a revolved-meridian distance
  oracle that knows nothing about the reduction.
- **The section** — `TestATorusPairSectionLiesOnBothTori`: the built curves sampled on both surfaces
  (worst 1.4e-14 across the corpus), each closing on itself, and the CURVE COUNT against the number of
  connected components of the chart's zero set, counted by union-find on a 512² sign grid. That count
  shares nothing with the reduction beyond the five coefficients, and it is what makes "the same
  topology the general path returns" a measurement.
- **The bodies** — `kernel/ops/boolean/boolean_torus_pair_test.go`, three pairs × three operations:

  | layer | gate |
  | --- | --- |
  | per face | every face's surface is one of the two OPERANDS (not merely "a torus"); every face's loop count |
  | boundary | each result's ANALYTIC area against a chart integral of the two tori's own membership |
  | whole body | Requicha at BOTH facetings; volumes against a stratified membership integral |

  The boundary gate is the one that earns its place. The section cuts the two boundaries into four
  regions; the three operations keep three different pairs of them, so matching all three pins all
  four, and a face that kept the complement of its region cannot hide. Measured over the nine rows:
  worst **9.3e-4** relative, best **2.0e-7** — and that is the ORACLE's resolution (a 1200² midpoint
  rule), not the kernel's.

  Requicha (`V(∪)+V(∩) = V(A)+V(B)`, `V(−)+V(∩) = V(A)`) holds at the display faceting and the property
  faceting alike, within 1e-5.

## What is still refused, and by what name

Over **2410 meeting random ring-torus pairs** with the chosen chart:

| outcome | count |
| --- | --- |
| built | 821 |
| `DeclineTorusLaneTracks` — the extremum tracks are not separable | 1538 |
| `DeclineTorusLaneFullTurn` — the branch pair never folds | 25 |
| `DeclineTorusLaneUnaccounted` — the loops do not account for every azimuth | 25 |
| `DeclineTorusSectionOffItsForm` — the section's own points are off the form | 1 |

Every one of those is a NAMED refusal that reaches the boolean as a `CodeSectionConditioningDemotion`
Defect. Each has a corpus row (`TestATorusPairOutsideTheEnvelopeIsRefusedByName` in `kernel/geom`,
`TestATorusPairOutsideTheEnvelopeIsRefusedByName` in `kernel/ops/boolean`).

**`DeclineTorusLaneTracks` is the dominant gap, and no single cause explains it.** `torusLaneAnchors`
seeds the lane labels from the station at v = 0 and requires all 720 stations to carry the same number
of extrema, tracking those seeds — *including stations that carry no section at all*. A torus co-form's
station changes between two and four extrema over the turn far more often than a quadric's: the boss
fixture carries roots at 77 of 720 stations and four extrema at 235, so there the 643 root-free stations
decide the gate.

**That mechanism is real but it is a MINORITY of the population, and the scope must be written from the
measurement rather than from the mechanism.** Over 300 sampled `LaneTracks` refusals the first station
that breaks the gate carries **no** roots in 127 and **does** carry roots in 173. Measuring the obvious
fix directly — seed from the first root-carrying station and judge only root-carrying stations —
recovers **268 of 1538 refusals, 17.4 %**, taking the build rate from 34.1 % to at most ~45 % of meeting
pairs. In population terms the follow-up is **64 % → about 53 % still refused**, not "closes the gap".
The remaining 173-in-300 majority breaks at a station that does carry roots, where no change to the seed
can help: those need the extremum tracks themselves to be continued through a count change, which is a
different and larger piece of work. Sizing the ticket by the mechanism instead of by the measurement
would send the next implementer after a sixth of the problem believing it was the whole.

Either way it is not done here, and relaxing the seed at all needs its own bit-for-bit reproduction
proof over the quadric corpus, because it moves the anchor for pairs that build today.
`DeclineTorusLaneFullTurn` is deleted by ADR-0065 on a sibling branch, which will convert its 25 rows.

## Two defects surfaced by the measurements, neither fixed here, both bisected

1. **A torus face bounded by a torus × torus section loop does not mesh AT PROPERTY FACETING.** The
   B-rep is exact — the boundary oracle agrees to a part in ten thousand — and so is every mesh a user
   can reach. At DefaultQuality, the display density, the face is correctly trimmed (284.98 against a
   whole-torus 296.09, no diagnostic), and all three export presets are clean and diagnostic-free
   (Low −4.76 %, Medium −0.56 %, High −0.14 %, which is faceting bias, not this gap); the finest
   export preset is about 12× coarser than the flip. The gap appears only at PropertyQuality
   (chord 0.001 / 1°), where the face meshes 296.06 — the whole domain — and both defects fire
   (+7.9 % join, +3.1 % cut, plus `mesh-not-watertight` on Cut). Swept: clean to chord 0.001 at 5°,
   declining from about 0.0005.

   It therefore does NOT reach feature health, and that is correct rather than a hole in the
   reporting: `bodyDegradations` reads `displayQuality()`, and at that density there is nothing
   wrong to report. Do not wire property diagnostics into it to "fix" this — that would report a
   defect no user can see. Mass properties integrate the analytic B-rep and are exact to 1.4e-16. It is not silent: the
   tessellator records `tessellate.trim-ignored-full-domain` and `tessellate.chart-mesher-declined` as
   Defects, and `TestTheHoledTorusFaceDeclinesItsMeshByName` pins that it keeps saying so, inverting
   when the chart mesher takes the shape. The gap is specific to this boundary: the same ring cut by an
   AXIAL DRILL leaves a two-loop torus face whose mesh IS bounded by its rim (291.88 against a
   whole-torus 296.09, no diagnostic).
2. **A built quadric-family section can carry a NaN point.** An unreadable station makes the lane
   reader answer NaN by design, and that NaN reaches a coordinate. Fixture: a cylinder at origin
   (−3.3617, 6.2435, 0.0778), axis (0.2568, −0.9663, 0.0158), radius 0.8992, against the corpus ring —
   two NaN samples at 1025 points per curve, none at 257. It is PRE-EXISTING, **bisected** (see below),
   and measure-zero, so no point-sampling gate can be its answer: the fix is that the reader should
   refuse rather than answer NaN. `withinWeld` is written as a positive test so that a NaN it does
   sample refuses rather than passing.

**Both were bisected against the wave base `d919a151`, in a clean worktree** (global constraint 2 — a
bisect is the only proof, and code locality is not one), and they came back DIFFERENT, which is why the
wording above separates them:

- **The NaN IS pre-existing.** The recorded fixture run on the base returns `ok=true why=none` with four
  faces and **4 NaN samples of 16 388**, worst finite distance to the ring 1.6e-15. The defect predates
  this branch; only its measurement is new.
- **The un-meshed face is NOT pre-existing, and the first draft of this ADR was wrong to imply it was.**
  The torus pair cannot be built on the base at all, so the exact face cannot exist there; the nearest
  question that CAN be asked is whether a two-loop torus face bounded by torus-section curves already
  fails to mesh, and on the base it does not — a ring cut by a skew rod, an axial drill and a tilted
  drill mesh at 289.31 / 291.88 / 291.88 against a whole-torus 296.09, with no
  `trim-ignored-full-domain` on any of them. **The gap arrives with this branch**, on a boundary kind
  the chart mesher has not met before. What that changes is only the word: it is still refused by name
  rather than shipped quietly, and `TestTheHoledTorusFaceDeclinesItsMeshByName` still pins that.

## Consequences

- One capability added; one named decline added (`DeclineTorusSectionOffItsForm`) and none removed.
- Two sampled topological decisions deleted (the coaxial level scan and the coaxial reach sweep), one
  bisection deleted (`bisectLevelRoot`), and no sampled decision added.
- **Every archguard ratchet is unmoved, with no pin edited**: `tolerance-constants` 214,
  `type-assertions` 684, `recognizers` 12, `fallback-sites` unchanged (no `diag.Code` added — a
  `SectionDecline` is not one), `mixed-decline-returns` 3. The role classification asserts on two types
  where `torusAgainstQuadric` asserted on two, so the count is flat by construction.
- `make fma-gate` is unmoved at `total 10, declared residual 10`.
- The torus section's types are renamed to say what they now carry: `TorusQuadricArc` →
  `TorusSectionArc`, `TorusQuadricLoop` → `TorusSectionLoop`, `CurveTorusQuadric` →
  `CurveTorusSection`, `TorusQuadricSection` → `TorusSection`, and the six files with them. A curve
  whose `Quad` field holds a torus is a name that lies.
- **`CodeSectionUnclaimedPair` now has no fixture in the kernel's primitive vocabulary.** A sweep of
  {torus, sphere, block, cylinder, cone} against each other in all three operations reaches it from
  nothing: the torus pair was the last pair no closed form claimed. Its routing is therefore tested
  where the decision is made (`kernel/brep`'s `TestAnOrdinaryRefusalIsRecordedAsInfo`) rather than
  through a fixture chosen to provoke it, and that test keeps holding when the vocabulary next gains a
  pair the intersector does not claim.
