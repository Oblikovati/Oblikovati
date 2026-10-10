# Planar sketch model — partial R3

Tracking: [#3704](https://github.com/Oblikovati/Oblikovati/issues/3704), under
[#3563](https://github.com/Oblikovati/Oblikovati/issues/3563) and
[ADR-0006](../decisions/ADR-0006-parameters-and-constraints.md).

## Geometry and identity

`model::sketch::Sketch2D` owns finite planar points, analytic line endpoint
references, and authored constraints in insertion order. Entity IDs are nonzero
and unique across points and lines; constraint IDs use a separate namespace.
The document supplies identities. Solver order never allocates or replaces them.
Two lines that share a point ID use the same coordinates; structural coincidence
needs no extra equation. Unused points still contribute two DOF, as in Go's full
variable universe. Lines contribute no independent coordinates.

Add/move operations validate before mutation. Missing point/line references,
identity collisions and non-finite coordinates return errors naming their values.
The first layer permits coincident line endpoints as authored geometry; it does
not claim that such a line is a valid profile edge. Profile classification and
kernel tolerance integration remain separate work.

## Constraints and dimensions

The initial residual library includes point coincidence, horizontal and vertical
point pairs, fixed coordinates, a point at a line midpoint, and aligned/horizontal/
vertical two-point distances. All derivatives use the shared dual-number layer.
Repeated point references map to the same global columns. Midpoint coordinates
are halved before addition to avoid avoidable overflow.

Dimensions retain a parameter ID, not a captured scalar or copied expression.
Each solve reads `ParameterCollection::model_value`, including its selected
engineering tolerance value, and requires a healthy finite nonnegative length.
Values are already in document working units; no second conversion occurs here.
Unknown, failed, nonnumeric, wrong-dimension or negative targets fail with the
constraint and parameter named. Dependencies report direct dimension references
in first-use order; the parameter DAG retains transitive formula dependencies.

Aligned distances use hypot; horizontal and vertical distances use absolute
separation, matching Go. Zero separation retains the shared zero-gradient
convention and can stall when asked to reach a positive distance. It is reported
as unsatisfied, never silently perturbed. Signed branch persistence is exercised
from negative coordinate warm starts.

## Solve boundary

`Sketch2D::solve` compiles the complete coordinate universe and typed residual
blocks into the [shared numerical core](implementation-solver.md). It returns a
`SketchSolveOutcome` with a candidate sketch, the final numerical evaluation,
termination and structure, and parameter dependency IDs. The original sketch
and parameter collection remain unchanged. Candidates retain authored identities,
line references and constraint definitions even when iteration changes coordinates.

Convergence and DOF remain separate. Nonconvergence and cancellation return the
last accepted candidate, which is not automatically adopted into a live document.
Conflicts map stored source indices back to constraint IDs. Redundant-constraint
queries reuse the captured Jacobian and identify alternative individual removals.
`analyze` reports local rank/DOF without iteration or geometry mutation. Document
health, transaction adoption and API routing still need integration.

## Evidence and remaining scope

A parameter-driven rectangle has eight point coordinates and eight independent
equations: one fixed corner, two horizontal and two vertical sides, width and
height dimensions. `width = 50 mm`, `height = width / 2` solves to 5 by 2.5 in the
default centimetre working units with rank 8, DOF 0 and no redundancy. Editing
width to 10 cm recomputes a 10 by 5 rectangle while preserving all references.
Selecting an upper engineering width drives 5.5 while the dependent height keeps
its nominal expression result, as required by parameter semantics.

Tests also cover free points, shared endpoints, repeated local references,
midpoints, structural self-coincidence, duplicate and inconsistent constraints,
identity/reference validation, bad parameter targets, cancellation, zero-length
failure, unchanged inputs, and deterministic repeated solves. Eight captured Go
residual/Jacobian fixtures cover every formula, with separate scalar central-
difference checks and exact expected dimensions. `go-planar-residuals.tsv` comes
from baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a`. Selected Go sketch solver,
dimension-edit, free-DOF and horizontal-distance tests pass as reference evidence.

This is the first planar model layer, not the complete R3 acceptance path. Planes,
circles/arcs/conics/splines, remaining constraints, projection, recipes, dimension
creation/deletion and driven dimensions, edit/drag semantics, decomposition,
mobility, sketch persistence, profiles, feature recompute, API/session adoption
and UI rendering remain to port. The in-memory rectangle does not prove save/
reopen identity, SolveSpace model parity, a kernel profile, or a rendered feature.

## Normalized line and incidence relations

Tracking: [#3705](https://github.com/Oblikovati/Oblikovati/issues/3705).
`PlanarConstraint::Lines` binds two existing line IDs to parallel, perpendicular,
collinear or equal-length relations. `PointOnLine` binds a point to an infinite
line, without segment clamping. Reference expansion preserves endpoint identity
and repeats global columns when the lines share a point. Collinearity contributes
both an orientation equation and a signed perpendicular-distance equation.

Parallel and perpendicular residuals use cross and dot products of individually
normalized directions. Incidence uses a unit direction crossed with the anchor
relative offset; equal length uses hypot-based lengths. No product of two raw
lengths is formed for normalization. The formulas share the existing derivative
and iteration path and do not introduce a second solver or topology predicate.

Unlike Go's epsilon-triggered raw cross/dot and anchor-distance fallbacks, a zero
line direction is refused for orientation and incidence. Such a direction has
no defined line orientation. The error retains its typed source and names the
authored constraint ID. Initial source errors in both solve and analysis now map
source block indices to that identity. Numerical failures in a trial remain
bounded rejections through the shared core; they do not overwrite accepted state.
Equal-length relations still permit zero lengths under the existing derivative
convention, with convergence checked independently.

Five committed Go fixtures compare residuals and full Jacobians on supported
nondegenerate domains. Separate scalar formulas provide central-difference checks.
Residual and derivative scale tests cover `1e-200` through `1e200`; complete line
solve and rank tests cover `1e-100` through `1e100`. Model tests reach DOF zero for
parallel/perpendicular lines with a driving length, collinear lines with two
length parameters, and perpendicular equal-length lines. They also cover shared
endpoints, structural self-parallelism, named conflicts, atomic missing-reference
errors, collapsed directions and unchanged original geometry. Selected Go
incidence, short-line, shared-vertex and near-parallel tests pass as reference
checks. The fixture baseline remains
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

These numerical relations do not prove profile topology, sketch persistence,
SolveSpace parity, API/session integration or kernel operation acceptance.

## Analytic circles and radius variables

Tracking: [#3706](https://github.com/Oblikovati/Oblikovati/issues/3706).
`SketchCircle2D` keeps its own entity ID, an owned center point ID and a positive
finite radius in working length units. Circles share the point/line entity
namespace and preserve center identity across candidate solves. Each radius adds
one independent variable after all point coordinates, including when no constraint
uses it. Multiple circles can share one center without duplicate coordinates.

Typed `SketchVariable` references now distinguish point coordinates from circle
radii. Constraint compilation resolves both into one full variable universe;
candidate adoption uses that same layout and checks its shape and circle radius
postconditions. This replaces the assumption that every variable comes in a
coordinate pair. It does not add a second numerical solver.

The model supports point-on-circle, concentric centers, equal radii and a radius
dimension backed by an existing healthy length parameter. Concentric equations
reuse point coincidence. Equal-radius and radius equations reference only the
radii they read. Incidence uses the analytic center distance, not sampled edges.
Radius dimensions retain their parameter IDs and selected engineering values;
zero radius targets are refused explicitly. Radius edits, source evaluation and
returned candidates all preserve positive finite radii. Nonpositive trial radii
are named domain rejections through bounded damping, never negative geometry
silently published as a result. Zero distances retain the documented AD
convention; failure is independent from the local DOF result.

A fixed center and driving radius solve with rank 3 and DOF 0. Editing a 5 cm
radius parameter to 8 cm recomputes that circle while preserving its IDs and
constraints. Concentric/equal-radius circles solve a six-variable system with
rank 6; shared centers leave each radius independent. Tests cover analytic point
incidence, free radii, structural concentric redundancy, invalid IDs/references,
nonpositive and failed targets, cancellation, immutable inputs and candidate
shape/domain errors. A point forced across its center rejects a negative-radius
trial and later converges to a valid positive-radius circle.

Four Go residual/Jacobian fixtures cover incidence, concentricity, equal radii
and a radius dimension. The radius-only equations project Go's unused center
columns out of the comparison; the full Rust DOF universe still includes them.
Independent scalar central differences check every derivative. Equation values
and partials survive unit sweeps `1e-200` through `1e200`; model solve/rank tests
cover `1e-100` through `1e100`. Selected Go circular incidence and radius tests
pass as reference evidence at the same recorded baseline.

Circular arcs, implicit arc circularity, line/circle and circle/circle tangency,
conic and spline geometry, circle profile topology, persistence and API/UI
integration remain to port. An in-memory analytic circle is not evidence of a
kernel profile, a saved/reopened sketch or a rendered feature.

An added free-radius invariant exposed QR roundoff in an unused variable. The
shared [step correction](implementation-solver.md#unused-variable-step-invariant)
eliminates zero working-Jacobian columns before factorization and preserves
free values bit-for-bit. The circle regression and the reduced solver regression
both pass; no circle-specific radius repair was added.

## Full-circle tangency with retained contact branches

Tracking: [#3708](https://github.com/Oblikovati/Oblikovati/issues/3708).
Line/circle tangency records `LineContactSide::Left` or `Right` relative to the
line's A-to-B direction. The residual is that signed perpendicular distance,
multiplied by the recorded side, minus the radius. It shares normalized direction
and positive-radius checks. Circle/circle tangency records external contact or
which circle encloses the other, then equates center distance to the radius sum
or positive radius difference. All equations use the existing typed-variable,
AD, candidate and bounded-iteration boundaries.

`add_line_tangent` selects the seed's side once. `add_circle_tangent` selects the
nearer radius-sum or radius-difference target, matching Go's creation-time rule;
external contact wins exact ties. The authored constraint retains its choice
after edits and solve. Internal contact also retains the enclosing circle, so
crossing radius order is a domain rejection rather than a containment switch.
Line side selection reads the signed distance directly, avoiding cancellation
with a large radius. Explicit typed constraints permit a caller to choose a
side or contact mode rather than infer it from a seed.

Collapsed lines, coincident circle centers, ambiguous zero-offset side selection,
and zero/negative internal gaps return typed errors naming the constraint and
expected domain. These replace degenerate Go behaviors that can converge to
coincident circles or lose a contact branch. Failed/cancelled solves do not adopt
candidate geometry into the original sketch. Internal gap failures during trials
use the shared damping rejection policy; no point nudge or alternate solver is
introduced.

Five Go fixtures compare ordinary-domain residuals and full Jacobians for left,
right, external and both enclosing-circle contacts. Circle columns are reordered
to the Rust coordinate-first/radius-last layout. Independent scalar central
finite differences check every partial; value/derivative unit sweeps cover
`1e-200` through `1e200`. Model tests solve parameter-driven DOF-zero tangencies,
retain sides after radius edits or seed crossing, preserve enclosing choices
after incompatible edits, exercise exact branch-selection ties, reject bad
references/domains atomically and preserve cancellation. Complete circular tangent
solve/rank sweeps cover `1e-100` through `1e100`. Selected Go full-circle contact
and free-touch-point rank tests pass at the recorded baseline.

Arc tangency and the shared defining-touch-point formulations require the arc
model and remain pending. These full-circle results do not establish profile
stitching, contact naming, persistence, API/UI integration or kernel acceptance.

## Circular arc ownership and structural circularity

Tracking: [#3709](https://github.com/Oblikovati/Oblikovati/issues/3709).
`SketchArc2D` owns a stable arc identity and references the sketch's center,
start, and end point identities. It retains the authored clockwise or
counterclockwise direction. Radius is derived from center/start; there is no
independent radius variable. `ArcRadius` reads a healthy, strictly positive
length parameter and reuses the aligned-distance equation. A noncircular draft
is allowed so that the solver can restore circularity.

Every owned arc contributes one structural equation:
`distance(center, end) - distance(center, start) = 0`. This ports Go's
`arc_circularity.go`, including the endpoint-following requirement of #1419.
Structural equations are compiled after authored constraints in stable arc
insertion order. They are derived from ownership and cannot be deleted through
the authored constraint collection. They share the existing AD evaluation,
variable mapping, nonlinear iteration, and diagnostics. Both endpoint radii
must be positive during evaluation; collapsed trial geometry is rejected rather
than moved or repaired. A collapsed draft can be edited but cannot be solved
as a valid arc.

`SketchRelationId` identifies either an authored constraint or an arc-owned
circularity equation. Complete conflict and individual redundancy reports use
these identities; the existing authored-only views explicitly exclude structural
sources. Evaluation failures name the owning arc and retain their typed cause.
A structural redundancy report is evidence about rank, not permission to delete
an arc invariant. Shared endpoint occurrences seed the same global variables;
coincident start/end references cancel their circularity row without adding
coordinates. Such draft geometry does not prove a valid profile sweep.

Tests drive both sweep directions from unequal endpoint radii to parameterized
DOF-zero arcs, edit radius again, and verify both endpoints, identities,
constraints, and parameter dependencies. Tests cover ownership validation,
cancellation, zero-radius parameters, collapsed radii, overflow, repeated
columns, two arcs sharing points, structural conflicts and redundancy, and
solve/rank scales from `1e-100` through `1e100`. Independent analytic and central
finite-difference derivatives cover every defining coordinate; residual unit
sweeps cover `1e-200` through `1e200`. Three residual/full-Jacobian fixtures match
Go at the fixed source baseline. Selected Go arc creation, circularity repair,
and radius-dimension regressions pass there.

Arc contact and midpoint relations, diameter dimensions, sketch persistence,
API/UI integration, sweep validity, profile stitching, and kernel acceptance
remain open. A converged sketch is numerical evidence and does not close any
analytic B-rep or feature validation gate.

## Shared circular frames and contact-point tangency

Tracking: [#3710](https://github.com/Oblikovati/Oblikovati/issues/3710).
The authored circular relations now reference either an owned circle or an arc
through one frame boundary. `Radius { curve, parameter }` replaces the separate
`ArcRadius` form. `PointOnCircularCurve`, `Concentric`, `EqualRadius`,
`LineTangent`, and `CircularTangent` share the same boundary. Incidence means
incidence on the supporting analytic circle; it does not restrict a point to
an arc's finite sweep. The previous circle-only authored names and variable
compiler are removed, rather than kept as a second implementation.

A circular frame exposes center coordinates and either the circle's independent
radius scalar or the arc's center/start-derived radius. Compilation records
one local projection into the existing canonical AD equations. Global variable
identity is preserved through projection, including repeated shared points.
Circle-only equations perform the same arithmetic as before: tests compare
residual and complete global derivative bit patterns for all six equation
families at three coordinate scales. The projection changes no geometric
formula and adds no numerical solver.

Shared defining contact is classified by point identity, independently of
placement or iteration. A full circle has no defining contact point. A line
and arc can share one endpoint; two arcs can share one endpoint. Two distinct
shared points are a named decline before a relation is added. Duplicate
occurrences of one point count as one identity; there is no first-match policy.

For a shared line/arc point, the equation is the unit line direction dotted
with center-minus-touch. For a shared arc/arc point, it is the cross product
of the unit first-center-to-touch direction and second-center-minus-touch.
These port the simple-root formulations from Go #2014. The distance-minus-radius
forms have stationary roots at structurally shared contact and would report
zero rank despite exact tangency. The selected formulation remains fixed
throughout solving because the defining identities remain fixed.

Line contact retains the recorded center side. Shared circular contact also
retains external/opposed or internal/aligned radial orientation and, for an
internal contact, which curve encloses the other. Creation classifies the
shared radial half-plane; free contact continues to select the nearer distance
target, with external contact winning exact ties. Trial evaluations reject an
incompatible side, radial half-plane, or enclosing-radius order. This strengthens
Go's shared-point equation, which accepts either collinear orientation without
certifying a retained contact branch. Collapsed directions and nonpositive
radii are named domain errors. No geometry is nudged or repaired.

Seven Go residual/full-Jacobian fixtures cover derived-radius incidence,
equal radius and dimensions, free line/arc and mixed circle/arc contact, and
both shared-point formulations. Independent scalar finite differences check
every raw defining coordinate. Residual unit and derivative sweeps cover
`1e-200` through `1e200`; complete mixed/free and shared-line solve/rank sweeps
cover `1e-100` through `1e100`. Go's shared line/arc and arc/arc rank regressions,
free-circle rank regression, and off-tangent solve regression pass at the fixed
baseline.

Application model tests cover mixed concentric/equal-radius/incidence equations,
parameter edits with zero DOF for free contact, external and both internal
orders, and every shared line/arc endpoint combination. Shared contact adds
exactly one rank at a tangent solution and converges from off-tangent seeds.
Equal coordinates with distinct point identities retain free contact and still
solve with zero DOF after a radius edit. Tests retain curve, point, constraint, sweep, and contact identities, reject
multiple shared points atomically, and refuse an incompatible edit that would
change internal contact to external contact. Authored radius failures now name
the authored relation; a standalone arc circularity failure still names its
structural arc source.

Arc midpoint and diameter relations, sketch persistence, API/UI integration,
finite sweep/profile validity, feature evaluation, and kernel acceptance remain
open. These contact equations describe sketch geometry and do not certify a
valid B-rep contact, blend, or profile.

## Directed arc midpoint queries and constraints

Tracking: [#3711](https://github.com/Oblikovati/Oblikovati/issues/3711).
`arc_midpoint` and `ArcMidpoint { point, arc }` use one AD construction. The
start and end radial directions determine a relative angle by normalized
cross/dot and `atan2`. The authored sweep direction selects the directed angle;
rotating the start radial vector by half that angle gives the midpoint. This
handles minor and major sweeps and avoids a discontinuity when an absolute
polar angle crosses its wrap boundary. Radius remains derived from center/start.
The query is read-only and uses the same construction on constant AD values.

The constraint has two direct coordinate residuals: point minus midpoint. Go's
`constraints_arc_midpoint.go` instead constrains circle incidence and the
perpendicular bisector, then seeds the point to select one of the two roots.
The Rust equations exclude the complementary root explicitly and do not move
an authored point when a relation is added. They still associate the point with
the arc as radius, endpoint directions, and sweep change. These are different
residual coordinates, so matching Go's midpoint positions is meaningful;
claiming identical Go residuals or Jacobians would not be meaningful.

Collapsed radial directions are named declines. Identical endpoint directions
are also refused: they do not distinguish zero sweep from full sweep. The Go
query and seed helpers disagree on this case, choosing full and zero sweeps
respectively. A full circle remains a separate entity. No epsilon perturbation,
implicit branch retry, or alternate midpoint engine resolves the ambiguity.
The query follows endpoint directions on a draft; it does not prove circularity
or finite-sweep profile validity. Solving still includes the arc's structural
circularity equation.

Six fixed-baseline Go query fixtures cover quarter, major, both half-circle
orientations, polar wrap, and a translated center. Go's midpoint seed/solve and
clockwise regressions pass. Independent polar-angle formulas and central finite
differences check both Rust residual rows and all eight coordinate partials.
Tests cover both wrap boundaries, near-zero/near-full sweeps, collapsed and
ambiguous domains, and residual/derivative scales from `1e-200` to `1e200`.

Model tests solve both sweep directions from the complementary side with zero
DOF, edit radius and endpoint direction, and preserve entity and constraint
identities. They check read-only queries, atomic reference errors, typed query
and authored-constraint failures, overflow, cancellation, and full solve/rank
scales from `1e-100` through `1e100`. Diameter dimensions, sketch persistence,
API/UI integration, profile stitching and kernel feature acceptance remain open.
