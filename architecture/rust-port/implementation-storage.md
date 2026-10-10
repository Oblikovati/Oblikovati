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
