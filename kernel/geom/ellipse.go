// SPDX-License-Identifier: GPL-2.0-only

package geom

import "github.com/Oblikovati/oblikovati/math"

// EllipseFull is a full 3D ellipse (contract: EllipseFull) in the plane through
// Center with unit Normal. MajorAxis (unit, in-plane) is the direction of the
// major radius. Parameterized t∈[0,1] mapping to angle 2πt:
// P(a) = Center + MajorRadius·cos(a)·MajorAxis + MinorRadius·sin(a)·(Normal×MajorAxis).
type EllipseFull struct {
	Center      math.Point3
	Normal      math.UnitVector3
	MajorAxis   math.UnitVector3
	MajorRadius float64
	MinorRadius float64
}

// NewEllipseFull builds a 3D ellipse; majorAxis is projected onto the plane and
// normalized. Errors on a zero normal.
func NewEllipseFull(center math.Point3, normal, majorAxis math.Vector3, majorR, minorR float64) (EllipseFull, error) {
	n, err := math.UnitVector3FromVector(normal)
	if err != nil {
		return EllipseFull{}, err
	}
	return EllipseFull{
		Center: center, Normal: n, MajorAxis: planarRef(n, majorAxis),
		MajorRadius: majorR, MinorRadius: minorR,
	}, nil
}

// minorAxis returns Normal × MajorAxis, the in-plane direction of the minor
// radius.
func (e EllipseFull) minorAxis() math.Vector3 {
	return e.Normal.Cross(e.MajorAxis)
}

// PointAt returns the point at angle 2πt.
func (e EllipseFull) PointAt(t float64) math.Point3 {
	return ellipsePoint3(e.Center, e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, twoPi*t)
}

// TangentAt returns the derivative dP/dt at parameter t.
func (e EllipseFull) TangentAt(t float64) math.Vector3 {
	return ellipseTangent3(e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, twoPi*t).Scale(twoPi)
}

// Domain returns [0, 1].
func (e EllipseFull) Domain() (lo, hi float64) { return 0, 1 }

// EllipticalArc is a partial 3D ellipse (contract: EllipticalArc): like
// [EllipseFull] but limited to StartAngle sweeping SweepAngle radians,
// parameterized t∈[0,1].
type EllipticalArc struct {
	Center      math.Point3
	Normal      math.UnitVector3
	MajorAxis   math.UnitVector3
	MajorRadius float64
	MinorRadius float64
	StartAngle  float64
	SweepAngle  float64
}

// NewEllipticalArc builds a partial 3D ellipse. Errors on a zero normal.
func NewEllipticalArc(center math.Point3, normal, majorAxis math.Vector3, majorR, minorR, startAngle, sweepAngle float64) (EllipticalArc, error) {
	n, err := math.UnitVector3FromVector(normal)
	if err != nil {
		return EllipticalArc{}, err
	}
	return EllipticalArc{
		Center: center, Normal: n, MajorAxis: planarRef(n, majorAxis),
		MajorRadius: majorR, MinorRadius: minorR, StartAngle: startAngle, SweepAngle: sweepAngle,
	}, nil
}

func (e EllipticalArc) minorAxis() math.Vector3 { return e.Normal.Cross(e.MajorAxis) }

// PointAt returns the point at parameter t.
func (e EllipticalArc) PointAt(t float64) math.Point3 {
	a := e.StartAngle + t*e.SweepAngle
	return ellipsePoint3(e.Center, e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, a)
}

// TangentAt returns the derivative dP/dt at parameter t.
func (e EllipticalArc) TangentAt(t float64) math.Vector3 {
	a := e.StartAngle + t*e.SweepAngle
	return ellipseTangent3(e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, a).Scale(e.SweepAngle)
}

// Domain returns [0, 1].
func (e EllipticalArc) Domain() (lo, hi float64) { return 0, 1 }

// ellipsePoint3 evaluates an ellipse point at angle a from its center and
// in-plane axes (each unit) and radii.
func ellipsePoint3(center math.Point3, major, minor math.Vector3, majorR, minorR, a float64) math.Point3 {
	cos, sin := cosSin(a)
	return center.TranslateBy(major.Scale(majorR * cos).Add(minor.Scale(minorR * sin)))
}

// ellipseTangent3 returns the derivative of [ellipsePoint3] with respect to a.
func ellipseTangent3(major, minor math.Vector3, majorR, minorR, a float64) math.Vector3 {
	cos, sin := cosSin(a)
	return major.Scale(-majorR * sin).Add(minor.Scale(minorR * cos))
}
