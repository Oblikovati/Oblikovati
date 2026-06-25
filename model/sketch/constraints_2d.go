// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

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

// residualAD: v = [P.X, P.Y, A.X, A.Y, B.X, B.Y]. The cross product of (P−A) with the
// line direction (B−A) is zero iff P lies on the line.
func (c *PointOnLineConstraint) residualAD(v []ad.Number) []ad.Number {
	p, a, b := ad.V2(v[0], v[1]), ad.V2(v[2], v[3]), ad.V2(v[4], v[5])
	return []ad.Number{b.Sub(a).Cross(p.Sub(a))}
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

// ParallelConstraint forces two lines to be parallel (zero direction cross product).
type ParallelConstraint struct {
	constraintBase
	L1, L2 *Line
}

// AddParallel constrains lines l1 and l2 to be parallel.
func (g *GeometricConstraints) AddParallel(l1, l2 *Line) *ParallelConstraint {
	c := &ParallelConstraint{constraintBase: newConstraint(), L1: l1, L2: l2}
	g.add(c)
	return c
}

// residualAD: the two line directions are parallel iff their cross product is zero.
func (c *ParallelConstraint) residualAD(v []ad.Number) []ad.Number {
	d1, d2 := adLineDirs(v)
	return []ad.Number{d1.Cross(d2)}
}
func (c *ParallelConstraint) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *ParallelConstraint) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *ParallelConstraint) Variables() []*math.Scalar { return lineVars(c.L1, c.L2) }

// PerpendicularConstraint forces two lines to meet at a right angle (zero dot product).
type PerpendicularConstraint struct {
	constraintBase
	L1, L2 *Line
}

// AddPerpendicular constrains lines l1 and l2 to be perpendicular.
func (g *GeometricConstraints) AddPerpendicular(l1, l2 *Line) *PerpendicularConstraint {
	c := &PerpendicularConstraint{constraintBase: newConstraint(), L1: l1, L2: l2}
	g.add(c)
	return c
}

// residualAD: the two line directions are perpendicular iff their dot product is zero.
func (c *PerpendicularConstraint) residualAD(v []ad.Number) []ad.Number {
	d1, d2 := adLineDirs(v)
	return []ad.Number{d1.Dot(d2)}
}
func (c *PerpendicularConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *PerpendicularConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}
func (c *PerpendicularConstraint) Variables() []*math.Scalar { return lineVars(c.L1, c.L2) }

// CollinearConstraint forces two lines onto the same infinite line.
type CollinearConstraint struct {
	constraintBase
	L1, L2 *Line
}

// AddCollinear constrains lines l1 and l2 to be collinear.
func (g *GeometricConstraints) AddCollinear(l1, l2 *Line) *CollinearConstraint {
	c := &CollinearConstraint{constraintBase: newConstraint(), L1: l1, L2: l2}
	g.add(c)
	return c
}

// residualAD: the lines are collinear iff they are parallel (cross of directions zero)
// AND L2.A lies on L1's line (cross of d1 with L2.A−L1.A zero).
func (c *CollinearConstraint) residualAD(v []ad.Number) []ad.Number {
	a1, b1, a2, b2 := adTwoLines(v)
	d1, d2 := b1.Sub(a1), b2.Sub(a2)
	return []ad.Number{d1.Cross(d2), d1.Cross(a2.Sub(a1))}
}
func (c *CollinearConstraint) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *CollinearConstraint) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *CollinearConstraint) Variables() []*math.Scalar { return lineVars(c.L1, c.L2) }

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

// EqualLengthConstraint forces two lines to the same length.
type EqualLengthConstraint struct {
	constraintBase
	L1, L2 *Line
}

// AddEqualLength constrains lines l1 and l2 to have equal length.
func (g *GeometricConstraints) AddEqualLength(l1, l2 *Line) *EqualLengthConstraint {
	c := &EqualLengthConstraint{constraintBase: newConstraint(), L1: l1, L2: l2}
	g.add(c)
	return c
}

// residualAD: the two segment lengths must be equal.
func (c *EqualLengthConstraint) residualAD(v []ad.Number) []ad.Number {
	a1, b1, a2, b2 := adTwoLines(v)
	return []ad.Number{b1.Sub(a1).Length().Sub(b2.Sub(a2).Length())}
}
func (c *EqualLengthConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *EqualLengthConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}
func (c *EqualLengthConstraint) Variables() []*math.Scalar { return lineVars(c.L1, c.L2) }

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

// TangentConstraint forces a line to be tangent to a circular curve (circle or arc):
// the unsigned perpendicular distance from the curve's center to the line equals the
// radius.
type TangentConstraint struct {
	constraintBase
	L *Line
	C CircularCurve
}

// AddTangent constrains line l to be tangent to circular curve c.
func (g *GeometricConstraints) AddTangent(l *Line, c CircularCurve) *TangentConstraint {
	t := &TangentConstraint{constraintBase: newConstraint(), L: l, C: c}
	g.add(t)
	return t
}

// residualAD: v = [L.A.X, L.A.Y, L.B.X, L.B.Y, <C circularVars>]. The unsigned
// perpendicular distance from the center to the infinite line equals the radius at
// tangency (the center may lie on either side, hence Abs).
func (t *TangentConstraint) residualAD(v []ad.Number) []ad.Number {
	a, b := ad.V2(v[0], v[1]), ad.V2(v[2], v[3])
	center, radius, _ := t.C.circularFrameAD(v, 4)
	dir := b.Sub(a)
	length := dir.Length()
	if length.Val() == 0 {
		return []ad.Number{radius} // degenerate line: cannot be tangent
	}
	signed := dir.Cross(center.Sub(a)).Div(length)
	return []ad.Number{signed.Abs().Sub(radius)}
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
type CircularTangentConstraint struct {
	constraintBase
	C1, C2   CircularCurve
	internal bool
}

// AddCircularTangent constrains circular curves c1 and c2 to be tangent.
func (g *GeometricConstraints) AddCircularTangent(c1, c2 CircularCurve) *CircularTangentConstraint {
	t := &CircularTangentConstraint{constraintBase: newConstraint(), C1: c1, C2: c2}
	t.internal = t.preferInternal()
	g.add(t)
	return t
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
	c2, r2, _ := t.C2.circularFrameAD(v, n)
	target := r1.Add(r2)
	if t.internal {
		target = r1.Sub(r2).Abs()
	}
	return []ad.Number{c1.Sub(c2).Length().Sub(target)}
}
func (t *CircularTangentConstraint) Residuals() []float64 {
	return adResiduals(t.Variables(), t.residualAD)
}
func (t *CircularTangentConstraint) Partials() [][]float64 {
	return adPartials(t.Variables(), t.residualAD)
}

func (t *CircularTangentConstraint) Variables() []*math.Scalar {
	return append(t.C1.circularVars(), t.C2.circularVars()...)
}

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
	dir := lb.Sub(la)
	mid := a.Add(b).Scale(0.5)
	return []ad.Number{dir.Cross(mid.Sub(la)), b.Sub(a).Dot(dir)}
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
}

// AddFix pins point p to its current location.
func (g *GeometricConstraints) AddFix(p *Point) *FixConstraint {
	c := &FixConstraint{constraintBase: newConstraint(), P: p, x0: p.X, y0: p.Y}
	g.add(c)
	return c
}

// residualAD: each coordinate must hold its captured value.
func (c *FixConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{v[0].AddConst(-float64(c.x0)), v[1].AddConst(-float64(c.y0))}
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
