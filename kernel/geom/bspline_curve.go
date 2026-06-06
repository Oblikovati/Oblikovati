// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati/math"

// BSplineCurve is a NURBS curve (contract: BSplineCurve): a rational B-spline of
// the given Degree over a control polygon (Ctrl) with per-point Weights and a
// Knots vector (length len(Ctrl)+Degree+1). It satisfies [Curve3].
type BSplineCurve struct {
	Degree  int
	Ctrl    []math.Point3
	Weights []float64
	Knots   []float64
}

// NewBSplineCurve builds a rational B-spline curve, validating the
// degree/control/weight/knot size relationships and that weights are positive.
// The input slices are copied so the value stays immutable.
func NewBSplineCurve(degree int, ctrl []math.Point3, weights, knots []float64) (BSplineCurve, error) {
	if err := validateBSpline(degree, len(ctrl), len(weights), len(knots)); err != nil {
		return BSplineCurve{}, err
	}
	if err := requirePositiveWeights(weights); err != nil {
		return BSplineCurve{}, err
	}
	return BSplineCurve{
		Degree:  degree,
		Ctrl:    append([]math.Point3(nil), ctrl...),
		Weights: append([]float64(nil), weights...),
		Knots:   append([]float64(nil), knots...),
	}, nil
}

// NewBSplineCurveUniformWeights builds a non-rational B-spline (all weights 1).
func NewBSplineCurveUniformWeights(degree int, ctrl []math.Point3, knots []float64) (BSplineCurve, error) {
	return NewBSplineCurve(degree, ctrl, unitWeights(len(ctrl)), knots)
}

// PointAt returns the curve position at parameter t.
func (c BSplineCurve) PointAt(t float64) math.Point3 {
	span := findSpan(len(c.Ctrl)-1, c.Degree, t, c.Knots)
	n := basisFuns(span, c.Degree, t, c.Knots)
	var h homog
	for k := 0; k <= c.Degree; k++ {
		i := span - c.Degree + k
		h.add(c.Ctrl[i], c.Weights[i]*n[k])
	}
	return h.point()
}

// TangentAt returns the curve derivative dP/dt at parameter t.
func (c BSplineCurve) TangentAt(t float64) math.Vector3 {
	span := findSpan(len(c.Ctrl)-1, c.Degree, t, c.Knots)
	n, dn := basisAndFirstDerivs(span, c.Degree, t, c.Knots)
	var value, deriv homog
	for k := 0; k <= c.Degree; k++ {
		i := span - c.Degree + k
		value.add(c.Ctrl[i], c.Weights[i]*n[k])
		deriv.add(c.Ctrl[i], c.Weights[i]*dn[k])
	}
	return value.deriv(deriv)
}

// Domain returns the parametric range [Knots[Degree], Knots[len−1−Degree]] over
// which the (clamped or open) curve is defined.
func (c BSplineCurve) Domain() (lo, hi float64) {
	return c.Knots[c.Degree], c.Knots[len(c.Knots)-1-c.Degree]
}
