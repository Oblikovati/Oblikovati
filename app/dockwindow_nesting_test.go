// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// nestedWindow is a window whose edited control lives two levels deep (tabs ▸ grid ▸ field),
// exercising that state updates and validation walk the whole tree (ADR-0019).
func nestedWindow() wire.DockableWindowSpec {
	field := wire.PanelControlSpec{Kind: types.PanelTextBox, ID: "startDepth", Value: "0"}
	grid := wire.PanelControlSpec{
		Kind: types.PanelGrid, ID: "depths",
		Columns:  []types.GridTrack{{Kind: types.GridTrackAuto}, {Kind: types.GridTrackFraction, Value: 1}},
		Children: []wire.PanelControlSpec{{Kind: types.PanelLabel, Text: "Start"}, field},
	}
	return wire.DockableWindowSpec{
		ID: "job.edit", Title: "Job Edit", Visible: true,
		Controls: []wire.PanelControlSpec{{
			Kind: types.PanelTabs, ID: "tabs",
			Children: []wire.PanelControlSpec{{Kind: types.PanelGroup, Title: "Setup", Children: []wire.PanelControlSpec{grid}}},
		}},
	}
}

// TestPanelValueChangedUpdatesNestedControl is the regression for the flat-only walk:
// editing a deeply nested grid field must update its stored value, not silently miss it.
func TestPanelValueChangedUpdatesNestedControl(t *testing.T) {
	s := NewSession()
	if err := s.SetDockableWindow(nestedWindow()); err != nil {
		t.Fatalf("SetDockableWindow: %v", err)
	}
	s.PanelValueChanged("job.edit", "startDepth", "12 mm")

	spec, _ := s.DockableWindows().Get("job.edit")
	got := findControlValue(spec.Controls, "startDepth")
	if got != "12 mm" {
		t.Errorf("nested stored value = %q, want %q", got, "12 mm")
	}
}

// findControlValue walks the control tree returning the value of the control with id.
func findControlValue(controls []wire.PanelControlSpec, id string) string {
	for _, c := range controls {
		if c.ID == id {
			return c.Value
		}
		if v := findControlValue(c.Children, id); v != "" {
			return v
		}
	}
	return ""
}

// TestSetDockableWindowRejectsTooDeep guards the host renderer against a runaway tree:
// nesting beyond the depth bound is rejected with a message naming the window.
func TestSetDockableWindowRejectsTooDeep(t *testing.T) {
	deep := wire.PanelControlSpec{Kind: types.PanelLabel, Text: "x"}
	for range maxControlNestDepth + 2 {
		deep = wire.PanelControlSpec{Kind: types.PanelGroup, Title: "g", Children: []wire.PanelControlSpec{deep}}
	}
	w := wire.DockableWindowSpec{ID: "deep", Title: "Deep", Controls: []wire.PanelControlSpec{deep}}
	if err := NewSession().SetDockableWindow(w); err == nil {
		t.Error("over-deep control tree should be rejected")
	}
}

// TestSetDockableWindowRejectsReservedRowSpan pins ADR-0020: row span is reserved, so a
// child requesting RowSpan > 1 is rejected until the feature ships.
func TestSetDockableWindowRejectsReservedRowSpan(t *testing.T) {
	child := wire.PanelControlSpec{Kind: types.PanelLabel, Text: "x", Cell: &types.GridCell{RowSpan: 2}}
	grid := wire.PanelControlSpec{Kind: types.PanelGrid, ID: "g", Columns: []types.GridTrack{{Kind: types.GridTrackAuto}}, Children: []wire.PanelControlSpec{child}}
	w := wire.DockableWindowSpec{ID: "rs", Title: "RowSpan", Controls: []wire.PanelControlSpec{grid}}
	if err := NewSession().SetDockableWindow(w); err == nil {
		t.Error("RowSpan > 1 should be rejected as reserved")
	}
}

// TestSetDockableWindowAcceptsValidNesting ensures the validator passes a normal nested tree.
func TestSetDockableWindowAcceptsValidNesting(t *testing.T) {
	if err := NewSession().SetDockableWindow(nestedWindow()); err != nil {
		t.Errorf("valid nested window rejected: %v", err)
	}
}
