# Workspace ownership implementation evidence

Tracking: [#3615](https://github.com/Oblikovati/Oblikovati/issues/3615).
This is partial R2 evidence for
[ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md) and
[ADR-0005](../decisions/ADR-0005-identity-and-references.md).

## Open ownership and routing

`WorkspaceSession` owns ordered document sessions and the persistent GUID index.
Registration takes an already validated history and file identity. Session IDs
increase with checked arithmetic; failed registrations consume their reservation,
and close never releases an ID for reuse. Failed identity generation leaves all
existing documents, GUID claims, focus, and lifecycle observations unchanged.

Foreground registration activates its document. Background registration preserves
focus, including an absent active document. Explicit activation rejects unknown
IDs before an observation. Named command access stays attached to its document
across tab switches. Active command access resolves focus when the command starts.
The returned `SessionCommands` capability forwards commands without exposing a
mutable owner reference that a caller could replace with another document.

Each document retains its own transaction stack and loaded recipe baseline.
Recipe edits require save; undo to the baseline clears that condition. GUID
collision reassignment always requires save, even when recipe history returns to
the baseline. Registration returns a must-use outcome and also reports it to the
workspace observer, so identity reassignment is not silently lost.

## Explicit discard-close

`close_discarding_changes` is deliberately explicit about discarding edits. It
asks the injected workspace policy before changing state. A veto preserves the
history, identity claim, open order, and focus. On acceptance it releases the
GUID claim and consumes the document session, which reports transaction-stream
deletion. The workspace then reports close with its resulting active document.
Closing the active document selects the first remaining document in open order,
matching Go `model/doc/workspace_query.go`. Closing another document preserves
focus. Reopening the original GUID after close can preserve that GUID, but uses
a new session ID.

Eight integration tests use named recording observers and GUID sources. They
cover focus, ordering, named transactions across focus switches, isolated
undo/redo, dirty state, collision notices and failures, close veto, identity
release, reopen, and invalid IDs. A boundary test proves session IDs cannot wrap
to zero. Go tests `TestActiveDocumentReassignedOnClose` and
`TestBackgroundOpenKeepsActiveAndHidden` pass at the fixed baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`.

New-document templates, references between documents, and workspace event
transport remain pending. This owner registers reconstructed
state; it does not claim that a whole feature model can open or that a live GUI
exists.

## Prepared snapshot saves — partial R2

Tracking: [#3616](https://github.com/Oblikovati/Oblikovati/issues/3616).
`WorkspaceSession::save` refuses missing documents and open transaction groups,
then asks workspace save policy before preparation or I/O. A veto preserves
ownership, identity, and dirty state. Accepted saves capture an owned recipe,
prepare its exact digest, prepare the next file identity, and persist the package.
Only successful storage permits identity adoption, replacement of the saved
recipe baseline, clearing collision dirtiness, and a Saved observation.

The model owns `DocumentSnapshotTarget`, extending its existing save boundary
with a non-writing preparation step. Persistence implements it with
`ParameterPackageTarget`. This adapter captures parameters and properties using
the collection's own unit context, retains other package sections, and checks
that exposed properties already match the snapshot. A failed preparation clears
any earlier candidate. A failed write changes neither the original package nor
workspace identity. Retry prepares a new candidate from the current recipe.

The adapter uses the existing atomic writer: stage beside the destination,
write the full contents, synchronize the file, then replace. No fallible step
follows successful replacement. Directory metadata durability after power loss
remains outside that writer's current guarantee. Undo after save compares dirty
state against the saved recipe; redo back to it is clean. Repeated saves of the
same recipe keep the database revision while advancing the file revision.

Nine integration tests use named storage and GUID fakes through the actual
package codec and workspace save path. They check written identity and recipe,
retained sections, undo/redo dirty state, retry, preparation and entropy failures,
stage and final replacement failures, stale properties, open/missing document
guards, save veto, and collision dirtiness. Go's save revision, failed-save
rollback, reopen, and save/open veto tests pass at the fixed baseline.

The lower-level save boundary accepts a prepared target. The file coordinator
below owns that binding for file-backed sessions. Unsupported feature state
remains retained and unexecuted; this save path does not establish full application
replay.

## File-backed workspace — partial R2

Tracking: [#3617](https://github.com/Oblikovati/Oblikovati/issues/3617).
`FileWorkspace` owns a retained package and canonical path for each session ID.
Ordinary save constructs its target internally from that binding, so a caller
cannot pair one live document with another document's retained package. The
coordinator exposes read-only workspace state and borrowed command capabilities.
Discard-close releases the path only after workspace policy accepts and the GUID
claim and transaction stream have been released.

New opens resolve an existing path through `DocumentFiles`, require an absolute
result, check open policy, reconstruct parameters and properties, restore identity,
then register ownership. Persistence's new `read_parameter_package` performs
typed preparation without claiming an identity. The existing registered loader
uses the same preparation path. Invalid recipes claim no path or session ID and
consume no identity entropy before recipe validation succeeds.

An already-open canonical path returns its live session without reloading or
discarding edits. An alias that resolves to that path also reuses it. Background
open preserves focus. A direct lookup of the stored canonical path can reuse
the session even if the external file is no longer present. Other aliases still
require successful filesystem resolution. Distinct canonical paths carrying one
GUID remain separate documents and use the existing visible reassignment rule.

`NativeDocumentFiles` wraps the package reader, atomic writer, and operating
system canonicalization. The application adds no case folding or path-separator
assumptions. This is canonical-path identity, not an inode or hard-link alias
index. External modification detection and explicit reload remain pending.

Eleven integration tests cover reuse, aliases, background focus, open veto,
malformed recipe retry, file-specific save destinations, identity collisions,
close/reopen, path/read/write failures, and close veto. A native temporary-file
test executes open, edit, atomic save, close, and reopen and checks both values
and file identity. The Go tests for already-open reuse, background reopen,
identity collision/release, and saved-document reopen pass at the fixed baseline.
This is file lifecycle evidence for the ported recipe domains, not a GUI or
feature-evaluation claim.

## Save As and save-before-close — partial R2

Tracking: [#3618](https://github.com/Oblikovati/Oblikovati/issues/3618).
`FileWorkspace::save_as` resolves the destination before asking save policy.
An open document cannot overwrite another open document's bound path. A save
to the current path is an ordinary save. A different path is adopted only after
atomic replacement succeeds: the GUID is preserved, the old file remains, and
the workspace reports the new path after updating its indexes. Any preparation,
policy, identity, or storage failure preserves the old binding and dirty state.

Native destination resolution accepts an existing regular file or a new file
under an existing canonical parent. It rejects directories, missing parents,
empty names, and dangling symlinks. It does not create parent directories or
apply case folding. Existing-file symlinks resolve to their actual target.

`close_saving_changes` refuses open transaction groups, asks close policy once,
saves only if the document is dirty, then releases identity, history, and file
binding. Save policy still applies. Close veto, save veto, and write failure keep
the document open. Clean close consumes no new revision and performs no write.
Explicit discard-close remains available as a separate caller choice.

Nine coordinator tests and native destination tests cover path conflicts, same-
path save, failed Save As, old-file retention, dirty/clean close, both veto phases,
open previews, and failed storage. A native Save As test writes a new path and
then closes cleanly. On Unix, a regression test proves a dangling symlink is not
overwritten as though its name were unused.

Go Save As, collision, close/save, and veto tests pass at the fixed baseline.
A direct failing-store check also confirms that Go changes its bound path before
the failing write. Rust deliberately corrects that behavior by committing the
path after storage, consistent with the atomic adoption rule.

## Save Copy — partial R2

Tracking: [#3619](https://github.com/Oblikovati/Oblikovati/issues/3619).
`FileWorkspace::save_copy` writes the current committed parameter/property recipe
with a fresh document GUID, fresh file revision, and save counter one. It retains
the database revision only when the prepared recipe has a known digest equal to
the source digest. Changed or unknown recipe digests require a fresh database
revision. The copy identity is not registered until the copy is opened.

Copy preparation, successful storage, and failed storage leave the source path,
identity, dirty baseline, retained package, and undo history unchanged. Open
transaction groups, destinations owned by any open document (including the
source), and generated GUIDs owned by another live document are refused.
The copy uses the same canonical destination resolution and atomic file writer
as Save As. No source-save or path-change event is emitted.

Optional canonical API `NewFileMetadata` supplies display-name and subtype
overrides. Empty fields inherit the original values. Explicit destinations take
precedence over metadata file/template fields, matching the Go router mapping.
Overrides affect only the copy package and do not change its recipe digest.

Seven coordinator tests cover clean and dirty copies, metadata isolation,
destination conflicts, open groups, identity failure, failed writes, and native
copy/open. Model tests cover matching, changed, and unknown digests and live GUID
collisions. The Go copy storage and identity tests pass at the fixed baseline.
Native copy/open confirms independent registration and unchanged source bytes.
This evidence covers the ported recipe domains; full feature replay,
new-document creation, and external modification detection remain pending.

## Batch save — partial R2

Tracking: [#3620](https://github.com/Oblikovati/Oblikovati/issues/3620).
`FileWorkspace::batch_save` consumes the canonical API `BatchSaveArgs` and returns
`BatchSaveResult`. It selects `save`, `saveAs`, or `saveCopyAs` once, validates all
source IDs, and resolves destinations before execution. Invalid operations,
missing sources, and duplicate resolved destinations refuse the whole batch
before writes. Distinct aliases of one canonical destination count as duplicates.

Each accepted item uses the existing atomic save transition. Results retain
request order, document IDs, destination names, and individual error messages.
One path, policy, identity, or storage failure does not stop later items. Success
counts include only completed writes. The batch is not an all-or-nothing disk
transaction: earlier successful files remain saved if a later item fails.

Ordinary save ignores requested target names and uses bound paths. A duplicate
source in that mode is refused because it would write the same destination twice.
This follows the API's ignored-target rule and tightens Go's literal-string
duplicate check. Native paths are never converted with lossy UTF-8 conversion;
an unrepresentable bound name is a per-item error before storage. An empty batch
returns an empty result array and zero successes.

Ten tests cover all three operations, order, policy and path failures, duplicate
aliases, missing sources, invalid operations, empty batches, failed writes,
native copy after a missing-parent failure, and Unix non-UTF-8 names. The request
is consumed by value; a retained `contract::BatchSave` facade and remote method
dispatch remain part of the automation host work.

## New document from a recipe package — partial R2

Tracking: [#3621](https://github.com/Oblikovati/Oblikovati/issues/3621).
`FileWorkspace::create` accepts an explicit destination and an already decoded
package. It reconstructs parameters and exposed properties, validates the
document kind and template identity, and registers a fresh unsaved identity.
Creation writes no file and emits Created followed by activation, rather than
Opened. The retained package supplies initial content and metadata; its saved
identity is replaced on the first save. The fresh identity cannot equal the
template GUID or any live document GUID.

The new document remains dirty even if edits are undone back to its initial
recipe. Its first successful save establishes the baseline and file revisions.
Save-before-close therefore writes an unchanged new document. A failed write
leaves it open and dirty. Discard-close releases it without writing. Creation
refuses an open destination, invalid recipe or identity, unsupported document
kind, or GUID failure without publishing a session or changing focus.

Eight tests cover creation events and focus, identity, undo-to-initial dirty
state, first save, failed close-save, discard, duplicate destinations, invalid
inputs, GUID reuse, and native create/save/reopen. The package may contain the
four known document kinds, but only the current parameter/property domains are
executed. Built-in content factories, template-file search, unnamed UI documents,
and full feature/assembly/drawing reconstruction remain pending.

## Explicit template files — partial R2

Tracking: [#3622](https://github.com/Oblikovati/Oblikovati/issues/3622).
`FileWorkspace::create_from_template` consumes canonical API `NewFileMetadata`.
It requires explicit `fileName` and `templateFile` paths, resolves both, and
refuses any destination that is open or resolves to the template itself. It
reads the saved template envelope through the injected file interface, applies
nonempty display-name/subtype overrides, then uses the validated creation path.
Empty overrides inherit the template's values. Creation writes neither file.

The template is not opened or registered as a document. An already-open template
keeps its identity and unsaved edits; those edits do not replace its saved file
as the source of the new recipe. The new document gets a fresh identity and stays
dirty until saved. Read, decode, recipe, path, or identity failures do not publish
a session. `read_package` exposes the persistence envelope reader without
claiming a document identity, so this path shares the ordinary loader's codec.

Six tests cover metadata inheritance and overrides, open-template isolation,
missing metadata paths, canonical aliases, read/codec failures, and native
template/create/first-save/reopen with unchanged template bytes. Path identity
uses canonical paths, as elsewhere in this coordinator; it is not an inode
index. Project template search, built-in content factories, subtype registration
and initialization, and remote `documents.create` dispatch remain pending.
