# Exact grounded sketch coordinates — partial R3

Tracking: [#3762](https://github.com/Oblikovati/Oblikovati/issues/3762), a
prerequisite for [owned point bindings](https://github.com/Oblikovati/Oblikovati/issues/3761).

`Sketch2D::solve_with_grounded_points` and `analyze_with_grounded_points` accept
current coordinates for owned point identities. The caller resolves those values
from the source recipe. The functions validate every identity, finite coordinate,
and duplicate before compiling the constraint system. The input sketch remains
unchanged. No source definition or constraint identity is created here.

Grounded coordinates are constants. Their two columns are removed from the
numerical variable universe. Each existing residual receives its usual input
shape with those constants substituted before analytic circle/arc projection.
Repeated point references keep their original column identity. Independent circle
radii and ungrounded points remain active. Arc circularity and all authored
relations keep their existing diagnostic source IDs.

The current coordinates enter the preparation seed before contact or angular
branch selection. The existing nonlinear solver, residual equations, automatic
derivatives, convergence checks, and rank analysis remain the shared pipeline.
Ordinary sketches call this same preparation with no grounded points; their
residuals use the original input path without an extra expansion allocation.
The unused former compiled-block builder is removed.

Candidate reconstruction combines accepted active values with the exact grounded
coordinate bits. A source coordinate cannot move during a numerical iteration,
including a conflict or cancellation. Signed zero, subnormal and extreme finite
coordinates survive reconstruction. Fully grounded conflicting relations still
report their authored constraint IDs. An arithmetic failure retains the existing
source error instead of returning an invalid result.

Tests cover zero-variable references, active DOF/rank, current dimensions,
independent free points, fully grounded conflicts, repeated references, circle
centres, both arc solution branches, cancellation, malformed inputs, arithmetic
overflow, exact coordinate bits, and unchanged authored definitions. Existing
ordinary-sketch corpus tests also exercise the shared path.

This numerical input seam is ready for the
[current datum projection query](implementation-sketch-point-projection.md).
It now serves [owned bindings and session recompute](implementation-associative-point-projections.md).
[Creation and origin preferences](implementation-sketch-creation-api.md) now use this path.
This step does not establish native application or kernel acceptance.
