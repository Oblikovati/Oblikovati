// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// TrimmedCurve3 restricts a base [Curve3] to the sub-parameter interval [Lo, Hi] of the base's
// OWN domain and re-presents that sub-arc over the canonical unit domain [0,1]. It is the generic
// (any-Curve3) analogue of [SplitCurve], which is BSplineCurve-only: a canal-blend boundary
// sub-edge must carry ONLY the little rail span between two consecutive samples, not the whole
// rail. Every normal kernel edge already obeys that convention (its curve's domain spans exactly
// that edge — curveSpan==vGap); a canal sub-edge carrying the whole rail makes the shared
// tessellator sweep the entire rail once per sub-edge → a self-overlapping loop that folds/half-
// covers the mesh (kernel/ops canal faces; N7 tessellation defect,
// .superpowers/sdd/n7-tessellation-diagnosis.md).
//
// Orientation is baked in through Lo/Hi: Lo maps to domain 0 and Hi to domain 1, so passing Lo>Hi
// presents the base curve REVERSED over the sub-span. Domain() is always [0,1] so a consumer's
// adaptive sampler always walks an increasing range regardless of direction.
//
// Example: sub := TrimmedCurve3{Base: rail, Lo: 0.5, Hi: 2.0/3.0} // rail restricted to that 1/6 span.
type TrimmedCurve3 struct {
	Base   Curve3
	Lo, Hi float64
}

var _ Curve3 = TrimmedCurve3{}

// Domain reports the canonical unit domain — the restriction always re-presents its sub-arc over
// [0,1], so Lo/Hi ordering (forward or reversed) never surfaces as a decreasing range.
func (t TrimmedCurve3) Domain() (lo, hi float64) { return 0, 1 }

// PointAt maps the unit parameter s∈[0,1] affinely onto the base sub-interval [Lo, Hi].
func (t TrimmedCurve3) PointAt(s float64) math.Point3 {
	return t.Base.PointAt(t.Lo + s*(t.Hi-t.Lo))
}

// TangentAt applies the chain rule dP/ds = (dP/dt)·(Hi-Lo). For a reversed sub-span (Lo>Hi) the
// factor is negative, so the tangent points along the 0→1 (reversed) direction, consistent with
// PointAt.
func (t TrimmedCurve3) TangentAt(s float64) math.Vector3 {
	return t.Base.TangentAt(t.Lo + s*(t.Hi-t.Lo)).Scale(t.Hi - t.Lo)
}
