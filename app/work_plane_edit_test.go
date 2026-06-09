// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// TestWorkPlaneOffsetIsEditable: an offset plane exposes its distance and re-derives when set,
// so the plane (and anything built on it) follows the edited value after recompute.
func TestWorkPlaneOffsetIsEditable(t *testing.T) {
	_, def := emptyPartSession(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	if d, ok := wp.OffsetDistance(); !ok || d != 2 {
		t.Fatalf("OffsetDistance = (%v,%v), want (2,true)", d, ok)
	}
	if !wp.SetOffsetDistance(5) {
		t.Fatal("SetOffsetDistance should succeed for an offset plane")
	}
	def.Recompute()
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), 1e-9) {
		t.Errorf("edited offset plane origin = %v, want (0,0,5)", wp.Plane().Origin())
	}
}

// TestOriginPlaneIsNotOffsetEditable: an origin coordinate-system plane has no offset to edit.
func TestOriginPlaneIsNotOffsetEditable(t *testing.T) {
	_, def := emptyPartSession(t)
	if _, ok := def.OriginPlanes()[0].OffsetDistance(); ok {
		t.Error("an origin plane must not report an editable offset")
	}
}

// TestBeginEditWorkPlaneOpensOffsetEditor: double-clicking a user offset plane opens the edit
// tool seeded with its current offset, and OK writes the new distance; origin planes are a no-op.
func TestBeginEditWorkPlaneOpensOffsetEditor(t *testing.T) {
	s, def := emptyPartSession(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	// An origin plane is not editable: no tool starts.
	s.BeginEditWorkPlane(WorkPlaneHandle{Plane: def.OriginPlanes()[0]})
	if s.ActiveWorkPlaneEdit() != nil {
		t.Fatal("origin plane double-click must not open an edit")
	}

	s.BeginEditWorkPlane(WorkPlaneHandle{Plane: wp})
	tool := s.ActiveWorkPlaneEdit()
	if tool == nil {
		t.Fatal("editing a user offset plane should open the work-plane edit tool")
	}
	if tool.Distance() != 2 {
		t.Errorf("edit tool seeded distance = %v, want 2", tool.Distance())
	}
	if _, active := s.EditScopeSeq(); !active {
		t.Error("editing a work plane should engage the edit scope")
	}

	tool.SetDistance(7)
	if err := s.OK(); err != nil {
		t.Fatalf("commit work-plane edit: %v", err)
	}
	if d, _ := wp.OffsetDistance(); d != 7 {
		t.Errorf("committed offset = %v, want 7", d)
	}
	if _, active := s.EditScopeSeq(); active {
		t.Error("edit scope must clear after committing the work-plane edit")
	}
}

// TestBeginEditWorkPlaneCancelRestoresOffset: cancelling restores the pre-edit distance.
func TestBeginEditWorkPlaneCancelRestoresOffset(t *testing.T) {
	s, def := emptyPartSession(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	s.BeginEditWorkPlane(WorkPlaneHandle{Plane: wp})
	s.ActiveWorkPlaneEdit().SetDistance(9)
	s.CancelTool()
	if d, _ := wp.OffsetDistance(); d != 2 {
		t.Errorf("offset after cancel = %v, want 2 (restored)", d)
	}
}

// TestBeginEditSketchEntersEnvironment: double-clicking a sketch enters the sketch environment.
func TestBeginEditSketchEntersEnvironment(t *testing.T) {
	s, def := extrudedBoxPart(t)
	sk := def.Sketches().Item(0)
	s.BeginEditSketch(SketchHandle{Sketch: sk})
	if !s.InSketch() || s.ActiveSketch() != sk {
		t.Error("BeginEditSketch should enter the sketch environment on the given sketch")
	}
}
