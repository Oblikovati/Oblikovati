// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// The work-plane constructors that were implemented and exposed over api/wire but had no ribbon
// path, so a user could not build them at all (#2044). Each mirrors the shape of the five that
// were already reachable: read the picks out of the selection, build, name, recompute, record.
//
// Three "toward" variants (AddByTwoPlanesToward, AddByPlaneAndTangentToward,
// AddByLineAndTangentToward) stay API-only on purpose: they take a proximity POINT that
// disambiguates which of two tangent solutions to take, and the interactive tools reach the same
// result by picking the face on the side the user wants. AddFixed likewise has no interactive
// form — it takes a raw origin and axes, which is an exchange/programmatic input, not a pick.

// selectedFaceRef returns the first selected B-rep face as a work reference.
func (s *Session) selectedFaceRef() (feature.WorkRef, bool) {
	f, ok := s.SelectedFace()
	if !ok {
		return "", false
	}
	return feature.FaceRef(f.ReferenceKey()), true
}

// CreateParallelThroughPointWorkPlane adds a work plane parallel to the selected plane, through
// the selected point.
func (s *Session) CreateParallelThroughPointWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	base, refs := s.SelectedWorkPlane(), s.selectedPointRefs()
	if base == nil || len(refs) < 1 {
		return nil, errors.New("app: select a plane and a point for a parallel-through-point plane")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByPlaneAndPoint(base.Key(), refs[0]))
	s.recordEdit(part, labelWorkPlane)
	return wp, nil
}

// canParallelThroughPointWorkPlane enables it: a plane and a point are selected.
func canParallelThroughPointWorkPlane(s *Session) bool {
	return !s.InSketch() && s.SelectedWorkPlane() != nil && len(s.selectedPointRefs()) >= 1
}

// CreateLineAndPointWorkPlane adds a work plane through the selected axis and point.
func (s *Session) CreateLineAndPointWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	axes, refs := s.SelectedWorkAxes(), s.selectedPointRefs()
	if len(axes) < 1 || len(refs) < 1 {
		return nil, errors.New("app: select an axis and a point for a plane through both")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByLineAndPoint(axes[0].Key(), refs[0]))
	s.recordEdit(part, labelWorkPlane)
	return wp, nil
}

// canLineAndPointWorkPlane enables it: an axis and a point are selected.
func canLineAndPointWorkPlane(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkAxes()) >= 1 && len(s.selectedPointRefs()) >= 1
}

// CreateTwoLinesWorkPlane adds a work plane containing the two selected axes.
func (s *Session) CreateTwoLinesWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	axes := s.SelectedWorkAxes()
	if len(axes) < 2 {
		return nil, errors.New("app: select two axes for a plane through both")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByTwoLines(axes[0].Key(), axes[1].Key()))
	s.recordEdit(part, labelWorkPlane)
	return wp, nil
}

// canTwoLinesWorkPlane enables it: two axes are selected.
func canTwoLinesWorkPlane(s *Session) bool {
	return !s.InSketch() && len(s.SelectedWorkAxes()) >= 2
}

// CreatePointAndTangentWorkPlane adds a work plane tangent to the selected curved face, through
// the selected point.
func (s *Session) CreatePointAndTangentWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	face, hasFace := s.selectedFaceRef()
	refs := s.selectedPointRefs()
	if !hasFace || len(refs) < 1 {
		return nil, errors.New("app: select a curved face and a point for a tangent-through-point plane")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByPointAndTangent(refs[0], face))
	s.recordEdit(part, labelWorkPlane)
	return wp, nil
}

// canPointAndTangentWorkPlane enables it: a face and a point are selected.
func canPointAndTangentWorkPlane(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, hasFace := s.SelectedFace()
	return hasFace && len(s.selectedPointRefs()) >= 1
}

// CreateLineAndTangentWorkPlane adds a work plane tangent to the selected curved face and
// containing the selected axis.
func (s *Session) CreateLineAndTangentWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	face, hasFace := s.selectedFaceRef()
	axes := s.SelectedWorkAxes()
	if !hasFace || len(axes) < 1 {
		return nil, errors.New("app: select a curved face and an axis for a tangent-through-axis plane")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByLineAndTangent(axes[0].Key(), face))
	s.recordEdit(part, labelWorkPlane)
	return wp, nil
}

// canLineAndTangentWorkPlane enables it: a face and an axis are selected.
func canLineAndTangentWorkPlane(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, hasFace := s.SelectedFace()
	return hasFace && len(s.SelectedWorkAxes()) >= 1
}

// CreateTorusMidPlaneWorkPlane adds the midplane of the selected toroidal face.
func (s *Session) CreateTorusMidPlaneWorkPlane() (*feature.WorkPlane, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	face, hasFace := s.selectedFaceRef()
	if !hasFace {
		return nil, errors.New("app: select a toroidal face for a torus midplane")
	}
	wp := finishWorkPlane(part, part.WorkPlanes().AddByTorusMidPlane(face))
	s.recordEdit(part, labelWorkPlane)
	return wp, nil
}

// canTorusMidPlaneWorkPlane enables it: a face is selected.
func canTorusMidPlaneWorkPlane(s *Session) bool {
	if s.InSketch() {
		return false
	}
	_, hasFace := s.SelectedFace()
	return hasFace
}
