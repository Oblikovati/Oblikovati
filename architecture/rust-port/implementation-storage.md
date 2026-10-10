# Storage implementation evidence

Tracking: [#3592](https://github.com/Oblikovati/Oblikovati/issues/3592).
Branch: `rust-port`; no PR is requested.

## Atomic byte replacement — partial R2

`oblikovati-persistence` is a new GPL workspace crate. Its `AtomicFiles` and
`StagedFile` interfaces separate orchestration from the native filesystem.
`replace_file` performs stage, complete write, synchronization, then commit.
Every failed phase returns its destination, phase, and underlying error.
Dropping an uncommitted stage removes its temporary. No fallible operation
follows a successful replacement, so a reported success permits save adoption.

The native implementation wraps `tempfile` 3.27.0. It creates a unique sibling
temporary and uses its platform replacement operation. The checked dependency
source uses `MoveFileExW` with replacement on Windows; the application contains
no native unsafe block. Unix permissions match the Go writer's `0644`, applied
before synchronization. No parent directory is created implicitly.

Two named-fake tests cover operation order and failures at every phase. Five
native filesystem tests on Unix cover create/replace/empty contents, retaining
the old file while staging, drop cleanup, failed commit cleanup, invalid paths,
and permissions. The shared tests also run in the native CI target matrix; the
permission assertion is Unix-only. Real filesystem tests complement the fakes
because replacement semantics cannot be established by a mock alone.

This ports Go `persistence/io.go` byte replacement. It is not the schema-2 YAML
package reader/writer, version retention, or a concrete `DocumentSaveTarget`.
The caller still needs to encode a complete snapshot with its candidate identity.
File contents are synchronized before replacement; directory metadata is not
synchronized after it. Power-loss durability and failures after a committed
directory update need separate outcome/reconciliation design. No full document
save/load or crash-durability claim is made by these tests.

## Schema-2 YAML envelope — partial R2

Tracking: [#3593](https://github.com/Oblikovati/Oblikovati/issues/3593).
`DocumentPackage` retains the manifest and all 17 known root fields from Go
`yamlcodec.onDisk`. Model and resource content stays native YAML. The codec
privately wraps [serde_yaml_ng](https://github.com/acatton/serde-yaml-ng) 0.10.0;
no library node type escapes the persistence interface.

The reader requires one mapping with schema version 2 and an unsigned document
kind. It refuses ZIP, missing/unsupported schemas, duplicate keys, multiple YAML
documents, unknown root fields, and incorrect root section shapes. Nested
section values remain intact for their domain decoders. Their retention does
not certify feature support, resource encoding validity, or permission to edit
the recipe. Manifest-free DataIO containers are not supported by this reader.

The Go fixture was produced by `yamlcodec.MarshalDocument` at baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. It contains a native model with typed
numbers/booleans/quoted strings, UTF-8 and base64 resources, binary add-in data,
identity, byte attributes, and body names. A local Go → Rust → Go check compared
all decoded metadata and byte sections plus the parsed model with Go
`reflect.DeepEqual`; all matched. Six Rust tests cover the fixture, all section
paths, malformed input, alias values, and repeated encoding stability.

Preservation is semantic: comments, alias spelling, scalar style, and whitespace
can change on encoding. Input mapping order is retained; repeated encoding is
stable. The public package is read-only at this stage. Identity replacement,
recipe restoration, and the concrete atomic save target remain open work.

## Typed resources and data streams — partial R2

Tracking: [#3594](https://github.com/Oblikovati/Oblikovati/issues/3594).
Package decoding now validates and decodes resource and data tables before
returning a package. Read-only typed views expose exact stream bytes and resource
type, origin, and one classified content form: UTF-8 text, binary bytes, or an
application-provided embedded resource. Classification follows encoding, not
the extensible type tag. Empty, absent, and null tables remain empty.

Base64 decoding matches Go `StdEncoding`: standard alphabet, required padding
where needed, ignored CR/LF, and accepted nonzero unused padding bits. Spaces,
URL-safe alphabet substitutions, bad padding, and malformed input are refused
with the section name and value. Unknown resource encodings and malformed entry
shapes are refused. An embedded entry with a nonempty stored value is also
refused: Go discarded that value, which conflicts with lossless storage.

The original YAML nodes remain authoritative for re-encoding, so extension
metadata is retained even where the typed resource view does not interpret it.
Six new tests cover exact Go fixture bytes, metadata, encoding classification,
empty values, base64 compatibility, malformed resources, and preservation of
nested extension fields. The corresponding Go resource/data tests pass at the
fixed baseline. Host validation of resource type support, embedded-resource
resolution, recipe content, and save-target integration remain open.

## Complete-package atomic save target — partial R2

Tracking: [#3595](https://github.com/Oblikovati/Oblikovati/issues/3595).
`PackageSaveTarget` connects the package encoder, model-owned save protocol, and
injected atomic filesystem. It borrows the complete package snapshot exclusively,
computes its recipe digest, refuses mismatched candidate identity digests, and
replaces only the identity node in a temporary encoding. Existing identity
records are parsed and restored for validation before replacement; unknown
identity fields cannot be discarded by saving.

Atomic replacement writes all retained sections. Only after success does the
target adopt the identity node in the in-memory package, followed by the model
index's adoption of the same candidate. An encoding, validation, or filesystem
error leaves both owners unchanged. Typed identity-record access is available
for host restoration; absent and null identity sections return no record.

Seven integration tests cover native save/reopen and repeated saves, exact identity
agreement, nonidentity section preservation, fake storage failure, mismatched
digests before I/O, unknown/malformed identity fields, and absent/null recipes. Repeated
saves keep the model revision when the serialized recipe is unchanged, while
advancing the file revision and counter.

The digest uses this codec's extracted recipe serialization. Go and Rust YAML
formatting can differ, so the first Rust save may mint a model revision even
when recipe values are unchanged. Subsequent Rust round trips are stable.
This saves an existing package snapshot; live model-to-recipe export, supported
feature validation, dirty state, user events, and document path ownership are
not yet integrated. Atomic rename still has the durability limits noted above.

## Registered package-file loading — partial R2

Tracking: [#3596](https://github.com/Oblikovati/Oblikovati/issues/3596).
`load_registered` reads through the owned `PackageFileReader` interface. The
native implementation buffers the file and releases its handle. The loader
decodes the package, validates/restores its identity record, then registers the
identity. Read, codec, entropy, identity, and registration errors retain the
requested path and their source chain. No failed load creates a registration.

Files without an identity retain a freshly generated identity, as in Go. A
copied-file collision returns the package plus an explicit reassignment outcome;
the retained on-disk identity is updated only on a later successful save. This
keeps load free of file writes. The host must process the outcome before exposing
the document, including its dirty state and changed sketch namespace.

Six tests include native load → save → close → reopen with exact identity and
package equality, copied-file collisions, absent identity, invalid identity and
YAML, duplicate session IDs, read/entropy failure, and missing native files.
These establish the package/identity lifecycle only. Document path indexing,
already-open-file reuse, feature support checks, live recipe reconstruction,
reference loading, dirty state, and event delivery remain open. If later recipe
restoration fails, its host owner must close the provisional registration.

## Parameter reconstruction before registration — partial R2

Tracking: [#3608](https://github.com/Oblikovati/Oblikovati/issues/3608).
`load_parameter_package` reads a file snapshot and reconstructs the implemented
parameter domain using stored units before it registers the document identity.
It returns the retained package, explicit identity-registration outcome, live
parameter collection, and recipe ID allocator. Raw envelope loading and typed
parameter loading share the same read, identity validation, and registration
implementation.

A recipe error leaves both identity indexes untouched and consumes no identity
entropy. A later identity/registration error discards the reconstructed candidate.
Copied-file collisions still return the reassignment outcome for the host to
report and mark dirty. Four integration tests cover invalid domain records,
identity failures, collisions and duplicate session IDs, plus a native
load/edit/export/save/close/reopen cycle at a non-default working scale. The
native cycle checks saved identity, changed value, and retained add-in bytes.

This entry point reconstructs parameters and, after #3610, typed properties. Other model sections are
not executed or certified as feature geometry. Full host opening must include
all required model-domain validation before it exposes a document to the user.

## Project reference and template lookup — partial R2

Tracking: [#3624](https://github.com/Oblikovati/Oblikovati/issues/3624).
`ProjectFileLookup` owns validated absolute workspace, library, and template roots.
It searches the workspace before libraries in configured order and stops at the
first regular file. Absolute references bypass search roots. Missing references
return no match; inaccessible candidates return a path-specific error instead of
silently choosing a different file from a later root. The regular-file probe is
injected; its native implementation follows symlinks and rejects directories.

Templates use the canonical API document kinds and file extensions. Missing
template roots and unknown document kinds return no match without consulting the
process working directory. Relative configured roots are refused; project-file
loading must first resolve them against the project's location.

Relative references are computed by normalized path components. Parent segments
that leave the workspace are refused, while names such as `..cache` remain valid
inside it. This corrects the Go prefix check, which treats every `..`-prefixed
name as outside. Relativization is lexical, not a symlink containment or security
check. Opening a result still uses the file coordinator's canonical-path rules.

Eight tests cover search priority, missing/absolute names, access errors, all
four template names, root validation, relative references, and native Linux
regular-file/directory lookup. Six corresponding Go reference tests pass at the
fixed baseline. Project catalog ownership, configuration persistence, and reference
graph loading remain pending. Host template selection is connected in
[#3625](https://github.com/Oblikovati/Oblikovati/issues/3625), with evidence in the
[workspace implementation record](implementation-workspace.md).

## Named stream edits and cache compaction — partial R2

Tracking: [#3680](https://github.com/Oblikovati/Oblikovati/issues/3680).
`DocumentPackage::with_stream` copies a byte payload into one named stream.
`without_stream` removes it; an absent name is a no-op. Both keep the typed
stream view and YAML section consistent. Unmodified streams retain their stored
encoding. Stream names are opaque keys, including empty names and path-like
strings; no filesystem interpretation occurs here.

`without_caches` removes only the case-sensitive `cache/` prefix and reports
uncompressed bytes reclaimed, matching Go `Compact`. It edits one candidate
package and keeps model, attributes, identity, resources, and other streams.
Tests cover owned bytes, replacement/deletion, empty payloads, unusual names,
YAML round trips, prefix boundaries, and unchanged model digests. Go DataIO and
compaction tests provide reference evidence. Standalone manifest-free DataIO
files and host access to these mutations remain separate work.

## Standalone stream files — partial R2

Tracking: [#3681](https://github.com/Oblikovati/Oblikovati/issues/3681).
`stream_file::read_stream_file` and `write_stream_file` port Go DataIO file
operations through the existing injected package reader and atomic writer.
A missing destination creates a manifest-free container. Other read failures
and invalid YAML stop before staging. Writes preserve unrelated sections and
streams; a failed commit leaves the previous file intact. Missing streams return
an error that names the file and stream. Stream names remain opaque keys.

The manifest-free exception is private to these operations. Normal document
loading still requires a supported manifest. No document identity is registered,
and these operations do not coordinate writes with open session documents.
Host DataIO access must use the session's ownership rules when it is ported.

The committed `go-stream-container.yml` fixture was produced through Go
`WriteDataToFile` at `aa8297d1b0ac33c024f19e54821536d76e8a447a` with binary
`[0, 255]` and empty streams. Tests cover this fixture, native Linux round trips,
creation, replacement, section preservation, missing streams, read failures,
malformed input, and failed commit. The atomic writer's existing tests cover
staging, write, synchronization, and commit failure independently.

## Document interest registry and storage — partial R2

Tracking: [#3682](https://github.com/Oblikovati/Oblikovati/issues/3682).
`DocumentInterests` owns canonical API interest records in insertion order.
Identity is the exact `(client_id, name)` pair. Updating a pair retains its
position; removal preserves the other records. Discovery matches either client
ID or name, including withdrawn interests. Zero strength defaults to interested.
Client payload and signed 64-bit data version remain uninterpreted.

`DocumentPackage::restore_interests` reads the Go YAML spelling of interest
strengths. `with_interests` validates the old section before replacing it and
omits an empty registry. Other sections and the model digest remain unchanged.
The model rejects empty identities and unknown numeric strengths. Storage also
rejects duplicate pairs, unknown fields, unknown strength spellings, and invalid
versions. Unlike the Go loader, Rust does not silently turn an unknown strength
into interested or retain duplicate identities. An omitted strength still defaults
to interested for compatibility.

The Go fixture was generated with `Package.Save` at baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. Tests cover both strength values,
update order, removal, default strength, independent copies, exact payloads,
version limits, malformed records, and preservation of model/attribute/stream
sections. Go registry and package round-trip tests provide reference evidence.
Session dirty state, undo, file lifecycle integration, the four API methods,
and absent-client notices remain pending; the package envelope alone continues
to retain interest sections without interpreting them.

## Session ownership of document interests — partial R2

Tracking: [#3683](https://github.com/Oblikovati/Oblikovati/issues/3683).
`DocumentSnapshot` now owns interest records. Interest-only edits participate in
transaction change detection, undo/redo, abort, refresh, and save baselines.
An identical update is a no-op. A failed commit restores the previous registry.
Interest metadata does not change the model-only geometry digest.

Typed package loading validates interests before allocating or claiming identity.
Open and template creation install owned registries. Normal save and save-copy
capture the live registry; a failed write retains live metadata and identity.
Copy does not clear the source's dirty state. Tests exercise clean/dirty changes
through undo, normal/copy save and reopen, template independence, observer veto,
refresh generation, and rejection before identity allocation.

This completes the session and file lifecycle work identified above. The four
public interest methods and unavailable-client notices remain to be connected.

## Document interest API — partial R2

Tracking: [#3684](https://github.com/Oblikovati/Oblikovati/issues/3684).
The typed host now implements `documents.listInterests`, `hasInterest`,
`addInterest`, and `removeInterest` with canonical API requests and responses.
Queries use immutable access. Commands select the explicit document ID and pass
through transaction validation, veto, response encoding, and commit. They do not
change the active document. Zero or unknown IDs are refused.

List preserves insertion order and returns an empty array when no records exist,
matching Go. Discovery matches the exact client ID or interest name, including
withdrawn records. Add updates an existing pair in place. Removing a missing pair
is an error. Validation and observer veto leave state and history unchanged.

Four typed-client tests cover all methods, explicit document selection, invalid
and malformed requests, observer veto, undo, save, and reopen. The corresponding
Go router test supplies reference evidence. The shared inventory now contains
29 queries, 22 edits, and six transaction methods. Native transport and
unavailable-client notices are still pending.

## Imported resource edits — partial R2

Tracking: [#3685](https://github.com/Oblikovati/Oblikovati/issues/3685).
`DocumentPackage::with_resource` stores an owned typed resource and updates its
YAML entry. Encoding follows `ResourceContents`, independent of the extensible
resource type tag. Text stays UTF-8, binary bytes use standard padded base64,
and application-provided resources carry no payload. Changing encoding removes
obsolete payload fields. Clearing an origin removes the old origin field.

Replacement retains unknown extension fields on that resource and all other
resource entries. `without_resource` deletes one entry and removes an empty
resource table. An absent ID leaves the original representation unchanged.
Both operations preserve other sections and the model-only digest. IDs are
opaque keys; no file lookup or recipe reference rewrite occurs at this layer.

Four tests cover all encoding transitions, extension retention, owned inputs,
empty payloads, unusual IDs, removal, YAML round trips, and values matching the
committed Go resource fixture. The Go resource codec tests provide reference
evidence. Model ownership, reference validation, and application resource
resolution remain separate work; callers must coordinate recipe references
when replacing or removing an imported resource.

## Exact resource byte access — partial R2

Tracking: [#3686](https://github.com/Oblikovati/Oblikovati/issues/3686).
`DocumentPackage::resource_bytes` resolves an exact resource ID. UTF-8 and binary
resources return borrowed stored bytes, including valid empty payloads. Their
origin remains display metadata; no filesystem read or application lookup occurs.

Only `embedded` resources call the injected `ApplicationResources` provider with
the exact type and origin key. Its result owns the returned bytes. Missing IDs
and provider failures are explicit `ResourceReadError` values. Provider failures
retain the resource ID, type, origin, and original I/O cause. No path lookup,
installed-font substitution, retry, or alternate family selection occurs here,
as required by ADR-0014. The provider must serve exact application bundle content.

Four tests verify stored byte access after encode/decode, empty payloads,
provider call isolation, exact key forwarding, result ownership, missing IDs,
and missing/denied/invalid bundle errors. This ports the byte-access boundary of
Go `ResourceBytes` and the explicit bundle lookup in font resolution. It does
not port Go's family-name fallback. Bundled asset catalog ownership, font parsing,
and feature-engine use of this boundary remain pending.

## Imported resource identity and reuse — partial R2

Tracking: [#3687](https://github.com/Oblikovati/Oblikovati/issues/3687).
`PackageResource::imported` owns the supplied bytes and chooses UTF-8 only when
all bytes are valid UTF-8 and none is NUL, matching Go exchange import. Other
input remains binary. The caller supplies the display-only origin; no file is
read by this constructor.

`DocumentPackage::with_imported_resource` returns a candidate package and its
resource key. Equality includes type, encoding, origin, and bytes. An identical
resource reuses an existing key without consuming a GUID or changing retained
YAML fields. When legacy entries duplicate a payload, the lowest key in the
ordered map wins; existing entries and references remain intact. This makes the
Go map-based reuse rule deterministic.

New content takes one GUID from the injected source. Entropy failures and key
collisions return `ResourceImportError` without package mutation or retry.
The native source already produces random version-4 GUIDs. Four tests cover
encoding selection, owned inputs, all equality fields, reopen/reuse, duplicate
legacy keys, collision refusal, and entropy failure. Recipe creation and session
resource ownership remain pending; this operation does not create geometry.

## Session ownership of imported resources — partial R2

Tracking: [#3688](https://github.com/Oblikovati/Oblikovati/issues/3688).
Resource value types now live in `oblikovati_model::resource`; persistence
re-exports its existing `PackageResource` and `ResourceContents` names. The
model owns content without depending on the YAML codec. `DocumentSnapshot`
contains an owned resource table, included in edit/no-op detection, refresh,
undo/redo, abort, and dirty-state comparison.

Open and template creation install the decoded table. Normal save and copy
capture the live table through `with_resources`. Unchanged records retain their
stored representation. Changed records retain extension fields, removed IDs
leave the table, and an empty table is omitted. Capture preserves other package
sections and the current model-only digest contract. Feature operations must
coordinate resource changes with recipe references when that layer is ported.

Tests cover transaction veto, undo/redo/abort, no-op updates, refresh generation,
resource-only dirty state, normal/copy save and reopen, failed writes, and
independent template ownership. Package tests cover batch insertion/removal,
extension retention, and unchanged base64 text. Import identity, byte-resolution,
and Go fixture tests remain in the suite. Feature reconstruction and the
application bundle catalog are still pending.
