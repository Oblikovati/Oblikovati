// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// TransitionalConstraint keeps a face in sliding contact with a transition face: the
// moving geometry's reference point stays on the transition plane (a single contact
// residual), so the part slides over the surface rather than into it. The full motion
// over a chain of adjacent faces is realized during a drive (M12-F03); the static
// constraint maintains contact at the current configuration.
type TransitionalConstraint struct {
	*constraintBase
}

// bind returns the transitional contact residual.
func (c *TransitionalConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := c.boundPlacements(b)
		return transitionalResiduals(c.a.prim, c.b.prim, pa.matrix(), pb.matrix())
	})
}

// transitionalResiduals order the inputs so the transition plane is the one carrying a
// normal, then measure the other input's signed distance to it (zero ⇒ in contact).
func transitionalResiduals(a, b Primitive, ma, mb math.Matrix4) []float64 {
	plane, planeM, other, otherM := b, mb, a, ma
	if b.kind != planeKind {
		plane, planeM, other, otherM = a, ma, b, mb
	}
	n := worldDir(planeM, plane)
	return []float64{worldPoint(planeM, plane).VectorTo(worldPoint(otherM, other)).Dot(n)}
}
