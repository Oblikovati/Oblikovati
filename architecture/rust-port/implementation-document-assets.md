# Document asset history and storage — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
[ADR-0020](../decisions/ADR-0020-documents-and-storage.md) supersedes ADR-0014 for
the complete storage boundary and adds occurrence-aware asset records.
[ADR-0019](../decisions/ADR-0019-materials-and-appearances.md) owns assignment
meaning. Historical accepted records remain unchanged.

`DocumentAssets` owns non-built-in physical and visual snapshots plus assignment
intent. Embedding copies a project asset into document source, including every API
property group and optional vector. Built-ins remain references. Document asset
sets compare by identity and value rather than insertion order.

`DocumentSnapshot` now carries this state. Create and open restore it before
history registration; snapshot save captures it in `model.materials`. History,
refresh and dirty-state guards share one metadata comparison. An assignment-only
edit enters undo/redo, cannot be discarded as unchanged, and requires save.
The lifecycle regression exposed the old dirty guard's missing asset comparison;
it was corrected in that common predicate. Failure preserves bytes, identity and
dirty state, and retry can commit the same intended assignment.

The codec reads modern Go material recipe names and flat assignment maps. Empty
and null collections restore empty state. It refuses duplicate asset IDs,
ambiguous equivalent reference forms, invalid keys/paths/asset IDs, nonfinite
values and malformed records before adoption. Scalar legacy appearances are a
named conversion prerequisite, not zero-filled OpenPBR records.

Occurrence maps retain the persistent reference and occurrence path in a structured
key and require schema 3. Part references keep Go's string-key representation.
The candidate promotes its envelope when occurrence context is present. Export
restores and compares all typed state before it can replace the retained package.
Opaque extensions survive known edits and clearing. Clearing a previously stored
optional vector removes its old value. All other recipe sections remain retained.

Tests cover complete typed definition and assignment round trips, project-free
resolution, deterministic occurrence encoding, legacy flat references, nullable
records, duplicate and malformed input, extension preservation, optional clearing,
snapshot commit/abort/undo/redo/refresh, normal file-owner create/save/reopen, and
write failure/retry using the existing named memory filesystem. A test observer
that samples width uses a valid parameter template rather than an empty document.
The shared snapshot fixtures now initialize the new owned domain explicitly.

Run `cargo test -p oblikovati-persistence --test document_assets`,
`cargo test -p oblikovati-model --test document_assets`, and the session lifecycle
suite, then `make gate` for full local Linux acceptance.

Public asset mutation/assignment commands, project library integration, topology
reference/incidence verification, lost-reference health, legacy appearance
conversion, renderer consumption, UI controls and analytic mass properties remain
open. Stored opaque keys do not certify live geometry. No kernel or exchange code
is added, Windows/macOS remain deferred, and no push or PR is included.

## Local validation receipt

On 2026-10-13, the final `make gate` passed for all 13 workspace crates with
llvmpipe and the local Vulkan validation layer. Model production coverage was
98.84%, persistence 96.66%, and all crates exceeded 80%. Repository duplication
was 0.63%. The four model tests, nine codec tests and two file-owner lifecycle
regressions passed. Formatting, clippy, build, tests, doctests, documentation,
SPDX, workflow and architecture checks passed. The first gate's nested-condition
lint was corrected before this final run. The AMD native presentation fault
remains open.
