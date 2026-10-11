# Rust implementation evidence

[Interactive selection engine evidence](implementation-interactive-selection.md).

## Parameter dependency graph — R2

Tracking: [#3561](https://github.com/Oblikovati/Oblikovati/issues/3561). Implementation: `crates/model/src/dependency`.
Go source: `model/param/graph.go` and `graph_test.go`, revision `aa8297d1b0ac33c024f19e54821536d76e8a447a`.

The model crate now supports caller-owned stable IDs, insertion-order edge
queries, atomic edge replacement, cycle rejection, protected removal, and
ordered recomputation of exactly the affected nodes. Equal-priority nodes use
insertion order. Traversal is iterative to support deep imported recipes.

Tests cover chains, diamonds, independent branches, duplicate edges, replacement,
unknown IDs, cycles, removal, cloned graph isolation, and a 10,000-node chain.
An independent reachability matrix checks all 64 forward-edge graphs on four
nodes and every single-driver replacement. Run `make gate` from the repo root.

The graph does not own parameter expressions, IDs, health, or feature evaluation.
The Go parameter collection retains a failed authored expression and marks its
parameter sick after a cycle. That owner-level behavior remains to be ported;
this graph rejects invalid edge edits without changing existing edges. The future
owner must explicitly detach failed bindings when required by the Go behavior.
Owner-cascade deletion and its health propagation also remain unimplemented.

This is partial R2 evidence. It is not evidence for document compatibility,
sketch solving, API conformance, geometry, GUI operation, or a completed milestone.

## Kernel package ownership — R0 dependency

Tracking: [kernel issue #3](https://github.com/Oblikovati/RustyCADKernel/issues/3).
The kernel release tooling is integrated into its local `main` branch. It creates
one versioned `.crate` archive per workspace package, plus the lockfile, toolchain,
license, shared test support, manifest, and checksums. It requires owner CI and
extracted-package tests in default and all-feature configurations before it can
publish an immutable stable `vX.Y.Z` release.

The owner verification record is
`RustyCADKernel/porting/integration/2026-10-10/release-packages.json`.
It records 31 packages at version 0.1.0, 16,748 tests and 1,478 doctests passed in
each feature configuration, with 106 documentation tests explicitly ignored.
It names the checked source revision and package hashes. Concurrent later kernel
work is not covered by that receipt. Release CI must check the final tagged tree.

The owner subsequently published v0.1.2. The application intake is recorded in
[kernel release evidence](implementation-kernel-release.md), including the tested
revision and the scope of application checks. The application does not take ownership of the
kernel's unit tests, oracle tests, or package publication.

## Parameter quantity arithmetic — partial R2

Tracking: [issue #3564](https://github.com/Oblikovati/Oblikovati/issues/3564).
`crates/model/src/quantity` ports `model/param/quantity.go` and `unit.go` from
Go baseline `aa8297d1b0ac33c024f19e54821536d76e8a447a`. Values stay in document
working units. Arithmetic returns a new value, checks category compatibility,
and rejects unnamed products or quotients and zero divisors.

The internal `Dimension` enum has no numeric API IDs or serialization encoding.
The API owns wire categories; its documented host mapping remains a later task.
For source parity, equal-category addition/subtraction and negation retain the
Go behavior even for flag/text categories. The future expression evaluator must
apply its own nonnumeric value restrictions. IEEE non-finite results are retained,
not clamped; parameter health validation is not implemented by this component.

An independent product table checks multiplication and inverse division for all
81 category pairs. Tests also cover addition/subtraction for all pairs, signed
zero, diagnostic operands, negation, and IEEE arithmetic. Parsing, named-unit
conversion, working-scale selection, persistence, and parameter health remain
open. This component does not complete R2 or provide an editable part.

## Named units and working-scale conversion — partial R2

Tracking: [issue #3566](https://github.com/Oblikovati/Oblikovati/issues/3566).
`crates/model/src/units` ports the 26 named-unit spellings, preferred-unit
conversion, literal scanner, scale exponents, and new-document recommendation
from Go `model/param/units_of_measure.go` at the fixed migration baseline.
Default storage remains centimetres; display preferences are independent owned
copies and cannot modify stored quantities. A suffix overrides the default
category; a bare number uses its category's preferred unit. Flag/text categories
retain the Go bare-number behavior until the expression owner validates them.

Scale construction rejects non-finite values and unrepresentable squared/cubed
scales, as well as non-positive values. This intentionally closes a Go validation
gap: its `<= 0` test admits NaN/infinity. Literal input rejects invalid or
non-finite numeric values. Arithmetic conversion order remains multiply then
divide, with explicit scale multiplication for area and volume.

Tests check every unit against physical values in metres, square metres, or cubic
metres across centimetre, picometre, micrometre, and kilometre working scales.
They also check preference conversion, clone isolation, unchanged stored values,
conditioning policy, malformed UTF-8-safe scanning, signed zero, and invalid
configuration. The numerical checks allow only eight machine epsilons of relative
rounding error; they are not geometric tolerances.

Display string rendering and precision/format preferences are not implemented.
The document owner must still freeze scale after geometry exists and restore
legacy missing-scale fields to the centimetre default. Expression parsing,
parameter health, document persistence, and the API boundary remain open.

## Parameter expressions — partial R2

Tracking: [issue #3568](https://github.com/Oblikovati/Oblikovati/issues/3568).
`crates/model/src/expression` ports the baseline expression grammar, 17 built-ins,
constant folding, and stable-ID reference binding. The public model interface
uses caller-owned IDs and does not define an API identity or wire DTO.
Binding creates an independent snapshot; evaluation reads current values by ID.
Unresolved names and distinct bound IDs remain available to the parameter owner.

Expressions retain authored source and first-seen reference order. Arithmetic is
left-associative with the Go precedence rules. Valid constant subtrees fold;
dimension and unresolved-reference failures retain their evaluation stage.
Domain-generated non-finite values remain visible for later parameter health checks. Unknown functions fail before argument evaluation. Go `min`/`max` rules
for NaN, infinities, and signed zero are explicit rather than using Rust's
numerically different primitive behavior.

Node storage, evaluation, and destruction are flat. A 20,000-reference formula
runs without recursive evaluation. Recursive syntax has an explicit 128-level
limit and a positioned error. Like the baseline, the grammar does not accept
unary plus or general caret powers. Registered area/volume suffixes were added
with authored-input qualification below. It accepts Go's identifier vocabulary
and byte offsets.

Unlike the Go expression parser's fixed centimetre literal conversion, the Rust
parser receives the document units. Explicit literals use that working scale;
bare expression numbers remain unitless. This avoids mixing centimetre literals
with micrometre or kilometre stored reference values. Document-scale freezing
remains open. The owner components below address parameter health, binding
cascades, and rename/source rewriting.

### Expression evidence

The fixture `crates/model/tests/fixtures/go-expressions.tsv` captures 50 results
from the fixed Go revision. The isolated capture used the unmodified baseline
`expr_ast.go`, `expr_parser.go`, `expr_token.go`, `expr_eval.go`,
`expr_functions.go`, `quantity.go`, `unit.go`, and `units_of_measure.go` with
Go 1.27.0 on Linux/amd64. It ran each row through `Parse`, then `Eval(nil)`,
and recorded the error stage or dimension name and `math.Float64bits` result.
The Apache API Go module supplied the display-format type needed by the unit file.

Rust tests check the capture, all 14 unary built-ins across all nine internal
categories, arity errors, variadic dimensions, binding isolation, source error
order, and scaled references. Arithmetic and signed-zero corpus values require
bit equality. Transcendental comparisons permit eight machine epsilons of relative
error; this does not prove cross-platform bit equality for transcendental results.
The remaining model and native target gates must retain that distinction.

## Numeric parameter ownership — partial R2

Tracking: [issue #3570](https://github.com/Oblikovati/Oblikovati/issues/3570).
`crates/model/src/parameter` connects expressions, unit conversion, and dependency
tracking. It imports `ParameterKind` directly from `oblikovati-api` at exact
revision `9b5da090a9260892f54b4762d1225b5f892cbeb4`, now merged into the API owner's
`develop` branch. This is an integration pin, not proof of a published release;
application publication remains blocked by the owner-release checks.

The collection accepts caller-owned stable IDs, enforces unique nonempty names,
uses canonical editability categories, and recomputes only affected readers.
It supports measured/reference updates, forward-reference recovery, deterministic
change draining, clone isolation, guarded deletion, and explicit owner deletion.
Owner deletion clears every removed-ID binding before recomputing a shared tail.

Parse failures change nothing. Cyclic edits retain authored source, detach the
invalid drivers, and report a typed failure. Last-good values remain available
for inspection, but failed or non-finite values cannot become current values.
Readers of failed parameters become blocked and recover after a valid edit.
This intentionally tightens the Go behavior: its cycle test leaves a reader
healthy, and its scope can supply stale values from a failed driver. Rust
quarantines affected readers so later features cannot consume stale results.

The eight collection tests cover dependency diamonds, recovery, exact affected
sets, failed edits, read-only kinds, non-finite results, scaled formulas, deletion
of a driver shared by a transitive reader, and insertion validation. Automatic ID allocation, groups, text/boolean literals, and persistence remain
open. Rename, tolerance-selected values, and read footprints are covered below.

## Working branch policy

The user requested a single working branch on 2026-10-10. The existing application
PRs #3562, #3565, #3567, and #3569 were merged into `rust` with merge commits; API
PR #296 was merged into its owner's `develop`. Their merged feature branches were
removed. Further application work stays on `rust-port`. Any delegated worktree
changes must merge into that branch. Create a PR only when the user requests one.
Pushes to `rust-port` run the full gate and native target matrix without a PR.

## Parameter rename — partial R2

Tracking: [issue #3571](https://github.com/Oblikovati/Oblikovati/issues/3571).
The parser records byte spans only for actual reference tokens. Parameter rename
rewrites spans bound to the stable ID, then compiles replacements before changing
the collection. Units, function names, whitespace, and similar identifiers remain
unchanged. Failed cyclic formulas also track renames even when their graph edges
were detached. Repeated Unicode renames retain valid source positions.

Normal renames preserve edges, values, health, and the value-change drain. A new
name can resolve a pending reference; that case installs all new bindings before
recomputing affected readers. A resulting cycle becomes failed health rather than
using stale values. Model parameters record their renamed flag; user parameters
do not. Renaming to the same name is a no-op.

Insertion and rename now reject names that the expression lexer cannot represent
as one identifier. This tightens the Go owner's nonempty-name check, which could
admit names that no formula can reference. Eight tests cover stable identities,
source preservation, multiple renames, failed formulas, pending-reference recovery,
cycle health, model flags, and atomic rejection. Names and spans remain internal;
no wire DTO or API enum is duplicated.

## Tolerance-selected values and read footprints — partial R2

Tracking: [issue #3572](https://github.com/Oblikovati/Oblikovati/issues/3572).
Parameters store the canonical API `Tolerance` and `ModelValueType` directly.
Default, asymmetric deviation, symmetric, absolute limits, basic, reference,
minimum, and maximum edits are implemented. The bands use document working units;
absolute limits become deviations from the current nominal, as in Go.

Features can select nominal, upper, lower, or median values. Expression references
continue to read nominal values. The nominal path preserves signed zero without
adding a zero offset. Invalid or non-finite bands are refused before mutation;
failed/nonnumeric parameters and overflowing selected values produce named errors.
Tolerance and selection edits add only that parameter to the change drain, because
nominal expression readers do not change. Identical edits are no-ops.

`ParameterCollection::track` supplies an immutable model-value read scope with an
explicit footprint. Nested scopes also add their reads to the parent. An error
result retains reads of existing parameters, so later recovery can invalidate its
consumer. Footprints are deduplicated in parameter insertion order. No global or
collection-wide mutable read sink is used; edits cannot occur through a read scope.

Eleven tests cover Go model-value examples, band storage, rejected input, nominal
bit preservation, selection overflow, no-op edits, nested read sets, recovery, and
minimal invalidation. Feature-engine consumption of these footprints, tolerance
persistence, and display rendering remain open. Fits resolution is recorded below.

## Fits tolerance resolution — partial R2

Tracking: [issue #3573](https://github.com/Oblikovati/Oblikovati/issues/3573).
The Rust resolver supports nominal sizes greater than 3 mm through 500 mm,
IT5–IT11, and d/f/g/h shaft classes with mirrored D/F/G/H hole classes and
symmetric JS/js classes. Unsupported inputs return named errors before mutation.
Both supplied annotations must be supported; the hole governs when both exist.
The canonical API tolerance stores deviations in document working units.

The fixed Go baseline has two calculation defects. Its rounded power formulas
produce incorrect d/f deviations in several size steps. Its JS/js calculation
rounds half-micrometres outward. The Rust resolver uses integer-micrometre
fundamental deviations and preserves symmetric half-widths. The numerical source
is [ISO 286-2:2010](https://www.iso.org/standard/54915.html), tables 1, 18, 20,
21, and 23, checked in the
[published standard](https://asremavad.com/wp-content/uploads/2018/12/ISO-286-2-2010-ISO-code-system-for-tolerances.pdf).
These are bounded supported classes, not a claim of complete ISO 286 coverage.

Eight tests cover 672 captured non-JS Go cases with explicit correction deltas,
all supported symmetric widths, exact size boundaries, invalid syntax and sizes,
hole precedence, no-op edits, atomic rejection, and four working scales. The Go
capture comes from unmodified `iso286.go` and `iso286_table.go` at the fixed
baseline, run with Go 1.27.0 on Linux/amd64. The Rust tables avoid platform-dependent
power evaluation. Fit bands remain fixed deviations after a later nominal edit,
as in Go; automatic fit recomputation, persistence, and UI display are not claimed.

## Windows build-tool isolation

Tracking: [issue #3574](https://github.com/Oblikovati/Oblikovati/issues/3574).
[Run 38056840105](https://github.com/Oblikovati/Oblikovati/actions/runs/38056840105)
passed the Linux gate and both Linux/macOS architectures, but both Windows jobs
failed when Cargo tried to replace the running `oblikovati-build.exe`. The Cargo
alias now builds its executable under `target/xtask`; workspace commands retain
their normal output directory. A regression check requires separate output paths.
All workspace crates remain in build, test, lint, and coverage discovery. Both
Windows architectures passed in
[run 38057558427](https://github.com/Oblikovati/Oblikovati/actions/runs/38057558427).

## Text and Boolean parameters — partial R2

Tracking: [issue #3575](https://github.com/Oblikovati/Oblikovati/issues/3575).
The parameter collection now owns numeric, text, and Boolean values with an
internal typed enum. Literal parameters have no compiled numeric formula.
Text and Boolean insertion creates user parameters; edits cannot change their
value kind. Numeric quantity insertion also refuses text/Boolean dimensions, so
callers cannot create fake numeric storage for literals. The API contract remains
the shared external interface; this internal enum is not a second wire DTO.

Text source is a quoted JSON string; Boolean source is `true` or `false`.
Printable text matches the Go examples. Control-character spelling uses JSON
escapes (for example `\u0000`) instead of Go-specific `\x00` escapes. The literal
value retains the original Unicode text. No text is parsed as a formula. Text and
Boolean edits mark their parameter and existing numeric readers as changed;
identical edits do not. Numeric readers report `NonNumeric` with the offending
IDs and retain dependency edges for rename, removal, and later recovery.

Typed read scopes retain text/Boolean identities and failed reads, including
nested scopes. Immutable text borrows can outlive the scope, but Rust prevents
collection edits while those borrows remain in use. Ten regression tests cover
the Go literal examples, quoting, wrong-kind edits, atomic insertion failure,
read-only numeric parameters, forward references, rename, removal, recovery, and
read footprints. Persistence, UI editing, and text/Boolean expressions remain
outside this component; the legacy literal APIs do not provide those expressions.

The four legacy text/Boolean tests passed at `aa8297d1` with Go 1.27.0 using an
external temporary workspace that maps the relocated API `go/` module. The full
Rust gate passed with model line coverage 98.99%, build-tool coverage 90.70%,
and duplication 0.12%. These checks cover the current crates; they do not certify
an application UI or persistence implementation.

## User-authored expressions — partial R2

Tracking: [issue #3576](https://github.com/Oblikovati/Oblikovati/issues/3576).
User-input insertion and edit methods qualify unitless expressions with the
requested category's document display unit. Bare `7` becomes `7 mm` in the default
length context; `3*2` becomes `(3*2) * 1 mm`. Dimensioned, unresolved, and failed
expressions retain their source. Bound unitless formulas are wrapped without
replacing their references with current values, so edits and renames remain live.
The category is a qualification hint, not a permanent dimension constraint.

The expression parser now accepts registered area/volume suffixes, including
`mm^2` and `mm^3`. The unit registry remains the source of valid spellings and
conversion factors. General exponentiation and unregistered powered units are
refused. This corrects the Go grammar gap that made its own area/volume display
preferences generate invalid expressions. The original Go corpus remains intact;
its `1 cm^2` row is explicitly checked as an intentional behavior change.

Eight new tests cover Go source examples, every numeric display category, four
working scales, live references, malformed/unresolved/nonnumeric source, atomic
and read-only edits, powered-unit rejection, and rename token boundaries. The
three Go `TestQualifyAuthored` tests passed at the fixed baseline with Go 1.27.0.
Future UI, commands, and API user-input paths must call these authored entry
points; raw expression edits remain available for already qualified source.

## Unit and expression services — partial R2

Tracking: [issue #3577](https://github.com/Oblikovati/Oblikovati/issues/3577).
Named-unit conversion and category lookup reuse the existing registry. Category
names retain the Go lowercase vocabulary. Conversion is independent of document
preferences and working scale, and keeps the Go multiply-then-divide arithmetic
order. Unknown units, category mismatches, and non-finite input/output return
errors. Mismatch diagnostics include both unit names and the supplied value.

Document expression evaluation uses healthy current nominal values and the
existing scale-aware parser. It does not use tolerance-selected model values,
create parameters, or mark parameters changed. Unresolved, failed, and literal
references are refused. Non-finite results carry the original expression and
result in a named error. Authored qualification reuses this service, so a
non-finite expression retains its original source for downstream diagnostics.
The expression engine's lower-level IEEE arithmetic remains unchanged.

Eight new tests cover all 676 registered-unit pairs against independent physical
factors, category names, signed zero, overflow, live nominal reads, micro-units
at three scales, unusable references, and unchanged collection state. Four Go
unit/expression service tests passed at the fixed baseline with Go 1.27.0.
Display formatting, precision preferences, and API router integration remain open.

## Custom parameter groups — partial R2

Tracking: [issue #3578](https://github.com/Oblikovati/Oblikovati/issues/3578).
The collection owns custom groups with canonical API `ParameterGroupKey` values,
immutable internal identity, editable nonempty display names, and add-in client
attribution. Groups enumerate in creation order; members enumerate in parameter
insertion order. Membership is by stable ID, supports multiple groups, and survives
parameter rename. Removing a parameter detaches every membership. Reusing a caller
ID does not inherit those memberships.

Plain group deletion preserves member parameters. Explicit member deletion first
checks the complete set against the same removal rule used for single parameters.
Outside readers and surviving feature dimensions refuse the cascade before any
mutation. Dependencies wholly inside the removed set are permitted. Deletion also
cleans membership in surviving groups. Missing parameter/group queries return named
errors rather than silently pretending the group or parameter is empty.

Eight tests cover lifecycle, attribution, ordering, typed literal membership,
atomic rejection, rename/removal cleanup, and feature-dimension protection. The
cascade test enumerates all 64 ordered four-node DAGs and all 16 member subsets;
it checks survivor state and atomic rejection against the graph's edge definition.
Eight legacy group tests passed at the fixed Go baseline. Group metadata edits do
not mark nominal parameter values changed. Session metadata events, persistence,
API routing, and the separate copy-to-user operation remain to be ported.

## Multi-value expression lists — partial R2

Tracking: [issue #3579](https://github.com/Oblikovati/Oblikovati/issues/3579).
Parameter choices use the API's canonical `ExpressionListInfo`. Setting a list
keeps authored order and the current value. Clearing it resets both flags without
changing the value. Disabling custom order sorts strings lexically, including
Unicode and duplicates; re-enabling it keeps that current order. Metadata edits
do not mark numeric values changed.

Selection checks exact list membership unless custom values are allowed. Numeric
choices are already-qualified expression source and pass through collection edits,
so dependencies recompute. Text choices remain literal text. Boolean choices use
explicit `true`/`false` literals rather than the Go path that wrongly sent them to
the numeric expression setter. Custom values replace only the current selection;
they never accumulate in the fixed list. Malformed numeric syntax and invalid
Boolean selections are refused without changing the current parameter.

All list edits enforce read-only kinds, including clear/sort operations that the
Go methods did not guard. Seven tests cover lifecycle, custom replacement, sorting,
live formulas, typed selection, and rejected edits. Five legacy expression-list
tests passed at the fixed Go baseline. Choice strings are stored as authored
options and parsed when selected; this does not eagerly evaluate every option.
Session metadata events, persistence, UI dropdowns, and API routing remain open.

## Parameter annotations and user copies — partial R2

Tracking: [issue #3580](https://github.com/Oblikovati/Oblikovati/issues/3580).
Parameters now hold comment, key/export flags, visibility, precision, and the
canonical API display-format enum. Annotation edits validate nonnegative precision
and supported formats without changing nominal values or their change drain.
Read-only numeric parameters can still receive presentation annotations, as in Go.
Custom-property formatting and actual property export remain open.

Copy-to-user accepts a fresh ID from the future document allocator and chooses
`name_copy`, then `name_copy2`, and so on without overwriting existing names.
Numeric copies clone the compiled expression and rebind it through the collection;
measured constants therefore do not depend on reparsing diagnostic source text.
Text and Boolean copies use their typed constructors. Formula copies retain their
drivers but can be edited independently; they are not frozen-value snapshots.

Copies carry comment, key/export flags, precision, tolerance, and list choices.
As in Go, visibility and display format reset to defaults, model selection resets
to nominal, and copied lists use authored-order mode. Memberships and the model
renamed flag do not transfer. Failed formulas are evaluated for the new parameter
without inventing a last-good value. Seven tests cover these rules, generated-name
collisions, read-only measurements, typed copies, and atomic refusal. The legacy
`TestCopyToUser` passed at the fixed baseline with Go 1.27.0. Persistence, property
rendering, and UI/API mutation entry points remain open.

## Collection-wide tolerance selection — partial R2

Tracking: [issue #3581](https://github.com/Oblikovati/Oblikovati/issues/3581).
The collection can select nominal, lower, upper, or median model values across
all parameters with a nondefault tolerance record. Eligibility and the returned
count match Go, including zero-width and bandless annotation records and members
already at the requested selection. Plain parameters and literals remain untouched.
Individual and collection selection share validation and mutation rules.

The Rust sweep validates every eligible parameter before mutation. An unhealthy
member therefore refuses the whole operation without partial selection changes
or new dirty flags. Repeated selections retain the eligible count but do not mark
unchanged parameters dirty. Nominal expression readers do not recompute.
Five regression tests cover these rules, and the corresponding Go sweep test
passed at the fixed baseline. Collection settings, session transactions, and
feature-engine consumption of the changed model values remain open.

## Document parameter settings — partial R2

Tracking: [issue #3582](https://github.com/Oblikovati/Oblikovati/issues/3582).
Collection settings retain the Go defaults: three linear and two angular decimal
places, value display, empty standard-tolerance strings, and disabled flags.
Dimension display uses the canonical API enum. The collection validates the Go
router's 0–8 precision range and all five display modes before replacing settings.
Callers edit an owned copy and submit it; mutable settings references cannot bypass
validation. Rejected updates leave flags and tolerance strings unchanged.

Four tests cover defaults, all modes, precision boundaries, atomic refusal,
independent collection clones, and unchanged nominal/model values and dirty flags.
The Go default-settings test passed at the fixed baseline. Standard-tolerance
strings remain authored metadata, as in Go; applying them to new dimensions,
rendering them, persistence, and session/API integration remain open.

## Parameter kind conversion and built-in dimensions — partial R2

Tracking: [issue #3583](https://github.com/Oblikovati/Oblikovati/issues/3583).
Kind conversion preserves parameter identity, source, metadata, health, and graph
edges. User, Model, and Reference are the only targets. Built-in and Derived
sources are refused, including same-kind attempts, as in Go. Reference conversion
activates existing read-only edit rules; conversion back restores editability.
No nominal recomputation occurs solely because a kind changes.

Automatic model insertion uses a caller-supplied ID and the first unused `dN`
name, then marks the parameter built-in. Rename retains that guard. User copies
start without it. Name allocation shares collision handling with user copies;
malformed input and direct self-reference consume no name or ID. The caller still
owns lifetime identity allocation; reusing a removed display name does not imply
permission to reuse a persistent entity ID.

Five regression tests cover conversion, dependent recomputation, reference edits,
invalid targets, built-in copies, name collisions, and atomic insertion failures.
Five matching Go conversion/built-in tests passed at the fixed baseline. Feature
creation, action-mask enforcement, persistence, and API routing remain open.

## Unit display formatting — partial R2

The precise/display formatter, Go capture, supported limits, and scale correction
are recorded in [display implementation evidence](implementation-display.md).

## Persistent sketch identity — partial R2

Document namespaces and exact Go reference UUID compatibility are recorded in
[identity implementation evidence](implementation-identity.md).

## Persistence — partial R2

The new persistence crate and atomic replacement tests are recorded in
[storage implementation evidence](implementation-storage.md).

## Parameter recipe restoration — partial R2

See [parameter recipe implementation evidence](implementation-parameter-recipes.md).

## Document properties — partial R2

See [properties](implementation-properties.md), [history](implementation-history.md), [CLI](implementation-cli.md), and [automation](implementation-automation.md) evidence.
