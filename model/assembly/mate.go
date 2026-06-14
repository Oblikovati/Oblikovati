// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/solve"
)

// MateConstraint makes its two geometry inputs coincident: planes opposed (the default
// mate) or aligned, axes collinear, or points coincident, at an optional offset. It is
// the workhorse positioning constraint.
type MateConstraint struct {
	*constraintBase
	offset   float64
	solution types.MateConstraintSolutionType
}

// Value returns the mate offset.
func (m *MateConstraint) Value() float64 { return m.offset }

// SolutionType returns the directed sense (opposed/aligned) the solver enforces.
func (m *MateConstraint) SolutionType() types.MateConstraintSolutionType { return m.solution }

// bind returns the mate's residual source over the two anchors' live placements.
func (m *MateConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := m.boundPlacements(b)
		return mateResiduals(m.a.prim, m.b.prim, pa.matrix(), pb.matrix(), m.offset, m.solution)
	})
}

// FlushConstraint makes two faces co-planar with aligned normals at an offset — a mate
// whose solution is fixed to aligned.
type FlushConstraint struct {
	*constraintBase
	offset float64
}

// Value returns the flush offset.
func (f *FlushConstraint) Value() float64 { return f.offset }

// bind returns the flush's residual source (a plane mate with aligned normals).
func (f *FlushConstraint) bind(b binder) []solve.Residual {
	return single(func() []float64 {
		pa, pb := f.boundPlacements(b)
		return mateResiduals(f.a.prim, f.b.prim, pa.matrix(), pb.matrix(), f.offset, types.MateSolutionAligned)
	})
}

// mateResiduals dispatches the residual math on the two inputs' kinds: plane-plane,
// axis-axis, point-on-plane, or point-point coincidence.
func mateResiduals(a, b Primitive, ma, mb math.Matrix4, offset float64, sol types.MateConstraintSolutionType) []float64 {
	switch {
	case a.kind == planeKind && b.kind == planeKind:
		return planeMateResiduals(a, b, ma, mb, offset, sol)
	case a.kind == lineKind && b.kind == lineKind:
		return axisMateResiduals(a, b, ma, mb, offset, sol)
	case a.kind == planeKind:
		return []float64{gapResidual(worldPoint(ma, a), worldPoint(mb, b), worldDir(ma, a), offset)}
	case b.kind == planeKind:
		return []float64{gapResidual(worldPoint(mb, b), worldPoint(ma, a), worldDir(mb, b), offset)}
	default:
		return pointMateResiduals(a, b, ma, mb)
	}
}

// planeMateResiduals aligns plane A's normal opposed (or aligned) to plane B's and holds
// the two planes the offset apart along B's normal: two rotational residuals + one gap.
func planeMateResiduals(a, b Primitive, ma, mb math.Matrix4, offset float64, sol types.MateConstraintSolutionType) []float64 {
	nA, nB := worldDir(ma, a), worldDir(mb, b)
	target := nB.Scale(-1) // opposed: A's normal points against B's
	if sol == types.MateSolutionAligned {
		target = nB
	}
	res := alignResiduals(nA, target)
	return append(res, gapResidual(worldPoint(ma, a), worldPoint(mb, b), nB, offset))
}

// axisMateResiduals makes two axes collinear: two rotational residuals aligning the
// directions plus two perpendicular-offset residuals pinning the lines together. Axes
// carry no inherent sense, so the warm-started branch picks parallel vs anti-parallel.
func axisMateResiduals(a, b Primitive, ma, mb math.Matrix4, _ float64, _ types.MateConstraintSolutionType) []float64 {
	dA, dB := worldDir(ma, a), worldDir(mb, b)
	res := alignResiduals(dA, dB)
	return append(res, perpDistanceResiduals(worldPoint(ma, a), worldPoint(mb, b), dB)...)
}

// pointMateResiduals makes two points coincident (three residuals).
func pointMateResiduals(a, b Primitive, ma, mb math.Matrix4) []float64 {
	d := worldPoint(mb, b).VectorTo(worldPoint(ma, a))
	return []float64{d.X, d.Y, d.Z}
}
