// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// This file holds the 3D-sketch conic entities (M22-F03): full and partial ellipses in
// model space, thin wrappers over kernel/geom.EllipseFull / EllipticalArc. The center is
// a constrainable point; the major/minor radii are extra scalar DOFs; the plane normal
// and major-axis direction are definitional (set by the placement plane).

// Ellipse3D is a full ellipse in model space: a center point, a plane normal, an in-plane
// major-axis direction, and major/minor radius DOFs.
type Ellipse3D struct {
	entityBase
	Center      *Point3D
	Normal      math.UnitVector3
	MajorAxis   math.UnitVector3
	MajorRadius math.Scalar
	MinorRadius math.Scalar
}

// scalarDOFs makes Ellipse3D a [scalar3DContributor]: both radii are free DOFs.
func (e *Ellipse3D) scalarDOFs() []*math.Scalar {
	return []*math.Scalar{&e.MajorRadius, &e.MinorRadius}
}

// Curve returns the kernel ellipse, or an error if the plane is degenerate.
func (e *Ellipse3D) Curve() (geom.EllipseFull, error) {
	return geom.NewEllipseFull(e.Center.Position(), e.Normal.AsVector(), e.MajorAxis.AsVector(), float64(e.MajorRadius), float64(e.MinorRadius))
}

// EllipticalArc3D is a bounded ellipse: an [Ellipse3D] spanning StartAngle..StartAngle+
// SweepAngle (radians, signed about the normal).
type EllipticalArc3D struct {
	entityBase
	Center      *Point3D
	Normal      math.UnitVector3
	MajorAxis   math.UnitVector3
	MajorRadius math.Scalar
	MinorRadius math.Scalar
	StartAngle  float64
	SweepAngle  float64
}

// scalarDOFs makes EllipticalArc3D a [scalar3DContributor].
func (e *EllipticalArc3D) scalarDOFs() []*math.Scalar {
	return []*math.Scalar{&e.MajorRadius, &e.MinorRadius}
}

// Curve returns the kernel elliptical arc, or an error if the plane is degenerate.
func (e *EllipticalArc3D) Curve() (geom.EllipticalArc, error) {
	return geom.NewEllipticalArc(e.Center.Position(), e.Normal.AsVector(), e.MajorAxis.AsVector(),
		float64(e.MajorRadius), float64(e.MinorRadius), e.StartAngle, e.SweepAngle)
}

// AddEllipse3D adds a full ellipse from a center, plane normal, major-axis direction and
// the two radii.
func (s *Sketch3D) AddEllipse3D(center math.Point3, normal, majorAxis math.UnitVector3, majorR, minorR float64) *Ellipse3D {
	return s.addEllipse3DPt(s.newPoint3D(center), normal, majorAxis, majorR, minorR)
}

// AddEllipticalArc3D adds a bounded ellipse spanning startAngle..startAngle+sweepAngle.
func (s *Sketch3D) AddEllipticalArc3D(center math.Point3, normal, majorAxis math.UnitVector3, majorR, minorR, startAngle, sweepAngle float64) *EllipticalArc3D {
	return s.addEllipticalArc3DPt(s.newPoint3D(center), normal, majorAxis, majorR, minorR, startAngle, sweepAngle)
}

// addEllipse3DPt builds a full ellipse over an existing center point (the restore seam).
func (s *Sketch3D) addEllipse3DPt(center *Point3D, normal, majorAxis math.UnitVector3, majorR, minorR float64) *Ellipse3D {
	e := &Ellipse3D{
		entityBase: newEntity(), Center: center, Normal: normal, MajorAxis: majorAxis,
		MajorRadius: math.Scalar(majorR), MinorRadius: math.Scalar(minorR),
	}
	s.addEntity3D(e)
	return e
}

// addEllipticalArc3DPt builds a bounded ellipse over an existing center point.
func (s *Sketch3D) addEllipticalArc3DPt(center *Point3D, normal, majorAxis math.UnitVector3, majorR, minorR, startAngle, sweepAngle float64) *EllipticalArc3D {
	e := &EllipticalArc3D{
		entityBase: newEntity(), Center: center, Normal: normal, MajorAxis: majorAxis,
		MajorRadius: math.Scalar(majorR), MinorRadius: math.Scalar(minorR),
		StartAngle: startAngle, SweepAngle: sweepAngle,
	}
	s.addEntity3D(e)
	return e
}
