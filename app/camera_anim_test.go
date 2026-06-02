// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// runCameraAnimation ticks an in-progress transition to completion (bounded so a bug
// cannot loop forever).
func runCameraAnimation(s *Session) {
	for i := 0; i < 1000 && s.CameraAnimating(); i++ {
		s.TickCameraAnimation(0.05)
	}
}

func TestEnterSketchSwingsCameraToFacePlane(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XZPlane())
	s.EnterSketch(sk)
	if !s.CameraAnimating() {
		t.Fatal("entering a sketch should start a camera transition")
	}
	runCameraAnimation(s)
	if s.CameraAnimating() {
		t.Fatal("camera transition did not complete")
	}
	// The view now looks straight at the sketch plane (forward ∥ plane normal).
	n := sk.Plane().Normal().AsVector()
	if dot := stdmath.Abs(float64(s.Camera().Forward().Dot(n))); dot < 1-1e-6 {
		t.Errorf("view not facing the sketch plane: |fwd·n| = %v", dot)
	}
}

func TestExitSketchRestoresCamera(t *testing.T) {
	s, def := emptyPartSession(t)
	// A distinctive starting view to detect a faithful restore.
	cam := s.Camera()
	cam.Eye = math.P3(7, 8, 9)
	cam.Target = math.P3(1, 1, 1)
	s.SetCamera(cam)
	saved := s.Camera()

	sk := def.Sketches().Add(sketch.XZPlane())
	s.EnterSketch(sk)
	runCameraAnimation(s) // swing to the plane
	s.ExitSketch()
	runCameraAnimation(s) // swing back

	got := s.Camera()
	if !got.Eye.IsEqualTo(saved.Eye, 1e-6) || !got.Target.IsEqualTo(saved.Target, 1e-6) {
		t.Errorf("exit did not restore the camera: eye %v/target %v, want %v/%v",
			got.Eye, got.Target, saved.Eye, saved.Target)
	}
}

func TestAnimateCameraZeroDurationSnaps(t *testing.T) {
	s, _ := emptyPartSession(t)
	target := s.Camera()
	target.Eye = math.P3(3, 4, 5)
	s.animateCameraTo(target, 0) // a non-positive duration applies immediately
	if s.CameraAnimating() {
		t.Error("a zero-duration transition should not start an animation")
	}
	if !s.Camera().Eye.IsEqualTo(math.P3(3, 4, 5), 1e-9) {
		t.Errorf("snap eye = %v, want (3,4,5)", s.Camera().Eye)
	}
}

func TestTickCameraAnimationNoopWhenIdle(t *testing.T) {
	s, _ := emptyPartSession(t)
	before := s.Camera()
	s.TickCameraAnimation(0.1) // nothing animating
	if s.Camera() != before {
		t.Error("ticking with no active animation should not move the camera")
	}
}

func TestEnterSketchOnXYKeepsTopDownView(t *testing.T) {
	// The emptyPartSession camera already looks down −Z at the XY origin, so entering a
	// sketch on XY leaves the view essentially unchanged after the (no-op) swing.
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	runCameraAnimation(s)
	if e := s.Camera().Eye; !e.IsEqualTo(math.P3(0, 0, 10), 1e-6) {
		t.Errorf("XY sketch view eye = %v, want the unchanged top-down (0,0,10)", e)
	}
}
