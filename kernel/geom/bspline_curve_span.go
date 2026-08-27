// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Exact B-spline curve sub-span extraction (Piegl & Tiller splitting: knot insertion to
// full multiplicity at both cut parameters, then the control run between them). Needed by
// the B-spline-host rim rebuild, whose re-aimed wall seam is a SPAN of the wall's own
// curved seam edge — a chord there puts the seam off the host surface (the J2/J4
// rimhost-carry lesson: a chorded meridian tiled the host face at +317%).

// SubSpanBSplineCurve returns the exact sub-curve of c on [t0, t1] (t0 < t1, inside the
// domain). The result is geometrically identical to c restricted to [t0, t1]: only the
// control net is subdivided, never refit.
//
//	sub, err := geom.SubSpanBSplineCurve(seam, 0.2, 0.8)
func SubSpanBSplineCurve(c BSplineCurve, t0, t1 float64) (BSplineCurve, error) {
	lo, hi := c.Domain()
	// NaN-REJECTING form, deliberate: every comparison with NaN is false, so `!(a > b)` fires on a
	// NaN operand while `a <= b` stays silent and lets it through. Do not "simplify" (sonar go:S1940).
	if !(t0 < t1) || t0 < lo-knotEps || t1 > hi+knotEps {
		return BSplineCurve{}, fmt.Errorf("SubSpanBSplineCurve: span [%g, %g] not increasing inside domain [%g, %g]", t0, t1, lo, hi)
	}
	full, err := raiseToFullMult(c, math.Clamp(t0, lo, hi))
	if err != nil {
		return BSplineCurve{}, err
	}
	full, err = raiseToFullMult(full, math.Clamp(t1, lo, hi))
	if err != nil {
		return BSplineCurve{}, err
	}
	return extractSpan(full, math.Clamp(t0, lo, hi), math.Clamp(t1, lo, hi))
}

// raiseToFullMult inserts t until its multiplicity is the degree; a domain end (already at
// degree+1 by clamping) and an existing full-multiplicity knot pass through unchanged.
func raiseToFullMult(c BSplineCurve, t float64) (BSplineCurve, error) {
	if isDomainEnd(c, t) {
		return c, nil
	}
	need := c.Degree - snappedMultiplicity(c.Knots, t)
	if need <= 0 {
		return c, nil
	}
	return c.InsertKnot(t, need)
}

// isDomainEnd reports t at (knot-eps) one of the clamped domain ends.
func isDomainEnd(c BSplineCurve, t float64) bool {
	lo, hi := c.Domain()
	return stdmath.Abs(t-lo) <= knotEps || stdmath.Abs(t-hi) <= knotEps
}

// snappedMultiplicity counts knots equal to t within knotEps (InsertKnot indexes by exact
// value, so the caller must not insert at a float-noise duplicate of an existing knot).
func snappedMultiplicity(knots []float64, t float64) int {
	m := 0
	for _, k := range knots {
		if stdmath.Abs(k-t) <= knotEps {
			m++
		}
	}
	return m
}

// extractSpan reads the sub-curve's control run and rebuilds its clamped knot vector once
// both cuts are at full multiplicity: controls P[a..b] with a = firstKnotIndex(t0)−1 (the
// on-curve point at a full-multiplicity knot), b = firstKnotIndex(t1)−1, and knots
// [t0×(p+1), interior between the cuts, t1×(p+1)].
func extractSpan(c BSplineCurve, t0, t1 float64) (BSplineCurve, error) {
	p := c.Degree
	a, b := spanControlIndex(c, t0), spanControlIndex(c, t1)
	if a < 0 || b < 0 || b < a {
		return BSplineCurve{}, fmt.Errorf("extractSpan: cut controls not found for [%g, %g]", t0, t1)
	}
	knots := spanKnots(c, t0, t1, p)
	ctrl := append([]math.Point3(nil), c.Ctrl[a:b+1]...)
	weights := append([]float64(nil), c.Weights[a:b+1]...)
	return NewBSplineCurve(p, ctrl, weights, knots)
}

// spanControlIndex is the control index the curve passes through at a full-multiplicity
// cut t: firstKnotIndex(t) − 1, or the clamped-end control for a domain end.
func spanControlIndex(c BSplineCurve, t float64) int {
	lo, hi := c.Domain()
	if stdmath.Abs(t-lo) <= knotEps {
		return 0
	}
	if stdmath.Abs(t-hi) <= knotEps {
		return len(c.Ctrl) - 1
	}
	for i, k := range c.Knots {
		if stdmath.Abs(k-t) <= knotEps {
			return i - 1
		}
	}
	return -1
}

// spanKnots assembles the sub-curve's clamped knot vector: p+1 copies of each cut with the
// parent's interior knots strictly between them.
func spanKnots(c BSplineCurve, t0, t1 float64, p int) []float64 {
	knots := make([]float64, 0, 2*(p+1)+len(c.Knots))
	for i := 0; i <= p; i++ {
		knots = append(knots, t0)
	}
	for _, k := range c.Knots {
		if k > t0+knotEps && k < t1-knotEps {
			knots = append(knots, k)
		}
	}
	for i := 0; i <= p; i++ {
		knots = append(knots, t1)
	}
	return knots
}
