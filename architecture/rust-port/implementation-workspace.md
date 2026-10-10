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

File path identity/cache rules, package-load orchestration, save-before-close,
new-document templates, references between documents, and workspace event
transport remain pending. This owner registers reconstructed state; it does not
claim that a whole feature model can open or that a live GUI exists.

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

The caller still binds the retained package and destination to its document.
Owning that binding, path conflicts, Save As, and save-before-close belong to the
pending file-workspace coordinator. Unsupported feature state remains retained
and unexecuted; this save path does not establish full application replay.
