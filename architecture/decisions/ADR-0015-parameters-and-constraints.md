# ADR-0015: Parameters, units, and constraint solving

Date: 2026-10-11. Status: Accepted for the Rust target design.
Supersedes [ADR-0006](ADR-0006-parameters-and-constraints.md).
Implementation evidence: R2/R3 remains partial; native ellipse work is #3788.

Boundary: dimensional inputs and the shared numerical constraint solver.
Owner: model parameter/sketch/assembly modules and `oblikovati-solve`.

## Context

Sketch constraints and static assembly placement both solve residual systems.
They need common convergence and diagnostic behavior without coupling sketches
to B-rep storage. Unit changes must preserve the physical meaning of formulas,
solver residuals, and kernel inputs.

## Decision

Port one deterministic numerical solver for sketches and static assembly
placement. The model constructs typed variable blocks and residuals; the solver
returns solved values, convergence, rank/DOF, conflicting or redundant constraints,
and achieved precision. It owns no document, B-rep, window, or I/O state.
Do not assume that Martensite's OCCT port supplies this application solver.

Retain decomposition, warm starts, bounded damped iteration, and scale-aware
residuals. Port and measure the existing derivative path; analytic or automatic
derivatives need independent checks. Any retained finite-difference path must
state its error and supported domain. Graph structure alone does not prove
numerical rank near a singular configuration.

Sketches and assemblies define their own variable parameterizations. Assembly
placements use normalized orientations and explicit rigid-body degrees of
freedom; grounded components remove freedom. Joints restrict that same system
rather than introduce a second solver. Dynamic/contact simulation is separate
work, not implied by static placement support.

Keep stable parameter IDs, expressions, dimensions, and dependency footprints.
Reject cycles and dimensionally invalid formulas. Evaluate explicit-unit input
into the document's working unit; keep bare-number behavior from the baseline.
Do not replace parameter expressions with captured scalar values during save.

## Interfaces and invariants

- Default lengths are centimetres and angles are radians. Stored geometric
  quantities use the recorded working scale; display-unit changes alone do
  not rescale a populated document. Length, area, volume, and derived units
  convert with the correct exponents.
- New-document scale selection and later scale freezing preserve the baseline
  rules. Assembly placement converts child and owner scales explicitly.
  Exchange conversion happens at its input/output interface, not at arbitrary
  modeling call sites.
- Solver achieved precision feeds the kernel tolerance contract in
  [ADR-0007](ADR-0007-martensite-integration.md). Meshing quality cannot loosen
  constraints or determine profile topology.
- A solved sketch supplies analytic profiles/paths and health to features.
  DOF = 0 is required for the fully constrained acceptance designs; other
  user sketches can remain underconstrained with an accurate diagnosis.
- Warm starts must not silently jump to another solution branch. Non-convergence,
  singularity, inconsistent constraints, or an unresolvable scale produces a
  named result with the offending constraint or quantity.

## Alternatives and consequences

Separate solvers would duplicate rank analysis, iteration, and failure policy.
Embedding B-rep knowledge in the solver would prevent isolated numerical tests.
Reject both. One core still needs separate sketch and assembly residual tests;
sharing the iteration code does not prove equivalent correctness.

## Legacy disposition

Consolidate Oblikovati ADR-0009 and 0011 with the parameter/working-scale part
of 0042 at the [fixed revision](legacy-map.md#source-baseline). Replace Go
callbacks and goroutines with explicit Rust inputs and owned results. Do not
inherit an exact algorithm label or "diagnostics for free" claim where the
source and numerical tests do not prove it. Kernel predicates remain
Martensite-owned.

## Validation required

Use Go `solve/`, `solve/ad/`, `model/param/units_of_measure.go`, expression,
sketch, and assembly fixtures. Test rank/DOF and residuals independently,
redundancy/conflicts, near-singular drags, cancellation, and deterministic order.
Use analytic systems and SolveSpace where applicable. R3 must produce a
parameter-driven, DOF-zero sketch; R6 must position repeated components with
mixed document units and retain their intended branch after edit/reopen.

## Analytic ellipse representation

The model owns planar sketch ellipse values, shared center identity, and the
three shape variables: major radius, minor radius, and one orientation angle.
Axis components do not add two independent freedoms. This preserves the Go
rotation intent in `model/sketch/ellipse_orientation.go` (#1879) while avoiding
an axis-magnitude freedom. Solver orientation writeback preserves the old axis
when the angle did not change. B-rep ellipse construction stays in Martensite.

Use exact analytic parametric position and tangent, with a normalized nonzero
major-axis direction. Reject non-finite values, zero radii, and a major radius
below the minor radius before adoption. Do not swap radii or rotate the axis to
make a failed edit pass. OCCT's `Geom_Ellipse` orders the radii; Rust additionally
refuses collapsed sketch ellipses. Equal radii remain a valid analytic value.
Do not classify them as circles or discard their named orientation.

Integrate ellipses into the existing sketch ownership, recipe, variable layout,
residual, and dimension creation paths. Completion removes the `ellipseRadius`
missing-native-geometry decline; no second sketch solver replaces that decline.
The first validated analytic value is not proof of owner or solver completion.

Acceptance requires five ellipse freedoms including the shared center's two
coordinates, independent derivative/oracle checks, near-degenerate domains,
fully constrained examples, major-radius formula edits, reference dimensions,
permanent identity, and session/storage/API acceptance in #3788. Degenerate
axes and radius crossings return named errors instead of coordinate repairs.
