// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// Scalar machinery shared by the curve evaluators (M01-F06, #603): adaptive
// Simpson quadrature for arc length, the Newton/bisection hybrid inverting it,
// and knot-aware interval splitting.

const (
	// lengthRelTol is the relative accuracy target of arc-length integration.
	lengthRelTol = 1e-10
	// lengthMaxDepth bounds the adaptive-Simpson recursion.
	lengthMaxDepth = 24
	// strokeMaxDepth bounds the chordal-subdivision recursion.
	strokeMaxDepth = 24
	// closestSamples is the multistart sampling density of the generic
	// closest-point search (per curve, before Newton refinement).
	closestSamples = 128
)

// adaptiveSimpson integrates f over [a, b] to the given absolute tolerance,
// recursing where the local Simpson estimates disagree (Richardson-corrected).
func adaptiveSimpson(f func(float64) float64, a, b, tol float64) float64 {
	m := (a + b) / 2
	fa, fm, fb := f(a), f(m), f(b)
	whole := simpsonRule(a, b, fa, fm, fb)
	return simpsonRecurse(f, a, b, fa, fm, fb, whole, tol, lengthMaxDepth)
}

// simpsonRule returns Simpson's estimate over [a, b] from the three samples.
func simpsonRule(a, b, fa, fm, fb float64) float64 {
	return (b - a) / 6 * (fa + 4*fm + fb)
}

// simpsonRecurse refines one interval until the half-sums agree within tol.
func simpsonRecurse(f func(float64) float64, a, b, fa, fm, fb, whole, tol float64, depth int) float64 {
	m := (a + b) / 2
	lm, rm := (a+m)/2, (m+b)/2
	flm, frm := f(lm), f(rm)
	left := simpsonRule(a, m, fa, flm, fm)
	right := simpsonRule(m, b, fm, frm, fb)
	if depth <= 0 || stdmath.Abs(left+right-whole) <= 15*tol {
		return left + right + (left+right-whole)/15
	}
	return simpsonRecurse(f, a, m, fa, flm, fm, left, tol/2, depth-1) +
		simpsonRecurse(f, m, b, fm, frm, fb, right, tol/2, depth-1)
}

// integrateSpans integrates f piecewise over [a, b] split at the given interior
// breakpoints (ascending; NURBS knots, where the speed has kinks).
func integrateSpans(f func(float64) float64, a, b float64, breaks []float64) float64 {
	total, prev := 0.0, a
	for _, k := range breaks {
		if k <= prev || k >= b {
			continue
		}
		total += adaptiveSimpson(f, prev, k, lengthRelTol*(k-prev))
		prev = k
	}
	return total + adaptiveSimpson(f, prev, b, lengthRelTol*(b-prev))
}

// interiorKnots returns the distinct knot values strictly inside the domain —
// the natural quadrature breakpoints of a B-spline.
func interiorKnots(knots []float64, degree int) []float64 {
	lo, hi := knots[degree], knots[len(knots)-1-degree]
	var out []float64
	for _, k := range knots[degree+1 : len(knots)-1-degree] {
		if k > lo && k < hi && (len(out) == 0 || k > out[len(out)-1]) {
			out = append(out, k)
		}
	}
	return out
}

// maxInteriorMultiplicity returns the highest repeat count among interior
// knots — what limits a B-spline's continuity order (Cᵖ⁻ᵐᵃˣ).
func maxInteriorMultiplicity(knots []float64, degree int) int {
	maxRun, run := 0, 1
	interior := knots[degree+1 : len(knots)-1-degree]
	for i := 1; i < len(interior); i++ {
		if interior[i] == interior[i-1] {
			run++
		} else {
			run = 1
		}
		if run > maxRun {
			maxRun = run
		}
	}
	if len(interior) == 1 {
		maxRun = 1
	}
	return maxRun
}

// invertLength solves signedLength(t) = target for t within [lo, hi] by a
// bracketed Newton iteration with bisection fallback: signedLength must be
// monotonically increasing (speed ≥ 0) and speed(t) is its derivative.
func invertLength(signedLength func(float64) float64, speed func(float64) float64, target, lo, hi float64) float64 {
	if signedLength(hi) <= target {
		return hi
	}
	if signedLength(lo) >= target {
		return lo
	}
	t := (lo + hi) / 2
	for i := 0; i < 64; i++ {
		miss := signedLength(t) - target
		if stdmath.Abs(miss) <= lengthRelTol*stdmath.Max(1, stdmath.Abs(target)) {
			return t
		}
		lo, hi = shrinkBracket(lo, hi, t, miss)
		t = newtonOrBisect(t, miss, speed(t), lo, hi)
	}
	return t
}

// shrinkBracket keeps [lo, hi] a sign-change bracket for the monotone miss.
func shrinkBracket(lo, hi, t, miss float64) (float64, float64) {
	if miss > 0 {
		return lo, t
	}
	return t, hi
}

// newtonOrBisect takes a Newton step when it stays inside the bracket and is
// well-defined, otherwise bisects.
func newtonOrBisect(t, miss, slope, lo, hi float64) float64 {
	if slope > 0 {
		next := t - miss/slope
		if next > lo && next < hi {
			return next
		}
	}
	return (lo + hi) / 2
}

// orderedInterval returns the interval with lo ≤ hi (length and stroking are
// direction-agnostic).
func orderedInterval(a, b float64) (lo, hi float64) {
	if a > b {
		return b, a
	}
	return a, b
}
