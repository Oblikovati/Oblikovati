// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"slices"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// 2D constraints — CIRCLE RELATIONS and TANGENTS (M48 #2241 split of constraints_2d.go). The circle/arc
// geometric constraints (concentric, equal-radius, line-tangent-to-circle, circle-tangent-to-circle) and
// their residual helpers. The point/incidence constraints and shared AD scaffolding live in
// constraints_2d.go; the line relations in constraints_2d_lines.go.

// ConcentricConstraint forces two circular curves (circles or arcs) to share a center.
type ConcentricConstraint struct {
	constraintBase
	C1, C2 CircularCurve
}

// AddConcentric constrains circular curves c1 and c2 to be concentric.
func (g *GeometricConstraints) AddConcentric(c1, c2 CircularCurve) *ConcentricConstraint {
	c := &ConcentricConstraint{constraintBase: newConstraint(), C1: c1, C2: c2}
	g.add(c)
	return c
}

// residualAD: v = [C1.X, C1.Y, C2.X, C2.Y]; the two centers must coincide.
func (c *ConcentricConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{v[0].Sub(v[2]), v[1].Sub(v[3])}
}

func (c *ConcentricConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *ConcentricConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *ConcentricConstraint) Variables() []*math.Scalar {
	a, b := c.C1.CenterPoint(), c.C2.CenterPoint()
	return []*math.Scalar{&a.X, &a.Y, &b.X, &b.Y}
}

// EqualRadiusConstraint forces two circular curves (circles or arcs) to the same radius.
type EqualRadiusConstraint struct {
	constraintBase
	C1, C2 CircularCurve
}

// AddEqualRadius constrains circular curves c1 and c2 to equal radii.
func (g *GeometricConstraints) AddEqualRadius(c1, c2 CircularCurve) *EqualRadiusConstraint {
	c := &EqualRadiusConstraint{constraintBase: newConstraint(), C1: c1, C2: c2}
	g.add(c)
	return c
}

// residualAD: v = [<C1 circularVars>, <C2 circularVars>]; the two radii must be equal.
func (c *EqualRadiusConstraint) residualAD(v []ad.Number) []ad.Number {
	_, r1, n := c.C1.circularFrameAD(v, 0)
	_, r2, _ := c.C2.circularFrameAD(v, n)
	return []ad.Number{r1.Sub(r2)}
}

func (c *EqualRadiusConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}

func (c *EqualRadiusConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *EqualRadiusConstraint) Variables() []*math.Scalar {
	return append(c.C1.circularVars(), c.C2.circularVars()...)
}

// TangentConstraint forces a line to be tangent to a circular curve (circle or arc).
//
// It has two formulations, chosen at creation, because the obvious one is degenerate in a
// common case (#2014). Normally the touch point is free and tangency is "the perpendicular
// distance from the centre to the line equals the radius". But when a line endpoint P is
// ALSO one of the curve's defining points — a slot's side meeting its cap arc, a fillet
// meeting its neighbour — the curve's radius is structurally |centre − P|, so that residual
// reduces to R(|sin φ| − 1) for φ the angle between line and radius. |sin φ| peaks at the
// φ = 90° solution, so the residual has a DOUBLE root: it is exactly zero there and its
// derivative is exactly zero too. The Jacobian row vanishes, the constraint contributes no
// rank, and the solver neither enforces nor reports it. (Dropping the Abs does not help —
// sin is stationary at 90° as well.)
//
// With P pinned, tangency is stated instead as the line being perpendicular to the radius
// at P: (B−A)·(centre−P) = 0. That has a simple root with gradient ≈ R, and it needs no
// sign or branch choice because the touch point cannot migrate.
type TangentConstraint struct {
	constraintBase
	L *Line
	C CircularCurve
	// touchAtB records the pinned formulation: nil-free when the line and curve share no
	// point, otherwise which line endpoint is the touch point.
	pinned   bool
	touchAtB bool
}

// AddTangent constrains line l to be tangent to circular curve c. It picks the
// perpendicular-radius formulation when l and c share a defining point (see
// [TangentConstraint]); the choice is topological, so it never flips mid-solve.
func (g *GeometricConstraints) AddTangent(l *Line, c CircularCurve) *TangentConstraint {
	t := &TangentConstraint{constraintBase: newConstraint(), L: l, C: c}
	t.pinned, t.touchAtB = lineTouchesCurve(l, c)
	g.add(t)
	return t
}

// lineTouchesCurve reports whether a line endpoint is one of the curve's defining points,
// and if so whether it is the line's B end. Identity is by pointer, so the test is exact.
func lineTouchesCurve(l *Line, c CircularCurve) (pinned, atB bool) {
	pts := curveDefiningPoints(c)
	for _, p := range pts {
		if p == l.A {
			return true, false
		}
		if p == l.B {
			return true, true
		}
	}
	return false, false
}

// curveDefiningPoints returns the points that structurally fix a curve's radius. An arc's
// radius is |centre − start| and its end is held on the circle by its circularity relation,
// so both endpoints count. A full circle carries an independent radius scalar, so no point
// pins it and the free formulation always applies.
func curveDefiningPoints(c CircularCurve) []*Point {
	a, ok := c.(*Arc)
	if !ok {
		return nil
	}
	return []*Point{a.Start, a.End}
}

// residualAD: v = [L.A.X, L.A.Y, L.B.X, L.B.Y, <C circularVars>].
func (t *TangentConstraint) residualAD(v []ad.Number) []ad.Number {
	a, b := ad.V2(v[0], v[1]), ad.V2(v[2], v[3])
	center, radius, _ := t.C.circularFrameAD(v, 4)
	dir := b.Sub(a)
	length := dir.Length()
	if length.Val() == 0 {
		return []ad.Number{radius} // degenerate line: cannot be tangent
	}
	if t.pinned {
		return []ad.Number{perpendicularRadiusResidual(a, b, center, dir, length, t.touchAtB)}
	}
	signed := dir.Cross(center.Sub(a)).Div(length)
	return []ad.Number{signed.Abs().Sub(radius)}
}

// perpendicularRadiusResidual is (B−A)·(centre−P) normalised by |B−A|, so the residual is a
// length comparable with the distance-based constraints it shares a system with. Without the
// normalisation the residual would be an area, and a long line would silently outweigh a
// short one.
func perpendicularRadiusResidual(a, b, center, dir ad.Vec2, length ad.Number, touchAtB bool) ad.Number {
	touch := a
	if touchAtB {
		touch = b
	}
	return dir.Dot(center.Sub(touch)).Div(length)
}

func (t *TangentConstraint) Residuals() []float64  { return adResiduals(t.Variables(), t.residualAD) }
func (t *TangentConstraint) Partials() [][]float64 { return adPartials(t.Variables(), t.residualAD) }

func (t *TangentConstraint) Variables() []*math.Scalar {
	return append([]*math.Scalar{&t.L.A.X, &t.L.A.Y, &t.L.B.X, &t.L.B.Y}, t.C.circularVars()...)
}

// CircularTangentConstraint forces two circular curves to be tangent: the distance
// between their centers equals r1+r2 (external tangency, the circles touching from
// outside) or |r1−r2| (internal, one curve inside the other). The mode is fixed at
// creation from the curves' current placement — whichever target the centers are
// already closer to — so the solver moves the geometry the least, matching how Inventor
// preserves the picked configuration.
// When the two curves share their touch point P, the centre-distance form is degenerate for
// the same reason [TangentConstraint]'s is: the triangle inequality gives |C1−C2| ≤ |C1−P| +
// |P−C2| = r1+r2 with equality exactly at tangency, so the residual reaches a MAXIMUM of zero
// there and its derivative vanishes. Tangency is then stated as collinearity of C1, P and C2,
// which has a simple root (#2014).
type CircularTangentConstraint struct {
	constraintBase
	C1, C2   CircularCurve
	internal bool
	// touch is the point both curves are pinned to, or nil when they share none.
	touch *Point
}

// AddCircularTangent constrains circular curves c1 and c2 to be tangent. It picks the
// collinear formulation when the curves share a defining point; the choice is topological, so
// it never flips mid-solve.
func (g *GeometricConstraints) AddCircularTangent(c1, c2 CircularCurve) *CircularTangentConstraint {
	t := &CircularTangentConstraint{constraintBase: newConstraint(), C1: c1, C2: c2}
	t.touch = sharedCurvePoint(c1, c2)
	t.internal = t.preferInternal()
	g.add(t)
	return t
}

// sharedCurvePoint returns the defining point two circular curves have in common, or nil.
// Identity is by pointer, so the test is exact and stable under solving.
func sharedCurvePoint(c1, c2 CircularCurve) *Point {
	second := curveDefiningPoints(c2)
	for _, p := range curveDefiningPoints(c1) {
		if slices.Contains(second, p) {
			return p
		}
	}
	return nil
}

// preferInternal reports whether internal tangency (|r1−r2|) is closer to the current
// center distance than external (r1+r2).
func (t *CircularTangentConstraint) preferInternal() bool {
	d := t.centerDistance()
	r1, r2 := t.C1.CurveRadius(), t.C2.CurveRadius()
	return stdmath.Abs(d-stdmath.Abs(r1-r2)) < stdmath.Abs(d-(r1+r2))
}

func (t *CircularTangentConstraint) centerDistance() float64 {
	a, b := t.C1.CenterPoint(), t.C2.CenterPoint()
	return stdmath.Hypot(a.X-b.X, a.Y-b.Y)
}

// residualAD: the center distance equals r1+r2 (external tangency) or |r1−r2| (internal).
// The mode was fixed at creation, so only the target form differs.
func (t *CircularTangentConstraint) residualAD(v []ad.Number) []ad.Number {
	c1, r1, n := t.C1.circularFrameAD(v, 0)
	c2, r2, m := t.C2.circularFrameAD(v, n)
	if t.touch != nil {
		return []ad.Number{collinearCentresResidual(c1, c2, ad.V2(v[n+m], v[n+m+1]))}
	}
	target := r1.Add(r2)
	if t.internal {
		target = r1.Sub(r2).Abs()
	}
	return []ad.Number{c1.Sub(c2).Length().Sub(target)}
}

// collinearCentresResidual states tangency at a shared touch point as "C1, P and C2 are
// collinear", normalised by |P−C1| so the residual is a length rather than an area. It does
// not distinguish internal from external tangency, which is correct here: with P pinned the
// touch point cannot migrate, so there is no branch to choose.
func collinearCentresResidual(c1, c2, p ad.Vec2) ad.Number {
	from := p.Sub(c1)
	length := from.Length()
	if length.Val() == 0 {
		return c2.Sub(p).Length() // degenerate: the touch point sits on the centre
	}
	return from.Cross(c2.Sub(p)).Div(length)
}

func (t *CircularTangentConstraint) Residuals() []float64 {
	return adResiduals(t.Variables(), t.residualAD)
}

func (t *CircularTangentConstraint) Partials() [][]float64 {
	return adPartials(t.Variables(), t.residualAD)
}

// Variables lists both curves' scalars, plus the shared touch point's when the collinear
// formulation is in use. The touch point may already appear among the curves' own variables;
// duplicate entries are harmless because the solver accumulates a column's contributions
// (scatterRow), which is exactly the chain rule for a repeated scalar.
func (t *CircularTangentConstraint) Variables() []*math.Scalar {
	vars := append(t.C1.circularVars(), t.C2.circularVars()...)
	if t.touch == nil {
		return vars
	}
	return append(vars, &t.touch.X, &t.touch.Y)
}
