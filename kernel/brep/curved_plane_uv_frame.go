// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The exact conic∩polygon numerics for the planeUV operand (partial curved-on-planar boolean, #1591,
// ADR-0049 D-d). The seat plane's (u,v) chart is an isometry (orthonormal Û,V̂), so the imprint conic —
// which lies IN the seat plane — maps into the chart with its radii unchanged, and every model-scaled 3-D
// tolerance transfers into (u,v) with no Jacobian rescaling. Crossings are solved EXACTLY (a stable
// quadratic; the ellipse affine-normalised to the unit circle) because a crossing is the topologically
// decisive vertex: a sampled crossing that two adjacent faces compute differently is the blind-straddle
// sliver leak. The gate is decline-biased — any tangency/eccentric/odd-parity contact defers to CSG.

// planeConic is the imprint conic in the seat plane's (u,v): centre, unit major-axis direction and the two
// semi-axes (A≥B; a circle has A==B==r, maj arbitrary). Built from the geom.Circle / geom.EllipseFull that
// IntersectSurfacesAnalytic returns, projected through to2D/to2Dvec.
type planeConic struct {
	center math.Point2
	maj    math.Vector2 // unit major-axis direction in (u,v)
	A, B   float64      // semi-axes, A>=B
}

// conicHit is one exact crossing of the conic with a polygon edge: the edge parameter in [0,1] and the
// crossing point in the (u,v) chart (exactly on the edge; on the conic to the quadratic's precision).
type conicHit struct {
	sEdge float64
	p     math.Point2
}

// toPlaneConic projects a 3-D imprint conic (which lies in the seat plane pl) into pl's (u,v) chart. It
// handles the two shapes a cylinder/cone∩plane yields; ok=false for an open arm (parabola/hyperbola) or any
// other curve, which the planeUV gate declines to CSG for now (#1591).
func toPlaneConic(curve geom.Curve3, pl geom.Plane) (planeConic, bool) {
	switch c := curve.(type) {
	case geom.Circle:
		return planeConic{center: to2D(pl, c.Center), maj: math.V2(1, 0), A: c.Radius, B: c.Radius}, true
	case geom.EllipseFull:
		maj := unitVec2(to2Dvec(pl, c.MajorAxis.AsVector()))
		return planeConic{center: to2D(pl, c.Center), maj: maj, A: c.MajorRadius, B: c.MinorRadius}, true
	default:
		return planeConic{}, false
	}
}

// conicEdgeHits solves C ∩ (a→b) EXACTLY, keeping only crossings STRICTLY inside the edge.
func conicEdgeHits(pc planeConic, a, b math.Point2, res geom.Resolution) (hits []conicHit, tangent bool) {
	return conicSegmentHits(pc, a, b, res, tjTol)
}

// conicFrameHits is conicEdgeHits over the CLOSED segment: a root AT an endpoint counts as a crossing.
// An imprint clipped to this very conic (uvPairSegments → curvedFaceLineIntervals) terminates ON it by
// construction, so the strict-interior window drops precisely the crossings the conic frame needs; the
// arrangement then resolves the contact against a SAMPLED chord of the circle instead, splitting the
// shared vertex into two copies 1.5e-4 apart at imprintSampleCount=256 (#3488).
func conicFrameHits(pc planeConic, a, b math.Point2, res geom.Resolution) (hits []conicHit, tangent bool) {
	return conicSegmentHits(pc, a, b, res, -tjTol)
}

// conicSegmentHits solves C ∩ (a→b) EXACTLY, accepting roots in the window [sPad, 1−sPad]. It affine-
// normalises the ellipse to the unit circle (a linear map whose condition number is A/B, exactly), then
// solves the STABLE quadratic |A'+s·d|²=1 with the Kahan form qq=-½(β+sign(β)√Δ) so the cancellation that
// afflicts one root when β²≫4αγ never bites. The edge parameter s is invariant under the affine map, so
// the crossing point is a+s·(b−a) in the original chart. tangent=true when the two roots coincide within
// a weld (a grazing double root — a sliver risk to decline).
func conicSegmentHits(pc planeConic, a, b math.Point2, res geom.Resolution, sPad float64) (hits []conicHit, tangent bool) {
	ax, ay := pc.normalize(a)
	bx, by := pc.normalize(b)
	dx, dy := bx-ax, by-ay
	alpha := dx*dx + dy*dy
	if alpha < res.Weld()*res.Weld() {
		return nil, false // a degenerate (sub-weld) edge
	}
	beta := 2 * (ax*dx + ay*dy)
	gamma := ax*ax + ay*ay - 1
	disc := beta*beta - 4*alpha*gamma
	if disc < 0 {
		return nil, false // the edge line misses the conic
	}
	s1, s2 := stableQuadraticRoots(alpha, beta, gamma, disc)
	hits = appendEdgeHit(hits, a, b, s1, sPad)
	hits = appendEdgeHit(hits, a, b, s2, sPad)
	return hits, conicTangent(a, b, s1, s2, disc, res)
}

// normalize maps a chart point into the frame where the conic is the unit circle: (ξ,η) = ((p−c)·maj/A,
// (p−c)·maj⊥/B). An affine map, so segment line-parameters are preserved.
func (pc planeConic) normalize(p math.Point2) (xi, eta float64) {
	d := pc.center.VectorTo(p)
	perp := math.V2(-pc.maj.Y, pc.maj.X)
	return float64(d.Dot(pc.maj)) / pc.A, float64(d.Dot(perp)) / pc.B
}

// stableQuadraticRoots returns the two roots of αs²+βs+γ=0 via the Kahan/Numerical-Recipes form, which
// avoids catastrophic cancellation by never subtracting two near-equal quantities in the numerator.
func stableQuadraticRoots(alpha, beta, gamma, disc float64) (float64, float64) {
	qq := -0.5 * (beta + stdmath.Copysign(stdmath.Sqrt(disc), beta))
	if qq == 0 { // β=0 and γ=0: the edge passes through the conic centre tangentially in the map
		return 0, 0
	}
	return qq / alpha, gamma / qq
}

// appendEdgeHit adds the crossing at edge parameter s when s lies in the acceptance window
// [sPad, 1−sPad]. A positive pad (conicEdgeHits) keeps the crossing strictly inside the edge, leaving
// endpoints to the vertex-snap path; a negative one (conicFrameHits) admits them — tjTol matches the
// arrangement welder's tolerance either way.
func appendEdgeHit(hits []conicHit, a, b math.Point2, s, sPad float64) []conicHit {
	if s < sPad || s > 1-sPad {
		return hits
	}
	p := math.P2(a.X+(b.X-a.X)*math.Scalar(s), a.Y+(b.Y-a.Y)*math.Scalar(s))
	return append(hits, conicHit{sEdge: s, p: p})
}

// conicTangent reports a grazing contact: the two crossing points fall within a weld of each other, so the
// edge touches the conic instead of transversally piercing it — a zero-width sliver the stitch cannot weld,
// which the gate declines. Measured in the original chart (a real length), so the band is model-scaled.
func conicTangent(a, b math.Point2, s1, s2, disc float64, res geom.Resolution) bool {
	if disc <= 0 {
		return true // a true double root (or a miss treated as grazing)
	}
	p1 := math.P2(a.X+(b.X-a.X)*math.Scalar(s1), a.Y+(b.Y-a.Y)*math.Scalar(s1))
	p2 := math.P2(a.X+(b.X-a.X)*math.Scalar(s2), a.Y+(b.Y-a.Y)*math.Scalar(s2))
	if s1 < -1 || s1 > 2 || s2 < -1 || s2 > 2 {
		return false // both roots far outside the edge — the near-miss is off this edge, not a grazing touch
	}
	return float64(p1.DistanceTo(p2)) < res.Weld()
}

// unitVec2 returns v normalised, or (1,0) for a degenerate vector (the conic's major axis is always
// well-defined for a real ellipse, so the fallback is unreachable in practice).
func unitVec2(v math.Vector2) math.Vector2 {
	l := float64(v.Length())
	if l < 1e-12 {
		return math.V2(1, 0)
	}
	return math.V2(v.X/math.Scalar(l), v.Y/math.Scalar(l))
}

// conicParamAt inverts a point on the imprint conic to its curve parameter t∈[0,1) — the closed-form
// inversion (no iteration) that is the exact weld currency: the seat arc, the overhang-cap arc and the tool
// wall's base all terminate at conic.PointAt(t) for the SAME t, so they weld byte-identically (ADR-0049 D-d).
// It matches geom.Circle/EllipseFull's own PointAt convention (angle 2πt from RefDir/MajorAxis).
func conicParamAt(cv geom.Curve3, p math.Point3) (float64, bool) {
	switch c := cv.(type) {
	case geom.Circle:
		d := c.Center.VectorTo(p)
		cos := d.Dot(c.RefDir.AsVector())
		sin := d.Dot(c.Normal.Cross(c.RefDir))
		return wrap01(stdmath.Atan2(float64(sin), float64(cos)) / (2 * stdmath.Pi)), true
	case geom.EllipseFull:
		d := c.Center.VectorTo(p)
		cos := float64(d.Dot(c.MajorAxis.AsVector())) / c.MajorRadius
		sin := float64(d.Dot(c.Normal.Cross(c.MajorAxis))) / c.MinorRadius
		return wrap01(stdmath.Atan2(sin, cos) / (2 * stdmath.Pi)), true
	default:
		return 0, false
	}
}

// wrap01 folds a real onto [0,1).
func wrap01(t float64) float64 {
	t -= stdmath.Floor(t)
	if t >= 1 {
		t -= 1
	}
	return t
}

// eccCap is the minor/major ratio below which the imprint ellipse is too eccentric to trust: the tool axis
// grazes the seat plane (φ→90°), the normalising map is ill-conditioned (cond = A/B), and the "seat contact"
// is a graze, not a real partial hole. Below it the gate declines to CSG (ADR-0049 D-d, pitfall 5).
const eccCap = 1e-3

// planeUVContactOK is the accept/decline gate generalising bossSeatFace/circleVsCap for a PARTIAL contact.
// It is decline-biased: an eccentric ellipse, any tangency, or an odd crossing parity (a missed grazing
// crossing that would leave a non-manifold kept region) returns false so the caller keeps CSG. A partial
// contact must cross the polygon boundary an EVEN number of times (in-and-out); zero crossings is not
// partial (the strictly-interior fast path or a clean miss owns that), so it declines too.
func planeUVContactOK(pc planeConic, uvLoops [][]math.Point2, res geom.Resolution) bool {
	if pc.B/pc.A < eccCap {
		return false
	}
	crossings := 0
	for _, ring := range uvLoops {
		for i, n := 0, len(ring); i < n; i++ {
			hits, tangent := conicEdgeHits(pc, ring[i], ring[(i+1)%n], res)
			if tangent {
				return false
			}
			crossings += len(hits)
		}
	}
	return crossings > 0 && crossings%2 == 0
}
