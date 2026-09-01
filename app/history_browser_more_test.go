// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"path/filepath"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// TestHistoryBrowserWindowState covers the open/close/toggle accessors.
func TestHistoryBrowserWindowState(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.HistoryBrowserOpen() {
		t.Fatal("a fresh session should not have the History Browser open")
	}
	s.OpenHistoryBrowser()
	if !s.HistoryBrowserOpen() {
		t.Fatal("OpenHistoryBrowser did not open it")
	}
	s.ToggleHistoryBrowser()
	if s.HistoryBrowserOpen() {
		t.Fatal("ToggleHistoryBrowser did not close it")
	}
	s.ToggleHistoryBrowser()
	s.CloseHistoryBrowser()
	if s.HistoryBrowserOpen() {
		t.Fatal("CloseHistoryBrowser did not close it")
	}
}

// TestRecordActiveEditAndAddInEdit records a delta made directly on the active part through the
// central seam and the add-in seam, covering both entry points and the resulting steps.
func TestRecordActiveEditAndAddInEdit(t *testing.T) {
	t.Parallel()
	s, id := partSessionWithEdits(t, 1)
	part := partOf(t, s)

	// A mutation applied directly to the recipe is recorded as one step by RecordActiveEdit.
	if _, err := part.Parameters().AddUserParameter("viaActive", "5 cm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	s.RecordActiveEdit("Active Edit")
	if tl, _ := s.DocumentHistoryView(id); tl.Position != 2 || tl.Labels[1] != "Active Edit" {
		t.Fatalf("after RecordActiveEdit: %+v, want position 2 ending in Active Edit", tl)
	}

	if _, err := part.Parameters().AddUserParameter("viaAddIn", "6 cm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	s.RecordAddInEdit(part, "AddIn Edit")
	if tl, _ := s.DocumentHistoryView(id); tl.Position != 3 || tl.Labels[2] != "AddIn Edit" {
		t.Fatalf("after RecordAddInEdit: %+v, want position 3 ending in AddIn Edit", tl)
	}
}

// TestEnsureActiveEditBaselineOpensStream covers the active-document branch: it opens the stream
// so the very first edit is recorded against the empty baseline.
func TestEnsureActiveEditBaselineOpensStream(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "baseline.obk", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	s.EnsureActiveEditBaseline()
	if err := s.AddNumericUserParameter("w", "4 cm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	if !s.CanUndo() {
		t.Fatal("first edit after EnsureActiveEditBaseline must be undoable")
	}
}

// TestUndoRedoRecordSeamsNoActiveDocument covers the no-active-document guards of the seams.
func TestUndoRedoRecordSeamsNoActiveDocument(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.RecordActiveEdit("noop")   // no active document — must not panic
	s.EnsureActiveEditBaseline() // ditto
	if err := s.Undo(); err == nil {
		t.Error("Undo with no active document should error")
	}
	if err := s.Redo(); err == nil {
		t.Error("Redo with no active document should error")
	}
}

// TestJumpDocumentToGuards covers the error branches: unknown document and an open transaction.
func TestJumpDocumentToGuards(t *testing.T) {
	t.Parallel()
	s, id := partSessionWithEdits(t, 1)

	if err := s.JumpDocumentTo(doc.ID(999999), 0); err == nil {
		t.Error("jump on an unknown document should error")
	}
	if _, ok := s.DocumentHistoryView(doc.ID(999999)); ok {
		t.Error("DocumentHistoryView on an unknown document should report !ok")
	}

	if err := s.BeginTransaction("batch"); err != nil {
		t.Fatalf("BeginTransaction: %v", err)
	}
	if err := s.JumpDocumentTo(id, 0); err != ErrHistoryTransactionOpen {
		t.Errorf("jump with an open transaction = %v, want ErrHistoryTransactionOpen", err)
	}
	_ = s.AbortTransaction()
}

// TestSaveCheckpointDedupAndTruncation covers markSaved's same-depth dedup and
// savedDepthsWithin dropping a checkpoint stranded on a truncated branch.
func TestSaveCheckpointDedupAndTruncation(t *testing.T) {
	t.Parallel()
	s, id := partSessionWithEdits(t, 2)
	d := s.ActiveDocument()

	if err := s.SaveDocumentAs(d, filepath.Join(t.TempDir(), "a.obk")); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.SaveDocumentAs(d, filepath.Join(t.TempDir(), "b.obk")); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	if tl, _ := s.DocumentHistoryView(id); len(tl.SavedDepths) != 1 {
		t.Fatalf("two saves at the same depth = %v, want one checkpoint (deduped)", tl.SavedDepths)
	}

	// Roll back below the checkpoint and edit: the redo branch (with the depth-2 save) is
	// truncated, so the stranded checkpoint must drop out of the view.
	if err := s.JumpDocumentTo(id, 0); err != nil {
		t.Fatalf("jump to 0: %v", err)
	}
	if err := s.AddNumericUserParameter("fresh", "1 cm"); err != nil {
		t.Fatalf("post-rollback edit: %v", err)
	}
	if tl, _ := s.DocumentHistoryView(id); len(tl.SavedDepths) != 0 {
		t.Fatalf("stranded checkpoint survived truncation: %v, want none", tl.SavedDepths)
	}
}
