// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// TangentConstraint keeps a planar face tangent to a cylindrical face: the cylinder axis
// stays parallel to the plane, and the axis is one radius from the plane (outside) or one
// radius on the other side (inside). Two residuals.
type TangentConstraint struct {
	*constraintBase
	inside bool
}

// Inside reports inside tangency (the curved face wraps the plane); false is outside.
func (c *TangentConstraint) Inside() bool { return c.inside }

// bind returns the tangent's residual source.
func (c *TangentConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := c.boundPlacements(b)
		return tangentResiduals(c.a.prim, c.b.prim, pa.matrix(), pb.matrix(), c.inside)
	})
}

// tangentResiduals orders the inputs so the plane is A and the cylinder is B, then returns
// [axis·normal (parallel-to-plane), axis-to-plane distance − radius].
func tangentResiduals(a, b Primitive, ma, mb math.Matrix4, inside bool) []float64 {
	plane, planeM, cyl, cylM := a, ma, b, mb
	if a.kind != planeKind {
		plane, planeM, cyl, cylM = b, mb, a, ma
	}
	nA := worldDir(planeM, plane)
	pA := worldPoint(planeM, plane)
	dB := worldDir(cylM, cyl)
	pB := worldPoint(cylM, cyl)
	radius := cyl.radius
	if inside {
		radius = -radius
	}
	return []float64{dB.Dot(nA), pA.VectorTo(pB).Dot(nA) - radius}
}
