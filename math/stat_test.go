// SPDX-License-Identifier: GPL-2.0-only

package math

import "testing"

func TestMedian(t *testing.T) {
	if m := Median([]float64{3, 1, 2}); m != 2 {
		t.Errorf("median odd = %v, want 2", m)
	}
	if m := Median([]float64{4, 1, 3, 2}); m != 2.5 {
		t.Errorf("median even = %v, want 2.5", m)
	}
	if m := Median(nil); m != 0 {
		t.Errorf("median empty = %v, want 0", m)
	}
}

func TestPercentile(t *testing.T) {
	v := []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if p := Percentile(v, 0); p != 0 {
		t.Errorf("p0 = %v, want 0", p)
	}
	if p := Percentile(append([]float64(nil), v...), 1); p != 10 {
		t.Errorf("p100 = %v, want 10", p)
	}
	if p := Percentile(append([]float64(nil), v...), 0.5); p != 5 {
		t.Errorf("p50 = %v, want 5", p)
	}
	// Linear interpolation between ranks.
	if p := Percentile([]float64{0, 10}, 0.25); p != 2.5 {
		t.Errorf("p25 of {0,10} = %v, want 2.5", p)
	}
	// Out-of-range q is clamped.
	if p := Percentile([]float64{1, 2, 3}, 2); p != 3 {
		t.Errorf("clamp q>1 = %v, want 3", p)
	}
	if p := Percentile([]float64{42}, 0.9); p != 42 {
		t.Errorf("single value = %v, want 42", p)
	}
}
