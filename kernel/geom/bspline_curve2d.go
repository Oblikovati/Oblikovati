// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// BSplineCurve2d is the 2D (sketch-space) NURBS curve (contract: BSplineCurve2d): a
// rational B-spline of the given Degree over a 2D control polygon (Ctrl) with per-point
// Weights and a Knots vector (length len(Ctrl)+Degree+1). It satisfies [Curve2] and is the
// planar analogue of [BSplineCurve].
type BSplineCurve2d struct {
	Degree  int
	Ctrl    []math.Point2
	Weights []float64
	Knots   []float64
}

// NewBSplineCurve2d builds a rational 2D B-spline curve, validating the
// degree/control/weight/knot size relationships and that weights are positive. The input
// slices are copied so the value stays immutable.
func NewBSplineCurve2d(degree int, ctrl []math.Point2, weights, knots []float64) (BSplineCurve2d, error) {
	if err := validateBSpline(degree, len(ctrl), len(weights), len(knots)); err != nil {
		return BSplineCurve2d{}, err
	}
	if err := requirePositiveWeights(weights); err != nil {
		return BSplineCurve2d{}, err
	}
	return BSplineCurve2d{
		Degree:  degree,
		Ctrl:    append([]math.Point2(nil), ctrl...),
		Weights: append([]float64(nil), weights...),
		Knots:   append([]float64(nil), knots...),
	}, nil
}

// NewBSplineCurve2dUniformWeights builds a non-rational 2D B-spline (all weights 1).
func NewBSplineCurve2dUniformWeights(degree int, ctrl []math.Point2, knots []float64) (BSplineCurve2d, error) {
	return NewBSplineCurve2d(degree, ctrl, unitWeights(len(ctrl)), knots)
}

// PointAt returns the curve position at parameter t.
func (c BSplineCurve2d) PointAt(t float64) math.Point2 {
	span := findSpan(len(c.Ctrl)-1, c.Degree, t, c.Knots)
	n := basisFuns(span, c.Degree, t, c.Knots)
	var h homog2
	for k := 0; k <= c.Degree; k++ {
		i := span - c.Degree + k
		h.add(c.Ctrl[i], c.Weights[i]*n[k])
	}
	return h.point()
}

// TangentAt returns the curve derivative dP/dt at parameter t.
func (c BSplineCurve2d) TangentAt(t float64) math.Vector2 {
	span := findSpan(len(c.Ctrl)-1, c.Degree, t, c.Knots)
	n, dn := basisAndFirstDerivs(span, c.Degree, t, c.Knots)
	var value, deriv homog2
	for k := 0; k <= c.Degree; k++ {
		i := span - c.Degree + k
		value.add(c.Ctrl[i], c.Weights[i]*n[k])
		deriv.add(c.Ctrl[i], c.Weights[i]*dn[k])
	}
	return value.deriv(deriv)
}

// Domain returns the parametric range [Knots[Degree], Knots[len−1−Degree]].
func (c BSplineCurve2d) Domain() (lo, hi float64) {
	return c.Knots[c.Degree], c.Knots[len(c.Knots)-1-c.Degree]
}

// homog2 is the 2D analogue of [homog]: a weighted homogeneous accumulator (numerator
// A = Σ wᵢ·Pᵢ, weight w = Σ wᵢ) for evaluating a rational planar B-spline and its tangent.
type homog2 struct {
	a math.Vector2
	w float64
}

// add accumulates control point p with the already-basis-scaled weight.
func (h *homog2) add(p math.Point2, weight float64) {
	h.a = h.a.Add(p.AsVector().Scale(math.Scalar(weight)))
	h.w += weight
}

// point returns the rational position A/w.
func (h homog2) point() math.Point2 {
	return h.a.Scale(math.Scalar(1 / h.w)).AsPoint()
}

// deriv applies the rational quotient rule d(A/w) = (d.a − point·d.w)/w.
func (h homog2) deriv(d homog2) math.Vector2 {
	point := h.point()
	return d.a.Sub(point.AsVector().Scale(math.Scalar(d.w))).Scale(math.Scalar(1 / h.w))
}

var _ Curve2 = BSplineCurve2d{}
