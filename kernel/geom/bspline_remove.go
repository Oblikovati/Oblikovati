// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

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
	newU, newPw, removed := removeKnotHomog(c.Degree, c.Knots, curveToHomog(c.Ctrl, c.Weights), u, r, s, eff, tol)
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
	newU, newPw, removed := removeKnotHomog(c.Degree, c.Knots, curve2dToHomog(c.Ctrl, c.Weights), u, r, s, eff, tol)
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
	uCount, vCount := len(s.Ctrl), len(s.Ctrl[0])
	removed = eff
	cols := make([][]hpoint4, vCount)
	for j := 0; j < vCount; j++ {
		col := make([]hpoint4, uCount)
		for i := 0; i < uCount; i++ {
			col[i] = hpoint4FromCurve(s.Ctrl[i][j], s.Weights[i][j])
		}
		_, _, got := removeKnotHomog(s.UDegree, s.UKnots, col, u, r, sm, eff, tol)
		removed = min(removed, got)
	}
	if removed == 0 {
		return nil, nil, nil, 0
	}
	for j := 0; j < vCount; j++ {
		col := make([]hpoint4, uCount)
		for i := 0; i < uCount; i++ {
			col[i] = hpoint4FromCurve(s.Ctrl[i][j], s.Weights[i][j])
		}
		newU, cols[j], _ = removeKnotHomog(s.UDegree, s.UKnots, col, u, r, sm, removed, tol)
	}
	ctrl, weights = netFromColumns(cols)
	return newU, ctrl, weights, removed
}

// removeSurfaceV is the V-direction analogue of [removeSurfaceU] (row-wise).
func removeSurfaceV(s BSplineSurface, v float64, r, sm, eff int, tol float64) (newV []float64, ctrl [][]math.Point3, weights [][]float64, removed int) {
	uCount := len(s.Ctrl)
	removed = eff
	for i := 0; i < uCount; i++ {
		row := rowToHomog(s, i)
		_, _, got := removeKnotHomog(s.VDegree, s.VKnots, row, v, r, sm, eff, tol)
		removed = min(removed, got)
	}
	if removed == 0 {
		return nil, nil, nil, 0
	}
	ctrl = make([][]math.Point3, uCount)
	weights = make([][]float64, uCount)
	for i := 0; i < uCount; i++ {
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
