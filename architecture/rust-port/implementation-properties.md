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

Parameter exposure projects the selected numeric model value, including its
tolerance selection, through the stored custom-property format into the
user-defined set. Text and Boolean parameters retain their types. Session edits
and history navigation refresh these projections. Direct model users must call
exposure again after parameter changes. A failed direct exposure leaves the
collection unchanged; bulk refresh removes unhealthy projections and rebuilds
them on recovery. Rename cleanup and storage are covered below.

Six tests cover standard/custom order, replacement/removal, every API payload
kind, signed zero, 64-bit integer boundaries, byte ownership, validation rollback,
canonical DTO projection, and parameter exposure/re-exposure. Go standard-set,
property access/removal, and numeric-exposure tests at the fixed baseline provide
the comparison. YAML property recipes and document-load integration are covered
below. The shared binary value and property collection codecs are covered below;
attribute collection encoding is also covered below; document integration remains open.

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
The flat legacy shape has no record for an empty custom set. Schema 3 stores
ordered custom-set names in `model.customPropertySets`, including empty sets
([#3670](https://github.com/Oblikovati/Oblikovati/issues/3670)). Declarations
must be unique, nonempty custom names; records must belong to declared or
standard sets. Schema 2 retains its implicit set reconstruction. Ordinary and
coupled exports share this capture path. File save/reopen tests cover a set
whose last property was removed. Standalone property export still refuses
parameter provenance; the coupled export below reconstructs it.

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
numeric custom-property formatting is now applied by the shared exposure path
([#3672](https://github.com/Oblikovati/Oblikovati/issues/3672)).

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
refresh paths. The schema-3 save/load support below extends this behavior across
file lifetimes; schema-2 import retains its original healthy-recipe validation.

## Failed numeric diagnostic restoration

[#3668](https://github.com/Oblikovati/Oblikovati/issues/3668) adds a model-only
`restore_numeric_diagnostics` boundary for unhealthy-recipe loading.
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
readers, and recovery. Package serialization uses this boundary as described below.

## Unhealthy recipe save and reopen

[#3669](https://github.com/Oblikovati/Oblikovati/issues/3669) persists recoverable
parameter failures in the unreleased schema-3 recipe. Expression text remains the
source of truth: restore evaluates it again rather than trusting serialized error
messages. Optional `lastGood` numeric diagnostics carry finite working quantities
and are accepted only for failed numeric parameters. They never become current
feature inputs. Tolerance and selection metadata use the checked model restorer.

Derived-table `failure` metadata records an unavailable source or a selected
missing-name subset. Restore validates the subset and applies link failure before
restoring diagnostics and exposed properties. Blocked readers retain their own
last-good values. Current source resolution on API open can repair these failures;
otherwise direct value access still fails and stale property projections stay absent.

Schema-2 input remains subject to its existing healthy-recipe validation. Syntax
errors, cycles, malformed types, nonfinite diagnostic values, invalid tolerance
metadata, and ambiguous link ownership remain errors. Saving a recoverable failure
promotes the candidate to schema 3 and verifies its reconstructed parameter state
before storage. Unsupported recipe state still produces a named save error.

Tests cover unresolved and evaluation failures, blocked readers, tolerance offsets,
signed zero, unavailable/partly missing sources, malformed diagnostics and subsets,
cycle refusal, and live-source recovery. Typed-client tests cover source close,
target save/close/reopen with an exposed failed parameter, and source reopen that
restores the parameter and property. All tests use owned Rust sessions and file
adapters; native UI acceptance remains pending.

## Numeric custom-property formatter — partial R2

Tracking: [#3671](https://github.com/Oblikovati/Oblikovati/issues/3671).
`CustomPropertyFormat::format_numeric` converts a working-unit quantity into
text or an unrounded numeric property. Text supports the 24 canonical precision
choices, explicit or document units, decimal zero flags, and unit labels.
Fractional lengths share the document fraction formatter. Angular text rounds
once to integer ticks and carries seconds/minutes into degrees. Angular precision
selects sexagesimal degrees even when the document angle unit is radians.

Tests cover every precision, dimensional rejection, invalid settings, overflow,
working-scale powers, signed numeric zero, and unchanged source values. These
are contract and analytic tests; the Go exposure helper ignores its stored format
settings and cannot act as a formatter oracle. The shared exposure path now calls this formatter for numeric parameters;
text and Boolean parameters retain their types. It uses the tolerance-selected
model value and the owning document's working scale and display units. There
is no second numeric projection path.

Integration evidence: [#3672](https://github.com/Oblikovati/Oblikovati/issues/3672).
Typed API tests cover unit changes, text/number switching, undo/redo, save/reopen,
and expression edits after reopening. Existing source-refresh tests verify the
same formatted projection for derived parameters. Incompatible units or precision
fail the command without changing properties, parameters, or history.

For schema 2 files, the export flag and stored parameter formatting remain
authoritative. A saved numeric property snapshot is regenerated using those
settings, including default text formatting when no settings were stored.
This changes the projected type of a legacy default numeric exposure to text;
its authored expression and evaluated model value do not change. No legacy
numeric fallback is retained. Non-exposed authored properties are unchanged.
Decimal flags, fractional text, angular precision, and parameter provenance
survive save/reopen tests.

## Binary metadata value codec — partial R2

Tracking: [#3673](https://github.com/Oblikovati/Oblikovati/issues/3673).
The persistence `metadata` module reads and writes the Go attribute/property
value layout. Compact tags 0 through 4 encode Boolean, signed 64-bit integer,
binary64 double, UTF-8 string, and bytes. These tags are independent of the
public API enum numbers. String and byte lengths are little-endian u32 values.
The decoder checks the available input before allocating and returns the unread
suffix for the future collection decoder.

Eleven fixtures were captured through Go `PropertySets.EncodeProperties` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`, using Go 1.27.0 on Linux. Tests
compare exact output bytes, retain signed zero and subnormal doubles, and
check every truncation boundary. Invalid tags, Boolean bytes other than 0 or 1,
invalid UTF-8, nonfinite doubles, and oversized declared blobs are refused.
These strict failures replace Go's permissive Boolean/string decoding and match
the Rust model's finite-value and UTF-8 requirements. Empty byte arrays normalize
to the Go nil-slice representation. Property and attribute collection encoding are covered below. Live reference-key
binding and document integration are still required.

## Binary property collections — partial R2

Tracking: [#3674](https://github.com/Oblikovati/Oblikovati/issues/3674).
`metadata::encode_properties` and `decode_properties` implement the Go binary
collection layout: set count, ordered set names, property counts, and each
property's name, source-parameter name, and typed value. Empty custom sets and
stored provenance survive without parameter evaluation. A model restoration
method validates a stored property before replacing its value and source name.

The `go-binary-properties.bin` fixture was generated through
`PropertySets.EncodeProperties` at Go baseline `aa8297d1`, using Go 1.27.0.
It includes the four standard sets, an empty custom set, all five value types,
signed zero, UTF-8 with a NUL, empty bytes, and an exposed `width` parameter.
Rust encoding matches those bytes; decoding and re-encoding are byte-identical.
Tests also reject every truncated prefix, excess bytes, impossible counts, and
duplicate names. The decoder requires all four standard sets in their canonical
order before custom sets, matching collections created by `NewPropertySets`.
Unlike the Go decoder, it refuses incomplete or reordered standard-set lists
instead of creating a collection that violates the Rust model's invariant.

This codec is not yet connected to document containers. The current YAML
parameter/property save path remains authoritative for live Rust documents.
Binary attribute collections are covered below; live reference-key binding remains open.

## Reference-key anchored attributes — partial R2

Tracking: [#3675](https://github.com/Oblikovati/Oblikovati/issues/3675).
The model `attribute` module stores ordered targets, namespaces, and canonical
API values. Replacement retains position; removal retains empty owners until
explicit namespace or target removal. Searches return matches in anchor and
attribute insertion order. Clones own independent value storage. Attributes
and properties now share finite-value validation, byte normalization, and
signed-zero-aware value comparison.

`AttributeTarget` validates the reference envelope and normalizes it to the
current encoding. This matches Go `RefKey.Encode`: envelope scheme is provenance,
not identity. Equal context/kind/payload triples therefore share metadata across
legacy/current envelope forms. This validates identity bytes only; it does not
resolve an entity or claim kernel naming support. Malformed keys are refused
before they can enter the collection.

Tests cover insertion order, replacement, empty namespace/anchor retention,
missing-target operations, filtered searches, all payload kinds, clone isolation,
signed zero, malformed envelopes, and invalid-edit atomicity. The Go
`model/attr` suite passes at baseline `aa8297d1`. Binary attribute storage is covered below. Session snapshots, API dispatch, and
live entity binding remain open.

## Binary attribute collections — partial R2

Tracking: [#3676](https://github.com/Oblikovati/Oblikovati/issues/3676).
`metadata::encode_attributes` and `decode_attributes` implement the Go layout
for ordered reference-key anchors, namespaces, and typed attributes. Empty
anchors and namespaces are retained. Property and attribute codecs share name
uniqueness checks and length-prefixed string/blob writing.

The Go `AttributeManager.Encode` fixture at baseline `aa8297d1` contains a
document anchor, empty namespace, string/integer attributes, and an empty
external-reference anchor. Rust reads and writes identical bytes. Legacy target
envelopes normalize to current encoding; two serialized keys that normalize to
the same identity are rejected as duplicates. Tests cover every truncated prefix,
malformed keys/counts, duplicate namespaces/attributes, and trailing input.

This completes the binary collection codecs. Attaching these collections to
live document snapshots, persistence sections, and typed API commands remains
open; decoding does not resolve an entity in the kernel.
