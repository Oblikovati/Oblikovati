// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/math"
	"oblikovati/model/sketch"
	"oblikovati/scene"
)

// TestRayPickerSelectsSketchCenterline checks the part-view ray picker can hit a sketch line
// (a centerline) under the SketchEntity filter — what lets the Revolve tool pick its axis.
func TestRayPickerSelectsSketchCenterline(t *testing.T) {
	s, profile := newPartWithSquare(t, 2) // 2×2 square on XY
	mid := profile.Sketch.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(2, 1))
	mid.SetCenterline(true)

	cam := scene.NewCamera(400, 400)
	cam.Eye = math.P3(1, 1, 20) // straight down over the midline's centre (1,1)
	cam.Target = math.P3(1, 1, 0)
	cam.Up = math.V3(0, 1, 0)
	s.SetPicker(NewRayPicker(cam, partBodies(s)).
		WithSketches(func() []*sketch.Sketch { return []*sketch.Sketch{profile.Sketch} }))

	s.Selection().SetFilter(NewSelectionFilter(SelectSketchEntity))
	s.Click(200, 200) // centre pixel → ray passes through the midline at (1,1)

	if s.Selection().Count() != 1 {
		t.Fatalf("sketch-curve pick selected %d, want 1", s.Selection().Count())
	}
	h, ok := s.Selection().First().(SketchEntityHandle)
	if !ok || h.Entity != mid {
		t.Fatalf("selected %T, want the centerline", s.Selection().First())
	}
}
