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

## Typed API creation

[#3651](https://github.com/Oblikovati/Oblikovati/issues/3651) routes
`documents.create` through `FileHost` and the canonical Rust API DTOs. The
injected observer factory is shared with document open. Names must be nonempty;
kind names accept case changes and surrounding whitespace, as in the Go parser.
Part, assembly, drawing, and presentation requests create unbound recipe owners
with fresh identities and the exact supplied title. New owners become active,
visible, and dirty. Duplicate titles do not merge documents or claim paths.

The persistence owner builds an empty schema-2 envelope with typed metadata.
Titles containing YAML punctuation, Unicode, or newlines remain values. Creation
does not synthesize serialized YAML from user input. The envelope has no saved
identity; registration supplies the new identity through the existing owner.

Typed-client tests cover all four kinds, duplicate titles, active-document
selection, refusal before registration, and a part creation followed by parameter
editing, first Save As, close, and reopen. The Go
`TestDocumentsCreateListActivate` reference test also passes locally. These checks
cover recipe lifecycle behavior, not component-definition or geometry parity.

## Remaining creation work

Port subtype registration and flavor setup;
do not accept a subtype without its required behavior. Parameter and property
recipes exist, but component definitions, sketches, features, and kernel-backed
geometry still require implementation. This file-owner change does not establish
full document creation parity or a graphical application.

## Document unit preference edits

[#3653](https://github.com/Oblikovati/Oblikovati/issues/3653) connects
`documents.setUnits` to the shared document transaction owner. Supplied fields
update display units, precision, or length format; omitted fields retain their
values. Validation completes on a cloned unit context before installation.
Updates use the `Set Units` history label and support undo/redo, saved-state
comparison, and existing package unit persistence. A no-op adds no history.

The parameter owner refuses a working-scale change through this preference
operation. Existing parameter records, expressions, and stored values remain
unchanged, while formatted API values use the new display units. Tests check
atomic refusal, partial updates, history, and save/close/reopen.

Rust display precision currently uses `u8`. Negative values and values above 255
are refused with an explicit diagnostic; they are never wrapped. Go accepts a
larger nonnegative range. [#3654](https://github.com/Oblikovati/Oblikovati/issues/3654)
tracks the model, formatting, and persistence changes needed to remove this
compatibility gap.
