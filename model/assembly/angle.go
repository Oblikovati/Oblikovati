// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// AngleConstraint holds a fixed angle between two directions (plane normals or axes). Its
// single residual is cos θ − cos(target), smooth and singularity-free across the range.
type AngleConstraint struct {
	*constraintBase
	angle    float64
	solution types.AngleConstraintSolutionType
}

// Value returns the target angle (radians).
func (c *AngleConstraint) Value() float64 { return c.angle }

// SetValue overrides the constrained angle (a positional representation, M12-F04).
func (c *AngleConstraint) SetValue(v float64) { c.angle = v }

// SolutionType returns how the angle is measured (undirected/directed/reference-vector).
func (c *AngleConstraint) SolutionType() types.AngleConstraintSolutionType { return c.solution }

// bind returns the angle's residual source.
func (c *AngleConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := c.boundPlacements(b)
		return angleResiduals(c.a.prim, c.b.prim, pa.matrix(), pb.matrix(), c.angle)
	})
}

// angleResiduals returns the cosine residual between the two directions.
func angleResiduals(a, b Primitive, ma, mb math.Matrix4, angle float64) []float64 {
	dA, dB := worldDir(ma, a), worldDir(mb, b)
	cos := dA.Dot(dB) / (dA.Length() * dB.Length())
	return []float64{cos - stdmath.Cos(angle)}
}
