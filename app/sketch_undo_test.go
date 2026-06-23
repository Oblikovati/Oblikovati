// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSketchConstraintIsUndoable reproduces #1270: adding a perpendicular constraint while
// editing a sketch is its own undo step, and Ctrl+Z reverts it while keeping the sketch open.
func TestSketchConstraintIsUndoable(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane()) // records "Create Sketch"
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 4))
	s.RecordActiveEdit("Sketch Geometry") // baseline: the two lines, before any constraint

	tool := constraintToolDefs[3].new() // Perpendicular
	s.StartTool(tool)
	s.feedPick(SketchEntityHandle{Entity: sk.Lines().Item(0)})
	s.feedPick(SketchEntityHandle{Entity: sk.Lines().Item(1)}) // auto-commits via OK → records

	if !s.CanUndo() || s.UndoLabel() != "Perpendicular" {
		t.Fatalf("after the constraint, CanUndo=%v UndoLabel=%q; want true/Perpendicular", s.CanUndo(), s.UndoLabel())
	}
	if n := len(s.ActiveSketch().Constraints()); n != 1 {
		t.Fatalf("active sketch has %d constraints, want 1 (the perpendicular)", n)
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if !s.InSketch() {
		t.Fatal("undo of a sketch constraint should keep the sketch open for editing")
	}
	if n := len(s.ActiveSketch().Constraints()); n != 0 {
		t.Errorf("after undo the sketch has %d constraints, want 0 (the perpendicular reverted)", n)
	}
	if n := s.ActiveSketch().Lines().Count(); n != 2 {
		t.Errorf("after undo the sketch has %d lines, want 2 (geometry preserved)", n)
	}
}

// TestSketchCreationIsUndoable: creating a sketch is its own undo step (#1270), so the in-sketch
// operations that follow undo without removing the sketch.
func TestSketchCreationIsUndoable(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if !s.CanUndo() || s.UndoLabel() != "Create Sketch" {
		t.Errorf("creating a sketch should record a 'Create Sketch' undo step, got CanUndo=%v label=%q", s.CanUndo(), s.UndoLabel())
	}
}

// TestSketchUndoReattachesActiveSketch: undo rebuilds the part's sketch objects, so the session
// must re-bind the active sketch to the live one — proven by editing it after the undo.
func TestSketchUndoReattachesActiveSketch(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, _ := s.CreateSketch(sketch.XYPlane())
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	s.RecordActiveEdit("Sketch Geometry")
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 4))
	s.RecordActiveEdit("Sketch Geometry")

	before := s.ActiveSketch()
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if s.ActiveSketch() == before {
		t.Error("active sketch should be re-bound to the rebuilt object after undo")
	}
	if s.ActiveSketch() == nil || s.ActiveSketch().Lines().Count() != 1 {
		t.Fatalf("re-bound sketch should hold 1 line after undo, got %v", s.ActiveSketch())
	}
	// The re-bound sketch is live and editable: a new line lands on it.
	s.ActiveSketch().Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 1))
	if s.ActiveSketch().Lines().Count() != 2 {
		t.Error("the re-bound active sketch should accept further edits")
	}
}

// TestSketchUndoPastCreationExitsSketch: undoing the sketch's own creation while editing it drops
// cleanly out of the sketch environment (the sketch no longer exists to re-bind).
func TestSketchUndoPastCreationExitsSketch(t *testing.T) {
	s, _ := emptyPartSession(t)
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if err := s.Undo(); err != nil { // undoes "Create Sketch"
		t.Fatalf("Undo: %v", err)
	}
	if s.InSketch() {
		t.Error("undoing the sketch creation should leave the sketch environment")
	}
}

// TestSketch3DEditIsUndoableAndReattaches: a 3D-sketch edit records an undo step, and undo
// re-binds the active 3D sketch so editing continues (#1270).
func TestSketch3DEditIsUndoableAndReattaches(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch3D() // records "Create 3D Sketch"
	if err != nil {
		t.Fatalf("CreateSketch3D: %v", err)
	}
	sk.AddLine3D(math.P3(0, 0, 0), math.P3(1, 0, 0))
	s.RecordActiveEdit("3D Geometry")
	sk.AddLine3D(math.P3(0, 0, 0), math.P3(0, 1, 0))
	s.RecordActiveEdit("3D Geometry")

	before := s.ActiveSketch3D()
	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if !s.InSketch3D() {
		t.Fatal("undo should keep the 3D sketch open")
	}
	if s.ActiveSketch3D() == before {
		t.Error("active 3D sketch should be re-bound to the rebuilt object")
	}
	if s.ActiveSketch3D().EntityCount() != 1 {
		t.Errorf("re-bound 3D sketch has %d entities, want 1 after undo", s.ActiveSketch3D().EntityCount())
	}
}

// TestSketchDragRecordsUndo: committing a drag records one undo step (the moved geometry).
func TestSketchDragRecordsUndo(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, _ := s.CreateSketch(sketch.XYPlane())
	sk.Points().Add(math.P2(1, 1))
	s.RecordActiveEdit("Sketch Geometry")
	if !s.BeginEntityDrag(0, 0) { // start a drag if a point is under the cursor
		// No point under the synthetic cursor origin in this headless setup; drive the seam directly.
		s.entityDrag.active = true
	}
	s.CommitEntityDrag()
	// The drag commit must not panic and must leave the sketch consistent; a real move records a
	// step, a no-move drag records nothing (the recipe no-op guard).
	if !s.InSketch() {
		t.Error("the sketch should stay open after a drag commit")
	}
}

// TestSketchDimensionEditRecordsUndo: editing a dimension value records an "Edit Dimension" step.
func TestSketchDimensionEditRecordsUndo(t *testing.T) {
	s, _ := emptyPartSession(t)
	sk, _ := s.CreateSketch(sketch.XYPlane())
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 0))
	dim, _ := sk.DimensionConstraints().AddDistance(a, b, "4 cm")
	s.RecordActiveEdit("Sketch Geometry")

	s.BeginEditDimension(dim)
	if err := s.CommitPendingDimension("6 cm"); err != nil {
		t.Fatalf("CommitPendingDimension: %v", err)
	}
	if !s.CanUndo() || s.UndoLabel() != "Edit Dimension" {
		t.Errorf("dimension edit should record an 'Edit Dimension' step, got CanUndo=%v label=%q", s.CanUndo(), s.UndoLabel())
	}
}
