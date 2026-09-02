// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Two conics in a plane, intersected exactly (Oblikovati/Oblikovati#3459, #3503).
//
// The pairing is by REPRESENTATION, not by kind: one conic is taken IMPLICITLY, as the quadratic
// form it satisfies, and the other PARAMETRICALLY, as a point moving on it. Substituting the second
// into the first collapses every pair — circle×ellipse, ellipse×hyperbola, line×anything — to one
// scalar equation in one angle, so there is a single solver instead of an N² table of type pairs.
// That is the bucketing the kernel rules ask for, and it is how OpenCASCADE's IntAna2d does it
// (IntAna2d_Conic carries the implicit coefficients; each IntAna2d_AnaIntersection::Perform
// substitutes its parametric partner and hands the result to one trigonometric root finder).

// Conic2dImplicit is a plane conic as the quadratic form it satisfies:
//
//	A·x² + B·y² + 2C·x·y + 2D·x + 2E·y + F = 0
//
// The doubled cross and linear coefficients are OCCT's convention (IntAna2d_Conic::Coefficients) and
// are kept because every substitution below is written against it; halving them here would put a
// factor of two into each one instead.
type Conic2dImplicit struct{ A, B, C, D, E, F float64 }

// Value evaluates the quadratic form. It is zero exactly on the conic, and its sign says which side
// of the conic a point lies on — the same one-sided test a half-plane gives for a line.
//
// Example: inside := c.Value(p.X, p.Y) < 0 // for an ellipse written with a negative interior
func (c Conic2dImplicit) Value(x, y float64) float64 {
	return c.A*x*x + c.B*y*y + 2*c.C*x*y + 2*c.D*x + 2*c.E*y + c.F
}

// ImplicitConic2dOf returns the quadratic form of a plane conic given by its centre, its two
// conjugate half-axes and its kind. It is the one place a conic's geometry becomes coefficients, so a
// new conic kind is added here rather than in every intersector.
//
// The axes are the conic's own frame: for an ellipse P(t) = C + a·cos(t)·U + b·sin(t)·V, and for a
// hyperbola branch P(t) = C + a·cosh(t)·U + b·sinh(t)·V. In that frame the form is
// (x/a)² ± (y/b)² − 1, and it is then rotated and translated back to the plane's own axes.
func ImplicitConic2dOf(center math.Point2, u, v math.Vector2, a, b float64, hyperbolic bool) (Conic2dImplicit, bool) {
	if a == 0 || b == 0 {
		return Conic2dImplicit{}, false
	}
	sy := 1.0
	if hyperbolic {
		sy = -1
	}
	// In the conic's frame: q(x', y') = (x'/a)² + sy·(y'/b)² − 1, with (x', y') = R·(p − centre).
	ux, uy := float64(u.X), float64(u.Y)
	vx, vy := float64(v.X), float64(v.Y)
	ia, ib := 1/(a*a), sy/(b*b)
	f := Conic2dImplicit{
		A: ia*ux*ux + ib*vx*vx,
		B: ia*uy*uy + ib*vy*vy,
		C: ia*ux*uy + ib*vx*vy,
	}
	cx, cy := float64(center.X), float64(center.Y)
	f.D = -(f.A*cx + f.C*cy)
	f.E = -(f.B*cy + f.C*cx)
	f.F = f.A*cx*cx + f.B*cy*cy + 2*f.C*cx*cy - 1
	return f, true
}

// EllipticalParams2d is a plane conic taken PARAMETRICALLY: P(t) = Center + a·cos(t)·U + b·sin(t)·V
// for an ellipse or circle (a == b), and P(t) = Center + a·cosh(t)·U + b·sinh(t)·V for one branch
// of a hyperbola.
type EllipticalParams2d struct {
	Center     math.Point2
	U, V       math.Vector2
	A, B       float64
	Hyperbolic bool
}

// PointAt evaluates the parametric conic at t — the angle for an ellipse, the hyperbolic angle for a
// hyperbola branch.
func (e EllipticalParams2d) PointAt(t float64) math.Point2 {
	c, s := stdmath.Cos(t), stdmath.Sin(t)
	if e.Hyperbolic {
		c, s = stdmath.Cosh(t), stdmath.Sinh(t)
	}
	return e.Center.TranslateBy(e.U.Scale(math.Scalar(e.A * c)).Add(e.V.Scale(math.Scalar(e.B * s))))
}

// PlaneConicParams puts a CURVE that lies in a plane into the parametric conic form of that plane's
// own (u, v) chart, with the parameter interval the curve covers in its own parameter.
//
// The type switch over curve kinds lives here, in geom, and not at the call sites: adding a conic kind
// then means teaching one function, rather than finding every switch that must learn about it and
// leaving the ones nobody finds to take their default branch silently (#2188).
//
// The curve is assumed to LIE in the plane — it is a face's own boundary edge — so its centre and axes
// project directly and the parameter carries over with no refitting. ok=false for a curve with no conic
// form here, such as a b-spline edge, which a caller declines on rather than approximating.
//
// Example:
//
//	p, lo, hi, ok := geom.PlaneConicParams(edge.Curve(), facePlane)
func PlaneConicParams(c Curve3, pl Plane) (params EllipticalParams2d, lo, hi float64, ok bool) {
	switch x := c.(type) {
	case Circle:
		return planeConicOf(x.Center, x.RefDir.AsVector(), x.Normal.Cross(x.RefDir), x.Radius, x.Radius, pl, 0, twoPi, false)
	case Arc3d:
		l, h := sweepInterval(x.StartAngle, x.SweepAngle)
		return planeConicOf(x.Center, x.RefDir.AsVector(), x.Normal.Cross(x.RefDir), x.Radius, x.Radius, pl, l, h, false)
	case EllipseFull:
		return planeConicOf(x.Center, x.MajorAxis.AsVector(), x.Normal.Cross(x.MajorAxis), x.MajorRadius, x.MinorRadius, pl, 0, twoPi, false)
	case EllipticalArc:
		l, h := sweepInterval(x.StartAngle, x.SweepAngle)
		return planeConicOf(x.Center, x.MajorAxis.AsVector(), x.Normal.Cross(x.MajorAxis), x.MajorRadius, x.MinorRadius, pl, l, h, false)
	case HyperbolicArc:
		l, h := ascending(x.Theta0, x.Theta1)
		return planeConicOf(x.Center, x.TransverseAxis.AsVector(), x.ConjugateAxis.AsVector(), x.A, x.B, pl, l, h, true)
	}
	return EllipticalParams2d{}, 0, 0, false
}

// IsStraightCurve reports a curve that is a line — the representation bucket a conic substitution does
// not apply to, asked here so no caller has to switch on curve kinds itself.
func IsStraightCurve(c Curve3) bool {
	switch c.(type) {
	case LineSegment, Line:
		return true
	}
	return false
}

// IsPlaneConicCurve reports a curve that PlaneConicParams can put in parametric conic form — the
// question "is this edge a conic in its face's plane", asked without needing the plane. It is here
// beside PlaneConicParams so the two cannot answer differently as kinds are added.
func IsPlaneConicCurve(c Curve3) bool {
	switch c.(type) {
	case Circle, Arc3d, EllipseFull, EllipticalArc, HyperbolicArc:
		return true
	}
	return false
}

// planeConicOf projects a conic's centre and its two axis directions into the plane. The axes stay unit
// there because the curve lies IN the plane, so the projection acts on them as a rotation.
func planeConicOf(center math.Point3, u, v math.Vector3, a, b float64, pl Plane, lo, hi float64, hyperbolic bool) (EllipticalParams2d, float64, float64, bool) {
	if a == 0 || b == 0 {
		return EllipticalParams2d{}, 0, 0, false
	}
	return EllipticalParams2d{
		Center: planePoint2(pl, center), U: planeVec2(pl, u), V: planeVec2(pl, v),
		A: a, B: b, Hyperbolic: hyperbolic,
	}, lo, hi, true
}

// planePoint2 is a point in the plane's own (u, v) coordinates.
func planePoint2(pl Plane, p math.Point3) math.Point2 {
	d := pl.Origin.VectorTo(p)
	return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
}

// planeVec2 is a direction in the plane's own (u, v) coordinates.
func planeVec2(pl Plane, v math.Vector3) math.Vector2 {
	return math.V2(v.Dot(pl.UAxis.AsVector()), v.Dot(pl.VAxis.AsVector()))
}

// sweepInterval turns a start angle and a signed sweep into an ascending interval, so an arc and its
// reversed twin cover the same parameters.
func sweepInterval(start, sweep float64) (lo, hi float64) {
	if sweep < 0 {
		return start + sweep, start
	}
	return start, start + sweep
}

// ascending returns an interval given either way round, low end first.
func ascending(a, b float64) (lo, hi float64) {
	if b < a {
		return b, a
	}
	return a, b
}
