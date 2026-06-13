// SPDX-License-Identifier: GPL-2.0-only

package proxy

import "oblikovati.org/math"

// The In*Context helpers report a proxied entity's geometry in assembly space. They
// are free generic functions rather than methods because Go methods cannot add a type
// constraint of their own: each is constrained to the capability the geometry needs
// (a range box, a point, a direction), so it applies to exactly the native entities
// that expose it — one transform path per geometry kind, shared by every entity that
// has it.

// Boxed is a definition-space entity that reports an axis-aligned range box. Topo
// faces, edges, vertices, and bodies all satisfy it.
type Boxed interface {
	RangeBox() math.Box
}

// RangeBoxInContext returns the proxied entity's range box in assembly space: its
// definition-space box transformed by the context. The native's own box is unchanged
// (the proxy is a view, not a mutation).
func RangeBoxInContext[E Boxed](p Proxy[E]) math.Box {
	return p.native.RangeBox().Transform(p.context.transform)
}

// Pointed is a definition-space entity that reports a single point — a vertex or a
// work point.
type Pointed interface {
	Point() math.Point3
}

// PointInContext returns the proxied entity's point in assembly space.
func PointInContext[E Pointed](p Proxy[E]) math.Point3 {
	return p.context.transform.TransformPoint(p.native.Point())
}

// Directed is a definition-space entity that reports a unit direction — a work axis.
type Directed interface {
	Direction() math.UnitVector3
}

// DirectionInContext returns the proxied entity's direction in assembly space, or
// false if the context transform collapses it (only possible under a degenerate,
// non-rigid transform; occurrence placements are rigid, so true in practice).
func DirectionInContext[E Directed](p Proxy[E]) (math.UnitVector3, bool) {
	d, err := p.context.transform.TransformUnitVector(p.native.Direction())
	return d, err == nil
}
