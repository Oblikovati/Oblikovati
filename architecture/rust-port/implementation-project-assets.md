# Project asset library loading — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0019/0020. Go `model/material/store.go` loads `appearances.yaml` and
`materials.yaml` from a project's design-data directory.

Rust project registration now reads and admits both files through the existing
injected package-reader boundary. No design-data location means no library I/O.
Missing files mean empty libraries. Permission, transport and parse failures are
named errors; they are not treated as first-run absence. Both candidates must
succeed before a project is registered or the active selection can change.
Already loaded canonical project identities retain their admitted snapshot.

`ProjectAssetLibrary` owns typed project-source definitions with unique IDs.
The document recipe decoder now shares definition admission with project files,
including all released API groups, null-list handling and legacy appearance
conversion. There is one unique-ID collection constructor; separate catalog and
project duplicate loops were removed. A file containing the wrong asset family
is refused instead of silently dropping definitions.

The typed catalog API now fills its project scope from the selected project.
Document definitions remain authoritative, project definitions replace built-ins,
and built-ins remain available. Selection does not copy project values into the
shared catalog or change an open document's owned definitions. Asset lists retain
one visible definition per ID in the existing catalog/scope order.

Named reader tests cover complete files, absent locations/files, source ownership,
legacy conversion, duplicate IDs, permission/parse/nonfinite failures and wrong
file shape. Workspace and typed-client tests cover registration atomicity,
project switching, visible project-only assets, failure retaining the prior
project, and a document snapshot retaining its definition after selection changes.
Memory filesystems now report missing fixtures with `NotFound`, matching the
real filesystem contract rather than an undifferentiated error.

The model/persistence tests and active-project lifecycle tests run with the
ordinary project and query owners. Run `cargo test -p oblikovati-persistence
--test project_assets`, the model project-assets tests, and `make gate`.

Library editing, persistent mutation, explicit reload, native project-picker
attachment, public project-control APIs, material assignments, UI, rendering and
analytic mass consumption remain open. This is project library read integration,
not complete project feature acceptance. No kernel/exchange code is added;
Windows/macOS remain deferred, with no push or PR.

## Local validation receipt

On 2026-10-13, `make gate` passed for all 13 workspace crates with llvmpipe and
the local Vulkan validation layer. Model production coverage was 98.84%,
persistence 96.65%, session 98.98%, and every crate exceeded 80%. Repository
duplication was 0.63%. Formatting,
clippy, build, tests, doctests, documentation, SPDX, workflow and architecture
checks passed. Five library-reader tests, model admission, typed API project
switching/failure tests, and the full active-project lifecycle suite passed.
The AMD native presentation fault remains open.
