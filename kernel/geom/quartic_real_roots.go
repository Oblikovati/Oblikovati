// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"math/cmplx"

	"oblikovati.org/math"
)

// Generic real-root extraction for a monic-normalizable quartic and the cubic it depends on
// (Ferrari's method with the resolvent cubic solved in closed form, per the torus-host corner
// derivation — geometry-math-advisor consultation, R4 curved-corner-patch campaign). This file is
// deliberately polynomial-only (no geometry).
//
// It lives in geom rather than beside its first caller because a quartic is what EVERY conic
// question reduces to: the blend engine's line-vs-offset-torus tangency supplies one set of
// coefficients, and a conic sectioned by a conic supplies another through the Weierstrass
// substitution (RealQuarticRoots' second caller, IntersectConic2d). One solver, at the layer both
// reach.
//
// Ferrari's factoring step is carried out in complex128 throughout (not a hand-rolled case split
// on the sign of each intermediate radicand): cmplx.Sqrt already handles every sign branch
// uniformly, which is the standard way to avoid the combinatorial branch-explosion that a
// real-only implementation invites (Numerical Recipes §5.6 for the cubic/quadratic root
// discipline this composes). Real roots are recovered by filtering the 4 complex roots for a
// near-zero imaginary part, then Newton-polished against the ORIGINAL (non-depressed) quartic.

// quarticNewtonPolishSteps is the number of Newton iterations applied to each candidate real root
// against the original quartic after Ferrari's closed-form factoring — cheap, and turns the
// closed form's ~1e-9 residual into ~1e-14 (the derivation's recommended polish step).
const quarticNewtonPolishSteps = 2

// quarticRealImagTol is the DIMENSIONLESS floor (relative to the largest root magnitude found)
// below which a complex root's imaginary part is treated as float noise rather than a genuine
// complex pair — a root of a real-coefficient polynomial is complex only in conjugate pairs, so
// this is a scale-relative "is this actually real" gate, not a length tolerance.
const quarticRealImagTol = 1e-9

// RealQuarticRoots returns the real roots of c4·t⁴+c3·t³+c2·t²+c1·t+c0=0 (c4≠0), via Ferrari's
// method (resolvent cubic, largest real root) in complex128 arithmetic, each candidate then
// Newton-polished against the original real quartic. May return 0–4 roots, deduplicated when two
// candidates coincide to within a relative floor (a genuinely repeated root, e.g. an exact
// tangency).
func RealQuarticRoots(c0, c1, c2, c3, c4 float64) []float64 {
	b3, b2, b1, b0 := c3/c4, c2/c4, c1/c4, c0/c4
	// Depress: t = y − b3/4 kills the cubic term, giving y⁴+p·y²+q·y+s = 0.
	p := b2 - 3*b3*b3/8
	q := b3*b3*b3/8 - b3*b2/2 + b1
	s := -3*b3*b3*b3*b3/256 + b3*b3*b2/16 - b3*b1/4 + b0

	ys := ferrariDepressedRoots(p, q, s)
	shift := complex(b3/4, 0)
	var out []float64
	for _, y := range ys {
		t := real(y) - real(shift)
		if stdmath.Abs(imag(y)) > quarticRealImagTol*stdmath.Max(1, cmplx.Abs(y)) {
			continue // genuinely complex (conjugate-pair) root — not physical
		}
		t = newtonPolishRoot(t, []float64{c0, c1, c2, c3, c4})
		out = appendDedupedRoot(out, t, monicScale(b3, b2, b1, b0))
	}
	return out
}

// appendDedupedRoot appends t to roots unless a near-duplicate (relative to the polynomial's own
// coefficient scale) is already present — Ferrari's factoring can rediscover the same root twice
// at an exact tangency (a genuinely repeated root of the quartic), which callers must see once.
func appendDedupedRoot(roots []float64, t, scale float64) []float64 {
	for _, r := range roots {
		if stdmath.Abs(r-t) < quarticRealImagTol*scale {
			return roots
		}
	}
	return append(roots, t)
}

// monicScale is the scale a monic polynomial's roots are deduplicated against: the sum of its
// remaining coefficients, floored at one so the comparison is meaningful for a small polynomial.
func monicScale(cs ...float64) float64 {
	sum := 0.0
	for _, c := range cs {
		sum += stdmath.Abs(c)
	}
	return stdmath.Max(1, sum)
}

// ferrariDepressedRoots factors the depressed quartic y⁴+p·y²+q·y+s=0 into its 4 (complex) roots.
// q≈0 is the biquadratic special case (solved directly as a quadratic in y²); otherwise it solves
// the resolvent cubic 8m³+8p·m²+(2p²−8s)·m−q² = 0 for its LARGEST real root m (the standard
// numerically-well-conditioned choice — it maximizes p+2m, minimizing cancellation in the two
// quadratic factors below) and factors y⁴+p·y²+q·y+s = (y²−√(2m)·y+(p/2+m+q/(2√(2m)))) ·
// (y²+√(2m)·y+(p/2+m−q/(2√(2m)))).
func ferrariDepressedRoots(p, q, s float64) [4]complex128 {
	if stdmath.Abs(q) < quarticRealImagTol*stdmath.Max(1, stdmath.Abs(p)+stdmath.Abs(s)) {
		z1, z2 := complexQuadraticRoots(complex(1, 0), complex(p, 0), complex(s, 0))
		r1, r2 := cmplx.Sqrt(z1), cmplx.Sqrt(z2)
		return [4]complex128{r1, -r1, r2, -r2}
	}
	m := largestRealRootOfCubic(8, 8*p, 2*p*p-8*s, -q*q)
	sq2m := cmplx.Sqrt(complex(2*m, 0))
	half := complex(p/2+m, 0)
	qTerm := complex(q, 0) / (2 * sq2m)
	z1a, z1b := complexQuadraticRoots(complex(1, 0), -sq2m, half+qTerm)
	z2a, z2b := complexQuadraticRoots(complex(1, 0), sq2m, half-qTerm)
	return [4]complex128{z1a, z1b, z2a, z2b}
}

// complexQuadraticRoots solves a·z²+b·z+c=0 (a,b,c complex, a≠0) via the standard quadratic
// formula — used inside Ferrari's factoring where the coefficients are complex by construction
// even though the original quartic is real.
func complexQuadraticRoots(a, b, c complex128) (complex128, complex128) {
	disc := cmplx.Sqrt(b*b - 4*a*c)
	return (-b + disc) / (2 * a), (-b - disc) / (2 * a)
}

// newtonPolishRoot refines a real root candidate against the ORIGINAL (non-depressed) real
// polynomial, whose coefficients are given in ASCENDING degree — the closed forms below are
// accurate to ~1e-9 relative; two Newton steps against F and F′ remove the accumulated
// depression/factoring error.
func newtonPolishRoot(t float64, coeffs []float64) float64 {
	for range quarticNewtonPolishSteps {
		f, fp := hornerValueAndSlope(coeffs, t)
		if fp == 0 {
			break
		}
		t -= f / fp
	}
	return t
}

// hornerValueAndSlope evaluates the polynomial and its derivative at t in one Horner sweep, which is
// the numerically strongest way to read both (Numerical Recipes §5.3).
func hornerValueAndSlope(coeffs []float64, t float64) (f, slope float64) {
	for i := len(coeffs) - 1; i >= 0; i-- {
		slope = slope*t + f
		f = f*t + coeffs[i]
	}
	return f, slope
}

// realRootsUpToQuartic returns the real roots of c4·t⁴+c3·t³+c2·t²+c1·t+c0 for ANY coefficients,
// deflating to the cubic, the quadratic or the line as the leading ones vanish.
//
// [RealQuarticRoots] needs c4 ≠ 0 — it divides by it — and a vanishing leading coefficient is not an
// exotic input: the Weierstrass substitution t = tan(u/2) drops a degree exactly when the equation has
// a root at the half-turn, which is common enough that trigQuadraticRoots already recovers that root
// by hand. Before this deflation the rest of the solve divided by zero there and every OTHER root of
// that station came back NaN — silently, since a NaN fails every "is this root real" comparison it is
// put through. A torus meeting a rod across its axis is exactly that station (ADR-0061 stage 5).
func realRootsUpToQuartic(c0, c1, c2, c3, c4 float64) []float64 {
	scale := polyScale(c0, c1, c2, c3, c4)
	switch {
	case stdmath.Abs(c4) > trigLeadingZero*scale:
		return RealQuarticRoots(c0, c1, c2, c3, c4)
	case stdmath.Abs(c3) > trigLeadingZero*scale:
		return realCubicRoots(c0, c1, c2, c3)
	case stdmath.Abs(c2) > trigLeadingZero*scale:
		return realQuadraticRoots(c0, c1, c2)
	case stdmath.Abs(c1) > trigLeadingZero*scale:
		return []float64{-c0 / c1}
	}
	return nil // a constant: either no root or every t, and neither is a root SET
}

// realCubicRoots returns every real root of c3·t³+c2·t²+c1·t+c0 = 0 (c3 ≠ 0), depressed to
// n³+p·n+q and solved by the same Cardano/Viète split the resolvent cubic above uses, then
// Newton-polished against the original.
func realCubicRoots(c0, c1, c2, c3 float64) []float64 {
	a, b, c := c2/c3, c1/c3, c0/c3
	p := b - a*a/3
	q := 2*a*a*a/27 - a*b/3 + c
	var out []float64
	for _, n := range depressedCubicRealRoots(p, q) {
		t := newtonPolishRoot(n-a/3, []float64{c0, c1, c2, c3})
		out = appendDedupedRoot(out, t, monicScale(a, b, c))
	}
	return out
}

// realQuadraticRoots returns the real roots of c2·t²+c1·t+c0 = 0 (c2 ≠ 0) by the cancellation-free
// form (q = −(b + sign(b)·√Δ)/2, roots q/a and c/q).
func realQuadraticRoots(c0, c1, c2 float64) []float64 {
	disc := c1*c1 - 4*c2*c0
	if disc < 0 {
		return nil
	}
	q := -0.5 * (c1 + stdmath.Copysign(stdmath.Sqrt(disc), nonZeroSign(c1)))
	if q == 0 {
		return []float64{0} // both roots are the origin
	}
	return appendDedupedRoot([]float64{q / c2}, c0/q, monicScale(c1/c2, c0/c2))
}

// largestRealRootOfCubic returns the largest real root of a·m³+b·m²+c·m+d=0 (a≠0) — a
// real-coefficient cubic always has at least one real root, so this never fails. Depresses to
// n³+p·n+q=0 (m=n−A/3) then solves via Cardano's radical form (one real root) or the
// trigonometric form (three real roots), per the sign of the Cardano discriminant
// (q/2)²+(p/3)³ (Numerical Recipes §5.6).
func largestRealRootOfCubic(a, b, c, d float64) float64 {
	bigA, bigB, bigC := b/a, c/a, d/a
	p := bigB - bigA*bigA/3
	q := 2*bigA*bigA*bigA/27 - bigA*bigB/3 + bigC
	roots := depressedCubicRealRoots(p, q)
	best := roots[0]
	for _, r := range roots[1:] {
		if r > best {
			best = r
		}
	}
	return best - bigA/3
}

// depressedCubicRealRoots returns every real root of n³+p·n+q=0 (always ≥1) via Cardano's
// radical form (disc>0: one real root) or the trigonometric Viète form (disc<0: three real
// roots), disc = (q/2)²+(p/3)³.
func depressedCubicRealRoots(p, q float64) []float64 {
	disc := q*q/4 + p*p*p/27
	switch {
	case disc > 0:
		sq := stdmath.Sqrt(disc)
		return []float64{stdmath.Cbrt(-q/2+sq) + stdmath.Cbrt(-q/2-sq)}
	case disc == 0:
		if p == 0 {
			return []float64{0}
		}
		u := stdmath.Cbrt(-q / 2)
		return []float64{2 * u, -u}
	default: // disc < 0 ⇒ p < 0 (three distinct real roots — Viète's trigonometric substitution)
		radius := stdmath.Sqrt(-p * p * p / 27)
		phi := stdmath.Acos(math.Clamp(-q/(2*radius), -1, 1))
		amp := 2 * stdmath.Sqrt(-p/3)
		roots := make([]float64, 3)
		for k := range roots {
			roots[k] = amp * stdmath.Cos((phi-2*stdmath.Pi*float64(k))/3)
		}
		return roots
	}
}
