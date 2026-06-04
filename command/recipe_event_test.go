// SPDX-License-Identifier: GPL-2.0-only

package command

import "testing"

// fakeRecipeStore is a named in-memory RecipeStore (CLAUDE.md: mock I/O with named
// fakes). It records the snapshot it currently holds and counts restores, so a test
// can assert exactly which snapshot a stream cursor restored.
type fakeRecipeStore struct {
	current  string
	restores int
}

func (f *fakeRecipeStore) RestoreRecipe(model []byte) error {
	f.current = string(model)
	f.restores++
	return nil
}

func TestRecipeEventApplyRevertRestoreSnapshots(t *testing.T) {
	store := &fakeRecipeStore{current: "v1"}
	e := NewRecipeEvent("Edit", []byte("v1"), []byte("v2"), store)
	if e.Label() != "Edit" {
		t.Fatalf("Label = %q", e.Label())
	}
	if err := e.Revert(); err != nil {
		t.Fatalf("Revert: %v", err)
	}
	if store.current != "v1" {
		t.Errorf("after Revert current = %q, want v1", store.current)
	}
	if err := e.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if store.current != "v2" {
		t.Errorf("after Apply current = %q, want v2", store.current)
	}
	if store.restores != 2 {
		t.Errorf("restores = %d, want 2 (one per Apply/Revert)", store.restores)
	}
}

// TestRecordNavigatesStreamWithoutReapplying is the core event-stream property: the
// model is mutated first, the event is Recorded (not re-applied), and undo/redo move
// a cursor through the stream restoring the right snapshot each step.
func TestRecordNavigatesStreamWithoutReapplying(t *testing.T) {
	store := &fakeRecipeStore{current: "base"}
	h := NewHistory()

	// Two interactions: base→a, then a→b. The model already holds the result when
	// each event is recorded.
	store.current = "a"
	h.Record(NewRecipeEvent("first", []byte("base"), []byte("a"), store))
	store.current = "b"
	h.Record(NewRecipeEvent("second", []byte("a"), []byte("b"), store))

	if got := store.restores; got != 0 {
		t.Fatalf("Record must not restore; restores = %d", got)
	}
	if !h.CanUndo() || h.CanRedo() {
		t.Fatalf("after two records: CanUndo=%v CanRedo=%v", h.CanUndo(), h.CanRedo())
	}

	if err := h.Undo(); err != nil { // cursor: b -> a
		t.Fatalf("Undo: %v", err)
	}
	if store.current != "a" {
		t.Errorf("after one undo current = %q, want a", store.current)
	}
	if err := h.Undo(); err != nil { // cursor: a -> base
		t.Fatalf("Undo: %v", err)
	}
	if store.current != "base" {
		t.Errorf("after two undos current = %q, want base", store.current)
	}
	if h.CanUndo() {
		t.Error("cursor at start but CanUndo is true")
	}

	if err := h.Redo(); err != nil { // cursor: base -> a
		t.Fatalf("Redo: %v", err)
	}
	if store.current != "a" {
		t.Errorf("after redo current = %q, want a", store.current)
	}
}

// TestRecordTruncatesRedoTail: an edit made after undo drops the events ahead of the
// cursor (the redo branch), exactly like the done/Do path.
func TestRecordTruncatesRedoTail(t *testing.T) {
	store := &fakeRecipeStore{}
	h := NewHistory()
	h.Record(NewRecipeEvent("a", []byte("0"), []byte("1"), store))
	h.Record(NewRecipeEvent("b", []byte("1"), []byte("2"), store))
	_ = h.Undo() // cursor before "b"
	if !h.CanRedo() {
		t.Fatal("expected a redo branch after undo")
	}
	h.Record(NewRecipeEvent("c", []byte("1"), []byte("3"), store)) // new edit
	if h.CanRedo() {
		t.Error("new edit did not truncate the redo branch")
	}
	if want := []string{"a", "c"}; !equalStrings(h.UndoLabels(), want) {
		t.Errorf("UndoLabels = %v, want %v", h.UndoLabels(), want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
