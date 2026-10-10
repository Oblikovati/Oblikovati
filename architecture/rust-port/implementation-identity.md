# Document and sketch identity implementation

Tracking: [#3585](https://github.com/Oblikovati/Oblikovati/issues/3585).
Branch: `rust-port`; no PR is requested.

## Persistent sketch namespaces — partial R2

The model owns `DocumentGuid` and `SketchReferenceUuid`; the `uuid` dependency
is private to that boundary. A saved document GUID plus a persisted local `u64`
ID derives a UUIDv5 with the exact Go `sketch:<id>` or `sketchEntity:<id>` name.
This preserves existing sketch reference bytes across language migration.
The kind tags, decimal ID spelling, and UUID algorithm are compatibility rules.
SHA-1 serves UUIDv5 compatibility, not authentication. Hash-derived names do not
provide a mathematical guarantee against all possible collisions.

Canonical 8-4-4-4-12 and compact 32-hex GUIDs are accepted, including uppercase
hexadecimal input. Output is canonical lowercase. Legacy namespace bytes are
preserved without imposing a new UUID version or variant restriction. Invalid
input returns an error with its value and expected shape. Misplaced separators,
braces, URNs, and whitespace are refused. Go previously removed hyphens at any
position; that accidental acceptance is not part of the file format contract.

The fixture contains 15 rows captured from Go `model/identity` at fixed revision
`aa8297d1b0ac33c024f19e54821536d76e8a447a`: three namespaces, five IDs, and both
reference kinds. IDs include zero, values beyond 32 bits, and `u64::MAX`.
Tests compare all 30 reference values exactly, check canonical formatting and
input rejection, and check separation over 1,024 document/kind/local-ID samples.
The corresponding Go identity tests also pass at that baseline.

This component does not allocate local IDs, generate document GUIDs, save files,
or resolve geometry. Those owners must supply the saved identity, not a session
ID or build counter. Storage round trips, topology provenance, key binding,
and collision/ambiguity rejection on model insertion remain open. ADR-0005's
complete identity acceptance gate is not met by this UUID derivation test.

## File revision transitions — partial R2

Tracking: [#3586](https://github.com/Oblikovati/Oblikovati/issues/3586).
`FileIdentity` ports Go `model/doc/file_identity.go`. Creation supplies a stable
document GUID and software version. Save preparation returns a new identity
with a checked `u64` counter and a fresh file revision. A known, unchanged
SHA-256 recipe digest retains the database revision. A changed digest replaces
it. Version stamps are caller supplied, so model code does not read build state.

An unknown digest always replaces the model revision. This corrects the Go
implementation, whose empty-string comparison could retain the revision on
successive unknown digests despite its conservative-invalidation comment.
The digest covers the exact serialized recipe bytes; canonical recipe encoding
is the persistence owner's responsibility.

Copy preparation supplies a new document GUID, file revision, counter of one,
and current creation/save versions. It preserves the source model revision and
digest, minting a database revision if absent. A storage owner must first ensure
that these describe the recipe being copied; this function does not inspect a
live document or its pending edits.

`GuidSource` is injected. `SystemGuidSource` wraps operating-system randomness
and UUIDv4 construction. Generation errors and repeated GUIDs are explicit
errors. New stamps cannot equal the source document/revision stamps or another
stamp issued during the same transition. This is a local guard, not a global
UUID uniqueness proof. Counter overflow is refused before consuming randomness.

Thirteen tests cover initial state, ordinary save, changed/unknown recipes,
copy identity, SHA-256 known vectors, v4 shape, failure at each copy-generation
step, save failure, repeated IDs, and counter overflow. Candidate preparation
borrows the source, so failed preparation or discarding a candidate cannot
mutate it. Actual storage failure/retry tests and committing a candidate only
after an atomic file write remain open.

## Identity record restoration — partial R2

Tracking: [#3587](https://github.com/Oblikovati/Oblikovati/issues/3587).
`FileIdentityRecord` maps the seven camelCase fields from Go's
`yamlcodec.FileIdentityRecord`. Empty fields are omitted when serialized.
The mapping uses Serde without coupling the model to a YAML parser or storage.
Seven tests use JSON as a mapping transport; they do not prove YAML package
compatibility or a complete `.obk` round trip.

Restoration validates GUIDs and the 64-hex SHA-256 fingerprint, preserves version
strings and counters, and normalizes accepted uppercase/compact hexadecimal
spellings. Absent revisions and fingerprints remain unknown. A missing
`internalName` keeps the fresh identity, matching Go's pre-identity file rule.
The caller owns that fresh identity; restoration does not consume randomness.

Unsigned counters reject negative, fractional, and string values during decode.
Unknown fields are refused so a newer identity meaning cannot be silently lost
on save. The tests cover every known field, omitted defaults, malformed values,
full-width counters, and exact model-to-record-to-model restoration. Integrating
the record into schema-2 package loading, migrations, and atomic saving remains
open.
