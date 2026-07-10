// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Ellipse orientation as a solver DOF (#1879 AC2). An ellipse's major-axis direction is a
// genuine rotational degree of freedom — a sketched ellipse can spin about its centre until
// constrained — so making its axis horizontal/vertical (or aligning it to a line) requires the
// solver to rotate it. We model that one DOF as a scalar angle (orientation) rather than the two
// components of MajorAxis: two free components would spuriously add a magnitude DOF and let the
// solver collapse the axis to a degenerate zero vector. MajorAxis stays the authoritative
// direction outside a solve; variables() seeds orientation from it and Solve writes it back, so
// every existing MajorAxis reader/writer is untouched.

// axisAngle exposes the orientation DOF pointer for the constraint solver (Ellipse and
// EllipticalArc both carry one).
func (e *Ellipse) axisAngle() *math.Scalar       { return &e.orientation }
func (e *EllipticalArc) axisAngle() *math.Scalar { return &e.orientation }

// seedOrientation refreshes the orientation angle from the current MajorAxis, so the DOF the
// solver reads always matches the authoritative direction (idempotent).
func (e *Ellipse) seedOrientation()       { e.orientation = axisToAngle(e.MajorAxis) }
func (e *EllipticalArc) seedOrientation() { e.orientation = axisToAngle(e.MajorAxis) }

// syncAxisFromOrientation rewrites MajorAxis from the (possibly solver-moved) orientation,
// preserving the vector's length so only its direction changes.
func (e *Ellipse) syncAxisFromOrientation() { e.MajorAxis = angleToAxis(e.orientation, e.MajorAxis) }
func (e *EllipticalArc) syncAxisFromOrientation() {
	e.MajorAxis = angleToAxis(e.orientation, e.MajorAxis)
}

// axisToAngle is the major-axis direction's angle from +X (0 for a degenerate zero axis).
func axisToAngle(axis math.Vector2) math.Scalar {
	return stdmath.Atan2(float64(axis.Y), float64(axis.X))
}

// angleToAxis rebuilds an axis vector at the given angle, keeping the reference vector's length
// (a zero-length reference is left as a unit vector rather than collapsing to the origin).
func angleToAxis(angle math.Scalar, ref math.Vector2) math.Vector2 {
	length := ref.Length()
	if length == 0 {
		length = 1
	}
	return math.V2(length*stdmath.Cos(float64(angle)), length*stdmath.Sin(float64(angle)))
}
