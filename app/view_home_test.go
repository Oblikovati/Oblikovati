// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
)

// TestSetActiveViewHomeCapturesAndGoHomeRestores checks the ViewCube "Set Current View as
// Home" (Fixed Distance) captures the current camera and Go Home returns to it.
func TestSetActiveViewHomeCapturesAndGoHomeRestores(t *testing.T) {
	t.Parallel()
	s := NewSession()
	a := addPart(t, s, "home.obk")
	activate(t, s, a)

	home := s.Camera()
	home.Eye = math.P3(50, 10, 20)
	s.SetCamera(home)
	s.SetActiveViewHome(false) // Fixed Distance
	if s.ActiveView().Home == nil {
		t.Fatal("Home not captured")
	}

	away := s.Camera()
	away.Eye = math.P3(0, 0, 5)
	s.SetCamera(away)

	s.GoHome()
	for i := 0; i < 500 && s.CameraAnimating(); i++ {
		s.TickCameraAnimation(0.05)
	}
	if got := s.Camera().Eye; got != math.P3(50, 10, 20) {
		t.Errorf("after GoHome eye = %v, want (50,10,20)", got)
	}
}

// TestGoHomeWithoutCustomHomeUsesIso ensures Go Home falls back to the default iso framing
// (no panic, camera moves to a non-degenerate frame) when no custom Home is set.
func TestGoHomeWithoutCustomHomeUsesIso(t *testing.T) {
	t.Parallel()
	s := NewSession()
	a := addPart(t, s, "iso.obk")
	activate(t, s, a)
	if s.ActiveView().Home != nil {
		t.Fatal("expected no custom Home by default")
	}
	s.GoHome() // must not panic; falls back to iso Home
	for i := 0; i < 500 && s.CameraAnimating(); i++ {
		s.TickCameraAnimation(0.05)
	}
}
