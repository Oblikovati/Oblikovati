# Document edit history implementation evidence

Tracking: [#3612](https://github.com/Oblikovati/Oblikovati/issues/3612).
This is partial R2 evidence for
[ADR-0004](../decisions/ADR-0004-session-and-feature-evaluation.md).

## Owned transactions

`oblikovati_model::history::DocumentEditHistory` owns the current parameter and
property snapshot. An edit runs on a separate candidate. A failed edit leaves
the current snapshot and evaluation generation unchanged. No public mutable
reference bypasses this path. These are owned recipe values; this implementation
does not assume that a cloned Martensite handle isolates kernel state.

Transactions nest. Only the active top frame can accept edits, commit, or abort.
A child commit folds into its parent. A parent abort also discards changes from
committed children. A top-level commit makes one named undo step; merge with the
previous step retains its original label and starting snapshot. A new commit
clears redo. Abort preserves existing undo and redo steps. Undo labels run from
oldest to newest; redo labels follow the order of subsequent redo operations.
Navigation is refused while a transaction is open.

Tokens contain a checked monotonic sequence and an opaque owner identity.
Equal sequences from different documents cannot address one another. Closed
tokens cannot address later frames. Owner identity is used only for ownership
checks, never for recipe ordering, persistence, naming, or geometry decisions.

The ID allocator stays outside all snapshots. Commands can reserve an ID but
cannot replace or rewind the allocator. Each new parameter must use an ID
reserved in that edit; invented IDs and previously unused reservations are
refused. Failed edits, abort, and undo do not release reservations. Successful
edits, abort, undo, and redo advance a checked evaluation generation. Commit
does not change that generation because it does not change the current recipe.
Exhaustion is refused before a mutation callback or history navigation occurs.

## Evidence and remaining work

Model integration tests cover nested commit and abort, merged steps, multiple
undo/redo labels, redo invalidation, owned parameter/property snapshots, failed
edits, permanent reservations, invalid IDs, stale and foreign tokens, and open
navigation guards. Internal boundary tests cover token and generation overflow.
The fixed Go baseline is `aa8297d1b0ac33c024f19e54821536d76e8a447a`;
its complete `go test ./command` suite passes. The source comparison uses
`command/transaction.go` and `command/undo.go`.

Checkpoint navigation, history retention, and worker result adoption remain
unimplemented. Snapshot scope must grow with the feature and sketch model.
This evidence does not prove geometry isolation, event parity, or a live GUI.
The current snapshots prioritize owned state; history memory scaling still
needs the planned retention and delta work.

## Session transaction lifecycle — partial R2

Tracking: [#3613](https://github.com/Oblikovati/Oblikovati/issues/3613).
`oblikovati-session` adds one `DocumentSession` owner per document. Bounded
groups share one model transaction: nested end closes one level, while abort
cancels the whole group, as in Go `app/undo.go`. Callers explicitly begin, edit,
and end a group. The edit path refreshes exposed properties in the same candidate
as parameter changes. A command error or failed refresh adopts neither change.

The injected `TransactionObserver` sees read-only snapshots. Commit, undo, and
redo have a Before phase that can refuse the move with a reason. A vetoed commit
aborts the group and reports Aborted after rollback; vetoed navigation preserves
the current state and cursor. Successful cursor movement produces one change
notification followed by its After observation. Explicit abort has only an
After observation. Close consumes the owner and reports Deleted without veto.
Invalid navigation is refused before any observer is called. Commit checks
rollback capacity before asking policy, so a veto can restore the group.

Payloads, event strings, and transaction-point enums come directly from the
pinned Rust API. No transport DTO is declared in the session crate. The payload
mapping matches Go `addin/events/lifecycle_relay.go`: current for commit/abort,
previous for undo, next for redo, and unknown for a deleted stream. Before is an
in-process policy call; this does not claim remote add-in veto support.

Ten integration tests use a named recording observer to check payloads, order,
visible state, nesting, rollback, veto, failed command/property refresh, redo
replacement, and close. An internal test checks nesting-counter exhaustion.
The five Go lifecycle tests in `app/transaction_events_test.go` pass at the fixed
baseline. These tests cover model state, not kernel evaluation or a viewport.

Workspace registration/routing, automatic command groups, recipe no-op filtering,
event subscriptions and transport delivery, checkpoint navigation, and worker
result adoption remain pending. This low-level owner currently records explicit
empty groups, like Go `command.History`; Go session recipe-delta filtering has
not yet been ported. Close consumes this owner but does not yet unregister file
identity or a workspace entry. Those belong to the workspace coordinator.
