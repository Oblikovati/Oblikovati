// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"math/cmplx"

	"oblikovati.org/math"
)

// Generic real-root extraction for a monic-normalizable quartic and the cubic it depends on
// (Ferrari's method with the resolvent cubic solved in closed form, per the torus-host corner
// derivation — geometry-math-advisor consultation, R4 curved-corner-patch campaign). This file is
// deliberately polynomial-only (no geometry): fillet_torus_corner.go supplies the coefficients
// from the line-vs-offset-torus tangency and consumes the real roots.
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

// realQuarticRoots returns the real roots of c4·t⁴+c3·t³+c2·t²+c1·t+c0=0 (c4≠0), via Ferrari's
// method (resolvent cubic, largest real root) in complex128 arithmetic, each candidate then
// Newton-polished against the original real quartic. May return 0–4 roots, deduplicated when two
// candidates coincide to within a relative floor (a genuinely repeated root, e.g. an exact
// tangency).
func realQuarticRoots(c0, c1, c2, c3, c4 float64) []float64 {
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
		t = quarticNewtonPolish(t, c0, c1, c2, c3, c4)
		out = appendDedupedRoot(out, t, b3, b2, b1, b0)
	}
	return out
}

// appendDedupedRoot appends t to roots unless a near-duplicate (relative to the quartic's own
// coefficient scale) is already present — Ferrari's factoring can rediscover the same root twice
// at an exact tangency (a genuinely repeated root of the quartic), which callers must see once.
func appendDedupedRoot(roots []float64, t, b3, b2, b1, b0 float64) []float64 {
	scale := stdmath.Max(1, stdmath.Abs(b3)+stdmath.Abs(b2)+stdmath.Abs(b1)+stdmath.Abs(b0))
	for _, r := range roots {
		if stdmath.Abs(r-t) < quarticRealImagTol*scale {
			return roots
		}
	}
	return append(roots, t)
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

// quarticNewtonPolish refines a real root candidate against the ORIGINAL (non-depressed) real
// quartic — Ferrari's closed form is accurate to ~1e-9 relative; two Newton steps against F and
// F' remove the accumulated depression/factoring error.
func quarticNewtonPolish(t, c0, c1, c2, c3, c4 float64) float64 {
	for range quarticNewtonPolishSteps {
		f := (((c4*t+c3)*t+c2)*t+c1)*t + c0
		fp := ((4*c4*t+3*c3)*t+2*c2)*t + c1
		if fp == 0 {
			break
		}
		t -= f / fp
	}
	return t
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
