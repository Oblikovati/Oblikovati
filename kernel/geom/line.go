// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Line is an infinite 3D line through Origin along the unit direction Dir
// (contract: Line). PointAt(t) = Origin + t·Dir, so the parameter t is signed
// arc length from Origin.
type Line struct {
	Origin math.Point3
	Dir    math.UnitVector3
}

// NewLine builds a Line through origin along dir. It returns an error when dir
// is zero length (no direction), via [math.NewUnitVector3].
func NewLine(origin math.Point3, dir math.Vector3) (Line, error) {
	u, err := math.UnitVector3FromVector(dir)
	if err != nil {
		return Line{}, err
	}
	return Line{Origin: origin, Dir: u}, nil
}

// LineThrough builds the line passing through two distinct points.
func LineThrough(from, to math.Point3) (Line, error) {
	return NewLine(from, from.VectorTo(to))
}

// PointAt returns Origin + t·Dir.
func (l Line) PointAt(t float64) math.Point3 {
	return l.Origin.TranslateBy(l.Dir.AsVector().Scale(t))
}

// TangentAt returns the line direction (constant for all t).
func (l Line) TangentAt(float64) math.Vector3 {
	return l.Dir.AsVector()
}

// Domain returns the full real line, as a Line is unbounded.
func (l Line) Domain() (lo, hi float64) {
	return stdmath.Inf(-1), stdmath.Inf(1)
}

// LineSegment is a bounded 3D line between StartPoint and EndPoint (contract:
// LineSegment), parameterized t∈[0,1] from start to end.
type LineSegment struct {
	StartPoint, EndPoint math.Point3
}

// NewLineSegment builds a segment between two points.
func NewLineSegment(start, end math.Point3) LineSegment {
	return LineSegment{StartPoint: start, EndPoint: end}
}

// PointAt linearly interpolates from StartPoint (t=0) to EndPoint (t=1).
func (s LineSegment) PointAt(t float64) math.Point3 {
	return s.StartPoint.TranslateBy(s.StartPoint.VectorTo(s.EndPoint).Scale(t))
}

// TangentAt returns the start→end vector (constant); its length equals the
// segment length because t spans [0,1].
func (s LineSegment) TangentAt(float64) math.Vector3 {
	return s.StartPoint.VectorTo(s.EndPoint)
}

// Domain returns [0, 1].
func (s LineSegment) Domain() (lo, hi float64) { return 0, 1 }

// Length returns the distance between the endpoints.
func (s LineSegment) Length() float64 {
	return s.StartPoint.DistanceTo(s.EndPoint)
}
