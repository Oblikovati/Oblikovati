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

// ImplicitLine2dOf returns the quadratic form of a LINE — degenerate as a conic, but the same
// currency, which is what lets a line and a conic meet through the one substitution below rather
// than through a special case.
func ImplicitLine2dOf(p math.Point2, dir math.Vector2) (Conic2dImplicit, bool) {
	n := math.V2(-dir.Y, dir.X)
	length := float64(n.Length())
	if length == 0 {
		return Conic2dImplicit{}, false
	}
	nx, ny := float64(n.X)/length, float64(n.Y)/length
	return Conic2dImplicit{D: nx / 2, E: ny / 2, F: -(nx*float64(p.X) + ny*float64(p.Y))}, true
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
