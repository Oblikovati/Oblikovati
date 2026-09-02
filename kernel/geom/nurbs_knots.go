// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// Knot-vector helpers shared by the NURBS refinement and degree-change primitives
// (M36-F01). A knot is treated as a repeat of an existing value when it lies within
// knotEps of it, so floating-point round-off in inserted knots does not spuriously
// raise or hide multiplicity.
const knotEps = 1e-9

// knotMultiplicity returns how many entries of knots equal value (within knotEps).
func knotMultiplicity(knots []float64, value float64) int {
	mult := 0
	for _, k := range knots {
		if stdmath.Abs(k-value) <= knotEps {
			mult++
		}
	}
	return mult
}

// findSpanMult returns the knot span k containing u (as [findSpan]) together with
// the multiplicity s of u in the knot vector. Knot insertion seeds its blend with
// the pair (k, s): a knot may be inserted at most degree−s more times (P&T §5.2).
func findSpanMult(n, p int, u float64, knots []float64) (span, mult int) {
	return findSpan(n, p, u, knots), knotMultiplicity(knots, u)
}

// normalizeKnots returns a copy of knots affinely rescaled so the vector spans
// [0, 1]; it leaves a degenerate (zero-width) vector unchanged. Reparametrizing to
// a common domain is the precondition for making two knot vectors compatible.
func normalizeKnots(knots []float64) []float64 {
	lo, hi := knots[0], knots[len(knots)-1]
	span := hi - lo
	out := make([]float64, len(knots))
	if span <= knotEps {
		copy(out, knots)
		return out
	}
	for i, k := range knots {
		out[i] = (k - lo) / span
	}
	return out
}

// mergedInteriorKnots returns the per-knot extra multiplicity needed to raise a in
// each of its distinct interior knots up to the maximum multiplicity that value has
// across a and b — the difference list that makes a's knots a superset of b's. Both
// vectors must already share a degree and domain. It is the heart of making two
// curves knot-compatible before a tensor-product surface op (network/loft).
func mergedInteriorKnots(p int, a, b []float64) (values []float64, extra []int) {
	for _, v := range distinctInterior(p, a, b) {
		need := max(knotMultiplicity(a, v), knotMultiplicity(b, v)) - knotMultiplicity(a, v)
		if need > 0 {
			values = append(values, v)
			extra = append(extra, need)
		}
	}
	return values, extra
}

// distinctInterior returns the sorted distinct interior knot values appearing in
// either a or b (the clamped end knots are excluded — they are shared by construction).
func distinctInterior(p int, a, b []float64) []float64 {
	lo, hi := a[p], a[len(a)-1-p]
	var seen []float64
	for _, src := range [][]float64{a, b} {
		for _, v := range src {
			if v <= lo+knotEps || v >= hi-knotEps {
				continue
			}
			if !containsKnot(seen, v) {
				seen = append(seen, v)
			}
		}
	}
	sortFloats(seen)
	return seen
}

// containsKnot reports whether value already appears in vs (within knotEps).
func containsKnot(vs []float64, value float64) bool {
	for _, v := range vs {
		if stdmath.Abs(v-value) <= knotEps {
			return true
		}
	}
	return false
}

// sortFloats sorts a small float slice ascending — knot values, and the at-most-four roots of a
// conic equation. Insertion sort because both are tiny and because an explicit total order matters
// more here than asymptotics: the ordering is part of the kernel's determinism contract.
func sortFloats(vs []float64) {
	for i := 1; i < len(vs); i++ {
		for j := i; j > 0 && vs[j] < vs[j-1]; j-- {
			vs[j], vs[j-1] = vs[j-1], vs[j]
		}
	}
}
