// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// 3D spline sampling on the kernel's NURBS machinery — the 3D analogue of
// curvesample_spline.go, replacing the Catmull–Rom approximation (M06-F12,
// Oblikovati/Oblikovati#627).

// sampleChain3D samples a 3D point chain as open or closed. fit selects
// interpolation through the points (fit/fixed splines) versus the
// control-polygon B-spline (control-point splines). Chains shorter than three
// points, or fits the solver rejects, degrade to the defining polygon.
func sampleChain3D(pts []math.Point3, closed, fit bool) []math.Point3 {
	if len(pts) < 3 {
		return pts
	}
	if fit {
		return sampleFitChain3D(pts, closed)
	}
	return sampleControlChain3D(pts, closed)
}

// sampleFitChain3D evaluates the interpolating B-spline span by span, with
// the defining points landing exactly on the polyline.
func sampleFitChain3D(pts []math.Point3, closed bool) []math.Point3 {
	curve, ubar, err := fitCurve3DFor(pts, closed)
	if err != nil {
		return pts
	}
	out := make([]math.Point3, 0, (len(ubar)-1)*splineSamplesPerSpan+1)
	for k := 0; k+1 < len(ubar); k++ {
		out = append(out, pts[k%len(pts)])
		for j := 1; j < splineSamplesPerSpan; j++ {
			t := ubar[k] + (ubar[k+1]-ubar[k])*float64(j)/splineSamplesPerSpan
			out = append(out, curve.PointAt(t))
		}
	}
	if !closed {
		out = append(out, pts[len(pts)-1])
	}
	return out
}

// fitCurve3DFor builds the open or periodic interpolating 3D curve. 3D
// splines have no fit-method surface; they use the centripetal default.
func fitCurve3DFor(pts []math.Point3, closed bool) (geom.BSplineCurve, []float64, error) {
	if closed {
		return geom.NewClosedFittedBSplineCurve(pts, geom.FitCentripetal)
	}
	return geom.NewFittedBSplineCurveParam(pts, geom.FitCentripetal)
}

// sampleControlChain3D evaluates the control-polygon B-spline uniformly.
func sampleControlChain3D(pts []math.Point3, closed bool) []math.Point3 {
	curve, err := controlCurve3DFor(pts, closed)
	if err != nil {
		return pts
	}
	lo, hi := curve.Domain()
	segments := splineSamplesPerSpan * len(pts)
	last := segments - 1 // closed: the loop wraps back to the first sample
	if !closed {
		last = segments
	}
	out := make([]math.Point3, 0, last+1)
	for i := 0; i <= last; i++ {
		out = append(out, curve.PointAt(lo+(hi-lo)*float64(i)/float64(segments)))
	}
	return out
}

// controlCurve3DFor builds the clamped or periodic control-polygon 3D curve.
func controlCurve3DFor(pts []math.Point3, closed bool) (geom.BSplineCurve, error) {
	if closed {
		return geom.NewClosedControlBSplineCurve(pts)
	}
	degree := min(len(pts)-1, 3)
	return geom.NewBSplineCurveUniformWeights(degree, pts, clampedUniformKnots(len(pts), degree))
}
