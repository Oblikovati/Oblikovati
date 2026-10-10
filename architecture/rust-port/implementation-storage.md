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
