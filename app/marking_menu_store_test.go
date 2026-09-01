// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app/markingmenu"
)

// TestMarkingMenuStorePersistsAndReloads mirrors TestKeymapStorePersistsAndReloads
// (#831): customizing a marking menu persists through the store and reloads into a
// fresh session.
func TestMarkingMenuStorePersistsAndReloads(t *testing.T) {
	t.Parallel()
	store := markingmenu.NewMemStore()
	s1 := NewSession()
	if err := s1.UseMarkingMenuStore(store); err != nil {
		t.Fatalf("UseMarkingMenuStore: %v", err)
	}

	custom := wire.MarkingMenuView{
		Environment: BaseEnvironment,
		Quadrants: []wire.MarkingMenuItem{
			{Quadrant: types.QuadrantNorth, CommandID: "Create.Extrude"},
		},
	}
	if err := s1.SetMarkingMenu(custom); err != nil {
		t.Fatalf("SetMarkingMenu: %v", err)
	}
	if store.Saved == 0 {
		t.Error("SetMarkingMenu should have persisted through the store")
	}

	s2 := NewSession()
	if err := s2.UseMarkingMenuStore(store); err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := s2.MarkingMenu(BaseEnvironment)
	if len(got.Quadrants) != 1 || got.Quadrants[0].CommandID != "Create.Extrude" {
		t.Errorf("reloaded base menu = %v, want one slot (Create.Extrude/north)", got.Quadrants)
	}
}

// TestClassicTogglePersistsAndReloads: toggling the classic/radial style persists
// across sessions.
func TestClassicTogglePersistsAndReloads(t *testing.T) {
	t.Parallel()
	store := markingmenu.NewMemStore()
	s1 := NewSession()
	if err := s1.UseMarkingMenuStore(store); err != nil {
		t.Fatalf("UseMarkingMenuStore: %v", err)
	}
	s1.ToggleContextMenuStyle()
	if store.Saved == 0 {
		t.Error("ToggleContextMenuStyle should have persisted through the store")
	}

	s2 := NewSession()
	if err := s2.UseMarkingMenuStore(store); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !s2.ClassicContextMenu() {
		t.Error("reloaded session should have the classic menu style")
	}
}

// TestFreshInstallUsesEnrichedDefaults: a fresh session (no stored customization)
// has the enriched default menus — WorkPlane.Offset in base and Sketch.Finish as
// a radial slot in sketch.
func TestFreshInstallUsesEnrichedDefaults(t *testing.T) {
	t.Parallel()
	s := NewSession()
	base := s.MarkingMenu(BaseEnvironment)
	foundWP := false
	for _, slot := range base.Quadrants {
		if slot.CommandID == "WorkPlane.Offset" {
			foundWP = true
		}
	}
	if !foundWP {
		t.Error("default base menu should include WorkPlane.Offset")
	}

	sketch := s.MarkingMenu(SketchEnvironment)
	foundFinish := false
	for _, slot := range sketch.Quadrants {
		if slot.CommandID == "Sketch.Finish" {
			foundFinish = true
		}
	}
	if !foundFinish {
		t.Error("default sketch menu should include Sketch.Finish as a radial slot")
	}
	if len(sketch.Overflow) != 0 {
		t.Errorf("Sketch.Finish promoted from overflow; overflow should be empty, got %v", sketch.Overflow)
	}
}
