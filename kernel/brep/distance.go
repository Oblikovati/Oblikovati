// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic minimum distance between B-rep sub-entities (M48/C3 #3458). The closest approach of two
// entities is computed against the EXACT trimmed geometry — never a face's triangle mesh, whose chord
// sagitta biases every convex-curved clearance inward. It is the OCCT BRepExtrema_DistShapeShape
// recursion: the closest approach between two entities is achieved either at an interior stationary
// point of the distance (both feet strictly inside their trims) or on a boundary sub-entity — a
// face's edges, an edge's ends — so every case reduces, one dimension at a time, to the exact
// point→curve and point→surface projections geom already provides.

// DistSupport is one operand of a distance query: EXACTLY one of a point (a vertex), a trimmed curve
// (an edge, or a transient probe segment — the curve's Domain is its trim), or a trimmed face (its
// loops are the trim). Build it with PointSupport / CurveSupport / FaceSupport.
type DistSupport struct {
	point *math.Point3
	curve geom.Curve3
	face  *topo.Face
}

// PointSupport wraps a point (a vertex) as a distance operand.
func PointSupport(p math.Point3) DistSupport { return DistSupport{point: &p} }

// CurveSupport wraps a trimmed curve (an edge's geometry, or a transient probe segment) as a distance
// operand; the curve's Domain bounds the trim.
func CurveSupport(c geom.Curve3) DistSupport { return DistSupport{curve: c} }

// FaceSupport wraps a trimmed face as a distance operand.
func FaceSupport(f *topo.Face) DistSupport { return DistSupport{face: f} }

// EntityDistance returns the minimum distance, in database units, between two B-rep sub-entities,
// computed against the exact trimmed geometry. It is 0 when the entities touch or cross.
//
// Example — clearance between two faces:
//
//	gap := brep.EntityDistance(brep.FaceSupport(fa), brep.FaceSupport(fb))
func EntityDistance(a, b DistSupport) float64 {
	a, b = orderByDimension(a, b) // a is the lower-dimensional operand
	switch {
	case a.point != nil:
		return pointSupportDistance(*a.point, b)
	case a.curve != nil:
		return curveSupportDistance(a.curve, b)
	default:
		return distFaceFace(a.face, b.face)
	}
}

// dimension is 0 for a point, 1 for a curve, 2 for a face — the recursion always drives the
// lower-dimensional operand's parameter and projects onto the higher one.
func (s DistSupport) dimension() int {
	switch {
	case s.point != nil:
		return 0
	case s.curve != nil:
		return 1
	default:
		return 2
	}
}

// orderByDimension returns the pair with the lower-dimensional operand first, so the caller dispatches
// on a's kind alone.
func orderByDimension(a, b DistSupport) (DistSupport, DistSupport) {
	if a.dimension() <= b.dimension() {
		return a, b
	}
	return b, a
}

// pointSupportDistance measures a point against b (a point, curve or face).
func pointSupportDistance(p math.Point3, b DistSupport) float64 {
	switch {
	case b.point != nil:
		return float64(p.DistanceTo(*b.point))
	case b.curve != nil:
		return distPointCurve(p, b.curve)
	default:
		return distPointFace(p, b.face)
	}
}

// curveSupportDistance measures a curve against b (a curve or face — b is never lower-dimensional here
// because orderByDimension put the point operand first).
func curveSupportDistance(c geom.Curve3, b DistSupport) float64 {
	if b.curve != nil {
		return distCurveCurve(c, b.curve)
	}
	return distCurveFace(c, b.face)
}

// distPointCurve is the exact distance from p to the trimmed curve c: the closest parameter on the
// unbounded curve, clamped into the curve's domain (the clamp lands on an endpoint when the foot of
// the perpendicular falls off the trim), evaluated back to a point.
func distPointCurve(p math.Point3, c geom.Curve3) float64 {
	t, _ := geom.CurveParamAtPoint3(c, p)
	lo, hi := c.Domain()
	t = clampParam(t, lo, hi)
	return float64(c.PointAt(t).DistanceTo(p))
}

// distPointFace is the exact distance from p to the trimmed face f: the foot of the perpendicular on
// the face's surface when it lands inside the trim, otherwise the closest of the face's boundary edges
// (the true minimum then lies on the boundary). A boundaryless face (a whole sphere or torus) has no
// edges and its whole surface is the trim, so the foot always wins.
func distPointFace(p math.Point3, f *topo.Face) float64 {
	_, _, foot := geom.ClosestPointOnSurface(f.Geometry(), p)
	if PointInFaceTrim(f, foot) {
		return float64(foot.DistanceTo(p))
	}
	best := stdmath.Inf(1)
	for _, e := range f.Edges() {
		best = stdmath.Min(best, distPointCurve(p, e.Geometry()))
	}
	return best
}

// distCurveCurve is the exact distance between two trimmed curves. For the optimal pair the foot on c2
// is the closest point on c2 to c1's point, so the distance is the minimum over c1's bounded domain of
// the exact point→c2 distance — one bounded scalar minimisation, its endpoints covering c1's ends and
// the inner projection covering c2's.
func distCurveCurve(c1, c2 geom.Curve3) float64 {
	scan, other := boundedCurveFirst(c1, c2)
	return minimizeOverCurve(scan, func(q math.Point3) float64 { return distPointCurve(q, other) })
}

// distCurveFace is the exact distance between a trimmed curve and a trimmed face: the minimum over the
// curve's bounded domain of the exact point→face distance. The inner point→face already recurses onto
// the face's boundary edges, so this one scan covers the curve's ends, the face's interior and the
// face's boundary.
func distCurveFace(c geom.Curve3, f *topo.Face) float64 {
	return minimizeOverCurve(c, func(q math.Point3) float64 { return distPointFace(q, f) })
}

// distFaceFace is the exact distance between two trimmed faces: the interior stationary approach (both
// feet inside their trims, found by alternating projection from a seed grid) versus each face's
// boundary edges against the other face. The boundary terms carry every clearance whose minimum lies
// on an edge; the interior term carries the parallel-plate / convex-facing case where it does not.
func distFaceFace(f1, f2 *topo.Face) float64 {
	best := interiorFacePairDistance(f1, f2)
	best = stdmath.Min(best, edgesToFaceDistance(f1, f2))
	return stdmath.Min(best, edgesToFaceDistance(f2, f1))
}

// edgesToFaceDistance is the minimum distance from any boundary edge of fa to face fb.
func edgesToFaceDistance(fa, fb *topo.Face) float64 {
	best := stdmath.Inf(1)
	for _, e := range fa.Edges() {
		best = stdmath.Min(best, distCurveFace(e.Geometry(), fb))
	}
	return best
}

// clampParam clamps t into [lo, hi], leaving an unbounded (±Inf) bound alone.
func clampParam(t, lo, hi float64) float64 {
	if !stdmath.IsInf(lo, 0) && t < lo {
		return lo
	}
	if !stdmath.IsInf(hi, 0) && t > hi {
		return hi
	}
	return t
}

// boundedCurveFirst returns the pair with a curve of bounded domain first (edges and probe segments
// are always bounded); it lets the scalar minimisation run over a finite interval. When both are
// bounded the order is unchanged.
func boundedCurveFirst(a, b geom.Curve3) (scan, other geom.Curve3) {
	if lo, hi := a.Domain(); !stdmath.IsInf(lo, 0) && !stdmath.IsInf(hi, 0) {
		return a, b
	}
	return b, a
}
