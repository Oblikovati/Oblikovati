// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// Plane is a planar sketch's host coordinate system in model space: an origin and
// two orthonormal in-plane axes, with the normal derived as xAxis × yAxis. It maps
// sketch 2D coordinates to model 3D and back (modeling/00). A real sketch hosts on
// a work-plane or planar face (reference-keyed, core/05); the geometry is the same.
type Plane struct {
	origin math.Point3
	xAxis  math.UnitVector3
	yAxis  math.UnitVector3
	normal math.UnitVector3
}

// NewPlane builds a sketch plane from an origin and two in-plane axes, which must
// be perpendicular unit vectors. The normal is xAxis × yAxis (right-handed).
func NewPlane(origin math.Point3, xAxis, yAxis math.UnitVector3) (Plane, error) {
	if !xAxis.IsPerpendicularTo(yAxis, math.DefaultTolerance) {
		return Plane{}, fmt.Errorf("sketch: plane axes are not perpendicular: x=%v y=%v", xAxis, yAxis)
	}
	normal, err := math.UnitVector3FromVector(xAxis.Cross(yAxis))
	if err != nil {
		return Plane{}, fmt.Errorf("sketch: degenerate plane axes: %w", err)
	}
	return Plane{origin: origin, xAxis: xAxis, yAxis: yAxis, normal: normal}, nil
}

// XYPlane, XZPlane and YZPlane are the three origin-based standard planes, matching
// the work-planes a new part document provides.
func XYPlane() Plane { return mustPlane(math.P3(0, 0, 0), axisX, axisY) }
func XZPlane() Plane { return mustPlane(math.P3(0, 0, 0), axisX, axisZ) }
func YZPlane() Plane { return mustPlane(math.P3(0, 0, 0), axisY, axisZ) }

// Origin returns the plane origin in model space.
func (p Plane) Origin() math.Point3 { return p.origin }

// XAxis, YAxis and Normal return the plane's frame in model space.
func (p Plane) XAxis() math.UnitVector3  { return p.xAxis }
func (p Plane) YAxis() math.UnitVector3  { return p.yAxis }
func (p Plane) Normal() math.UnitVector3 { return p.normal }

// ToModel maps a sketch-space 2D point to its model-space 3D position:
// origin + u·xAxis + v·yAxis.
func (p Plane) ToModel(s math.Point2) math.Point3 {
	return p.origin.
		TranslateBy(p.xAxis.AsVector().Scale(s.X)).
		TranslateBy(p.yAxis.AsVector().Scale(s.Y))
}

// ToSketch maps a model-space 3D point to sketch space by projecting it onto the
// plane: u = (q-origin)·xAxis, v = (q-origin)·yAxis. Any out-of-plane component is
// dropped (orthogonal projection).
func (p Plane) ToSketch(q math.Point3) math.Point2 {
	d := p.origin.VectorTo(q)
	return math.P2(d.Dot(p.xAxis.AsVector()), d.Dot(p.yAxis.AsVector()))
}

// axisX/Y/Z are the model-space basis vectors used to build the standard planes.
var (
	axisX = mustUnit(1, 0, 0)
	axisY = mustUnit(0, 1, 0)
	axisZ = mustUnit(0, 0, 1)
)

func mustUnit(x, y, z math.Scalar) math.UnitVector3 {
	u, err := math.NewUnitVector3(x, y, z)
	if err != nil {
		panic(err) // basis vectors are constant and always valid
	}
	return u
}

func mustPlane(origin math.Point3, x, y math.UnitVector3) Plane {
	pl, err := NewPlane(origin, x, y)
	if err != nil {
		panic(err) // standard planes are constant and always valid
	}
	return pl
}
