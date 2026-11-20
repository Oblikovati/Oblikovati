# Sketch drag preference certification

Tracking: [#4171](https://github.com/Oblikovati/Oblikovati/issues/4171).

The current preference result does not certify a constrained minimum. A zero
projected tangent step shows only that a first-order search found no direction.
On the unit circle, the point `(1, 0)` is stationary for the preference
`x = -1`, but it is the farthest feasible point from that target. A singular
constraint can also have a zero Jacobian at a constrained point. The host must
therefore decline `Unavailable`, `ZeroProjectedStep`, and `Stalled` rather than
adopt their candidate as a certified drag.

The solver now also reports `ZeroProjectedStep` when its computed step is
exactly zero but the augmented linearized system has a numerical free
direction. `Unavailable` covers a zero step with no numerical free direction
and model-level blocked targets. The rank includes fixed coordinates and uses
the configured numerical tolerance. Neither result proves that a nonlinear
minimum exists, that it is unique, or that a useful drag is impossible. The
circle counterexample and a constrained coordinate with an unrelated free
coordinate exercise the distinction. Partial drag remains disabled until an
appropriate optimality certificate is available.

The residual interface currently accepts arbitrary Rust in `evaluate`, including
branches that inspect scalar values. Two residual evaluations can have identical
Jacobians while their values do not follow the same affine equation. Samples of
the current Jacobian do not certify the residual's global degree or domain.

One possible general extension is a typed expression representation that carries
its algebra and domain predicates, with no untracked value-based branches.
Affine expressions permit a rank-aware constrained least-squares certificate.
For nonlinear expressions, a local certificate needs a regularity check,
first-order conditions, and positive constrained second-order evidence. A local
certificate does not prove the global closest point. Singular, nonsmooth, and
numerically indeterminate cases need explicit outcomes, not a success fallback.
Evaluation and certificate costs must be measured on realistic sketches.

A synthetic release-mode probe of dense exact-rational KKT elimination took
20.416 ms median for 32 variables and 129.027 ms for 64 variables on the
recorded machine. It used a simple equality chain and three samples per size;
it did not run the application or compare application memory. This rules out
assuming that this prototype meets a 16.67 ms frame budget at those sizes.
Use a scalable method and measure real sketch interaction before enabling it.
The input, output, environment and limitations are in
`receipts/2026-11-20-affine-cost-4171/`.

A test-only variant skips exact zero factors and coefficients during the same
elimination. It matched the original result, feasibility, and KKT stationarity
on 240 small deterministic systems. On a 64-variable equality chain, median
time fell from 133.786 ms to 8.384 ms in one run. A denser 32-variable case
took 23.153 ms in a single sample. These are synthetic measurements, not a
frame-rate or application-memory result. The method still lacks sketch input
bounds, nonlinear certification, and an interactive workload test. See
`receipts/2026-11-20-affine-zero-skip-4171/`.

The solver now has a bounded certificate for explicitly supplied affine
equations whose binary64 inputs are treated as exact binary rationals. It
checks feasibility and KKT stationarity with exact arithmetic. The squared
coordinate-preference objective is convex, so these conditions establish a
global best fit for those supplied equations. Free coordinates can be
non-unique. The returned binary64 bounds enclose an exact optimizer; a rounded
coordinate is not asserted to satisfy the equations exactly.

This does not establish that the supplied equations represent the sketch,
bound parameter-source uncertainty, certify nonlinear geometry, or satisfy
the sketch tolerance policy after rounding. It is not connected to dragging.
The bounded solver has explicit cancellation and resource refusals. A release
probe of this component took 24.027 ms median on a denser 32-variable example.
Real sketch and UI performance remain unmeasured. Test and probe details are
in `receipts/2026-11-20-affine-component-4171/`.

SolveSpace forces dragged points toward the pointer, then solves authored
constraints with weighted Newton iterations. Its success and rank checks do not
certify the closest feasible drag target. OCCT's `math_Uzawa` accepts an
explicit linear system for a minimum-distance problem; `math_NewtonMinimum`
uses supplied Hessians and distinguishes nonpositive curvature, but operates as
a local unconstrained optimizer. These are design references, not proofs for
the current residual interface or the counterexamples above.
