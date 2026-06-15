// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Move operation list (M20-F20, #654). A MoveDefinition composes an ordered sequence of
// independently parametric operations rather than a single baked matrix, so a move like
// "rotate 30° about this edge, then slide 5 mm along it" round-trips as two separately
// driven operations and each one's scalar can be edited on its own.

// MoveOperationType aliases the public discriminator so existing call sites keep one
// spelling (ADR-0018: the canonical definition lives in api/types).
type MoveOperationType = types.MoveOperationType

// MoveOperation is one entry in a MoveDefinition's operation list. The active fields
// depend on Kind:
//   - MoveFreeDrag:        X, Y, Z translation offsets (length closures).
//   - MoveAlongRay:        Dir direction + Dist distance along it (length closure).
//   - MoveRotateAboutLine: AxisPoint/AxisDir axis + Angle rotation (angle closure, radians).
//
// The scalars are func() float64 so a parameter edit is re-read on the next recompute.
type MoveOperation struct {
	Kind      MoveOperationType
	X, Y, Z   func() float64
	Dir       math.Vector3
	Dist      func() float64
	AxisPoint math.Point3
	AxisDir   math.Vector3
	Angle     func() float64
}

// Matrix returns the operation's current transform, reading its parametric closures live.
// A degenerate ray/axis or an unknown kind yields the identity (a no-op) so composing the
// list is always total — a sick operation never corrupts the others.
func (o MoveOperation) Matrix() math.Matrix4 {
	switch o.Kind {
	case types.MoveFreeDrag:
		return math.Translation4(math.V3(callOrZero(o.X), callOrZero(o.Y), callOrZero(o.Z)))
	case types.MoveAlongRay:
		dir, err := math.UnitVector3FromVector(o.Dir)
		if err != nil {
			return math.Identity4()
		}
		return math.Translation4(dir.AsVector().Scale(callOrZero(o.Dist)))
	case types.MoveRotateAboutLine:
		axis, err := math.UnitVector3FromVector(o.AxisDir)
		if err != nil {
			return math.Identity4()
		}
		return math.Rotation4(callOrZero(o.Angle), axis, o.AxisPoint)
	default:
		return math.Identity4()
	}
}

// composeMoveOps multiplies the operations into a single transform applied in list order
// (operation 0 first): each step left-multiplies, so the result is opₙ·…·op₀.
func composeMoveOps(operations []MoveOperation) math.Matrix4 {
	m := math.Identity4()
	for _, op := range operations {
		m = op.Matrix().Mul(m)
	}
	return m
}

// FreeDragOp builds a free-drag operation from X/Y/Z offset closures.
func FreeDragOp(x, y, z func() float64) MoveOperation {
	return MoveOperation{Kind: types.MoveFreeDrag, X: x, Y: y, Z: z}
}

// AlongRayOp builds a move-along-ray operation from a direction and a distance closure.
func AlongRayOp(dir math.Vector3, dist func() float64) MoveOperation {
	return MoveOperation{Kind: types.MoveAlongRay, Dir: dir, Dist: dist}
}

// RotateAboutLineOp builds a rotate-about-line operation from an axis and an angle closure.
func RotateAboutLineOp(axisPoint math.Point3, axisDir math.Vector3, angle func() float64) MoveOperation {
	return MoveOperation{Kind: types.MoveRotateAboutLine, AxisPoint: axisPoint, AxisDir: axisDir, Angle: angle}
}
