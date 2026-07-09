// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Surface-derived datums read an analytic B-rep face (resolved through the work resolver's
// surface() — the same path the tangent planes use) and extract a frame from it: the axis of a
// surface of revolution, or the centre of a sphere/torus. They need no ADR-0040 edge selectors,
// only the existing face→surface resolution (#1840, #1842).

// revolvedFaceAxisDef is the axis of revolution of a cylindrical, conical, or toroidal face
// (Inventor's WorkAxes.AddByRevolvedFace). A face with no axis of revolution (plane, sphere, NURBS)
// goes sick rather than producing garbage. #1840.
type revolvedFaceAxisDef struct{ face WorkRef }

func (d revolvedFaceAxisDef) kindName() string { return "revolved-face" }
func (d revolvedFaceAxisDef) refs() []WorkRef  { return []WorkRef{d.face} }
func (d revolvedFaceAxisDef) eval(r workResolver) (math.Point3, math.UnitVector3, error) {
	s, err := r.surface(d.face)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	return revolvedFaceAxis(s)
}

// revolvedFaceAxis returns an axis point and unit direction for a surface of revolution.
func revolvedFaceAxis(s geom.Surface) (math.Point3, math.UnitVector3, error) {
	switch g := s.(type) {
	case geom.Cylinder:
		return g.Origin, g.AxisDir, nil
	case geom.Cone:
		return g.Apex, g.AxisDir, nil
	case geom.Torus:
		return g.Center, g.AxisDir, nil
	default:
		return math.Point3{}, math.UnitVector3{}, fmt.Errorf("a %T face has no axis of revolution", s)
	}
}

// AddByRevolvedFace creates the axis of revolution of a cylindrical/conical/toroidal face (#1840).
func (c *WorkAxes) AddByRevolvedFace(face WorkRef) *WorkAxis {
	return c.addUser(revolvedFaceAxisDef{face: face})
}

// faceCenterPointDef is the centre of a spherical or toroidal face (Inventor's
// AddByCenterOfSphereFace / AddByCenterOfTorusFace). A face with no centre point goes sick. #1842.
type faceCenterPointDef struct{ face WorkRef }

func (d faceCenterPointDef) kindName() string { return "face-center" }
func (d faceCenterPointDef) refs() []WorkRef  { return []WorkRef{d.face} }
func (d faceCenterPointDef) eval(r workResolver) (math.Point3, error) {
	s, err := r.surface(d.face)
	if err != nil {
		return math.Point3{}, err
	}
	return faceCenter(s)
}

// faceCenter returns the centre point of a sphere or torus surface.
func faceCenter(s geom.Surface) (math.Point3, error) {
	switch g := s.(type) {
	case geom.Sphere:
		return g.Center, nil
	case geom.Torus:
		return g.Center, nil
	default:
		return math.Point3{}, fmt.Errorf("a %T face has no centre point", s)
	}
}

// AddByFaceCenter creates the datum point at the centre of a spherical/toroidal face (#1842).
func (c *WorkPoints) AddByFaceCenter(face WorkRef) *WorkPoint {
	return c.addUser(faceCenterPointDef{face: face})
}
