// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Rebuilding a curve to a clean Class-A NURBS (M36-F02): sample the source densely, fit a
// fresh low-degree B-spline with a chosen, small control-point count by least squares
// (approximate_bspline.go), and report the worst geometric deviation from the original so a
// caller can accept the rebuild only when it stays within tolerance. A single span
// (nctrl = degree+1) gives the cleanest result where the shape allows.

// defaultRebuildSamples is the sample density used when a caller does not specify one; dense
// enough to capture the wiggle of a multi-span source, cheap enough for interactive use.
const defaultRebuildSamples = 200

// NewApproximatedBSplineCurve fits a degree-`degree` B-spline with `nctrl` control points to
// the points by least squares (endpoints interpolated). It errors when the size relationships
// (degree+1 <= nctrl <= len(points)) are violated or the points are degenerate.
func NewApproximatedBSplineCurve(points []math.Point3, degree, nctrl int, param FitParameterization) (BSplineCurve, error) {
	ubar, err := alphaParams(coords3(points), param.alpha())
	if err != nil {
		return BSplineCurve{}, err
	}
	ctrl, knots, err := approximateLS(coords3(points), degree, nctrl, ubar)
	if err != nil {
		return BSplineCurve{}, err
	}
	return NewBSplineCurveUniformWeights(degree, points3(ctrl), knots)
}

// NewApproximatedBSplineCurve2d is the 2D analogue of [NewApproximatedBSplineCurve].
func NewApproximatedBSplineCurve2d(points []math.Point2, degree, nctrl int, param FitParameterization) (BSplineCurve2d, error) {
	ubar, err := alphaParams(coords2(points), param.alpha())
	if err != nil {
		return BSplineCurve2d{}, err
	}
	ctrl, knots, err := approximateLS(coords2(points), degree, nctrl, ubar)
	if err != nil {
		return BSplineCurve2d{}, err
	}
	return NewBSplineCurve2dUniformWeights(degree, points2(ctrl), knots)
}

// RebuildCurve refits src to a clean degree-`degree` B-spline with `nctrl` control points,
// returning the rebuilt curve and the maximum geometric deviation from src. It samples src
// at `samples` intervals (0 ⇒ a sensible default); pass nctrl = degree+1 for a single-span
// Class-A rebuild. The deviation lets the caller decide whether to accept the result.
//
// Unlike [NewApproximatedBSplineCurve] (which re-parameterizes an arbitrary point list by
// chord length), Rebuild fits in src's OWN parameterization — sample parameters spaced
// uniformly across src's domain — so the rebuild changes only the control-point structure,
// not the parameterization. That makes it idempotent and exact on an already-clean curve.
//
// Example: clean, dev, _ := RebuildCurve(wavy, 3, 4, 0); if dev <= tol { use(clean) }.
func RebuildCurve(src Curve3, degree, nctrl, samples int) (BSplineCurve, float64, error) {
	if samples <= 0 {
		samples = defaultRebuildSamples
	}
	pts, ubar := sampleCurveAt(src, samples)
	ctrl, knots, err := approximateLS(pts, degree, nctrl, ubar)
	if err != nil {
		return BSplineCurve{}, 0, err
	}
	rebuilt, err := NewBSplineCurveUniformWeights(degree, points3(ctrl), knots)
	if err != nil {
		return BSplineCurve{}, 0, err
	}
	return rebuilt, curveDeviation(src, rebuilt, samples), nil
}

// sampleCurveAt returns samples+1 points of src and their uniform parameters in [0,1] (src's
// domain mapped linearly), the parameterization the rebuild preserves.
func sampleCurveAt(src Curve3, samples int) (pts [][]float64, ubar []float64) {
	lo, hi := src.Domain()
	pts = make([][]float64, samples+1)
	ubar = make([]float64, samples+1)
	for i := range pts {
		f := float64(i) / float64(samples)
		p := src.PointAt(lo + (hi-lo)*f)
		pts[i] = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
		ubar[i] = f
	}
	return pts, ubar
}

// curveDeviation returns the largest distance from a point sampled along src to its closest
// point on the rebuilt curve — the achieved approximation error. It projects each src sample
// onto the rebuilt curve via the curve's own nearest-point query.
func curveDeviation(src Curve3, rebuilt BSplineCurve, samples int) float64 {
	lo, hi := src.Domain()
	maxDev := 0.0
	for i := 0; i <= samples; i++ {
		t := lo + (hi-lo)*float64(i)/float64(samples)
		p := src.PointAt(t)
		u, _ := CurveParamAtPoint3(rebuilt, p)
		if d := float64(p.DistanceTo(rebuilt.PointAt(u))); d > maxDev {
			maxDev = d
		}
	}
	return maxDev
}
