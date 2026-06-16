// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestSheetMetalSessionAccessors each Active… accessor returns the running tool of its type
// and nil for the others.
func TestSheetMetalSessionAccessors(t *testing.T) {
	s, _ := sheetMetalSession(t)
	if s.ActiveSheetMetalFace() != nil {
		t.Error("no tool active, accessor should be nil")
	}
	// Starting one tool: its accessor is non-nil; a different one's is nil.
	s.StartTool(NewSheetMetalFlangeTool())
	if s.ActiveSheetMetalFlange() == nil {
		t.Error("flange accessor nil while the flange tool runs")
	}
	if s.ActiveSheetMetalHem() != nil {
		t.Error("hem accessor non-nil while the flange tool runs")
	}

	for _, c := range []struct {
		name  string
		start func()
		live  func() bool
	}{
		{"face", func() { s.StartTool(NewSheetMetalFaceTool()) }, func() bool { return s.ActiveSheetMetalFace() != nil }},
		{"hem", func() { s.StartTool(NewSheetMetalHemTool()) }, func() bool { return s.ActiveSheetMetalHem() != nil }},
		{"contourFlange", func() { s.StartTool(NewSheetMetalContourFlangeTool()) }, func() bool { return s.ActiveSheetMetalContourFlange() != nil }},
		{"loftedFlange", func() { s.StartTool(NewSheetMetalLoftedFlangeTool()) }, func() bool { return s.ActiveSheetMetalLoftedFlange() != nil }},
		{"contourRoll", func() { s.StartTool(NewSheetMetalContourRollTool()) }, func() bool { return s.ActiveSheetMetalContourRoll() != nil }},
		{"bend", func() { s.StartTool(NewSheetMetalBendTool()) }, func() bool { return s.ActiveSheetMetalBend() != nil }},
		{"fold", func() { s.StartTool(NewSheetMetalFoldTool()) }, func() bool { return s.ActiveSheetMetalFold() != nil }},
		{"corner", func() { s.StartTool(NewSheetMetalCornerTool()) }, func() bool { return s.ActiveSheetMetalCorner() != nil }},
		{"cornerSeam", func() { s.StartTool(NewSheetMetalCornerSeamTool()) }, func() bool { return s.ActiveSheetMetalCornerSeam() != nil }},
		{"cut", func() { s.StartTool(NewSheetMetalCutTool()) }, func() bool { return s.ActiveSheetMetalCut() != nil }},
		{"unfold", func() { s.StartTool(NewSheetMetalUnfoldTool()) }, func() bool { return s.ActiveSheetMetalUnfold() != nil }},
		{"refold", func() { s.StartTool(NewSheetMetalRefoldTool()) }, func() bool { return s.ActiveSheetMetalRefold() != nil }},
		{"style", func() { s.StartTool(NewSheetMetalStyleTool()) }, func() bool { return s.ActiveSheetMetalStyle() != nil }},
	} {
		c.start()
		if !c.live() {
			t.Errorf("%s accessor nil while its tool runs", c.name)
		}
	}
}

// TestSheetMetalPickAccessors PickCount tracks picks and ClearPicks resets them, single- and
// multi-pick.
func TestSheetMetalPickAccessors(t *testing.T) {
	s, part := faceSheet(t, 4)
	edge := EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])}

	flange := NewSheetMetalFlangeTool()
	if flange.PickCount() != 0 {
		t.Fatal("fresh flange should have 0 picks")
	}
	flange.Pick(s, edge)
	if flange.PickCount() != 1 {
		t.Errorf("flange pick count = %d, want 1", flange.PickCount())
	}
	flange.ClearPicks()
	if flange.PickCount() != 0 {
		t.Error("ClearPicks did not reset the flange pick")
	}

	corner := NewSheetMetalCornerTool()
	corner.Pick(s, edge)
	corner.Pick(s, edge)
	if corner.PickCount() != 2 {
		t.Errorf("corner pick count = %d, want 2", corner.PickCount())
	}
	corner.ClearPicks()
	if corner.PickCount() != 0 {
		t.Error("ClearPicks did not reset the corner picks")
	}
}
