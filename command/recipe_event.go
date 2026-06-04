// SPDX-License-Identifier: GPL-2.0-only

package command

// RecipeStore is the seam a [RecipeEvent] navigates: something that can replace its
// entire recipe with a previously-captured snapshot and recompute the geometry that
// snapshot implies. A part's component definition satisfies it (MarshalRecipe captures
// the snapshot the event stores; RestoreRecipe replaces+recomputes from it). Declared
// here — and injected — so the command engine stays decoupled from the model packages
// (no import of model/compdef; the dependency flows the other way).
type RecipeStore interface {
	// RestoreRecipe replaces the store's recipe (parameters, sketches, features, …)
	// with the snapshot in model and recomputes the resulting geometry. It must be a
	// full replace, not a merge, so navigating to any snapshot yields exactly that
	// state regardless of what the store currently holds.
	RestoreRecipe(model []byte) error
}

// RecipeEvent is one transaction event in a document's edit stream: it remembers
// the recipe snapshot before and after a single interaction. Navigating the stream
// (undo/redo) restores the corresponding snapshot and recomputes — so the current
// model state is always the fold of the events up to the cursor, and moving the
// cursor is O(1) restore rather than an O(n) replay from the document's open state.
//
// The mutation has already happened in the model when the event is built (the app
// edits, then captures before/after), so the event is recorded via [History.Record]
// (not Do): Apply re-establishes the after-snapshot for redo, Revert re-establishes
// the before-snapshot for undo.
type RecipeEvent struct {
	label  string
	before []byte
	after  []byte
	store  RecipeStore
}

// NewRecipeEvent builds a snapshot event from the before/after recipe bytes captured
// around an interaction. before is the recipe as it was when the document reached
// this point; after is the recipe the interaction produced.
//
// Example:
//
//	before, _ := def.MarshalRecipe()
//	mutate()                          // the interaction
//	after, _ := def.MarshalRecipe()
//	history.Record(command.NewRecipeEvent("Extrude", before, after, def))
func NewRecipeEvent(label string, before, after []byte, store RecipeStore) *RecipeEvent {
	return &RecipeEvent{label: label, before: before, after: after, store: store}
}

// Label returns the event's undo-menu text.
func (e *RecipeEvent) Label() string { return e.label }

// Apply restores the after-snapshot and recomputes — the redo direction.
func (e *RecipeEvent) Apply() error { return e.restore(e.after) }

// Revert restores the before-snapshot and recomputes — the undo direction.
func (e *RecipeEvent) Revert() error { return e.restore(e.before) }

// restore replaces the store's recipe with one snapshot (which recomputes the geometry
// it implies).
func (e *RecipeEvent) restore(snapshot []byte) error {
	return e.store.RestoreRecipe(snapshot)
}
