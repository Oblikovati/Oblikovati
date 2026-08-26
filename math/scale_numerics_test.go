// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

// Audit A15 (#1611): foundation-layer primitives must be scale-invariant.

// TestInvert3x3ScaleSweep: a perfectly conditioned uniform-scale matrix must invert at every
// scale (the retired absolute |det| test rejected s ≤ 1e-3, whose det is 1e-9), and a genuinely
// rank-deficient matrix must be rejected at every scale.
func TestInvert3x3ScaleSweep(t *testing.T) {
	for _, s := range []float64{1e-6, 1e-3, 1, 1e3, 1e6} {
		m := [9]Scalar{Scalar(s), 0, 0, 0, Scalar(s), 0, 0, 0, Scalar(s)}
		inv, ok := invert3x3(m)
		if !ok {
			t.Errorf("scale %g: conditioned diagonal falsely singular", s)
			continue
		}
		for i := range 3 {
			if got := float64(inv[4*i] * m[4*i]); stdmath.Abs(got-1) > 1e-12 {
				t.Errorf("scale %g: M·M⁻¹ diagonal = %g, want 1", s, got)
			}
		}
		deficient := [9]Scalar{Scalar(s), 0, 0, Scalar(2 * s), 0, 0, 0, 0, Scalar(s)} // rows 0∥1
		if _, ok := invert3x3(deficient); ok {
			t.Errorf("scale %g: rank-deficient matrix inverted", s)
		}
	}
}

// TestAngleToAccurateNearZeroAndPi: pairs with analytic angles 1e-9…1e-5 rad (constructed by
// exact small rotations) must compute to within a tenth of the 1e-9 angular tolerance — the
// retired acos form had a ~1e-8 noise floor at the clamped ±1 ends.
func TestAngleToAccurateNearZeroAndPi(t *testing.T) {
	for _, theta := range []float64{1e-9, 1e-8, 1e-7, 1e-6, 1e-5} {
		u := V3(1, 0, 0)
		v := V3(Scalar(stdmath.Cos(theta)), Scalar(stdmath.Sin(theta)), 0)
		if got := float64(u.AngleTo(v)); stdmath.Abs(got-theta) > 1e-10+1e-6*theta {
			t.Errorf("angle %g: computed %g (err %g)", theta, got, got-theta)
		}
		w := V3(Scalar(-stdmath.Cos(theta)), Scalar(stdmath.Sin(theta)), 0) // π−θ from u
		want := stdmath.Pi - theta
		if got := float64(u.AngleTo(w)); stdmath.Abs(got-want) > 1e-10+1e-6*want {
			t.Errorf("angle near π (%g): computed %g (err %g)", want, got, got-want)
		}
	}
}
