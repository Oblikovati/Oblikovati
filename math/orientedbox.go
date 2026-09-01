// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// OrientedBox is a box with an arbitrary orientation in 3D (contract:
// OrientedBox): the region within HalfExtents of Center along three orthonormal
// Axes. Unlike a [Box] it can be tight around rotated geometry. This is the ONE
// in-module oriented-box value type — kernel/query.OrientedMinimumRangeBox returns
// it, and the router serializes it via [OrientedBox.MinCorner]/[OrientedBox.EdgeVectors]
// (audit B9, #1620: the former kernel/ops.OrientedBox corner+edge-vectors twin was
// collapsed into this center+axes form).
//
// Deliberate math↔types duality (audit B9, #1620): the same concept is defined a
// second time as types.OrientedBox in the Apache-2.0 api module, in the reference
// corner+axes+extents form. This is NOT accidental duplication — the dependency
// direction (ADR-0018) forbids the api module from importing this GPL math package
// and forbids this package from taking on the api's serialization concerns, so
// each side owns its own value type. The two are kept in lockstep by hand; a field
// added to one MUST be mirrored on the other. Cross-reference: keep this in sync
// with types.OrientedBox in Oblikovati.API/types/geom_box.go.
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
	for i := range 3 {
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
	for i := range 8 {
		offset := Vector3{}
		for axis := range 3 {
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

// Volume returns the box volume — the product of the three full edge lengths
// (each axis's full extent is twice its half-extent).
func (b OrientedBox) Volume() Scalar {
	return 8 * b.HalfExtents[0] * b.HalfExtents[1] * b.HalfExtents[2]
}

// MinCorner returns the corner at the negative end of every axis — the origin of
// the reference corner+edge-vectors form (the inverse of [NewOrientedBoxFromEdges]).
func (b OrientedBox) MinCorner() Point3 {
	c := b.Center
	for i := range 3 {
		c = c.TranslateBy(b.Axes[i].AsVector().Scale(-b.HalfExtents[i]))
	}
	return c
}

// EdgeVectors returns the three full-length edge vectors spanning from [MinCorner]
// (each axis scaled by its full extent) — the corner+edges form the reference
// OrientedBox and the wire DTO use.
func (b OrientedBox) EdgeVectors() [3]Vector3 {
	var e [3]Vector3
	for i := range 3 {
		e[i] = b.Axes[i].AsVector().Scale(2 * b.HalfExtents[i])
	}
	return e
}
