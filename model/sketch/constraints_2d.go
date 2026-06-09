// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
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

func (c *CoincidentConstraint) Residuals() []float64 {
	return []float64{c.A.X - c.B.X, c.A.Y - c.B.Y}
}

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

func (c *PointOnLineConstraint) Residuals() []float64 {
	dx, dy := lineDir(c.L)
	// Cross product of (P−A) with the line direction is zero iff P is on the line.
	ox, oy := c.P.X-c.L.A.X, c.P.Y-c.L.A.Y
	return []float64{dx*oy - dy*ox}
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

func (c *MidpointConstraint) Residuals() []float64 {
	return []float64{
		c.P.X - (c.L.A.X+c.L.B.X)/2,
		c.P.Y - (c.L.A.Y+c.L.B.Y)/2,
	}
}

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

func (c *PointOnCircleConstraint) Residuals() []float64 {
	ctr := c.C.CenterPoint()
	return []float64{stdmath.Hypot(c.P.X-ctr.X, c.P.Y-ctr.Y) - c.C.CurveRadius()}
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
func (c *HorizontalConstraint) Residuals() []float64      { return []float64{c.A.Y - c.B.Y} }
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
func (c *VerticalConstraint) Residuals() []float64      { return []float64{c.A.X - c.B.X} }
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

func (c *ParallelConstraint) Residuals() []float64 {
	d1x, d1y := lineDir(c.L1)
	d2x, d2y := lineDir(c.L2)
	return []float64{d1x*d2y - d1y*d2x}
}
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

func (c *PerpendicularConstraint) Residuals() []float64 {
	d1x, d1y := lineDir(c.L1)
	d2x, d2y := lineDir(c.L2)
	return []float64{d1x*d2x + d1y*d2y}
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

func (c *CollinearConstraint) Residuals() []float64 {
	d1x, d1y := lineDir(c.L1)
	d2x, d2y := lineDir(c.L2)
	// parallel, and L2.A lies on L1's line.
	ox, oy := c.L2.A.X-c.L1.A.X, c.L2.A.Y-c.L1.A.Y
	return []float64{d1x*d2y - d1y*d2x, d1x*oy - d1y*ox}
}
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

func (c *ConcentricConstraint) Residuals() []float64 {
	a, b := c.C1.CenterPoint(), c.C2.CenterPoint()
	return []float64{a.X - b.X, a.Y - b.Y}
}

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

func (c *EqualLengthConstraint) Residuals() []float64 {
	return []float64{c.L1.Length() - c.L2.Length()}
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

func (c *EqualRadiusConstraint) Residuals() []float64 {
	return []float64{c.C1.CurveRadius() - c.C2.CurveRadius()}
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

func (t *TangentConstraint) Residuals() []float64 {
	dx, dy := lineDir(t.L)
	length := stdmath.Hypot(dx, dy)
	if length == 0 {
		return []float64{t.C.CurveRadius()} // degenerate line: cannot be tangent
	}
	// Unsigned perpendicular distance from the center to the line equals the
	// radius at tangency. (The center may lie on either side of the line.)
	ctr := t.C.CenterPoint()
	signed := (dx*(ctr.Y-t.L.A.Y) - dy*(ctr.X-t.L.A.X)) / length
	return []float64{stdmath.Abs(signed) - t.C.CurveRadius()}
}

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

func (t *CircularTangentConstraint) Residuals() []float64 {
	r1, r2 := t.C1.CurveRadius(), t.C2.CurveRadius()
	target := r1 + r2
	if t.internal {
		target = stdmath.Abs(r1 - r2)
	}
	return []float64{t.centerDistance() - target}
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

func (c *SymmetryConstraint) Residuals() []float64 {
	dx, dy := lineDir(c.About)
	mx, my := (c.A.X+c.B.X)/2, (c.A.Y+c.B.Y)/2
	onLine := dx*(my-c.About.A.Y) - dy*(mx-c.About.A.X)
	perp := (c.B.X-c.A.X)*dx + (c.B.Y-c.A.Y)*dy
	return []float64{onLine, perp}
}

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
func (c *FixConstraint) Residuals() []float64      { return []float64{c.P.X - c.x0, c.P.Y - c.y0} }
func (c *FixConstraint) Variables() []*math.Scalar { return []*math.Scalar{&c.P.X, &c.P.Y} }

// lineDir returns a line's direction components (B-A).
func lineDir(l *Line) (dx, dy math.Scalar) {
	return l.B.X - l.A.X, l.B.Y - l.A.Y
}

// lineVars returns the eight endpoint coordinates of two lines.
func lineVars(l1, l2 *Line) []*math.Scalar {
	return []*math.Scalar{&l1.A.X, &l1.A.Y, &l1.B.X, &l1.B.Y, &l2.A.X, &l2.A.Y, &l2.B.X, &l2.B.Y}
}
