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
the comparison. YAML/binary property codecs and document-load integration are
separate remaining work.
