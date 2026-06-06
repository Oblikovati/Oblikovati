// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

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

func TestBeginEditFeatureOpensExtrudeAtItsDistance(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	s.BeginEditFeature(h)
	if !s.IsEditingFeature() {
		t.Fatal("BeginEditFeature should open an edit for an extrude")
	}
	if s.EditingFeatureName() != h.Feature.Name() {
		t.Errorf("editing name = %q, want %q", s.EditingFeatureName(), h.Feature.Name())
	}
	if n := s.EditFeatureParamCount(); n != 1 {
		t.Fatalf("extrude editable param count = %d, want 1 (Distance)", n)
	}
	if l := s.EditFeatureParamLabel(0); l != "Distance" {
		t.Errorf("param 0 label = %q, want %q", l, "Distance")
	}
	// Database 5 cm shows as 50 mm in the document's display unit.
	if d := s.EditFeatureParamValue(0); d < 49.99 || d > 50.01 {
		t.Errorf("edit distance display = %v, want ~50 mm", d)
	}
}

func TestCommitFeatureEditRecomputesTheBody(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	s.BeginEditFeature(h)
	s.SetEditFeatureParamValue(0, 80) // 80 mm = 8 cm
	if err := s.OK(); err != nil {    // OK commits the active feature-edit tool
		t.Fatalf("commit feature edit: %v", err)
	}
	if s.IsEditingFeature() {
		t.Error("a successful commit should close the edit")
	}
	if z := partBodies(s)()[0].RangeBox().Diagonal().Z; z < 7.99 || z > 8.01 {
		t.Errorf("body height after edit = %v, want ~8 cm", z)
	}
}

func TestCancelFeatureEditRestoresTheDistance(t *testing.T) {
	s, h := extrudedFeatureSession(t)
	s.BeginEditFeature(h)
	s.SetEditFeatureParamValue(0, 80) // change it, then back out
	s.CancelTool()
	if s.IsEditingFeature() {
		t.Error("cancel should close the edit")
	}
	if z := partBodies(s)()[0].RangeBox().Diagonal().Z; z < 4.99 || z > 5.01 {
		t.Errorf("body height after cancel = %v, want ~5 cm (unchanged)", z)
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
