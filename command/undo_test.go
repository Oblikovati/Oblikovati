// SPDX-License-Identifier: GPL-2.0-only

package command

import "testing"

func TestUndoThenRedoReturnsToIdenticalState(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	d.SetDisplayName("base")

	tx := h.Begin("Rename to final")
	_ = tx.Do(Rename(d, "final"))
	_ = tx.Commit()
	if d.DisplayName() != "final" {
		t.Fatal("edit not applied")
	}

	if err := h.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if d.DisplayName() != "base" {
		t.Errorf("after undo name = %q, want base", d.DisplayName())
	}
	if err := h.Redo(); err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if d.DisplayName() != "final" {
		t.Errorf("after redo name = %q, want final", d.DisplayName())
	}
}

func TestNewEditClearsRedoStack(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	a := h.Begin("A")
	_ = a.Do(Rename(d, "a"))
	_ = a.Commit()

	if err := h.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if !h.CanRedo() {
		t.Fatal("redo not available after undo")
	}

	b := h.Begin("B")
	_ = b.Do(Rename(d, "b"))
	_ = b.Commit()
	if h.CanRedo() {
		t.Error("a new edit did not clear the redo stack")
	}
}

func TestUndoRedoLabelsAndGuards(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	if err := h.Undo(); err == nil {
		t.Error("Undo on empty history did not error")
	}
	if err := h.Redo(); err == nil {
		t.Error("Redo on empty history did not error")
	}
	for _, n := range []string{"A", "B"} {
		tx := h.Begin(n)
		_ = tx.Do(Rename(d, n))
		_ = tx.Commit()
	}
	if got := h.UndoLabels(); len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Errorf("UndoLabels = %v, want [A B]", got)
	}
	_ = h.Undo()
	if got := h.RedoLabels(); len(got) != 1 || got[0] != "B" {
		t.Errorf("RedoLabels = %v, want [B]", got)
	}

	// Undo/redo must refuse while a transaction is open.
	open := h.Begin("open")
	if err := h.Undo(); err == nil {
		t.Error("Undo allowed with an open transaction")
	}
	if err := h.Redo(); err == nil {
		t.Error("Redo allowed with an open transaction")
	}
	_ = open.Abort()
}

func TestUndoRedoFireOneUpdateEach(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	tx := h.Begin("edit")
	_ = tx.Do(Rename(d, "x"))
	_ = tx.Commit()

	updates := 0
	h.OnChange(func() { updates++ })
	_ = h.Undo()
	_ = h.Redo()
	if updates != 2 {
		t.Errorf("undo+redo fired %d updates, want 2", updates)
	}
}
