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

File path identity/cache rules, typed package load/save orchestration, save-before-
close, new-document templates, references between documents, and workspace event
transport remain pending. This owner registers reconstructed state; it does not
claim that a whole feature model can open or that a live GUI exists. Save will
need to advance both the file identity and recipe baseline only after durable
replacement succeeds.
