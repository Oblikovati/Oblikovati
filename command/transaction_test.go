// SPDX-License-Identifier: GPL-2.0-only

package command

import (
	"errors"
	"testing"

	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
)

func newPart(t *testing.T) *doc.Document {
	t.Helper()
	ws := doc.NewWorkspace(nil, contentset.Default())
	d, err := ws.Add(doc.Part, "p.obk", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return d
}

func TestAbortRestoresPreTransactionState(t *testing.T) {
	d := newPart(t)
	d.SetDisplayName("original")
	h := NewHistory()

	tx := h.Begin("Edit batch")
	if err := tx.Do(Rename(d, "step1")); err != nil {
		t.Fatalf("Do: %v", err)
	}
	_ = tx.Do(SetVisibility(d, false))
	if d.DisplayName() != "step1" || d.Visible() {
		t.Fatal("transaction edits not applied during recording")
	}

	if err := tx.Abort(); err != nil {
		t.Fatalf("Abort: %v", err)
	}
	if d.DisplayName() != "original" || !d.Visible() {
		t.Errorf("abort did not restore state: name=%q visible=%v", d.DisplayName(), d.Visible())
	}
	if h.Len() != 0 {
		t.Errorf("aborted transaction left %d history steps, want 0", h.Len())
	}
	if tx.State() != Aborted {
		t.Errorf("state = %v, want Aborted", tx.State())
	}
}

func TestNestedTransactionCommitsIntoParent(t *testing.T) {
	d := newPart(t)
	h := NewHistory()

	parent := h.Begin("Parent")
	_ = parent.Do(Rename(d, "a"))
	child := h.Begin("Child")
	_ = child.Do(Rename(d, "b"))
	if err := child.Commit(); err != nil {
		t.Fatalf("child Commit: %v", err)
	}
	// Child commit must not create its own history step.
	if h.Len() != 0 {
		t.Fatalf("child commit pushed a history step; want it folded into parent")
	}
	if err := parent.Commit(); err != nil {
		t.Fatalf("parent Commit: %v", err)
	}
	if h.Len() != 1 {
		t.Fatalf("history len = %d, want 1 combined step", h.Len())
	}
	if labels := h.Labels(); labels[0] != "Parent" {
		t.Errorf("undo label = %q, want Parent", labels[0])
	}
}

func TestDisplayNamesAppearAsUndoLabels(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	for _, name := range []string{"First edit", "Second edit"} {
		tx := h.Begin(name)
		_ = tx.Do(Rename(d, name))
		_ = tx.Commit()
	}
	labels := h.Labels()
	if len(labels) != 2 || labels[0] != "First edit" || labels[1] != "Second edit" {
		t.Errorf("undo labels = %v, want [First edit, Second edit]", labels)
	}
}

func TestMergeWithPreviousUndoesAsOneStep(t *testing.T) {
	d := newPart(t)
	h := NewHistory()

	a := h.Begin("A")
	_ = a.Do(Rename(d, "a"))
	_ = a.Commit()

	b := h.Begin("B")
	b.MergeWithPrevious()
	_ = b.Do(Rename(d, "b"))
	_ = b.Commit()

	if h.Len() != 1 {
		t.Errorf("history len = %d, want 1 merged step", h.Len())
	}
}

func TestEditMidTransactionJoinsIt(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	tx := h.Begin("Combined")
	// A bare History.Do while a transaction is open routes into the transaction.
	if err := h.Do(Rename(d, "x")); err != nil {
		t.Fatalf("Do during transaction: %v", err)
	}
	if h.Len() != 0 {
		t.Fatal("mid-transaction Do created its own history step")
	}
	_ = tx.Commit()
	if h.Len() != 1 {
		t.Error("transaction did not produce one step after absorbing the edit")
	}
}

func TestSuppressionCoalescesBatchToOneUpdate(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	updates := 0
	h.OnChange(func() { updates++ })

	tx := h.Begin("Big batch")
	for i := 0; i < 1000; i++ {
		_ = tx.Do(Rename(d, "n"))
	}
	if updates != 0 {
		t.Fatalf("recording fired %d updates, want 0 before commit", updates)
	}
	_ = tx.Commit()
	if updates != 1 {
		t.Errorf("commit fired %d updates, want exactly 1 coalesced update", updates)
	}
}

func TestSuppressNotificationsGatesBareEdits(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	updates := 0
	h.OnChange(func() { updates++ })

	h.SuppressNotifications(true)
	for i := 0; i < 5; i++ {
		_ = h.Do(Rename(d, "n"))
	}
	if updates != 0 {
		t.Fatalf("suppressed edits fired %d updates, want 0", updates)
	}
	h.SuppressNotifications(false) // resume → one coalesced update
	if updates != 1 {
		t.Errorf("resume fired %d updates, want 1", updates)
	}
}

func TestApplyFailureRollsBackBatchPrefix(t *testing.T) {
	applied := 0
	ok := NewFunc("ok",
		func() error { applied++; return nil },
		func() error { applied--; return nil })
	boom := NewFunc("boom", func() error { return errors.New("boom") }, func() error { return nil })

	b := NewBatch("group", ok, ok, boom)
	if err := b.Apply(); err == nil {
		t.Fatal("batch Apply did not surface the failure")
	}
	if applied != 0 {
		t.Errorf("after failed batch, applied=%d; want the prefix rolled back to 0", applied)
	}
}

func TestClosedTransactionRejectsFurtherUse(t *testing.T) {
	d := newPart(t)
	h := NewHistory()
	tx := h.Begin("t")
	_ = tx.Commit()
	if err := tx.Do(Rename(d, "x")); err == nil {
		t.Error("Do on a committed transaction did not error")
	}
	if err := tx.Commit(); err == nil {
		t.Error("second Commit did not error")
	}
	if err := tx.Abort(); err == nil {
		t.Error("Abort on a committed transaction did not error")
	}
}
