// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"strconv"

	"oblikovati.org/model/feature"
)

// Building datum axes from the current selection. The model implements nine constructors and
// api/wire exposes them all, but the Work Features panel shipped five buttons and every one was
// a work PLANE — there was no Work Axis command anywhere in the application, so a user could not
// create a single axis (#2043, follow-up to #1851 which closed on the API alone).
//
// AddByLine (a raw origin + direction) has no interactive form: it takes coordinates rather than
// picks, and is the exchange/programmatic entry point.

// labelWorkAxis names a datum axis in the browser.
const labelWorkAxis = "Work Axis"

// uniqueWorkAxisName returns "Work Axis{n}" with the smallest n not already used, so browser
// nodes carry distinct labels (Dear ImGui derives a node id from the label).
func uniqueWorkAxisName(axes *feature.WorkAxes) string {
	taken := make(map[string]bool, axes.Count())
	for i := 0; i < axes.Count(); i++ {
		taken[axes.Item(i).Name()] = true
	}
	for n := 1; ; n++ {
		if name := labelWorkAxis + strconv.Itoa(n); !taken[name] {
			return name
		}
	}
}

// partWorkAxes is the slice of the active part finishWorkAxis needs.
type partWorkAxes interface {
	WorkAxes() *feature.WorkAxes
	Recompute()
}

// finishWorkAxis names a freshly created datum axis and recomputes the part — the shared tail of
// every Create*WorkAxis action.
func finishWorkAxis(part partWorkAxes, wa *feature.WorkAxis) *feature.WorkAxis {
	wa.SetName(uniqueWorkAxisName(part.WorkAxes()))
	part.Recompute()
	return wa
}

// addWorkAxis runs build against the active part's axis collection and finishes the datum, the
// shape every constructor below shares.
func (s *Session) addWorkAxis(build func(*feature.WorkAxes) *feature.WorkAxis) (*feature.WorkAxis, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	wa := finishWorkAxis(part, build(part.WorkAxes()))
	s.recordEdit(part, labelWorkAxis)
	return wa, nil
}

// selectedEdgeRefs gathers the selected B-rep edges as work references.
func (s *Session) selectedEdgeRefs() []feature.WorkRef {
	var refs []feature.WorkRef
	for _, it := range s.selection.Items() {
		if h, ok := it.(EdgeHandle); ok {
			refs = append(refs, feature.EdgeRef(h.Edge.ReferenceKey()))
		}
	}
	return refs
}

// CreateEdgeWorkAxis adds the axis along the selected linear model edge.
func (s *Session) CreateEdgeWorkAxis() (*feature.WorkAxis, error) {
	edges := s.selectedEdgeRefs()
	if len(edges) < 1 {
		return nil, errors.New("app: select a linear edge for an axis along it")
	}
	return s.addWorkAxis(func(c *feature.WorkAxes) *feature.WorkAxis { return c.AddByAnalyticEdge(edges[0]) })
}

// canEdgeWorkAxis enables it: an edge is selected.
func canEdgeWorkAxis(s *Session) bool { return !s.InSketch() && len(s.selectedEdgeRefs()) >= 1 }

// CreateTwoPointWorkAxis adds the axis through the two selected points.
func (s *Session) CreateTwoPointWorkAxis() (*feature.WorkAxis, error) {
	refs := s.selectedPointRefs()
	if len(refs) < 2 {
		return nil, errors.New("app: select two points (datum points or model vertices) for an axis through both")
	}
	return s.addWorkAxis(func(c *feature.WorkAxes) *feature.WorkAxis {
		return c.AddByTwoPoints(refs[0], refs[1])
	})
}

// canTwoPointWorkAxis enables it: two points are selected.
func canTwoPointWorkAxis(s *Session) bool { return !s.InSketch() && len(s.selectedPointRefs()) >= 2 }

// CreateRevolvedFaceWorkAxis adds the axis of the selected revolved face (a cylinder, cone,
// sphere or torus).
func (s *Session) CreateRevolvedFaceWorkAxis() (*feature.WorkAxis, error) {
	face, ok := s.selectedFaceRef()
	if !ok {
		return nil, errors.New("app: select a revolved face for its axis")
	}
	return s.addWorkAxis(func(c *feature.WorkAxes) *feature.WorkAxis { return c.AddByRevolvedFace(face) })
}

// canRevolvedFaceWorkAxis enables it: a face is selected.
func canRevolvedFaceWorkAxis(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, ok := s.SelectedFace()
	return ok
}

// CreatePlaneIntersectionWorkAxis adds the axis where the two selected planes meet.
func (s *Session) CreatePlaneIntersectionWorkAxis() (*feature.WorkAxis, error) {
	planes := s.SelectedWorkPlanes()
	if len(planes) < 2 {
		return nil, errors.New("app: select two planes for the axis where they intersect")
	}
	return s.addWorkAxis(func(c *feature.WorkAxes) *feature.WorkAxis {
		return c.AddByPlaneIntersection(planes[0].Key(), planes[1].Key())
	})
}

// canPlaneIntersectionWorkAxis enables it: two planes are selected.
func canPlaneIntersectionWorkAxis(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkPlanes()) >= 2
}

// CreateNormalToPlaneWorkAxis adds the axis through the selected point, normal to the selected
// plane.
func (s *Session) CreateNormalToPlaneWorkAxis() (*feature.WorkAxis, error) {
	plane, refs := s.SelectedWorkPlane(), s.selectedPointRefs()
	if plane == nil || len(refs) < 1 {
		return nil, errors.New("app: select a plane and a point for an axis normal to the plane")
	}
	return s.addWorkAxis(func(c *feature.WorkAxes) *feature.WorkAxis {
		return c.AddByPointAndPlane(refs[0], plane.Key())
	})
}

// canNormalToPlaneWorkAxis enables it: a plane and a point are selected.
func canNormalToPlaneWorkAxis(s *Session) bool {
	return !s.InSketch() && s.SelectedWorkPlane() != nil && len(s.selectedPointRefs()) >= 1
}

// CreateParallelToAxisWorkAxis adds the axis through the selected point, parallel to the
// selected axis.
func (s *Session) CreateParallelToAxisWorkAxis() (*feature.WorkAxis, error) {
	axes, refs := s.SelectedWorkAxes(), s.selectedPointRefs()
	if len(axes) < 1 || len(refs) < 1 {
		return nil, errors.New("app: select an axis and a point for a parallel axis through the point")
	}
	return s.addWorkAxis(func(c *feature.WorkAxes) *feature.WorkAxis {
		return c.AddByLineAndPoint(axes[0].Key(), refs[0])
	})
}

// canParallelToAxisWorkAxis enables it: an axis and a point are selected.
func canParallelToAxisWorkAxis(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkAxes()) >= 1 && len(s.selectedPointRefs()) >= 1
}

// CreateAxisOnPlaneWorkAxis adds the selected axis projected onto the selected plane.
func (s *Session) CreateAxisOnPlaneWorkAxis() (*feature.WorkAxis, error) {
	axes, plane := s.SelectedWorkAxes(), s.SelectedWorkPlane()
	if len(axes) < 1 || plane == nil {
		return nil, errors.New("app: select an axis and a plane to project the axis onto")
	}
	return s.addWorkAxis(func(c *feature.WorkAxes) *feature.WorkAxis {
		return c.AddByLineAndPlane(axes[0].Key(), plane.Key())
	})
}

// canAxisOnPlaneWorkAxis enables it: an axis and a plane are selected.
func canAxisOnPlaneWorkAxis(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkAxes()) >= 1 && s.SelectedWorkPlane() != nil
}
