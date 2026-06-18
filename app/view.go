// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// View navigation commands operate on the session camera. FitView frames the active
// part in the viewport keeping the current orientation (Inventor's Zoom All); HomeView
// switches to the default isometric view and frames it. Both are no-ops on an empty
// model. A ribbon command's Run calls these, and a test can call them directly.

// FitView reframes the camera so the whole active part fits the viewport. It writes
// through the active view (SetCamera) so the framing is remembered per view.
func (s *Session) FitView() {
	s.PushViewHistory() // record the view Previous View (F5) returns to
	s.SetCamera(s.Camera().Fit(s.modelBounds()))
}

// HomeView switches to the default isometric view, framed to fit the active part, writing
// through the active view so the framing is remembered.
func (s *Session) HomeView() { s.SetCamera(s.Camera().Home(s.modelBounds())) }

// LookAtSelection reorients the camera to look straight at the selected planar reference — a work
// plane or a planar face (Inventor's Look At, N18). It keeps the eye–target distance and swings the
// view with the standard tween, recording view history so Previous View returns. A selected work
// plane wins over a selected face. Reports whether a planar reference was selected (false ⇒ no-op).
func (s *Session) LookAtSelection() bool {
	if wp := s.SelectedWorkPlane(); wp != nil {
		p := wp.Plane()
		s.lookAtPlane(p.Origin(), p.Normal().AsVector(), p.YAxis().AsVector())
		return true
	}
	if f, ok := s.SelectedFace(); ok {
		if pl, planar := f.Geometry().(geom.Plane); planar {
			_, up := pl.DerivativesAt(0, 0) // the plane's in-plane v-axis is a stable screen-up
			s.lookAtPlane(f.RangeBox().Center(), pl.Normal(), up)
			return true
		}
	}
	return false
}

// CanLookAt reports whether the current selection has a planar reference LookAtSelection can face —
// the enable predicate for the Look At command.
func (s *Session) CanLookAt() bool {
	if s.SelectedWorkPlane() != nil {
		return true
	}
	f, ok := s.SelectedFace()
	if !ok {
		return false
	}
	_, planar := f.Geometry().(geom.Plane)
	return planar
}

// SetOrbitPivot recenters the orbit on the world point under viewport pixel (x,y) — Free Orbit's
// click-to-set-pivot (#913 N9). The clicked point becomes the orbit centre and is brought to the
// view centre, keeping the view direction and distance.
func (s *Session) SetOrbitPivot(x, y float64) {
	s.SetCamera(s.Camera().SetPivotUnderCursor(x, y))
}

// lookAtPlane swings the camera to face the plane at target with the given normal and up.
func (s *Session) lookAtPlane(target math.Point3, normal, up math.Vector3) {
	s.PushViewHistory()
	s.animateCameraTo(s.Camera().Facing(target, normal, up), sketchViewTweenSeconds)
}

// modelBounds is the union of the active part's body bounding boxes (empty if none).
func (s *Session) modelBounds() math.Box {
	box := math.EmptyBox()
	for _, b := range s.sceneBodies() {
		box = box.Union(b.RangeBox())
	}
	return box
}
