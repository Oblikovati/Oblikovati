// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// closedLoop mints a shared point for each position and connects them with lines in a
// closed loop (last → first), returning the created lines. Sharing endpoints keeps the
// loop a single closed profile.
func (s *Sketch) closedLoop(pts []math.Point2) []Entity {
	lines, _ := s.closedLoopPoints(pts)
	return lines
}

// closedLoopPoints is closedLoop that also returns the shared corner points, so a
// caller (the regular polygon) can constrain them.
func (s *Sketch) closedLoopPoints(pts []math.Point2) ([]Entity, []*Point) {
	pp := make([]*Point, len(pts))
	for i, p := range pts {
		pp[i] = s.newPoint(p)
	}
	out := make([]Entity, len(pp))
	for i := range pp {
		out[i] = s.lines.Add(pp[i], pp[(i+1)%len(pp)])
	}
	return out, pp
}

// AddPolyline connects the given points with straight lines that share endpoints (so the
// chain is one connected profile). When closed it also joins the last point back to the
// first, yielding a single closed profile (the way to author an arbitrary outline — an
// L-bracket, a custom extrusion section — without hand-placing each line). An open polyline
// needs ≥2 points; a closed one needs ≥3.
//
// Example: s.AddPolyline([]math.Point2{{0,0},{3,0},{3,1.5},{1.5,1.5},{1.5,3},{0,3}}, true)
// builds a closed L-shaped profile.
func (s *Sketch) AddPolyline(pts []math.Point2, closed bool) ([]Entity, error) {
	if closed {
		if len(pts) < 3 {
			return nil, fmt.Errorf("polyline: a closed polyline needs at least 3 points, got %d", len(pts))
		}
		return s.closedLoop(pts), nil
	}
	if len(pts) < 2 {
		return nil, fmt.Errorf("polyline: an open polyline needs at least 2 points, got %d", len(pts))
	}
	pp := make([]*Point, len(pts))
	for i, p := range pts {
		pp[i] = s.newPoint(p)
	}
	out := make([]Entity, 0, len(pp)-1)
	for i := 0; i+1 < len(pp); i++ {
		out = append(out, s.lines.Add(pp[i], pp[i+1]))
	}
	return out, nil
}

// AddTextOutlines adds rendered text glyph contours (from package text) as closed polylines,
// each translated by anchor — every contour is one glyph loop, and a letter's counter (the
// hole in A/O/B) is a nested contour the profile detector turns into a hole. Degenerate
// contours (<3 points) are skipped. Returns all created line entities. This is how real
// true-type text becomes embossable/extrudable sketch geometry.
func (s *Sketch) AddTextOutlines(anchor math.Point2, contours [][]math.Point2) []Entity {
	var out []Entity
	for _, c := range contours {
		if len(c) < 3 {
			continue
		}
		moved := make([]math.Point2, len(c))
		for i, p := range c {
			moved[i] = math.P2(anchor.X+p.X, anchor.Y+p.Y)
		}
		if lines, err := s.AddPolyline(moved, true); err == nil {
			out = append(out, lines...)
		}
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
func (s *Sketch) AddPolygon(center, through math.Point2, sides int, inscribed bool) ([]Entity, *Point, error) {
	if sides < 3 {
		return nil, nil, fmt.Errorf("polygon: needs at least 3 sides, got %d", sides)
	}
	v := center.VectorTo(through)
	r := v.Length()
	if r == 0 {
		return nil, nil, fmt.Errorf("polygon: through point coincides with center")
	}
	pts := polygonVertices(center, stdmath.Atan2(float64(v.Y), float64(v.X)), float64(r), sides, inscribed)
	lines, verts := s.closedLoopPoints(pts)

	// Make it a rigid REGULAR polygon (Inventor auto-adds these), so it is fully
	// determined by centre + circumradius + rotation. A construction circumscribed
	// circle pins every vertex to a common radius, and equal consecutive edges make
	// the chords equal — equal chords on one circle are equally spaced, i.e. regular.
	centerPt := s.newPoint(center)
	circ := s.circles.Add(centerPt, centerPt.Position().DistanceTo(verts[0].Position()))
	circ.SetConstruction(true)
	g := s.GeometricConstraints()
	for _, vtx := range verts {
		g.AddPointOnCircle(vtx, circ)
	}
	for i := 0; i+1 < len(lines); i++ {
		g.AddEqualLength(lines[i].(*Line), lines[i+1].(*Line))
	}
	return lines, centerPt, nil
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

// AddArcSlot builds an arc-shaped slot: a centerline arc (center, from start to end,
// CCW per ccw) thickened by width into inner/outer concentric arcs capped by a semicircle
// at each end. Returns the four boundary arcs. It errors on a zero radius or width ≥ 2R.
func (s *Sketch) AddArcSlot(center, start, end math.Point2, width math.Scalar, ccw bool) ([]Entity, error) {
	dirS, okS := unitVec(center.VectorTo(start))
	dirE, okE := unitVec(center.VectorTo(end))
	if !okS || !okE {
		return nil, fmt.Errorf("arc slot: start/end coincides with center")
	}
	r := center.DistanceTo(start)
	half := width / 2
	if r-half <= 0 {
		return nil, fmt.Errorf("arc slot: width %.4g ≥ 2·radius %.4g", float64(width), float64(r))
	}
	so := s.newPoint(center.TranslateBy(dirS.Scale(float64(r + half))))
	si := s.newPoint(center.TranslateBy(dirS.Scale(float64(r - half))))
	eo := s.newPoint(center.TranslateBy(dirE.Scale(float64(r + half))))
	ei := s.newPoint(center.TranslateBy(dirE.Scale(float64(r - half))))
	cOut, cIn := s.newPoint(center), s.newPoint(center)
	capE, capS := s.newPoint(end), s.newPoint(start)
	// Loop: so →(outer, ccw)→ eo →(end cap, ccw)→ ei →(inner, !ccw)→ si →(start cap, ccw)→ so.
	return []Entity{
		s.arcs.Add(cOut, so, eo, ccw),
		s.arcs.Add(capE, eo, ei, ccw),
		s.arcs.Add(cIn, ei, si, !ccw),
		s.arcs.Add(capS, si, so, ccw),
	}, nil
}

// AddStraightSlotByOverall builds a straight slot whose end0/end1 are the OUTER cap tips
// (the overall extents, Inventor AddStraightSlotByOverall): it shrinks each end inward by
// width/2 to the cap centers and delegates to AddStraightSlot (#149).
func (s *Sketch) AddStraightSlotByOverall(end0, end1 math.Point2, width math.Scalar) ([]Entity, error) {
	d := end0.VectorTo(end1)
	if d.Length() <= float64(width) {
		return nil, fmt.Errorf("slot by overall: overall length %.4g must exceed the width %.4g", d.Length(), float64(width))
	}
	du := d.Scale(1 / d.Length()).Scale(float64(width / 2))
	return s.AddStraightSlot(end0.TranslateBy(du), end1.TranslateBy(du.Negate()), width)
}

// AddStraightSlotBySlotCenter builds a straight slot from its center point and one end cap
// center; the other cap is the reflection of endCenter through slotCenter (Inventor
// AddStraightSlotBySlotCenter, #149).
func (s *Sketch) AddStraightSlotBySlotCenter(slotCenter, endCenter math.Point2, width math.Scalar) ([]Entity, error) {
	other := math.P2(2*slotCenter.X-endCenter.X, 2*slotCenter.Y-endCenter.Y)
	return s.AddStraightSlot(other, endCenter, width)
}

// AddArcSlotByCenterPoint builds an arc slot from the arc center, the start cap center, and a
// signed sweep angle in radians: the end cap is start rotated about center by sweep (Inventor
// AddArcSlotByCenterPointArc, #149).
func (s *Sketch) AddArcSlotByCenterPoint(center, start math.Point2, sweep, width math.Scalar) ([]Entity, error) {
	v := center.VectorTo(start)
	cos, sin := stdmath.Cos(float64(sweep)), stdmath.Sin(float64(sweep))
	end := center.TranslateBy(math.V2(v.X*math.Scalar(cos)-v.Y*math.Scalar(sin), v.X*math.Scalar(sin)+v.Y*math.Scalar(cos)))
	return s.AddArcSlot(center, start, end, width, sweep > 0)
}

// AddArcSlotByThreePoints builds an arc slot whose centerline passes through three points on
// it (start, a mid point, end); the arc center is their circumcenter, the sweep direction from
// the points' turn (Inventor AddArcSlotByThreePointArc, #149).
func (s *Sketch) AddArcSlotByThreePoints(p0, p1, p2 math.Point2, width math.Scalar) ([]Entity, error) {
	center, _, err := circumcircle(p0, p1, p2)
	if err != nil {
		return nil, fmt.Errorf("arc slot by three points: %w", err)
	}
	turn := (p1.X-p0.X)*(p2.Y-p0.Y) - (p1.Y-p0.Y)*(p2.X-p0.X) // >0 ⇒ p0→p1→p2 turns left (ccw)
	return s.AddArcSlot(center, p0, p2, width, turn > 0)
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
