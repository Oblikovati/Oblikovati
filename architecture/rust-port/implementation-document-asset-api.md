# Document asset API reads — partial R5/R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
under ADR-0019/0020. Document asset storage is already part of the history and
package paths; its definitions now reach the four typed catalog read methods.

Materials/appearances List/Get use the active document's owned definitions over
the immutable built-in catalog. A document definition with the same ID replaces
the shared definition in the response, without mutating the catalog or another
document. With no active document, the complete built-in catalog still works.
Focus, close and reopen therefore select the relevant retained document snapshot.
No worker, script VM, or separate asset cache owns that state.

The model's `ScopedAssets` supplies display order and lookup priority together.
Shared catalog positions remain stable; an override appears once at the same
position, and new project/document IDs follow in their scope order. The host
uses that resolver rather than implementing another shadowing rule. API DTOs,
method constants and typed clients remain release-owned. Runtime project library
loading is still open, so the host's project scope is currently empty.

This extends the Go session-library read behavior to include portable document
snapshots, consistent with the Rust document-before-project-before-builtin
resolution decision. It is not a second global catalog assembled from all open
documents. Each request uses the owner's active document at that call boundary,
as other active-context API reads do. Asset IDs alone do not pin a document
across later focus changes; consumers needing stable document context must
control focus or use a future explicit-context API contract.

Tests cover scope-priority ordering, exact-ID de-duplication, overriding a shared
ID, appended document-only definitions, switching two documents with conflicting
definitions, closing their contexts, save/reopen through the typed file host, and
real CLI Lua reads with only the portable package present. Existing no-document
catalog and native add-in workflow tests remain required by `make gate`.

Public asset edits/assignment commands, project persistence, topology verification,
UI, render appearance consumption and physical analysis remain open. This change
adds no kernel or exchange implementation. Validation stays on local Linux;
Windows/macOS remain deferred, with no push or PR.

## Local validation receipt

On 2026-10-13, the final `make gate` passed for all 13 workspace crates with
llvmpipe and the local Vulkan validation layer. Model production coverage was
98.84%, automation 96.45%, and every crate exceeded 80%. Repository duplication
was 0.63%. Formatting, clippy,
build, tests, doctests, documentation, SPDX, workflow and architecture checks
passed, including the real CLI portable-asset read. The initial Clippy item-order
warning was fixed before this final run. The AMD native presentation fault
remains open.
