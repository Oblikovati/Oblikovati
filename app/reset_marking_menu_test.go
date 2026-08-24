// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app/markingmenu"
)

// TestResetMarkingMenuRestoresEnrichedDefaults: after overwriting a menu,
// ResetMarkingMenu restores the enriched defaults (WorkPlane.Offset in base,
// Sketch.Finish in sketch).
func TestResetMarkingMenuRestoresEnrichedDefaults(t *testing.T) {
	store := markingmenu.NewMemStore()
	s := NewSession()
	if err := s.UseMarkingMenuStore(store); err != nil {
		t.Fatalf("UseMarkingMenuStore: %v", err)
	}

	// Overwrite base with a single-slot menu.
	custom := wire.MarkingMenuView{
		Environment: BaseEnvironment,
		Quadrants: []wire.MarkingMenuItem{
			{Quadrant: types.QuadrantNorth, CommandID: "Create.Extrude"},
		},
	}
	if err := s.SetMarkingMenu(custom); err != nil {
		t.Fatalf("SetMarkingMenu: %v", err)
	}
	if len(s.MarkingMenu(BaseEnvironment).Quadrants) != 1 {
		t.Fatal("expected overwrite to leave 1 quadrant slot")
	}

	// Reset to defaults.
	s.ResetMarkingMenu(BaseEnvironment)

	got := s.MarkingMenu(BaseEnvironment)
	if len(got.Quadrants) < 7 {
		t.Errorf("after reset, base menu should have 7 quadrant slots; got %d", len(got.Quadrants))
	}
	foundWP := false
	for _, slot := range got.Quadrants {
		if slot.CommandID == "WorkPlane.Offset" {
			foundWP = true
		}
	}
	if !foundWP {
		t.Errorf("after reset, base menu should include WorkPlane.Offset; got %v", got.Quadrants)
	}
	if store.Saved < 2 {
		t.Errorf("ResetMarkingMenu should have persisted; store.Saved = %d", store.Saved)
	}
}

// TestResetMarkingMenuSketch: ResetMarkingMenu restores sketch defaults, including
// Sketch.Finish as a radial slot and empty overflow.
func TestResetMarkingMenuSketch(t *testing.T) {
	s := NewSession()
	// Overwrite sketch with a single-slot menu.
	custom := wire.MarkingMenuView{
		Environment: SketchEnvironment,
		Quadrants: []wire.MarkingMenuItem{
			{Quadrant: types.QuadrantNorth, CommandID: "Sketch.Line"},
		},
		Overflow: []string{"Sketch.Rectangle"},
	}
	if err := s.SetMarkingMenu(custom); err != nil {
		t.Fatalf("SetMarkingMenu: %v", err)
	}

	s.ResetMarkingMenu(SketchEnvironment)

	got := s.MarkingMenu(SketchEnvironment)
	foundFinish := false
	for _, slot := range got.Quadrants {
		if slot.CommandID == "Sketch.Finish" {
			foundFinish = true
		}
	}
	if !foundFinish {
		t.Error("after reset, sketch menu should include Sketch.Finish as a radial slot")
	}
	if len(got.Overflow) != 0 {
		t.Errorf("after reset, sketch overflow should be empty; got %v", got.Overflow)
	}
}
