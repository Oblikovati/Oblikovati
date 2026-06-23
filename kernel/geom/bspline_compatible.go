// SPDX-License-Identifier: GPL-2.0-only

package geom

// Making two curves compatible — a common degree and a common knot vector — so they can
// skin a tensor-product surface (loft between two sections, or one row/column of a curve
// network). Both curves are first reparametrized to [0, 1], then raised to the higher of
// the two degrees and refined to the union of their interior knots. The results are
// geometrically unchanged (reparametrization, degree elevation and knot insertion are all
// shape-preserving), differing only in representation (M36-F01).

// MakeCompatible returns a and b raised to a common degree and knot vector. The inputs are
// reparametrized to the domain [0, 1] first, so curves with different parameter ranges still
// align. The outputs trace the same geometry as the inputs and share Degree and Knots, the
// precondition for [NewBSplineSurface] over a pair (or a grid) of section curves.
func MakeCompatible(a, b BSplineCurve) (BSplineCurve, BSplineCurve, error) {
	a = a.reparametrizedUnit()
	b = b.reparametrizedUnit()
	a, b, err := matchDegree(a, b)
	if err != nil {
		return BSplineCurve{}, BSplineCurve{}, err
	}
	return matchKnots(a, b)
}

// matchDegree elevates whichever curve has the lower degree up to the other's.
func matchDegree(a, b BSplineCurve) (BSplineCurve, BSplineCurve, error) {
	switch {
	case a.Degree < b.Degree:
		ea, err := a.ElevateDegree(b.Degree - a.Degree)
		return ea, b, err
	case b.Degree < a.Degree:
		eb, err := b.ElevateDegree(a.Degree - b.Degree)
		return a, eb, err
	default:
		return a, b, nil
	}
}

// matchKnots refines each curve with the interior knots it is missing relative to the
// other, so both end on the union knot vector (both already share degree and domain).
func matchKnots(a, b BSplineCurve) (BSplineCurve, BSplineCurve, error) {
	va, ea := mergedInteriorKnots(a.Degree, a.Knots, b.Knots)
	vb, eb := mergedInteriorKnots(b.Degree, b.Knots, a.Knots)
	ra, err := a.RefineKnots(expandKnots(va, ea))
	if err != nil {
		return BSplineCurve{}, BSplineCurve{}, err
	}
	rb, err := b.RefineKnots(expandKnots(vb, eb))
	if err != nil {
		return BSplineCurve{}, BSplineCurve{}, err
	}
	return ra, rb, nil
}

// reparametrizedUnit returns c with its knot vector rescaled to [0, 1]; the geometry is
// unchanged (a reparametrization), only the knot values move.
func (c BSplineCurve) reparametrizedUnit() BSplineCurve {
	return BSplineCurve{Degree: c.Degree, Ctrl: c.Ctrl, Weights: c.Weights, Knots: normalizeKnots(c.Knots)}
}

// expandKnots flattens a (values, extra) difference list into a flat slice with each value
// repeated extra[i] times, the form [BSplineCurve.RefineKnots] consumes.
func expandKnots(values []float64, extra []int) []float64 {
	var out []float64
	for i, v := range values {
		for k := 0; k < extra[i]; k++ {
			out = append(out, v)
		}
	}
	return out
}
