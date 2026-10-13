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

## Recipe ID allocation — partial R2

Tracking: [#3588](https://github.com/Oblikovati/Oblikovati/issues/3588).
`RecipeIdAllocator` owns a monotonic nonzero `u64` namespace in a document.
It replaces reliance on a process-global wrapping counter for new Rust model
owners. One owner must serve all entities sharing that namespace. Allocation
reserves an ID even if a subsequent edit fails; deletion and undo must not
rewind the owner. It is deliberately not cloneable as an undo snapshot.

Restoration checks zero IDs, duplicates, and IDs above the saved reservation
mark. A saved mark keeps deleted IDs reserved even when no entities remain.
Legacy records without that mark derive it from the largest live ID, matching
the intent of Go sketch `raiseIDSeq`. This cannot recover IDs deleted before
the legacy file was saved; the source format did not retain that history.
Exhaustion is an explicit error and does not wrap or change the mark.

Six tests cover independent document namespaces, abandoned/deleted reservations,
legacy order independence, malformed restored IDs, full-width exhaustion, and
stable sketch UUID derivation after restoration. The storage owner must persist
the mark, and future sketch/parameter owners must use this allocator. Existing
parameter collections still accept caller-owned IDs. This component does not
yet establish an application-wide allocation or undo integration gate.

## Reference envelope codec — partial R2

Tracking: [#3589](https://github.com/Oblikovati/Oblikovati/issues/3589).
`ReferenceEnvelope` reads the legacy 16-byte header and scheme-1/2 `OKRK`
21-byte header. Context (`u64`), entity kind (`u32`), and payload length (`u32`)
are little-endian. Payload bytes are borrowed without interpretation or copying.
The decoder requires the declared length to equal all remaining bytes. It
rejects short headers, trailing bytes, and unsupported schemes before binding.
Unknown entity-kind numbers remain opaque, as in Go; decoding is not permission
to resolve an unsupported entity.

Encoding upgrades legacy input to scheme 1, preserves scheme 2, and retains the
identity triple exactly. [ADR-0021](../decisions/ADR-0021-semantic-topology-reference-keys.md)
defines the separate scheme-2 semantic payload. It checks
the payload width and the allocation limit before constructing the envelope.
This upgrades the container only; it cannot migrate Go lineage meaning into
Martensite. That requires the explicit semantic mapping in ADR-0005.

Normal decode classifies by the magic prefix and never retries malformed
versioned input as legacy. Go's comment claimed a legacy context could not
collide with that prefix, but the full `u64` range permits it. An explicit
`decode_legacy` entry point handles bytes whose legacy origin the storage owner
can establish. It must not be used as a fallback after a failed normal decode.
An ambiguous file without reliable format context still needs a migration rule.

Fifteen fixtures captured from Go `DecodeKey` and `Encode` at the fixed baseline
cover zero/full-width contexts and kinds plus binary payloads. Tests compare
both decoded schemes and exact re-encoded bytes, every truncation of a sample,
extra bytes, false lengths, unknown schemes, and the magic/context collision.
An allocation-size unit test uses lengths without allocating huge buffers.
[Exact reference binding](implementation-reference-binding.md) now covers
supported-kind admission and current-history lookup. The
[named prism adapter](implementation-named-prism.md) verifies construction provenance
for prism body and face names. General topology publication and malformed package
import remain separate integration gates.

## Open-document identity index — partial R2

Tracking: [#3590](https://github.com/Oblikovati/Oblikovati/issues/3590).
`DocumentIdentityIndex` owns identity records keyed by nonzero session IDs and
indexes their persistent GUIDs. Unique registrations preserve every field and
consume no randomness. Opening a copied file with a live GUID collision mints
one new document GUID, preserves the revision metadata, and returns an explicit
`IdentityRegistration::Reassigned` result. This matches the Go workspace rule
without hiding the identity change inside an index operation.

Generated GUIDs are checked against all live document GUIDs and the incoming
document's revision stamps. A repeated GUID or entropy error refuses the new
registration without changing either index. Duplicate session IDs are refused
before generation. Closing releases both claims, allowing the original file to
reopen with its saved GUID. Identity records are exposed read-only so callers
cannot change a GUID without maintaining the index.

Six tests cover unique registration, collision metadata preservation, atomic
failure, duplicate/zero session IDs, close/reopen, and error diagnostics. The
matching Go collision and close/reopen tests provide the source behavior.
This component does not load files, own recipe content, or dispatch host events.
The host must inspect the reassignment result, mark the incoming document dirty,
notify the user, and account for its changed sketch namespace before exposing
the document. Full workspace lifecycle remains open.

## Registered save adoption — partial R2

Tracking: [#3591](https://github.com/Oblikovati/Oblikovati/issues/3591).
The identity index now prepares a save candidate, passes it to an injected
`DocumentSaveTarget`, and adopts it only after that target returns success.
Preparation and storage errors leave both identity indexes unchanged. The
exclusive mutable borrow serializes this synchronous operation. Ordinary saves
preserve the document GUID, so adoption cannot invalidate the reverse lookup.

The target owns the recipe snapshot and must atomically persist it with the
candidate identity. Its caller supplies the digest for that same snapshot.
Six tests with a named fake target cover exact adoption, failed writes and
retry, unchanged and changed recipe revisions, unknown session IDs, and counter
overflow before storage. Failed writes consume generated revision IDs but do
not increment the registered save counter; retry uses fresh stamps.

This tests the storage boundary, not filesystem durability. The actual target,
package encoding, snapshot/digest agreement, and dirty/event state remain host
and persistence integration work. A storage implementation that writes a file
and then reports failure needs an explicit reconciliation policy; this layer
cannot infer whether an external write took effect.
