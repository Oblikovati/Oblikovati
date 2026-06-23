// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Public knot-refinement on the rational B-spline surface, per parametric direction.
// A surface refines as a tensor product: inserting a U-knot refines every v-column as
// a U-direction curve (all columns share the new U knot vector); inserting a V-knot
// refines every u-row as a V-direction curve. Each method returns a NEW, geometrically
// identical surface (M36-F01).

// InsertKnotU returns a surface identical to s with the knot u inserted r times in the
// U direction (exact; A5.3 column-wise). It errors as [BSplineCurve.InsertKnot] does.
func (s BSplineSurface) InsertKnotU(u float64, r int) (BSplineSurface, error) {
	if err := validateInsert(s.UDegree, s.UKnots, u, r); err != nil {
		return BSplineSurface{}, err
	}
	newU, ctrl, weights := insertSurfaceU(s, u, r)
	return NewBSplineSurface(s.UDegree, s.VDegree, ctrl, weights, newU, s.VKnots)
}

// InsertKnotV is the V-direction analogue of [BSplineSurface.InsertKnotU] (row-wise).
func (s BSplineSurface) InsertKnotV(v float64, r int) (BSplineSurface, error) {
	if err := validateInsert(s.VDegree, s.VKnots, v, r); err != nil {
		return BSplineSurface{}, err
	}
	newV, ctrl, weights := insertSurfaceV(s, v, r)
	return NewBSplineSurface(s.UDegree, s.VDegree, ctrl, weights, s.UKnots, newV)
}

// RefineKnotsU inserts every value in us into the U direction (repeats raise
// multiplicity), returning the exactly-equal refined surface.
func (s BSplineSurface) RefineKnotsU(us []float64) (BSplineSurface, error) {
	out := s
	for _, u := range us {
		next, err := out.InsertKnotU(u, 1)
		if err != nil {
			return BSplineSurface{}, err
		}
		out = next
	}
	return out, nil
}

// RefineKnotsV is the V-direction analogue of [BSplineSurface.RefineKnotsU].
func (s BSplineSurface) RefineKnotsV(vs []float64) (BSplineSurface, error) {
	out := s
	for _, v := range vs {
		next, err := out.InsertKnotV(v, 1)
		if err != nil {
			return BSplineSurface{}, err
		}
		out = next
	}
	return out, nil
}

// insertSurfaceU refines each v-column as a U-direction curve; all columns share the
// resulting U knot vector.
func insertSurfaceU(s BSplineSurface, u float64, r int) (newU []float64, ctrl [][]math.Point3, weights [][]float64) {
	vCount := len(s.Ctrl[0])
	cols := make([][]hpoint4, vCount)
	for j := 0; j < vCount; j++ {
		newU, cols[j] = insertKnotHomog(s.UDegree, s.UKnots, columnToHomog(s, j), u, r)
	}
	ctrl, weights = netFromColumns(cols)
	return newU, ctrl, weights
}

// columnToHomog converts the v-column j of the surface to homogeneous control points
// (the U-direction curve at constant v).
func columnToHomog(s BSplineSurface, j int) []hpoint4 {
	col := make([]hpoint4, len(s.Ctrl))
	for i := range s.Ctrl {
		col[i] = hpoint4FromCurve(s.Ctrl[i][j], s.Weights[i][j])
	}
	return col
}

// insertSurfaceV refines each u-row as a V-direction curve.
func insertSurfaceV(s BSplineSurface, v float64, r int) (newV []float64, ctrl [][]math.Point3, weights [][]float64) {
	uCount := len(s.Ctrl)
	ctrl = make([][]math.Point3, uCount)
	weights = make([][]float64, uCount)
	for i := 0; i < uCount; i++ {
		row := make([]hpoint4, len(s.Ctrl[i]))
		for j := range s.Ctrl[i] {
			row[j] = hpoint4FromCurve(s.Ctrl[i][j], s.Weights[i][j])
		}
		var refined []hpoint4
		newV, refined = insertKnotHomog(s.VDegree, s.VKnots, row, v, r)
		ctrl[i], weights[i] = curveFromHomog(refined)
	}
	return newV, ctrl, weights
}

// netFromColumns assembles a control net from per-v refined U-columns: ctrl[i][j] is
// row i of column j (the transpose back from column-major refinement).
func netFromColumns(cols [][]hpoint4) (ctrl [][]math.Point3, weights [][]float64) {
	rows := len(cols[0])
	ctrl = make([][]math.Point3, rows)
	weights = make([][]float64, rows)
	for i := 0; i < rows; i++ {
		ctrl[i] = make([]math.Point3, len(cols))
		weights[i] = make([]float64, len(cols))
		for j := range cols {
			ctrl[i][j], weights[i][j] = cols[j][i].point3()
		}
	}
	return ctrl, weights
}
