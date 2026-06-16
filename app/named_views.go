// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/scene"
)

// CaptureNamedView saves the active view's current camera under name (M16-F03 #404), so
// RestoreNamedView can return to the exact framing. It errors when there is no active document.
func (s *Session) CaptureNamedView(name string) (doc.NamedView, error) {
	d := s.ActiveDocument()
	if d == nil {
		return doc.NamedView{}, fmt.Errorf("captureNamedView %q: no active document", name)
	}
	c := s.Camera()
	frame := doc.ViewHome{Eye: c.Eye, Target: c.Target, Up: c.Up, FOV: c.FOV}
	d.Views().CaptureNamed(name, frame)
	return doc.NamedView{Name: name, Home: frame}, nil
}

// NamedViews returns the active document's saved named views (empty when none / no document).
func (s *Session) NamedViews() []doc.NamedView {
	d := s.ActiveDocument()
	if d == nil {
		return nil
	}
	return d.Views().NamedViews()
}

// RestoreNamedView animates the active view to a saved named view's camera and fires the
// camera-changed event. It errors when the named view is absent.
func (s *Session) RestoreNamedView(name string) error {
	d := s.ActiveDocument()
	if d == nil {
		return fmt.Errorf("restoreNamedView %q: no active document", name)
	}
	h, ok := d.Views().NamedView(name)
	if !ok {
		return fmt.Errorf("restoreNamedView: no named view %q", name)
	}
	// Apply directly (not animated) so the logical camera is correct the instant the call
	// returns — an add-in reading get_camera right after must see the restored frame.
	s.SetCamera(s.homeCamera(&h))
	s.fireCameraChanged(d)
	return nil
}

// DeleteNamedView removes a saved named view, erroring when it is absent.
func (s *Session) DeleteNamedView(name string) error {
	d := s.ActiveDocument()
	if d == nil {
		return fmt.Errorf("deleteNamedView %q: no active document", name)
	}
	if !d.Views().DeleteNamed(name) {
		return fmt.Errorf("deleteNamedView: no named view %q", name)
	}
	return nil
}

// SetViewOrientation jumps the active view to a standard orientation (front/top/iso…),
// optionally fitting the model, and fires the camera-changed event. Orientations with no
// fixed direction (Current/Arbitrary/Saved/Flat*) keep the current camera.
func (s *Session) SetViewOrientation(o types.ViewOrientationTypeEnum, fit bool) error {
	d := s.ActiveDocument()
	if d == nil {
		return fmt.Errorf("setViewOrientation %s: no active document", o)
	}
	dir, up, ok := orientationFrame(o)
	if !ok {
		return nil // no-op for orientations without a fixed direction
	}
	s.SetCamera(s.orientedCamera(dir, up, fit))
	s.fireCameraChanged(d)
	return nil
}

// orientedCamera builds a camera looking along dir at the model center, keeping the current
// eye-target distance, then fits to the model when fit is set. An empty model (a sketch-only
// part) keeps the current target and skips the fit, so the frame never goes non-finite.
func (s *Session) orientedCamera(dir, up math.Vector3, fit bool) scene.Camera {
	c := s.Camera()
	b := s.modelBounds()
	dist := c.Eye.DistanceTo(c.Target)
	if dist <= 0 {
		dist = 10
	}
	target := c.Target
	if !b.IsEmpty() {
		target = b.Center()
		if d := b.Diagonal().Length(); d > 0 {
			dist = d * 1.5
		}
	}
	c.Target = target
	c.Eye = target.TranslateBy(dir.Scale(dist))
	c.Up = up
	if fit && !b.IsEmpty() {
		c = c.Fit(b)
	}
	return c
}

// fireCameraChanged emits the camera-changed event for document d.
func (s *Session) fireCameraChanged(d *doc.Document) {
	event.Emit(s.bus, event.After, CameraChanged{Document: d.ID()})
}

// orientationFrame maps a standard orientation to its (eye-direction, up) unit vectors in the
// Y-up world (eye = target + dir·distance). It returns ok=false for orientations with no fixed
// direction (current/arbitrary/saved/flat variants), which the caller treats as a no-op.
func orientationFrame(o types.ViewOrientationTypeEnum) (dir, up math.Vector3, ok bool) {
	yUp, zUp := math.V3(0, 1, 0), math.V3(0, 0, 1)
	switch o {
	case types.FrontViewOrientation:
		return math.V3(0, 0, 1), yUp, true
	case types.BackViewOrientation:
		return math.V3(0, 0, -1), yUp, true
	case types.RightViewOrientation:
		return math.V3(1, 0, 0), yUp, true
	case types.LeftViewOrientation:
		return math.V3(-1, 0, 0), yUp, true
	case types.TopViewOrientation:
		return math.V3(0, 1, 0), zUp.Negate(), true
	case types.BottomViewOrientation:
		return math.V3(0, -1, 0), zUp, true
	case types.IsoTopRightViewOrientation, types.DefaultViewOrientation:
		return math.V3(1, 1, 1), yUp, true
	case types.IsoTopLeftViewOrientation:
		return math.V3(-1, 1, 1), yUp, true
	case types.IsoBottomRightViewOrientation:
		return math.V3(1, -1, 1), yUp, true
	case types.IsoBottomLeftViewOrientation:
		return math.V3(-1, -1, 1), yUp, true
	}
	return math.Vector3{}, math.Vector3{}, false
}
