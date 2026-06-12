// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A sketch spline only stores its defining points; region detection, trim
// targeting and the faceted extruded cross-section need a polyline that
// follows the true curve. Splines are evaluated on the kernel's NURBS
// machinery — interpolating (fitted) B-splines for fit splines, periodic/
// clamped control-polygon B-splines for control splines — replacing the
// Catmull–Rom approximation that under-delivered region and area fidelity
// (M06-F12, Oblikovati/Oblikovati#627; deferred-work registry #599).

// splineSamplesPerSpan is how many straight segments approximate each span
// between two consecutive defining points.
const splineSamplesPerSpan = 8

// sampleSplineEntity approximates a spline with a polyline evaluated on its
// true NURBS curve. A closed spline wraps to its first point (no duplicate
// closing vertex, like sampleCircle); an open one runs first→last with both
// endpoints included. Degenerate inputs (under 3 points, or a fit the solver
// rejects) degrade to the defining polygon — strictly the pre-NURBS behavior.
func sampleSplineEntity(sp *Spline) []math.Point2 {
	return sampleSplineEntityN(sp, splineSamplesPerSpan)
}

// sampleSplineEntityN is sampleSplineEntity at caller-chosen per-span density
// (region properties scale it with the requested accuracy — M06-F08, #623).
func sampleSplineEntityN(sp *Spline, perSpan int) []math.Point2 {
	pts := splinePositions(sp)
	if len(pts) < 3 {
		return pts
	}
	if sp.IsFitType() {
		return sampleFitSpline(pts, sp.Closed, fitParameterization(sp.FitMethod), perSpan)
	}
	return sampleControlSpline(pts, sp.Closed, perSpan)
}

// fitParameterization maps the public fit method onto the kernel
// parameterization; the zero value (no method chosen) is the smooth default.
func fitParameterization(m SplineFitMethod) geom.FitParameterization {
	switch m {
	case types.SplineFitChord:
		return geom.FitChordLength
	case types.SplineFitSweet:
		return geom.FitUniform
	default:
		return geom.FitCentripetal
	}
}

// sampleFitSpline evaluates the interpolating B-spline span by span — the
// defining points land exactly on the polyline (at their fit parameters), so
// region detection sees them as vertices just as it always did.
func sampleFitSpline(pts []math.Point2, closed bool, param geom.FitParameterization, perSpan int) []math.Point2 {
	curve, ubar, err := fitCurveFor(pts, closed, param)
	if err != nil {
		return pts
	}
	out := make([]math.Point2, 0, (len(ubar)-1)*perSpan+1)
	for k := 0; k+1 < len(ubar); k++ {
		out = append(out, pts[k%len(pts)])
		for j := 1; j < perSpan; j++ {
			t := ubar[k] + (ubar[k+1]-ubar[k])*float64(j)/float64(perSpan)
			out = append(out, curve.PointAt(t))
		}
	}
	if !closed {
		out = append(out, pts[len(pts)-1])
	}
	return out
}

// fitCurveFor builds the open (clamped) or closed (periodic) interpolating
// curve plus the defining points' parameters.
func fitCurveFor(pts []math.Point2, closed bool, param geom.FitParameterization) (geom.BSplineCurve2d, []float64, error) {
	if closed {
		return geom.NewClosedFittedBSplineCurve2d(pts, param)
	}
	return geom.NewFittedBSplineCurve2dParam(pts, param)
}

// sampleControlSpline evaluates the control-polygon B-spline (the points
// shape, not pin, the curve) uniformly over its domain.
func sampleControlSpline(pts []math.Point2, closed bool, perSpan int) []math.Point2 {
	curve, err := controlCurveFor(pts, closed)
	if err != nil {
		return pts
	}
	lo, hi := curve.Domain()
	segments := perSpan * len(pts)
	out := make([]math.Point2, 0, segments+1)
	last := segments - 1 // closed: the loop wraps back to the first sample
	if !closed {
		last = segments
	}
	for i := 0; i <= last; i++ {
		out = append(out, curve.PointAt(lo+(hi-lo)*float64(i)/float64(segments)))
	}
	return out
}

// controlCurveFor builds the clamped (open) or periodic (closed)
// control-polygon B-spline.
func controlCurveFor(pts []math.Point2, closed bool) (geom.BSplineCurve2d, error) {
	if closed {
		return geom.NewClosedControlBSplineCurve2d(pts)
	}
	degree := len(pts) - 1
	if degree > 3 {
		degree = 3
	}
	return geom.NewBSplineCurve2dUniformWeights(degree, pts, clampedUniformKnots(len(pts), degree))
}

// clampedUniformKnots is the open-curve knot vector: degree+1 zeros, evenly
// spaced interior knots, degree+1 ones.
func clampedUniformKnots(n, degree int) []float64 {
	knots := make([]float64, n+degree+1)
	interior := n - degree
	for i := range knots {
		switch {
		case i <= degree:
			knots[i] = 0
		case i >= n:
			knots[i] = 1
		default:
			knots[i] = float64(i-degree) / float64(interior)
		}
	}
	return knots
}

// splinePositions returns the spline's defining-point positions.
func splinePositions(sp *Spline) []math.Point2 {
	out := make([]math.Point2, len(sp.Points))
	for i, p := range sp.Points {
		out[i] = p.Position()
	}
	return out
}
