// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// testCubicBSpline is a non-trivial clamped cubic (two interior knots, non-uniform weights) used
// to exercise SubSpanBSplineCurve/ReverseBSplineCurve against a curve that is neither a single
// Bezier segment nor rational-trivial.
func testCubicBSpline(t *testing.T) BSplineCurve {
	t.Helper()
	ctrl := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 3, 0), math.P3(3, 4, 0),
		math.P3(5, 3, 0), math.P3(7, 4, 0), math.P3(9, 0, 0),
	}
	weights := []float64{1, 1, 1.5, 1, 1, 1}
	knots := []float64{0, 0, 0, 0, 0.4, 0.6, 1, 1, 1, 1}
	c, err := NewBSplineCurve(3, ctrl, weights, knots)
	if err != nil {
		t.Fatalf("testCubicBSpline: %v", err)
	}
	return c
}

// TestSubSpanBSplineCurveMatchesParentPointwise is the core correctness proof: the extracted
// sub-curve's own domain must reproduce the PARENT's points at the same parameter, densely
// sampled across an interior span that straddles both interior knots.
func TestSubSpanBSplineCurveMatchesParentPointwise(t *testing.T) {
	t.Parallel()
	parent := testCubicBSpline(t)
	t0, t1 := 0.2, 0.8
	sub, err := SubSpanBSplineCurve(parent, t0, t1)
	if err != nil {
		t.Fatalf("SubSpanBSplineCurve: %v", err)
	}
	lo, hi := sub.Domain()
	if stdmath.Abs(lo-t0) > 1e-12 || stdmath.Abs(hi-t1) > 1e-12 {
		t.Fatalf("sub-curve domain = [%g, %g], want [%g, %g]", lo, hi, t0, t1)
	}
	for i := 0; i <= 50; i++ {
		u := t0 + (t1-t0)*float64(i)/50
		want := parent.PointAt(u)
		got := sub.PointAt(u)
		if d := float64(want.DistanceTo(got)); d > 1e-9 {
			t.Fatalf("sub-curve diverges from parent at t=%g: got %v, want %v (d=%.3g)", u, got, want, d)
		}
	}
}

// TestSubSpanBSplineCurveEndpointsExact checks the sub-curve's endpoints land exactly on the
// parent's own points at t0/t1 — the invariant the rail/rim splicers rely on to weld crack-free.
func TestSubSpanBSplineCurveEndpointsExact(t *testing.T) {
	t.Parallel()
	parent := testCubicBSpline(t)
	t0, t1 := 0.35, 0.9
	sub, err := SubSpanBSplineCurve(parent, t0, t1)
	if err != nil {
		t.Fatalf("SubSpanBSplineCurve: %v", err)
	}
	if d := float64(sub.PointAt(t0).DistanceTo(parent.PointAt(t0))); d > 1e-12 {
		t.Errorf("sub-curve start off parent by %.3g", d)
	}
	if d := float64(sub.PointAt(t1).DistanceTo(parent.PointAt(t1))); d > 1e-12 {
		t.Errorf("sub-curve end off parent by %.3g", d)
	}
}

// TestSubSpanBSplineCurveRejectsBadSpan covers the guard: a non-increasing or out-of-domain span
// must error, never silently clamp to something else.
func TestSubSpanBSplineCurveRejectsBadSpan(t *testing.T) {
	t.Parallel()
	parent := testCubicBSpline(t)
	cases := []struct {
		name   string
		t0, t1 float64
	}{
		{"reversed", 0.8, 0.2},
		{"equal", 0.5, 0.5},
		{"below domain", -0.1, 0.5},
		{"above domain", 0.5, 1.1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := SubSpanBSplineCurve(parent, c.t0, c.t1); err == nil {
				t.Errorf("SubSpanBSplineCurve(%g, %g): want error, got none", c.t0, c.t1)
			}
		})
	}
}

// TestReverseBSplineCurveMatchesParentMirrored proves ReverseBSplineCurve is an EXACT
// re-traversal: PointAt(lo+hi-t) on the parent must equal PointAt(t) on the reversal, densely
// sampled — not merely at the endpoints.
func TestReverseBSplineCurveMatchesParentMirrored(t *testing.T) {
	t.Parallel()
	parent := testCubicBSpline(t)
	rev, err := ReverseBSplineCurve(parent)
	if err != nil {
		t.Fatalf("ReverseBSplineCurve: %v", err)
	}
	lo, hi := parent.Domain()
	for i := 0; i <= 50; i++ {
		u := lo + (hi-lo)*float64(i)/50
		want := parent.PointAt(lo + hi - u)
		got := rev.PointAt(u)
		if d := float64(want.DistanceTo(got)); d > 1e-9 {
			t.Fatalf("reversed curve diverges at t=%g: got %v, want %v (d=%.3g)", u, got, want, d)
		}
	}
}

// TestReverseBSplineCurveIsInvolution checks reversing twice returns (up to the same domain and
// pointwise sampling) the original curve — the property bsplineSubSeg's from/to swap relies on.
func TestReverseBSplineCurveIsInvolution(t *testing.T) {
	t.Parallel()
	parent := testCubicBSpline(t)
	once, err := ReverseBSplineCurve(parent)
	if err != nil {
		t.Fatalf("ReverseBSplineCurve: %v", err)
	}
	twice, err := ReverseBSplineCurve(once)
	if err != nil {
		t.Fatalf("ReverseBSplineCurve (2nd): %v", err)
	}
	lo, hi := parent.Domain()
	for i := 0; i <= 20; i++ {
		u := lo + (hi-lo)*float64(i)/20
		if d := float64(parent.PointAt(u).DistanceTo(twice.PointAt(u))); d > 1e-9 {
			t.Fatalf("double reversal diverges at t=%g by %.3g", u, d)
		}
	}
}
