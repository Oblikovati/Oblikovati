// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati/math"
)

// homog accumulates a weighted sum of control points in homogeneous form: the
// numerator vector A = Σ wᵢ·Pᵢ and the weight total w = Σ wᵢ. A rational
// B-spline point is A/w; its derivative uses the quotient rule via [homog.deriv].
type homog struct {
	a math.Vector3
	w float64
}

// add accumulates control point p with the already-basis-scaled weight.
func (h *homog) add(p math.Point3, weight float64) {
	h.a = h.a.Add(p.AsVector().Scale(weight))
	h.w += weight
}

// point returns the rational position A/w.
func (h homog) point() math.Point3 {
	return h.a.Scale(1 / h.w).AsPoint()
}

// deriv applies the rational quotient rule: given the value accumulator (this)
// and a derivative accumulator d (built from basis-function derivatives), it
// returns d(A/w) = (d.a − point·d.w)/w.
func (h homog) deriv(d homog) math.Vector3 {
	point := h.point()
	return d.a.Sub(point.AsVector().Scale(d.w)).Scale(1 / h.w)
}

// validateBSpline checks the size relationships every B-spline (curve direction)
// must satisfy: degree ≥ 1, at least degree+1 control points, one weight per
// control point, and exactly ctrl+degree+1 knots.
func validateBSpline(degree, ctrlCount, weightCount, knotCount int) error {
	if degree < 1 {
		return fmt.Errorf("geom: B-spline degree %d must be >= 1", degree)
	}
	if ctrlCount < degree+1 {
		return fmt.Errorf("geom: B-spline needs >= degree+1 (%d) control points, got %d", degree+1, ctrlCount)
	}
	if weightCount != ctrlCount {
		return fmt.Errorf("geom: B-spline has %d control points but %d weights", ctrlCount, weightCount)
	}
	if want := ctrlCount + degree + 1; knotCount != want {
		return fmt.Errorf("geom: B-spline with %d control points of degree %d needs %d knots, got %d", ctrlCount, degree, want, knotCount)
	}
	return nil
}

// requirePositiveWeights rejects non-positive weights, which would make the
// rational denominator vanish or flip sign.
func requirePositiveWeights(weights []float64) error {
	for i, w := range weights {
		if w <= 0 {
			return fmt.Errorf("geom: B-spline weight[%d] = %g must be > 0", i, w)
		}
	}
	return nil
}

// unitWeights returns a slice of n ones, for non-rational (uniform-weight) use.
func unitWeights(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 1
	}
	return w
}
