// SPDX-License-Identifier: GPL-2.0-only

package geom

import "fmt"

// Knot insertion and refinement on homogeneous control points (Boehm's algorithm,
// Piegl & Tiller A5.1 / A5.4). Inserting a knot is an *exact* operation: it adds a
// control point without changing the curve or surface geometry, and is the building
// block of refine, make-compatible, Bézier extraction and (with removal) rebuild.

// insertKnotHomog inserts the knot u into the homogeneous B-spline (degree p, knots
// knots, control points pw) r times, returning the refined knot vector and control
// points. It is Boehm's A5.1 generalized over the seeded span/multiplicity (k, s).
// It panics on r+s > p (more insertions than the degree allows, a caller bug).
func insertKnotHomog(p int, knots []float64, pw []hpoint4, u float64, r int) (newU []float64, newPw []hpoint4) {
	n := len(pw) - 1
	k, s := findSpanMult(n, p, u, knots)
	if r+s > p {
		panic(insertOverflow(u, r, s, p))
	}
	newU = insertedKnots(knots, u, k, r)
	newPw = insertedCtrl(p, knots, pw, u, k, s, r)
	return newU, newPw
}

// insertedKnots builds the knot vector after inserting u (r times) at span k.
func insertedKnots(knots []float64, u float64, k, r int) []float64 {
	out := make([]float64, len(knots)+r)
	copy(out[:k+1], knots[:k+1])
	for i := 1; i <= r; i++ {
		out[k+i] = u
	}
	for i := k + 1; i < len(knots); i++ {
		out[i+r] = knots[i]
	}
	return out
}

// insertedCtrl builds the control points after inserting u (r times) at span k with
// pre-existing multiplicity s, via the affine corner-cutting blend of A5.1.
func insertedCtrl(p int, knots []float64, pw []hpoint4, u float64, k, s, r int) []hpoint4 {
	n := len(pw) - 1
	out := make([]hpoint4, len(pw)+r)
	for i := 0; i <= k-p; i++ {
		out[i] = pw[i]
	}
	for i := k - s; i <= n; i++ {
		out[i+r] = pw[i]
	}
	tmp := make([]hpoint4, p-s+1)
	copy(tmp, pw[k-p:k-s+1])
	insertBlend(p, knots, tmp, out, u, k, s, r)
	return out
}

// insertBlend runs the r corner-cutting passes of A5.1, writing the new control
// points into out and threading the working row tmp between passes.
func insertBlend(p int, knots []float64, tmp, out []hpoint4, u float64, k, s, r int) {
	var L int
	for j := 1; j <= r; j++ {
		L = k - p + j
		for i := 0; i <= p-j-s; i++ {
			alpha := (u - knots[L+i]) / (knots[i+k+1] - knots[L+i])
			tmp[i] = tmp[i].lerp(tmp[i+1], alpha)
		}
		out[L] = tmp[0]
		out[k+r-j-s] = tmp[p-j-s]
	}
	for i := L + 1; i < k-s; i++ {
		out[i] = tmp[i-L]
	}
}

// validateInsert checks an insertion request against the curve-direction invariants:
// r ≥ 1, u strictly inside the clamped domain, and r + u's existing multiplicity ≤ p.
// Both curve and surface (per direction) public refiners share it.
func validateInsert(p int, knots []float64, u float64, r int) error {
	if r < 1 {
		return fmt.Errorf("geom: knot insertion count %d must be >= 1", r)
	}
	lo, hi := knots[p], knots[len(knots)-1-p]
	if u <= lo || u >= hi {
		return fmt.Errorf("geom: knot %g must lie strictly inside the domain (%g, %g)", u, lo, hi)
	}
	if s := knotMultiplicity(knots, u); r+s > p {
		return fmt.Errorf("geom: inserting knot %g %d time(s) would exceed degree %d (current multiplicity %d)", u, r, p, s)
	}
	return nil
}
