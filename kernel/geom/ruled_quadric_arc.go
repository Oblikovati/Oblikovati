// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The PARAMETRIC side of the intersector's representation split (see quadric.go). A surface with
// STRAIGHT RULINGS — a cylinder, a cone, a plane — is affine in its second parameter:
//
//	P(u, v) = A(u) + v·D(u)
//
// Substituting that into a quadric's F(X) = W·(M W) + 2 G·W + K (W = X − Anchor) leaves a QUADRATIC in
// the ruling parameter v, whose coefficients are exact functions of u:
//
//	a(u)·v² + b(u)·v + c(u) = 0,  a = D·(M D),  b = 2(W₀·(M D) + G·D),  c = W₀·(M W₀) + 2 G·W₀ + K
//
// with W₀ = A(u) − Anchor. So the whole ruled∩quadric family — cylinder∩cylinder, cone∩cylinder,
// cone∩cone, ruled∩sphere — has ONE exact closed form v(u) = the quadratic root, evaluated on the base
// surface. That is what [RuledQuadricArc] stores: not an approximation of the section, the section.
//
// This replaces nothing at the topology level: it is the closed-form fast path INSIDE
// [IntersectSurfacesAnalytic], gated on conditioning (see intersectRuledQuadric), and every pair the
// gate refuses falls to the same general marcher as before (#3489).

// RuledQuadricArc is a bounded run of the EXACT intersection of a straight-ruled surface with an
// implicit quadric, evaluated on the ruled surface's own chart so the point is exactly on it and on
// the quadric to the quadratic solve's rounding. The parameter t∈[0,1] maps to the base azimuth
// u = U0 + t·(U1−U0), and the point is Base.PointAt(u, v) with v the Upper (larger) or lower root of
// the ruling quadratic — the sibling of [SpiricArc] (torus∩plane) for the ruled∩quadric family.
//
// The two roots are ordered by VALUE, which is a continuous, well-defined branch label exactly while
// the discriminant stays positive — the condition intersectRuledQuadric gates on before building one.
//
// Example — the two saddle loops where a rod crosses a wall, on the rod's chart:
//
//	arcs, ok := geom.IntersectSurfacesAnalytic(rodCylinder, wallCylinder, res)
type RuledQuadricArc struct {
	Base   Surface // the straight-ruled surface the arc is evaluated on
	Quad   Quadric // the implicit form of the other surface
	Upper  bool    // which ordered root of the ruling quadratic this branch follows
	U0, U1 float64 // azimuth range; t∈[0,1] maps to u = U0 + t·(U1−U0)
}

// Domain returns [0, 1].
func (a RuledQuadricArc) Domain() (lo, hi float64) { return 0, 1 }

// uAt maps the curve parameter to the base azimuth.
func (a RuledQuadricArc) uAt(t float64) float64 { return a.U0 + t*(a.U1-a.U0) }

// PointAt returns the point at t∈[0,1], evaluated on the base surface at (u, v(u)).
func (a RuledQuadricArc) PointAt(t float64) math.Point3 {
	u := a.uAt(t)
	return a.Base.PointAt(u, a.vAt(u))
}

// TangentAt returns dP/dt = (U1−U0)·(∂P/∂u + dv/du·∂P/∂v), with dv/du from implicit differentiation
// of the ruling quadratic: dv/du = −(a′v² + b′v + c′)/(2av + b). The denominator is ±√(b²−4ac), which
// the conditioning gate keeps away from zero over the arc's whole range.
func (a RuledQuadricArc) TangentAt(t float64) math.Vector3 {
	u := a.uAt(t)
	co := a.Quad.alongRuling(straightRulingAt(a.Base, u))
	v := co.root(a.Upper)
	du, dv := a.Base.DerivativesAt(u, v)
	return du.Add(dv.Scale(math.Scalar(co.dvdu(v)))).Scale(math.Scalar(a.U1 - a.U0))
}

// vAt returns the ruling parameter of this branch at azimuth u.
func (a RuledQuadricArc) vAt(u float64) float64 {
	return a.Quad.alongRuling(straightRulingAt(a.Base, u)).root(a.Upper)
}

// straightRuling is a ruled surface's ruling at one azimuth: the ruling line P = Origin + v·Dir, plus
// both terms' u-derivatives, which the coefficient derivatives a′, b′, c′ need.
type straightRuling struct {
	Origin        math.Point3
	Dir           math.Vector3
	DOrigin, DDir math.Vector3 // ∂Origin/∂u, ∂Dir/∂u
	// SecondDiffScale is |P(u,2) − 2·P(u,1) + P(u,0)|: exactly zero when the surface is affine in v,
	// and the v-curvature otherwise. It is the certificate that puts a surface in the ruled bucket.
	SecondDiffScale float64
}

// straightRulingAt reads a surface's ruling at azimuth u from the Surface interface alone — no type
// switch: a surface affine in v satisfies P(u,v) = P(u,0) + v·(P(u,1) − P(u,0)) identically, and
// ∂P/∂u(u,v) = ∂P/∂u(u,0) + v·(∂P/∂u(u,1) − ∂P/∂u(u,0)). It also measures the second difference
// P(u,2) − 2P(u,1) + P(u,0), which is exactly zero for a straight-ruled surface and grows with the
// v-curvature otherwise — the certificate that puts the surface in the ruled bucket or keeps it out.
func straightRulingAt(s Surface, u float64) straightRuling {
	p0, p1, p2 := s.PointAt(u, 0), s.PointAt(u, 1), s.PointAt(u, 2)
	d0, _ := s.DerivativesAt(u, 0)
	d1, _ := s.DerivativesAt(u, 1)
	second := p0.AsVector().Add(p2.AsVector()).Sub(p1.AsVector().Scale(2))
	return straightRuling{
		Origin: p0, Dir: p0.VectorTo(p1), DOrigin: d0, DDir: d1.Sub(d0),
		SecondDiffScale: float64(second.Length()),
	}
}

// ruledQuadricCoeffs are the ruling quadratic a·v² + b·v + c and its u-derivatives at one azimuth.
type ruledQuadricCoeffs struct{ a, b, c, da, db, dc float64 }

// alongRuling substitutes a ruling into the quadric, giving the quadratic in the ruling parameter and
// its u-derivatives (see the file comment for the algebra).
func (q Quadric) alongRuling(r straightRuling) ruledQuadricCoeffs {
	w := q.Anchor.VectorTo(r.Origin)
	mw, md, mdd := q.M.Apply(w), q.M.Apply(r.Dir), q.M.Apply(r.DDir)
	dot := func(x, y math.Vector3) float64 { return float64(x.Dot(y)) }
	return ruledQuadricCoeffs{
		a:  dot(r.Dir, md),
		b:  2 * (dot(w, md) + dot(q.G, r.Dir)),
		c:  dot(w, mw) + 2*dot(q.G, w) + q.K,
		da: 2 * dot(r.DDir, md),
		db: 2 * (dot(r.DOrigin, md) + dot(w, mdd) + dot(q.G, r.DDir)),
		dc: 2 * (dot(r.DOrigin, mw) + dot(q.G, r.DOrigin)),
	}
}

// discriminant returns b² − 4ac — positive exactly where the ruling crosses the quadric twice.
func (c ruledQuadricCoeffs) discriminant() float64 { return c.b*c.b - 4*c.a*c.c }

// root returns the upper or lower root of a·v² + b·v + c = 0, computed by the cancellation-free form
// (q = −(b + sign(b)·√Δ)/2, roots q/a and c/q) and then ORDERED by value, so "upper" names the same
// branch at every azimuth. It returns NaN where the ruling misses the quadric or runs along it — a
// configuration the conditioning gate refuses before an arc is ever built.
func (c ruledQuadricCoeffs) root(upper bool) float64 {
	disc := c.discriminant()
	if disc <= 0 || c.a == 0 {
		return stdmath.NaN()
	}
	q := -0.5 * (c.b + stdmath.Copysign(stdmath.Sqrt(disc), nonZeroSign(c.b)))
	lo, hi := q/c.a, c.c/q
	if lo > hi {
		lo, hi = hi, lo
	}
	if upper {
		return hi
	}
	return lo
}

// separation returns |v₊ − v₋| = √Δ/|a|, the two branches' gap along the ruling — the length the
// conditioning gate reads to decide the branches stay apart across the whole azimuth sweep.
func (c ruledQuadricCoeffs) separation() float64 {
	disc := c.discriminant()
	if disc <= 0 || c.a == 0 {
		return 0
	}
	return stdmath.Sqrt(disc) / stdmath.Abs(c.a)
}

// dvdu returns dv/du on the branch passing through v, by implicit differentiation of the ruling
// quadratic: (a′v² + b′v + c′) + (2av + b)·dv/du = 0.
func (c ruledQuadricCoeffs) dvdu(v float64) float64 {
	den := 2*c.a*v + c.b
	if den == 0 {
		return 0 // a fold; the conditioning gate keeps a built arc away from one
	}
	return -(c.da*v*v + c.db*v + c.dc) / den
}

// nonZeroSign returns x when x ≠ 0 and +1 when it is, so Copysign never picks the sign of a zero
// (which would depend on the zero's sign bit rather than on geometry).
func nonZeroSign(x float64) float64 {
	if x == 0 {
		return 1
	}
	return x
}
