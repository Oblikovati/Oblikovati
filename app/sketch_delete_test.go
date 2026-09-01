// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// selectSketchEntities replaces the selection with the given sketch entities (the state a
// user reaches by clicking them in the editor).
func selectSketchEntities(s *Session, ents ...sketch.Entity) {
	s.selection.Clear()
	for _, e := range ents {
		s.selection.Add(SketchEntityHandle{Entity: e})
	}
}

// TestDeleteSelectedSketchEntitiesRemovesThem is the regression guard for issue #1232:
// selecting sketch geometry and deleting it actually removes it from the active sketch.
func TestDeleteSelectedSketchEntitiesRemovesThem(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	keep := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	gone := sk.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(4, 1))
	selectSketchEntities(s, gone)

	if err := s.DeleteSelectedSketchEntities(); err != nil {
		t.Fatalf("DeleteSelectedSketchEntities: %v", err)
	}
	if sk.Lines().Count() != 1 || sk.Lines().Item(0) != keep {
		t.Errorf("after delete, lines = %d, want only the kept line", sk.Lines().Count())
	}
	if s.selection.Count() != 0 {
		t.Errorf("selection after delete = %d, want 0 (cleared)", s.selection.Count())
	}
}

// TestDeleteKeyDeletesSketchEntities proves the binding: the Delete key routed through the
// engine reaches DeleteSelectedSketchEntities and removes the selection (issue #1232).
func TestDeleteKeyDeletesSketchEntities(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	selectSketchEntities(s, l)

	if _, ok := s.Bindings().ResolveChord(keyEventToChord(KeyEvent{Key: "Delete"})); !ok {
		t.Fatal("the Delete key is not bound to any action")
	}
	if err := s.PressKey(KeyEvent{Key: "Delete"}); err != nil {
		t.Fatalf("PressKey(Delete): %v", err)
	}
	if sk.Lines().Count() != 0 {
		t.Errorf("after Delete key, lines = %d, want 0", sk.Lines().Count())
	}
}

// TestDeleteSelectedSketchEntitiesNoOps: outside a sketch, mid-tool, or with nothing
// selected, Delete does nothing and never errors — it must not destroy 3D geometry.
func TestDeleteSelectedSketchEntitiesNoOps(t *testing.T) {
	t.Parallel()
	s, def := emptyPartSession(t)

	// No active sketch.
	if err := s.DeleteSelectedSketchEntities(); err != nil {
		t.Fatalf("delete with no sketch: %v", err)
	}

	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))

	// Nothing selected.
	if err := s.DeleteSelectedSketchEntities(); err != nil || sk.Lines().Count() != 1 {
		t.Fatalf("delete with empty selection mutated the sketch (err=%v, lines=%d)", err, sk.Lines().Count())
	}

	// A tool is mid-operation: the selection must be left to the tool, not deleted.
	selectSketchEntities(s, l)
	s.StartTool(NewRectangleTool())
	if err := s.DeleteSelectedSketchEntities(); err != nil || sk.Lines().Count() != 1 {
		t.Fatalf("delete mid-tool mutated the sketch (err=%v, lines=%d)", err, sk.Lines().Count())
	}
}
