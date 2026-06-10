// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// TestWorkPlaneOffsetIsEditable: an offset plane exposes its distance via EditableScalars
// (the seam the edit tool, router, and serializer all drive) and re-derives when set, so the
// plane (and anything built on it) follows the edited value after recompute.
func TestWorkPlaneOffsetIsEditable(t *testing.T) {
	_, def := emptyPartSession(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	sc := wp.EditableScalars()
	if len(sc) != 1 || sc[0].Get() != 2 {
		t.Fatalf("offset plane EditableScalars = %+v, want one scalar reading 2", sc)
	}
	sc[0].Set(5)
	def.Recompute()
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), 1e-9) {
		t.Errorf("edited offset plane origin = %v, want (0,0,5)", wp.Plane().Origin())
	}
}

// TestOriginPlaneIsNotOffsetEditable: an origin coordinate-system plane has no offset to edit.
func TestOriginPlaneIsNotOffsetEditable(t *testing.T) {
	_, def := emptyPartSession(t)
	if sc := def.OriginPlanes()[0].EditableScalars(); len(sc) != 0 {
		t.Error("an origin plane must not report an editable offset")
	}
}

// TestBeginEditWorkPlaneOpensOffsetEditor: double-clicking a user offset plane opens the edit
// tool seeded with its current offset (in the document's mm), and OK writes the new distance;
// origin planes are a no-op.
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
	if s.ActiveWorkPlaneEdit() == nil {
		t.Fatal("editing a user offset plane should open the work-plane edit tool")
	}
	if s.EditPlaneScalarCount() != 1 || s.EditPlaneScalarLabel(0) != "Offset" {
		t.Fatalf("offset edit should expose one Offset scalar, got count=%d label=%q",
			s.EditPlaneScalarCount(), s.EditPlaneScalarLabel(0))
	}
	if got := s.EditPlaneScalarValue(0); got != 20 { // 2 cm shown as 20 mm
		t.Errorf("seeded offset = %v mm, want 20", got)
	}
	if _, active := s.EditScopeSeq(); !active {
		t.Error("editing a work plane should engage the edit scope")
	}

	s.SetEditPlaneScalarValue(0, 70) // 70 mm = 7 cm
	if err := s.OK(); err != nil {
		t.Fatalf("commit work-plane edit: %v", err)
	}
	if d := wp.EditableScalars()[0].Get(); d != 7 {
		t.Errorf("committed offset = %v, want 7 cm", d)
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
	s.SetEditPlaneScalarValue(0, 90)
	s.CancelTool()
	if d := wp.EditableScalars()[0].Get(); d != 2 {
		t.Errorf("offset after cancel = %v, want 2 (restored)", d)
	}
}

// TestRedefineThreePointPlaneViaPick: double-clicking a three-point plane opens a redefine with
// three point slots; arming one and feeding a point pick re-points the plane and recomputes.
func TestRedefineThreePointPlaneViaPick(t *testing.T) {
	s, def := emptyPartSession(t)
	a := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 0) })
	b := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 0, 0) })
	c := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 0) })
	wp := def.WorkPlanes().AddByThreePoints(a.Key(), b.Key(), c.Key())
	def.Recompute()

	s.BeginEditWorkPlane(WorkPlaneHandle{Plane: wp})
	tool := s.ActiveWorkPlaneEdit()
	if tool == nil {
		t.Fatal("a three-point plane should be redefinable")
	}
	if s.EditPlaneRefSlotCount() != 3 {
		t.Fatalf("three-point redefine slots = %d, want 3", s.EditPlaneRefSlotCount())
	}

	// Re-point the third corner above the plane → it tilts off +Z.
	up := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 1) })
	s.EditPlaneArmRefSlot(2)
	if !s.EditPlaneRefSlotArmed(2) {
		t.Fatal("slot 2 should be armed after EditPlaneArmRefSlot(2)")
	}
	tool.Pick(s, WorkPointHandle{Point: up})
	if wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), 1e-6) {
		t.Error("re-picking point 3 did not re-tilt the plane")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("commit redefine: %v", err)
	}
}

// TestRedefineCancelRestoresReference: cancelling a redefine restores the original references.
func TestRedefineCancelRestoresReference(t *testing.T) {
	s, def := emptyPartSession(t)
	a := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 0) })
	b := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 0, 0) })
	c := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 0) })
	wp := def.WorkPlanes().AddByThreePoints(a.Key(), b.Key(), c.Key())
	def.Recompute()
	want := wp.Plane().Normal().AsVector()

	s.BeginEditWorkPlane(WorkPlaneHandle{Plane: wp})
	up := def.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 1) })
	s.EditPlaneArmRefSlot(2)
	s.ActiveWorkPlaneEdit().Pick(s, WorkPointHandle{Point: up})
	s.CancelTool()
	def.Recompute()
	if !wp.Plane().Normal().AsVector().IsEqualTo(want, 1e-9) {
		t.Errorf("after cancel, plane normal = %v, want restored %v", wp.Plane().Normal(), want)
	}
}

// TestEditPlaneDialogAccessors covers the session seam the (untestable, cgo) head dialog
// renders from: the dialog title, slot labels, and per-unit scalar names for both the
// length (offset) and angle (line-plane-angle) cases.
func TestEditPlaneDialogAccessors(t *testing.T) {
	s, def := emptyPartSession(t)
	angle := def.WorkPlanes().AddByLinePlaneAndAngle(
		feature.OriginXAxis, feature.OriginXYPlane, func() float64 { return 0 })
	def.Recompute()

	if s.IsEditingWorkPlane() || s.EditPlaneName() != "" || s.EditPlaneScalarUnitName(0) != "" {
		t.Fatal("accessors must report empty state while no edit is open")
	}
	s.BeginEditWorkPlane(WorkPlaneHandle{Plane: angle})
	if !s.IsEditingWorkPlane() || s.EditPlaneName() != angle.Name() {
		t.Fatalf("dialog title = %q, want %q", s.EditPlaneName(), angle.Name())
	}
	if got := s.EditPlaneScalarUnitName(0); got != s.AngleUnitName() {
		t.Errorf("angle scalar unit = %q, want the document's angle unit %q", got, s.AngleUnitName())
	}
	if got := s.EditPlaneRefSlotLabel(0); got != "Line" {
		t.Errorf("slot 0 label = %q, want Line", got)
	}
	s.CancelTool()
}

// TestRedefineSelfPickRefused: picking the edited plane itself for its base slot is refused
// (it would self-reference and drift), surfaces a notice, and leaves the slot armed so the
// user can pick something else.
func TestRedefineSelfPickRefused(t *testing.T) {
	s, def := emptyPartSession(t)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	def.Recompute()

	s.BeginEditWorkPlane(WorkPlaneHandle{Plane: wp})
	s.EditPlaneArmRefSlot(0)
	s.ActiveWorkPlaneEdit().Pick(s, WorkPlaneHandle{Plane: wp})
	if s.Notice() == "" {
		t.Error("a refused self-reference pick must surface a notice")
	}
	if !s.EditPlaneRefSlotArmed(0) {
		t.Error("the slot must stay armed after a refused pick")
	}
	s.CancelTool()
	def.Recompute()
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 2), 1e-9) {
		t.Errorf("plane moved after a refused self-pick: origin = %v, want (0,0,2)", wp.Plane().Origin())
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
