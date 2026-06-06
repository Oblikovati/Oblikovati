// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// This file holds the 3D-sketch curve entities (M22-F02): straight lines, circles and
// arcs that live in model space. Each is a thin model wrapper over a kernel/geom 3D
// primitive, constrainable through the dimension-agnostic solver (its defining points
// carry three DOFs each; a circle's radius is an extra scalar DOF). Point3D itself is in
// constraints_3d.go (it predates this file as the solver's 3D point carrier).

// Line3D is a straight segment between two constrainable 3D endpoints.
type Line3D struct {
	entityBase
	A *Point3D
	B *Point3D
}

// StartPoint and EndPoint return the line's endpoints.
func (l *Line3D) StartPoint() *Point3D { return l.A }
func (l *Line3D) EndPoint() *Point3D   { return l.B }

// Direction returns the (unnormalized) vector from A to B.
func (l *Line3D) Direction() math.Vector3 { return l.A.Position().VectorTo(l.B.Position()) }

// Length returns the current endpoint distance.
func (l *Line3D) Length() math.Scalar { return l.A.Position().DistanceTo(l.B.Position()) }

// Segment returns the kernel line segment for this entity (rendering/tessellation).
func (l *Line3D) Segment() geom.LineSegment {
	return geom.NewLineSegment(l.A.Position(), l.B.Position())
}

// Circle3D is a full circle in model space: a center point, a plane normal (axis), and a
// radius DOF.
type Circle3D struct {
	entityBase
	Center *Point3D
	Axis   math.UnitVector3
	Radius math.Scalar
}

// CenterPoint returns the circle's center; CircleRadius its radius.
func (c *Circle3D) CenterPoint() *Point3D     { return c.Center }
func (c *Circle3D) CircleRadius() math.Scalar { return c.Radius }

// scalarDOFs makes Circle3D a [scalar3DContributor]: its radius is a free DOF the solver
// must count beyond the center point's three.
func (c *Circle3D) scalarDOFs() []*math.Scalar { return []*math.Scalar{&c.Radius} }

// Curve returns the kernel circle for this entity, or an error if the axis is degenerate.
func (c *Circle3D) Curve() (geom.Circle, error) {
	return geom.NewCircle(c.Center.Position(), c.Axis.AsVector(), float64(c.Radius))
}

// Arc3D is a circular arc through model space, defined by a center and two endpoints; the
// radius is the center-to-start distance. CounterClockwise sets the sweep direction about
// the arc plane's normal (center→start × center→end).
type Arc3D struct {
	entityBase
	Center           *Point3D
	Start            *Point3D
	End              *Point3D
	CounterClockwise bool
}

// Radius returns the current center-to-start distance.
func (a *Arc3D) Radius() math.Scalar { return a.Center.Position().DistanceTo(a.Start.Position()) }

// Curve returns the kernel arc through this entity's three points, or an error if they
// are collinear/degenerate.
func (a *Arc3D) Curve() (geom.Arc3d, error) {
	return geom.Arc3dByThreePoints(a.Start.Position(), a.midOnArc(), a.End.Position())
}

// midOnArc returns a point on the arc between start and end, used to seed the kernel's
// three-point arc construction from this entity's center/start/end definition.
func (a *Arc3D) midOnArc() math.Point3 {
	r := float64(a.Radius())
	mid := a.Start.Position().Midpoint(a.End.Position())
	dir := a.Center.Position().VectorTo(mid)
	if float64(dir.Length()) < math.DefaultTolerance {
		return mid // start and end are antipodal through the center; mid is on the arc
	}
	u := dir.Scale(math.Scalar(r) / dir.Length())
	return a.Center.Position().TranslateBy(u)
}

// AddLine3D adds a straight 3D segment between two new endpoints.
func (s *Sketch3D) AddLine3D(a, b math.Point3) *Line3D {
	return s.addLine3DPts(s.newPoint3D(a), s.newPoint3D(b))
}

// AddCircle3D adds a full circle from a center, plane axis, and radius.
func (s *Sketch3D) AddCircle3D(center math.Point3, axis math.UnitVector3, radius float64) *Circle3D {
	return s.addCircle3DPts(s.newPoint3D(center), axis, radius)
}

// AddArc3D adds a circular arc from a center, start and end point.
func (s *Sketch3D) AddArc3D(center, start, end math.Point3, ccw bool) *Arc3D {
	return s.addArc3DPts(s.newPoint3D(center), s.newPoint3D(start), s.newPoint3D(end), ccw)
}

// addLine3DPts builds a line over existing points (the seam restore and chaining use).
func (s *Sketch3D) addLine3DPts(a, b *Point3D) *Line3D {
	l := &Line3D{entityBase: newEntity(), A: a, B: b}
	s.addEntity3D(l)
	return l
}

// addCircle3DPts builds a circle over an existing center point.
func (s *Sketch3D) addCircle3DPts(center *Point3D, axis math.UnitVector3, radius float64) *Circle3D {
	c := &Circle3D{entityBase: newEntity(), Center: center, Axis: axis, Radius: math.Scalar(radius)}
	s.addEntity3D(c)
	return c
}

// addArc3DPts builds an arc over existing center/start/end points.
func (s *Sketch3D) addArc3DPts(center, start, end *Point3D, ccw bool) *Arc3D {
	a := &Arc3D{entityBase: newEntity(), Center: center, Start: start, End: end, CounterClockwise: ccw}
	s.addEntity3D(a)
	return a
}
