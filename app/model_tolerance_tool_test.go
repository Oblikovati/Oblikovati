// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/feature"
)

// #2049: ModelToleranceDefinition and ToleranceFeatures.AddModelTolerance were implemented and
// routed over the API, but ToleranceFeatures was 0-of-1 UI-reachable. The Drawing tab's frame
// and datum annotate a drawing VIEW; this is the model-level feature MBD consumers read.

// TestFeatureControlFrameAnnotatesModelGeometry drives the UI: start the tool, click a model
// face, set the characteristic, tolerance and datums, OK — and asserts the recorded frame.
func TestFeatureControlFrameAnnotatesModelGeometry(t *testing.T) {
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: face, Body: body}})

	tool := NewModelFrameTool()
	s.StartTool(tool)
	if tool.CanCommit() {
		t.Fatal("a frame should need the annotated geometry first")
	}
	s.Click(50, 50)
	tool.SetCharacteristicIndex(1) // flatness
	tool.SetValue(0.05)
	tool.SetDatums("A, B")
	if !tool.CanCommit() {
		t.Fatal("a frame with geometry and a tolerance should commit")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	mt, ok := tool.AddedFeature().Definition().(*feature.ModelToleranceFeature)
	if !ok {
		t.Fatalf("added %T, want a ModelToleranceFeature", tool.AddedFeature().Definition())
	}
	frames := mt.Definition().Frames
	if len(frames) != 1 {
		t.Fatalf("recorded %d frames, want 1", len(frames))
	}
	if frames[0].Characteristic != types.CharacteristicFlatness {
		t.Errorf("characteristic = %v, want flatness", frames[0].Characteristic)
	}
	if frames[0].Value != 0.05 {
		t.Errorf("tolerance = %g, want 0.05", frames[0].Value)
	}
	if got := frames[0].Datums; len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Errorf("datums = %v, want [A B] — the field is parsed and trimmed", got)
	}
	if len(frames[0].GeometryKey) == 0 {
		t.Error("the frame carries no geometry reference")
	}
}

// A datum feature records the label instead, against the same kind of reference.
func TestDatumFeatureAnnotatesModelGeometry(t *testing.T) {
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: face, Body: body}})

	tool := NewModelDatumTool()
	s.StartTool(tool)
	s.Click(50, 50)
	tool.SetLabel(" C ") // the field trims
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	mt := tool.AddedFeature().Definition().(*feature.ModelToleranceFeature)
	datums := mt.Definition().Datums
	if len(datums) != 1 || datums[0].Label != "C" {
		t.Errorf("recorded datums = %+v, want one labelled C", datums)
	}
	if len(mt.Definition().Frames) != 0 {
		t.Error("datum mode should record no feature-control frame")
	}
}

// An empty datum letter is not a datum: the OK button refuses rather than recording a blank.
func TestDatumFeatureNeedsALabel(t *testing.T) {
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	tool := NewModelDatumTool()
	s.StartTool(tool)
	tool.Pick(s, FaceHandle{Face: face, Body: body})
	tool.SetLabel("   ")
	if tool.CanCommit() {
		t.Error("a datum with a blank label should not commit")
	}
}

// The annotation is a history feature that changes no geometry — the body passes through.
func TestModelToleranceLeavesTheBodyUnchanged(t *testing.T) {
	s, def := emptyPartSession(t)
	face, body := boxTopFace(t, def)
	before := def.SurfaceBodies().Count()
	s.SetPicker(stubPicker{sel: FaceHandle{Face: face, Body: body}})

	tool := NewModelFrameTool()
	s.StartTool(tool)
	s.Click(50, 50)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if got := def.SurfaceBodies().Count(); got != before {
		t.Errorf("the part holds %d bodies after annotating, want %d unchanged", got, before)
	}
	if !tool.AddedFeature().Health().OK() {
		t.Errorf("the annotation feature is sick: %+v", tool.AddedFeature().Health())
	}
}

// Both annotations are reachable from the Inspect tab's Annotate panel.
func TestModelToleranceCommandsAreOnTheInspectTab(t *testing.T) {
	s := registeredSession(t)
	tab, ok := BuildRibbon(s).Tab("Inspect")
	if !ok {
		t.Fatal("ribbon has no Inspect tab")
	}
	panel, ok := tab.Panel("Annotate")
	if !ok {
		t.Fatal("Inspect tab has no Annotate panel")
	}
	for _, name := range []string{"Feature Control Frame", "Datum Feature"} {
		if _, ok := buttonNamed(panel, name); !ok {
			t.Errorf("the Annotate panel has no %q button", name)
		}
	}
}

// Every characteristic in the combo is a real enum member with a wire spelling, so the panel
// cannot offer one the API could not round-trip.
func TestGeometricCharacteristicOptionsAreParseable(t *testing.T) {
	opts := GeometricCharacteristicOptions()
	if len(opts) != len(geometricCharacteristics) {
		t.Fatalf("%d labels for %d characteristics", len(opts), len(geometricCharacteristics))
	}
	for i, label := range opts {
		parsed, ok := types.ParseGeometricCharacteristic(label)
		if !ok || parsed != geometricCharacteristics[i] {
			t.Errorf("option %d %q does not parse back to its characteristic", i, label)
		}
	}
}
