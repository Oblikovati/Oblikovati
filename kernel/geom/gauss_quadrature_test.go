// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"
)

// An n-point Gauss–Legendre rule integrates polynomials up to degree 2n−1 exactly.
// x⁴ over [−1,1] is 2/5; a 3-point rule (exact to degree 5) must nail it.
func TestGaussLegendreExactForPolynomial(t *testing.T) {
	nodes, weights := GaussLegendre(3)
	var got float64
	for i, x := range nodes {
		got += weights[i] * x * x * x * x
	}
	if want := 2.0 / 5.0; stdmath.Abs(got-want) > 1e-14 {
		t.Fatalf("∫x⁴ dx on [-1,1] = %v, want %v", got, want)
	}
}

// The nodes are symmetric about 0 and the weights sum to the interval length 2.
func TestGaussLegendreWeightsSumAndSymmetry(t *testing.T) {
	for n := 1; n <= 24; n++ {
		nodes, weights := GaussLegendre(n)
		if len(nodes) != n || len(weights) != n {
			t.Fatalf("n=%d: got %d nodes, %d weights", n, len(nodes), len(weights))
		}
		var sum float64
		for i := range weights {
			sum += weights[i]
		}
		if stdmath.Abs(sum-2) > 1e-13 {
			t.Errorf("n=%d: weights sum = %v, want 2", n, sum)
		}
		for i := 0; i < n/2; i++ {
			if stdmath.Abs(nodes[i]+nodes[n-1-i]) > 1e-13 {
				t.Errorf("n=%d: node %d not symmetric: %v vs %v", n, i, nodes[i], nodes[n-1-i])
			}
		}
	}
}

// IntegrateRule maps a rule to an arbitrary interval; ∫₀^π sin = 2, reached by a
// modest-order rule (the integrand is smooth).
func TestIntegrate1DSine(t *testing.T) {
	got := Integrate1D(12, 0, stdmath.Pi, stdmath.Sin)
	if want := 2.0; stdmath.Abs(got-want) > 1e-9 {
		t.Fatalf("∫₀^π sin = %v, want %v", got, want)
	}
}
