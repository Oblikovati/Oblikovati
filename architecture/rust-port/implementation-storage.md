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
