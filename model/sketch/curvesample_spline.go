// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "github.com/Oblikovati/oblikovati/math"

// A sketch spline only stores its defining points; region detection and the faceted
// extruded cross-section need a smooth polyline through them. Both fit and control
// splines are interpolated here with a uniform Catmull–Rom: it passes through the
// points and is naturally periodic for a closed loop. Exact approximating (control-
// point) NURBS fidelity is a B-rep curve concern, deferred to the kernel; this
// representative polygon only has to follow the curve closely enough to detect the
// region and facet the prism — strictly better than the chord it replaced.

// splineSamplesPerSpan is how many straight segments approximate each span between
// two consecutive spline points.
const splineSamplesPerSpan = 8

// sampleSplineEntity approximates a spline with a smooth polyline. A closed spline
// wraps to its first point (no duplicate closing vertex, like sampleCircle); an open
// one runs first→last with both endpoints included.
func sampleSplineEntity(sp *Spline) []math.Point2 {
	pts := splinePositions(sp)
	if len(pts) < 3 {
		return pts // a 1- or 2-point spline is just its chord
	}
	if sp.Closed {
		return catmullRomClosed(pts)
	}
	return catmullRomOpen(pts)
}

// splinePositions returns the spline's defining-point positions.
func splinePositions(sp *Spline) []math.Point2 {
	out := make([]math.Point2, len(sp.Points))
	for i, p := range sp.Points {
		out[i] = p.Position()
	}
	return out
}

// catmullRomOpen samples an open spline through pts, endpoints duplicated as the
// phantom neighbours so the curve starts and ends exactly on the first/last point.
func catmullRomOpen(pts []math.Point2) []math.Point2 {
	n := len(pts)
	out := []math.Point2{}
	for i := 0; i < n-1; i++ {
		out = append(out, spanSamples(pts[clampIndex(i-1, n-1)], pts[i], pts[i+1], pts[clampIndex(i+2, n-1)])...)
	}
	return append(out, pts[n-1]) // include the final endpoint
}

// catmullRomClosed samples a closed spline, wrapping neighbour indices around the
// loop; the closing vertex is omitted (the loop wraps back to pts[0]).
func catmullRomClosed(pts []math.Point2) []math.Point2 {
	n := len(pts)
	out := []math.Point2{}
	for i := 0; i < n; i++ {
		out = append(out, spanSamples(pts[(i-1+n)%n], pts[i], pts[(i+1)%n], pts[(i+2)%n])...)
	}
	return out
}

// spanSamples returns the Catmull–Rom samples on the p1→p2 span at t∈[0,1): p1 and
// the interior points, but not p2 (the next span contributes it).
func spanSamples(p0, p1, p2, p3 math.Point2) []math.Point2 {
	out := make([]math.Point2, splineSamplesPerSpan)
	for j := 0; j < splineSamplesPerSpan; j++ {
		out[j] = catmullRomPoint(p0, p1, p2, p3, float64(j)/float64(splineSamplesPerSpan))
	}
	return out
}

// catmullRomPoint evaluates the uniform Catmull–Rom spline of the p1→p2 span (with
// neighbours p0,p3) at local parameter t∈[0,1].
func catmullRomPoint(p0, p1, p2, p3 math.Point2, t float64) math.Point2 {
	t2, t3 := t*t, t*t*t
	return math.P2(
		0.5*(2*p1.X+(-p0.X+p2.X)*t+(2*p0.X-5*p1.X+4*p2.X-p3.X)*t2+(-p0.X+3*p1.X-3*p2.X+p3.X)*t3),
		0.5*(2*p1.Y+(-p0.Y+p2.Y)*t+(2*p0.Y-5*p1.Y+4*p2.Y-p3.Y)*t2+(-p0.Y+3*p1.Y-3*p2.Y+p3.Y)*t3),
	)
}

// clampIndex clamps i to [0, hi] — used to reuse an open spline's endpoints as their
// own phantom neighbours.
func clampIndex(i, hi int) int {
	if i < 0 {
		return 0
	}
	if i > hi {
		return hi
	}
	return i
}
