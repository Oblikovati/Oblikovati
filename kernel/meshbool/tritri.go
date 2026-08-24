// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Exact triangle-triangle intersection — the segment where two faces of the two
// operands meet. Because it is computed once from exact crossings, the segment is
// identical for every face incident to a shared edge, which is the conforming
// property whose absence tears #2084.

// TriTriKind classifies how two triangles meet.
type TriTriKind int

const (
	Disjoint TriTriKind = iota // no shared point
	Crossing                   // transversal: a proper segment P!=Q
	Touching                   // meet at a single point P==Q
	Coplanar                   // lie in the same plane (2D overlap; resolved separately)
)

// TriTriResult is the outcome of IntersectTriangles. P and Q carry the segment
// endpoints when Kind==Crossing, the single point (in P, with Q==P) when
// Touching, and are unset for Disjoint/Coplanar.
type TriTriResult struct {
	Kind TriTriKind
	P, Q Point
}

// IntersectTriangles returns the exact intersection of triangles t1 and t2.
// PRECONDITION: both triangles are non-degenerate. The coplanar case is reported
// as Coplanar without computing the 2D overlap polygon — that is a distinct
// sub-problem the co-refinement resolves separately.
func IntersectTriangles(t1, t2 [3]Point) TriTriResult {
	if onPlane(t2, t1[0]) && onPlane(t2, t1[1]) && onPlane(t2, t1[2]) {
		return TriTriResult{Kind: Coplanar}
	}
	c1 := triPlaneCrossPoints(t1, t2) // t1 ∩ plane(t2)
	c2 := triPlaneCrossPoints(t2, t1) // t2 ∩ plane(t1)
	if len(c1) == 0 || len(c2) == 0 {
		return TriTriResult{Kind: Disjoint}
	}
	return overlapOnLine(t1, t2, c1, c2)
}

// onPlane reports whether p lies exactly on the plane of triangle tri.
func onPlane(tri [3]Point, p Point) bool {
	return Orient3D(tri[0], tri[1], tri[2], p) == 0
}

// triPlaneCrossPoints returns the (0, 1, or 2) points where tri's boundary meets
// the plane of triangle plane: the segment tri ∩ plane(plane) for a transversal
// tri. Vertices exactly on the plane and edges that strictly straddle it both
// contribute; duplicates are removed exactly.
func triPlaneCrossPoints(tri, plane [3]Point) []Point {
	s := [3]int{
		Orient3D(plane[0], plane[1], plane[2], tri[0]),
		Orient3D(plane[0], plane[1], plane[2], tri[1]),
		Orient3D(plane[0], plane[1], plane[2], tri[2]),
	}
	var pts []Point
	for i := 0; i < 3; i++ {
		if s[i] == 0 {
			pts = appendUnique(pts, tri[i])
		}
	}
	for _, e := range [3][2]int{{0, 1}, {1, 2}, {2, 0}} {
		if s[e[0]]*s[e[1]] < 0 { // endpoints strictly straddle the plane
			pts = appendUnique(pts, EdgePlaneCross(tri[e[0]], tri[e[1]], plane[0], plane[1], plane[2]))
		}
	}
	return pts
}

// overlapOnLine intersects the two collinear sub-segments c1 and c2 (both lying on
// the plane-intersection line) and returns their overlap. Points are ordered by an
// exact linear parameter along the line direction n1×n2.
func overlapOnLine(t1, t2 [3]Point, c1, c2 []Point) TriTriResult {
	origin := t1[0]
	dir := rcross(triNormal(t1), triNormal(t2))
	aLo, aHi := paramSpan(c1, origin, dir)
	bLo, bHi := paramSpan(c2, origin, dir)
	lo := maxEnd(aLo, bLo)
	hi := minEnd(aHi, bHi)
	switch lo.t.Cmp(hi.t) {
	case 1:
		return TriTriResult{Kind: Disjoint}
	case 0:
		return TriTriResult{Kind: Touching, P: lo.pt, Q: lo.pt}
	default:
		return TriTriResult{Kind: Crossing, P: lo.pt, Q: hi.pt}
	}
}

// endpoint pairs a point with its exact parameter along the intersection line.
type endpoint struct {
	pt Point
	t  *big.Rat
}

// paramSpan returns the lowest- and highest-parameter endpoints of pts along dir.
func paramSpan(pts []Point, origin Point, dir [3]*big.Rat) (lo, hi endpoint) {
	lo = endpoint{pts[0], rdot(dir, pts[0].sub(origin))}
	hi = lo
	for _, p := range pts[1:] {
		t := rdot(dir, p.sub(origin))
		if t.Cmp(lo.t) < 0 {
			lo = endpoint{p, t}
		}
		if t.Cmp(hi.t) > 0 {
			hi = endpoint{p, t}
		}
	}
	return lo, hi
}

// maxEnd returns the endpoint with the larger parameter (the start of the overlap).
func maxEnd(a, b endpoint) endpoint {
	if a.t.Cmp(b.t) >= 0 {
		return a
	}
	return b
}

// minEnd returns the endpoint with the smaller parameter (the end of the overlap).
func minEnd(a, b endpoint) endpoint {
	if a.t.Cmp(b.t) <= 0 {
		return a
	}
	return b
}

// appendUnique appends p unless an exactly-equal point is already present.
func appendUnique(pts []Point, p Point) []Point {
	for _, q := range pts {
		if q.Equal(p) {
			return pts
		}
	}
	return append(pts, p)
}
