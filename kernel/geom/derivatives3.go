// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Closed-form first/second/third parametric derivatives for every 3D curve
// (M01-F06, #603). Analytic curves differentiate their trigonometric forms
// exactly; NURBS uses [BSplineCurve.DersAt]; an unknown [Curve3] implementation
// falls back to central finite differences so the evaluator stays total.

// CurveDerivatives3 returns dP/dt, d²P/dt² and d³P/dt³ at t.
func CurveDerivatives3(c Curve3, t float64) (d1, d2, d3 math.Vector3) {
	switch g := c.(type) {
	case Line, LineSegment, Polyline:
		return c.TangentAt(t), math.Vector3{}, math.Vector3{}
	case Circle:
		return circularDers3(g.RefDir.AsVector(), g.binormal(), g.Radius, g.Radius, twoPi*t, twoPi)
	case Arc3d:
		return circularDers3(g.RefDir.AsVector(), g.binormal(), g.Radius, g.Radius, g.StartAngle+t*g.SweepAngle, g.SweepAngle)
	case EllipseFull:
		return circularDers3(g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, twoPi*t, twoPi)
	case EllipticalArc:
		return circularDers3(g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, g.StartAngle+t*g.SweepAngle, g.SweepAngle)
	case Helix3d:
		return helixDers(g, t)
	case BSplineCurve:
		ders := g.DersAt(t, 3)
		return ders[1], ders[2], ders[3]
	default:
		return numericDers3(c, t)
	}
}

// circularDers3 differentiates P(t) = a·cos(angle)·major + b·sin(angle)·minor
// with angle = … + t·rate: each derivative advances the phase by π/2 in each
// axis term and gains a factor of rate.
func circularDers3(major, minor math.Vector3, a, b, angle, rate float64) (d1, d2, d3 math.Vector3) {
	cos, sin := cosSin(angle)
	d1 = major.Scale(-a * sin).Add(minor.Scale(b * cos)).Scale(rate)
	d2 = major.Scale(-a * cos).Add(minor.Scale(-b * sin)).Scale(rate * rate)
	d3 = major.Scale(a * sin).Add(minor.Scale(-b * cos)).Scale(rate * rate * rate)
	return d1, d2, d3
}

// helixDers differentiates the helix P(t) = O + r(t)·u(θ(t)) + h(t)·axis with
// linear r, θ, h: with s = dθ/dt and ρ = dr/dt constants,
// P″ = 2ρs·u′ − rs²·u and P‴ = −3ρs²·u − rs³·u′.
func helixDers(h Helix3d, t float64) (d1, d2, d3 math.Vector3) {
	cos, sin := cosSin(h.angleAt(t))
	ref, bin := h.RefDir.AsVector(), h.binormal()
	u := ref.Scale(cos).Add(bin.Scale(sin))          // radial unit
	w := ref.Scale(-sin).Add(bin.Scale(cos))         // its angular derivative
	rho, s := h.RadialPerTurn*h.Turns, twoPi*h.Turns // dr/dt, |dθ/dt|
	if h.Clockwise {
		s = -s
	}
	r := h.radiusAt(t)
	d1 = u.Scale(rho).Add(w.Scale(r * s)).Add(h.Axis.AsVector().Scale(h.AxialPerTurn * h.Turns))
	d2 = w.Scale(2 * rho * s).Sub(u.Scale(r * s * s))
	d3 = u.Scale(-3 * rho * s * s).Sub(w.Scale(r * s * s * s))
	return d1, d2, d3
}

// numericDers3 estimates the derivatives by central differences — the fallback
// for [Curve3] implementations without a closed form.
func numericDers3(c Curve3, t float64) (d1, d2, d3 math.Vector3) {
	const h = 1e-5
	p2m, pm := c.PointAt(t-2*h).AsVector(), c.PointAt(t-h).AsVector()
	p0 := c.PointAt(t).AsVector()
	pp, p2p := c.PointAt(t+h).AsVector(), c.PointAt(t+2*h).AsVector()
	d1 = pp.Sub(pm).Scale(1 / (2 * h))
	d2 = pp.Add(pm).Sub(p0.Scale(2)).Scale(1 / (h * h))
	d3 = p2p.Sub(pp.Scale(2)).Add(pm.Scale(2)).Sub(p2m).Scale(1 / (2 * h * h * h))
	return d1, d2, d3
}

// CurveCurvature3 returns the unit principal-normal direction and curvature
// magnitude at t: κ = |P′×P″|/|P′|³, direction the normalized rejection of P″
// from P′. Both are zero where the curve is straight or degenerate.
func CurveCurvature3(c Curve3, t float64) (direction math.Vector3, magnitude float64) {
	d1, d2, _ := CurveDerivatives3(c, t)
	speed := d1.Length()
	if speed == 0 {
		return math.Vector3{}, 0
	}
	cross := d1.Cross(d2)
	magnitude = cross.Length() / (speed * speed * speed)
	if magnitude == 0 {
		return math.Vector3{}, 0
	}
	tangent := d1.Scale(1 / speed)
	normal := d2.Sub(tangent.Scale(d2.Dot(tangent))) // reject P″ from the tangent
	return unitOrZero(normal), magnitude
}
