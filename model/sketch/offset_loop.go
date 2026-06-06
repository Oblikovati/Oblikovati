// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati/math"
)

// OffsetClosedLoop adds the rounded offset of a closed boundary polygon as a new closed loop
// of line segments (the OpenSCAD offset(r) of a 2D region — d>0 grows, d<0 shrinks, convex
// corners rounded with radius |d|), returning the created line entities. The input is the
// sampled boundary of a closed profile (e.g. profile.OuterLoop().Polygon()).
func (s *Sketch) OffsetClosedLoop(poly []math.Point2, d float64, arcSegs int) []Entity {
	off := offsetClosedPolygon(poly, d, arcSegs)
	n := len(off)
	if n < 3 {
		return nil
	}
	pts := make([]*Point, n)
	for i, p := range off {
		pts[i] = s.points.Add(p)
	}
	ents := make([]Entity, 0, n)
	for i := 0; i < n; i++ {
		ents = append(ents, s.lines.Add(pts[i], pts[(i+1)%n]))
	}
	return ents
}

// offsetPolyline offsets an OPEN polyline by signed distance d along its left normal (d>0 ⇒
// to the left of travel), rounding the corners that the offset opens with arcs and mitring the
// rest — the corner-aware replacement for the old per-vertex normal shift. Smooth curves (a
// sampled spline) are unaffected; a polyline with real corners now offsets correctly.
func offsetPolyline(pts []math.Point2, d float64) []math.Point2 {
	pts = dropDuplicateVertices(pts)
	n := len(pts)
	if n < 2 || d == 0 {
		return pts
	}
	segs := make([]offsetEdge, n-1)
	for i := 0; i < n-1; i++ {
		p, q := pts[i], pts[i+1]
		dx, dy := float64(q.X-p.X), float64(q.Y-p.Y)
		l := stdmath.Hypot(dx, dy)
		off := math.V2(math.Scalar(-dy/l*d), math.Scalar(dx/l*d)) // left normal
		segs[i] = offsetEdge{a: p.TranslateBy(off), b: q.TranslateBy(off)}
	}
	out := []math.Point2{segs[0].a}
	for i := 1; i < n-1; i++ {
		out = append(out, openCorner(pts[i], segs[i-1], segs[i], d)...)
	}
	return append(out, segs[n-2].b)
}

// openCorner joins two consecutive left-offset segments at the interior vertex v: an arc when
// the left offset opens the corner (sign(turn) != sign(d)), otherwise the mitred intersection.
func openCorner(v math.Point2, prev, cur offsetEdge, d float64) []math.Point2 {
	turn := cross2(prev.a.VectorTo(prev.b), cur.a.VectorTo(cur.b))
	if stdmath.Abs(turn) < 1e-12 {
		return []math.Point2{cur.a}
	}
	if (turn > 0) != (d > 0) { // left offset opens this corner → round it
		return arcBetween(v, prev.b, cur.a, 6)
	}
	if p, ok := lineIntersection(prev.a, prev.b, cur.a, cur.b); ok {
		return []math.Point2{p}
	}
	return []math.Point2{cur.a}
}

// offsetClosedPolygon returns the rounded offset of a closed polygon by signed distance d —
// the boundary of the Minkowski sum with a disk of radius |d| for d>0 (OpenSCAD's offset(r)).
// d>0 grows the region outward, d<0 shrinks it. A corner that opens a gap under the offset is
// filled with an arc of radius |d| (sampled into arcSegs spans); a corner that overlaps is
// mitred to the intersection of the two offset edges. Winding is normalised to CCW, so the
// result is independent of the input order.
func offsetClosedPolygon(poly []math.Point2, d float64, arcSegs int) []math.Point2 {
	poly = dropDuplicateVertices(poly)
	if len(poly) < 3 || d == 0 {
		return poly
	}
	if arcSegs < 1 {
		arcSegs = 8
	}
	poly = ensureCCW(poly)
	segs := offsetEdges(poly, d)
	n := len(poly)
	var out []math.Point2
	for i := 0; i < n; i++ {
		prev, cur := segs[(i+n-1)%n], segs[i]
		out = append(out, cornerPoints(poly[i], prev, cur, d, arcSegs)...)
	}
	return out
}

// offsetEdge is one polygon edge shifted outward by the offset.
type offsetEdge struct {
	a, b math.Point2 // shifted endpoints
}

// offsetEdges shifts every edge by d along its outward (right-of-direction, for CCW) normal.
func offsetEdges(poly []math.Point2, d float64) []offsetEdge {
	n := len(poly)
	segs := make([]offsetEdge, n)
	for i := 0; i < n; i++ {
		p, q := poly[i], poly[(i+1)%n]
		dx, dy := float64(q.X-p.X), float64(q.Y-p.Y)
		l := stdmath.Hypot(dx, dy)
		nx, ny := dy/l, -dx/l // outward normal of a CCW edge
		off := math.V2(math.Scalar(nx*d), math.Scalar(ny*d))
		segs[i] = offsetEdge{a: p.TranslateBy(off), b: q.TranslateBy(off)}
	}
	return segs
}

// cornerPoints connects the previous offset edge's end to the current one's start across the
// original vertex v: an arc when the corner opens (sign(turn) == sign(d)), otherwise the two
// edges' intersection (a near-straight corner is just the shared point).
func cornerPoints(v math.Point2, prev, cur offsetEdge, d float64, arcSegs int) []math.Point2 {
	turn := cross2(prev.a.VectorTo(prev.b), cur.a.VectorTo(cur.b))
	if stdmath.Abs(turn) < 1e-12 {
		return []math.Point2{cur.a}
	}
	if (turn > 0) == (d > 0) { // gap → round it with an arc of radius |d|
		return arcBetween(v, prev.b, cur.a, arcSegs)
	}
	if p, ok := lineIntersection(prev.a, prev.b, cur.a, cur.b); ok { // overlap → mitre
		return []math.Point2{p}
	}
	return []math.Point2{cur.a}
}

// arcBetween samples the minor arc centred at c from start to end (both at radius |d| from c).
func arcBetween(c, start, end math.Point2, arcSegs int) []math.Point2 {
	a0 := stdmath.Atan2(float64(start.Y-c.Y), float64(start.X-c.X))
	a1 := stdmath.Atan2(float64(end.Y-c.Y), float64(end.X-c.X))
	delta := stdmath.Mod(a1-a0, 2*stdmath.Pi)
	if delta > stdmath.Pi {
		delta -= 2 * stdmath.Pi
	} else if delta < -stdmath.Pi {
		delta += 2 * stdmath.Pi
	}
	r := c.DistanceTo(start)
	pts := make([]math.Point2, 0, arcSegs+1)
	for k := 0; k <= arcSegs; k++ {
		a := a0 + delta*float64(k)/float64(arcSegs)
		pts = append(pts, math.P2(c.X+math.Scalar(r*stdmath.Cos(a)), c.Y+math.Scalar(r*stdmath.Sin(a))))
	}
	return pts
}

// lineIntersection returns the intersection of lines a1a2 and b1b2 (ok=false if parallel).
func lineIntersection(a1, a2, b1, b2 math.Point2) (math.Point2, bool) {
	r := a1.VectorTo(a2)
	s := b1.VectorTo(b2)
	denom := cross2(r, s)
	if stdmath.Abs(denom) < 1e-15 {
		return math.Point2{}, false
	}
	t := cross2(a1.VectorTo(b1), s) / denom
	return a1.TranslateBy(r.Scale(t)), true
}

// cross2 is the 2D scalar cross product.
func cross2(a, b math.Vector2) float64 {
	return float64(a.X*b.Y - a.Y*b.X)
}

// ensureCCW returns poly wound counter-clockwise (reversing it if its signed area is negative).
func ensureCCW(poly []math.Point2) []math.Point2 {
	if polygonSignedArea(poly) >= 0 {
		return poly
	}
	out := make([]math.Point2, len(poly))
	for i, p := range poly {
		out[len(poly)-1-i] = p
	}
	return out
}

// polygonSignedArea is the shoelace signed area (positive ⇒ CCW).
func polygonSignedArea(poly []math.Point2) float64 {
	var s float64
	for i := range poly {
		p, q := poly[i], poly[(i+1)%len(poly)]
		s += float64(p.X*q.Y - q.X*p.Y)
	}
	return s / 2
}

// dropDuplicateVertices removes consecutive coincident points (including the wrap) so edge
// normals are well defined.
func dropDuplicateVertices(poly []math.Point2) []math.Point2 {
	out := poly[:0:0]
	for _, p := range poly {
		if len(out) > 0 && out[len(out)-1].DistanceTo(p) < 1e-12 {
			continue
		}
		out = append(out, p)
	}
	for len(out) > 1 && out[0].DistanceTo(out[len(out)-1]) < 1e-12 {
		out = out[:len(out)-1]
	}
	return out
}
