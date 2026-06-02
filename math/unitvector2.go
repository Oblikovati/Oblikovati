// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	"fmt"
	stdmath "math"
)

// UnitVector2 is an immutable 2D direction guaranteed to have length 1
// (contract: UnitVector2d). The invariant is established by [NewUnitVector2]
// and, because the type is immutable, can never drift.
type UnitVector2 struct {
	x, y Scalar
}

// NewUnitVector2 normalizes (x, y) into a UnitVector2. It returns an error when
// the input is below [DefaultTolerance] in magnitude (no defined direction),
// reporting the offending magnitude per the exception-message convention.
func NewUnitVector2(x, y Scalar) (UnitVector2, error) {
	length := stdmath.Sqrt(x*x + y*y)
	if length <= DefaultTolerance {
		return UnitVector2{}, fmt.Errorf("math: cannot normalize zero-length vector (%g, %g), magnitude %g <= %g", x, y, length, DefaultTolerance)
	}
	return UnitVector2{x / length, y / length}, nil
}

// UnitVector2FromVector normalizes v, see [NewUnitVector2].
func UnitVector2FromVector(v Vector2) (UnitVector2, error) {
	return NewUnitVector2(v.X, v.Y)
}

// X returns the unit vector's X component.
func (u UnitVector2) X() Scalar { return u.x }

// Y returns the unit vector's Y component.
func (u UnitVector2) Y() Scalar { return u.y }

// AsVector returns the equivalent (length-1) [Vector2].
func (u UnitVector2) AsVector() Vector2 {
	return Vector2{u.x, u.y}
}

// Negate returns the opposite direction; still unit length.
func (u UnitVector2) Negate() UnitVector2 {
	return UnitVector2{-u.x, -u.y}
}

// Dot returns the dot product with o (equals cos of the angle between them).
func (u UnitVector2) Dot(o UnitVector2) Scalar {
	return u.x*o.x + u.y*o.y
}

// AngleTo returns the unsigned angle in radians between u and o, in [0, π].
func (u UnitVector2) AngleTo(o UnitVector2) Scalar {
	return stdmath.Acos(clampUnit(u.Dot(o)))
}

// IsEqualTo reports whether u and o are componentwise equal within tol. Pass
// tol <= 0 to use [DefaultTolerance].
func (u UnitVector2) IsEqualTo(o UnitVector2, tol Scalar) bool {
	t := resolveTolerance(tol)
	return approxEqual(u.x, o.x, t) && approxEqual(u.y, o.y, t)
}

// IsParallelTo reports whether u and o lie along the same line within the
// angular tolerance tol (radians). Pass tol <= 0 to use [AngleTolerance].
func (u UnitVector2) IsParallelTo(o UnitVector2, tol Scalar) bool {
	return u.AsVector().IsParallelTo(o.AsVector(), tol)
}

// IsPerpendicularTo reports whether u and o are at right angles within the
// angular tolerance tol (radians). Pass tol <= 0 to use [AngleTolerance].
func (u UnitVector2) IsPerpendicularTo(o UnitVector2, tol Scalar) bool {
	t := resolveAngleTolerance(tol)
	return stdmath.Abs(u.Dot(o)) <= t
}
