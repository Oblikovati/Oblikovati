// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"sort"
	"testing"
)

// quarticTol is the residual/coordinate tolerance for the synthetic-polynomial cross-checks — an
// independent verification of the Ferrari coefficients per the derivation's "cross-check once
// against an independent source" instruction: here the independent source is direct expansion of
// (t−r1)(t−r2)(t−r3)(t−r4) for known roots, not a borrowed reference implementation.
const quarticTol = 1e-8

// sortedFloats is a small helper so every case can assert against a canonical root ordering.
func sortedFloats(xs []float64) []float64 {
	out := append([]float64(nil), xs...)
	sort.Float64s(out)
	return out
}

// assertRootsMatch fails unless got and want agree elementwise after sorting, within quarticTol.
func assertRootsMatch(t *testing.T, name string, got, want []float64) {
	t.Helper()
	g, w := sortedFloats(got), sortedFloats(want)
	if len(g) != len(w) {
		t.Fatalf("%s: got %d real roots %v, want %d %v", name, len(g), g, len(w), w)
	}
	for i := range g {
		if stdmath.Abs(g[i]-w[i]) > quarticTol {
			t.Fatalf("%s: root[%d] = %.12f, want %.12f (Δ=%.3e)", name, i, g[i], w[i], g[i]-w[i])
		}
	}
}

// TestRealQuarticRoots_FourDistinctRoots cross-checks against (t−1)(t−2)(t−3)(t−4), expanded by
// hand: t⁴−10t³+35t²−50t+24 = 0.
func TestRealQuarticRoots_FourDistinctRoots(t *testing.T) {
	t.Parallel()
	got := RealQuarticRoots(24, -50, 35, -10, 1)
	assertRootsMatch(t, "four distinct", got, []float64{1, 2, 3, 4})
}

// TestRealQuarticRoots_TwoRealTwoComplex cross-checks (t−1)(t+1)(t²+1) = t⁴−1 = 0: real roots
// ±1, the other two (±i) must NOT appear.
func TestRealQuarticRoots_TwoRealTwoComplex(t *testing.T) {
	t.Parallel()
	got := RealQuarticRoots(-1, 0, 0, 0, 1)
	assertRootsMatch(t, "two real two complex", got, []float64{-1, 1})
}

// TestRealQuarticRoots_AllComplex cross-checks t⁴+1=0 (all four roots complex, the classic
// "no real roots" quartic): the physical filter must return nothing.
func TestRealQuarticRoots_AllComplex(t *testing.T) {
	t.Parallel()
	got := RealQuarticRoots(1, 0, 0, 0, 1)
	if len(got) != 0 {
		t.Fatalf("t⁴+1=0: got real roots %v, want none", got)
	}
}

// TestRealQuarticRoots_RepeatedRoot cross-checks (t−2)²(t−5)(t+1) = 0, expanded:
// t⁴−8t³+9t²+22t−20... recompute: (t-2)^2=(t²-4t+4); (t-5)(t+1)=t²-4t-5;
// product = (t²-4t+4)(t²-4t-5). Let u=t²-4t: (u+4)(u-5)=u²-u-20 = (t²-4t)²-(t²-4t)-20
// = t⁴-8t³+16t²-t²+4t-20 = t⁴-8t³+15t²+4t-20.
func TestRealQuarticRoots_RepeatedRoot(t *testing.T) {
	t.Parallel()
	got := RealQuarticRoots(-20, 4, 15, -8, 1)
	assertRootsMatch(t, "repeated root", got, []float64{-1, 2, 5}) // 2 appears once (deduplicated)
}

// TestRealQuarticRoots_Biquadratic exercises the q≈0 fast path directly: t⁴−5t²+4=0 factors as
// (t²−1)(t²−4) = 0, roots ±1, ±2.
func TestRealQuarticRoots_Biquadratic(t *testing.T) {
	t.Parallel()
	got := RealQuarticRoots(4, 0, -5, 0, 1)
	assertRootsMatch(t, "biquadratic", got, []float64{-2, -1, 1, 2})
}

// TestLargestRealRootOfCubic_KnownRoots cross-checks (m−1)(m−2)(m+5) = m³+2m²-13m+10... recompute:
// (m-1)(m-2)=m²-3m+2; ×(m+5) = m³+5m²-3m²-15m+2m+10 = m³+2m²-13m+10. Largest root is 2.
func TestLargestRealRootOfCubic_KnownRoots(t *testing.T) {
	t.Parallel()
	got := largestRealRootOfCubic(1, 2, -13, 10)
	if stdmath.Abs(got-2) > quarticTol {
		t.Fatalf("largest root = %.12f, want 2", got)
	}
}

// TestLargestRealRootOfCubic_ThreeRealRoots exercises the trigonometric (disc<0) branch directly:
// m³−7m+6 = (m−1)(m−2)(m+3), three real roots, largest is 2.
func TestLargestRealRootOfCubic_ThreeRealRoots(t *testing.T) {
	t.Parallel()
	got := largestRealRootOfCubic(1, 0, -7, 6)
	if stdmath.Abs(got-2) > quarticTol {
		t.Fatalf("largest root = %.12f, want 2", got)
	}
}

// TestLargestRealRootOfCubic_OneRealRoot exercises the Cardano radical (disc>0) branch: m³+m+10
// has exactly one real root (the other two are complex); Newton from a bracket confirms it's ≈-1.847.
func TestLargestRealRootOfCubic_OneRealRoot(t *testing.T) {
	t.Parallel()
	got := largestRealRootOfCubic(1, 0, 1, 10)
	f := got*got*got + got + 10
	if stdmath.Abs(f) > quarticTol {
		t.Fatalf("largest root %.12f is not a root: f(root) = %.3e, want ≈0", got, f)
	}
}

// TestRealRootsUpToQuarticDeflatesAVanishingLeadingCoefficient is the regression for the silent NaN
// (ADR-0061 stage 5, third slice). [RealQuarticRoots] divides by c4, and the Weierstrass substitution
// t = tan(u/2) makes c4 the equation's value at the half-turn — zero whenever a root sits there, which
// is not exotic: a rod driven across a ring hits it at every tube station. The division then produced
// ±Inf, every root came back NaN, and a NaN fails every "is this root real" comparison it is put
// through, so the whole station returned EMPTY rather than wrong. realRootsUpToQuartic deflates instead.
func TestRealRootsUpToQuarticDeflatesAVanishingLeadingCoefficient(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name               string
		c0, c1, c2, c3, c4 float64
		want               []float64
	}{
		// 169·t³ − 169·t, the derivative of the rod-across-a-ring station in tan(u/2): roots at the
		// azimuths 0 and ±π/2, with the fourth escaped to the half-turn the substitution cannot reach.
		{"cubic (c4 = 0)", 0, -169, 0, 169, 0, []float64{-1, 0, 1}},
		{"quadratic (c4 = c3 = 0)", -6, 1, 1, 0, 0, []float64{-3, 2}},
		{"linear (only c1, c0)", 3, -2, 0, 0, 0, []float64{1.5}},
		{"constant", 5, 0, 0, 0, 0, nil},
		// The non-degenerate case must be untouched: it is still Ferrari's.
		{"quartic (c4 ≠ 0)", 24, -50, 35, -10, 1, []float64{1, 2, 3, 4}},
	} {
		got := realRootsUpToQuartic(c.c0, c.c1, c.c2, c.c3, c.c4)
		for _, r := range got {
			if stdmath.IsNaN(r) || stdmath.IsInf(r, 0) {
				t.Fatalf("%s: returned %v — a non-finite root is the silent failure this guards", c.name, got)
			}
		}
		assertRootsMatch(t, c.name, got, c.want)
	}
}
