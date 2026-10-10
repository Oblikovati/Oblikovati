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

Session command dispatch, lifecycle events and veto, checkpoint navigation,
history retention, property refresh policy, and worker result adoption remain
unimplemented. Snapshot scope must grow with the feature and sketch model.
This evidence does not prove geometry isolation, event parity, or a live GUI.
The current snapshots prioritize owned state; history memory scaling still
needs the planned retention and delta work.
