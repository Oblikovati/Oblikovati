// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Point-in-curved-face membership (M2 Phase 1, Oblikovati/Oblikovati#1334) for a face on a CLOSED
// surface — a sphere or a torus — where no parameter-space ray has an exterior end, so the even-odd
// classification [pointInTrimUV] uses everywhere else has nothing to cast toward.
//
// It classifies by the NEAREST boundary point: the shortest path from the query point to the trim
// boundary cannot cross that boundary (a crossing would itself be a nearer boundary point), so the query
// point lies on whichever side the boundary's inward direction names at its own foot. The verdict is
// therefore a LOCAL, exactly-evaluated measurement at one point of the true trim curve — it never
// integrates the loop, so it holds for a region of ANY size, up to the whole surface minus a pinhole.
//
// This replaced a tangent-plane geodesic winding number (Oblikovati/Oblikovati#3453, #3429). That
// projected the loops ORTHOGRAPHICALLY onto the tangent plane at the query point, which is 2-to-1 on a
// closed surface: it folded the far hemisphere onto the near one, so a region whose rim lies more than a
// quarter turn away wound to ~0 and read as OUTSIDE at its own centre. A Ø10 ball joined with a Ø6 rod
// disowned the 9/10-of-the-sphere cap it kept, and the ball's own centre read outside the solid.
//
// CERTIFICATION. The argument needs the 3-D chordal distance used here to order boundary points the same
// way the SURFACE metric does. On a sphere that holds exactly (chord = 2R·sin(geodesic/2R), monotone), so
// the verdict is exact there. On a TORUS it does not: a point on the inner wall can be chordally nearer a
// rim straight across the hole than any rim it can walk to, and such a query is answered from the wrong
// foot. Oblikovati/Oblikovati#3453 tracks giving the torus its own certified classification (the same
// nearest-foot test lifted into the covering parameter plane, where the flat metric IS the surface's
// topology); until then a torus verdict from this classifier is UNCERTIFIED, and a caller that needs
// certainty must gate on the surface type.

// footScanStations is the coarse scan that brackets an edge's closest approach before refinement. The
// distance along a conic trim edge has at most four critical points, so this many stations cannot step
// over the global one. It is not a tessellation: no result is read off the scan, only the bracket.
const footScanStations = 24

// footRefineSteps drives the golden-section refinement of that bracket. 0.618²⁸ ≈ 1.4e-6 narrows the
// two-station bracket to ~1e-7 of the edge — far below the displacement at which the side test could
// change its verdict.
const footRefineSteps = 28

// footVertexBand accepts a refined foot as sitting ON the edge's end vertex rather than in its interior,
// in the edge's own normalized parameter. The refinement converges to within ~1e-7 of an endpoint
// minimum, so this band separates the two cases with three decades to spare.
const footVertexBand = 1e-4 // tol:parametric — normalized edge parameter, dimensionless

// trimFoot is one loop edge's closest approach to the query point: where on the edge it lands, how far
// away it is, and which way the trimmed region lies from there.
type trimFoot struct {
	loop, edge int
	u          float64 // normalized position along the edge: 0 at t0, 1 at t1
	point      math.Point3
	dist       float64
	inward     math.Vector3 // unit, tangent to the surface, pointing into the trimmed region
}

// trimEdge is one loop edge with its place in the face, so the classifier can reach the NEIGHBOURING
// edge when a foot lands on a vertex the two share.
type trimEdge struct {
	le         loopEdge
	loop, edge int
}

// pointInCurvedFace reports whether p (assumed on f's surface) lies within f's trimmed region. A
// boundary-less face — a whole sphere or torus — contains every surface point.
func pointInCurvedFace(f curvedFace, p math.Point3) bool {
	if len(f.loops) == 0 {
		return true
	}
	border := borderEdges(f)
	best, ok := closestTrimFoot(f, border, p)
	if !ok {
		return true // no bordering edge names a side, so the whole surface is the region
	}
	return float64(best.point.VectorTo(p).Dot(cornerInward(f, border, best))) > 0
}

// borderEdges lists the loop edges that actually BORDER the region — every edge except the seams. A face
// on a closed surface is routinely walked with a SLIT: a bridge run out to a pole (or across the
// parameter seam) and straight back along the same curve, so the loop closes as a polygon in the
// parameter plane. The region lies on BOTH sides of such an edge, so it borders nothing, and a foot on it
// would answer the query from a cut rather than from a boundary. The retired winding number cancelled the
// two traversals arithmetically; a nearest-foot test has to drop them instead. An imported STEP
// hemisphere is exactly this shape — the equator plus a meridian slit to the pole, walked both ways — and
// reading a side from that slit cost the sphere corpus its convex edges (#3453 follow-up).
func borderEdges(f curvedFace) []trimEdge {
	all := flatTrimEdges(f)
	out := make([]trimEdge, 0, len(all))
	for i, te := range all {
		if !walkedBothWays(all, i) {
			out = append(out, te)
		}
	}
	return out
}

// flatTrimEdges lists every loop edge of the face together with where it sits.
func flatTrimEdges(f curvedFace) []trimEdge {
	out := make([]trimEdge, 0, len(f.loops))
	for li := range f.loops {
		for ei, le := range f.loops[li].edges {
			out = append(out, trimEdge{le: le, loop: li, edge: ei})
		}
	}
	return out
}

// walkedBothWays reports whether another listed edge traces the same span BACKWARDS — the signature of a
// seam. Both uses of one topo.Edge derive their parameters from that edge, so a seam's two intervals are
// the same pair of floats swapped, and the comparison needs no tolerance.
func walkedBothWays(all []trimEdge, i int) bool {
	a := all[i].le
	for j, b := range all {
		if j != i && a.t0 == b.le.t1 && a.t1 == b.le.t0 && sameCurveSpan(a, b.le) {
			return true
		}
	}
	return false
}

// sameCurveSpan reports whether two loop edges trace the same points over the interval they share.
// Evaluating both curves at the SAME parameters is bit-exact when they are the same curve value, so this
// compares points rather than the curve interfaces, which need not be comparable at all.
func sameCurveSpan(a, b loopEdge) bool {
	for _, t := range []float64{a.t0, (a.t0 + a.t1) / 2, a.t1} {
		if a.curve.PointAt(t) != b.curve.PointAt(t) {
			return false
		}
	}
	return true
}

// closestTrimFoot returns the closest approach to p over every BORDERING edge. One global minimum is
// enough for the whole face — a hole's loop always sits between a point inside that hole and the outer
// loop, so the nearest boundary point already belongs to whichever loop separates p from the region.
// Ties break on the first edge in loop order, so the verdict is byte-identical across runs.
func closestTrimFoot(f curvedFace, border []trimEdge, p math.Point3) (trimFoot, bool) {
	best, found := trimFoot{}, false
	for _, te := range border {
		foot, ok := edgeFoot(f, te, p)
		if !ok || (found && foot.dist >= best.dist) {
			continue
		}
		best, found = foot, true
	}
	return best, found
}

// edgeFoot locates one edge's closest point to p and reads the inward direction there.
func edgeFoot(f curvedFace, te trimEdge, p math.Point3) (trimFoot, bool) {
	le := te.le
	loop, edge := te.loop, te.edge
	u := closestParamOnEdge(le, p)
	t := le.t0 + (le.t1-le.t0)*u
	inward, ok := inwardAt(f.surface, le, t)
	if !ok {
		return trimFoot{}, false
	}
	q := le.curve.PointAt(t)
	return trimFoot{loop: loop, edge: edge, u: u, point: q, dist: float64(q.DistanceTo(p)), inward: inward}, true
}

// inwardAt returns the unit direction that leaves the edge and enters the trimmed region at parameter t:
// the surface normal crossed with the LOOP-ORIENTED unit tangent (negated when the loop walks the curve
// backwards). A loop is walked with its region on the left seen from outside the surface, so n × T points
// into it. ok is false where the tangent runs along the normal — a degenerate edge, which names no side.
//
// Example: inwardAt(sphere, equatorWalkedClockwise, t) points at the SOUTH pole.
func inwardAt(s geom.Surface, le loopEdge, t float64) (math.Vector3, bool) {
	tangent := le.curve.TangentAt(t)
	if le.t1 < le.t0 {
		tangent = tangent.Negate()
	}
	u, v := s.ParamAt(le.curve.PointAt(t))
	inward := s.NormalAt(u, v).Cross(tangent)
	if inward.LengthSquared() == 0 {
		return math.V3(0, 0, 0), false
	}
	return inward.Scale(1 / inward.Length()), true
}

// cornerInward is the direction the side test measures against: the closest edge's own inward direction,
// plus the neighbouring edge's when the closest approach landed on the vertex the two SHARE. That sum is
// the angle-weighted pseudonormal (Bærentzen & Aanæs, IEEE TVCG 2005): at a sharp corner either edge
// alone reads the wrong side for a query sitting in the other's half-plane, while their sum reads it
// right for every query in the corner's normal cone.
func cornerInward(f curvedFace, border []trimEdge, best trimFoot) math.Vector3 {
	neighbour, ok := inwardAcrossVertex(f, border, best)
	if !ok {
		return best.inward
	}
	return best.inward.Add(neighbour)
}

// inwardAcrossVertex returns the inward direction of the edge that shares the vertex the closest approach
// landed on — the next edge when the foot is at the end of its own edge, the previous one when it is at
// the start. ok is false for a foot in the edge's interior, where there is no corner to blend, and for a
// neighbour that is a SEAM rather than a border: the region runs straight across a slit, so the boundary
// has no corner there to blend with.
func inwardAcrossVertex(f curvedFace, border []trimEdge, best trimFoot) (math.Vector3, bool) {
	n := len(f.loops[best.loop].edges)
	if best.u > 1-footVertexBand {
		return borderInwardAt(f, border, best.loop, (best.edge+1)%n, true)
	}
	if best.u < footVertexBand {
		return borderInwardAt(f, border, best.loop, (best.edge-1+n)%n, false)
	}
	return math.V3(0, 0, 0), false
}

// borderInwardAt reads the neighbouring BORDER edge's inward direction at the vertex it shares with the
// foot's edge: its t0 when it follows, its t1 when it precedes.
func borderInwardAt(f curvedFace, border []trimEdge, loop, edge int, follows bool) (math.Vector3, bool) {
	for _, te := range border {
		if te.loop != loop || te.edge != edge {
			continue
		}
		if follows {
			return inwardAt(f.surface, te.le, te.le.t0)
		}
		return inwardAt(f.surface, te.le, te.le.t1)
	}
	return math.V3(0, 0, 0), false
}

// closestParamOnEdge returns the NORMALIZED parameter (0 at t0, 1 at t1) of the edge point nearest p: a
// coarse scan brackets the global minimum, then golden section narrows the bracket.
func closestParamOnEdge(le loopEdge, p math.Point3) float64 {
	lo, hi := scanBracket(le, p)
	return goldenMinimum(le, p, lo, hi)
}

// scanBracket returns the normalized-parameter window around the nearest of footScanStations evenly
// spaced stations — one station either side of it, which brackets the true minimum.
func scanBracket(le loopEdge, p math.Point3) (lo, hi float64) {
	best, bestDist := 0, stdmath.Inf(1)
	for k := 0; k <= footScanStations; k++ {
		if d := footDistance(le, p, float64(k)/footScanStations); d < bestDist {
			best, bestDist = k, d
		}
	}
	return stdmath.Max(0, float64(best-1)/footScanStations), stdmath.Min(1, float64(best+1)/footScanStations)
}

// footDistance is the distance from p to the edge point at normalized parameter u.
func footDistance(le loopEdge, p math.Point3, u float64) float64 {
	return float64(le.curve.PointAt(le.t0 + (le.t1-le.t0)*u).DistanceTo(p))
}

// goldenMinimum narrows [lo, hi] onto the normalized parameter of least distance by golden-section
// search — derivative-free, so it holds for every curve kind, and the bracket shrinks by a fixed ratio
// each step, which makes the step count (and so the result) identical on every platform.
func goldenMinimum(le loopEdge, p math.Point3, lo, hi float64) float64 {
	const goldenRatio = 0.6180339887498949 // (√5−1)/2
	a, b := hi-(hi-lo)*goldenRatio, lo+(hi-lo)*goldenRatio
	fa, fb := footDistance(le, p, a), footDistance(le, p, b)
	for range footRefineSteps {
		if fa < fb {
			hi, b, fb = b, a, fa
			a = hi - (hi-lo)*goldenRatio
			fa = footDistance(le, p, a)
			continue
		}
		lo, a, fa = a, b, fb
		b = lo + (hi-lo)*goldenRatio
		fb = footDistance(le, p, b)
	}
	return (lo + hi) / 2
}
