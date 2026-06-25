// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// The 2D analogue of derivatives3.go (M01-F06, #603).

// CurveDerivatives2 returns dP/dt, d²P/dt² and d³P/dt³ at t.
func CurveDerivatives2(c Curve2, t float64) (d1, d2, d3 math.Vector2) {
	switch g := c.(type) {
	case Line2d, LineSegment2d, Polyline2d:
		return c.TangentAt(t), math.Vector2{}, math.Vector2{}
	case Circle2d:
		return circularDers2(math.V2(1, 0), math.V2(0, 1), g.Radius, g.Radius, twoPi*t, twoPi)
	case Arc2d:
		return circularDers2(math.V2(1, 0), math.V2(0, 1), g.Radius, g.Radius, g.angleAt(t), g.SweepAngle)
	case EllipseFull2d:
		return circularDers2(g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, twoPi*t, twoPi)
	case EllipticalArc2d:
		return circularDers2(g.MajorAxis.AsVector(), g.minorAxis(), g.MajorRadius, g.MinorRadius, g.StartAngle+t*g.SweepAngle, g.SweepAngle)
	case BSplineCurve2d:
		ders := g.DersAt(t, 3)
		return ders[1], ders[2], ders[3]
	default:
		return numericDers2(c, t)
	}
}

// circularDers2 differentiates P(t) = a·cos(angle)·major + b·sin(angle)·minor
// with angle = … + t·rate (the 2D twin of circularDers3).
func circularDers2(major, minor math.Vector2, a, b, angle, rate float64) (d1, d2, d3 math.Vector2) {
	cos, sin := cosSin(angle)
	d1 = major.Scale(-a * sin).Add(minor.Scale(b * cos)).Scale(rate)
	d2 = major.Scale(-a * cos).Add(minor.Scale(-b * sin)).Scale(rate * rate)
	d3 = major.Scale(a * sin).Add(minor.Scale(-b * cos)).Scale(rate * rate * rate)
	return d1, d2, d3
}

// numericDers2 estimates the derivatives by central differences — the 2D twin of
// numericDers3. Each order uses its OWN optimal step (stepD1/2/3 = ε^{1/(n+2)}) so
// the higher-order stencils are not swamped by roundoff; a single fixed 1e-5 step
// drove the 3rd-derivative denominator to ~2e-15 and amplified value roundoff into
// 5–50% error (#1323, #1402).
func numericDers2(c Curve2, t float64) (d1, d2, d3 math.Vector2) {
	h1, h2, h3 := stepD1, stepD2, stepD3
	d1 = c.PointAt(t + h1).AsVector().Sub(c.PointAt(t - h1).AsVector()).Scale(1 / (2 * h1))
	pm, p0, pp := c.PointAt(t-h2).AsVector(), c.PointAt(t).AsVector(), c.PointAt(t+h2).AsVector()
	d2 = pp.Add(pm).Sub(p0.Scale(2)).Scale(1 / (h2 * h2))
	q2m, qm := c.PointAt(t-2*h3).AsVector(), c.PointAt(t-h3).AsVector()
	qp, q2p := c.PointAt(t+h3).AsVector(), c.PointAt(t+2*h3).AsVector()
	d3 = q2p.Sub(qp.Scale(2)).Add(qm.Scale(2)).Sub(q2m).Scale(1 / (2 * h3 * h3 * h3))
	return d1, d2, d3
}

// CurveCurvature2 returns the signed curvature at t: κ = (P′×P″)/|P′|³,
// positive when the curve turns left along increasing parameter.
func CurveCurvature2(c Curve2, t float64) float64 {
	d1, d2, _ := CurveDerivatives2(c, t)
	speed := d1.Length()
	if speed == 0 {
		return 0
	}
	return d1.Cross(d2) / (speed * speed * speed)
}
