// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Vector2 is an immutable 2D displacement/direction (contract: Vector2d), used
// in sketch space. It transforms by the linear part of a [Matrix3], ignoring
// translation.
type Vector2 struct {
	X, Y Scalar
}

// V2 constructs a Vector2. Example: right := V2(1, 0).
func V2(x, y Scalar) Vector2 {
	return Vector2{X: x, Y: y}
}

// Add returns v + o.
func (v Vector2) Add(o Vector2) Vector2 {
	return Vector2{v.X + o.X, v.Y + o.Y}
}

// Sub returns v - o.
func (v Vector2) Sub(o Vector2) Vector2 {
	return Vector2{v.X - o.X, v.Y - o.Y}
}

// Scale returns v multiplied by the scalar s.
func (v Vector2) Scale(s Scalar) Vector2 {
	return Vector2{v.X * s, v.Y * s}
}

// Negate returns -v.
func (v Vector2) Negate() Vector2 {
	return Vector2{-v.X, -v.Y}
}

// Dot returns the dot product v·o.
func (v Vector2) Dot(o Vector2) Scalar {
	return v.X*o.X + v.Y*o.Y
}

// Cross returns the scalar 2D cross product (the signed area of the
// parallelogram, i.e. the Z of the 3D cross). Positive means o is
// counter-clockwise from v.
func (v Vector2) Cross(o Vector2) Scalar {
	return v.X*o.Y - v.Y*o.X
}

// LengthSquared returns |v|².
func (v Vector2) LengthSquared() Scalar {
	return v.Dot(v)
}

// Length returns the Euclidean magnitude |v|.
func (v Vector2) Length() Scalar {
	return stdmath.Sqrt(v.LengthSquared())
}

// AsPoint reinterprets the displacement as a position from the origin.
func (v Vector2) AsPoint() Point2 {
	return Point2(v)
}

// AngleTo returns the unsigned angle in radians between v and o, in [0, π].
// Returns 0 when either vector is zero-length.
func (v Vector2) AngleTo(o Vector2) Scalar {
	denom := v.Length() * o.Length()
	if denom == 0 {
		return 0
	}
	return stdmath.Acos(Clamp(v.Dot(o)/denom, -1, 1))
}

// IsEqualTo reports whether v and o are componentwise equal within tol. Pass
// tol <= 0 to use [DefaultTolerance].
func (v Vector2) IsEqualTo(o Vector2, tol Scalar) bool {
	t := resolveTolerance(tol)
	return approxEqual(v.X, o.X, t) && approxEqual(v.Y, o.Y, t)
}

// IsParallelTo reports whether v and o lie along the same line within the
// angular tolerance tol (radians). Pass tol <= 0 to use [AngleTolerance].
func (v Vector2) IsParallelTo(o Vector2, tol Scalar) bool {
	denom := v.LengthSquared() * o.LengthSquared()
	if denom == 0 {
		return false
	}
	cross := v.Cross(o)
	t := resolveAngleTolerance(tol)
	return cross*cross/denom <= t*t
}

// IsPerpendicularTo reports whether v and o are at right angles within the
// angular tolerance tol (radians). Pass tol <= 0 to use [AngleTolerance].
func (v Vector2) IsPerpendicularTo(o Vector2, tol Scalar) bool {
	denom := v.Length() * o.Length()
	if denom == 0 {
		return false
	}
	t := resolveAngleTolerance(tol)
	return stdmath.Abs(v.Dot(o))/denom <= t
}
