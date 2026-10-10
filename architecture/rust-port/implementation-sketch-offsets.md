# Rust sketch offset dimensions

Tracking: [#3714](https://github.com/Oblikovati/Oblikovati/issues/3714).
This extends the application-owned sketch solver in ADR-0006. It does not add
kernel geometry, API DTOs, viewport commands, or sketch persistence.

## Authored measure and branch

`TangentDistance` references an owned line, circle or arc, and a length parameter.
The selected line side stays part of the constraint. Near distance is `d - R`;
far distance is `d + R`, where `d` is the nonnegative perpendicular center
distance on that side. A near distance can be negative when the line crosses
the supporting circle. Diameter display drives half the authored length.
The dimension uses the circular frame and AD projection used by other relations;
an arc radius remains derived from its center and start point.

The factory selects the side once. A zero-side seed requires an explicit side.
A shared defining endpoint also retains the sign of the center-to-contact
projection along the line. Zero projection selects the positive lean. Multiple
shared point identities are refused. A copied position with a different point
identity does not create a shared-contact relation.

## Shared-contact rank

At a shared tangent point, the metric distance row has a double root. When one
unique radius target is supplied by radius or diameter dimensions on the same
curve, the offset uses the equivalent signed angle between the line and the
contact-to-center radius. The angle is computed from `d / R` in `[0, 1]` and the
retained lean. The radius equations remain present with their original source
identities. Duplicate equal radius targets permit the same substitution;
contradictory targets retain the metric row and all conflicting radius sources.
No geometric point is moved to start the solve.

Without a radius driver, zero near offset uses the existing shared line-tangent
row. Other offsets use the signed metric row. A singular unfixed-radius seed
can stop with `Stalled`, unchanged input, and constraint health. That result is
not an accepted solved sketch. General nonlinear continuation beyond such seeds
remains work for the shared solver; no private retry or nudge is introduced.

## Evidence

Sixteen fixtures from Go commit `aa8297d1b0ac33c024f19e54821536d76e8a447a`
cover circles and arcs, both line sides, near/far distance and diameter display.
Residuals and every partial match after the explicit diameter factor of two.
Independent scalar measurements and central differences check all seven metric
variables. Shared angular rows have independent polar derivative checks at both
line endpoints and on both sides, including exact tangency.

Complete model tests drive circles, free arcs and shared-contact arcs, retain
curve/constraint identities, and verify zero DOF after parameter changes.
They cover negative lean, a diameter radius driver, cancellation, wrong units,
invalid shared distances, bad contact identity, underflow of half dimensions,
contradictory radius dimensions and the singular seed described above. Metric
AD scale checks cover `1e-200` through `1e200`; complete arc solve/rank checks
cover `1e-100` through `1e100`. These checks do not certify a finite arc profile,
B-rep validity, kernel tolerances or application feature acceptance.

## Point-to-line offsets

Tracking: [#3715](https://github.com/Oblikovati/Oblikovati/issues/3715).
`Offset` references an owned point, line, length parameter, retained line side,
and diameter-display option. It uses the signed analytic point-on-line measure
minus its target. Diameter display uses half the authored length. Zero is a valid
target; the signed row retains its derivative there. Go's absolute-value row
has a stationary zero and can prevent a later parameter edit from moving the
point. An explicit side permits creation at zero offset; the factory requires
a nonzero seed side. Geometry is not moved during creation.

Point-on-line, circular tangency and both offset forms now share the same
normalized signed-distance calculation. Its operand order is unchanged, and
existing residual fixtures still pass. Diameter and both offset forms share one
half-dimension validator, which refuses nonzero values whose half underflows.
A collapsed line is a named orientation error. Go's offset dimension changes
such a line into a point-distance measure; Rust retains the authored meaning
and refuses it instead. A point that shares a line endpoint remains the same
variable: the dimension adds no artificial independent coordinate or rank.

Four fixtures from the same Go baseline cover both sides and diameter display,
with residual/derivative comparison after the explicit factor of two. Independent
scalar metrics and central finite differences check every coordinate. A zero
row has explicit derivative assertions. Fully constrained model tests edit
formula-backed targets through zero and back to nonzero values, preserve
identities, and verify rank six and zero DOF. They check retained-side warm
starts, wrong dimensions, unhealthy/negative targets, half-dimension underflow,
bad references, atomic creation failures, shared endpoint health, collapsed-line
source identity and cancellation. Metric AD scale checks cover `1e-200` through
`1e200`; complete solve/rank checks cover `1e-100` through `1e100`.
This does not implement offset-curve construction, sketch persistence or UI/API
commands.
