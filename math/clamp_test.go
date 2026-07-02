// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func TestClampFloat64(t *testing.T) {
	cases := []struct {
		name            string
		v, lo, hi, want float64
	}{
		{"below", -2, -1, 1, -1},
		{"above", 2, -1, 1, 1},
		{"inside", 0.5, -1, 1, 0.5},
		{"at-lo", -1, -1, 1, -1},
		{"at-hi", 1, -1, 1, 1},
		{"degenerate-range", 3, 2, 2, 2},
	}
	for _, c := range cases {
		if got := Clamp(c.v, c.lo, c.hi); got != c.want {
			t.Errorf("%s: Clamp(%v, %v, %v) = %v, want %v", c.name, c.v, c.lo, c.hi, got, c.want)
		}
	}
}

func TestClampInt(t *testing.T) {
	cases := []struct {
		name            string
		v, lo, hi, want int
	}{
		{"below", -5, 0, 9, 0},
		{"above", 12, 0, 9, 9},
		{"inside", 4, 0, 9, 4},
	}
	for _, c := range cases {
		if got := Clamp(c.v, c.lo, c.hi); got != c.want {
			t.Errorf("%s: Clamp(%d, %d, %d) = %d, want %d", c.name, c.v, c.lo, c.hi, got, c.want)
		}
	}
}

// A NaN v fails both ordered comparisons, so it passes through — the documented
// comparison-based contract.
func TestClampNaNPassesThrough(t *testing.T) {
	if got := Clamp(stdmath.NaN(), 0.0, 1.0); !stdmath.IsNaN(got) {
		t.Errorf("Clamp(NaN, 0, 1) = %v, want NaN", got)
	}
}

func TestClamp01(t *testing.T) {
	cases := []struct{ v, want float64 }{
		{-0.5, 0}, {0, 0}, {0.25, 0.25}, {1, 1}, {1.5, 1},
	}
	for _, c := range cases {
		if got := Clamp01(c.v); got != c.want {
			t.Errorf("Clamp01(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}
