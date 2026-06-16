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
	s.PushViewHistory() // record the view Previous View (F5) returns to
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
	s.PushViewHistory() // record the view Previous View (F5) returns to
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

// OpenNamedViewsPanel opens the Named Views panel (M16-F03 #404).
func (s *Session) OpenNamedViewsPanel() { s.namedViewsPanelOpen = true }

// CloseNamedViewsPanel closes the Named Views panel.
func (s *Session) CloseNamedViewsPanel() { s.namedViewsPanelOpen = false }

// NamedViewsPanelOpen reports whether the Named Views panel is open.
func (s *Session) NamedViewsPanelOpen() bool { return s.namedViewsPanelOpen }

// orientationVector pairs a standard orientation's eye-direction with its up vector (Y-up
// world; eye = target + dir·distance).
type orientationVector struct{ dir, up math.Vector3 }

// orientationFrames is the table of standard orientations that have a fixed camera direction.
// Orientations absent from it (current/arbitrary/saved/flat variants) keep the current camera.
var orientationFrames = map[types.ViewOrientationTypeEnum]orientationVector{
	types.FrontViewOrientation:          {math.V3(0, 0, 1), math.V3(0, 1, 0)},
	types.BackViewOrientation:           {math.V3(0, 0, -1), math.V3(0, 1, 0)},
	types.RightViewOrientation:          {math.V3(1, 0, 0), math.V3(0, 1, 0)},
	types.LeftViewOrientation:           {math.V3(-1, 0, 0), math.V3(0, 1, 0)},
	types.TopViewOrientation:            {math.V3(0, 1, 0), math.V3(0, 0, -1)},
	types.BottomViewOrientation:         {math.V3(0, -1, 0), math.V3(0, 0, 1)},
	types.IsoTopRightViewOrientation:    {math.V3(1, 1, 1), math.V3(0, 1, 0)},
	types.DefaultViewOrientation:        {math.V3(1, 1, 1), math.V3(0, 1, 0)},
	types.IsoTopLeftViewOrientation:     {math.V3(-1, 1, 1), math.V3(0, 1, 0)},
	types.IsoBottomRightViewOrientation: {math.V3(1, -1, 1), math.V3(0, 1, 0)},
	types.IsoBottomLeftViewOrientation:  {math.V3(-1, -1, 1), math.V3(0, 1, 0)},
}

// orientationFrame returns the (eye-direction, up) for a standard orientation, with ok=false
// for orientations with no fixed direction (the caller treats those as a no-op).
func orientationFrame(o types.ViewOrientationTypeEnum) (dir, up math.Vector3, ok bool) {
	f, ok := orientationFrames[o]
	return f.dir, f.up, ok
}
