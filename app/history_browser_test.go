// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"path/filepath"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// partSessionWithEdits opens an active part and records n parameter edits, returning the
// session and the document id. Each AddNumericUserParameter is one recording "Edit Parameters"
// step, so the stream has n entries at position n.
func partSessionWithEdits(t *testing.T, n int) (*Session, doc.ID) {
	t.Helper()
	s := NewSessionWithStore(newFakeDocStore())
	d, err := compdef.AddPart(s.Workspace(), "history.obk", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	trackFromHere(s)
	for i := range n {
		if err := s.AddNumericUserParameter("p"+string(rune('a'+i)), "1 cm"); err != nil {
			t.Fatalf("add parameter %d: %v", i, err)
		}
	}
	return s, d.ID()
}

// TestDocumentHistoryViewListsEverySince open lists every step since the document opened, with
// the cursor at the latest, regardless of which document is active.
func TestDocumentHistoryViewListsEverySinceOpen(t *testing.T) {
	s, id := partSessionWithEdits(t, 3)

	tl, ok := s.DocumentHistoryView(id)
	if !ok {
		t.Fatal("DocumentHistoryView(active part) = !ok, want a timeline")
	}
	if tl.Position != 3 || len(tl.Labels) != 3 {
		t.Fatalf("timeline position=%d labels=%v, want position 3 with 3 labels", tl.Position, tl.Labels)
	}
	for _, l := range tl.Labels {
		if l != "Edit Parameters" {
			t.Fatalf("label = %q, want Edit Parameters", l)
		}
	}
}

// TestJumpDocumentToNavigatesAbsolutely jumps the cursor to an absolute position in one call
// (multi-step undo), then forward again — the click-to-jump a long history needs.
func TestJumpDocumentToNavigatesAbsolutely(t *testing.T) {
	s, id := partSessionWithEdits(t, 4)

	if err := s.JumpDocumentTo(id, 1); err != nil {
		t.Fatalf("JumpDocumentTo(1): %v", err)
	}
	tl, _ := s.DocumentHistoryView(id)
	if tl.Position != 1 {
		t.Fatalf("after jump to 1: position=%d, want 1", tl.Position)
	}
	if len(tl.Labels) != 4 {
		t.Fatalf("jump truncated the stream to %d labels; a jump is non-destructive (want 4)", len(tl.Labels))
	}

	if err := s.JumpDocumentTo(id, 4); err != nil {
		t.Fatalf("JumpDocumentTo(4): %v", err)
	}
	if tl, _ := s.DocumentHistoryView(id); tl.Position != 4 {
		t.Fatalf("after jump to 4: position=%d, want 4", tl.Position)
	}
}

// TestJumpDocumentToRejectsOutOfRange guards the bounds.
func TestJumpDocumentToRejectsOutOfRange(t *testing.T) {
	s, id := partSessionWithEdits(t, 2)
	if err := s.JumpDocumentTo(id, 5); err == nil {
		t.Error("jump past the end should error")
	}
	if err := s.JumpDocumentTo(id, -1); err == nil {
		t.Error("jump before the baseline should error")
	}
}

// TestSaveAddsCheckpointWithoutErasingHistory pins the save-checkpoint behaviour: saving flags
// the current depth as a save point and leaves every prior step navigable.
func TestSaveAddsCheckpointWithoutErasingHistory(t *testing.T) {
	s, id := partSessionWithEdits(t, 2)
	d := s.ActiveDocument()

	if err := s.SaveDocumentAs(d, filepath.Join(t.TempDir(), "saved.obk")); err != nil {
		t.Fatalf("SaveDocumentAs: %v", err)
	}
	// One more edit after the save.
	if err := s.AddNumericUserParameter("after", "2 cm"); err != nil {
		t.Fatalf("post-save edit: %v", err)
	}

	tl, _ := s.DocumentHistoryView(id)
	if len(tl.Labels) != 3 {
		t.Fatalf("save erased history: %d labels, want all 3 preserved", len(tl.Labels))
	}
	if len(tl.SavedDepths) != 1 || tl.SavedDepths[0] != 2 {
		t.Fatalf("save checkpoints = %v, want one at depth 2", tl.SavedDepths)
	}
}
