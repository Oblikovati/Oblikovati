// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestFeatureEditAccessorGuards covers the feature-edit accessors' no-edit guards: with no edit
// active they return zero values rather than panicking (the head dialog reads them every frame).
func TestFeatureEditAccessorGuards(t *testing.T) {
	s := NewSession()
	if s.IsEditingFeature() || s.ActiveFeatureEdit() != nil {
		t.Fatal("a fresh session is not editing a feature")
	}
	if s.EditingFeatureName() != "" || s.EditFeatureParamCount() != 0 || s.EditFeatureRefSlotCount() != 0 {
		t.Error("no-edit accessors should report empty/zero")
	}
	// Indexed accessors must be safe with no edit active.
	_ = s.EditFeatureParamLabel(0)
	_ = s.EditFeatureParamUnitName(0)
	_ = s.EditFeatureParamIsInteger(0)
	_ = s.EditFeatureParamValue(0)
	s.SetEditFeatureParamValue(0, 1)
	_ = s.EditFeatureRefSlotLabel(0)
	_ = s.EditFeatureRefSlotRefCount(0)
	_ = s.EditFeatureRefSlotClearable(0)
	_ = s.EditFeatureRefSlotArmed(0)
}

// TestBeginEditExtrudeStartsDedicatedTool covers editToolFor / the extrude edit tool path: editing
// an extrude starts its dedicated edit tool and rolls the edit scope back to that feature.
func TestBeginEditExtrudeStartsDedicatedTool(t *testing.T) {
	s, _, f1, f2 := twoExtrudePart(t)
	s.BeginEditFeature(FeatureHandle{Feature: f1})

	if _, active := s.EditScopeSeq(); !active {
		t.Fatal("edit scope should be active after BeginEditFeature")
	}
	if !s.EditScopeHides(f2.Seq()) {
		t.Error("the later feature should be hidden by the edit scope")
	}
	at := s.ActiveTool()
	if at == nil {
		t.Fatal("BeginEditFeature should start a dedicated edit tool for the extrude")
	}
	if pt, ok := at.Tool().(interface{ Params() ToolParams }); ok {
		if pt.Params().Empty() {
			t.Error("the extrude edit tool should expose editable params")
		}
	}
}
