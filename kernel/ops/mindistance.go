// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Minimum-distance geometry primitives (M18-F01 PBI-164, Oblikovati/Oblikovati#428): closest
// approach between two simplices (segment or triangle). Topology entities reduce to these — a
// vertex to a zero-length segment, an edge to its polyline segments, a face to its mesh triangles —
// so the entity-level minimum distance is the smallest of all simplex-pair distances. The closest
// points of two disjoint convex simplices lie on their boundaries, so each pair routine is exact;
// intersecting triangles report zero. Builds on closestPointOnTriangle and rayTriangleDist.

// SegmentSegmentDistance returns the minimum distance between segments p1q1 and p2q2, clamped to
// both endpoints (Ericson, Real-Time Collision Detection §5.1.9). Handles zero-length segments
// (a point), so it doubles as point-point and point-segment distance.
func SegmentSegmentDistance(p1, q1, p2, q2 math.Point3) float64 {
	d1 := p1.VectorTo(q1)
	d2 := p2.VectorTo(q2)
	r := p2.VectorTo(p1)
	a := float64(d1.Dot(d1))
	e := float64(d2.Dot(d2))
	s, t := segmentParams(a, e, float64(d2.Dot(r)), float64(d1.Dot(r)), float64(d1.Dot(d2)))
	c1 := p1.TranslateBy(d1.Scale(math.Scalar(s)))
	c2 := p2.TranslateBy(d2.Scale(math.Scalar(t)))
	return float64(c1.DistanceTo(c2))
}

// segmentParams solves the clamped closest-point parameters s,t∈[0,1] for two segments from the
// precomputed dot products: a=|d1|², e=|d2|², f=d2·r, c=d1·r, b=d1·d2, with r = p1−p2.
func segmentParams(a, e, f, c, b float64) (s, t float64) {
	const eps = 1e-15
	if a <= eps && e <= eps {
		return 0, 0
	}
	if a <= eps { // segment 1 degenerates to a point
		return 0, clamp01(f / e)
	}
	if e <= eps { // segment 2 degenerates to a point
		return clamp01(-c / a), 0
	}
	if denom := a*e - b*b; denom > eps {
		s = clamp01((b*f - c*e) / denom)
	}
	return reclampSForT(s, (b*s+f)/e, a, b, c)
}

// reclampSForT re-derives s after clamping t back into [0,1] (the unclamped t left the segment).
func reclampSForT(s, t, a, b, c float64) (float64, float64) {
	if t < 0 {
		return clamp01(-c / a), 0
	}
	if t > 1 {
		return clamp01((b - c) / a), 1
	}
	return s, t
}

// SegmentTriangleDistance returns the minimum distance between segment s0s1 and triangle abc; zero
// when the segment pierces the triangle.
func SegmentTriangleDistance(s0, s1, a, b, c math.Point3) float64 {
	if segmentHitsTriangle(s0, s1, a, b, c) {
		return 0
	}
	best := pointTriangleDistance(s0, a, b, c)
	best = stdmath.Min(best, pointTriangleDistance(s1, a, b, c))
	best = stdmath.Min(best, SegmentSegmentDistance(s0, s1, a, b))
	best = stdmath.Min(best, SegmentSegmentDistance(s0, s1, b, c))
	return stdmath.Min(best, SegmentSegmentDistance(s0, s1, c, a))
}

// segmentHitsTriangle reports whether segment s0s1 crosses triangle abc within its extent.
func segmentHitsTriangle(s0, s1, a, b, c math.Point3) bool {
	t, ok := rayTriangleDist(s0, s0.VectorTo(s1), a, b, c)
	return ok && t <= 1
}

// TriangleTriangleDistance returns the minimum distance between two triangles (zero when they
// intersect): the closest approach of disjoint triangles lies on an edge of one against the other,
// which the six edge-vs-triangle tests (covering edge-edge and vertex-face) cover exhaustively.
func TriangleTriangleDistance(a1, b1, c1, a2, b2, c2 math.Point3) float64 {
	best := SegmentTriangleDistance(a1, b1, a2, b2, c2)
	best = stdmath.Min(best, SegmentTriangleDistance(b1, c1, a2, b2, c2))
	best = stdmath.Min(best, SegmentTriangleDistance(c1, a1, a2, b2, c2))
	best = stdmath.Min(best, SegmentTriangleDistance(a2, b2, a1, b1, c1))
	best = stdmath.Min(best, SegmentTriangleDistance(b2, c2, a1, b1, c1))
	return stdmath.Min(best, SegmentTriangleDistance(c2, a2, a1, b1, c1))
}
