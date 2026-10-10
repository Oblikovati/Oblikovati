# Document property implementation evidence

## Typed ordered property sets — partial R2

Tracking: [#3609](https://github.com/Oblikovati/Oblikovati/issues/3609).
The model now owns the four standard document property sets and ordered custom
sets. Properties use the canonical API `VariantValue`; no replacement wire DTO
or value-type tag is declared in the GPL model. Query projection returns the
API's `PropertyInfo` and preserves exposure provenance.

Creation order is stable across updates. Replacing a value retains its property
position and source parameter. Removal retains the set; inserting the same name
again appends it. Read-only collection views prevent edits that bypass validation.
A clone owns independent text/byte payloads. Empty names, uninitialized values,
and nonfinite numbers are refused before any set is created or property changed.
Go's unrestricted floating-point setter can retain nonfinite values, but the
canonical JSON API cannot represent them; Rust reports that failure at mutation.

The initial parameter exposure copies the selected numeric model value, including
tolerance selection, into the user-defined set. Typed/bulk exposure is covered below. The host must call exposure again after a
parameter changes; this is not an automatic subscription. Invalid or failed parameters leave existing properties unchanged. Bulk refresh
and rename cleanup are covered below. Custom-property string formatting and
session event integration remain pending.

Six tests cover standard/custom order, replacement/removal, every API payload
kind, signed zero, 64-bit integer boundaries, byte ownership, validation rollback,
canonical DTO projection, and parameter exposure/re-exposure. Go standard-set,
property access/removal, and numeric-exposure tests at the fixed baseline provide
the comparison. YAML property recipes and document-load integration are covered
below. Binary attribute/property codecs remain separate work.

## Authored property recipes — partial R2

Tracking: [#3610](https://github.com/Oblikovati/Oblikovati/issues/3610).
`restore_properties` and `with_properties` read and write the legacy flat
set/name/type/value records. Standard sets are implicit; nonempty custom sets and
property insertion order survive. Typed values retain integer width and floating
precision. Bytes share the resource decoder's Go-compatible padded base64 rules.
Boolean decoding accepts Go's true/false aliases. Empty byte input normalizes to
the same nil-slice form used by Go's owned property values.

Malformed, nonfinite, duplicate, and unknown-field records refuse reconstruction.
This intentionally replaces Go's skipped-invalid-record behavior with a visible
error. Export validates existing property records before replacing them, preserves
unrelated package/model sections, and verifies the reconstructed collection.
The flat legacy shape has no record for an empty custom set or parameter source
provenance. Standalone property export refuses those states rather than dropping them. The
coupled parameter/property export described below reconstructs provenance.

Typed file loading now reconstructs properties before identity registration and
returns them alongside parameters. Tests cover all five payload kinds, signed
zero, finite extremes, integer bounds, byte ownership/empty normalization, Go
Boolean/base64 rules, malformed input, order, retained sections, and unsupported
state. A Go-generated fixture contains text, Boolean, integer, double, binary,
and empty-byte properties. A load integration test proves an invalid property
claims no identity and a valid one is returned as a typed value. Go's part,
assembly, and value-type property round-trip tests provide the baseline.

## Parameter exposure refresh and coupled export — partial R2

Tracking: [#3611](https://github.com/Oblikovati/Oblikovati/issues/3611).
`sync_exposed_parameters` now refreshes all parameters with the export flag in a
separate candidate. Numeric values use the selected tolerance-band value; text
and Boolean parameters retain their API types. An unhealthy parameter has no
current property projection; its diagnostic health remains on the parameter.
Known owned projections are removed after rename, unexport, or
parameter deletion. Unrelated authored properties remain. An export flag claims
the same-named user-defined property, matching the existing single-parameter
exposure operation's replacement behavior.

The fixed Go source contains the exposure helper but no application refresh
call. Rust now supplies that missing connection explicitly. Typed file loading
reconstructs exposure provenance from the saved export flags after decoding
property records. The legacy flat file therefore needs no new provenance field.
Stored exposed values are snapshots; the flagged parameter is authoritative.

`with_parameter_properties` refreshes a candidate, captures parameters and
property snapshots together, then restores both and checks the reconstructed
collection. Standalone `with_properties` still refuses exposed state because it
has no parameter context. Session save commands must use the coupled path when
parameters are exposed. This resolves numeric/text/Boolean provenance persistence;
custom-property string formatting and empty custom-set persistence remain pending.

Persistence and model/load tests cover typed values, stale and
missing saved projections, provenance, selected-value updates, rename/unexport/
delete cleanup, retained authored content, and atomic export failure.

[#3659](https://github.com/Oblikovati/Oblikovati/issues/3659) removes stale current
projections when exposed parameters become unhealthy, without blocking publication
of their failure state. Recovery recreates the typed value and provenance. The
export flag continues to reserve its property name; the API refuses a manual
replacement and reports unhealthy parameter diagnostics on a direct property read.
Explicit exposure of an already unhealthy parameter remains an error.

This changes automatic projection refresh, including the live session and derived
refresh paths. Package validation still refuses unresolved parameter recipes;
saving and reopening sick recipes requires further port work. The property change
does not bypass that validation or claim sick-package support.

## Failed numeric diagnostic restoration

[#3668](https://github.com/Oblikovati/Oblikovati/issues/3668) adds a model-only
`restore_numeric_diagnostics` boundary for future unhealthy-recipe loading.
It restores a last-good numeric value, saved tolerance, and model-value selection
without clearing failure health or recomputing blocked readers. Current-value
reads still fail. Real expression recovery replaces the diagnostic value through
the normal evaluation path.

The candidate is checked before adoption: it must be a failed numeric recipe,
with a finite arithmetic diagnostic quantity when present, a valid tolerance band,
a known selection, and a finite selected diagnostic value. The zero-valued empty
API tolerance remains valid as the existing no-tolerance representation. Tests
cover atomic refusal, unknown/healthy parameters, nonnumeric/nonfinite quantities,
invalid bands/selections, overflow, absent diagnostics, signed zero, blocked
readers, and recovery. Package serialization of unhealthy recipes remains pending;
this change does not weaken the current strict package validator.
