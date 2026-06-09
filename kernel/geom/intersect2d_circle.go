// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Analytic 2D line/segment↔circle and circle↔circle intersection. Unlike
// [Segment2dIntersection] (which the region detector uses on faceted curves), these are
// exact on the analytic circle — what sketch trim/extend/split needs to cut a curve at
// its true crossing rather than at a facet vertex. Arc crossings are these circle
// results filtered by [Arc2d.ContainsPoint].

// LineCircle2dIntersection returns the points where the infinite line l crosses circle c:
// none (line misses), one (tangent, within tol), or two. tol<=0 uses the default
// geometric tolerance.
func LineCircle2dIntersection(l Line2d, c Circle2d, tol float64) []math.Point2 {
	tol = geomTol(tol)
	dir := l.Dir.AsVector() // unit
	foot := l.Origin.TranslateBy(dir.Scale(l.Origin.VectorTo(c.Center).Dot(dir)))
	dist := foot.DistanceTo(c.Center)
	if dist > c.Radius+tol {
		return nil
	}
	half := stdmath.Sqrt(stdmath.Max(0, c.Radius*c.Radius-dist*dist))
	if half <= tol {
		return []math.Point2{foot}
	}
	return []math.Point2{foot.TranslateBy(dir.Scale(half)), foot.TranslateBy(dir.Scale(-half))}
}

// SegmentCircle2dIntersection is [LineCircle2dIntersection] restricted to the segment's
// extent: only crossings whose foot lies on seg (within tol) are returned. A degenerate
// (zero-length) segment yields none.
func SegmentCircle2dIntersection(seg LineSegment2d, c Circle2d, tol float64) []math.Point2 {
	tol = geomTol(tol)
	line, err := NewLine2d(seg.StartPoint, seg.StartPoint.VectorTo(seg.EndPoint))
	if err != nil {
		return nil
	}
	var out []math.Point2
	for _, p := range LineCircle2dIntersection(line, c, tol) {
		if onSegment2d(seg, p, tol) {
			out = append(out, p)
		}
	}
	return out
}

// Circle2dCircle2dIntersection returns the points where two circles cross: none
// (disjoint, one inside the other, or concentric), one (tangent, within tol), or two.
func Circle2dCircle2dIntersection(c1, c2 Circle2d, tol float64) []math.Point2 {
	tol = geomTol(tol)
	d := c1.Center.DistanceTo(c2.Center)
	if d <= tol { // concentric: no isolated crossing (none, or the whole circle if equal)
		return nil
	}
	if d > c1.Radius+c2.Radius+tol || d < stdmath.Abs(c1.Radius-c2.Radius)-tol {
		return nil
	}
	// Distance from c1.Center to the radical line, along the center direction.
	a := (d*d + c1.Radius*c1.Radius - c2.Radius*c2.Radius) / (2 * d)
	dir := c1.Center.VectorTo(c2.Center).Scale(1 / d) // unit
	mid := c1.Center.TranslateBy(dir.Scale(a))
	h2 := c1.Radius*c1.Radius - a*a
	if h2 <= tol*tol {
		return []math.Point2{mid}
	}
	h := stdmath.Sqrt(h2)
	perp := math.V2(-dir.Y, dir.X)
	return []math.Point2{mid.TranslateBy(perp.Scale(h)), mid.TranslateBy(perp.Scale(-h))}
}

// onSegment2d reports whether p (assumed on the segment's support line) lies between the
// endpoints, within a tol-derived parameter slack so an endpoint touch still counts.
func onSegment2d(seg LineSegment2d, p math.Point2, tol float64) bool {
	v := seg.StartPoint.VectorTo(seg.EndPoint)
	lenSq := v.LengthSquared()
	if lenSq == 0 {
		return false
	}
	t := seg.StartPoint.VectorTo(p).Dot(v) / lenSq
	eps := tol / stdmath.Sqrt(lenSq) // geometric tol → parameter slack
	return t >= -eps && t <= 1+eps
}

// geomTol resolves a non-positive tolerance to the package's default geometric tolerance.
func geomTol(tol float64) float64 {
	if tol <= 0 {
		return math.DefaultTolerance
	}
	return tol
}
