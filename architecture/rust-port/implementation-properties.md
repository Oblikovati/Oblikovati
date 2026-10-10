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

Parameter exposure copies the selected numeric model value, including tolerance
selection, into the user-defined set. The host must call exposure again after a
parameter changes; this is not an automatic subscription. Invalid, failed, or
nonnumeric parameters leave existing properties unchanged. Local rename cleanup,
custom-property string formatting, and session event integration remain pending.

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
provenance. Export refuses those states until the host exposure/recipe migration
can represent them; it does not drop them silently.

Typed file loading now reconstructs properties before identity registration and
returns them alongside parameters. Tests cover all five payload kinds, signed
zero, finite extremes, integer bounds, byte ownership/empty normalization, Go
Boolean/base64 rules, malformed input, order, retained sections, and unsupported
state. A Go-generated fixture contains text, Boolean, integer, double, binary,
and empty-byte properties. A load integration test proves an invalid property
claims no identity and a valid one is returned as a typed value. Go's part,
assembly, and value-type property round-trip tests provide the baseline.
