# Typed automation queries — partial R2

Tracking: [#3627](https://github.com/Oblikovati/Oblikovati/issues/3627) and
[#3628](https://github.com/Oblikovati/Oblikovati/issues/3628), plus parameter
inspection in [#3629](https://github.com/Oblikovati/Oblikovati/issues/3629).
The `oblikovati-automation` crate provides `SessionQueryHost`, an implementation
of the canonical API client `Caller` transport over an immutable workspace
borrow. All method names, argument types, and result DTOs are imported from the
API crate. `QUERY_METHODS` publishes the exact implemented inventory.

## Implemented inventory

| Method | Context and behavior |
| --- | --- |
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

Dispatch selects a supported read-only method before executing it. Other methods,
including edits, are refused explicitly. Request/response serialization uses the
canonical types, including their nullable field behavior. A root JSON null maps
to the argument type's default, matching Go decoding; malformed JSON is refused.
Physical conversion and formatting preserve the model's finite-value and
dimension checks. No DTO or numeric-unit implementation is duplicated here.

Fourteen integration tests call the host through the real Rust API `Client`. They
cover all sixteen methods, active-document changes, display formats, non-default
working scale, missing document context, dimensional failures, malformed calls,
and unsupported mutation methods. They assert unchanged dirty state, generation,
notification counts, and undo history. The API transport is exercised directly;
these are local Linux tests, not C ABI or native add-in tests.

The host currently has no mutation dispatch, session queue, C callback registry,
remote transport, or add-in lifecycle. Those remain R2/R4 work. The limited method
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
state. Parameter detail, settings, and mutations remain to be wired.
