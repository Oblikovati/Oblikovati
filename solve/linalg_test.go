// SPDX-License-Identifier: GPL-2.0-only

package solve

import (
	stdmath "math"
	"testing"
)

func maxAbsDiff(a, b []float64) float64 {
	m := 0.0
	for i := range a {
		if d := stdmath.Abs(a[i] - b[i]); d > m {
			m = d
		}
	}
	return m
}

func TestLeastSquaresSolvesSquareSystem(t *testing.T) {
	// A·x = b with a known solution x = (1, −2, 3).
	a := [][]float64{{2, 1, 1}, {1, 3, 2}, {1, 0, 0}}
	want := []float64{1, -2, 3}
	b := make([]float64, 3)
	for i := range a {
		for j := range want {
			b[i] += a[i][j] * want[j]
		}
	}
	got, ok := leastSquares(a, b)
	if !ok {
		t.Fatal("leastSquares failed on a full-rank square system")
	}
	if d := maxAbsDiff(got, want); d > 1e-9 {
		t.Errorf("solution %v, want %v (max diff %g)", got, want, d)
	}
}

func TestLeastSquaresSatisfiesNormalEquations(t *testing.T) {
	// Over-determined (4×2): the QR least-squares solution must satisfy the normal
	// equations Aᵀ(A·x − b) = 0 (the residual is orthogonal to A's columns).
	a := [][]float64{{1, 1}, {1, 2}, {1, 3}, {1, 4}}
	b := []float64{6, 5, 7, 10}
	x, ok := leastSquares(a, b)
	if !ok {
		t.Fatal("leastSquares failed on a full-rank over-determined system")
	}
	resid := make([]float64, len(b)) // A·x − b
	for i := range a {
		s := -b[i]
		for j := range x {
			s += a[i][j] * x[j]
		}
		resid[i] = s
	}
	for col := range x { // each column of A must be orthogonal to the residual
		dot := 0.0
		for i := range a {
			dot += a[i][col] * resid[i]
		}
		if stdmath.Abs(dot) > 1e-9 {
			t.Errorf("normal equation %d violated: Aᵀresid = %g, want 0", col, dot)
		}
	}
}

func TestLeastSquaresRejectsRankDeficient(t *testing.T) {
	// A column of zeros ⇒ rank-deficient ⇒ ok=false (the LM caller adds √λ·I to avoid this).
	a := [][]float64{{1, 0}, {2, 0}}
	if _, ok := leastSquares(a, []float64{1, 2}); ok {
		t.Error("leastSquares accepted a rank-deficient system")
	}
}
