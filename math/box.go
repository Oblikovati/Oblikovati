// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Box is an axis-aligned bounding box in 3D (contract: Box), the half-open
// region between Min and Max. It is the range-query primitive used throughout
// the model. The empty box (see [EmptyBox]) has Min > Max so that extending it
// with the first point yields a degenerate box at that point.
//
// Deliberate math↔types duality (audit B9, #1620): the same shape is defined a
// second time as types.Box in the Apache-2.0 api module. This is NOT accidental
// duplication — it is forced by the dependency direction (ADR-0018): the api
// module must never import this GPL module, and this kernel-facing math package
// must not depend on the api's serialization concerns (json tags, the wire
// Point). So each side owns a Box in its own value type (math.Point3 here,
// types.Point there). The two are kept in lockstep by hand; a field added to one
// MUST be mirrored on the other. Cross-reference: keep this in sync with
// types.Box in Oblikovati.API/types/geom_box.go.
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

// IntersectsRay reports whether the forward ray (origin + t·dir, t ≥ 0) crosses the box,
// returning the entry parameter tEnter (0 when the origin is inside). It is the slab test:
// for each axis the ray is clipped to the box's [Min,Max] span, and the surviving t-interval
// is the box crossing. dir need not be unit; an axis-parallel ray (zero component) is handled
// by treating that slab as the whole line unless the origin is already outside it. This is the
// broad-phase test a spatial index uses to reject placements a pick ray cannot hit (M34-F5).
//
//	if t, ok := box.IntersectsRay(origin, dir); ok { /* candidate at depth t */ }
func (b Box) IntersectsRay(origin Point3, dir Vector3) (tEnter Scalar, ok bool) {
	if b.IsEmpty() {
		return 0, false
	}
	tMin, tMax := stdmath.Inf(-1), stdmath.Inf(1)
	axes := [3]struct{ o, d, lo, hi Scalar }{
		{origin.X, dir.X, b.Min.X, b.Max.X},
		{origin.Y, dir.Y, b.Min.Y, b.Max.Y},
		{origin.Z, dir.Z, b.Min.Z, b.Max.Z},
	}
	for _, a := range axes {
		if a.d == 0 { // parallel to this slab: a miss only if the origin is outside it
			if a.o < a.lo || a.o > a.hi {
				return 0, false
			}
			continue
		}
		t1, t2 := (a.lo-a.o)/a.d, (a.hi-a.o)/a.d
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tMin, tMax = max(tMin, t1), min(tMax, t2)
		if tMin > tMax {
			return 0, false
		}
	}
	if tMax < 0 { // the whole crossing is behind the ray origin
		return 0, false
	}
	return max(tMin, 0), true
}

// Transform returns the axis-aligned box bounding this box's eight corners after the
// affine transform m — the AABB of a placed (rotated/translated) component, used to
// accumulate an assembly's range box from its occurrences. A rotation enlarges the
// AABB, which is correct (the tight box of a rotated box is not axis-aligned). The
// empty box is returned unchanged: it has no corners to place.
func (b Box) Transform(m Matrix4) Box {
	if b.IsEmpty() {
		return b
	}
	out := EmptyBox()
	for _, c := range b.Corners() {
		out = out.ExtendPoint(m.TransformPoint(c))
	}
	return out
}
