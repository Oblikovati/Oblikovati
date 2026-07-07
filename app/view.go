// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
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
	target, normal, up, ok := s.lookAtTarget()
	if !ok {
		return false
	}
	s.lookAtPlane(target, normal, up)
	return true
}

// lookAtTarget resolves the current selection to a plane the camera can face — a selected work plane
// or a planar face — returning its centre, normal and a stable screen-up. ok=false when the selection
// has no such target. Shared by LookAtSelection and CanLookAtSelection so the action and its
// enablement can never disagree (#1468 follow-up).
func (s *Session) lookAtTarget() (target math.Point3, normal, up math.Vector3, ok bool) {
	if wp := s.SelectedWorkPlane(); wp != nil {
		p := wp.Plane()
		return p.Origin(), p.Normal().AsVector(), p.YAxis().AsVector(), true
	}
	if f, sel := s.SelectedFace(); sel {
		if pl, planar := f.Geometry().(geom.Plane); planar {
			_, v := pl.DerivativesAt(0, 0) // the plane's in-plane v-axis is a stable screen-up
			return f.RangeBox().Center(), pl.Normal(), v, true
		}
	}
	return math.Point3{}, math.Vector3{}, math.Vector3{}, false
}

// CanLookAt reports whether the current selection has a planar reference LookAtSelection can face (a
// work plane or planar face) — the enable predicate for the Look At command, used to disable the
// Navigation Bar's Look At button when a click would be a no-op (#1468 follow-up). It shares
// lookAtTarget with the action, so the two can never disagree.
func (s *Session) CanLookAt() bool {
	_, _, _, ok := s.lookAtTarget()
	return ok
}

// ToggleSteeringWheel shows or hides the SteeringWheels radial navigation menu (#913 N26) — a
// transient on-cursor wheel of nav tools, entered from the View tab / Navigation Bar.
func (s *Session) ToggleSteeringWheel() { s.steeringWheel = !s.steeringWheel }

// SteeringWheelActive reports whether the SteeringWheels menu is shown.
func (s *Session) SteeringWheelActive() bool { return s.steeringWheel }

// DisarmSteeringWheel hides the SteeringWheels menu (e.g. Esc, or after a tool is chosen).
func (s *Session) DisarmSteeringWheel() { s.steeringWheel = false }

// ShowNavBar reports whether the floating Navigation Bar is shown in viewports (default true).
func (s *Session) ShowNavBar() bool { return !s.prefs.NavBarHidden }

// SetShowNavBar shows or hides the Navigation Bar (View-tab toggle), persisted as a global pref.
func (s *Session) SetShowNavBar(show bool) {
	s.prefs.NavBarHidden = !show
	s.savePrefs()
}

// ToggleConstrainedOrbit turns the Constrained Orbit tool (#913 N10) on or off — a turntable locked
// to the model vertical, entered from the View tab / Navigation Bar. While on, a viewport left-drag
// orbits about the vertical axis (horizontal = turn, vertical = tilt) instead of selecting.
func (s *Session) ToggleConstrainedOrbit() { s.constrainedOrbit = !s.constrainedOrbit }

// ConstrainedOrbitActive reports whether the Constrained Orbit tool is on.
func (s *Session) ConstrainedOrbitActive() bool { return s.constrainedOrbit }

// DisarmConstrainedOrbit turns the Constrained Orbit tool off (e.g. Esc).
func (s *Session) DisarmConstrainedOrbit() { s.constrainedOrbit = false }

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

// modelBounds is the union of the active part's body bounding boxes, its visible sketch geometry,
// its visible point clouds AND its placed mesh references, so Fit/Home frame a sketch-only part —
// e.g. a DWG/DXF import that produces a 2D sketch or a Sketch3D with no solid body (issue #1146) —
// a scan-only part whose only visible geometry is an attached point cloud (#1645), or a part whose
// only geometry is a placed reference mesh (#1773). Empty when there is nothing visible.
func (s *Session) modelBounds() math.Box {
	box := math.EmptyBox()
	for _, b := range s.sceneBodies() {
		box = box.Union(b.RangeBox())
	}
	return s.unionMeshBounds(s.unionCloudBounds(s.unionSketchBounds(box)))
}

// unionMeshBounds widens box by the model-space extent of the active part's visible placed mesh
// references, so a reference mesh placed into an otherwise empty part is framed by Fit/Home (#1773).
// A suppressed mesh contributes nothing.
func (s *Session) unionMeshBounds(box math.Box) math.Box {
	part, err := activePart(s)
	if err != nil {
		return box
	}
	feats := part.Features()
	for i := 0; i < feats.Count(); i++ {
		pf := feats.Item(i)
		if pf.Suppressed() {
			continue
		}
		mf, ok := pf.Definition().(*feature.MeshFeature)
		if !ok {
			continue
		}
		for _, v := range mf.Geometry().Vertices {
			box = box.ExtendPoint(v)
		}
	}
	return box
}

// unionCloudBounds widens box by the model-space extent of the active part's visible point clouds,
// so a scan attached into an otherwise empty part is framed by Fit/Home (#1645). An empty or hidden
// cloud contributes an empty box (the union identity), leaving box unchanged.
func (s *Session) unionCloudBounds(box math.Box) math.Box {
	part, err := activePart(s)
	if err != nil {
		return box
	}
	clouds := part.PointClouds()
	for i := 0; i < clouds.Count(); i++ {
		if pc := clouds.Item(i); pc.Visible() {
			box = box.Union(pc.RangeBox())
		}
	}
	return box
}

// PointCloudBounds returns the model-space extent of the active part's visible point clouds, or an
// empty box when none are attached. Point clouds render from a separate retained GL-points buffer
// that the viewport's instanced/overlay framing bounds never see, so the far clip plane must consult
// this to enclose a large or distant scan — otherwise the scan falls beyond the fixed far plane and
// renders as nothing (#1789).
func (s *Session) PointCloudBounds() math.Box {
	return s.unionCloudBounds(math.EmptyBox())
}

// unionSketchBounds widens box by the model-space extent of the active part's visible 2D and 3D
// sketches. Curves are sampled the same way the viewport overlay and ray picker sample them
// (EntityPolyline / SamplePolyline3D), so the framed box matches what is drawn. The sample
// points are framed ROBUSTLY (robustPointBox) so a handful of far-flung entities — e.g. a
// georeferenced DWG import's off-sheet strays — do not shrink Fit/Home to a sub-pixel dot;
// ordinary geometry, with no strays, is framed exactly.
func (s *Session) unionSketchBounds(box math.Box) math.Box {
	part, err := activePart(s)
	if err != nil {
		return box
	}
	var pts []math.Point3
	for i := 0; i < part.Sketches().Count(); i++ {
		if sk := part.Sketches().Item(i); sk.Visible() {
			pts = appendSketch2DPoints(pts, sk)
		}
	}
	for i := 0; i < part.Sketches3D().Count(); i++ {
		if sk := part.Sketches3D().Item(i); sk.Visible() {
			pts = appendSketch3DPoints(pts, sk)
		}
	}
	if len(pts) == 0 {
		return box
	}
	return box.Union(math.RobustPointBox(pts))
}

// appendSketch2DPoints appends a 2D sketch's entity sample points, mapped from sketch space to
// model space through the sketch plane.
func appendSketch2DPoints(pts []math.Point3, sk *sketch.Sketch) []math.Point3 {
	for _, e := range sk.Entities() {
		poly, _ := sketch.EntityPolyline(e)
		for _, p := range poly {
			pts = append(pts, sk.ToModel(p))
		}
	}
	return pts
}

// appendSketch3DPoints appends a 3D sketch's entity sample points (already in model space).
func appendSketch3DPoints(pts []math.Point3, sk *sketch.Sketch3D) []math.Point3 {
	for _, e := range sk.Entities() {
		pts = append(pts, sketch.SamplePolyline3D(e, sketch3DBoundsSegments)...)
	}
	return pts
}

// sketch3DBoundsSegments is the per-curve sample count for the 3D-sketch framing bounds — coarse
// is fine (Fit adds margin), and a low count keeps a dense import's Fit responsive.
const sketch3DBoundsSegments = 4
