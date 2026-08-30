// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// This file defines the 2D geometric constraints and their factory methods on
// [GeometricConstraints]. Each constraint reads live entity state in Residuals and
// exposes its variables by pointer; the solver (F05) drives them to zero.

// CoincidentConstraint makes two points share a location.
type CoincidentConstraint struct {
	constraintBase
	A, B *Point
}

// AddCoincident constrains points a and b to coincide.
func (g *GeometricConstraints) AddCoincident(a, b *Point) *CoincidentConstraint {
	c := &CoincidentConstraint{constraintBase: newConstraint(), A: a, B: b}
	g.add(c)
	return c
}

// residualAD: v = [A.X, A.Y, B.X, B.Y]; the two points must coincide.
func (c *CoincidentConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{v[0].Sub(v[2]), v[1].Sub(v[3])}
}

func (c *CoincidentConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *CoincidentConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *CoincidentConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.A.X, &c.A.Y, &c.B.X, &c.B.Y}
}

// PointOnLineConstraint forces a point to lie on a line's infinite line — the
// point-to-curve coincidence (zero perpendicular distance to the line).
type PointOnLineConstraint struct {
	constraintBase
	P *Point
	L *Line
}

// AddPointOnLine constrains point p to lie on line l.
func (g *GeometricConstraints) AddPointOnLine(p *Point, l *Line) *PointOnLineConstraint {
	c := &PointOnLineConstraint{constraintBase: newConstraint(), P: p, L: l}
	g.add(c)
	return c
}

// residualAD: v = [P.X, P.Y, A.X, A.Y, B.X, B.Y]. The signed perpendicular distance of P
// from the line is zero iff P lies on it — a true distance residual, not the area
// |line|·|offset| (#1418).
func (c *PointOnLineConstraint) residualAD(v []ad.Number) []ad.Number {
	p, a, b := ad.V2(v[0], v[1]), ad.V2(v[2], v[3]), ad.V2(v[4], v[5])
	return []ad.Number{adSignedPerpDistance(b.Sub(a), p.Sub(a))}
}

func (c *PointOnLineConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}

func (c *PointOnLineConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *PointOnLineConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.P.X, &c.P.Y, &c.L.A.X, &c.L.A.Y, &c.L.B.X, &c.L.B.Y}
}

// MidpointConstraint forces a point to the midpoint of a line.
type MidpointConstraint struct {
	constraintBase
	P *Point
	L *Line
}

// AddMidpoint constrains point p to the midpoint of line l.
func (g *GeometricConstraints) AddMidpoint(p *Point, l *Line) *MidpointConstraint {
	c := &MidpointConstraint{constraintBase: newConstraint(), P: p, L: l}
	g.add(c)
	return c
}

// residualAD: v = [P.X, P.Y, A.X, A.Y, B.X, B.Y]; P must equal (A+B)/2.
func (c *MidpointConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{
		v[0].Sub(v[2].Add(v[4]).Scale(0.5)),
		v[1].Sub(v[3].Add(v[5]).Scale(0.5)),
	}
}

func (c *MidpointConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *MidpointConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *MidpointConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.P.X, &c.P.Y, &c.L.A.X, &c.L.A.Y, &c.L.B.X, &c.L.B.Y}
}

// PointOnCircleConstraint forces a point onto a circular curve's outline (distance from
// the center equals the radius). The curve is a circle or an arc.
type PointOnCircleConstraint struct {
	constraintBase
	P *Point
	C CircularCurve
}

// AddPointOnCircle constrains point p to lie on circular curve c's outline.
func (g *GeometricConstraints) AddPointOnCircle(p *Point, c CircularCurve) *PointOnCircleConstraint {
	con := &PointOnCircleConstraint{constraintBase: newConstraint(), P: p, C: c}
	g.add(con)
	return con
}

// residualAD: v = [P.X, P.Y, <curve circularVars>]; |P − center| must equal the radius.
func (c *PointOnCircleConstraint) residualAD(v []ad.Number) []ad.Number {
	p := ad.V2(v[0], v[1])
	center, radius, _ := c.C.circularFrameAD(v, 2)
	return []ad.Number{p.Sub(center).Length().Sub(radius)}
}

func (c *PointOnCircleConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}

func (c *PointOnCircleConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *PointOnCircleConstraint) Variables() []*math.Scalar {
	return append([]*math.Scalar{&c.P.X, &c.P.Y}, c.C.circularVars()...)
}

// HorizontalConstraint forces two points to the same Y (a horizontal segment).
type HorizontalConstraint struct {
	constraintBase
	A, B *Point
}

// AddHorizontal constrains the segment a–b to be horizontal.
func (g *GeometricConstraints) AddHorizontal(a, b *Point) *HorizontalConstraint {
	c := &HorizontalConstraint{constraintBase: newConstraint(), A: a, B: b}
	g.add(c)
	return c
}

// residualAD: v = [A.Y, B.Y]; the two endpoints must share a Y.
func (c *HorizontalConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{v[0].Sub(v[1])}
}

func (c *HorizontalConstraint) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *HorizontalConstraint) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *HorizontalConstraint) Variables() []*math.Scalar { return []*math.Scalar{&c.A.Y, &c.B.Y} }

// VerticalConstraint forces two points to the same X (a vertical segment).
type VerticalConstraint struct {
	constraintBase
	A, B *Point
}

// AddVertical constrains the segment a–b to be vertical.
func (g *GeometricConstraints) AddVertical(a, b *Point) *VerticalConstraint {
	c := &VerticalConstraint{constraintBase: newConstraint(), A: a, B: b}
	g.add(c)
	return c
}

// residualAD: v = [A.X, B.X]; the two endpoints must share an X.
func (c *VerticalConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{v[0].Sub(v[1])}
}

func (c *VerticalConstraint) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *VerticalConstraint) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *VerticalConstraint) Variables() []*math.Scalar { return []*math.Scalar{&c.A.X, &c.B.X} }

// SymmetryConstraint forces two points to mirror across a line: their midpoint lies
// on the line and the segment between them is perpendicular to it.
type SymmetryConstraint struct {
	constraintBase
	A, B  *Point
	About *Line
}

// AddSymmetry constrains points a and b to be symmetric about line about.
func (g *GeometricConstraints) AddSymmetry(a, b *Point, about *Line) *SymmetryConstraint {
	c := &SymmetryConstraint{constraintBase: newConstraint(), A: a, B: b, About: about}
	g.add(c)
	return c
}

// residualAD: v = [A.X, A.Y, B.X, B.Y, About.A.X, About.A.Y, About.B.X, About.B.Y]. The
// midpoint of A,B lies on the mirror line (onLine) and the A→B segment is perpendicular
// to it (perp).
func (c *SymmetryConstraint) residualAD(v []ad.Number) []ad.Number {
	a, b := ad.V2(v[0], v[1]), ad.V2(v[2], v[3])
	la, lb := ad.V2(v[4], v[5]), ad.V2(v[6], v[7])
	return adPointSymmetry(a, b, la, lb)
}

func (c *SymmetryConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *SymmetryConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *SymmetryConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.A.X, &c.A.Y, &c.B.X, &c.B.Y, &c.About.A.X, &c.About.A.Y, &c.About.B.X, &c.About.B.Y}
}

// FixConstraint pins a point to a fixed position (captured at creation).
type FixConstraint struct {
	constraintBase
	P      *Point
	x0, y0 math.Scalar
	// weight scales the residual. 0 means a hard fix (⇒ weight 1) — a user or persisted Fix. A
	// small positive value makes it a SOFT pin: because the solver minimises the raw sum of
	// squares, a small weight lets the pin yield to hard constraints instead of fighting them.
	// Drag pins are soft so dragging an entity whose points are held by a dimension or a
	// tangency does not violate that constraint, and instead moves the geometry within its
	// remaining freedom (#2160). Zero-valued (cloned/restored) fixes stay hard.
	weight float64
}

// residualWeight returns the residual scale, treating the zero value as a hard fix (weight 1).
func (c *FixConstraint) residualWeight() float64 {
	if c.weight > 0 {
		return c.weight
	}
	return 1
}

// AddFix pins point p to its current location.
func (g *GeometricConstraints) AddFix(p *Point) *FixConstraint {
	c := &FixConstraint{constraintBase: newConstraint(), P: p, x0: p.X, y0: p.Y}
	g.add(c)
	return c
}

// residualAD: each coordinate must hold its captured value (scaled by the fix's weight).
func (c *FixConstraint) residualAD(v []ad.Number) []ad.Number {
	w := ad.Const(c.residualWeight())
	return []ad.Number{v[0].AddConst(-float64(c.x0)).Mul(w), v[1].AddConst(-float64(c.y0)).Mul(w)}
}

func (c *FixConstraint) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *FixConstraint) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *FixConstraint) Variables() []*math.Scalar { return []*math.Scalar{&c.P.X, &c.P.Y} }

// lineVars returns the eight endpoint coordinates of two lines.
func lineVars(l1, l2 *Line) []*math.Scalar {
	return []*math.Scalar{&l1.A.X, &l1.A.Y, &l1.B.X, &l1.B.Y, &l2.A.X, &l2.A.Y, &l2.B.X, &l2.B.Y}
}

// adLineDirs returns the two line directions (B−A) from a seeded lineVars row.
func adLineDirs(v []ad.Number) (d1, d2 ad.Vec2) {
	a1, b1, a2, b2 := adTwoLines(v)
	return b1.Sub(a1), b2.Sub(a2)
}

// adTwoLines returns the four endpoints (L1.A, L1.B, L2.A, L2.B) from a seeded lineVars row.
func adTwoLines(v []ad.Number) (a1, b1, a2, b2 ad.Vec2) {
	return ad.V2(v[0], v[1]), ad.V2(v[2], v[3]), ad.V2(v[4], v[5]), ad.V2(v[6], v[7])
}
