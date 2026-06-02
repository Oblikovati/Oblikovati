// SPDX-License-Identifier: GPL-2.0-only

package app

import "github.com/Oblikovati/oblikovati/scene"

// Camera transitions: entering a sketch swings the view to face the sketch plane
// head-on (Inventor's behavior), and finishing restores the prior view. The motion is
// a short eased tween the head advances each frame (TickCameraAnimation); the logic is
// pure so the "ends facing the plane" / "restores" behavior is unit-testable.

// sketchViewTweenSeconds is how long the enter/exit camera swing takes.
const sketchViewTweenSeconds = 0.35

// cameraTween is an in-progress camera transition from one view to another.
type cameraTween struct {
	from, to scene.Camera
	elapsed  float64
	duration float64
	active   bool
}

// animateCameraTo starts an eased transition of the camera to target over duration
// seconds; a non-positive duration snaps immediately.
func (s *Session) animateCameraTo(target scene.Camera, duration float64) {
	if duration <= 0 {
		s.SetCamera(target)
		return
	}
	s.camTween = cameraTween{from: s.camera, to: target, duration: duration, active: true}
}

// CameraAnimating reports whether a camera transition is running, so the head drives it
// (TickCameraAnimation) instead of applying user navigation this frame.
func (s *Session) CameraAnimating() bool { return s.camTween.active }

// TickCameraAnimation advances an in-progress camera transition by dt seconds, applying
// the eased interpolation and finishing exactly on the target. It is a no-op when no
// transition is active.
func (s *Session) TickCameraAnimation(dt float64) {
	if !s.camTween.active || dt < 0 {
		return
	}
	s.camTween.elapsed += dt
	f := s.camTween.elapsed / s.camTween.duration
	if f >= 1 {
		f = 1
		s.camTween.active = false
	}
	s.SetCamera(scene.Lerp(s.camTween.from, s.camTween.to, smoothstep(f)))
}

// smoothstep is the ease-in/ease-out curve 3f²−2f³ on [0,1].
func smoothstep(f float64) float64 { return f * f * (3 - 2*f) }
