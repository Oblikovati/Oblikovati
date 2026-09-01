// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/scene"
)

// eyeTargetDist is the camera's look-at distance — what a wheel-zoom dolly changes.
func eyeTargetDist(c scene.Camera) float64 { return float64(c.Eye.DistanceTo(c.Target)) }

// scrollSession returns a session looking down −Z from (0,0,10) at the origin.
func scrollSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	cam := scene.NewCamera(200, 200)
	cam.Eye, cam.Target, cam.Up = math.P3(0, 0, 10), math.P3(0, 0, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	return s
}

// TestScrollViewportZoomsIn: a positive dy notch dollies the camera closer (zoom in).
func TestScrollViewportZoomsIn(t *testing.T) {
	t.Parallel()
	s := scrollSession(t)
	before := eyeTargetDist(s.Camera())
	s.ScrollViewport(0, 1, 100, 100)
	if after := eyeTargetDist(s.Camera()); after >= before {
		t.Errorf("distance after a +dy scroll = %.4f, want < %.4f (zoom in)", after, before)
	}
}

// TestScrollViewportZeroDeltaNoOp: dy=0 leaves the camera untouched, even with a non-zero dx —
// dx carries no default binding.
func TestScrollViewportZeroDeltaNoOp(t *testing.T) {
	t.Parallel()
	s := scrollSession(t)
	before := s.Camera()
	s.ScrollViewport(3, 0, 100, 100)
	if got := s.Camera(); got.Eye != before.Eye || got.Target != before.Target {
		t.Errorf("camera moved on a zero-dy scroll: eye %v→%v", before.Eye, got.Eye)
	}
}

// TestScrollViewportWheelInvertFlipsDirection: with Wheel Invert set, a positive dy zooms OUT.
func TestScrollViewportWheelInvertFlipsDirection(t *testing.T) {
	t.Parallel()
	s := scrollSession(t)
	if err := s.SetWheelInvert(true); err != nil {
		t.Fatalf("SetWheelInvert: %v", err)
	}
	before := eyeTargetDist(s.Camera())
	s.ScrollViewport(0, 1, 100, 100)
	if after := eyeTargetDist(s.Camera()); after <= before {
		t.Errorf("with wheel-invert, distance after +dy = %.4f, want > %.4f (zoom out)", after, before)
	}
}

// TestScrollViewportZoomToCursorStillZooms: the zoom-toward-cursor branch also dollies in on a
// positive dy (it aims the zoom at the pixel rather than the view centre).
func TestScrollViewportZoomToCursorStillZooms(t *testing.T) {
	t.Parallel()
	s := scrollSession(t)
	if err := s.SetZoomToCursor(true); err != nil {
		t.Fatalf("SetZoomToCursor: %v", err)
	}
	before := eyeTargetDist(s.Camera())
	s.ScrollViewport(0, 2, 40, 160)
	if after := eyeTargetDist(s.Camera()); after >= before {
		t.Errorf("zoom-to-cursor distance after +dy = %.4f, want < %.4f", after, before)
	}
}
