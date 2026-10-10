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
