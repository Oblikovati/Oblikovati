// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati/math"
)

// Arc2d is a 2D circular arc (contract: Arc2d): the locus from StartAngle
// sweeping SweepAngle radians (signed; positive is counter-clockwise) on a
// circle of the given Center and Radius. Parameterized t∈[0,1] mapping to angle
// StartAngle + t·SweepAngle.
type Arc2d struct {
	Center     math.Point2
	Radius     float64
	StartAngle float64
	SweepAngle float64
}

// NewArc2d builds an arc from center, radius, start angle, and signed sweep.
func NewArc2d(center math.Point2, radius, startAngle, sweepAngle float64) Arc2d {
	return Arc2d{Center: center, Radius: radius, StartAngle: startAngle, SweepAngle: sweepAngle}
}

// Arc2dByThreePoints builds the arc that starts at start, passes through onArc,
// and ends at end. It errors when the three points are collinear.
func Arc2dByThreePoints(start, onArc, end math.Point2) (Arc2d, error) {
	center, ok := circumcenter2d(start, onArc, end)
	if !ok {
		return Arc2d{}, &CollinearPointsError{A: start, B: onArc, C: end}
	}
	a0 := angleOf2d(center, start)
	sweep := wrapPositive(angleOf2d(center, end) - a0)
	if signedArea2(start, onArc, end) < 0 { // clockwise winding ⇒ negative sweep
		sweep -= twoPi
	}
	return Arc2d{Center: center, Radius: center.DistanceTo(start), StartAngle: a0, SweepAngle: sweep}, nil
}

// angleOf2d returns the angle of the vector from center to p, in (−π, π].
func angleOf2d(center, p math.Point2) float64 {
	v := center.VectorTo(p)
	return stdmath.Atan2(v.Y, v.X)
}

// angleAt returns the absolute angle StartAngle + t·SweepAngle.
func (a Arc2d) angleAt(t float64) float64 {
	return a.StartAngle + t*a.SweepAngle
}

// PointAt returns the point at parameter t.
func (a Arc2d) PointAt(t float64) math.Point2 {
	cos, sin := cosSin(a.angleAt(t))
	return math.P2(a.Center.X+a.Radius*cos, a.Center.Y+a.Radius*sin)
}

// TangentAt returns the derivative dP/dt (includes the sweep chain factor).
func (a Arc2d) TangentAt(t float64) math.Vector2 {
	cos, sin := cosSin(a.angleAt(t))
	return math.V2(-a.Radius*sin*a.SweepAngle, a.Radius*cos*a.SweepAngle)
}

// Domain returns [0, 1].
func (a Arc2d) Domain() (lo, hi float64) { return 0, 1 }

// Length returns the arc length |Radius · SweepAngle|.
func (a Arc2d) Length() float64 {
	return stdmath.Abs(a.Radius * a.SweepAngle)
}

// ContainsAngle reports whether the absolute angle theta lies within the arc's swept
// range (inclusive, within tol radians), for either sweep direction. A full-circle arc
// (|SweepAngle| ≥ 2π) contains every angle.
func (a Arc2d) ContainsAngle(theta, tol float64) bool {
	if tol < 0 {
		tol = 0
	}
	if a.SweepAngle >= 0 {
		return wrapPositive(theta-a.StartAngle) <= a.SweepAngle+tol
	}
	return wrapPositive(a.StartAngle-theta) <= -a.SweepAngle+tol
}

// ContainsPoint reports whether p — assumed on or near the arc's circle — lies within the
// arc's angular sweep. The geometric tol is converted to an angular slack (tol/Radius) so
// an endpoint touch still counts; used to filter circle-crossing points onto the arc.
func (a Arc2d) ContainsPoint(p math.Point2, tol float64) bool {
	if tol < 0 {
		tol = 0
	}
	angTol := 0.0
	if a.Radius > 0 {
		angTol = tol / a.Radius
	}
	return a.ContainsAngle(angleOf2d(a.Center, p), angTol)
}
