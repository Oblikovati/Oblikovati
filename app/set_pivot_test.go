// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestSetOrbitPivotRecenters: clicking off-centre to set the Free-Orbit pivot moves the orbit centre
// (the camera target) to the clicked point while keeping the view direction (#913 N9).
func TestSetOrbitPivotRecenters(t *testing.T) {
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
