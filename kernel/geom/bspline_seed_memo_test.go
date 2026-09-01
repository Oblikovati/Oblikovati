// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"sync"
	"testing"

	"oblikovati.org/math"
)

// The seed-lattice memo must change NO inversion verdict — asserted directly, cold against warm,
// rather than inferred from "a cache cannot change answers" (#3490).

// memoTestSurface is a modest bicubic patch with a wave, so inversions have non-trivial basins.
func memoTestSurface(t *testing.T) BSplineSurface {
	t.Helper()
	const n = 6
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range n {
		ctrl[i] = make([]math.Point3, n)
		w[i] = make([]float64, n)
		for j := range n {
			z := 0.4 * float64((i*j)%3)
			ctrl[i][j] = math.P3(math.Scalar(i), math.Scalar(j), math.Scalar(z))
			w[i][j] = 1
		}
	}
	knots := []float64{0, 0, 0, 0, 1, 2, 3, 3, 3, 3}
	s, err := NewBSplineSurface(3, 3, ctrl, w, knots, knots)
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

// TestSeedMemoChangesNoInversion inverts a grid of query points on the SAME geometry twice: once
// through a zero-value copy whose memo slot is nil (the compute-per-call path, byte-for-byte the
// pre-memo behaviour) and once through the constructed surface with its memo warm. Every (u, v)
// must agree exactly — not approximately — because the memo stores the identical lattice the cold
// path computes.
func TestSeedMemoChangesNoInversion(t *testing.T) {
	warm := memoTestSurface(t)
	cold := BSplineSurface{UDegree: warm.UDegree, VDegree: warm.VDegree,
		Ctrl: warm.Ctrl, Weights: warm.Weights, UKnots: warm.UKnots, VKnots: warm.VKnots}
	if cold.seed != nil {
		t.Fatal("the cold copy must have no memo slot for this comparison to mean anything")
	}
	for i := 0; i <= 8; i++ {
		for j := 0; j <= 8; j++ {
			q := math.P3(math.Scalar(0.61*float64(i)), math.Scalar(0.55*float64(j)), 1.3)
			cu, cv := cold.ParamAt(q)
			wu, wv := warm.ParamAt(q)
			if cu != wu || cv != wv {
				t.Fatalf("inversion of (%v) differs: cold (%.17g, %.17g) vs warm (%.17g, %.17g)", q, cu, cv, wu, wv)
			}
		}
	}
}

// TestSeedMemoFillsOnceUnderConcurrency: parallel inversions may race only on who fills the memo,
// never on its content.
func TestSeedMemoFillsOnceUnderConcurrency(t *testing.T) {
	s := memoTestSurface(t)
	var wg sync.WaitGroup
	results := make([][2]float64, 32)
	for k := range results {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			u, v := s.ParamAt(math.P3(1.7, 2.2, 0.9))
			results[k] = [2]float64{u, v}
		}(k)
	}
	wg.Wait()
	for k := 1; k < len(results); k++ {
		if results[k] != results[0] {
			t.Fatalf("concurrent inversion %d returned %v, want %v", k, results[k], results[0])
		}
	}
}
