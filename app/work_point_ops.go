// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"strconv"

	"oblikovati.org/model/feature"
)

// Building datum points from the current selection. The model implements ten constructors and
// api/wire exposes them all, but exactly one was reachable from the UI — AddByCloudPoint, via the
// point-cloud snap — so a user could not place a datum point on a vertex, an edge midpoint or a
// plane intersection (#2043, follow-up to #1842 which closed on the API alone).

// uniqueWorkPointName returns "Work Point{n}" with the smallest n not already used. Every work
// point is minted as "WorkPoint", so a second one collided in the browser (Dear ImGui derives a
// node id from the label and asserts on duplicates — the same reason work planes are named here).
func uniqueWorkPointName(points *feature.WorkPoints) string {
	taken := make(map[string]bool, points.Count())
	for i := 0; i < points.Count(); i++ {
		taken[points.Item(i).Name()] = true
	}
	for n := 1; ; n++ {
		if name := labelWorkPoint + strconv.Itoa(n); !taken[name] {
			return name
		}
	}
}

// partWorkPoints is the slice of the active part finishWorkPoint needs.
type partWorkPoints interface {
	WorkPoints() *feature.WorkPoints
	Recompute()
}

// finishWorkPoint names a freshly created datum point and recomputes the part — the shared tail
// of every Create*WorkPoint action.
func finishWorkPoint(part partWorkPoints, wp *feature.WorkPoint) *feature.WorkPoint {
	wp.SetName(uniqueWorkPointName(part.WorkPoints()))
	part.Recompute()
	return wp
}

// addWorkPoint runs build against the active part's point collection and finishes the datum.
func (s *Session) addWorkPoint(build func(*feature.WorkPoints) *feature.WorkPoint) (*feature.WorkPoint, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	wp := finishWorkPoint(part, build(part.WorkPoints()))
	s.recordEdit(part, labelWorkPoint)
	return wp, nil
}

// CreateVertexWorkPoint adds a datum point at the selected vertex or datum point.
func (s *Session) CreateVertexWorkPoint() (*feature.WorkPoint, error) {
	refs := s.selectedPointRefs()
	if len(refs) < 1 {
		return nil, errors.New("app: select a vertex or datum point to place a work point on")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint { return c.AddByPoint(refs[0]) })
}

// canVertexWorkPoint enables it: a point reference is selected.
func canVertexWorkPoint(s *Session) bool { return !s.InSketch() && len(s.selectedPointRefs()) >= 1 }

// CreateMidpointWorkPoint adds a datum point at the midpoint of the selected edge.
func (s *Session) CreateMidpointWorkPoint() (*feature.WorkPoint, error) {
	edges := s.selectedEdgeRefs()
	if len(edges) < 1 {
		return nil, errors.New("app: select an edge to place a work point at its midpoint")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint {
		return c.AddByMidpointOfEdge(edges[0])
	})
}

// canMidpointWorkPoint enables it: an edge is selected.
func canMidpointWorkPoint(s *Session) bool { return !s.InSketch() && len(s.selectedEdgeRefs()) >= 1 }

// CreateCentroidWorkPoint adds a datum point at the length-weighted centroid of the selected
// edges.
func (s *Session) CreateCentroidWorkPoint() (*feature.WorkPoint, error) {
	edges := s.selectedEdgeRefs()
	if len(edges) < 1 {
		return nil, errors.New("app: select one or more edges to place a work point at their centroid")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint { return c.AddAtCentroid(edges...) })
}

// canCentroidWorkPoint enables it: at least one edge is selected.
func canCentroidWorkPoint(s *Session) bool { return !s.InSketch() && len(s.selectedEdgeRefs()) >= 1 }

// CreateFaceCenterWorkPoint adds a datum point at the centre of the selected spherical or
// toroidal face.
func (s *Session) CreateFaceCenterWorkPoint() (*feature.WorkPoint, error) {
	face, ok := s.selectedFaceRef()
	if !ok {
		return nil, errors.New("app: select a spherical or toroidal face for its centre point")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint { return c.AddByFaceCenter(face) })
}

// canFaceCenterWorkPoint enables it: a face is selected.
func canFaceCenterWorkPoint(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, ok := s.SelectedFace()
	return ok
}

// CreateThreePlaneWorkPoint adds a datum point where the three selected planes meet.
func (s *Session) CreateThreePlaneWorkPoint() (*feature.WorkPoint, error) {
	planes := s.SelectedWorkPlanes()
	if len(planes) < 3 {
		return nil, errors.New("app: select three planes for the point where they intersect")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint {
		return c.AddByThreePlanes(planes[0].Key(), planes[1].Key(), planes[2].Key())
	})
}

// canThreePlaneWorkPoint enables it: three planes are selected.
func canThreePlaneWorkPoint(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkPlanes()) >= 3
}

// CreateTwoAxisWorkPoint adds a datum point where the two selected axes meet.
func (s *Session) CreateTwoAxisWorkPoint() (*feature.WorkPoint, error) {
	axes := s.SelectedWorkAxes()
	if len(axes) < 2 {
		return nil, errors.New("app: select two axes for the point where they intersect")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint {
		return c.AddByTwoLines(axes[0].Key(), axes[1].Key())
	})
}

// canTwoAxisWorkPoint enables it: two axes are selected.
func canTwoAxisWorkPoint(s *Session) bool { return !s.InSketch() && len(s.SelectedWorkAxes()) >= 2 }

// CreatePlaneAndAxisWorkPoint adds a datum point where the selected axis pierces the selected
// plane.
func (s *Session) CreatePlaneAndAxisWorkPoint() (*feature.WorkPoint, error) {
	plane, axes := s.SelectedWorkPlane(), s.SelectedWorkAxes()
	if plane == nil || len(axes) < 1 {
		return nil, errors.New("app: select a plane and an axis for the point where they intersect")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint {
		return c.AddByPlaneAndAxisIntersection(plane.Key(), axes[0].Key())
	})
}

// canPlaneAndAxisWorkPoint enables it: a plane and an axis are selected.
func canPlaneAndAxisWorkPoint(s *Session) bool {
	return !s.InSketch() && s.SelectedWorkPlane() != nil && len(s.SelectedWorkAxes()) >= 1
}

// CreateCurveAndEntityWorkPoint adds a datum point where the selected edge meets the selected
// plane or face. The proximity hint is nil: with several intersections the definition takes the
// first, and the user disambiguates by picking a shorter edge — the API path carries the hint.
func (s *Session) CreateCurveAndEntityWorkPoint() (*feature.WorkPoint, error) {
	edges := s.selectedEdgeRefs()
	entity, ok := s.selectedCurveTarget()
	if len(edges) < 1 || !ok {
		return nil, errors.New("app: select an edge and a plane or face for the point where they intersect")
	}
	return s.addWorkPoint(func(c *feature.WorkPoints) *feature.WorkPoint {
		return c.AddByCurveAndEntity(edges[0], entity, nil)
	})
}

// selectedCurveTarget returns the surface a curve is intersected against: a selected plane, or a
// selected face.
func (s *Session) selectedCurveTarget() (feature.WorkRef, bool) {
	if wp := s.SelectedWorkPlane(); wp != nil {
		return wp.Key(), true
	}
	return s.selectedFaceRef()
}

// canCurveAndEntityWorkPoint enables it: an edge and a plane or face are selected.
func canCurveAndEntityWorkPoint(s *Session) bool {
	if s.InSketch() || len(s.selectedEdgeRefs()) < 1 {
		return false
	}
	_, ok := s.selectedCurveTarget()
	return ok
}
