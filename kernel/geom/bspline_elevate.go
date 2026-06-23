// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Public degree elevation on the rational B-spline types. Elevation is exact: it raises
// the degree by t and adds control points while preserving the geometry. It is how two
// curves are brought to a common degree before a tensor-product surface op, and how a
// surface reaches the degree a G2/G3 blend needs (M36-F01).

// ElevateDegree returns a curve of degree+t geometrically identical to c (A5.9).
// It errors when t < 1.
//
// Example: cubic, _ := quad.ElevateDegree(1) // degree 2 → 3, same curve.
func (c BSplineCurve) ElevateDegree(t int) (BSplineCurve, error) {
	if t < 1 {
		return BSplineCurve{}, fmt.Errorf("geom: degree elevation amount %d must be >= 1", t)
	}
	newU, newPw := elevateDegreeHomog(c.Degree, t, c.Knots, curveToHomog(c.Ctrl, c.Weights))
	ctrl, weights := curveFromHomog(newPw)
	return NewBSplineCurve(c.Degree+t, ctrl, weights, newU)
}

// ElevateDegree is the 2D analogue of [BSplineCurve.ElevateDegree].
func (c BSplineCurve2d) ElevateDegree(t int) (BSplineCurve2d, error) {
	if t < 1 {
		return BSplineCurve2d{}, fmt.Errorf("geom: degree elevation amount %d must be >= 1", t)
	}
	newU, newPw := elevateDegreeHomog(c.Degree, t, c.Knots, curve2dToHomog(c.Ctrl, c.Weights))
	ctrl, weights := curve2dFromHomog(newPw)
	return NewBSplineCurve2d(c.Degree+t, ctrl, weights, newU)
}

// ElevateDegreeU returns the surface with its U degree raised by t, geometry preserved
// (each v-column is elevated as a U-direction curve, all sharing the new U knot vector).
func (s BSplineSurface) ElevateDegreeU(t int) (BSplineSurface, error) {
	if t < 1 {
		return BSplineSurface{}, fmt.Errorf("geom: degree elevation amount %d must be >= 1", t)
	}
	vCount := len(s.Ctrl[0])
	cols := make([][]hpoint4, vCount)
	var newU []float64
	for j := 0; j < vCount; j++ {
		newU, cols[j] = elevateDegreeHomog(s.UDegree, t, s.UKnots, columnToHomog(s, j))
	}
	ctrl, weights := netFromColumns(cols)
	return NewBSplineSurface(s.UDegree+t, s.VDegree, ctrl, weights, newU, s.VKnots)
}

// ElevateDegreeV is the V-direction analogue of [BSplineSurface.ElevateDegreeU] (row-wise).
func (s BSplineSurface) ElevateDegreeV(t int) (BSplineSurface, error) {
	if t < 1 {
		return BSplineSurface{}, fmt.Errorf("geom: degree elevation amount %d must be >= 1", t)
	}
	uCount := len(s.Ctrl)
	ctrl := make([][]math.Point3, uCount)
	weights := make([][]float64, uCount)
	var newV []float64
	for i := 0; i < uCount; i++ {
		var elevated []hpoint4
		newV, elevated = elevateDegreeHomog(s.VDegree, t, s.VKnots, rowToHomog(s, i))
		ctrl[i], weights[i] = curveFromHomog(elevated)
	}
	return NewBSplineSurface(s.UDegree, s.VDegree+t, ctrl, weights, s.UKnots, newV)
}
