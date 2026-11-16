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
anchors. Non-point identities are explicitly unknown. For coincident, coordinate,
fixed, grounded, midpoint, distance, and scalar length equations, each row
records the indices it reads and its prepared
numeric target with the authored parameter identity where applicable. Other
rows and projected transformations explicitly remain unattributed.

These binary64 values are observations, not enclosures of input uncertainty or
error bounds. They neither establish conditioning nor qualify any operation.
A future interval evaluator must bind the missing parameter targets, operand
provenance, and projection uncertainty to the same solve before certification.

## Conditional interval equation evaluation

The sketch model can evaluate coincident, coordinate, fixed, grounded,
midpoint, and horizontal or vertical distance equations over a finite box in
full solver-variable order. Dimensional targets require caller-supplied bounds
that name the relation and parameter and contain the prepared target. The
caller is responsible for proving that these bounds enclose the true target.
Absolute-distance boxes that touch or cross zero are refused. Fixed and grounded
targets are treated as their stored binary64 values; this does not enclose
uncertainty from external sources. The point solver and interval path share
those expressions. Results retain the source relation, equation number,
residual enclosure, and a derivative enclosure for each input variable. Other
relations, including aligned distances, projected equation layouts, malformed
boxes, and nonfinite results are refused. A caller-supplied box is not bound to
a solve or to projected input uncertainty and does not imply a sketch
certificate.

The project-owned arithmetic boundary uses pinned `maryada` 0.2.0 interval
subtraction, addition, and multiplication by one half. Exact binary-rational
tests cover representative endpoints, subnormal values, and overflow refusal
on the test platform. This is not a
cross-platform proof of the dependency's rounding contract. Qualification
also needs validated input boxes, complete residual and Jacobian coverage,
conditioning, existence and uniqueness checks, and a policy shared by every
consumer. The interval evaluator does not enable that policy.

## Conditional root inclusion

A retained converged solve can be checked with a caller-provided finite box,
target bounds, and a finite square preconditioner. The check verifies that the
candidate values, point identities, relation inputs, and prepared targets match
the retained solve. Externally grounded inputs, unsupported equations,
non-square systems, and boxes that do not contain the accepted candidate are
refused.

For the supported continuously differentiable equations, the check evaluates
`K = x0 - C f(x0, P) + (I - C J(X, P))(X - x0)` with interval arithmetic. It
requires strict inclusion of `K` in `X` and an upper infinity-norm bound below
one for `I - C J(X, P)`. The result retains the accepted sketch and solve evidence, box, target
assumptions, preconditioner, enclosure, and contraction bound. It does not
carry a document identity or prove that a later document still matches it.
Subject to the interval arithmetic contract and the supplied target assumptions, this is
conditional root evidence for each target in `P`.

It does not establish source or projection uncertainty, cross-platform
arithmetic conformance, general constraint coverage, or feature tolerance
qualification. It does not change profile discovery or extrusion admission.

## Authored parameter bounds

A conditional source-bound path supports healthy nominal model or user length
parameters with decimal literals in `mm`, `cm`, or `m`, unitless literals,
references, parentheses, negation, and the four arithmetic operators. It uses
the model expression parser without constant folding, checks each reference
against its bound identity, and retains source identities and text. It requires
the default centimetre working scale. It encloses authored decimals, unit
factors and arithmetic, and includes each parameter's rounded binary64 value.
Division by a box containing zero and unsupported functions decline. Independent
rational checks cover nested arithmetic; a cancellation test checks that an
exactly zero divisor is refused even if binary64 evaluation is nonzero. A
canonical projected-origin rectangle also retains the formula source through
its conditional root check and rejects a later same-value source edit. These
checks do not authenticate interval arithmetic across platforms. Fixed and grounded coordinate
targets additionally require explicit caller-supplied exact-value assumptions;
their stored binary64 values are not treated as proven physical inputs. Other
units, scales, engineering-value selections, failed sources, and cycles decline.

The source-bound root check compares the current parameter graph and input
state with a snapshot captured by the accepted solve. A source edit therefore
invalidates the check even if the rounded target remains equal. This is a
solve-input check, not a document lifetime guarantee: document sketch health
currently retains diagnostics rather than the full solve outcome. Projected
anchors also remain unsupported; their source geometry and placement
uncertainty are not yet bounded. Neither path enables feature admission.

## Conditional origin projection

A current document snapshot can produce a conditional bound for an origin
projection on a fixed sketch frame. The caller supplies bounds for the resolved
frame origin and axes; the operation checks that they contain the resolved
binary64 frame, then encloses the analytic origin projection and the current
rounded result. The bounds are assumptions, not bounds proven from the frame's
source. Feature-point projections, derived frames and derived orientations
decline.

The evidence borrows the immutable snapshot and checks current sketch health,
accepted solve diagnostics and geometry, the solve-time frame, and the retained
parameter-source stamp. This catches changed projected inputs and same-value
expression edits. It does not prove a document or session identity across
snapshots. For a single origin anchor, the conditional root check retains the
original full-to-active variable mapping. It checks the original reduced equation count,
then appends two full-coordinate anchor equations using the projected bounds.
The anchor source identity and bounds remain available with the result. A
non-square or singular augmented system declines. This equivalence holds under
the supplied frame bounds; it does not validate those assumptions or enable
feature admission.

## Canonical origin-plane source

New sketches created on an origin plane retain `XY`, `XZ`, or `YZ` as an
explicit placement identity in sketch recipe version 13. The matching canonical
frame supplies exact model-origin and unit-axis bounds for conditional origin
projection. An orientation override declines this path. Older fixed frames
remain fixed; equal coordinates do not establish origin-plane provenance.

The conditional result still borrows one immutable document snapshot and does
not establish session identity across snapshots or validate cross-platform
interval arithmetic. It does not enable feature admission. Rust's native
planar-sketch section uses document schema 3. The Go host tested with this
change supports schema 2 and rejects schema 3 before model restoration.

## Conditional prism displacement sources

A session-borrowed sketch result can inspect an authored prism for the same
sketch and retain conditional bounds for its three local displacement sources.
Each parameter uses the authored formula bound and retains its parameter and
source identities. Each stored constant requires explicit caller-supplied
finite bounds containing the resolved value. The normal interval must exclude
zero. The evidence cannot outlive the session borrow, and the session tests
rederive it after edits, undo, redo, save and reopen.

These bounds remain conditional on arithmetic conformance and the constant
assumptions. They do not bound a derived frame, prove a topology mapping, or
establish a tolerance budget. They do not discharge the `ExtrusionSource`
obligation, authorize preview or creation, or qualify an existing result.

## Conditional swept-point bounds

A canonical-plane sketch root and the same session's prism displacement
source bounds can be combined into a conditional bound for each retained sketch
point over sweep parameter values from zero through one. The upper bound uses
the point displacement bound plus the sum of the three displacement-component
errors. Directed interval arithmetic encloses the differences and sum. This
conservative L1 bound does not assume correlations between the sketch and
parameter errors. Exact-rational checks cover source-box corners and sweep
parameters zero, one half and one.

The source constants remain caller assumptions and the parameter enclosures
remain conditional on the arithmetic backend. The result is a mathematical
input bound, not a bound on kernel operations or the final B-rep. It does not
prove the coincidence mapping, kernel tolerance composition, output validity,
or admission coverage. The existing prism obligations remain unchanged.

## Conditional profile input mapping

The host adapter can now attribute the complete converted line-profile input
back to the current conditional sketch root. It calls the same converter as
region discovery, retaining its line and vertex identities and its transitive
coincidence source groups. Each converted vertex receives the maximum of its
source points' conditional displacement bounds. Construction curves are
excluded by that converter; other unselected normal lines remain in the input.
Curved inputs decline because their complete source bounds are not yet covered.

The mapping borrows the root evidence. It does not prove that the input forms
an admissible region, authenticate the interval backend, establish a tolerance
budget, or qualify a feature. Exact coincidence of the true root and the
kernel's tolerance composition remain separate obligations (#3928 and
Martensite #494).
