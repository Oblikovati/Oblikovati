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

The existing YAML package preserves its recipe, but does not yet build live
parameter collections from it. Restoration must handle ordered creation,
forward references, all five parameter kinds, literal flavors, annotations,
tolerance/selection, groups, settings, and the state above. Failed restoration
must not replace a live collection. Export must distinguish authored formulas
from compiled constant expressions whose source is diagnostic text. The legacy
parameter recipe uses names rather than persisted parameter IDs; Rust identity
allocation and future schema evolution must account for that limitation.
