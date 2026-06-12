// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Closed (periodic) control-polygon B-splines: the control points wrap around
// the loop (c_{n+j} = c_j) over a uniform periodic knot vector, so the curve
// is exactly C2 everywhere including the seam. Unlike the closed *fitted*
// curves in fitted_bspline_params.go, the curve approximates its polygon
// rather than passing through it (M06-F12, Oblikovati/Oblikovati#627).

// NewClosedControlBSplineCurve2d builds the cubic periodic B-spline over a
// closed 2D control polygon. Domain is [0, 1] for one full loop.
func NewClosedControlBSplineCurve2d(ctrl []math.Point2) (BSplineCurve2d, error) {
	if len(ctrl) < 3 {
		return BSplineCurve2d{}, fmt.Errorf("geom: closed control B-spline needs >= 3 points, got %d", len(ctrl))
	}
	wrapped := append(append([]math.Point2{}, ctrl...), ctrl[:periodicDegree]...)
	return NewBSplineCurve2dUniformWeights(periodicDegree, wrapped, uniformPeriodicKnots(len(ctrl)))
}

// NewClosedControlBSplineCurve is the 3D analogue of
// [NewClosedControlBSplineCurve2d].
func NewClosedControlBSplineCurve(ctrl []math.Point3) (BSplineCurve, error) {
	if len(ctrl) < 3 {
		return BSplineCurve{}, fmt.Errorf("geom: closed control B-spline needs >= 3 points, got %d", len(ctrl))
	}
	wrapped := append(append([]math.Point3{}, ctrl...), ctrl[:periodicDegree]...)
	return NewBSplineCurveUniformWeights(periodicDegree, wrapped, uniformPeriodicKnots(len(ctrl)))
}

// uniformPeriodicKnots is the unclamped knot vector knots[i] = (i−p)/n for
// n+p wrapped control points: the curve's valid domain [knots[p],
// knots[len−1−p]] comes out as exactly [0, 1], one loop.
func uniformPeriodicKnots(n int) []float64 {
	p := periodicDegree
	knots := make([]float64, n+2*p+1)
	for i := range knots {
		knots[i] = float64(i-p) / float64(n)
	}
	return knots
}
