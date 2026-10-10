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

Unknown fields and inconsistent value-flavor fields are refused. Linked
derived tables, tolerance, and model-value selection are covered below. This is an intermediate decoder, not completion of the parameter recipe
port or permission to execute unported features elsewhere in the model.

Eight tests cover live edit/recompute after loading, all kinds and literal
flavors, metadata, choices, malformed records, cycles, duplicates, absent
tables, and refusal of unsupported state. Four matching Go parameter recipe
round-trip tests pass at the fixed baseline. Export and the remaining recipe
decoders are the next integration work.

## Group reconstruction — partial R2

Tracking: [#3599](https://github.com/Oblikovati/Oblikovati/issues/3599).
The loader now restores `parameterGroups` after all parameters have loaded and
passed health checks. Group creation order, internal names, display names, and
client IDs are preserved. Membership resolves saved parameter names to the new
table's IDs. Members enumerate in parameter creation order, matching Go/model
behavior; a parameter can belong to multiple groups. Repeated member names are
idempotent. An omitted display name defaults to the internal name.

Four tests cover order and metadata, multi-membership, repeated names,
empty/null/memberless groups, unknown fields, duplicate group keys, and missing
members. Invalid groups discard only the separate reconstruction candidate.
The existing live table is not an input to this operation. Linked tables and
tolerance/selection reconstruction are covered below.

## Collection settings reconstruction — partial R2

Tracking: [#3600](https://github.com/Oblikovati/Oblikovati/issues/3600).
`parameterSettings` now restores standard tolerance strings, use/export flags,
linear/angular precision, dimension display mode, and expression-display state.
All five display spellings map to canonical API enum values. The model validates
both precisions in `0..=8`; malformed or unsupported settings discard the
separate candidate rather than partially adopting it.

Absent/null settings retain new-document defaults. A present record follows
Go's sparse decoding: omitted precision means zero, while an absent or unknown
display-mode spelling is refused. Standard tolerance strings stay metadata and
do not silently apply tolerance bands to existing parameters.

Four tests cover every field and display mode, defaults versus sparse records,
invalid precision, unknown fields, and unchanged numeric values/dirty state.

## Custom-property formatting reconstruction — partial R2

Tracking: [#3601](https://github.com/Oblikovati/Oblikovati/issues/3601).
`customProperty` now restores text/number property type, unit preference, all
24 precision spellings, and leading/trailing-zero and unit-string flags. The
values map to canonical API types and use the existing model setter. Numeric,
literal, and read-only parameters all retain this metadata.

Absent/null records keep the model's defaults. A present empty record uses
default type/precision but false display flags, matching Go's recipe decoder.
Unknown fields, property types, precisions, and incorrect shapes are refused.
Five tests include a 24-row precision-name/ID capture from the Go API plus the
metadata, sparse-record, literal/read-only, and failure cases. Actual formatted
property output remains separate work. Recipe export is covered below.

## Tolerance reconstruction — partial R2

Tracking: [#3602](https://github.com/Oblikovati/Oblikovati/issues/3602).
All 14 Go tolerance names and four model-value selections now restore through
canonical API types. Saved working-unit deviations and hole/shaft labels are
retained exactly. Fit tables are not resolved again, and saved limit deviations
are not interpreted as absolute limits. Later nominal edits keep the saved band.

Numeric metadata is applied after all expression drivers resolve. This permits
forward references and read-only numeric parameters. Unknown types, malformed
fields, nonfinite or reversed bands, nonnumeric targets, and overflowing selected
values refuse the separate candidate. Unlike Go's unchecked raw tolerance setter,
the Rust model validates the numeric band before it stores it. Absent/null
records retain default tolerance and nominal selection.

Tests cover all types/selections, retained fit labels/bands, forward references,
read-only values, default records, atomic rejection, and unchanged dirty state.
The Go fit, parameter-field, and introspection round-trip tests are the baseline.
Parameter recipe export is covered below; full live-model recipe export remains pending.

## Derived-table reconstruction and refresh — partial R2

Tracking: [#3603](https://github.com/Oblikovati/Oblikovati/issues/3603).
`derivedParameterTables` restores table IDs, source document names, linked-name
order, and feature ownership after the parameter list. Existing derived values
reconnect by name. Missing local names stay in the linked subset without a
produced reference. Canonical API references use the actual current local name,
so local renaming does not change the source name. Removing a local parameter
removes its produced reference and marks the table unresolved.

Restoration does not read another document or claim the source is healthy.
Tables start unresolved. The caller can supply resolved numeric values or an
unavailable-source indication. Refresh is atomic on invalid source quantities;
missing source values fail their links, keep last-good values, and block their
readers. Missing drivers fail before updated drivers recompute, so readers do
not retain a partial source snapshot as their last valid value. A later complete
refresh restores evaluation. This intentionally tightens Go's missing-source
behavior, which retained stale reader health. Identity, source, duplicate link,
kind, and multiple-owner checks also reject malformed or ambiguous records.

Six model tests and three persistence tests cover restoration, canonical
references, refresh/recovery, missing values, rename/removal, malformed input,
atomic rollback, and last-good preservation. The Go sync and derived-table
round-trip tests pass at the fixed baseline. Model-side table edits are covered
below. Source-document resolution and full model export still need host integration;
this does not complete the cross-document feature.

## Derived-table editing — partial R2

Tracking: [#3604](https://github.com/Oblikovati/Oblikovati/issues/3604).
The model now creates tables, replaces linked subsets, marks feature ownership,
and deletes tables through user or owner paths. Table IDs and new parameter IDs
come from the document. A subset edit takes exactly the IDs required for new or
previously unproduced links. Kept links retain identity, local names, annotations,
and tolerance state. Their measured values refresh from the caller's snapshot.

The complete edit is prepared in a separate collection. Missing source names,
invalid quantities, duplicate links/IDs, occupied names, and incorrect allocation
sets leave the original collection untouched. Removed links invalidate readers
and remove group memberships. A new link may reuse a removed local name with a
new identity. Feature-owned tables reject user deletion; deleting the owning
component can remove them explicitly. External ID allocators remain responsible
for preventing lifetime ID reuse; failed edits do not rewind those allocators.

Ten model tests cover creation, retained state, subset replacement, pending-link
production, malformed input, name reuse, ownership, and deletion. A persistence
integration test loads a pending link and supplies its new ID from the restored
document allocator. Go's corresponding add/subset/delete tests are the baseline.
Document source resolution and session commands/events remain pending.
Parameter recipe export is covered below.

## Edited parameter export — partial R2

Tracking: [#3605](https://github.com/Oblikovati/Oblikovati/issues/3605).
`DocumentPackage::with_parameters` builds a new package containing the current
parameter records, groups, settings, and derived tables. It retains all other
model fields and package sections, including add-in streams and resources.
Capture uses the same private recipe structs as restoration and canonical API
enum values. Output includes explicit defaults; it is valid legacy YAML but is
not byte-identical to Go's sparse writer.

Before returning, export restores the candidate and compares each parameter's
state in recipe order. A changed working scale, unhealthy value, or state the
current recipe cannot represent refuses the candidate without changing the
original package. The legacy format stores derived source/local names as one
name, so renamed local links are explicitly refused, as are built-in ownership
flags that require the future feature recipe decoder. These are remaining export
work, not silently dropped state. Measured-constant export is covered below. Parameter IDs still follow
the legacy load-order allocation; this does not add persisted parameter IDs.

Eight export tests cover every numeric kind, literals, all tolerance/selection
spellings, display modes, all 24 property precisions, annotations, choices,
groups, settings, linked tables, deterministic repeated export, empty replacement,
and refusal cases. A native-file test edits an expression, exports, saves through
the registered-identity/atomic-file path, and reopens the edited value while
retaining add-in streams.

The committed `tests/fixtures/rust-parameter-export.yaml` was generated by Rust
and accepted by Go `PartComponentDefinition.ApplyRecipe` at the fixed baseline.
The Go check asserted the dependency value, tolerance selection, text, Boolean,
and group. Rust restores and re-exports the same fixture in its local gate.
This completes a parameter-only save path; full feature/model export, document
source resolution, and session integration remain incomplete.

## Measured-constant recipe support — partial R2

Tracking: [#3606](https://github.com/Oblikovati/Oblikovati/issues/3606).
Go `constantExpr` stores a number followed by a quantity category, such as
`1.25 length`. This is not an authored expression or a display-unit literal.
The recipe decoder now classifies this exact two-token shape and restores a
measured constant directly in working units. It does not convert through a named
unit or approximate an equivalent expression. Normal authored expressions still
use the expression parser; that parser's accepted syntax is unchanged.

One explicit category table supplies stable names and their inverse. Rust debug
type names no longer enter saved quantities or constant sources. Earlier Rust
capitalized diagnostic categories are accepted and normalized. Nonfinite values,
Boolean/text categories, and malformed records are refused.

The committed Go-generated measured-parameter fixture covers all seven numeric
categories. The fixed Go baseline writes this fixture but its `ApplyRecipe`
rejects the first `1.25 unitless` expression. Rust now reads it. This is a repair
of the legacy reader/writer mismatch, not a new schema or a promise that the old
Go reader can load measured-constant recipes written by either implementation.

Tests cover all three editable kinds, seven numeric categories, three working
scales, and seven floating-point values including signed zero, the smallest
subnormal, and the largest finite number. Every saved value reloads bit-for-bit.
A copied read-only area measurement exports, reopens as an editable user
parameter, and recomputes its reader after an authored edit. Additional tests
cover category-name references, named-unit literals, malformed input, and old
Rust diagnostic capitalization. Renamed derived links, built-in feature
ownership, and full feature/model export remain pending.

## Document unit context — partial R2

Tracking: [#3607](https://github.com/Oblikovati/Oblikovati/issues/3607).
`document_units` decodes the six unit preferences, working scale, length/angle
precision, and length/angle formats. Absent keys use the legacy defaults,
including centimetres as the working length unit. `restore_document_parameters`
uses this stored context before it evaluates expressions. The lower-level
`restore_parameters(units)` remains available for an explicitly supplied context.

Parameter export now includes the unit context and validates the candidate by
loading that context back from the file. Unknown or invalid stored unit settings
refuse export rather than disappearing during replacement. An existing nonempty
model cannot acquire a different working scale through a parameter-only export;
that would reinterpret retained geometry. A new empty document can select its
initial scale. Decimal, fractional, architectural, and DMS display preferences
remain presentation only; measured constants do not undergo display conversion.

Five tests cover sparse/default maps, every setting, format and precision
boundaries, invalid keys/values, non-default-scale export/reload, and refusal of
scale changes on retained models. Go's reserved-key and working-scale recipe
tests are the baseline. Rust's current display precision representation is
bounded to `0..=255`; larger Go values are explicitly refused rather than clamped.
Full document reconstruction and feature-owned units remain host integration work.
