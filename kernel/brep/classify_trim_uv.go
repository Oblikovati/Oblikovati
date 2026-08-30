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
	loops := faceUVLoops(f)
	if len(loops) == 0 {
		return true // no loop had enough extent to bound a region: treat the face as untrimmed
	}
	uPer, vPer := surfacePeriodic(f.surface)
	up, vp := f.surface.ParamAt(p)
	c := loopCentroid(loops[0])
	if uPer {
		up = unwrapAzimuthNear(c.X, up)
	}
	if vPer {
		vp = unwrapAzimuthNear(c.Y, vp)
	}
	return pointInUVLoops(math.P2(up, vp), loops)
}

// faceUVLoops projects a face's boundary loops into the surface (u, v) domain (outer loop first, to
// match pointInUVLoops), each edge sampled and its periodic parameters unwrapped into a continuous run.
func faceUVLoops(f curvedFace) [][]math.Point2 {
	uPer, vPer := surfacePeriodic(f.surface)
	out := make([][]math.Point2, 0, len(f.loops))
	for _, loop := range f.loops {
		if ring := loopToUV(f.surface, loop, uPer, vPer); len(ring) >= 3 {
			out = append(out, ring)
		}
	}
	return out
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
