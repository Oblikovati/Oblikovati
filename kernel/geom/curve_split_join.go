// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Curve split & join (M36 N-sided fill support): the two B-spline curve surgeries an N-sided
// boundary fill needs to map an N-edge opening onto four logical sides — SplitCurve quad-ifies an
// odd opening (split one edge to reach four), JoinCurves merges a group of consecutive edges into one
// compatible side curve. Both are exact (knot-insertion split; degree-elevate + C0 concatenation).

// SplitCurve splits c at parameter t (strictly inside its domain) into two clamped B-spline curves
// that together reproduce c: the first over [lo,t], the second over [t,hi], each renormalized to
// [0,1]. It inserts t to full multiplicity (degree) so the curve separates exactly at t.
//
// Example: a, b, _ := SplitCurve(edge, 0.5) // two halves meeting G(p-1)-continuously at edge(0.5).
func SplitCurve(c BSplineCurve, t float64) (BSplineCurve, BSplineCurve, error) {
	lo, hi := c.Domain()
	if t <= lo || t >= hi {
		return BSplineCurve{}, BSplineCurve{}, fmt.Errorf("geom.SplitCurve: t=%g outside open domain (%g,%g)", t, lo, hi)
	}
	p := c.Degree
	cc := c
	if r := p - knotMultiplicity(c.Knots, t); r > 0 {
		var err error
		if cc, err = c.InsertKnot(t, r); err != nil {
			return BSplineCurve{}, BSplineCurve{}, fmt.Errorf("geom.SplitCurve: %w", err)
		}
	}
	k := firstIndexOf(cc.Knots, t) // start of the p-fold knot block
	left := clampedSegment(p, cc.Ctrl[:k], cc.Weights[:k], cc.Knots[:k], t, true)
	right := clampedSegment(p, cc.Ctrl[k-1:], cc.Weights[k-1:], cc.Knots[k+p:], t, false)
	return renormalize(left), renormalize(right), nil
}

// clampedSegment assembles one half of a split: ctrl/weights are already the segment's, knots are the
// surviving knots on the far side of t; we add the (degree+1)-fold t clamp on the cut end.
func clampedSegment(p int, ctrl []math.Point3, w, farKnots []float64, t float64, leftHalf bool) BSplineCurve {
	clamp := repeatKnot(t, p+1)
	var knots []float64
	if leftHalf {
		knots = append(append([]float64{}, farKnots...), clamp...) // [lo-clamp, interior, t×(p+1)]
	} else {
		knots = append(append([]float64{}, clamp...), farKnots...) // [t×(p+1), interior, hi-clamp]
	}
	bc, _ := NewBSplineCurve(p, append([]math.Point3{}, ctrl...), append([]float64{}, w...), knots)
	return bc
}

// JoinCurves concatenates curves end-to-end into one B-spline (C0 at each join). The curves must be
// ordered so each one's end coincides with the next one's start; they are degree-elevated to a common
// degree first. A single curve is returned unchanged. The result is renormalized to [0,1].
func JoinCurves(curves []BSplineCurve) (BSplineCurve, error) {
	if len(curves) == 0 {
		return BSplineCurve{}, fmt.Errorf("geom.JoinCurves: no curves")
	}
	deg := 0
	for _, c := range curves {
		deg = maxInt(deg, c.Degree)
	}
	acc, err := elevateTo(curves[0], deg)
	if err != nil {
		return BSplineCurve{}, err
	}
	for i := 1; i < len(curves); i++ {
		next, err := elevateTo(curves[i], deg)
		if err != nil {
			return BSplineCurve{}, err
		}
		if acc, err = concatC0(acc, next); err != nil {
			return BSplineCurve{}, fmt.Errorf("geom.JoinCurves: joining curve %d: %w", i, err)
		}
	}
	return renormalize(acc), nil
}

// concatC0 joins b after a (same degree) with a C0 (degree-multiplicity) knot at the seam. b is
// reparameterized to start where a ends; the shared seam control point is dropped from b.
func concatC0(a, b BSplineCurve) (BSplineCurve, error) {
	p := a.Degree
	if !a.Ctrl[len(a.Ctrl)-1].IsEqualTo(b.Ctrl[0], 1e-7) {
		return BSplineCurve{}, fmt.Errorf("geom: curves do not meet end-to-start (%v vs %v)", a.Ctrl[len(a.Ctrl)-1], b.Ctrl[0])
	}
	aHi := a.Knots[len(a.Knots)-1]
	bShift := shiftKnots(b.Knots, aHi)
	knots := append(append([]float64{}, a.Knots[:len(a.Knots)-(p+1)]...), repeatKnot(aHi, p)...)
	knots = append(knots, bShift[p+1:]...)
	ctrl := append(append([]math.Point3{}, a.Ctrl...), b.Ctrl[1:]...)
	w := append(append([]float64{}, a.Weights...), b.Weights[1:]...)
	return NewBSplineCurve(p, ctrl, w, knots)
}

// elevateTo raises c to the target degree (no-op if already there).
func elevateTo(c BSplineCurve, deg int) (BSplineCurve, error) {
	if c.Degree >= deg {
		return c, nil
	}
	return c.ElevateDegree(deg - c.Degree)
}

// renormalize rescales a curve's knot domain to [0,1].
func renormalize(c BSplineCurve) BSplineCurve {
	lo, hi := c.Knots[0], c.Knots[len(c.Knots)-1]
	if hi-lo == 0 || (lo == 0 && hi == 1) {
		return c
	}
	k := make([]float64, len(c.Knots))
	for i, u := range c.Knots {
		k[i] = (u - lo) / (hi - lo)
	}
	bc, _ := NewBSplineCurve(c.Degree, c.Ctrl, c.Weights, k)
	return bc
}

// shiftKnots returns b's knots translated so its first knot becomes start.
func shiftKnots(knots []float64, start float64) []float64 {
	off := start - knots[0]
	out := make([]float64, len(knots))
	for i, u := range knots {
		out[i] = u + off
	}
	return out
}

// firstIndexOf returns the index of the first knot equal to t (-1 if absent).
func firstIndexOf(knots []float64, t float64) int {
	for i, u := range knots {
		if u == t {
			return i
		}
	}
	return -1
}

// repeatKnot returns n copies of u.
func repeatKnot(u float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = u
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
