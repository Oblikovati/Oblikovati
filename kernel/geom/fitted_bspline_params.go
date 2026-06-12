// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Parameterized and closed variants of the fitted (interpolating) B-splines in
// fitted_bspline.go (M06-F11/F12, Oblikovati/Oblikovati#626/#627). The
// parameterization choice is the geometric meaning of the public spline
// fit-method setting: it decides how "speed" is distributed along the curve
// and therefore how the curve bellies between unevenly spaced points.

// FitParameterization selects how interpolation parameters are spaced.
type FitParameterization int

const (
	// FitCentripetal (α = ½) avoids cusps/self-crossings near tight point
	// clusters — the public "smooth" fit method and the default.
	FitCentripetal FitParameterization = iota
	// FitChordLength (α = 1) spaces parameters by chord distance — the
	// public "chord" fit method.
	FitChordLength
	// FitUniform (α = 0) ignores spacing entirely, approximating the
	// minimum-energy ("sweet") fit of the public surface.
	FitUniform
)

// alpha is the exponent the chord length is raised to for this spacing.
func (p FitParameterization) alpha() float64 {
	switch p {
	case FitChordLength:
		return 1
	case FitUniform:
		return 0
	default:
		return 0.5
	}
}

// NewFittedBSplineCurve2dParam interpolates the points like
// [NewFittedBSplineCurve2d] but with a selectable parameterization. It also
// returns each input point's parameter on the curve (ū₀=0 … ūₙ=1), which
// callers use to sample span-by-span or to anchor per-fit-point handles.
func NewFittedBSplineCurve2dParam(points []math.Point2, param FitParameterization) (BSplineCurve2d, []float64, error) {
	degree, err := fitDegree(len(points))
	if err != nil {
		return BSplineCurve2d{}, nil, err
	}
	ubar, err := alphaParams(coords2(points), param.alpha())
	if err != nil {
		return BSplineCurve2d{}, nil, err
	}
	ctrl, knots, err := fitInterpolationAt(coords2(points), degree, ubar)
	if err != nil {
		return BSplineCurve2d{}, nil, err
	}
	curve, err := NewBSplineCurve2dUniformWeights(degree, points2(ctrl), knots)
	return curve, ubar, err
}

// NewFittedBSplineCurveParam is the 3D analogue of
// [NewFittedBSplineCurve2dParam].
func NewFittedBSplineCurveParam(points []math.Point3, param FitParameterization) (BSplineCurve, []float64, error) {
	degree, err := fitDegree(len(points))
	if err != nil {
		return BSplineCurve{}, nil, err
	}
	ubar, err := alphaParams(coords3(points), param.alpha())
	if err != nil {
		return BSplineCurve{}, nil, err
	}
	ctrl, knots, err := fitInterpolationAt(coords3(points), degree, ubar)
	if err != nil {
		return BSplineCurve{}, nil, err
	}
	curve, err := NewBSplineCurveUniformWeights(degree, points3(ctrl), knots)
	return curve, ubar, err
}

// periodicDegree is the closed fit's degree; cubic gives the C2 seam.
const periodicDegree = 3

// NewClosedFittedBSplineCurve2d interpolates a closed C2 loop through the
// points (the curve returns to points[0] after the last point). It solves the
// true periodic collocation system — the control polygon wraps (c_{n+j} =
// c_j) over an unclamped periodic knot vector — so the seam at points[0] is
// exactly curvature-continuous. The returned parameters have length
// len(points)+1; the last one is the closing return to points[0], and the
// curve is only meaningful within [ū₀, ūₙ] (its Domain).
func NewClosedFittedBSplineCurve2d(points []math.Point2, param FitParameterization) (BSplineCurve2d, []float64, error) {
	ctrl, knots, ubar, err := periodicFit(coords2(points), param.alpha())
	if err != nil {
		return BSplineCurve2d{}, nil, err
	}
	curve, err := NewBSplineCurve2dUniformWeights(periodicDegree, points2(ctrl), knots)
	return curve, ubar, err
}

// NewClosedFittedBSplineCurve is the 3D analogue of
// [NewClosedFittedBSplineCurve2d].
func NewClosedFittedBSplineCurve(points []math.Point3, param FitParameterization) (BSplineCurve, []float64, error) {
	ctrl, knots, ubar, err := periodicFit(coords3(points), param.alpha())
	if err != nil {
		return BSplineCurve{}, nil, err
	}
	curve, err := NewBSplineCurveUniformWeights(periodicDegree, points3(ctrl), knots)
	return curve, ubar, err
}

// periodicFit solves the periodic interpolation: n loop stations, n unknown
// control points (indices wrap mod n), collocated at the loop parameters over
// a periodic knot vector. Returns the wrapped control rows (n + degree), the
// knots, and the n+1 station parameters (last = the closing wrap).
func periodicFit(rows [][]float64, alpha float64) (ctrl [][]float64, knots, ubar []float64, err error) {
	n := len(rows)
	if n < 3 {
		return nil, nil, nil, fmt.Errorf("geom: closed fitted B-spline needs >= 3 points, got %d", n)
	}
	loop := append(append([][]float64{}, rows...), rows[0])
	if ubar, err = alphaParams(loop, alpha); err != nil {
		return nil, nil, nil, err
	}
	knots = periodicKnots(ubar)
	a := periodicCollocation(ubar, knots, n)
	b := cloneRows(rows)
	if err := gaussSolve(a, b); err != nil {
		return nil, nil, nil, err
	}
	return append(b, b[:periodicDegree]...), knots, ubar, nil
}

// periodicKnots extends the n+1 loop parameters periodically by degree knots
// on each side: knots[degree+k] = ūₖ, the left/right tails repeating the
// loop's end spans so every basis function sees a full periodic neighborhood.
func periodicKnots(ubar []float64) []float64 {
	n, p := len(ubar)-1, periodicDegree
	span := ubar[n] - ubar[0]
	knots := make([]float64, n+1+2*p)
	for k := 0; k <= n; k++ {
		knots[p+k] = ubar[k]
	}
	for j := 1; j <= p; j++ {
		knots[p-j] = ubar[0] - (span - (ubar[n-j] - ubar[0]))
		knots[p+n+j] = ubar[n] + (ubar[j] - ubar[0])
	}
	return knots
}

// periodicCollocation builds the n×n basis matrix with wrapped control
// indices: A[k][i mod n] += Nᵢ,ₚ(ūₖ).
func periodicCollocation(ubar, knots []float64, n int) [][]float64 {
	p := periodicDegree
	a := make([][]float64, n)
	for k := range a {
		row := make([]float64, n)
		span := findSpan(n+p-1, p, ubar[k], knots)
		basis := basisFuns(span, p, ubar[k], knots)
		for l := 0; l <= p; l++ {
			row[(span-p+l)%n] += basis[l]
		}
		a[k] = row
	}
	return a
}

// alphaParams returns interpolation parameters ū∈[0,1] spaced by chord
// length raised to alpha (centripetal ½, chordal 1, uniform 0). Erroring when
// two consecutive points coincide under alpha>0 semantics is preserved from
// chordParams: a zero total means all points coincide.
func alphaParams(rows [][]float64, alpha float64) ([]float64, error) {
	n := len(rows)
	cum := make([]float64, n)
	for k := 1; k < n; k++ {
		step := 1.0
		if alpha > 0 {
			step = stdmath.Pow(coordDist(rows[k-1], rows[k]), alpha)
		}
		cum[k] = cum[k-1] + step
	}
	total := cum[n-1]
	if total == 0 {
		return nil, fmt.Errorf("geom: fitted B-spline needs distinct points (total chord length 0)")
	}
	u := make([]float64, n)
	for k := range u {
		u[k] = cum[k] / total
	}
	u[n-1] = 1
	return u, nil
}

// fitInterpolationAt is fitInterpolation with caller-supplied parameters: it
// returns the control points and averaged knot vector of the degree-p
// B-spline interpolating the rows at parameters ubar.
func fitInterpolationAt(pts [][]float64, p int, ubar []float64) (ctrl [][]float64, knots []float64, err error) {
	knots = averagedKnots(ubar, p)
	a := collocation(ubar, p, knots)
	b := cloneRows(pts)
	if err := gaussSolve(a, b); err != nil {
		return nil, nil, err
	}
	return b, knots, nil
}
