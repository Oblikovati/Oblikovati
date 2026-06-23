// SPDX-License-Identifier: GPL-2.0-only

package geom

// Public knot-refinement on the rational B-spline curve types. Each method returns
// a NEW geometrically-identical curve with a refined knot vector; the receiver is
// unchanged (the BSpline types are value-immutable). Refinement underpins the
// higher-level Class-A operations — rebuild, CV editing, match, extend (M36-F01).

// InsertKnot returns a curve geometrically identical to c with the knot u inserted r
// times (Boehm, A5.1). It is exact: the curve is unchanged, only its control polygon
// is subdivided. It errors when r < 1, u lies outside the open domain, or r would
// raise u's multiplicity past the degree.
//
// Example: refined, _ := c.InsertKnot(0.5, 1) // add a control point at the midspan.
func (c BSplineCurve) InsertKnot(u float64, r int) (BSplineCurve, error) {
	if err := validateInsert(c.Degree, c.Knots, u, r); err != nil {
		return BSplineCurve{}, err
	}
	newU, newPw := insertKnotHomog(c.Degree, c.Knots, curveToHomog(c.Ctrl, c.Weights), u, r)
	ctrl, weights := curveFromHomog(newPw)
	return NewBSplineCurve(c.Degree, ctrl, weights, newU)
}

// RefineKnots inserts every value in us into the curve (a value repeated in us is
// inserted with that multiplicity), returning the exactly-equal refined curve. It is
// the bulk form of [BSplineCurve.InsertKnot] used to make two curves knot-compatible.
func (c BSplineCurve) RefineKnots(us []float64) (BSplineCurve, error) {
	U, pw := append([]float64(nil), c.Knots...), curveToHomog(c.Ctrl, c.Weights)
	for _, u := range us {
		if err := validateInsert(c.Degree, U, u, 1); err != nil {
			return BSplineCurve{}, err
		}
		U, pw = insertKnotHomog(c.Degree, U, pw, u, 1)
	}
	ctrl, weights := curveFromHomog(pw)
	return NewBSplineCurve(c.Degree, ctrl, weights, U)
}

// InsertKnot is the 2D analogue of [BSplineCurve.InsertKnot].
func (c BSplineCurve2d) InsertKnot(u float64, r int) (BSplineCurve2d, error) {
	if err := validateInsert(c.Degree, c.Knots, u, r); err != nil {
		return BSplineCurve2d{}, err
	}
	newU, newPw := insertKnotHomog(c.Degree, c.Knots, curve2dToHomog(c.Ctrl, c.Weights), u, r)
	ctrl, weights := curve2dFromHomog(newPw)
	return NewBSplineCurve2d(c.Degree, ctrl, weights, newU)
}

// RefineKnots is the 2D analogue of [BSplineCurve.RefineKnots].
func (c BSplineCurve2d) RefineKnots(us []float64) (BSplineCurve2d, error) {
	U, pw := append([]float64(nil), c.Knots...), curve2dToHomog(c.Ctrl, c.Weights)
	for _, u := range us {
		if err := validateInsert(c.Degree, U, u, 1); err != nil {
			return BSplineCurve2d{}, err
		}
		U, pw = insertKnotHomog(c.Degree, U, pw, u, 1)
	}
	ctrl, weights := curve2dFromHomog(pw)
	return NewBSplineCurve2d(c.Degree, ctrl, weights, U)
}
