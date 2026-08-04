// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Link 2 of the torus∩cylinder far-runout chain (torus-cyl-springs-feet-derivation.md §Link 2): the runout
// FEET where a host-contact SPRING crosses a capping CYLINDER (O₂,â₂,R₂). A spring circle X(t) = Q +
// a(cos t·ê₁ + sin t·ê₂) substituted into the cylinder implicit F(X)=|X−O₂|²−((X−O₂)·â₂)²−R₂²=0 gives a
// degree-4 trig equation (quadratic in (cos t, sin t) through the (·â₂)² term) → the classical quartic in
// τ=tan(t/2). When the circle axis is PARALLEL to the capping axis (ê₁·â₂=ê₂·â₂=0, i.e. P5's coaxial-with-Z
// capping) the quadratic term vanishes and F collapses to the linear-trig p·cos t + q·sin t = γ — identical
// in form to circlePlaneFoot, reusing its nearerCircleRoot selection. The general skew case is solved on the
// trig form directly (dense-bracketed + safeguarded-Newton to machine ε — the derivation-sanctioned
// bracketed alternative to the τ-quartic, without its τ→∞ singularity at t=π). Both keep the root nearest
// the far vertex; no real root ⇒ the spring clears the capping ⇒ decline (do-no-harm).

// circleCylinderFoot returns the foot where the spring circle crosses the capping cylinder — the nearer root
// to the far vertex. Routes to the linear-trig fast path when the circle axis ∥ the capping axis (P5), else
// the general skew form.
func circleCylinderFoot(c geom.Circle, cyl geom.Cylinder, near math.Point3, res Resolution) (math.Point3, bool) {
	e1, e2 := c.RefDir.AsVector(), c.Normal.Cross(c.RefDir)
	if stdmath.Abs(float64(c.Normal.AsVector().Dot(cyl.AxisDir.AsVector()))) >= 1-sinFloor {
		return circleCylinderFootParallel(c, cyl, e1, e2, near, res) // g=h=0: linear-trig fast path
	}
	return circleCylinderFootSkew(c, cyl, e1, e2, near)
}

// circleCylinderFootParallel solves the degenerate ∥-axes case p·cos t + q·sin t = γ,
// γ = (m² + R₂² − |D|² − a²)/(2a), D = Q − O₂, byte-identical machinery to circlePlaneFoot.
func circleCylinderFootParallel(c geom.Circle, cyl geom.Cylinder, e1, e2 math.Vector3, near math.Point3, res Resolution) (math.Point3, bool) {
	d := cyl.Origin.VectorTo(c.Center) // D = Q − O₂
	a := c.Radius
	p, q := float64(d.Dot(e1)), float64(d.Dot(e2))
	m := float64(d.Dot(cyl.AxisDir.AsVector()))
	gamma := (m*m + cyl.Radius*cyl.Radius - float64(d.LengthSquared()) - a*a) / (2 * a)
	mag := stdmath.Hypot(p, q)
	if mag < res.Weld() || stdmath.Abs(gamma) > mag+res.Weld() {
		return math.Point3{}, false // circle axis ∥ cyl axis but the cylinder clears/contains it: no crossing
	}
	base, off := stdmath.Atan2(q, p), stdmath.Acos(math.Clamp(gamma/mag, -1, 1))
	return nearerCircleRoot(c, base+off, base-off, near), true
}

// circleCylinderFootSkew solves the general degree-4 trig section F(t)=0 on the spring circle and returns the
// point nearest the far vertex; declines when there is no real crossing.
func circleCylinderFootSkew(c geom.Circle, cyl geom.Cylinder, e1, e2 math.Vector3, near math.Point3) (math.Point3, bool) {
	coef := circleCylCoef(c, cyl, e1, e2)
	roots := bracketedTrigRoots(coef)
	return nearestCircleFootT(c, roots, near)
}

// circleCylCoef returns [A₀, B, C, E, G, H] of F(t)=A₀+B·cos t+C·sin t+E·cos²t+G·cos t·sin t+H·sin²t for the
// spring circle (Q,a,ê₁,ê₂) ∩ capping cylinder, per the derivation: A₀=|D|²+a²−m²−R₂², B=2a(p−m g),
// C=2a(q−m h), E=−a²g², G=−2a²g h, H=−a²h², with p=D·ê₁, q=D·ê₂, m=D·â₂, g=ê₁·â₂, h=ê₂·â₂, D=Q−O₂.
func circleCylCoef(c geom.Circle, cyl geom.Cylinder, e1, e2 math.Vector3) [6]float64 {
	d := cyl.Origin.VectorTo(c.Center)
	a2 := cyl.AxisDir.AsVector()
	a := c.Radius
	p, q := float64(d.Dot(e1)), float64(d.Dot(e2))
	m := float64(d.Dot(a2))
	g, h := float64(e1.Dot(a2)), float64(e2.Dot(a2))
	a0 := float64(d.LengthSquared()) + a*a - m*m - cyl.Radius*cyl.Radius
	return [6]float64{a0, 2 * a * (p - m*g), 2 * a * (q - m*h), -a * a * g * g, -2 * a * a * g * h, -a * a * h * h}
}

// fCircleCyl evaluates F(t) and F′(t) for the packed coefficients [A₀,B,C,E,G,H].
func fCircleCyl(coef [6]float64, t float64) (f, fp float64) {
	ct, st := stdmath.Cos(t), stdmath.Sin(t)
	f = coef[0] + coef[1]*ct + coef[2]*st + coef[3]*ct*ct + coef[4]*ct*st + coef[5]*st*st
	fp = -coef[1]*st + coef[2]*ct - 2*coef[3]*ct*st + coef[4]*(ct*ct-st*st) + 2*coef[5]*st*ct
	return f, fp
}

// bracketedTrigRoots returns every root of F(t)=0 in (−π,π]: it scans a fixed grid and safeguarded-Newton
// polishes each sign change to machine ε. F has ≤4 roots per period; a near-double (grazing) root has no
// sign change and is intentionally NOT emitted — a tangent spring is a grazing degeneracy the caller
// declines by yielding no crossing rather than fabricating a foot.
func bracketedTrigRoots(coef [6]float64) []float64 {
	const n = 1440
	roots := make([]float64, 0, 4)
	t0 := -stdmath.Pi
	f0, _ := fCircleCyl(coef, t0)
	for i := 1; i <= n; i++ {
		t1 := -stdmath.Pi + 2*stdmath.Pi*float64(i)/float64(n)
		f1, _ := fCircleCyl(coef, t1)
		if f0*f1 < 0 {
			roots = append(roots, safeguardedTrigRoot(coef, t0, t1))
		}
		t0, f0 = t1, f1
	}
	return roots
}

// safeguardedTrigRoot converges the root bracketed in [lo,hi] (opposite F signs at the ends) to machine ε by
// Newton steps that fall back to bisection whenever a step would leave the bracket (rtsafe).
func safeguardedTrigRoot(coef [6]float64, lo, hi float64) float64 {
	flo, _ := fCircleCyl(coef, lo)
	t := 0.5 * (lo + hi)
	for iter := 0; iter < 80; iter++ {
		f, fp := fCircleCyl(coef, t)
		if (flo < 0) == (f < 0) {
			lo = t
		} else {
			hi = t
		}
		if fp != 0 {
			if nt := t - f/fp; nt > lo && nt < hi {
				t = nt
			} else {
				t = 0.5 * (lo + hi)
			}
		} else {
			t = 0.5 * (lo + hi)
		}
		if hi-lo < 2e-16*stdmath.Max(1, stdmath.Abs(t)) {
			break
		}
	}
	return t
}

// nearestCircleFootT maps each root t to the spring point X(t) and keeps the one nearest the far vertex;
// declines when there is no root (the spring clears the capping).
func nearestCircleFootT(c geom.Circle, ts []float64, near math.Point3) (math.Point3, bool) {
	if len(ts) == 0 {
		return math.Point3{}, false
	}
	best, bestDist := math.Point3{}, stdmath.Inf(1)
	for _, t := range ts {
		pt := c.PointAt(t / (2 * stdmath.Pi))
		if dd := float64(pt.DistanceTo(near)); dd < bestDist {
			best, bestDist = pt, dd
		}
	}
	return best, true
}
