// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"
)

func TestKnotMultiplicity(t *testing.T) {
	knots := []float64{0, 0, 0, 0.5, 0.5, 1, 1, 1}
	cases := []struct {
		value float64
		want  int
	}{{0, 3}, {0.5, 2}, {1, 3}, {0.25, 0}}
	for _, c := range cases {
		if got := knotMultiplicity(knots, c.value); got != c.want {
			t.Errorf("knotMultiplicity(%g) = %d, want %d", c.value, got, c.want)
		}
	}
}

func TestFindSpanMult(t *testing.T) {
	knots := []float64{0, 0, 0, 0.5, 1, 1, 1} // degree 2, n=3
	span, mult := findSpanMult(3, 2, 0.5, knots)
	if span != 3 || mult != 1 {
		t.Errorf("findSpanMult(0.5) = (%d,%d), want (3,1)", span, mult)
	}
	_, mult = findSpanMult(3, 2, 0.25, knots)
	if mult != 0 {
		t.Errorf("findSpanMult(0.25) multiplicity = %d, want 0", mult)
	}
}

func TestNormalizeKnots(t *testing.T) {
	got := normalizeKnots([]float64{2, 2, 2, 4, 6, 6, 6})
	want := []float64{0, 0, 0, 0.5, 1, 1, 1}
	for i := range want {
		if stdmath.Abs(got[i]-want[i]) > 1e-12 {
			t.Fatalf("normalizeKnots = %v, want %v", got, want)
		}
	}
}

func TestNormalizeKnotsDegenerate(t *testing.T) {
	in := []float64{3, 3, 3, 3}
	got := normalizeKnots(in)
	for i := range in {
		if got[i] != in[i] {
			t.Fatalf("degenerate normalizeKnots = %v, want unchanged %v", got, in)
		}
	}
}

func TestMergedInteriorKnots(t *testing.T) {
	// a has a single interior knot at 0.5; b has interior knots at 0.5 (double) and
	// 0.25. To match the maximum multiplicity per value, a must gain one more 0.5 and
	// one fresh 0.25.
	p := 3
	a := []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1}
	b := []float64{0, 0, 0, 0, 0.25, 0.5, 0.5, 1, 1, 1, 1}
	values, extra := mergedInteriorKnots(p, a, b)
	if len(values) != 2 {
		t.Fatalf("merged values = %v (extra %v), want 2 entries", values, extra)
	}
	got := map[float64]int{values[0]: extra[0], values[1]: extra[1]}
	if got[0.25] != 1 || got[0.5] != 1 {
		t.Errorf("merged extras = %v, want {0.25:1, 0.5:1}", got)
	}
}
