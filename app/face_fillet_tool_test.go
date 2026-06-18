// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/feature"
)

// pickFaceFilletSets drives the two-set face pick on a fresh 2×2×2 block: the top face into set A
// and the +X side face into set B (they share one edge), returning the tool and block.
func pickFaceFilletSets(t *testing.T) (*Session, *FaceFilletTool) {
	t.Helper()
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	tool := NewFaceFilletTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, block), Body: block}) // set A active by default
	tool.ArmSetB()
	tool.Pick(s, FaceHandle{Face: plusXFaceOf(t, block), Body: block})
	return s, tool
}

// TestFaceFilletToolEndToEnd drives the Face Fillet UI: pick a face into each set, set the radius,
// OK — and asserts a valid solid that rounds the shared edge (a cylinder face, volume below the box).
func TestFaceFilletToolEndToEnd(t *testing.T) {
	s, tool := pickFaceFilletSets(t)
	tool.SetRadius(0.3)
	if !tool.CanCommit() {
		t.Fatal("face fillet not ready after two sets + radius")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if _, ok := tool.AddedFeature().Definition().(*feature.FaceFilletFeature); !ok {
		t.Fatalf("added feature is %T, want *FaceFilletFeature", tool.AddedFeature().Definition())
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("face-filleted body not a valid solid: %+v", r)
	}
	cyls := 0
	for _, fc := range body.Faces() {
		if _, ok := fc.Geometry().(geom.Cylinder); ok {
			cyls++
		}
	}
	if cyls != 1 {
		t.Errorf("face-filleted body has %d cylinder faces, want 1 (the rounded shared edge)", cyls)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; v >= 8 {
		t.Errorf("face fillet did not round material: volume %g, want < 8", v)
	}
}

// TestFaceFilletToolActiveSet checks picks land in the armed set and a face is never double-counted.
func TestFaceFilletToolActiveSet(t *testing.T) {
	_, tool := pickFaceFilletSets(t)
	if tool.ActiveSet() != 1 {
		t.Errorf("active set = %d after ArmSetB, want 1", tool.ActiveSet())
	}
	if tool.CountA() != 1 || tool.CountB() != 1 {
		t.Fatalf("counts = (A:%d, B:%d), want (1, 1)", tool.CountA(), tool.CountB())
	}
	// Re-picking a face already in set A must be ignored even while set B is armed.
	tool.Pick(nil, tool.Faces()[0])
	if tool.CountB() != 1 {
		t.Errorf("re-picking a set-A face added to B: CountB = %d, want 1", tool.CountB())
	}
	tool.ArmSetA()
	if tool.ActiveSet() != 0 {
		t.Errorf("active set = %d after ArmSetA, want 0", tool.ActiveSet())
	}
}

// TestFaceFilletToolNeedsBothSets gates commit on two non-empty sets and a positive radius.
func TestFaceFilletToolNeedsBothSets(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	tool := NewFaceFilletTool()
	s.StartTool(tool)
	tool.SetRadius(0.3)
	if tool.CanCommit() {
		t.Error("ready with no faces picked")
	}
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, block), Body: block})
	if tool.CanCommit() {
		t.Error("ready with only set A picked")
	}
	tool.ArmSetB()
	tool.Pick(s, FaceHandle{Face: plusXFaceOf(t, block), Body: block})
	if !tool.CanCommit() {
		t.Error("not ready after both sets + radius")
	}
	tool.SetRadius(0)
	if tool.CanCommit() {
		t.Error("ready with a zero radius")
	}
}

// TestFaceFilletToolClearSets checks each set's clear empties only that set.
func TestFaceFilletToolClearSets(t *testing.T) {
	_, tool := pickFaceFilletSets(t)
	tool.ClearSetA()
	if tool.CountA() != 0 || tool.CountB() != 1 {
		t.Errorf("after ClearSetA, counts = (A:%d, B:%d), want (0, 1)", tool.CountA(), tool.CountB())
	}
	tool.ClearSetB()
	if tool.CountB() != 0 {
		t.Errorf("after ClearSetB, CountB = %d, want 0", tool.CountB())
	}
}

// TestFaceFilletEditSeed checks re-editing a committed face fillet seeds both sets' counts and the
// radius back into the panel.
func TestFaceFilletEditSeed(t *testing.T) {
	s, tool := pickFaceFilletSets(t)
	tool.SetRadius(0.3)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	pf := tool.AddedFeature()
	ff := pf.Definition().(*feature.FaceFilletFeature)
	et := editFaceFilletTool(pf, ff)
	if !et.IsEditing() {
		t.Error("editFaceFilletTool should bind edit mode")
	}
	if et.CountA() != 1 || et.CountB() != 1 || et.Radius() != 0.3 {
		t.Errorf("seeded edit = (A:%d, B:%d, r:%g), want (1, 1, 0.3)", et.CountA(), et.CountB(), et.Radius())
	}
}

// TestFaceFilletViaRibbonCommand checks the Face Fillet command (a Fillet split-button variant)
// starts the tool.
func TestFaceFilletViaRibbonCommand(t *testing.T) {
	s, _ := newPartWithBlock(t, 2)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.FaceFillet"); err != nil {
		t.Fatalf("execute Modify.FaceFillet: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*FaceFilletTool); !ok {
		t.Fatal("Face Fillet command did not start the face-fillet tool")
	}
}
