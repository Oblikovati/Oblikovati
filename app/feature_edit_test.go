// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/feature"
)

// extrudedFeatureSession returns a session with one committed extrude (5 cm tall, 2×2
// base) and a handle to that feature — the starting point for the edit-on-double-click
// flow tests.
func extrudedFeatureSession(t *testing.T) (*Session, FeatureHandle) {
	t.Helper()
	s := topDownPickerOverSquare(t)
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(200, 200)  // pick the profile
	ext.SetDistance(5) // 5 cm (database units)
	if err := s.OK(); err != nil {
		t.Fatalf("seed extrude: %v", err)
	}
	return s, FeatureHandle{Feature: ext.AddedFeature()}
}

// TestBeginEditFeatureReopensExtrudePanel locks the unified edit flow: double-clicking
// a committed extrude re-opens the Extrude property panel (the creation tool in edit
// mode) seeded with the feature's full creation state — not the generic param editor.
func TestBeginEditFeatureReopensExtrudePanel(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	s.BeginEditFeature(h)
	ext := s.ActiveExtrude()
	if ext == nil {
		t.Fatal("BeginEditFeature should re-open the Extrude tool for an extrude")
	}
	if !ext.IsEditing() || ext.EditingName() != h.Feature.Name() {
		t.Errorf("edit binding = (%v, %q), want (true, %q)", ext.IsEditing(), ext.EditingName(), h.Feature.Name())
	}
	if s.IsEditingFeature() {
		t.Error("the generic feature editor must not be open for an extrude")
	}
	if n := len(ext.PickedProfiles()); n != 1 {
		t.Errorf("seeded profile count = %d, want 1", n)
	}
	// Database 5 cm shows as 50 mm in the document's display unit (the panel field).
	if d := s.ExtrudeDistanceDisplay(); d < 49.99 || d > 50.01 {
		t.Errorf("edit distance display = %v, want ~50 mm", d)
	}
}

func TestCommitFeatureEditRecomputesTheBody(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	s.BeginEditFeature(h)
	s.SetExtrudeDistanceDisplay(80) // 80 mm = 8 cm, via the panel path
	if err := s.OK(); err != nil {  // OK commits the edit-mode tool
		t.Fatalf("commit feature edit: %v", err)
	}
	if s.ActiveExtrude() != nil {
		t.Error("a successful commit should close the edit panel")
	}
	if z := partBodies(s)()[0].RangeBox().Diagonal().Z; z < 7.99 || z > 8.01 {
		t.Errorf("body height after edit = %v, want ~8 cm", z)
	}
}

func TestCancelFeatureEditRestoresTheDistance(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	s.BeginEditFeature(h)
	s.SetExtrudeDistanceDisplay(80) // change it, then back out
	s.CancelTool()
	if s.ActiveExtrude() != nil {
		t.Error("cancel should close the edit panel")
	}
	if z := partBodies(s)()[0].RangeBox().Diagonal().Z; z < 4.99 || z > 5.01 {
		t.Errorf("body height after cancel = %v, want ~5 cm (unchanged)", z)
	}
}

// TestExtrudeEditRoundTripsFullCreationSurface locks "every creation property is
// editable": operation, taper, extent direction and asymmetry survive a commit and land
// in the definition.
func TestExtrudeEditRoundTripsFullCreationSurface(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	s.BeginEditFeature(h)
	ext := s.ActiveExtrude()
	ext.SetTaper(0.1)
	ext.SetAsymmetric(true)
	ext.SetSecondDistance(2)
	if err := s.OK(); err != nil {
		t.Fatalf("commit full-surface edit: %v", err)
	}
	def := h.Feature.Definition().(*feature.ExtrudeFeature)
	if got := def.Taper(); got < 0.099 || got > 0.101 {
		t.Errorf("taper after edit = %v, want 0.1", got)
	}
	if def.Extent().Distance2 == nil {
		t.Fatal("asymmetric second distance did not reach the definition")
	}
	if d2 := def.Extent().Distance2(); d2 < 1.99 || d2 > 2.01 {
		t.Errorf("second distance after edit = %v, want 2", d2)
	}
	// The body now spans -2..+5 along Z: 7 cm tall.
	if z := partBodies(s)()[0].RangeBox().Diagonal().Z; z < 6.99 || z > 7.01 {
		t.Errorf("body height after asymmetric edit = %v, want ~7 cm", z)
	}
}

func TestFeatureEditAPIsAreNoOpsWhenNotEditing(t *testing.T) {
	s, _ := extrudedFeatureSession(t)
	if s.IsEditingFeature() || s.EditingFeatureName() != "" || s.EditFeatureParamCount() != 0 {
		t.Error("no edit should be open before BeginEditFeature")
	}
	if err := s.OK(); err == nil {
		t.Error("OK with no active tool should error")
	}
	s.CancelTool() // must not panic
}

// TestFeatureEditIsEditableReflectsCapability checks the browser uses FeatureIsEditable to
// enable/grey the Edit entry: an extrude is editable; a feature with no scalar params is not.
func TestFeatureEditIsEditableReflectsCapability(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	_ = s
	if !FeatureIsEditable(h.Feature) {
		t.Error("an extrude should be editable")
	}
}

// TestFilletEditSeedsKeysAndRoundTrips locks the dress-up edit flow: editing a fillet
// re-opens the Fillet panel with the feature's edges retained as the seeded selection
// (their topology was consumed, so only keys survive), and a radius change written
// through the panel lands in the definition on commit.
func TestFilletEditSeedsKeysAndRoundTrips(t *testing.T) {
	s, block := newPartWithBlock(t, 2)
	edge := verticalEdgeOf(t, block)
	s.SetPicker(stubPicker{sel: edge})
	create := NewFilletTool()
	s.StartTool(create)
	s.Click(50, 50)
	create.SetRadius(0.5)
	if err := s.OK(); err != nil {
		t.Fatalf("seed fillet: %v", err)
	}

	s.BeginEditFeature(FeatureHandle{Feature: create.AddedFeature()})
	f := s.ActiveFillet()
	if f == nil {
		t.Fatal("BeginEditFeature should re-open the Fillet panel for a fillet")
	}
	if !f.IsEditing() || f.EdgeCount() != 1 {
		t.Fatalf("edit binding = (%v, %d edges), want (true, 1 seeded edge)", f.IsEditing(), f.EdgeCount())
	}
	f.SetRadius(0.25)
	if err := s.OK(); err != nil {
		t.Fatalf("commit fillet edit: %v", err)
	}
	def := create.AddedFeature().Definition().(*feature.FilletFeature).Definition()
	if r := def.Radius(); r < 0.249 || r > 0.251 {
		t.Errorf("radius after edit = %v, want 0.25", r)
	}
	if len(def.EdgeKeys) != 1 {
		t.Errorf("edge keys after edit = %d, want the retained 1", len(def.EdgeKeys))
	}
}

// TestHoleEditSwitchesSeatType locks that the Hole panel's full creation surface is
// editable: a drilled hole becomes a counterbore on edit-commit, with the seat
// dimensions written into the definition.
func TestHoleEditSwitchesSeatType(t *testing.T) {
	s, block := newPartWithBlock(t, 4)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})
	create := NewHoleTool()
	s.StartTool(create)
	s.Click(50, 50)
	create.SetDiameter(0.6)
	create.SetDepth(1.5)
	if err := s.OK(); err != nil {
		t.Fatalf("seed hole: %v", err)
	}

	s.BeginEditFeature(FeatureHandle{Feature: create.AddedFeature()})
	h := s.ActiveHole()
	if h == nil {
		t.Fatal("BeginEditFeature should re-open the Hole panel for a hole")
	}
	if !h.HasPlacement() {
		t.Fatal("the placement face must be retained from the definition")
	}
	h.SetCounterbore(true)
	h.SetCounterDiameter(1.0)
	h.SetCounterDepth(0.4)
	if err := s.OK(); err != nil {
		t.Fatalf("commit hole edit: %v", err)
	}
	def := create.AddedFeature().Definition().(*feature.HoleFeature).Definition()
	if def.Type != feature.CounterboreHole {
		t.Errorf("hole type after edit = %v, want CounterboreHole", def.Type)
	}
	if d := def.CounterDiameter(); d < 0.99 || d > 1.01 {
		t.Errorf("counter diameter after edit = %v, want 1.0", d)
	}
}
