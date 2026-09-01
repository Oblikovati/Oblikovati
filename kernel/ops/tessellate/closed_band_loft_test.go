// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// TestIsFullCircleArc pins the ring/seam discriminator behind the torus-band loft fix: a
// full-sweep Arc3d (|sweep| ≈ 2π) — how the fillet re-weld mints a closed rim — is a RING; a
// partial arc (the tube seam) is not. Regression for the P2 full-torus-fallback bug, where
// bandRingsAndSeam saw only geom.Circle as a ring and missed these Arc3d rims.
func TestIsFullCircleArc(t *testing.T) {
	t.Parallel()
	mk := func(sweep float64) geom.Arc3d {
		a, err := geom.NewArc3d(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 5, 0, math.Abs(sweep))
		if err != nil {
			t.Fatalf("NewArc3d: %v", err)
		}
		if sweep < 0 {
			a.SweepAngle = -a.SweepAngle // reversed rims arrive negated (survivorCurve)
		}
		return a
	}
	cases := []struct {
		name  string
		sweep float64
		want  bool
	}{
		{"full circle +2π", 2 * math.Pi, true},
		{"full circle −2π (reversed rim)", -2 * math.Pi, true},
		{"quarter-tube seam π/2", math.Pi / 2, false},
		{"half arc π", math.Pi, false},
		{"near-2π within angular tol", 2*math.Pi - SeamAngularTol/2, true},
	}
	for _, c := range cases {
		if got := isFullCircleArc(mk(c.sweep)); got != c.want {
			t.Errorf("isFullCircleArc(%s, sweep=%.5f) = %v, want %v", c.name, c.sweep, got, c.want)
		}
	}
}
