// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// reversedCurve3 wraps a Curve3 so it is traversed from its high parameter to its low. A loop
// assembler that chains mixed-direction boundary segments needs to reverse a segment's underlying
// curve; an Arc3d is reconstructed by its three points, but a non-analytic section curve (e.g. a
// SpiricArc) has no such by-endpoints reconstruction, so it is wrapped instead. Over the same domain
// [lo,hi], PointAt(lo) returns the inner curve's PointAt(hi) and vice-versa; the tangent is negated.
type reversedCurve3 struct{ inner Curve3 }

// ReverseCurve3 returns c traversed backwards. Reversing an already-reversed curve unwraps to the
// original object (so a double reversal is identity, not a two-deep wrapper). Example:
//
//	back := geom.ReverseCurve3(section) // back.PointAt(lo) == section.PointAt(hi)
func ReverseCurve3(c Curve3) Curve3 {
	if r, ok := c.(reversedCurve3); ok {
		return r.inner
	}
	return reversedCurve3{c}
}

func (r reversedCurve3) PointAt(t float64) math.Point3 {
	lo, hi := r.inner.Domain()
	return r.inner.PointAt(lo + hi - t)
}

func (r reversedCurve3) TangentAt(t float64) math.Vector3 {
	lo, hi := r.inner.Domain()
	return r.inner.TangentAt(lo + hi - t).Scale(-1)
}

func (r reversedCurve3) Domain() (lo, hi float64) { return r.inner.Domain() }

// InnerCurve unwraps a ReverseCurve3 wrapper to the underlying curve, so a caller can dispatch on the
// CONCRETE type of a possibly-reversed rail (CN4b-2's reverseSegmentCurve: a cone-canal spring reversed at
// build time is a reversedCurve3, and the type switch must still see the spring underneath). ReverseCurve3
// never nests (it unwraps), so a single unwrap suffices; a non-reversed curve is returned unchanged.
func InnerCurve(c Curve3) Curve3 {
	if r, ok := c.(reversedCurve3); ok {
		return r.inner
	}
	return c
}
