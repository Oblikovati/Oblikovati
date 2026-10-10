# Typed automation host — partial R2

Tracking: [#3627](https://github.com/Oblikovati/Oblikovati/issues/3627) and
[#3628](https://github.com/Oblikovati/Oblikovati/issues/3628), plus parameter
inspection in [#3629](https://github.com/Oblikovati/Oblikovati/issues/3629) and
collection queries in [#3631](https://github.com/Oblikovati/Oblikovati/issues/3631),
and member detail in [#3632](https://github.com/Oblikovati/Oblikovati/issues/3632).
The `oblikovati-automation` crate provides `SessionQueryHost`, an implementation
of the canonical API client `Caller` transport over an immutable workspace
borrow. All method names, argument types, and result DTOs are imported from the
API crate. `QUERY_METHODS` publishes the exact implemented inventory.

## Implemented inventory

| Method | Context and behavior |
| --- | --- |
| `parameters.getDetail` | Inspect member presentation, engineering value, tolerance, choices, and dependencies. |
| `parameters.getSettings` | Read all document parameter settings and the canonical display mode. |
| `parameters.groups.list` | Read groups in creation order and members in parameter order. |
| `parameters.list` | Inspect parameters in insertion order, including failed parameters. |
| `parameters.get` | Inspect a named parameter's expression, kind, value, and health. |
| `parameters.drivenBy` | Read direct driver names in parameter insertion order. |
| `parameters.dependents` | Read direct reader names in parameter insertion order. |
| `documents.getUnits` | Read active document preferences, display precision/format, and working scale. |
| `units.convert` | Convert compatible physical unit names; no document is required. |
| `units.getTypeFromString` | Resolve a registered unit name to its category without a document. |
| `units.getStringFromType` | Read the active document's preferred unit for a category. |
| `units.getStringFromValue` | Format a stored working-unit value using active document display rules. |
| `units.getPreciseStringFromValue` | Format a stored working-unit value at full precision. |
| `units.getValueFromExpression` | Evaluate nominal parameter values; convert bare numbers through document preferences. |
| `units.getDatabaseUnitsFromExpression` | Evaluate and return the resolved quantity category. |
| `units.isExpressionValid` | Return evaluation or dimension errors as a validity result. |
| `units.compatibleUnits` | Test dimensional compatibility; evaluation failures return false. |
| `units.getDrivingParameters` | Return distinct names in source order without resolving them. |
| `units.getLocaleCorrectedExpression` | Validate canonical syntax and preserve the source text. |

`SessionQueryHost` selects a supported read-only method before executing it. Other methods,
including edits, are refused explicitly. Request/response serialization uses the
canonical types, including their nullable field behavior. A root JSON null maps
to the argument type's default, matching Go decoding; malformed JSON is refused.
Physical conversion and formatting preserve the model's finite-value and
dimension checks. No DTO or numeric-unit implementation is duplicated here.

Forty-two integration tests call the host through the real Rust API `Client`. They
cover all nineteen query methods and eleven edits, active-document changes, display formats, non-default
working scale, missing document context, dimensional failures, malformed calls,
and unsupported mutation methods. They assert unchanged dirty state, generation,
notification counts, and undo history. The API transport is exercised directly;
these are local Linux tests, not C ABI or native add-in tests.

The host has no session queue, C callback registry, remote transport, or add-in
lifecycle. Those remain R2/R4 work. The limited method
inventory is not evidence that all operations in the API version are implemented.

Expression evaluation uses the existing model evaluator and reads healthy nominal
parameter values, without applying model tolerance offsets. As in Go,
`getValueFromExpression` converts only unitless results through the requested
category's preferred unit; already dimensioned values retain their resolved
working-unit value. Callers use `isExpressionValid` or `compatibleUnits` to test
the requested category. Syntax-only calls need no active document and do not
resolve names or require a successful numeric evaluation. Empty dependency lists
serialize as an array, not null.

The expression query tests cover non-default working scale, preferred inches,
nominal parameter references, mismatched dimensions, unresolved names, parse
failures, nonfinite results, and division by zero. The existing Go router unit
service tests also pass on Linux. Native transport and mutation dispatch remain
unimplemented.

Parameter inspection returns precise nominal values, literal text, and Boolean
values. Failed parameters retain their last-good value and report the failure;
a numeric parameter that never evaluated has the Go-compatible unitless zero
value. Health messages identify unresolved names, unavailable external links,
cycles, evaluation failures, blocked dependencies, nonfinite values, or nonnumeric
dependencies. Parameter-kind spellings are shared with package export.
The canonical DTO omits empty neighbor lists; an empty parameter list is an array.
The query tests check order, values, errors, direct links, and unchanged session
state. Parameter value/member mutations remain to be wired.

Collection queries preserve all eight parameter settings fields, including empty
standard-tolerance strings, false flags, and zero angular precision. Display-mode
names share the model conversion used by package export. Group queries return
stable internal names, display names, add-in client IDs, and ordered member names;
the canonical API omits empty group/member lists. Tests cover all five display
modes, all settings fields, empty collections, missing document context, creation
and membership order, and unchanged session state.

Member detail includes annotations, action restrictions, built-in/renamed flags,
custom-property formatting, optional expression choices, and graph neighbors.
Numeric parameters include their selected engineering value and tolerance band;
text and Boolean parameters omit the numeric tolerance. The stored zero tolerance
has the canonical default type. Displayed values remain precise nominal values.
Inspection retains the last-good engineering value of a failed parameter with its
health diagnostic; the model's feature-value read continues to reject that state.
Precision and tolerance vocabularies are shared with persistence. Tests cover
numeric defaults, tolerance offsets, failed values, typed literals, presentation,
choices, graph links, missing context/names, and unchanged session state.

## Transactional settings edits

[#3633](https://github.com/Oblikovati/Oblikovati/issues/3633) adds `SessionHost`, a
synchronous canonical `Caller` over exclusive workspace command access. It routes
known queries to `SessionQueryHost` and publishes supported edits separately in
`EDIT_METHODS`. The first edit is `parameters.setSettings`. An unsupported method
is refused before document mutation. Overlapping synchronous calls return an
explicit busy/reentry error instead of a borrow panic.

Settings updates clone and validate the collection settings, then run through
`SessionCommands::run` with the label `Edit Parameter Settings`. The response is
encoded inside the candidate edit before commit. Session policy controls veto,
property synchronization, notifications, grouped edits, no-op detection, and
undo/redo. No separate host undo log exists. Omitted and null fields preserve
values; explicit false, zero, and empty strings replace them. Root null means an
empty update. Arrays and scalar request roots are refused by the shared decoder.

An owner-borrow regression test checks rejection of overlapping query/edit calls
and successful use after the borrow is released. Six additional integration tests exercise the real typed client and wire calls:
undo/redo, live query results, atomic validation errors, optional/reset fields,
no-op redo preservation, existing groups, missing context, unported edits, and
observer veto rollback. The display-mode parser is shared with package restore.
This is an owner-side transport, not a bounded queue or native callback registry;
those boundaries must be added before native add-in use is claimed.

## Transactional group edits

[#3634](https://github.com/Oblikovati/Oblikovati/issues/3634) adds the five group
edits: `parameters.groups.add`, `delete`, `setDisplayName`, `addMember`, and
`removeMember`. The host decodes each call into a prepared edit before opening its
transaction. Settings and group edits share the same session command and response
encoding path. Labels preserve the Go session's group-operation names.

Group creation preserves add-in provenance and defaults an empty display name to
the internal name. Display renaming keeps the internal key unchanged. Membership
requires both an existing group and parameter; it cannot use the UI's automatic
group-creation path. Removing one membership preserves other groups. Repeated
membership operations are no-ops and do not add undo records.

Deletion preserves parameters unless `deleteParameters` is true. A cascade uses
the model aggregate's dependency checks and refuses deletion that would sicken a
surviving reader. The candidate transaction prevents partial deletion; a successful
cascade is one undo step. The canonical void reply is `{}`. Tests cover the typed
lifecycle, order, independent membership, strict lookup, validation errors,
cascade refusal/success with undo/redo, grouped abort, and observer veto. Existing
settings and reentry tests also run against the shared edit path.

## Parameter identity and lifecycle edits

[#3635](https://github.com/Oblikovati/Oblikovati/issues/3635) adds
`parameters.rename`, `parameters.convert`, and `parameters.delete`. Rename uses
the model's stable-ID rewrite so dependent expressions and group projections
follow the new name. Conversion accepts trimmed, case-insensitive user/model/
reference names, preserves identity and graph links, and refuses built-in or
derived parameters through the model. Deletion uses the dependency-aware public
remove operation, never the owner-only forced removal path.

All three calls use the shared candidate transaction, including exposed-property
synchronization and response encoding. Undo restores parameter names, category,
expressions, group membership, and exposed properties together. No-op rename and
conversion add no undo entry. Tests cover typed responses, dependency rewrites,
stable IDs, conversion rules, invalid names, deletion refusal, undo/redo, veto,
and exposed-property rename/deletion. The selected Go rename/conversion tests
also pass on Linux.

Session transaction observations are active. The Go host's `parameter.changed`
push notifications are not yet delivered to add-ins; that requires the pending
outbound event and native add-in transport boundary. These edit implementations
do not claim full event-surface parity.

## Parameter creation and numeric expression edits

[#3636](https://github.com/Oblikovati/Oblikovati/issues/3636) adds `parameters.add`
and `parameters.set`. Creation reserves identities through the document's
`RecipeReservations`, outside undo snapshots. A failed or undone creation never
makes its reserved ID reusable in the live document. Numeric creation uses the
user category unless the trimmed kind is model, matching the Go router. Text
creation permits empty strings; Boolean creation preserves Go `ParseBool`
spellings. Literal creation always creates user parameters.

Numeric edits qualify bare values with the existing parameter's preferred unit,
including the last-good category of a failed parameter. They use the model graph
to rebind and recompute dependent parameters, and report health when evaluation
fails. Syntax, cyclic edits, read-only edits, invalid creation types, and name
collisions are refused through the candidate transaction. The current model
keeps text/Boolean values typed; numeric expression edits cannot replace those
literal kinds. Their separate typed setter surface remains to be connected.

Tests cover allocation after undo, typed creation, Boolean spellings, model
creation, dependency recomputation, failed-value recovery, invalid input, group
abort, and veto. Model parameters also report `inUse` without parameter readers,
matching the Go model contract. Numeric parameter recomputation is implemented;
feature/geometry recomputation after an edit still requires the pending feature
engine. This does not claim feature or outbound notification completion.
