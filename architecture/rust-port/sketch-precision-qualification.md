# Sketch precision qualification

Issue #3928 tracks qualification of solved sketches for profile extrusion.
The current implementation retains raw residual rows, source identities, convergence
scaling domains, and conservative physical-unit labels. These labels are necessary
inputs for qualification; they are not numerical error bounds and are not used to
admit extrusion. Unknown or mixed rows remain `Unknown`.

A qualification implementation must use the current solved sketch and its current
projection inputs. For each relevant constraint, it must retain source identities,
identify operand provenance and physical units, and determine whether conditioning
permits a conservative spatial or angular error bound. A local Jacobian or a raw
residual alone is not such a bound. Angular quantities need a certified geometric
conversion before comparison with a length tolerance. Degenerate, ambiguous,
unsupported, stale, failed, and unconverged inputs must produce named declines.

The qualified bounds must be checked using the tolerance class appropriate to each
operand's origin. The same evidence and policy must reach profile discovery,
preview, creation, edit, and ordered replay. Result validation, identity and
provenance checks remain required. No geometry may be moved to satisfy a bound.

A future gate must cover known and unknown constraint domains, short and long
operands, ill-conditioned and degenerate inputs, free and fully constrained
sketches, mixed-unit rows, projected inputs, edits and replay. Independent analytic
oracles must check the reported bounds. Only after that gate passes can the general
zero-DOF restriction be reconsidered for #3927.

## Integration boundary

The constraint evidence must remain bound to the accepted solve and resolved
projection inputs. The owner must provide one versioned, provenance-aware
policy to both region discovery and the ordered prism evaluator; preview,
creation, edit, and replay must make the same decision. A bound on the
exact difference of stored coordinate operands is only one input: it does not
bound uncertainty in the solved coordinates or qualify the whole sketch.

The current feature status cannot report incomplete evidence while retaining a
body: `Failed` blocks dependents, and `Healthy` would hide the gap. A stable
typed incomplete reason or structured evidence requires contract-first work in
Oblikovati.API and explicit host and UI mapping. The wire's existing warning
string alone does not define that contract. Existing results need a migration
policy that does not label unqualified evidence as certified. See #3928.

## Interval evaluation requirements

The current `ResidualSource` accepts point-valued `DualNumber` inputs. It does
not provide an interval enclosure for either the residual or the Jacobian over
an input box. The nonlinear solver's row values and point Jacobian therefore
cannot establish existence, uniqueness, or a spatial error bound. For example,
`f(x) = x` and `g(x) = x + 2^20*x*x` have the same value and derivative at
`x = 0`, but `g` has another exact binary root at `x = -2^-20`. Evaluating both
with `evaluate_residuals` at those inputs reproduces the distinction without a
floating-point tolerance.

A general interval route needs a separate constraint interface that encloses
both residuals and derivatives over a box, preserves source and operand
identity, and reports unsupported domains. It must use an outward-rounded
backend with validated arithmetic and transcendental enclosures on every
supported platform. In particular, the standard Rust documentation gives
`sin`, `cos`, `atan2`, and `hypot` unspecified precision; a fixed number of
extra ULPs around those point results is not a certificate. See the
[Rust f64 documentation](https://doc.rust-lang.org/stable/std/primitive.f64.html).

A Krawczyk or interval-Newton consumer then needs a verified preconditioner,
a strict inclusion test for the whole box, and explicit declines for singular
or poorly conditioned systems, unrepresentable intervals, nonsmooth branches,
undefined domains, and interval functions that lack a validated enclosure. A
square-system certificate cannot by itself certify an underconstrained sketch;
a choice of free-variable slice needs its own justified semantics. The
[cited Krawczyk reference](https://par.nsf.gov/servlets/purl/10128975)
describes the interval operator as a certificate rather than a point-residual
check. Independent interval or exact-arithmetic oracles and cross-platform
rounding tests are required before this can gate extrusion.

## Accepted operand evidence

Each raw residual row retains the prepared relation's source-level variable
operands in original solver-column order. Point coordinates carry their point
identity and axis; supplied grounded anchors are marked separately from active
solver variables. Values come from the accepted solve after restoring eliminated
anchors. Non-point identities are explicitly unknown. A relation can use only a
subset of its source operands, and embedded target values are not included.

These binary64 values are observations, not enclosures of input uncertainty or
error bounds. They neither establish conditioning nor qualify any operation.
A future interval evaluator must bind the missing parameter targets, operand
provenance, and projection uncertainty to the same solve before certification.
