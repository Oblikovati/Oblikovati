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

func (f *fakeRecipeStore) RestoreSnapshot(snapshot []byte) error {
	f.current = string(snapshot)
	f.restores++
	return nil
}

// logWith seeds a snapshot log with the given position snapshots and returns it, the helper
// every event-stream test records its before/after positions against.
func logWith(snaps ...string) *SnapshotLog {
	l := NewSnapshotLog()
	for _, s := range snaps {
		l.Append([]byte(s))
	}
	return l
}

func TestRecipeEventApplyRevertRestoreSnapshots(t *testing.T) {
	store := &fakeRecipeStore{current: "v1"}
	log := logWith("v1", "v2") // positions 0 and 1
	e := NewRecipeEvent("Edit", 0, 1, log, store)
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
	log := logWith("base") // position 0 = open-state baseline

	// Two interactions: base→a, then a→b. The model already holds the result when
	// each event is recorded; each appends its after-snapshot to the shared log.
	store.current = "a"
	h.Record(NewRecipeEvent("first", 0, log.Append([]byte("a")), log, store))
	store.current = "b"
	h.Record(NewRecipeEvent("second", 1, log.Append([]byte("b")), log, store))

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
	log := logWith("0") // baseline
	h.Record(NewRecipeEvent("a", 0, log.Append([]byte("1")), log, store))
	h.Record(NewRecipeEvent("b", 1, log.Append([]byte("2")), log, store))
	_ = h.Undo() // cursor before "b"
	if !h.CanRedo() {
		t.Fatal("expected a redo branch after undo")
	}
	log.TruncateTo(2)                                                     // a new edit discards the redo branch position
	h.Record(NewRecipeEvent("c", 1, log.Append([]byte("3")), log, store)) // new edit
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
