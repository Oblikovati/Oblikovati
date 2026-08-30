// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// trimUVSamples is the per-edge sampling when a loop is projected into the surface's (u, v) domain.
// A point well inside the face classifies correctly from a modest polyline; a point nearer a curved
// trim edge than the sampling error is handled by faceBoundaryBand reselection, not by more samples —
// so this is a balance (fewer reselections) rather than an accuracy floor. It is not a face tessellation.
const trimUVSamples = 32

// domainPeriodTol accepts a parameter span as a full 2π turn; the analytic surfaces return an angular
// domain of exactly [0, 2π], so this only guards floating comparison, not a model quantity.
const domainPeriodTol = 1e-9 // tol:parametric — angular domain span ≈ 2π

// pointInTrimUV reports whether p (on f's surface) lies within f's trimmed region, by classifying its
// surface parameters against the loops projected into the (u, v) domain — the parameter-space
// classification production kernels use (OCCT BRepClass_FaceClassifier). Unlike the tangent-plane
// geodesic winding, it is correct for a full periodic band (a whole cylinder/cone side bounded by two
// complete circles), because a periodic parameter is unwrapped into one continuous branch before the
// even-odd test. A boundary-less face (a whole sphere/torus) contains every surface point.
func pointInTrimUV(f curvedFace, p math.Point3) bool {
	if len(f.loops) == 0 {
		return true
	}
	uPer, vPer := surfacePeriodic(f.surface)
	alongV := castAlongV(uPer, vPer)
	up, vp := f.surface.ParamAt(p)
	total := 0
	for _, loop := range f.loops {
		poly := loopToUV(f.surface, loop, uPer, vPer)
		if len(poly) < 2 {
			continue
		}
		total += loopRayCrossings(math.P2(up, vp), poly, uPer, vPer, alongV)
	}
	return total%2 == 1
}

// castAlongV chooses the parameter direction the containment ray travels: a NON-periodic axis, whose
// +∞ end is unambiguously outside every trim loop. It casts along v unless v is the periodic axis and
// u is not (a torus, periodic in both, has no non-periodic axis and falls back to v).
func castAlongV(uPer, vPer bool) bool { return !vPer || uPer }

// loopRayCrossings counts how many times a ray from q along the chosen non-periodic axis crosses one
// trim loop. The query is first shifted into that loop's own parameter branch (loops unwrap
// independently), and a loop that WRAPS the periodic cross-axis a full turn — a tunnel-wall end curve,
// a wrapped-pad rim — is left OPEN, since closing it with a seam-spanning chord would add a spurious
// second crossing and cancel the real one. This even-odd count over each loop is why a strip face
// (two non-nested boundary loops) and a nested outer+hole face both classify correctly.
func loopRayCrossings(q math.Point2, poly []math.Point2, uPer, vPer, alongV bool) int {
	c := loopCentroid(poly)
	qu, qv := q.X, q.Y
	if uPer {
		qu = unwrapAzimuthNear(c.X, qu)
	}
	if vPer {
		qv = unwrapAzimuthNear(c.Y, qv)
	}
	closed := !loopWrapsCrossAxis(poly, uPer, vPer, alongV)
	return rayCrossingCount(math.P2(qu, qv), poly, closed, alongV)
}

// loopWrapsCrossAxis reports whether the loop makes a full 2π circuit of the periodic axis that is
// PERPENDICULAR to the cast direction (u when casting along v). Such a loop is an open curve spanning
// the whole periodic range, not a closed ring.
func loopWrapsCrossAxis(poly []math.Point2, uPer, vPer, alongV bool) bool {
	crossPeriodic := uPer
	net := poly[len(poly)-1].X - poly[0].X
	if !alongV {
		crossPeriodic = vPer
		net = poly[len(poly)-1].Y - poly[0].Y
	}
	return crossPeriodic && stdmath.Abs(stdmath.Abs(net)-2*stdmath.Pi) < 0.5 // tol:parametric — full-turn circuit
}

// rayCrossingCount counts crossings of a ray from q toward +axis (v when alongV, else u) with the
// polyline, standard crossing-number even-odd. An open polyline (closed=false) omits the last→first
// segment.
func rayCrossingCount(q math.Point2, poly []math.Point2, closed, alongV bool) int {
	n, segs := len(poly), len(poly)-1
	if closed {
		segs = len(poly)
	}
	count := 0
	for i := 0; i < segs; i++ {
		if raySegmentCrosses(q, poly[i], poly[(i+1)%n], alongV) {
			count++
		}
	}
	return count
}

// raySegmentCrosses reports whether the ray from q toward +axis crosses segment ab. The segment must
// straddle q on the cross-axis, and its interpolated axis coordinate there must lie beyond q.
func raySegmentCrosses(q, a, b math.Point2, alongV bool) bool {
	ac, bc, qc := a.X, b.X, q.X // cross-axis (u) coordinates when casting along v
	ap, bp, qp := a.Y, b.Y, q.Y // ray-axis (v) coordinates
	if !alongV {
		ac, bc, qc, ap, bp, qp = a.Y, b.Y, q.Y, a.X, b.X, q.X
	}
	if (ac > qc) == (bc > qc) {
		return false // does not straddle q on the cross-axis
	}
	t := (qc - ac) / (bc - ac)
	return ap+t*(bp-ap) > qp
}

// loopToUV walks a loop's edges into a continuous (u, v) polyline: each sample's periodic parameter is
// shifted to the branch of the previous sample, so an edge crossing the u=0≡2π seam stays monotone
// rather than jumping a full turn.
func loopToUV(s geom.Surface, loop curvedLoop, uPer, vPer bool) []math.Point2 {
	var ring []math.Point2
	for _, e := range loop.edges {
		for k := 0; k < trimUVSamples; k++ {
			t := e.t0 + (e.t1-e.t0)*float64(k)/trimUVSamples
			u, v := s.ParamAt(e.curve.PointAt(t))
			u, v = continueUV(ring, u, v, uPer, vPer)
			ring = append(ring, math.P2(u, v))
		}
	}
	return ring
}

// continueUV shifts (u, v) by whole turns so each periodic coordinate stays within half a turn of the
// previous sample, keeping the projected loop continuous across the seam.
func continueUV(ring []math.Point2, u, v float64, uPer, vPer bool) (float64, float64) {
	if len(ring) == 0 {
		return u, v
	}
	last := ring[len(ring)-1]
	if uPer {
		u = unwrapAzimuthNear(last.X, u)
	}
	if vPer {
		v = unwrapAzimuthNear(last.Y, v)
	}
	return u, v
}

// surfacePeriodic reports which of the surface's parameter directions wrap around a full 2π turn.
func surfacePeriodic(s geom.Surface) (uPer, vPer bool) {
	return domainPeriodic(s.UDomain()), domainPeriodic(s.VDomain())
}

// domainPeriodic reports whether a parameter domain is a finite full turn [lo, lo+2π].
func domainPeriodic(lo, hi float64) bool {
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return false
	}
	return stdmath.Abs((hi-lo)-2*stdmath.Pi) < domainPeriodTol
}

// loopCentroid returns the average of a ring's (u, v) vertices — the branch reference the query point
// is unwrapped toward, so it lands in the same turn as the loop rather than a neighbouring one.
func loopCentroid(ring []math.Point2) math.Point2 {
	var su, sv float64
	for _, q := range ring {
		su += q.X
		sv += q.Y
	}
	n := float64(len(ring))
	return math.P2(su/n, sv/n)
}

// faceBoundaryBand is twice the largest gap between a trim edge and its sampled polyline (the chord
// sagitta over one trimUVSamples segment). Inside this band of a curved trim edge the projected
// polygon may misclassify a point, so the classifier reselects the ray rather than trust the polygon;
// beyond it the sampled trim is reliable. A face with only straight edges (a planar polygon, a
// cylinder/cone band whose bounds are circles-as-v-const) reports 0.
func faceBoundaryBand(f curvedFace) float64 {
	worst := 0.0
	for _, loop := range f.loops {
		for _, e := range loop.edges {
			if s := edgeChordSagitta(e); s > worst {
				worst = s
			}
		}
	}
	return 2 * worst
}

// edgeChordSagitta is the largest distance from an edge's true midpoint to its chord midpoint over
// the trimUVSamples segments the trim polygon uses — the polygon's worst deviation from the edge.
func edgeChordSagitta(e loopEdge) float64 {
	worst := 0.0
	for k := 0; k < trimUVSamples; k++ {
		t0 := e.t0 + (e.t1-e.t0)*float64(k)/trimUVSamples
		t1 := e.t0 + (e.t1-e.t0)*float64(k+1)/trimUVSamples
		chordMid := e.curve.PointAt(t0).Midpoint(e.curve.PointAt(t1))
		if s := float64(e.curve.PointAt((t0 + t1) / 2).DistanceTo(chordMid)); s > worst {
			worst = s
		}
	}
	return worst
}
