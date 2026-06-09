// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// AddOffsetDim dimensions the perpendicular distance from point p to line l.
func (dc *DimensionConstraints) AddOffsetDim(p *Point, l *Line, expression string) (*DimensionConstraint, error) {
	measure := func() float64 { return perpDistance(p, l) }
	vars := []*math.Scalar{&p.X, &p.Y, &l.A.X, &l.A.Y, &l.B.X, &l.B.Y}
	return dc.create(OffsetDim, expression, []Entity{p, l}, measure, vars)
}

// AddThreePointAngle dimensions the angle a–vertex–b (the included angle at vertex).
func (dc *DimensionConstraints) AddThreePointAngle(vertex, a, b *Point, expression string) (*DimensionConstraint, error) {
	measure := func() float64 {
		ax, ay := float64(a.X-vertex.X), float64(a.Y-vertex.Y)
		bx, by := float64(b.X-vertex.X), float64(b.Y-vertex.Y)
		return stdmath.Atan2(stdmath.Abs(ax*by-ay*bx), ax*bx+ay*by)
	}
	vars := []*math.Scalar{&vertex.X, &vertex.Y, &a.X, &a.Y, &b.X, &b.Y}
	return dc.create(ThreePointAngleDim, expression, []Entity{vertex, a, b}, measure, vars)
}

// AddEllipseRadius dimensions an ellipse's major radius.
func (dc *DimensionConstraints) AddEllipseRadius(e *Ellipse, expression string) (*DimensionConstraint, error) {
	measure := func() float64 { return float64(e.MajorRadius) }
	return dc.create(EllipseRadiusDim, expression, []Entity{e}, measure, []*math.Scalar{&e.MajorRadius})
}

// perpDistance is the absolute perpendicular distance from point p to the line through l.
func perpDistance(p *Point, l *Line) float64 {
	a, b := l.A.Position(), l.B.Position()
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	length := stdmath.Hypot(dx, dy)
	if length == 0 {
		return float64(p.Position().DistanceTo(a))
	}
	cross := float64(p.X-a.X)*dy - float64(p.Y-a.Y)*dx
	return stdmath.Abs(cross) / length
}
