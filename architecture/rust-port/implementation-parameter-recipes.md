# Parameter recipe implementation evidence

## Formatting and introspection state — partial R2

Tracking: [#3597](https://github.com/Oblikovati/Oblikovati/issues/3597).
Branch: `rust-port`; no PR is requested.

The Go recipe audit uses `model/compdef/serialize.go` and `param_recipe.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. A parameter recipe stores values,
annotations, tolerance, choices, custom-property formatting, rename state, and
disabled-action flags. These must all have model owners before restoration can
claim to preserve a supported parameter.

`ParameterState` now retains a `CustomPropertyFormat` and the canonical API
`ActionType` mask. Formatting defaults match Go: text property, document display
unit, three decimal places, leading zeros and unit strings shown, trailing zeros
hidden. Both property types and all 24 canonical precision values are accepted.
Undefined enum values and action bits are refused before mutation. Unit strings
are retained as formatting preferences; interpreting them belongs to property
output, not numeric parameter evaluation.

Collection setters restore the custom-property format, disabled-action mask,
and rename marker without recomputing nominal values or dirtying numeric
dependents. They also work for literal and read-only parameters, as required for
recipe loading. User copies reset these fields to defaults, matching Go's
`CopyToUser` behavior. The source stays unchanged.

Six tests cover defaults, every precision value and action mask, dependent
stability, invalid edits, read-only/literal metadata, and user-copy defaults.
Matching Go introspection/copy tests pass. Property value rendering, property
export, and host enforcement of restricted edit/rename/delete actions are not
implemented by these metadata setters.

## Remaining recipe integration

Full restoration must handle ordered creation, forward references, all five
parameter kinds, literal flavors, annotations, tolerance/selection, groups,
settings, and the state above. Failed restoration must not replace a live
collection. Export must distinguish authored formulas
from compiled constant expressions whose source is diagnostic text. The legacy
parameter recipe uses names rather than persisted parameter IDs; Rust identity
allocation and future schema evolution must account for that limitation.

## Live values and dependency reconstruction — partial R2

Tracking: [#3598](https://github.com/Oblikovati/Oblikovati/issues/3598).
`DocumentPackage::restore_parameters` now constructs a separate live collection
from the model's ordered `parameters` sequence. It restores all five numeric
kinds, text/Boolean users, annotations, rename/action state, and value choices.
The caller supplies document working units. Reference and derived values use
their stored working-unit quantity category; they are not converted as display
units. Editable formulas use the normal expression and dependency machinery.

Forward references can resolve after later entries load. Final unresolved,
cyclic, or nonfinite values refuse reconstruction. Successful loading clears
the initial changed-ID set. The result includes a fresh document-owned ID
allocator advanced past every reconstructed parameter. IDs follow recipe order,
matching the legacy name-based format; this does not add persistent parameter
IDs to the file schema.

Unknown fields and inconsistent value-flavor fields are refused. Groups,
settings, linked derived tables, tolerance, model-value selection, and
custom-property recipe records remain explicit unsupported cases. Their source
YAML stays retained in the package, and reconstruction cannot silently discard
them. This is an intermediate decoder, not completion of the parameter recipe
port or permission to execute unported features elsewhere in the model.

Eight tests cover live edit/recompute after loading, all kinds and literal
flavors, metadata, choices, malformed records, cycles, duplicates, absent
tables, and refusal of unsupported state. Four matching Go parameter recipe
round-trip tests pass at the fixed baseline. Export and the remaining recipe
decoders are the next integration work.
