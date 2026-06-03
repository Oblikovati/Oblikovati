// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"github.com/Oblikovati/oblikovati/math"
)

// closedLoop mints a shared point for each position and connects them with lines in a
// closed loop (last → first), returning the created lines. Sharing endpoints keeps the
// loop a single closed profile.
func (s *Sketch) closedLoop(pts []math.Point2) []Entity {
	pp := make([]*Point, len(pts))
	for i, p := range pts {
		pp[i] = s.newPoint(p)
	}
	out := make([]Entity, len(pp))
	for i := range pp {
		out[i] = s.lines.Add(pp[i], pp[(i+1)%len(pp)])
	}
	return out
}

// AddRectangleByCorners builds an axis-aligned rectangle from two opposite corners,
// returning its four lines.
func (s *Sketch) AddRectangleByCorners(p0, opposite math.Point2) []Entity {
	c1 := math.P2(opposite.X, p0.Y)
	c3 := math.P2(p0.X, opposite.Y)
	return s.closedLoop([]math.Point2{p0, c1, opposite, c3})
}

// AddRectangleByCenter builds an axis-aligned rectangle from its center and one corner.
func (s *Sketch) AddRectangleByCenter(center, corner math.Point2) []Entity {
	hx := stdmath.Abs(float64(corner.X - center.X))
	hy := stdmath.Abs(float64(corner.Y - center.Y))
	p0 := math.P2(center.X-math.Scalar(hx), center.Y-math.Scalar(hy))
	opp := math.P2(center.X+math.Scalar(hx), center.Y+math.Scalar(hy))
	return s.AddRectangleByCorners(p0, opp)
}

// AddRectangleByThreePoints builds a rectangle whose first edge is p0→p1 and whose depth
// is set by p2 (projected onto the edge's perpendicular). It errors if p0==p1.
func (s *Sketch) AddRectangleByThreePoints(p0, p1, p2 math.Point2) ([]Entity, error) {
	w := p0.VectorTo(p1)
	perp := math.V2(-w.Y, w.X)
	if perp.Length() == 0 {
		return nil, fmt.Errorf("rectangle: first edge has zero length (%v == %v)", p0, p1)
	}
	n := perp.Scale(1 / perp.Length())
	depth := n.Scale(p1.VectorTo(p2).Dot(n))
	c2 := p1.TranslateBy(depth)
	c3 := p0.TranslateBy(depth)
	return s.closedLoop([]math.Point2{p0, p1, c2, c3}), nil
}

// AddPolygon builds a regular n-gon centered at center. The through point lies on a
// vertex when inscribed, or on an edge midpoint (the apothem) when circumscribed. It
// errors for fewer than 3 sides or a zero radius.
func (s *Sketch) AddPolygon(center, through math.Point2, sides int, inscribed bool) ([]Entity, error) {
	if sides < 3 {
		return nil, fmt.Errorf("polygon: needs at least 3 sides, got %d", sides)
	}
	v := center.VectorTo(through)
	r := v.Length()
	if r == 0 {
		return nil, fmt.Errorf("polygon: through point coincides with center")
	}
	pts := polygonVertices(center, stdmath.Atan2(float64(v.Y), float64(v.X)), float64(r), sides, inscribed)
	return s.closedLoop(pts), nil
}

// AddStraightSlot builds a center-to-center straight slot of the given width: two
// parallel side lines capped by a semicircular arc at each center. Returns the two lines
// and two arcs. It errors when the two centers coincide.
func (s *Sketch) AddStraightSlot(c0, c1 math.Point2, width math.Scalar) ([]Entity, error) {
	d := c0.VectorTo(c1)
	if d.Length() == 0 {
		return nil, fmt.Errorf("slot: the two centers coincide (%v)", c0)
	}
	du := d.Scale(1 / d.Length())
	half := math.V2(-du.Y, du.X).Scale(width / 2)
	a0 := s.newPoint(c0.TranslateBy(half))
	a1 := s.newPoint(c1.TranslateBy(half))
	b1 := s.newPoint(c1.TranslateBy(half.Negate()))
	b0 := s.newPoint(c0.TranslateBy(half.Negate()))
	cen1, cen0 := s.newPoint(c1), s.newPoint(c0)
	// Both end caps wind clockwise (ccw=false) so they bulge outward, away from the slot.
	return []Entity{
		s.lines.Add(a0, a1),
		s.arcs.Add(cen1, a1, b1, false),
		s.lines.Add(b1, b0),
		s.arcs.Add(cen0, b0, a0, false),
	}, nil
}

// polygonVertices returns the n vertex positions. For circumscribed polygons the vertex
// radius is the apothem / cos(π/n) and the ring is rotated half a step so the through
// point lands on an edge midpoint.
func polygonVertices(center math.Point2, baseAngle, r float64, sides int, inscribed bool) []math.Point2 {
	step := 2 * stdmath.Pi / float64(sides)
	vertexR, offset := r, 0.0
	if !inscribed {
		vertexR = r / stdmath.Cos(step/2)
		offset = step / 2
	}
	pts := make([]math.Point2, sides)
	for i := range pts {
		a := baseAngle - offset + step*float64(i)
		pts[i] = math.P2(center.X+math.Scalar(vertexR*stdmath.Cos(a)), center.Y+math.Scalar(vertexR*stdmath.Sin(a)))
	}
	return pts
}
