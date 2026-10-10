# Document creation — partial R2

## Documents without storage paths

[#3650](https://github.com/Oblikovati/Oblikovati/issues/3650) adds
`FileWorkspace::create_unbound`. This is a prerequisite for the Go-compatible
`documents.create` API: a document name is not a file path.

A retained document can now own a validated recipe, fresh identity, title,
transaction history, and dirty state without a storage binding. Two documents
with the same title remain independent. Creation does not perform file I/O.
Existing creation from an explicit destination or template keeps its behavior.

Save and close-with-save report that Save As is required. Save As commits the
package before assigning the first path. Failure preserves the unbound state,
identity, title, and dirty state. The existing path-change observer receives an
empty previous path on the first binding. Save Copy As produces an independent
file without binding or cleaning the original. Batch Save reports an unbound
item as a per-item failure; batch Save As and Save Copy As accept such sources.

Session tests check independent identity, discard, refused Save, successful and
failed first Save As, saved recipe reopening, copy identity, and batch binding.
The full Linux gate covers the existing Rust crates. Windows and macOS testing
remain deferred, with their pipelines unchanged.

## Remaining creation work

Connect the canonical `documents.create` request to document-kind construction
and an injected observer factory. Port subtype registration and flavor setup;
do not accept a subtype without its required behavior. Parameter and property
recipes exist, but component definitions, sketches, features, and kernel-backed
geometry still require implementation. This file-owner change does not establish
full document creation parity or a graphical application.
