// SPDX-License-Identifier: GPL-2.0-only

package command

// RecipeStore is the seam a [RecipeEvent] navigates: something that can replace its
// entire recipe with a previously-captured snapshot and recompute the geometry that
// snapshot implies. A part's component definition satisfies it (MarshalSnapshot captures
// the snapshot the log stores; RestoreSnapshot replaces+recomputes from it). Declared
// here — and injected — so the command engine stays decoupled from the model packages
// (no import of model/compdef; the dependency flows the other way).
type RecipeStore interface {
	// RestoreSnapshot replaces the store's recipe (parameters, sketches, features, …)
	// with the snapshot bytes and recomputes the resulting geometry. It must be a full
	// replace, not a merge, so navigating to any snapshot yields exactly that state
	// regardless of what the store currently holds. The bytes are the undo stream's
	// internal snapshot format (not the on-disk recipe), paired with MarshalSnapshot.
	RestoreSnapshot(snapshot []byte) error
}

// SnapshotSource reconstructs the recipe snapshot recorded at a stream position. A
// [SnapshotLog] is the production implementation; a test can supply its own. Keeping it an
// interface lets [RecipeEvent] hold positions rather than bytes (Oblikovati#1424) without
// coupling the event to the log's delta encoding.
type SnapshotSource interface {
	At(position int) ([]byte, error)
}

// RecipeEvent is one transaction event in a document's edit stream. Rather than carry the
// recipe snapshot before and after the interaction (two full copies per edit, the O(N²)
// storage #1424 replaces), it records the BEFORE and AFTER positions in a shared
// [SnapshotLog] and reconstructs the bytes on demand. Navigating the stream (undo/redo)
// reconstructs the corresponding snapshot and recomputes — so the current model state is
// always the fold of the events up to the cursor, and moving the cursor is one restore.
//
// The mutation has already happened in the model when the event is built (the app edits,
// then appends the after-snapshot to the log), so the event is recorded via [History.Record]
// (not Do): Apply re-establishes the after-snapshot for redo, Revert re-establishes the
// before-snapshot for undo.
type RecipeEvent struct {
	label  string
	before int
	after  int
	snaps  SnapshotSource
	store  RecipeStore
}

// NewRecipeEvent builds a snapshot event from the before/after positions of an interaction
// in the shared snapshot log. before is the position the document held when it reached this
// point; after is the position the interaction produced.
//
// Example:
//
//	before := log.Next() - 1              // the current position
//	after := log.Append(afterSnapshot)    // record the new state
//	history.Record(command.NewRecipeEvent("Extrude", before, after, log, def))
func NewRecipeEvent(label string, before, after int, snaps SnapshotSource, store RecipeStore) *RecipeEvent {
	return &RecipeEvent{label: label, before: before, after: after, snaps: snaps, store: store}
}

// Label returns the event's undo-menu text.
func (e *RecipeEvent) Label() string { return e.label }

// Apply restores the after-snapshot and recomputes — the redo direction.
func (e *RecipeEvent) Apply() error { return e.restore(e.after) }

// Revert restores the before-snapshot and recomputes — the undo direction.
func (e *RecipeEvent) Revert() error { return e.restore(e.before) }

// restore reconstructs the snapshot at one stream position and replaces the store's recipe
// with it (which recomputes the geometry it implies). A reconstruction failure surfaces
// rather than restoring a wrong recipe.
func (e *RecipeEvent) restore(position int) error {
	snapshot, err := e.snaps.At(position)
	if err != nil {
		return err
	}
	return e.store.RestoreSnapshot(snapshot)
}
