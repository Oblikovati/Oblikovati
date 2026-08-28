// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// 2D constraints — LINE RELATIONS (M48 #2241 split of constraints_2d.go). The two-line geometric
// constraints (parallel, perpendicular, collinear, equal-length): each as a residual-and-Jacobian
// constraint whose residualAD is auto-differentiated. The point/incidence constraints and shared AD
// scaffolding live in constraints_2d.go; the circle relations in constraints_2d_circles.go.

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

// residualAD: the two line directions are parallel iff the sine of their angle is zero —
// the length-normalised cross product, scale-invariant (#1418).
func (c *ParallelConstraint) residualAD(v []ad.Number) []ad.Number {
	d1, d2 := adLineDirs(v)
	return []ad.Number{adSineAngle(d1, d2)}
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

// residualAD: the two line directions are perpendicular iff the cosine of their angle is
// zero — the length-normalised dot product, scale-invariant (#1418).
func (c *PerpendicularConstraint) residualAD(v []ad.Number) []ad.Number {
	d1, d2 := adLineDirs(v)
	return []ad.Number{adCosAngle(d1, d2)}
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
	// Parallel (sine of the angle) AND L2.A on L1's line (perpendicular distance) — both
	// scale-invariant rather than area-scaled (#1418).
	return []ad.Number{adSineAngle(d1, d2), adSignedPerpDistance(d1, a2.Sub(a1))}
}

func (c *CollinearConstraint) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *CollinearConstraint) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *CollinearConstraint) Variables() []*math.Scalar { return lineVars(c.L1, c.L2) }

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
