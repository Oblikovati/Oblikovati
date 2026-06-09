// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// This file holds the line/point-based 3D geometric constraints (M22-F05) that extend the
// point-only set in constraints_3d.go: parallel, perpendicular, midpoint, ground, and the
// orientation-to-origin-frame constraints (parallel-to-X/Y/Z-axis, parallel-to-XY/XZ/YZ-
// plane). Each implements [Constraint] over the dimension-agnostic solver — a line's
// direction residual reads its two endpoints' nine coordinates by pointer.

// line3DDir returns the current A→B direction components of a 3D line.
func line3DDir(l *Line3D) (dx, dy, dz float64) {
	return float64(l.B.X - l.A.X), float64(l.B.Y - l.A.Y), float64(l.B.Z - l.A.Z)
}

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

func (c *Parallel3D) Residuals() []float64 {
	ax, ay, az := line3DDir(c.L1)
	bx, by, bz := line3DDir(c.L2)
	return cross3(ax, ay, az, bx, by, bz)
}

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

func (c *Perpendicular3D) Residuals() []float64 {
	ax, ay, az := line3DDir(c.L1)
	bx, by, bz := line3DDir(c.L2)
	return []float64{ax*bx + ay*by + az*bz}
}

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

func (c *Midpoint3D) Residuals() []float64 {
	return []float64{
		float64(c.P.X - (c.L.A.X+c.L.B.X)/2),
		float64(c.P.Y - (c.L.A.Y+c.L.B.Y)/2),
		float64(c.P.Z - (c.L.A.Z+c.L.B.Z)/2),
	}
}

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

func (c *Ground3D) Residuals() []float64 {
	return []float64{
		float64(c.P.X - c.Target.X),
		float64(c.P.Y - c.Target.Y),
		float64(c.P.Z - c.Target.Z),
	}
}

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

func (c *ParallelToAxis3D) Residuals() []float64 {
	dx, dy, dz := line3DDir(c.L)
	return cross3(dx, dy, dz, float64(c.Axis.X), float64(c.Axis.Y), float64(c.Axis.Z))
}

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

func (c *ParallelToPlane3D) Residuals() []float64 {
	dx, dy, dz := line3DDir(c.L)
	return []float64{dx*float64(c.Normal.X) + dy*float64(c.Normal.Y) + dz*float64(c.Normal.Z)}
}

func (c *ParallelToPlane3D) Variables() []*math.Scalar { return line3DVars(c.L) }

// cross3 returns the three components of the cross product of two free vectors.
func cross3(ax, ay, az, bx, by, bz float64) []float64 {
	return []float64{ay*bz - az*by, az*bx - ax*bz, ax*by - ay*bx}
}
