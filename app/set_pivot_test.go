// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestToggleSteeringWheel covers the SteeringWheels menu's on/off/disarm state (#913 N26).
func TestToggleSteeringWheel(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.SteeringWheelActive() {
		t.Fatal("SteeringWheels should be off by default")
	}
	s.ToggleSteeringWheel()
	if !s.SteeringWheelActive() {
		t.Fatal("toggle should show the wheel")
	}
	s.DisarmSteeringWheel()
	if s.SteeringWheelActive() {
		t.Fatal("disarm should hide the wheel")
	}
}

// TestToggleConstrainedOrbit covers the Constrained Orbit tool's on/off/disarm state (#913 N10).
func TestToggleConstrainedOrbit(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.ConstrainedOrbitActive() {
		t.Fatal("Constrained Orbit should be off by default")
	}
	s.ToggleConstrainedOrbit()
	if !s.ConstrainedOrbitActive() {
		t.Fatal("toggle should turn it on")
	}
	s.ToggleConstrainedOrbit()
	if s.ConstrainedOrbitActive() {
		t.Fatal("toggle should turn it off")
	}
	s.ToggleConstrainedOrbit()
	s.DisarmConstrainedOrbit()
	if s.ConstrainedOrbitActive() {
		t.Fatal("disarm should turn it off")
	}
}

// TestSetOrbitPivotRecenters: clicking off-centre to set the Free-Orbit pivot moves the orbit centre
// (the camera target) to the clicked point while keeping the view direction (#913 N9).
func TestSetOrbitPivotRecenters(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)
	cam := s.Camera()
	cam.Width, cam.Height = 800, 600
	s.SetCamera(cam)

	before := s.Camera().Target
	s.SetOrbitPivot(650, 180) // off-centre viewport pixel
	if s.Camera().Target.IsEqualTo(before, 1e-9) {
		t.Error("set-pivot should move the orbit centre (target) to the clicked point")
	}
	if !s.Camera().Forward().IsEqualTo(cam.Forward(), 1e-9) {
		t.Error("set-pivot must keep the view direction")
	}
}
