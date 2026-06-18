// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"
)

// TestZoomWindowDragZoomsAndDisarms: arm → drag a centred half-size box → release zooms the view in
// (eye–target distance halves) and disarms the tool.
func TestZoomWindowDragZoomsAndDisarms(t *testing.T) {
	s, _ := emptyPartSession(t)
	cam := s.Camera()
	cam.Width, cam.Height = 800, 600
	s.SetCamera(cam)
	d0 := float64(s.Camera().Eye.DistanceTo(s.Camera().Target))

	s.ArmZoomWindow()
	if !s.ZoomWindowArmed() {
		t.Fatal("ArmZoomWindow should arm the tool")
	}
	s.BeginZoomWindow(200, 150)
	s.UpdateZoomWindow(600, 450) // a 400×300 box (half the 800×600 viewport)
	if !s.ZoomWindowDragging() {
		t.Fatal("a begun drag should report dragging")
	}
	s.CommitZoomWindow()

	if s.ZoomWindowArmed() || s.ZoomWindowDragging() {
		t.Error("commit should disarm the tool and end the drag")
	}
	d1 := float64(s.Camera().Eye.DistanceTo(s.Camera().Target))
	if stdmath.Abs(d1-d0/2) > 1e-6 {
		t.Errorf("zoom-window distance = %v, want %v (half: box is half the viewport)", d1, d0/2)
	}
}

// TestZoomWindowDegenerateIsNoOp: a click (no real rectangle) disarms without moving the camera.
func TestZoomWindowDegenerateIsNoOp(t *testing.T) {
	s, _ := emptyPartSession(t)
	before := s.Camera()
	s.ArmZoomWindow()
	s.BeginZoomWindow(300, 300)
	s.UpdateZoomWindow(301, 301) // sub-threshold
	s.CommitZoomWindow()
	if s.ZoomWindowArmed() {
		t.Error("a degenerate Zoom Window should still disarm")
	}
	if s.Camera() != before {
		t.Error("a degenerate Zoom Window must not move the camera")
	}
}

// TestDisarmZoomWindowCancels: Esc/disarm drops an armed tool and any in-progress rubber band.
func TestDisarmZoomWindowCancels(t *testing.T) {
	s, _ := emptyPartSession(t)
	s.ArmZoomWindow()
	s.BeginZoomWindow(100, 100)
	s.DisarmZoomWindow()
	if s.ZoomWindowArmed() || s.ZoomWindowDragging() {
		t.Error("DisarmZoomWindow should clear both the armed flag and the rubber band")
	}
}

// TestBeginZoomWindowRequiresArm: Begin is a no-op unless the tool is armed.
func TestBeginZoomWindowRequiresArm(t *testing.T) {
	s, _ := emptyPartSession(t)
	s.BeginZoomWindow(10, 10)
	if s.ZoomWindowDragging() {
		t.Error("BeginZoomWindow without arming should not start a drag")
	}
}
