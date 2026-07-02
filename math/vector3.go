// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Vector3 is an immutable 3D displacement/direction (contract: Vector). Unlike
// a [Point3] it has no position; transforming it by a [Matrix4] ignores the
// translation component.
type Vector3 struct {
	X, Y, Z Scalar
}

// V3 constructs a Vector3. Example: d := V3(0, 0, 1) // +Z direction.
func V3(x, y, z Scalar) Vector3 {
	return Vector3{X: x, Y: y, Z: z}
}

// Add returns v + o.
func (v Vector3) Add(o Vector3) Vector3 {
	return Vector3{v.X + o.X, v.Y + o.Y, v.Z + o.Z}
}

// Sub returns v - o.
func (v Vector3) Sub(o Vector3) Vector3 {
	return Vector3{v.X - o.X, v.Y - o.Y, v.Z - o.Z}
}

// Scale returns v multiplied by the scalar s.
func (v Vector3) Scale(s Scalar) Vector3 {
	return Vector3{v.X * s, v.Y * s, v.Z * s}
}

// Negate returns -v.
func (v Vector3) Negate() Vector3 {
	return Vector3{-v.X, -v.Y, -v.Z}
}

// Dot returns the dot product v·o.
func (v Vector3) Dot(o Vector3) Scalar {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

// Cross returns the cross product v×o.
func (v Vector3) Cross(o Vector3) Vector3 {
	return Vector3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

// LengthSquared returns |v|², avoiding the square root when only comparisons
// are needed.
func (v Vector3) LengthSquared() Scalar {
	return v.Dot(v)
}

// Length returns the Euclidean magnitude |v|.
func (v Vector3) Length() Scalar {
	return stdmath.Sqrt(v.LengthSquared())
}

// AsPoint reinterprets the displacement as a position from the origin.
func (v Vector3) AsPoint() Point3 {
	return Point3(v)
}

// AngleTo returns the unsigned angle in radians between v and o, in [0, π].
// Returns 0 when either vector is zero-length (no defined direction).
func (v Vector3) AngleTo(o Vector3) Scalar {
	denom := v.Length() * o.Length()
	if denom == 0 {
		return 0
	}
	// clampUnit guards against |cos| slightly exceeding 1 from rounding.
	return stdmath.Acos(Clamp(v.Dot(o)/denom, -1, 1))
}

// IsEqualTo reports whether v and o are componentwise equal within tol. Pass
// tol <= 0 to use [DefaultTolerance].
func (v Vector3) IsEqualTo(o Vector3, tol Scalar) bool {
	t := resolveTolerance(tol)
	return approxEqual(v.X, o.X, t) && approxEqual(v.Y, o.Y, t) && approxEqual(v.Z, o.Z, t)
}

// IsParallelTo reports whether v and o point along the same line (same or
// opposite direction) within the angular tolerance tol (radians). Pass tol <= 0
// to use [AngleTolerance].
func (v Vector3) IsParallelTo(o Vector3, tol Scalar) bool {
	denom := v.LengthSquared() * o.LengthSquared()
	if denom == 0 {
		return false
	}
	// |v×o|² / (|v|²|o|²) = sin²θ; parallel ⇔ sinθ ≈ 0.
	t := resolveAngleTolerance(tol)
	return v.Cross(o).LengthSquared()/denom <= t*t
}

// IsPerpendicularTo reports whether v and o are at right angles within the
// angular tolerance tol (radians). Pass tol <= 0 to use [AngleTolerance].
func (v Vector3) IsPerpendicularTo(o Vector3, tol Scalar) bool {
	denom := v.Length() * o.Length()
	if denom == 0 {
		return false
	}
	// v·o / (|v||o|) = cosθ; perpendicular ⇔ cosθ ≈ 0.
	t := resolveAngleTolerance(tol)
	return stdmath.Abs(v.Dot(o))/denom <= t
}
