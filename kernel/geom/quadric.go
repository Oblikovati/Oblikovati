// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The IMPLICIT representation bucket of the surface–surface intersector (ADR-0058). OCCT's
// IntAna/IntSurf split a surface pair by REPRESENTATION — one side supplies a parametrisation, the
// other an implicit equation — rather than by an N² table of type pairs. Quadric is that implicit
// side for every second-degree surface the kernel has: substituting a parametrisation into F(X)=0
// gives one polynomial whose roots ARE the intersection, so a closed form follows from the
// substitution, not from a per-pair formula.

// Quadric is a second-degree implicit surface written relative to an anchor point (so its
// coefficients stay well-scaled next to the model's absolute coordinates):
//
//	F(X) = W·(M W) + 2·G·W + K,   W = X − Anchor
//
// F is zero exactly on the surface, and its sign is the surface's own inside/outside sense. A plane
// is the degenerate member (M = 0), a sphere/cylinder/cone the revolution members.
type Quadric struct {
	Anchor math.Point3
	M      SymmetricTensor3
	G      math.Vector3
	K      float64
}

// ValueAt evaluates F at a world point — zero on the surface.
//
//	q := cyl.QuadricForm()
//	onSurface := q.ValueAt(p) // ≈ 0
func (q Quadric) ValueAt(p math.Point3) float64 {
	w := q.Anchor.VectorTo(p)
	return float64(w.Dot(q.M.Apply(w))) + 2*float64(q.G.Dot(w)) + q.K
}

// ImplicitQuadric is the optional interface a surface implements when it HAS an exact second-degree
// implicit form. The intersector asks for it rather than switching on a type: a surface that does not
// answer (torus, B-spline, offset, threaded cylinder) simply is not in the implicit-quadric bucket and
// the pair takes the general marcher.
type ImplicitQuadric interface {
	Surface
	QuadricForm() Quadric
}

var (
	_ ImplicitQuadric = Plane{}
	_ ImplicitQuadric = Sphere{}
	_ ImplicitQuadric = Cylinder{}
	_ ImplicitQuadric = Cone{}
)

// QuadricForm returns the plane's degenerate quadric F(X) = n·(X − Origin): M is zero, so the
// substitution into a ruled parametrisation stays linear in the ruling parameter.
func (p Plane) QuadricForm() Quadric {
	n := unitVec3(p.Normal())
	return Quadric{Anchor: p.Origin, G: n.Scale(0.5)}
}

// QuadricForm returns the sphere's F(X) = |X − Center|² − R².
func (s Sphere) QuadricForm() Quadric {
	return Quadric{Anchor: s.Center, M: IdentityTensor3(), K: -s.Radius * s.Radius}
}

// QuadricForm returns the cylinder's F(X) = |W|² − (W·â)² − R², W = X − Origin: the squared radial
// distance from the axis, less the squared radius.
func (c Cylinder) QuadricForm() Quadric {
	return Quadric{Anchor: c.Origin, M: axialTensor3(c.AxisDir.AsVector(), 1), K: -c.Radius * c.Radius}
}

// QuadricForm returns the cone's F(X) = |W|² − sec²α·(W·â)², W = X − Apex: the half-angle condition
// (W·â)² = cos²α·|W|² rearranged, so it holds on BOTH nappes (as the cone's own surface does not —
// the caller's v-domain, v ≥ 0, is what keeps a section on the modelled nappe).
func (c Cone) QuadricForm() Quadric {
	cos, _ := cosSin(c.HalfAngle)
	return Quadric{Anchor: c.Apex, M: axialTensor3(c.AxisDir.AsVector(), 1/(cos*cos))}
}

// SymmetricTensor3 is a symmetric 3×3 tensor, the quadratic part of a [Quadric]. It is stored by its
// six independent entries rather than as a math.Matrix3 (which is a 2D affine transform, not a 3D
// linear operator).
type SymmetricTensor3 struct {
	XX, YY, ZZ, XY, YZ, ZX float64
}

// IdentityTensor3 returns the identity, whose quadratic form is |v|².
func IdentityTensor3() SymmetricTensor3 { return SymmetricTensor3{XX: 1, YY: 1, ZZ: 1} }

// axialTensor3 returns I − alpha·d̂d̂ᵀ for a unit axis d̂ — the quadratic part every quadric of
// revolution in the kernel shares (sphere alpha=0, cylinder alpha=1, cone alpha=sec²α).
func axialTensor3(axis math.Vector3, alpha float64) SymmetricTensor3 {
	d := unitVec3(axis)
	x, y, z := float64(d.X), float64(d.Y), float64(d.Z)
	return SymmetricTensor3{
		XX: 1 - alpha*x*x, YY: 1 - alpha*y*y, ZZ: 1 - alpha*z*z,
		XY: -alpha * x * y, YZ: -alpha * y * z, ZX: -alpha * z * x,
	}
}

// Apply returns M·v.
func (m SymmetricTensor3) Apply(v math.Vector3) math.Vector3 {
	x, y, z := float64(v.X), float64(v.Y), float64(v.Z)
	return math.V3(
		math.Scalar(m.XX*x+m.XY*y+m.ZX*z),
		math.Scalar(m.XY*x+m.YY*y+m.YZ*z),
		math.Scalar(m.ZX*x+m.YZ*y+m.ZZ*z),
	)
}

// Norm returns the Frobenius norm √(Σ mᵢⱼ²) — the magnitude the intersector's conditioning gate
// divides by to turn a quadratic coefficient into a dimensionless ratio.
func (m SymmetricTensor3) Norm() float64 {
	off := m.XY*m.XY + m.YZ*m.YZ + m.ZX*m.ZX
	return stdmath.Sqrt(m.XX*m.XX + m.YY*m.YY + m.ZZ*m.ZZ + 2*off)
}
