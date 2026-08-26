// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Public degree reduction on the rational B-spline types. Reduction is approximate and
// reports ok=false (leaving the input unchanged) when the shape cannot be represented one
// degree lower within tol. It is the inverse of degree elevation: a curve produced by
// [BSplineCurve.ElevateDegree] reduces back exactly. After dropping to the C^0 form the
// redundant join knots are removed within tol to restore the natural knot structure (M36-F01).

// ReduceDegree returns a curve of degree−1 approximating c within tol, with ok=false (and
// c unchanged) when c is not reducible at that tolerance. It errors when c is already
// degree 1 (degree 0 is not a meaningful curve).
//
// Example: quad, ok, _ := cubic.ReduceDegree(1e-6) // back to degree 2 if the shape allows.
func (c BSplineCurve) ReduceDegree(tol float64) (BSplineCurve, bool, error) {
	if c.Degree < 2 {
		return c, false, fmt.Errorf("geom: cannot reduce a degree-%d curve below degree 1", c.Degree)
	}
	newU, newPw, ok := degreeReduceHomog(c.Degree, c.Knots, curveToHomog(c.Ctrl, c.Weights), tol)
	if !ok {
		return c, false, nil
	}
	ctrl, weights := curveFromHomog(newPw)
	reduced, err := NewBSplineCurve(c.Degree-1, ctrl, weights, newU)
	if err != nil {
		return c, false, err
	}
	return cleanCurveKnots(reduced, tol), true, nil
}

// ReduceDegree is the 2D analogue of [BSplineCurve.ReduceDegree].
func (c BSplineCurve2d) ReduceDegree(tol float64) (BSplineCurve2d, bool, error) {
	if c.Degree < 2 {
		return c, false, fmt.Errorf("geom: cannot reduce a degree-%d curve below degree 1", c.Degree)
	}
	newU, newPw, ok := degreeReduceHomog(c.Degree, c.Knots, curve2dToHomog(c.Ctrl, c.Weights), tol)
	if !ok {
		return c, false, nil
	}
	ctrl, weights := curve2dFromHomog(newPw)
	reduced, err := NewBSplineCurve2d(c.Degree-1, ctrl, weights, newU)
	if err != nil {
		return c, false, err
	}
	return clean2dCurveKnots(reduced, tol), true, nil
}

// ReduceDegreeU returns the surface with its U degree lowered by one within tol, ok=false
// when any v-column is not reducible. Every column reduces to the same C^0 knot vector, so
// the redundant knots are then removed across all columns together (RemoveKnotU).
func (s BSplineSurface) ReduceDegreeU(tol float64) (BSplineSurface, bool, error) {
	if s.UDegree < 2 {
		return s, false, fmt.Errorf("geom: cannot reduce a U-degree-%d surface below degree 1", s.UDegree)
	}
	vCount := len(s.Ctrl[0])
	cols := make([][]hpoint4, vCount)
	var newU []float64
	for j := range vCount {
		var ok bool
		newU, cols[j], ok = degreeReduceHomog(s.UDegree, s.UKnots, columnToHomog(s, j), tol)
		if !ok {
			return s, false, nil
		}
	}
	ctrl, weights := netFromColumns(cols)
	reduced, err := NewBSplineSurface(s.UDegree-1, s.VDegree, ctrl, weights, newU, s.VKnots)
	if err != nil {
		return s, false, err
	}
	return cleanSurfaceU(reduced, tol), true, nil
}

// ReduceDegreeV is the V-direction analogue of [BSplineSurface.ReduceDegreeU].
func (s BSplineSurface) ReduceDegreeV(tol float64) (BSplineSurface, bool, error) {
	if s.VDegree < 2 {
		return s, false, fmt.Errorf("geom: cannot reduce a V-degree-%d surface below degree 1", s.VDegree)
	}
	uCount := len(s.Ctrl)
	ctrl := make([][]math.Point3, uCount)
	weights := make([][]float64, uCount)
	var newV []float64
	for i := range uCount {
		var rowPw []hpoint4
		var ok bool
		newV, rowPw, ok = degreeReduceHomog(s.VDegree, s.VKnots, rowToHomog(s, i), tol)
		if !ok {
			return s, false, nil
		}
		ctrl[i], weights[i] = curveFromHomog(rowPw)
	}
	reduced, err := NewBSplineSurface(s.UDegree, s.VDegree-1, ctrl, weights, s.UKnots, newV)
	if err != nil {
		return s, false, err
	}
	return cleanSurfaceV(reduced, tol), true, nil
}

// cleanCurveKnots removes every redundant interior knot left by the C^0 reduction, within
// tol — restoring the highest continuity the reduced shape supports.
func cleanCurveKnots(c BSplineCurve, tol float64) BSplineCurve {
	for _, v := range distinctInterior(c.Degree, c.Knots, c.Knots) {
		if next, n, err := c.RemoveKnot(v, c.Degree, tol); err == nil && n > 0 {
			c = next
		}
	}
	return c
}

// clean2dCurveKnots is the 2D analogue of [cleanCurveKnots].
func clean2dCurveKnots(c BSplineCurve2d, tol float64) BSplineCurve2d {
	for _, v := range distinctInterior(c.Degree, c.Knots, c.Knots) {
		if next, n, err := c.RemoveKnot(v, c.Degree, tol); err == nil && n > 0 {
			c = next
		}
	}
	return c
}

// cleanSurfaceU removes the redundant interior U-knots across all columns together.
func cleanSurfaceU(s BSplineSurface, tol float64) BSplineSurface {
	for _, v := range distinctInterior(s.UDegree, s.UKnots, s.UKnots) {
		if next, n, err := s.RemoveKnotU(v, s.UDegree, tol); err == nil && n > 0 {
			s = next
		}
	}
	return s
}

// cleanSurfaceV is the V-direction analogue of [cleanSurfaceU].
func cleanSurfaceV(s BSplineSurface, tol float64) BSplineSurface {
	for _, v := range distinctInterior(s.VDegree, s.VKnots, s.VKnots) {
		if next, n, err := s.RemoveKnotV(v, s.VDegree, tol); err == nil && n > 0 {
			s = next
		}
	}
	return s
}
