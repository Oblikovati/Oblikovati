// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Public knot removal on the rational B-spline types. Removal is approximate and so
// reports how many of the requested removals were actually applied within tol; it is
// the simplification half of a rebuild (insert-dense → remove-redundant). tol is the
// allowed deviation of the homogeneous control points (equal to model space for a
// non-rational curve/surface). See [removeKnotHomog] (M36-F01).

// RemoveKnot returns the curve with the interior knot u removed up to num times,
// stopping as soon as a removal would move the curve more than tol; the second result
// is how many removals were applied (0 when none was within tol). It errors when num<1
// or u is not an interior knot.
//
// Example: simplified, n, _ := c.RemoveKnot(0.5, 1, 1e-6) // drop a redundant knot.
func (c BSplineCurve) RemoveKnot(u float64, num int, tol float64) (BSplineCurve, int, error) {
	r, s, eff, err := interiorKnot(c.Degree, c.Knots, u, num)
	if err != nil {
		return BSplineCurve{}, 0, err
	}
	newU, newPw, removed := removeKnotHomog(c.Degree, c.Knots, curveToHomog(c.Ctrl, c.Weights), u, r, s, eff, rationalRemovalTol(tol, ctrlNorms3(c.Ctrl), c.Weights))
	if removed == 0 {
		return c, 0, nil
	}
	ctrl, weights := curveFromHomog(newPw)
	out, err := NewBSplineCurve(c.Degree, ctrl, weights, newU)
	return out, removed, err
}

// RemoveKnot is the 2D analogue of [BSplineCurve.RemoveKnot].
func (c BSplineCurve2d) RemoveKnot(u float64, num int, tol float64) (BSplineCurve2d, int, error) {
	r, s, eff, err := interiorKnot(c.Degree, c.Knots, u, num)
	if err != nil {
		return BSplineCurve2d{}, 0, err
	}
	newU, newPw, removed := removeKnotHomog(c.Degree, c.Knots, curve2dToHomog(c.Ctrl, c.Weights), u, r, s, eff, rationalRemovalTol(tol, ctrlNorms2(c.Ctrl), c.Weights))
	if removed == 0 {
		return c, 0, nil
	}
	ctrl, weights := curve2dFromHomog(newPw)
	out, err := NewBSplineCurve2d(c.Degree, ctrl, weights, newU)
	return out, removed, err
}

// RemoveKnotU removes the interior U-knot u from the surface up to num times, but only
// while every refined v-column stays within tol; the removal count is the minimum
// achievable across all columns (a knot is only droppable when removable from all).
func (s BSplineSurface) RemoveKnotU(u float64, num int, tol float64) (BSplineSurface, int, error) {
	r, sm, eff, err := interiorKnot(s.UDegree, s.UKnots, u, num)
	if err != nil {
		return BSplineSurface{}, 0, err
	}
	newU, ctrl, weights, removed := removeSurfaceU(s, u, r, sm, eff, tol)
	if removed == 0 {
		return s, 0, nil
	}
	out, err := NewBSplineSurface(s.UDegree, s.VDegree, ctrl, weights, newU, s.VKnots)
	return out, removed, err
}

// RemoveKnotV is the V-direction analogue of [BSplineSurface.RemoveKnotU].
func (s BSplineSurface) RemoveKnotV(v float64, num int, tol float64) (BSplineSurface, int, error) {
	r, sm, eff, err := interiorKnot(s.VDegree, s.VKnots, v, num)
	if err != nil {
		return BSplineSurface{}, 0, err
	}
	newV, ctrl, weights, removed := removeSurfaceV(s, v, r, sm, eff, tol)
	if removed == 0 {
		return s, 0, nil
	}
	out, err := NewBSplineSurface(s.UDegree, s.VDegree, ctrl, weights, s.UKnots, newV)
	return out, removed, err
}

// removeSurfaceU removes the U-knot from each v-column and keeps the largest count
// common to every column, so all columns share one consistent U knot vector. A column
// is re-refined to that common count when it could individually remove more.
func removeSurfaceU(s BSplineSurface, u float64, r, sm, eff int, tol float64) (newU []float64, ctrl [][]math.Point3, weights [][]float64, removed int) {
	vCount := len(s.Ctrl[0])
	removed = eff
	for j := range vCount {
		_, _, got := removeKnotHomog(s.UDegree, s.UKnots, columnToHomog(s, j), u, r, sm, eff, tol)
		removed = min(removed, got)
	}
	if removed == 0 {
		return nil, nil, nil, 0
	}
	cols := make([][]hpoint4, vCount)
	for j := range vCount {
		newU, cols[j], _ = removeKnotHomog(s.UDegree, s.UKnots, columnToHomog(s, j), u, r, sm, removed, tol)
	}
	ctrl, weights = netFromColumns(cols)
	return newU, ctrl, weights, removed
}

// removeSurfaceV is the V-direction analogue of [removeSurfaceU] (row-wise).
func removeSurfaceV(s BSplineSurface, v float64, r, sm, eff int, tol float64) (newV []float64, ctrl [][]math.Point3, weights [][]float64, removed int) {
	uCount := len(s.Ctrl)
	removed = eff
	for i := range uCount {
		row := rowToHomog(s, i)
		_, _, got := removeKnotHomog(s.VDegree, s.VKnots, row, v, r, sm, eff, tol)
		removed = min(removed, got)
	}
	if removed == 0 {
		return nil, nil, nil, 0
	}
	ctrl = make([][]math.Point3, uCount)
	weights = make([][]float64, uCount)
	for i := range uCount {
		var refined []hpoint4
		newV, refined, _ = removeKnotHomog(s.VDegree, s.VKnots, rowToHomog(s, i), v, r, sm, removed, tol)
		ctrl[i], weights[i] = curveFromHomog(refined)
	}
	return newV, ctrl, weights, removed
}

// rowToHomog converts the u-row i of the surface to homogeneous control points.
func rowToHomog(s BSplineSurface, i int) []hpoint4 {
	row := make([]hpoint4, len(s.Ctrl[i]))
	for j := range s.Ctrl[i] {
		row[j] = hpoint4FromCurve(s.Ctrl[i][j], s.Weights[i][j])
	}
	return row
}

// rationalRemovalTol converts a GEOMETRIC (3D) knot-removal tolerance into the homogeneous
// 4-space threshold that bounds it (Piegl & Tiller eq. 5.30): a homogeneous control move of d
// displaces the rational curve by at most d·(1+maxᵢ|Pᵢ|)/minᵢwᵢ, so the homogeneous test uses
// tol·min(w)/(1+max|P|). For a non-rational curve near the origin this reduces to ≈tol, and for
// heavy weights it prevents the silent over-removal the raw 4-space distance allowed (audit
// A15, #1611).
func rationalRemovalTol(tol float64, maxNorm float64, weights []float64) float64 {
	minW := weights[0]
	for _, w := range weights {
		if w < minW {
			minW = w
		}
	}
	return tol * minW / (1 + maxNorm)
}

// ctrlNorms3 is the largest control-point distance from the origin.
func ctrlNorms3(ctrl []math.Point3) float64 {
	m := 0.0
	for _, p := range ctrl {
		if d := stdmath.Sqrt(float64(p.X*p.X + p.Y*p.Y + p.Z*p.Z)); d > m {
			m = d
		}
	}
	return m
}

// ctrlNorms2 is the 2D analogue of ctrlNorms3.
func ctrlNorms2(ctrl []math.Point2) float64 {
	m := 0.0
	for _, p := range ctrl {
		if d := stdmath.Sqrt(float64(p.X*p.X + p.Y*p.Y)); d > m {
			m = d
		}
	}
	return m
}
