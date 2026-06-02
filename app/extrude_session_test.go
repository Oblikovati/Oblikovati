// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

func TestActiveExtrudeNilWhenNoExtrudeTool(t *testing.T) {
	s, _ := emptyPartSession(t)
	if s.ActiveExtrude() != nil {
		t.Error("ActiveExtrude should be nil with no tool")
	}
	s.StartTool(NewLineTool())
	if s.ActiveExtrude() != nil {
		t.Error("ActiveExtrude should be nil when a non-extrude tool is active")
	}
}

func TestExtrudeDistanceDisplayRoundTripsThroughDocUnit(t *testing.T) {
	s, _ := topDownPickerOverSquare(t)
	s.StartTool(NewExtrudeTool())
	// The document length unit defaults to mm; the model database unit is cm. Setting
	// 50 mm must store 5 (cm) on the tool and read back as 50 mm.
	s.SetExtrudeDistanceDisplay(50)
	if got := s.ActiveExtrude().Distance(); got < 4.999 || got > 5.001 {
		t.Errorf("distance stored = %v db units, want ~5 (cm) for 50 mm", got)
	}
	if got := s.ExtrudeDistanceDisplay(); got < 49.99 || got > 50.01 {
		t.Errorf("distance display = %v, want ~50 mm", got)
	}
}

func TestExtrudeDialogPathBuildsSolid(t *testing.T) {
	s, _ := topDownPickerOverSquare(t) // 2×2 square at origin, top-down camera
	s.StartTool(NewExtrudeTool())
	s.Click(200, 200) // pick the profile (center pixel)
	ext := s.ActiveExtrude()
	if _, picked := ext.PickedProfile(); !picked {
		t.Fatal("clicking inside the square should pick its profile")
	}
	s.SetExtrudeDistanceDisplay(40) // 40 mm via the dialog path
	if !ext.CanCommit() {
		t.Fatal("profile + distance should make the extrude committable")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("extrude via the dialog path: %v", err)
	}
	if !ext.AddedFeature().Health().OK() {
		t.Errorf("extruded feature unhealthy: %s", ext.AddedFeature().Health().Reason)
	}
}
