// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Exact convex clipping of one triangle against another in their shared plane —
// the geometry the coplanar co-refinement case needs. When two operand faces are
// coplanar and overlap, the overlap is a convex polygon whose boundary must be
// imprinted on both faces; trianglePolygonOverlap computes it exactly by
// Sutherland-Hodgman clipping with exact orient2 and exact edge crossings.

// trianglePolygonOverlap returns the convex overlap of triangles subject and clip
// (assumed coplanar in the projection of axis) as an ordered vertex list, or fewer
// than three points when they meet only in a segment/point/not at all.
func trianglePolygonOverlap(subject, clip [3]Point, axis int) []Point {
	poly := []Point{subject[0], subject[1], subject[2]}
	cl := clip
	if orient2(cl[0], cl[1], cl[2], axis) < 0 {
		cl[0], cl[2] = cl[2], cl[0] // clip must be CCW so "inside" is its left side
	}
	for i := 0; i < 3 && len(poly) > 0; i++ {
		poly = clipByLine(poly, cl[i], cl[(i+1)%3], axis)
	}
	return dedupConsecutive(poly)
}

// dedupConsecutive drops vertices equal to their cyclic predecessor. Clipping a
// triangle whose edges graze the clip boundary can emit duplicate vertices (a
// degenerate edge/point overlap); removing them collapses that to its 2-or-1-point
// form so the caller sees a clean shared edge rather than a zero-area sliver.
func dedupConsecutive(poly []Point) []Point {
	if len(poly) < 2 {
		return poly
	}
	out := []Point{poly[0]}
	for i := 1; i < len(poly); i++ {
		if !poly[i].Equal(out[len(out)-1]) {
			out = append(out, poly[i])
		}
	}
	if len(out) > 1 && out[len(out)-1].Equal(out[0]) {
		out = out[:len(out)-1]
	}
	return out
}

// clipByLine keeps the part of polygon poly on or to the left of the directed line
// a→b (one Sutherland-Hodgman half-plane pass), inserting exact crossing points
// where the boundary enters or leaves.
func clipByLine(poly []Point, a, b Point, axis int) []Point {
	var out []Point
	n := len(poly)
	for i := 0; i < n; i++ {
		prev := poly[(i+n-1)%n]
		cur := poly[i]
		curIn := orient2(a, b, cur, axis) >= 0
		prevIn := orient2(a, b, prev, axis) >= 0
		if curIn != prevIn {
			out = append(out, SegSegCross(prev, cur, a, b, axis))
		}
		if curIn {
			out = append(out, cur)
		}
	}
	return out
}
