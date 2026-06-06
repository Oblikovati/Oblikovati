// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/model/compdef"
)

// partOf returns the active part definition — the geometry/recipe under test.
func partOf(t *testing.T, s *Session) *compdef.PartComponentDefinition {
	t.Helper()
	def, ok := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if !ok {
		t.Fatal("active document is not a part")
	}
	return def
}

// trackFromHere opens the active document's transaction stream at the current recipe,
// the baseline the first edit will undo back to. NewPart/OpenDocument do this in the
// real app; the lower-level test setup (Workspace().Add) does not, so the stream-level
// tests call it explicitly to mark "the document is now open with this content".
func trackFromHere(s *Session) { s.documentHistory(s.ActiveDocument()) }

// TestUndoRedoExtrudeNavigatesGeometry is the headline test: an extrude is one
// transaction event; undo removes the solid (recompute from the prior recipe) while the
// sketch survives; redo restores the identical body. Undo/redo are cursor moves, not a
// geometry snapshot stack.
func TestUndoRedoExtrudeNavigatesGeometry(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	trackFromHere(s) // baseline = part + sketch, no body yet
	def := partOf(t, s)

	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(120, 90)
	ext.SetDistance(5)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after extrude: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	if !s.CanUndo() || s.UndoLabel() != "Extrude" {
		t.Fatalf("CanUndo=%v UndoLabel=%q, want true/\"Extrude\"", s.CanUndo(), s.UndoLabel())
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if def.SurfaceBodies().Count() != 0 {
		t.Errorf("after undo: %d bodies, want 0", def.SurfaceBodies().Count())
	}
	if def.Sketches().Count() != 1 {
		t.Errorf("undo removed the sketch (count=%d); only the extrude should be undone", def.Sketches().Count())
	}
	if !s.CanRedo() || s.RedoLabel() != "Extrude" {
		t.Fatalf("CanRedo=%v RedoLabel=%q, want true/\"Extrude\"", s.CanRedo(), s.RedoLabel())
	}

	if err := s.Redo(); err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Errorf("after redo: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	if z := def.SurfaceBodies().Item(0).RangeBox().Diagonal().Z; z < 4.99 || z > 5.01 {
		t.Errorf("redone extrude height = %v, want 5", z)
	}
}

// TestUndoParameterAndRedoTruncation covers the parameter edit path and the rule that a
// new edit made after undo truncates the forward (redo) branch of the stream.
func TestUndoParameterAndRedoTruncation(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	trackFromHere(s)
	def := partOf(t, s)
	base := def.Parameters().Count()

	if err := s.AddNumericUserParameter("d0", "10 mm"); err != nil {
		t.Fatalf("add param: %v", err)
	}
	if def.Parameters().Count() != base+1 {
		t.Fatalf("param count = %d, want %d", def.Parameters().Count(), base+1)
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if def.Parameters().Count() != base {
		t.Errorf("after undo: param count = %d, want %d", def.Parameters().Count(), base)
	}
	if !s.CanRedo() {
		t.Fatal("expected a redo branch after undo")
	}

	// A new edit while the cursor is behind the tip discards the redo branch.
	if err := s.AddNumericUserParameter("d1", "20 mm"); err != nil {
		t.Fatalf("add second param: %v", err)
	}
	if s.CanRedo() {
		t.Error("new edit did not truncate the redo branch")
	}
	if _, ok := def.Parameters().ByName("d0"); ok {
		t.Error("d0 should be gone — its event was discarded by the new edit")
	}
	if _, ok := def.Parameters().ByName("d1"); !ok {
		t.Error("d1 should be present")
	}
}

// TestCursorRoundTripFidelity: undoing k steps then redoing k restores the same logical
// model state — the cursor is a lossless navigator over the event stream. (We compare
// the parameter set rather than the raw recipe bytes because reset+apply reassigns
// internal IDs, so the marshaled form is not byte-stable; the state is what undo must
// preserve.)
func TestCursorRoundTripFidelity(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	trackFromHere(s)
	def := partOf(t, s)

	names := []string{"p1", "p2", "p3"}
	for i, expr := range []string{"1 mm", "2 mm", "3 mm"} {
		if err := s.AddNumericUserParameter(names[i], expr); err != nil {
			t.Fatalf("add %q: %v", names[i], err)
		}
	}
	tipCount := def.Parameters().Count()

	for i := 0; i < 3; i++ {
		if err := s.Undo(); err != nil {
			t.Fatalf("undo %d: %v", i, err)
		}
	}
	if got := def.Parameters().Count(); got != tipCount-3 {
		t.Fatalf("after undo×3: param count = %d, want %d", got, tipCount-3)
	}
	for i := 0; i < 3; i++ {
		if err := s.Redo(); err != nil {
			t.Fatalf("redo %d: %v", i, err)
		}
	}
	if got := def.Parameters().Count(); got != tipCount {
		t.Errorf("after redo×3: param count = %d, want %d", got, tipCount)
	}
	for _, n := range names {
		if _, ok := def.Parameters().ByName(n); !ok {
			t.Errorf("parameter %q missing after undo×3→redo×3; navigation is lossy", n)
		}
	}
}

// TestPerDocumentStreamsAreIsolated: each document owns its own stream — an edit in one
// is invisible to another, and undo acts on the active document only.
func TestPerDocumentStreamsAreIsolated(t *testing.T) {
	s := NewSession()
	a, err := s.NewPart()
	if err != nil {
		t.Fatalf("new part A: %v", err)
	}
	if err := s.AddNumericUserParameter("a0", "1 mm"); err != nil {
		t.Fatalf("edit A: %v", err)
	}

	b, err := s.NewPart() // B becomes active, with an empty stream
	if err != nil {
		t.Fatalf("new part B: %v", err)
	}
	if s.CanUndo() {
		t.Error("freshly created document B should have nothing to undo")
	}

	if err := s.Workspace().SetActiveDocument(a); err != nil {
		t.Fatalf("activate A: %v", err)
	}
	if !s.CanUndo() || s.UndoLabel() != "Edit Parameters" {
		t.Errorf("document A should still have its edit: CanUndo=%v label=%q", s.CanUndo(), s.UndoLabel())
	}
	_ = b
}

// TestUndoRedoKeyboardShortcuts checks the Ctrl+Z / Ctrl+Y / Ctrl+Shift+Z bindings the
// head forwards through PressKey, and that a bare "z" (no Ctrl) is not an undo.
func TestUndoRedoKeyboardShortcuts(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	trackFromHere(s)
	def := partOf(t, s)
	if err := s.AddNumericUserParameter("d0", "10 mm"); err != nil {
		t.Fatalf("add param: %v", err)
	}
	base := def.Parameters().Count()

	if err := s.PressKey(KeyEvent{Key: "z", Mods: CtrlMod}); err != nil { // Ctrl+Z → undo
		t.Fatalf("Ctrl+Z: %v", err)
	}
	if def.Parameters().Count() != base-1 {
		t.Errorf("after Ctrl+Z: count = %d, want %d", def.Parameters().Count(), base-1)
	}

	if err := s.PressKey(KeyEvent{Key: "y", Mods: CtrlMod}); err != nil { // Ctrl+Y → redo
		t.Fatalf("Ctrl+Y: %v", err)
	}
	if def.Parameters().Count() != base {
		t.Errorf("after Ctrl+Y: count = %d, want %d", def.Parameters().Count(), base)
	}

	if err := s.PressKey(KeyEvent{Key: "z", Mods: CtrlMod | ShiftMod}); err == nil {
		// Ctrl+Shift+Z is redo; nothing to redo here, so it returns the "nothing to redo"
		// error — which proves it routed to redo, not undo.
		t.Error("Ctrl+Shift+Z should have nothing to redo and error, but succeeded")
	}

	// A bare "z" is a command-alias lookup, never an undo.
	if err := s.PressKey(KeyEvent{Key: "z"}); err != nil {
		t.Fatalf("bare z: %v", err)
	}
	if def.Parameters().Count() != base {
		t.Errorf("bare z changed the model (count=%d); only Ctrl+Z should undo", def.Parameters().Count())
	}
}
