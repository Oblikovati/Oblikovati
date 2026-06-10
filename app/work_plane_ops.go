// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"strconv"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// Work-plane creation from the 3D Model ribbon's Work Features panel. These mirror the
// Sketch.Create2D flow: read the current selection, add the datum to the active part's
// work geometry, and recompute so the browser and 3D view show it. The selection-driven
// constructors (midplane, three points, tangent, normal-to-axis) build directly from the
// picked entities; the Offset constructor needs a distance value, so it has its own tool
// with a dialog ([OffsetWorkPlaneTool]) rather than a one-shot creation method here.

// SelectedWorkPlanes returns every selected work plane, in selection order — the inputs
// for the multi-plane constructors (e.g. a midplane between two planes).
func (s *Session) SelectedWorkPlanes() []*feature.WorkPlane {
	var planes []*feature.WorkPlane
	for _, it := range s.selection.Items() {
		if h, ok := it.(WorkPlaneHandle); ok {
			planes = append(planes, h.Plane)
		}
	}
	return planes
}

// PickableWorkPlanes returns the work planes the viewport hit-test should offer: every
// origin plane (always pickable as a sketch host — Inventor's Origin folder — even though
// origin planes default to hidden) plus every VISIBLE user-created datum not hidden by the
// active edit scope (a plane the overlays don't draw must not be clickable either). User
// planes were previously absent from the picker, so a ribbon-created plane could not be
// clicked as a new sketch's reference in the 3D view (issue #132); the head feeds this to
// the RayPicker.
func (s *Session) PickableWorkPlanes() []*feature.WorkPlane {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	planes := part.WorkPlanes()
	out := make([]*feature.WorkPlane, 0, planes.Count())
	for i := 0; i < planes.Count(); i++ {
		wp := planes.Item(i)
		if wp.IsCoordinateSystemElement() || (wp.Visible() && !s.EditScopeHides(wp.Seq())) {
			out = append(out, wp)
		}
	}
	return out
}

// CreateMidplaneWorkPlane adds a work plane bisecting the two selected planes, then
// recomputes. It errors when fewer than two work planes are selected.
func (s *Session) CreateMidplaneWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	planes := s.SelectedWorkPlanes()
	if len(planes) < 2 {
		return nil, errors.New("app: select two work planes to make a midplane between them")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByTwoPlanes(planes[0].Key(), planes[1].Key()))
	s.recordEdit(part, "Work Plane")
	return wp, nil
}

// uniqueWorkPlaneName returns "Work Plane{n}" with the smallest n not already used by a
// work plane, so browser nodes carry distinct labels (Dear ImGui derives a node id from
// the label and asserts on duplicates — see PartFeatures.UniqueName).
func uniqueWorkPlaneName(planes *feature.WorkPlanes) string {
	taken := make(map[string]bool, planes.Count())
	for i := 0; i < planes.Count(); i++ {
		taken[planes.Item(i).Name()] = true
	}
	for n := 1; ; n++ {
		if name := "Work Plane" + strconv.Itoa(n); !taken[name] {
			return name
		}
	}
}

// canMidplaneWorkPlane enables the Midplane command: two work planes are selected.
func canMidplaneWorkPlane(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkPlanes()) >= 2
}

// ToggleSelectedWorkPlaneVisibility flips the Visible flag of every selected work plane —
// the action behind the browser's Visibility menu item and the V keyboard shortcut. The
// viewport rebuilds from live state each frame, so the change shows immediately with no
// recompute (visibility is display-only).
func (s *Session) ToggleSelectedWorkPlaneVisibility() {
	for _, wp := range s.SelectedWorkPlanes() {
		wp.SetVisible(!wp.Visible())
	}
}

// SelectedWorkAxes returns every selected datum axis, in selection order.
func (s *Session) SelectedWorkAxes() []*feature.WorkAxis {
	var axes []*feature.WorkAxis
	for _, it := range s.selection.Items() {
		if h, ok := it.(WorkAxisHandle); ok {
			axes = append(axes, h.Axis)
		}
	}
	return axes
}

// SelectedFace returns the first selected B-rep face, or false.
func (s *Session) SelectedFace() (*topo.Face, bool) {
	for _, it := range s.selection.Items() {
		if h, ok := it.(FaceHandle); ok {
			return h.Face, true
		}
	}
	return nil, false
}

// selectedPointRefs gathers point references from the selection — datum points by their
// key and B-rep vertices by a vertex reference — the point inputs the three-point and
// normal-to-axis constructors accept.
func (s *Session) selectedPointRefs() []feature.WorkRef {
	var refs []feature.WorkRef
	for _, it := range s.selection.Items() {
		switch h := it.(type) {
		case WorkPointHandle:
			refs = append(refs, h.Point.Key())
		case VertexHandle:
			refs = append(refs, feature.VertexRef(h.Vertex.ReferenceKey()))
		}
	}
	return refs
}

// CreateThreePointWorkPlane adds a work plane through the first three selected points
// (datum points or model vertices), then recomputes.
func (s *Session) CreateThreePointWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	refs := s.selectedPointRefs()
	if len(refs) < 3 {
		return nil, errors.New("app: select three points (datum points or model vertices) for a three-point plane")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByThreePoints(refs[0], refs[1], refs[2]))
	s.recordEdit(part, "Work Plane")
	return wp, nil
}

// CreateNormalToAxisWorkPlane adds a work plane through the selected point, normal to the
// selected axis, then recomputes.
func (s *Session) CreateNormalToAxisWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	axes, refs := s.SelectedWorkAxes(), s.selectedPointRefs()
	if len(axes) < 1 || len(refs) < 1 {
		return nil, errors.New("app: select an axis and a point for a normal-to-axis plane")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByNormalToCurve(axes[0].Key(), refs[0]))
	s.recordEdit(part, "Work Plane")
	return wp, nil
}

// CreateTangentWorkPlane adds a work plane parallel to the selected plane and tangent to
// the selected face (a cylinder/sphere), then recomputes.
func (s *Session) CreateTangentWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	base := s.SelectedWorkPlane()
	face, ok := s.SelectedFace()
	if base == nil || !ok {
		return nil, errors.New("app: select a plane and a face for a tangent plane")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByPlaneAndTangent(base.Key(), feature.FaceRef(face.ReferenceKey())))
	s.recordEdit(part, "Work Plane")
	return wp, nil
}

// finishWorkPlane gives a freshly created datum a unique browser name and recomputes the
// part, the shared tail of every Create*WorkPlane action.
func finishWorkPlane(part partWorkPlanes, wp *feature.WorkPlane) *feature.WorkPlane {
	wp.SetName(uniqueWorkPlaneName(part.WorkPlanes()))
	part.Recompute()
	return wp
}

// partWorkPlanes is the slice of the active part finishWorkPlane needs (its datum-plane
// collection and recompute), kept small so the helper does not depend on the whole part.
type partWorkPlanes interface {
	WorkPlanes() *feature.WorkPlanes
	Recompute()
}

// canThreePointWorkPlane enables Three Points: three point references are selected.
func canThreePointWorkPlane(s *Session) bool {
	return !s.InSketch() && len(s.selectedPointRefs()) >= 3
}

// canNormalToAxisWorkPlane enables Normal to Axis: an axis and a point are selected.
func canNormalToAxisWorkPlane(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkAxes()) >= 1 && len(s.selectedPointRefs()) >= 1
}

// canTangentWorkPlane enables Tangent to Face: a plane and a face are selected.
func canTangentWorkPlane(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, hasFace := s.SelectedFace()
	return hasFace && s.SelectedWorkPlane() != nil
}
