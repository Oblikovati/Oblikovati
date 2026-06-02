// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/scene"
)

// topDownPickerOverSquare returns a session whose picker looks straight down the XY
// sketch with its 2×2 square — so a center-pixel click ray lands inside the profile.
func topDownPickerOverSquare(t *testing.T) *Session {
	t.Helper()
	s, profile := newPartWithSquare(t, 2)
	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(1, 1, 20) // above the square's center (1,1)
	cam.Target = math.P3(1, 1, 0)
	cam.Up = math.V3(0, 1, 0)
	s.SetPicker(NewRayPicker(cam, partBodies(s)).
		WithSketches(func() []*sketch.Sketch { return []*sketch.Sketch{profile.Sketch} }))
	return s
}

func TestRayPickerSelectsSketchProfile(t *testing.T) {
	s := topDownPickerOverSquare(t)
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile))
	s.Click(200, 200) // center pixel → inside the square
	if s.Selection().Count() != 1 {
		t.Fatalf("profile pick selected %d, want 1", s.Selection().Count())
	}
	ph, ok := s.Selection().First().(ProfileHandle)
	if !ok || ph.ProfileIndex != 0 {
		t.Fatalf("selected %T (index %d), want ProfileHandle index 0", s.Selection().First(), ph.ProfileIndex)
	}
}

func TestRayPickerMissesOutsideProfile(t *testing.T) {
	s := topDownPickerOverSquare(t)
	s.Selection().SetFilter(NewSelectionFilter(SelectProfile))
	s.Click(10, 10) // a corner pixel, well outside the 2×2 square
	if s.Selection().Count() != 0 {
		t.Errorf("clicking outside the profile selected %d, want 0", s.Selection().Count())
	}
}

func TestExtrudeFromPickedProfile(t *testing.T) {
	s := topDownPickerOverSquare(t)
	ext := NewExtrudeTool()
	s.StartTool(ext) // sets the filter to SelectProfile
	s.Click(200, 200)
	ext.SetDistance(5)
	if !ext.CanCommit() {
		t.Fatal("clicking inside the square should have picked the profile for extrude")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("extrude from picked profile: %v", err)
	}
	if !ext.AddedFeature().Health().OK() {
		t.Errorf("extruded feature unhealthy: %s", ext.AddedFeature().Health().Reason)
	}
}
