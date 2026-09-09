# ADR-0065 — The torus section carries full-period branches, and certifies them by position

**Status:** Accepted — on `m48/torus-lane-branches` (Oblikovati#3515). · **Supersedes**
[ADR-0061](ADR-0061-csg-fallback-retirement.md) §"What is still refused, and what is still bespoke",
row **G2**, which records the fat rod as a standing refusal. G2 is retired here; the rest of ADR-0061
stands. · **Deletes:** `geom.DeclineTorusLaneFullTurn` and `periodicRootWindows`'s rise/fall parity
branch. · **Touches:** `kernel/geom` (the torus∩quadric bucket), and the three demotion guards in
`kernel/brep` and `kernel/ops/boolean` that were driven by the refusal this ADR removes.

> **Reserved.** A second section is expected on this ADR later in the same wave: Oblikovati#3516 owes
> a further correction to ADR-0061's close-out table. Append it; do not edit what is below.

## Context

ADR-0061 stage 5 gave the torus∩quadric bucket two reductions selected by one classification
(`quadricIsAxisInvariant`): a one-harmonic arccos where the quadric's quadratic form is invariant
about the torus axis, and a general second-harmonic form where it is not. The general form built
FOLDED loops only — a pair of azimuths bounded by the tube angles where the two merge — and named
every other shape a conditioning demotion.

One of those shapes is an ordinary part. A rod driven across a ring, **thicker than the ring's tube**,
contains the whole tube cross-section it crosses: it swallows the ring's flank rather than piercing
it, and severs the ring into a C. Measured on the corpus ring (R = 5, r = 1.5) against a cylinder of
radius 2 on the ring's own centre-circle tangent, the station polynomial carries **four certified real
roots at all 720 station probes** and every lane's discriminant is positive at all 720. There is no
fold anywhere in the turn. The reduction declined `DeclineTorusLaneFullTurn`, the boolean recorded a
`CodeSectionConditioningDemotion` defect and refused to build, and ADR-0061 wrote that refusal down as
a deliberate boundary (G2).

It was not a boundary. It was a missing case: the one-harmonic half of the same bucket has always
carried exactly this shape, as two full-period `TorusSectionArc`s.

## Decision

### 1. The unit that wraps is the TRACK, not the lane's pair

A lane is one extremum of the station polynomial with the root on each side of it. Each of those roots
is a **track**: the arc of azimuths between two neighbouring extrema, which carries at most one root
because *f* is monotone there. `torusLane.trackDiscriminant(upper)` is `−f(centre)·f(that flank)`,
positive exactly while that one azimuth exists as a root distinct from an extremum. The lane's own
discriminant is now literally `min(trackDiscriminant(false), trackDiscriminant(true))` — the pair
exists while both its tracks do — spelled in the same operand order it had before, so it is bit-for-bit
the previous expression.

A track that never merges with a neighbour over the whole turn is a branch running the tube's whole
period, and it is built as a `TorusSectionArc` over `[0, 2π)`.

### 2. Consecutive lanes share a track, so each lane reads its UPPER track only

Lane *i*'s upper track is lane *i+1*'s lower track. The upper tracks therefore tile the azimuth circle,
and reading each lane's upper track once names every azimuth the station carries **exactly once, for
any number of lanes** — two for the axis-invariant family, four for the fat rod, whatever a station
happens to produce. Nothing in the reduction contains a branch count.

The two questions are independent: a lane may contribute an arc *and* a folded loop. That is what a rod
which reaches the ring over part of its turn and swallows it over the rest looks like, and it now
builds (a rod along +x offset to y = 2, radius 2: one wrapping lane, three windowed ones, root counts
of 2 at 145 stations and 4 at 575, section `LAA` — one loop and two arcs).

### 3. The certificate compares POSITIONS, not a tally

`torusCurvesAccountForEveryAzimuth` walks the stations and requires the azimuths the finished curves
**take** to be exactly the azimuths the station's quartic certifies there: as many carried as
certified, no two curves on one branch, and every carried azimuth one of the certified roots.
Coincidence is judged by `sortedDedupedAngles`, the same weld `azimuths()` itself uses to decide that
two roots are one, so the certificate and the solver agree about what "the same branch" means.

It was a tally in the first cut, and review round 1 planted the case that proves a tally is not a
certificate: four arcs forced onto one lane carry the right *number* of branches, three of them
duplicates and three certified roots carried by nothing, and the section came back `ok=true why=none`.
The ground rule is a statement about where the curve **is** — "certify a root or branch choice at
runtime against the geometry (position, second-order test)" — so the certificate reads positions.
`TestFourArcsOnOneLaneAreRefused` is that plant, kept as a row.

The certificate samples at **half** the construction's station step, so it reads every station the
sweep read and the midpoint between each neighbouring pair. A grid stepping in lockstep with the thing
it certifies cannot correct a sweep that stepped over a feature. This halves the step it can step over;
it does not abolish it, and refining the sweep to the minimum of the track discriminant (the way
`foldStation` refines a crossing) is the standing follow-up, not something a finer grid replaces.

### 4. Two different refusals stop sharing one name

A station where the quadric **touches** the tube circle instead of crossing it carries a double root
that no pairing of branches can bound. That is a statement about the INPUT. A station with no tangency
whose azimuths still do not balance is a statement about this REDUCTION. They arrived as one name
(`DeclineTorusLaneUnaccounted`), so a user whose surfaces graze was told the kernel's loops did not add
up, and an engineer debugging a real census disagreement could not tell it from a tangency.

`DeclineTorusTangentStation` is the input's name. The classification is exact where it is made — the
station polynomial vanishes at one of its own extrema, tested with the residual certificate
`azimuths()` already applies to a candidate root, against the polynomial's own coefficient scale, so it
carries no model scale and needs no tolerance of its own. Finding the degenerate station is still a
sampling question; classifying it once a probe lands on it is not.

## What this deletes, and why deleting it is safe

### `DeclineTorusLaneFullTurn`

Deleted, not renamed. Every input that reached it now reaches a build or another named refusal, never
silence. The argument is structural: `folded=false` from `periodicRootWindows` means the lane
discriminant is positive at every probe; the lane discriminant is the **minimum** of the two track
discriminants; so the upper track is positive at every probe too, `torusUpperTrackSweep` answers
`wraps=true`, and the lane either builds an arc or declines `Separation` — after which the census gates
the set.

Measured independently in review (`.superpowers/sdd/csg-leftovers-plan/L13-review-1.md`), base against
head, over **4000 random torus × cylinder pairs**: 942 rows changed, and every
one of them from `the torus section's branch pair never folds` to `ok=true`. Zero rows went the other
way, zero produced `ok=true` with an empty section, zero produced NaN. The other 3058 rows are
identical **including the sampled-point checksum**, so the 1252 previously built sections reproduce
bit-for-bit. Independently: 2194 sections built over that sweep, 160 351 sampled points, worst distance
to *both* surfaces 1.753e-13 — the new full-period arcs are exact, not merely balanced.

### `periodicRootWindows`'s rise/fall parity branch

The window finder used to refuse a discriminant whose sampled rises and falls came back in unequal
numbers, calling it numerical noise. That branch was **unreachable**: the classification is taken from
a fixed sample array walked as a cycle, and on a cycle of two classes the up-transitions and the
down-transitions are equal in number whatever the samples are.

Three confirmations. The cycle argument. `TestFoldsAlternateOnEverySignPattern`, which drives all 4096
twelve-probe sign patterns through the classifier and requires the windows to come back well formed.
And a `panic()` planted in that branch **on the base commit**, under which base `kernel/geom`,
`kernel/brep` and all eleven `kernel/ops/*` packages pass without firing it (`go vet` on the
instrumented base reports the following `return` as unreachable code, which is the compiler agreeing).

The deletion is not housekeeping: that dead refusal is *what hid the wrap*. It made `folded=false` mean
two different things — "positive everywhere" and "unreadable" — so no caller could act on the first.
`folded=false` now means exactly one thing, and the surviving non-torus caller
(`ruledDiscriminantWindows`) reads it correctly.

## What is still refused

ADR-0061's G2 is retired. In its place, for the same reduction:

- **A tangent station** — the other surface touches the tube circle without crossing it, so a double root has
  no pair of branches to bound it. Guard: `TestAGrazingStationIsNamedATangency` (geom),
  `TestALaneConditioningDemotionIsReported` (ops/boolean),
  `TestAnIllConditionedLaneDeclinesByName` (brep), `TestTheDeclineSaysWhichGateRefused` (ops/boolean),
  all four now asserting `DeclineTorusTangentStation`'s own sentence rather than a generic one, with
  `CodeSectionConditioningDemotion`. Driven by a rod lying tangent to the top of the ring's tube, and
  by a ring grazing the inside of a rod at radius `major + minor`. Measured: over a sweep of rod radii
  (3.00 / 4.50 / 6.00 / 7.40 / 7.50 / 7.60) **only the exact tangency** at `major + minor` refuses;
  7.60 returns the honest empty section, because the rod then contains the ring.
- **Unreadable extremum tracks** (`DeclineTorusLaneTracks`), **a station with no azimuth dependence**
  (`DeclineTorusLaneStation`) and **branches under the stitch resolution**
  (`DeclineTorusLaneSeparation`) — unchanged, and `Separation` still fires on a sub-resolution rod.
- **An azimuth set this reduction cannot account for** (`DeclineTorusLaneUnaccounted`) — now narrowed
  to its internal meaning, with no tangency to explain it. Guard:
  `TestADroppedSectionLoopIsCaughtByTheAzimuthCount` and `TestFourArcsOnOneLaneAreRefused`.

G1, G3 and G4 of ADR-0061's table are untouched.

## Consequences

- One capability added, two named refusals net (`FullTurn` deleted, `TangentStation` added), one
  unreachable branch removed. `make fma-gate` is unmoved at `total 10, declared residual 10`, and
  `archguard`'s net-delta and CSG-debt ratchets are unmoved with no pin edited — `mixed-decline-returns`
  stays at 3, `diag-codes` at 35 (`SectionDecline` is not a `diag.Code`).
- Roughly a quarter of random torus × cylinder sections move from "declined, marched" to "exact
  analytic". `model/feature` (1297 s) and `model/feature/occtparity` (820 s) both pass unchanged, which
  is the check that matters for a change of that shape: no downstream tessellation or mass-properties
  golden moved.
- **Not decided here, and owed a ticket:** the two separation gates. `torusWrapConditioning` measures a
  *pair's* arccos span for the one-harmonic wrap; `torusArcClearanceAt` measures a *single branch's*
  clearance from its nearest neighbour for a lane wrap. They are not an ordered try-list — one
  classification selects exactly one family and each family has one certificate for the construction it
  builds — but they are one concept ("can the stitch tell these branches apart") in two formulas.
  Unifying on the nearest-other-root form is strictly tighter than the arccos form wherever the level is
  positive, so it could newly decline axis-invariant pairs that build today, and it needs a corpus of
  axis-invariant wraps proving it declines none of them. That is its own change, with its own evidence.
