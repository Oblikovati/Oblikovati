// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// minusXFaceOf returns the body's −X-facing planar face.
func minusXFaceOf(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).X < -0.9 {
			return f
		}
	}
	t.Fatal("no -X face found")
	return nil
}

// pickFullRoundSets drives the three picks on a fresh 2×2×2 block: the −X face into Side 1, the top
// face into Center, and the +X face into Side 2 (the top sits between the two parallel sides).
func pickFullRoundSets(t *testing.T) (*Session, *FullRoundFilletTool) {
	t.Helper()
	s, block := newPartWithBlock(t, 2) // 2×2×2, vol 8
	tool := NewFullRoundFilletTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{Face: minusXFaceOf(t, block), Body: block}) // Side 1 active by default
	tool.ArmCenter()
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, block), Body: block})
	tool.ArmSide2()
	tool.Pick(s, FaceHandle{Face: plusXFaceOf(t, block), Body: block})
	return s, tool
}

// TestFullRoundToolEndToEnd drives the Full Round UI: pick the three faces, OK — and asserts a valid
// solid that rounded the center face (volume below the box, the top corners gone).
func TestFullRoundToolEndToEnd(t *testing.T) {
	s, tool := pickFullRoundSets(t)
	if !tool.CanCommit() {
		t.Fatal("full round not ready after all three sets")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if _, ok := tool.AddedFeature().Definition().(*feature.FullRoundFilletFeature); !ok {
		t.Fatalf("added feature is %T, want *FullRoundFilletFeature", tool.AddedFeature().Definition())
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("full-round body not a valid solid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-3}).Volume; v >= 8 {
		t.Errorf("full round did not remove the top corners: volume %g, want < 8", v)
	}
}

// TestFullRoundToolActiveSet checks picks land in the armed set and a face is never double-counted.
func TestFullRoundToolActiveSet(t *testing.T) {
	_, tool := pickFullRoundSets(t)
	if tool.ActiveSet() != 2 {
		t.Errorf("active set = %d after ArmSide2, want 2", tool.ActiveSet())
	}
	if tool.Count1() != 1 || tool.CountCenter() != 1 || tool.Count2() != 1 {
		t.Fatalf("counts = (%d, %d, %d), want (1, 1, 1)", tool.Count1(), tool.CountCenter(), tool.Count2())
	}
	tool.Pick(nil, tool.Faces()[0]) // a face already in Side 1: ignored even while Side 2 is armed
	if tool.Count2() != 1 {
		t.Errorf("re-picking a Side-1 face added to Side 2: Count2 = %d, want 1", tool.Count2())
	}
	tool.ArmSide1()
	if tool.ActiveSet() != 0 {
		t.Errorf("active set = %d after ArmSide1, want 0", tool.ActiveSet())
	}
}

// TestFullRoundToolNeedsThreeSets gates commit until all three sets have a face.
func TestFullRoundToolNeedsThreeSets(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	tool := NewFullRoundFilletTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{Face: minusXFaceOf(t, block), Body: block})
	tool.ArmCenter()
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, block), Body: block})
	if tool.CanCommit() {
		t.Error("ready with only two sets")
	}
	tool.ArmSide2()
	tool.Pick(s, FaceHandle{Face: plusXFaceOf(t, block), Body: block})
	if !tool.CanCommit() {
		t.Error("not ready after all three sets")
	}
}

// TestFullRoundToolClearSets checks each set's clear empties only that set.
func TestFullRoundToolClearSets(t *testing.T) {
	_, tool := pickFullRoundSets(t)
	tool.ClearCenter()
	if tool.CountCenter() != 0 || tool.Count1() != 1 || tool.Count2() != 1 {
		t.Errorf("after ClearCenter, counts = (%d, %d, %d), want (1, 0, 1)",
			tool.Count1(), tool.CountCenter(), tool.Count2())
	}
	tool.ClearSide1()
	tool.ClearSide2()
	if tool.Count1() != 0 || tool.Count2() != 0 {
		t.Errorf("after clearing sides, counts = (%d, %d), want (0, 0)", tool.Count1(), tool.Count2())
	}
}

// TestFullRoundToolPromptAndName covers the name, the step-by-step prompt, the session accessor,
// and the non-edit Cancel path.
func TestFullRoundToolPromptAndName(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	tool := NewFullRoundFilletTool()
	s.StartTool(tool)
	if tool.Name() != "Full Round Fillet" {
		t.Errorf("Name = %q, want Full Round Fillet", tool.Name())
	}
	if s.ActiveFullRoundFillet() != tool {
		t.Error("ActiveFullRoundFillet did not return the running tool")
	}
	if got := tool.Prompt(s); got != "Click the first side face" {
		t.Errorf("prompt with no faces = %q", got)
	}
	tool.Pick(s, FaceHandle{Face: minusXFaceOf(t, block), Body: block})
	if got := tool.Prompt(s); got != "Arm Center, then click the face to round" {
		t.Errorf("prompt with Side 1 only = %q", got)
	}
	tool.ArmCenter()
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, block), Body: block})
	if got := tool.Prompt(s); got != "Arm Side 2, then click the opposite side face" {
		t.Errorf("prompt with Side 1 + Center = %q", got)
	}
	tool.ArmSide2()
	tool.Pick(s, FaceHandle{Face: plusXFaceOf(t, block), Body: block})
	if got := tool.Prompt(s); got != "Click OK to round the center face" {
		t.Errorf("prompt with all three = %q", got)
	}
	tool.Cancel(s) // non-edit: restores the default selection filter
}

// TestFullRoundDraftPreview offers the viewport draft only once all three sets are picked.
func TestFullRoundDraftPreview(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	tool := NewFullRoundFilletTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{Face: minusXFaceOf(t, block), Body: block})
	if _, ok := tool.DraftFeature(s); ok {
		t.Error("draft offered with only one set")
	}
	tool.ArmCenter()
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, block), Body: block})
	tool.ArmSide2()
	tool.Pick(s, FaceHandle{Face: plusXFaceOf(t, block), Body: block})
	if _, ok := tool.DraftFeature(s); !ok {
		t.Error("no draft preview after all three sets")
	}
}

// TestFullRoundEditRoundTrip re-edits a committed full round and asserts the three sets seed back,
// then re-commits and cancels an edit (covering commitEdit and the edit-cancel branch).
func TestFullRoundEditRoundTrip(t *testing.T) {
	s, tool := pickFullRoundSets(t)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	pf := tool.AddedFeature()
	fr := pf.Definition().(*feature.FullRoundFilletFeature)
	et := editFullRoundFilletTool(pf, fr)
	if !et.IsEditing() || et.Count1() != 1 || et.CountCenter() != 1 || et.Count2() != 1 {
		t.Fatalf("seeded edit = (editing:%v, %d, %d, %d), want (true, 1, 1, 1)",
			et.IsEditing(), et.Count1(), et.CountCenter(), et.Count2())
	}
	s.StartTool(et)
	if err := s.OK(); err != nil {
		t.Fatalf("commit edit: %v", err)
	}
	// A fresh edit that is cancelled must leave the feature healthy.
	ce := editFullRoundFilletTool(pf, pf.Definition().(*feature.FullRoundFilletFeature))
	s.StartTool(ce)
	ce.Cancel(s)
	if !pf.Health().OK() {
		t.Error("cancelling the edit should leave the feature healthy")
	}
}

// TestFullRoundViaRibbonCommand checks the Full Round command (a Fillet split-button variant) starts
// the tool.
func TestFullRoundViaRibbonCommand(t *testing.T) {
	s, _ := newPartWithBlock(t, 2)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.FullRoundFillet"); err != nil {
		t.Fatalf("execute Modify.FullRoundFillet: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*FullRoundFilletTool); !ok {
		t.Fatal("Full Round command did not start the full-round tool")
	}
}
