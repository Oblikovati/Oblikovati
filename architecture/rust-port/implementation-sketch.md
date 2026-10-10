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
