// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// ReverseBSplineCurve returns the geometrically identical curve traversed the other way:
// control points and weights reversed, knots mirrored about the domain (Piegl & Tiller
// curve reversal — exact, no refit). Domain [lo, hi] is preserved with PointAt(lo+hi−t)
// mapping onto the parent's PointAt(t).
func ReverseBSplineCurve(c BSplineCurve) (BSplineCurve, error) {
	n := len(c.Ctrl)
	ctrl := make([]math.Point3, n)
	weights := make([]float64, n)
	for i := 0; i < n; i++ {
		ctrl[i], weights[i] = c.Ctrl[n-1-i], c.Weights[n-1-i]
	}
	m := len(c.Knots)
	lo, hi := c.Knots[0], c.Knots[m-1]
	knots := make([]float64, m)
	for i := 0; i < m; i++ {
		knots[i] = lo + hi - c.Knots[m-1-i]
	}
	return NewBSplineCurve(c.Degree, ctrl, weights, knots)
}
