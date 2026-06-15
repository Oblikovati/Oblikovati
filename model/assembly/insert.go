// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// InsertConstraint combines an axis mate (the two axes collinear) with an axial offset (a
// plane mate along the axis) — the "bolt into a hole" relationship. It removes five DOF,
// leaving the spin about the common axis free.
type InsertConstraint struct {
	*constraintBase
	offset  float64
	aligned bool
}

// Value returns the axial offset (seating depth).
func (c *InsertConstraint) Value() float64 { return c.offset }

// SetValue overrides the insert offset (a positional representation, M12-F04).
func (c *InsertConstraint) SetValue(v float64) { c.offset = v }

// Aligned reports the aligned plane sense; false is the default opposed sense.
func (c *InsertConstraint) Aligned() bool { return c.aligned }

// bind returns the insert's residual source.
func (c *InsertConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := c.boundPlacements(b)
		return insertResiduals(c.a.prim, c.b.prim, pa.matrix(), pb.matrix(), c.offset)
	})
}

// insertResiduals makes the two axes collinear (align + perpendicular offset) and seats
// the first the offset along the second's axis — five residuals.
func insertResiduals(a, b Primitive, ma, mb math.Matrix4, offset float64) []float64 {
	dA, dB := worldDir(ma, a), worldDir(mb, b)
	pA, pB := worldPoint(ma, a), worldPoint(mb, b)
	res := alignResiduals(dA, dB)
	res = append(res, perpDistanceResiduals(pA, pB, dB)...)
	return append(res, gapResidual(pA, pB, dB, offset))
}
