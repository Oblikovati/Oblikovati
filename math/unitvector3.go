// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	"fmt"
	stdmath "math"
)

// UnitVector3 is an immutable 3D direction guaranteed to have length 1
// (contract: UnitVector). The invariant is established at construction by
// [NewUnitVector3]; because the type is immutable it can never drift afterwards,
// so consumers may skip re-normalizing.
type UnitVector3 struct {
	x, y, z Scalar
}

// NewUnitVector3 normalizes (x, y, z) into a UnitVector3. It returns an error
// when the input is too close to zero length to define a direction (the
// magnitude is below [DefaultTolerance]); the error reports the offending
// magnitude, per the project's exception-message convention.
func NewUnitVector3(x, y, z Scalar) (UnitVector3, error) {
	length := stdmath.Sqrt(x*x + y*y + z*z)
	if length <= DefaultTolerance {
		return UnitVector3{}, fmt.Errorf("math: cannot normalize zero-length vector (%g, %g, %g), magnitude %g <= %g", x, y, z, length, DefaultTolerance)
	}
	return UnitVector3{x / length, y / length, z / length}, nil
}

// UnitVector3FromVector normalizes v, see [NewUnitVector3].
func UnitVector3FromVector(v Vector3) (UnitVector3, error) {
	return NewUnitVector3(v.X, v.Y, v.Z)
}

// X returns the unit vector's X component.
func (u UnitVector3) X() Scalar { return u.x }

// Y returns the unit vector's Y component.
func (u UnitVector3) Y() Scalar { return u.y }

// Z returns the unit vector's Z component.
func (u UnitVector3) Z() Scalar { return u.z }

// AsVector returns the equivalent (length-1) [Vector3].
func (u UnitVector3) AsVector() Vector3 {
	return Vector3{u.x, u.y, u.z}
}

// Negate returns the opposite direction; the result is still unit length.
func (u UnitVector3) Negate() UnitVector3 {
	return UnitVector3{-u.x, -u.y, -u.z}
}

// Dot returns the dot product with o, which equals cos of the angle between
// them because both are unit length.
func (u UnitVector3) Dot(o UnitVector3) Scalar {
	return u.x*o.x + u.y*o.y + u.z*o.z
}

// AngleTo returns the unsigned angle in radians between u and o, in [0, π].
func (u UnitVector3) AngleTo(o UnitVector3) Scalar {
	return stdmath.Acos(Clamp(u.Dot(o), -1, 1))
}

// Cross returns the cross product u×o as a plain [Vector3]; it is unit length
// only when u and o are perpendicular, so the result is intentionally not a
// UnitVector3.
func (u UnitVector3) Cross(o UnitVector3) Vector3 {
	return u.AsVector().Cross(o.AsVector())
}

// IsEqualTo reports whether u and o are componentwise equal within tol. Pass
// tol <= 0 to use [DefaultTolerance].
func (u UnitVector3) IsEqualTo(o UnitVector3, tol Scalar) bool {
	t := resolveTolerance(tol)
	return approxEqual(u.x, o.x, t) && approxEqual(u.y, o.y, t) && approxEqual(u.z, o.z, t)
}

// IsParallelTo reports whether u and o lie along the same line (same or opposite
// direction) within the angular tolerance tol (radians). Pass tol <= 0 to use
// [AngleTolerance].
func (u UnitVector3) IsParallelTo(o UnitVector3, tol Scalar) bool {
	return u.AsVector().IsParallelTo(o.AsVector(), tol)
}

// IsPerpendicularTo reports whether u and o are at right angles within the
// angular tolerance tol (radians). Pass tol <= 0 to use [AngleTolerance].
func (u UnitVector3) IsPerpendicularTo(o UnitVector3, tol Scalar) bool {
	t := resolveAngleTolerance(tol)
	return stdmath.Abs(u.Dot(o)) <= t
}
