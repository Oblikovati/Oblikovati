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

// pointInCurvedFace reports whether p (assumed on f's surface) lies within f's trimmed region. A
// boundary-less face — a whole sphere or torus — contains every surface point.
func pointInCurvedFace(f curvedFace, p math.Point3) bool {
	if len(f.loops) == 0 {
		return true
	}
	best, ok := closestTrimFoot(f, p)
	if !ok {
		return true // every edge degenerate: no boundary names a side, so the whole surface is the region
	}
	return float64(best.point.VectorTo(p).Dot(cornerInward(f, best))) > 0
}

// closestTrimFoot returns the closest approach to p over EVERY edge of every loop. One global minimum is
// enough for the whole face — a hole's loop always sits between a point inside that hole and the outer
// loop, so the nearest boundary point already belongs to whichever loop separates p from the region.
// Ties break on the first edge in loop order, so the verdict is byte-identical across runs.
func closestTrimFoot(f curvedFace, p math.Point3) (trimFoot, bool) {
	best, found := trimFoot{}, false
	for li := range f.loops {
		for ei := range f.loops[li].edges {
			foot, ok := edgeFoot(f, li, ei, p)
			if !ok || (found && foot.dist >= best.dist) {
				continue
			}
			best, found = foot, true
		}
	}
	return best, found
}

// edgeFoot locates one edge's closest point to p and reads the inward direction there.
func edgeFoot(f curvedFace, loop, edge int, p math.Point3) (trimFoot, bool) {
	le := f.loops[loop].edges[edge]
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
func cornerInward(f curvedFace, best trimFoot) math.Vector3 {
	neighbour, ok := inwardAcrossVertex(f, best)
	if !ok {
		return best.inward
	}
	return best.inward.Add(neighbour)
}

// inwardAcrossVertex returns the inward direction of the edge that shares the vertex the closest approach
// landed on — the next edge when the foot is at the end of its own edge, the previous one when it is at
// the start. ok is false for a foot in the edge's interior, where there is no corner to blend.
func inwardAcrossVertex(f curvedFace, best trimFoot) (math.Vector3, bool) {
	edges := f.loops[best.loop].edges
	if best.u > 1-footVertexBand {
		next := edges[(best.edge+1)%len(edges)]
		return inwardAt(f.surface, next, next.t0)
	}
	if best.u < footVertexBand {
		prev := edges[(best.edge-1+len(edges))%len(edges)]
		return inwardAt(f.surface, prev, prev.t1)
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
