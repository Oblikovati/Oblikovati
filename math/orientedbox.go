// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// OrientedBox is a box with an arbitrary orientation in 3D (contract:
// OrientedBox): the region within HalfExtents of Center along three orthonormal
// Axes. Unlike a [Box] it can be tight around rotated geometry.
type OrientedBox struct {
	Center      Point3
	Axes        [3]UnitVector3
	HalfExtents [3]Scalar
}

// NewOrientedBox builds an oriented box from a center, three (assumed
// orthonormal) axes, and the half-extent along each.
func NewOrientedBox(center Point3, u, v, w UnitVector3, halfExtents [3]Scalar) OrientedBox {
	return OrientedBox{Center: center, Axes: [3]UnitVector3{u, v, w}, HalfExtents: halfExtents}
}

// Contains reports whether p lies within the box, by projecting the offset from
// the center onto each axis and comparing to that axis's half-extent.
func (b OrientedBox) Contains(p Point3) bool {
	d := b.Center.VectorTo(p)
	for i := 0; i < 3; i++ {
		if stdmath.Abs(d.Dot(b.Axes[i].AsVector())) > b.HalfExtents[i] {
			return false
		}
	}
	return true
}

// Corners returns the eight corners, indexed by the bits of i (bit0→Axis0,
// bit1→Axis1, bit2→Axis2) choosing the −(0)/+(1) half-extent on that axis.
func (b OrientedBox) Corners() [8]Point3 {
	var c [8]Point3
	for i := 0; i < 8; i++ {
		offset := Vector3{}
		for axis := 0; axis < 3; axis++ {
			sign := Scalar(1)
			if i&(1<<axis) == 0 {
				sign = -1
			}
			offset = offset.Add(b.Axes[axis].AsVector().Scale(sign * b.HalfExtents[axis]))
		}
		c[i] = b.Center.TranslateBy(offset)
	}
	return c
}

// ToAABB returns the axis-aligned [Box] that bounds this oriented box.
func (b OrientedBox) ToAABB() Box {
	corners := b.Corners()
	return BoxFromPoints(corners[:]...)
}
