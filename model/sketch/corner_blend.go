// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// AddFillet rounds the corner where two lines meet with a tangent arc of the given
// radius: it trims each line back to its tangent point and inserts the arc between them.
// It errors when the lines do not share a corner or are (anti)parallel.
//
//	arc, err := sk.AddFillet(l1, l2, 0.5)
func (s *Sketch) AddFillet(l1, l2 *Line, radius math.Scalar) (*Arc, error) {
	corner, dir1, dir2, half, err := cornerFrame("fillet", l1, l2)
	if err != nil {
		return nil, err
	}
	t := float64(radius) / stdmath.Tan(half)
	t1 := s.newPoint(corner.TranslateBy(dir1.Scale(t)))
	t2 := s.newPoint(corner.TranslateBy(dir2.Scale(t)))
	bis := dir1.Add(dir2)
	center := corner.TranslateBy(bis.Scale(float64(radius) / stdmath.Sin(half) / bis.Length()))
	trimTo(l1, l2, corner, t1, t2)
	c := s.newPoint(center)
	ccw := center.VectorTo(t1.Position()).Cross(center.VectorTo(t2.Position())) > 0
	return s.arcs.Add(c, t1, t2, ccw), nil
}

// AddChamfer bevels the corner where two lines meet with a straight line, trimming each
// line back by its distance (d1 on l1, d2 on l2). It errors like AddFillet.
//
//	line, err := sk.AddChamfer(l1, l2, 0.5, 0.5)
func (s *Sketch) AddChamfer(l1, l2 *Line, d1, d2 math.Scalar) (*Line, error) {
	corner, dir1, dir2, _, err := cornerFrame("chamfer", l1, l2)
	if err != nil {
		return nil, err
	}
	t1 := s.newPoint(corner.TranslateBy(dir1.Scale(float64(d1))))
	t2 := s.newPoint(corner.TranslateBy(dir2.Scale(float64(d2))))
	trimTo(l1, l2, corner, t1, t2)
	return s.lines.Add(t1, t2), nil
}

// cornerFrame finds the shared corner of two lines and returns it plus the unit
// directions from the corner toward each line's far end and the half-angle between them.
func cornerFrame(op string, l1, l2 *Line) (math.Point2, math.Vector2, math.Vector2, float64, error) {
	cornerA, far1, cornerB, far2, ok := sharedCorner(l1, l2)
	if !ok {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, 0, fmt.Errorf("%s: the two lines do not share a corner", op)
	}
	dir1, ok1 := unitVec(cornerA.Position().VectorTo(far1.Position()))
	dir2, ok2 := unitVec(cornerB.Position().VectorTo(far2.Position()))
	if !ok1 || !ok2 {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, 0, fmt.Errorf("%s: a line has zero length", op)
	}
	angle := stdmath.Atan2(stdmath.Abs(float64(dir1.Cross(dir2))), float64(dir1.Dot(dir2)))
	if angle < 1e-9 || stdmath.Abs(angle-stdmath.Pi) < 1e-9 {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, 0, fmt.Errorf("%s: the lines are parallel (angle %.4g rad)", op, angle)
	}
	return cornerA.Position(), dir1, dir2, angle / 2, nil
}

// sharedCorner finds the corner the two lines meet at (by point identity or coincident
// position), returning each line's corner point and its far endpoint.
func sharedCorner(l1, l2 *Line) (corner1, far1, corner2, far2 *Point, ok bool) {
	for _, c := range []*Point{l1.A, l1.B} {
		for _, d := range []*Point{l2.A, l2.B} {
			if c == d || c.Position().IsEqualTo(d.Position(), 1e-9) {
				return c, lineOther(l1, c), d, lineOther(l2, d), true
			}
		}
	}
	return nil, nil, nil, nil, false
}

// lineOther returns the line's endpoint that is not end.
func lineOther(l *Line, end *Point) *Point {
	if l.A == end {
		return l.B
	}
	return l.A
}

// trimTo repoints each line's corner endpoint (the one at cornerPos) to its new (tangent)
// point, shortening the lines to the blend.
func trimTo(l1, l2 *Line, cornerPos math.Point2, t1, t2 *Point) {
	setLineEnd(l1, cornerPos, t1)
	setLineEnd(l2, cornerPos, t2)
}

// setLineEnd replaces whichever endpoint of l sits at cornerPos by p.
func setLineEnd(l *Line, cornerPos math.Point2, p *Point) {
	if l.A.Position().IsEqualTo(cornerPos, 1e-9) {
		l.A = p
		return
	}
	l.B = p
}

// unitVec normalizes v, reporting false for a zero vector.
func unitVec(v math.Vector2) (math.Vector2, bool) {
	if v.Length() == 0 {
		return math.Vector2{}, false
	}
	return v.Scale(1 / v.Length()), true
}
