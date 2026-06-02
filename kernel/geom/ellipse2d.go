// SPDX-License-Identifier: GPL-2.0-only

package geom

import "github.com/Oblikovati/oblikovati/math"

// EllipseFull2d is a full 2D ellipse (contract: EllipseFull2d). MajorAxis (unit)
// is the major-radius direction; the minor axis is its +90° rotation. Angle 2πt
// parameterizes t∈[0,1].
type EllipseFull2d struct {
	Center      math.Point2
	MajorAxis   math.UnitVector2
	MajorRadius float64
	MinorRadius float64
}

// NewEllipseFull2d builds a 2D ellipse. Errors on a zero major-axis direction.
func NewEllipseFull2d(center math.Point2, majorAxis math.Vector2, majorR, minorR float64) (EllipseFull2d, error) {
	u, err := math.UnitVector2FromVector(majorAxis)
	if err != nil {
		return EllipseFull2d{}, err
	}
	return EllipseFull2d{Center: center, MajorAxis: u, MajorRadius: majorR, MinorRadius: minorR}, nil
}

// minorAxis returns the major axis rotated +90° (counter-clockwise).
func (e EllipseFull2d) minorAxis() math.Vector2 {
	return math.V2(-e.MajorAxis.Y(), e.MajorAxis.X())
}

// PointAt returns the point at angle 2πt.
func (e EllipseFull2d) PointAt(t float64) math.Point2 {
	return ellipsePoint2(e.Center, e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, twoPi*t)
}

// TangentAt returns the derivative dP/dt at parameter t.
func (e EllipseFull2d) TangentAt(t float64) math.Vector2 {
	return ellipseTangent2(e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, twoPi*t).Scale(twoPi)
}

// Domain returns [0, 1].
func (e EllipseFull2d) Domain() (lo, hi float64) { return 0, 1 }

// EllipticalArc2d is a partial 2D ellipse (contract: EllipticalArc2d).
type EllipticalArc2d struct {
	Center      math.Point2
	MajorAxis   math.UnitVector2
	MajorRadius float64
	MinorRadius float64
	StartAngle  float64
	SweepAngle  float64
}

// NewEllipticalArc2d builds a partial 2D ellipse. Errors on a zero major axis.
func NewEllipticalArc2d(center math.Point2, majorAxis math.Vector2, majorR, minorR, startAngle, sweepAngle float64) (EllipticalArc2d, error) {
	u, err := math.UnitVector2FromVector(majorAxis)
	if err != nil {
		return EllipticalArc2d{}, err
	}
	return EllipticalArc2d{
		Center: center, MajorAxis: u, MajorRadius: majorR, MinorRadius: minorR,
		StartAngle: startAngle, SweepAngle: sweepAngle,
	}, nil
}

func (e EllipticalArc2d) minorAxis() math.Vector2 {
	return math.V2(-e.MajorAxis.Y(), e.MajorAxis.X())
}

// PointAt returns the point at parameter t.
func (e EllipticalArc2d) PointAt(t float64) math.Point2 {
	a := e.StartAngle + t*e.SweepAngle
	return ellipsePoint2(e.Center, e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, a)
}

// TangentAt returns the derivative dP/dt at parameter t.
func (e EllipticalArc2d) TangentAt(t float64) math.Vector2 {
	a := e.StartAngle + t*e.SweepAngle
	return ellipseTangent2(e.MajorAxis.AsVector(), e.minorAxis(), e.MajorRadius, e.MinorRadius, a).Scale(e.SweepAngle)
}

// Domain returns [0, 1].
func (e EllipticalArc2d) Domain() (lo, hi float64) { return 0, 1 }

// ellipsePoint2 evaluates a 2D ellipse point at angle a.
func ellipsePoint2(center math.Point2, major, minor math.Vector2, majorR, minorR, a float64) math.Point2 {
	cos, sin := cosSin(a)
	return center.TranslateBy(major.Scale(majorR * cos).Add(minor.Scale(minorR * sin)))
}

// ellipseTangent2 returns the derivative of [ellipsePoint2] with respect to a.
func ellipseTangent2(major, minor math.Vector2, majorR, minorR, a float64) math.Vector2 {
	cos, sin := cosSin(a)
	return major.Scale(-majorR * sin).Add(minor.Scale(minorR * cos))
}
