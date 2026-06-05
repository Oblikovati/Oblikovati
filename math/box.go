// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Box is an axis-aligned bounding box in 3D (contract: Box), the half-open
// region between Min and Max. It is the range-query primitive used throughout
// the model. The empty box (see [EmptyBox]) has Min > Max so that extending it
// with the first point yields a degenerate box at that point.
type Box struct {
	Min, Max Point3
}

// NewBox returns the box spanning the two corners, normalizing so Min holds the
// per-axis minimum and Max the maximum regardless of the argument order.
func NewBox(a, b Point3) Box {
	return Box{
		Min: Point3{min(a.X, b.X), min(a.Y, b.Y), min(a.Z, b.Z)},
		Max: Point3{max(a.X, b.X), max(a.Y, b.Y), max(a.Z, b.Z)},
	}
}

// EmptyBox returns the identity box for union: it contains nothing and extends
// to exactly the first point/box it is combined with.
func EmptyBox() Box {
	inf := stdmath.Inf(1)
	return Box{Min: Point3{inf, inf, inf}, Max: Point3{-inf, -inf, -inf}}
}

// BoxFromPoints returns the smallest box containing every point, or the empty
// box when none are given.
func BoxFromPoints(points ...Point3) Box {
	b := EmptyBox()
	for _, p := range points {
		b = b.ExtendPoint(p)
	}
	return b
}

// IsEmpty reports whether the box contains no points (any axis has Min > Max).
func (b Box) IsEmpty() bool {
	return b.Min.X > b.Max.X || b.Min.Y > b.Max.Y || b.Min.Z > b.Max.Z
}

// Center returns the midpoint of the box.
func (b Box) Center() Point3 {
	return b.Min.Midpoint(b.Max)
}

// Diagonal returns the Max−Min vector; its components are the side lengths.
func (b Box) Diagonal() Vector3 {
	return b.Min.VectorTo(b.Max)
}

// Volume returns the enclosed volume, or 0 when empty.
func (b Box) Volume() Scalar {
	if b.IsEmpty() {
		return 0
	}
	d := b.Diagonal()
	return d.X * d.Y * d.Z
}

// Contains reports whether p lies within the closed box.
func (b Box) Contains(p Point3) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X &&
		p.Y >= b.Min.Y && p.Y <= b.Max.Y &&
		p.Z >= b.Min.Z && p.Z <= b.Max.Z
}

// ContainsBox reports whether o is wholly inside this box.
func (b Box) ContainsBox(o Box) bool {
	return b.Contains(o.Min) && b.Contains(o.Max)
}

// Intersects reports whether this box and o overlap (touching faces count).
func (b Box) Intersects(o Box) bool {
	return b.Min.X <= o.Max.X && b.Max.X >= o.Min.X &&
		b.Min.Y <= o.Max.Y && b.Max.Y >= o.Min.Y &&
		b.Min.Z <= o.Max.Z && b.Max.Z >= o.Min.Z
}

// ExtendPoint returns the smallest box containing this box and p.
func (b Box) ExtendPoint(p Point3) Box {
	return Box{
		Min: Point3{min(b.Min.X, p.X), min(b.Min.Y, p.Y), min(b.Min.Z, p.Z)},
		Max: Point3{max(b.Max.X, p.X), max(b.Max.Y, p.Y), max(b.Max.Z, p.Z)},
	}
}

// Union returns the smallest box containing both boxes.
func (b Box) Union(o Box) Box {
	return b.ExtendPoint(o.Min).ExtendPoint(o.Max)
}

// FarthestPoint returns the box corner farthest along dir — the box's support point in
// that direction (Inventor's TransientGeometry.GetFarmostPoint). On each axis it takes Max
// when dir's component is non-negative and Min when it is negative; dir need not be unit.
// On the empty box the result is degenerate (Min>Max), like the box itself.
func (b Box) FarthestPoint(dir Vector3) Point3 {
	return Point3{farAxis(b.Min.X, b.Max.X, dir.X), farAxis(b.Min.Y, b.Max.Y, dir.Y), farAxis(b.Min.Z, b.Max.Z, dir.Z)}
}

// farAxis picks the box extent (lo or hi) farther along a single axis: hi when the
// direction component points that way (>= 0), lo otherwise.
func farAxis(lo, hi, d Scalar) Scalar {
	if d < 0 {
		return lo
	}
	return hi
}

// Corners returns the eight corner points, ordered by the bits of the index:
// bit0=X, bit1=Y, bit2=Z choosing Min(0)/Max(1) on that axis.
func (b Box) Corners() [8]Point3 {
	var c [8]Point3
	xs := [2]Scalar{b.Min.X, b.Max.X}
	ys := [2]Scalar{b.Min.Y, b.Max.Y}
	zs := [2]Scalar{b.Min.Z, b.Max.Z}
	for i := 0; i < 8; i++ {
		c[i] = Point3{xs[i&1], ys[(i>>1)&1], zs[(i>>2)&1]}
	}
	return c
}
