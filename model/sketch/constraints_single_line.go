// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// Single-line horizontal/vertical (#1871): Inventor's AddHorizontal(line)/AddVertical(line)
// make ONE line horizontal/vertical — a distinct constraint type from the two-point
// HorizontalAlign/VerticalAlign (which the existing HorizontalConstraint/VerticalConstraint
// carry). Keeping them distinct lets enumeration and the exporter tell "this line is
// horizontal" from "these two points are level". The residual is the line's own endpoints
// sharing a coordinate; the constraint relates the LINE, not its points.

// SingleLineHorizontalConstraint makes a single line horizontal (its two endpoints share Y).
type SingleLineHorizontalConstraint struct {
	constraintBase
	L *Line
}

// AddLineHorizontal makes line l horizontal.
func (g *GeometricConstraints) AddLineHorizontal(l *Line) *SingleLineHorizontalConstraint {
	c := &SingleLineHorizontalConstraint{constraintBase: newConstraint(), L: l}
	g.add(c)
	return c
}

// residualAD: v = [A.Y, B.Y]; the endpoints must share a Y.
func (c *SingleLineHorizontalConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{v[0].Sub(v[1])}
}
func (c *SingleLineHorizontalConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *SingleLineHorizontalConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}
func (c *SingleLineHorizontalConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.L.A.Y, &c.L.B.Y}
}

// SingleLineVerticalConstraint makes a single line vertical (its two endpoints share X).
type SingleLineVerticalConstraint struct {
	constraintBase
	L *Line
}

// AddLineVertical makes line l vertical.
func (g *GeometricConstraints) AddLineVertical(l *Line) *SingleLineVerticalConstraint {
	c := &SingleLineVerticalConstraint{constraintBase: newConstraint(), L: l}
	g.add(c)
	return c
}

// residualAD: v = [A.X, B.X]; the endpoints must share an X.
func (c *SingleLineVerticalConstraint) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{v[0].Sub(v[1])}
}
func (c *SingleLineVerticalConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *SingleLineVerticalConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}
func (c *SingleLineVerticalConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.L.A.X, &c.L.B.X}
}
