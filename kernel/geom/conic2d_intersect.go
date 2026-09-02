// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// IntersectConic2d returns the parameters on the PARAMETRIC conic at which it meets the IMPLICIT one.
//
// Substituting P(t) into the quadratic form collapses every conic pair to one scalar equation in one
// angle, which is why this is the only conic×conic routine in the kernel: the caller picks which side
// to take parametrically and the pairing is done. An ellipse gives a trigonometric quartic solved in
// closed form; a hyperbola branch gives a quadratic in e^t, since cosh and sinh are that exponential's
// half-sum and half-difference.
//
// The parameters come back in ascending order, deduplicated at a tangency (where two roots coincide),
// so a caller can walk them as an interval sequence. `infinite` is true when the two conics are the
// same curve, which no finite root set can describe and every caller must refuse rather than treat as
// "no crossings".
//
// Example:
//
//	ts, inf := geom.IntersectConic2d(rim, geom.Conic2dImplicit(face))
//	if inf { return errCoincident }
func IntersectConic2d(p EllipticalParams2d, q Conic2dImplicit) (ts []float64, infinite bool) {
	if p.A == 0 || p.B == 0 {
		return nil, false
	}
	if p.Hyperbolic {
		return hyperbolicConicRoots(p, q)
	}
	return ellipticConicRoots(p, q)
}

// conicSubstitution expresses the implicit form along the parametric conic's own axes, so the
// substitution below can be written in (cos, sin) alone. The six returned coefficients are those of
// q evaluated at Center + x·U + y·V, with x and y in the conic's frame.
func conicSubstitution(p EllipticalParams2d, q Conic2dImplicit) (a, b, c, d, e, f float64) {
	cx, cy := float64(p.Center.X), float64(p.Center.Y)
	ux, uy := float64(p.U.X), float64(p.U.Y)
	vx, vy := float64(p.V.X), float64(p.V.Y)
	a = q.A*ux*ux + q.B*uy*uy + 2*q.C*ux*uy
	b = q.A*vx*vx + q.B*vy*vy + 2*q.C*vx*vy
	c = q.A*ux*vx + q.B*uy*vy + q.C*(ux*vy+uy*vx)
	d = q.A*cx*ux + q.B*cy*uy + q.C*(cx*uy+cy*ux) + q.D*ux + q.E*uy
	e = q.A*cx*vx + q.B*cy*vy + q.C*(cx*vy+cy*vx) + q.D*vx + q.E*vy
	f = q.Value(cx, cy)
	return a, b, c, d, e, f
}

// ellipticConicRoots solves q(P(t)) = 0 for P(t) = C + a·cos(t)·U + b·sin(t)·V.
//
// Writing the substitution in cos and sin and folding sin² = 1 − cos² gives
// A·cos²t + 2B·cos t·sin t + C·cos t + D·sin t + E = 0, which the Weierstrass substitution
// u = tan(t/2) turns into a quartic. u misses t = π exactly, and that root is recovered
// analytically: at t = π the equation reduces to A − C + E, which is the quartic's OWN leading
// coefficient, so t = π is a root exactly when the quartic drops to cubic.
func ellipticConicRoots(p EllipticalParams2d, q Conic2dImplicit) ([]float64, bool) {
	sa, sb, sc, sd, se, sf := conicSubstitution(p, q)
	aa, bb := sa*p.A*p.A, sb*p.B*p.B
	cc := aa - bb            // cos² (sin² folded away)
	cs := 2 * sc * p.A * p.B // 2·cos·sin
	co := 2 * sd * p.A       // cos
	si := 2 * se * p.B       // sin
	k := sf + bb             // constant
	// The folded coefficients are judged against the scale of the terms they were folded FROM, not
	// against each other. For a conic that satisfies the form identically they all CANCEL, so the
	// residue is float noise — and comparing noise with noise makes the test vacuous in exactly the
	// case it exists to catch: a circle against itself left cc and k at 2e-16, which is enormous
	// relative to the other four and so read as a genuine equation with no roots.
	scale := polyScale(aa, bb, cs, co, si, sf)
	if allNearZero(scale, cc, cs, co, si, k) {
		return nil, true // the parametric conic satisfies the implicit form everywhere: the same curve
	}
	return trigQuadraticRoots(cc, cs/2, co, si, k), false
}

// trigQuadraticRoots solves a·cos²t + 2b·cos t·sin t + c·cos t + d·sin t + e = 0 over [0, 2π).
//
// The Weierstrass substitution cos t = (1−u²)/(1+u²), sin t = 2u/(1+u²), multiplied through by
// (1+u²)², gives the quartic whose coefficients are formed below — the same reduction OCCT's
// math_TrigonometricFunctionRoots performs. Its leading coefficient a−c+e is the equation's value at
// t = π, which is the root the substitution cannot reach, so that root is added exactly when the
// leading coefficient vanishes rather than inferred from a near-infinite u.
func trigQuadraticRoots(a, b, c, d, e float64) []float64 {
	k4, k3, k2, k1, k0 := a-c+e, 2*d-4*b, 2*e-2*a, 4*b+2*d, a+c+e
	var ts []float64
	if stdmath.Abs(k4) <= trigLeadingZero*polyScale(k4, k3, k2, k1, k0) {
		ts = append(ts, stdmath.Pi) // u = ∞: the half-turn the substitution omits
	}
	for _, u := range RealQuarticRoots(k0, k1, k2, k3, k4) {
		ts = append(ts, wrapAngle(2*stdmath.Atan(u)))
	}
	return sortedDedupedAngles(ts)
}

// hyperbolicConicRoots solves q(P(t)) = 0 for P(t) = C + a·cosh(t)·U + b·sinh(t)·V.
//
// With w = e^t the hyperbolic functions are (w ± 1/w)/2, so the substitution multiplied by 4w² is a
// QUARTIC in w — the same solver, no separate machinery — and only the positive roots are branch
// parameters, since w = e^t is positive by construction.
func hyperbolicConicRoots(p EllipticalParams2d, q Conic2dImplicit) ([]float64, bool) {
	sa, sb, sc, sd, se, sf := conicSubstitution(p, q)
	ah, bh := sa*p.A*p.A, sb*p.B*p.B
	ch, dh, eh := sc*p.A*p.B, sd*p.A, se*p.B
	// cosh = (w+1/w)/2, sinh = (w−1/w)/2; times 4w²:
	k4 := ah + bh + 2*ch
	k3 := 4 * (dh + eh)
	k2 := 2*ah - 2*bh + 4*sf
	k1 := 4 * (dh - eh)
	k0 := ah + bh - 2*ch
	if allNearZero(polyScale(ah, bh, ch, dh, eh, sf), k4, k3, k2, k1, k0) {
		return nil, true // the branch lies on the implicit conic everywhere
	}
	var ts []float64
	for _, w := range RealQuarticRoots(k0, k1, k2, k3, k4) {
		if w > 0 {
			ts = append(ts, stdmath.Log(w))
		}
	}
	return sortedDedupedAngles(ts), false
}

// allNearZero reports every coefficient indistinguishable from zero at the given SCALE — the
// signature of an equation satisfied identically rather than at isolated parameters. The scale is the
// caller's, never the coefficients' own: these coefficients vanish by cancellation, and a set that
// cancelled carries no scale of its own to be judged against.
func allNearZero(scale float64, vs ...float64) bool {
	for _, v := range vs {
		if stdmath.Abs(v) > trigLeadingZero*scale {
			return false
		}
	}
	return true
}

// polyScale is the largest coefficient magnitude, the scale a polynomial's own coefficients are
// judged against. It never returns zero, so a comparison against it is always meaningful.
func polyScale(vs ...float64) float64 {
	scale := 0.0
	for _, v := range vs {
		scale = stdmath.Max(scale, stdmath.Abs(v))
	}
	if scale == 0 {
		return 1
	}
	return scale
}

// trigLeadingZero is how small a polynomial coefficient must be, RELATIVE to the largest in the same
// polynomial, to count as absent. It compares coefficients with each other and so carries no model
// scale — a conic written in metres and the same conic in millimetres reduce identically.
const trigLeadingZero = 1e-12 // tol:numeric — relative vanishing of a polynomial coefficient

// wrapAngle folds an angle onto [0, 2π), the domain a closed conic's parameter is reported in.
func wrapAngle(t float64) float64 {
	t = stdmath.Mod(t, 2*stdmath.Pi)
	if t < 0 {
		t += 2 * stdmath.Pi
	}
	return t
}

// sortedDedupedAngles returns the parameters in ascending order with coincident ones collapsed: two
// roots meeting is a TANGENCY, one contact rather than two crossings, and a caller walking the list
// as interval boundaries must not see it as an interval of zero width.
func sortedDedupedAngles(ts []float64) []float64 {
	if len(ts) < 2 {
		return ts
	}
	sortFloats(ts)
	out := ts[:1]
	for _, t := range ts[1:] {
		if t-out[len(out)-1] > trigRootWeld {
			out = append(out, t)
		}
	}
	return out
}

// trigRootWeld is how close two roots must be to count as one tangency. It is an ANGLE, so like every
// angular comparison in the kernel it carries no model scale; the arc length it stands for is the
// caller's radius times this.
const trigRootWeld = 1e-9 // tol:angular — coincident roots of one conic equation
