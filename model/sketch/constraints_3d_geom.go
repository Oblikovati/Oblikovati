// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// This file holds the line/point-based 3D geometric constraints (M22-F05) that extend the
// point-only set in constraints_3d.go: parallel, perpendicular, midpoint, ground, and the
// orientation-to-origin-frame constraints (parallel-to-X/Y/Z-axis, parallel-to-XY/XZ/YZ-
// plane). Each implements [Constraint] over the dimension-agnostic solver — a line's
// direction residual reads its two endpoints' nine coordinates by pointer.

// line3DVars returns the six endpoint coordinate DOFs of a 3D line.
func line3DVars(l *Line3D) []*math.Scalar {
	return []*math.Scalar{&l.A.X, &l.A.Y, &l.A.Z, &l.B.X, &l.B.Y, &l.B.Z}
}

// Parallel3D forces two 3D lines to be parallel: their direction cross product vanishes
// (three residual components; the solver's rank analysis accounts for the redundant one).
type Parallel3D struct {
	constraintBase
	L1, L2 *Line3D
}

// NewParallel3D constrains two 3D lines to be parallel.
func NewParallel3D(l1, l2 *Line3D) *Parallel3D {
	return &Parallel3D{constraintBase: newConstraint(), L1: l1, L2: l2}
}

// residualAD: v = [L1.A.xyz, L1.B.xyz, L2.A.xyz, L2.B.xyz]; the directions' cross vanishes.
func (c *Parallel3D) residualAD(v []ad.Number) []ad.Number {
	cr := adLine3DDir(v, 0).Cross(adLine3DDir(v, 6))
	return []ad.Number{cr.X, cr.Y, cr.Z}
}
func (c *Parallel3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Parallel3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *Parallel3D) Variables() []*math.Scalar {
	return append(line3DVars(c.L1), line3DVars(c.L2)...)
}

// Perpendicular3D forces two 3D lines to be perpendicular: their direction dot product
// vanishes.
type Perpendicular3D struct {
	constraintBase
	L1, L2 *Line3D
}

// NewPerpendicular3D constrains two 3D lines to be perpendicular.
func NewPerpendicular3D(l1, l2 *Line3D) *Perpendicular3D {
	return &Perpendicular3D{constraintBase: newConstraint(), L1: l1, L2: l2}
}

// residualAD: the two line directions are perpendicular iff their dot product is zero.
func (c *Perpendicular3D) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{adLine3DDir(v, 0).Dot(adLine3DDir(v, 6))}
}
func (c *Perpendicular3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Perpendicular3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *Perpendicular3D) Variables() []*math.Scalar {
	return append(line3DVars(c.L1), line3DVars(c.L2)...)
}

// Midpoint3D forces a point to be the midpoint of a 3D line.
type Midpoint3D struct {
	constraintBase
	P *Point3D
	L *Line3D
}

// NewMidpoint3D constrains p to lie at the midpoint of l.
func NewMidpoint3D(p *Point3D, l *Line3D) *Midpoint3D {
	return &Midpoint3D{constraintBase: newConstraint(), P: p, L: l}
}

// residualAD: v = [P.xyz, A.xyz, B.xyz]; P must equal (A+B)/2.
func (c *Midpoint3D) residualAD(v []ad.Number) []ad.Number {
	d := adV3(v, 0).Sub(adV3(v, 3).Add(adV3(v, 6)).Scale(0.5))
	return []ad.Number{d.X, d.Y, d.Z}
}
func (c *Midpoint3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Midpoint3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

func (c *Midpoint3D) Variables() []*math.Scalar {
	return []*math.Scalar{&c.P.X, &c.P.Y, &c.P.Z, &c.L.A.X, &c.L.A.Y, &c.L.A.Z, &c.L.B.X, &c.L.B.Y, &c.L.B.Z}
}

// Ground3D pins a 3D point at a fixed model position (the Inventor "fix"/ground
// constraint). The target is captured when the constraint is created.
type Ground3D struct {
	constraintBase
	P      *Point3D
	Target math.Point3
}

// NewGround3D fixes p at its current position.
func NewGround3D(p *Point3D) *Ground3D {
	return &Ground3D{constraintBase: newConstraint(), P: p, Target: p.Position()}
}

// residualAD: v = [P.xyz]; the point must hold its captured target position.
func (c *Ground3D) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{
		v[0].AddConst(-float64(c.Target.X)),
		v[1].AddConst(-float64(c.Target.Y)),
		v[2].AddConst(-float64(c.Target.Z)),
	}
}
func (c *Ground3D) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *Ground3D) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *Ground3D) Variables() []*math.Scalar { return []*math.Scalar{&c.P.X, &c.P.Y, &c.P.Z} }

// ParallelToAxis3D forces a 3D line to be parallel to a fixed world axis (its direction
// cross the axis vanishes). Use the X/Y/Z constructors for the origin frame.
type ParallelToAxis3D struct {
	constraintBase
	L    *Line3D
	Axis math.Vector3
}

// NewParallelToAxis3D constrains l parallel to the given axis; the X/Y/Z helpers pick the
// origin-frame axes.
func NewParallelToAxis3D(l *Line3D, axis math.Vector3) *ParallelToAxis3D {
	return &ParallelToAxis3D{constraintBase: newConstraint(), L: l, Axis: axis}
}

// NewParallelToXAxis3D / YAxis / ZAxis constrain l parallel to the named origin-frame axis.
func NewParallelToXAxis3D(l *Line3D) *ParallelToAxis3D {
	return NewParallelToAxis3D(l, math.V3(1, 0, 0))
}

func NewParallelToYAxis3D(l *Line3D) *ParallelToAxis3D {
	return NewParallelToAxis3D(l, math.V3(0, 1, 0))
}

func NewParallelToZAxis3D(l *Line3D) *ParallelToAxis3D {
	return NewParallelToAxis3D(l, math.V3(0, 0, 1))
}

// residualAD: v = [L.A.xyz, L.B.xyz]; the line direction crossed with the fixed axis
// vanishes when they are parallel.
func (c *ParallelToAxis3D) residualAD(v []ad.Number) []ad.Number {
	cr := adLine3DDir(v, 0).Cross(adConstVec3(c.Axis))
	return []ad.Number{cr.X, cr.Y, cr.Z}
}
func (c *ParallelToAxis3D) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *ParallelToAxis3D) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *ParallelToAxis3D) Variables() []*math.Scalar { return line3DVars(c.L) }

// ParallelToPlane3D forces a 3D line to be parallel to a fixed origin-frame plane (its
// direction is perpendicular to the plane normal). Use the XY/XZ/YZ constructors.
type ParallelToPlane3D struct {
	constraintBase
	L      *Line3D
	Normal math.Vector3
}

// NewParallelToPlane3D constrains l parallel to a plane with the given normal; the XY/XZ/
// YZ helpers pick the origin-frame planes.
func NewParallelToPlane3D(l *Line3D, normal math.Vector3) *ParallelToPlane3D {
	return &ParallelToPlane3D{constraintBase: newConstraint(), L: l, Normal: normal}
}

// NewParallelToXYPlane3D / XZPlane / YZPlane constrain l parallel to the named origin
// plane (its direction perpendicular to the plane normal).
func NewParallelToXYPlane3D(l *Line3D) *ParallelToPlane3D {
	return NewParallelToPlane3D(l, math.V3(0, 0, 1))
}

func NewParallelToXZPlane3D(l *Line3D) *ParallelToPlane3D {
	return NewParallelToPlane3D(l, math.V3(0, 1, 0))
}

func NewParallelToYZPlane3D(l *Line3D) *ParallelToPlane3D {
	return NewParallelToPlane3D(l, math.V3(1, 0, 0))
}

// residualAD: the line direction is perpendicular to the plane normal (dot product zero)
// iff the line is parallel to the plane.
func (c *ParallelToPlane3D) residualAD(v []ad.Number) []ad.Number {
	return []ad.Number{adLine3DDir(v, 0).Dot(adConstVec3(c.Normal))}
}
func (c *ParallelToPlane3D) Residuals() []float64      { return adResiduals(c.Variables(), c.residualAD) }
func (c *ParallelToPlane3D) Partials() [][]float64     { return adPartials(c.Variables(), c.residualAD) }
func (c *ParallelToPlane3D) Variables() []*math.Scalar { return line3DVars(c.L) }

// adLine3DDir returns a 3D line's A→B direction (B−A) from a seeded line3DVars segment at
// off (the line occupies v[off..off+5]).
func adLine3DDir(v []ad.Number, off int) ad.Vec3 {
	return adV3(v, off+3).Sub(adV3(v, off))
}
