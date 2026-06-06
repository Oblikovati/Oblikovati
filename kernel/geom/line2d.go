// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati/math"
)

// Line2d is an infinite 2D line through Origin along the unit direction Dir
// (contract: Line2d).
type Line2d struct {
	Origin math.Point2
	Dir    math.UnitVector2
}

// NewLine2d builds a 2D line through origin along dir; errors on a zero dir.
func NewLine2d(origin math.Point2, dir math.Vector2) (Line2d, error) {
	u, err := math.UnitVector2FromVector(dir)
	if err != nil {
		return Line2d{}, err
	}
	return Line2d{Origin: origin, Dir: u}, nil
}

// Line2dThrough builds the 2D line through two distinct points.
func Line2dThrough(from, to math.Point2) (Line2d, error) {
	return NewLine2d(from, from.VectorTo(to))
}

// PointAt returns Origin + t·Dir.
func (l Line2d) PointAt(t float64) math.Point2 {
	return l.Origin.TranslateBy(l.Dir.AsVector().Scale(t))
}

// TangentAt returns the constant line direction.
func (l Line2d) TangentAt(float64) math.Vector2 {
	return l.Dir.AsVector()
}

// Domain returns the full real line.
func (l Line2d) Domain() (lo, hi float64) {
	return stdmath.Inf(-1), stdmath.Inf(1)
}

// LineSegment2d is a bounded 2D line, parameterized t∈[0,1] (contract:
// LineSegment2d).
type LineSegment2d struct {
	StartPoint, EndPoint math.Point2
}

// NewLineSegment2d builds a 2D segment between two points.
func NewLineSegment2d(start, end math.Point2) LineSegment2d {
	return LineSegment2d{StartPoint: start, EndPoint: end}
}

// PointAt linearly interpolates from start (t=0) to end (t=1).
func (s LineSegment2d) PointAt(t float64) math.Point2 {
	return s.StartPoint.TranslateBy(s.StartPoint.VectorTo(s.EndPoint).Scale(t))
}

// TangentAt returns the start→end vector (constant).
func (s LineSegment2d) TangentAt(float64) math.Vector2 {
	return s.StartPoint.VectorTo(s.EndPoint)
}

// Domain returns [0, 1].
func (s LineSegment2d) Domain() (lo, hi float64) { return 0, 1 }

// Length returns the distance between the endpoints.
func (s LineSegment2d) Length() float64 {
	return s.StartPoint.DistanceTo(s.EndPoint)
}
