// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Box2d is an axis-aligned bounding box in 2D (contract: Box2d), the analogue of
// [Box] for sketch space.
type Box2d struct {
	Min, Max Point2
}

// NewBox2d returns the box spanning the two corners, normalizing the bounds.
func NewBox2d(a, b Point2) Box2d {
	return Box2d{
		Min: Point2{min(a.X, b.X), min(a.Y, b.Y)},
		Max: Point2{max(a.X, b.X), max(a.Y, b.Y)},
	}
}

// EmptyBox2d returns the union identity box (Min > Max).
func EmptyBox2d() Box2d {
	inf := stdmath.Inf(1)
	return Box2d{Min: Point2{inf, inf}, Max: Point2{-inf, -inf}}
}

// Box2dFromPoints returns the smallest box containing every point.
func Box2dFromPoints(points ...Point2) Box2d {
	b := EmptyBox2d()
	for _, p := range points {
		b = b.ExtendPoint(p)
	}
	return b
}

// IsEmpty reports whether the box contains no points.
func (b Box2d) IsEmpty() bool {
	return b.Min.X > b.Max.X || b.Min.Y > b.Max.Y
}

// Center returns the midpoint of the box.
func (b Box2d) Center() Point2 {
	return b.Min.Midpoint(b.Max)
}

// Diagonal returns the Max−Min vector; its components are the side lengths.
func (b Box2d) Diagonal() Vector2 {
	return b.Min.VectorTo(b.Max)
}

// Area returns the enclosed area, or 0 when empty.
func (b Box2d) Area() Scalar {
	if b.IsEmpty() {
		return 0
	}
	d := b.Diagonal()
	return d.X * d.Y
}

// Contains reports whether p lies within the closed box.
func (b Box2d) Contains(p Point2) bool {
	return p.X >= b.Min.X && p.X <= b.Max.X && p.Y >= b.Min.Y && p.Y <= b.Max.Y
}

// Intersects reports whether this box and o overlap (touching edges count).
func (b Box2d) Intersects(o Box2d) bool {
	return b.Min.X <= o.Max.X && b.Max.X >= o.Min.X &&
		b.Min.Y <= o.Max.Y && b.Max.Y >= o.Min.Y
}

// ExtendPoint returns the smallest box containing this box and p.
func (b Box2d) ExtendPoint(p Point2) Box2d {
	return Box2d{
		Min: Point2{min(b.Min.X, p.X), min(b.Min.Y, p.Y)},
		Max: Point2{max(b.Max.X, p.X), max(b.Max.Y, p.Y)},
	}
}

// Union returns the smallest box containing both boxes.
func (b Box2d) Union(o Box2d) Box2d {
	return b.ExtendPoint(o.Min).ExtendPoint(o.Max)
}
