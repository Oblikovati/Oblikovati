// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// AssemblySymmetryConstraint positions two geometry inputs symmetrically about a plane:
// reflecting input A's point across the plane lands it on input B's point. Three
// residuals. The symmetry plane is a third anchor (on its own occurrence).
type AssemblySymmetryConstraint struct {
	*constraintBase
	plane anchor
}

// anchors returns all three inputs (the two symmetric geometries and the plane) so the
// per-occurrence view sees the plane's occurrence too.
func (c *AssemblySymmetryConstraint) anchors() []anchor {
	return []anchor{c.a, c.b, c.plane}
}

// bind returns the symmetry residual source.
func (c *AssemblySymmetryConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := c.boundPlacements(b)
		pp := b(c.plane.occ)
		return symmetryResiduals(c.a.prim, c.b.prim, c.plane.prim, pa.matrix(), pb.matrix(), pp.matrix())
	})
}

// symmetryResiduals reflect A's point across the plane and measure its offset from B's
// point (three residuals, zero when the two are mirror images).
func symmetryResiduals(a, b, plane Primitive, ma, mb, mp math.Matrix4) []float64 {
	pA := worldPoint(ma, a)
	pB := worldPoint(mb, b)
	planePt := worldPoint(mp, plane)
	n := worldDir(mp, plane)
	reflected := reflectAcrossPlane(pA, planePt, n)
	d := pB.VectorTo(reflected)
	return []float64{d.X, d.Y, d.Z}
}

// reflectAcrossPlane mirrors p across the plane through planePt with unit normal n:
// p − 2·((p − planePt)·n)·n.
func reflectAcrossPlane(p, planePt math.Point3, n math.Vector3) math.Point3 {
	signed := planePt.VectorTo(p).Dot(n)
	return p.TranslateBy(n.Scale(-2 * signed))
}
