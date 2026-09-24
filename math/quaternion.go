// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Quaternion is an orientation in 3D space, carried as the four components of a
// (usually unit) quaternion w + xi + yj + zk. It is the assembly solver's rotation
// parameterization (ADR-0011): six rigid-body DOF per occurrence are three position
// scalars plus this quaternion's four components, kept unit by a normalization residual.
// Quaternions avoid the gimbal singularities of Euler angles and give smooth,
// branch-free Jacobians for the orientation variables.
type Quaternion struct {
	W, X, Y, Z Scalar
}

// QuaternionIdentity is the no-rotation orientation.
func QuaternionIdentity() Quaternion { return Quaternion{W: 1} }

// QuaternionFromAxisAngle returns the rotation of angle radians about axis (the
// right-hand rule). The result is unit length.
//
//	q := QuaternionFromAxisAngle(zAxis, math.Pi/2) // 90° about +Z
func QuaternionFromAxisAngle(axis UnitVector3, angle Scalar) Quaternion {
	half := Scalar(angle / 2)
	s := stdmath.Sin(half)
	return Quaternion{
		W: stdmath.Cos(half),
		X: Scalar(axis.X() * s),
		Y: Scalar(axis.Y() * s),
		Z: Scalar(axis.Z() * s),
	}
}

// LengthSquared returns w²+x²+y²+z².
func (q Quaternion) LengthSquared() Scalar {
	return Scalar(q.W*q.W) + Scalar(q.X*q.X) + Scalar(q.Y*q.Y) + Scalar(q.Z*q.Z)
}

// Length returns the quaternion's magnitude.
func (q Quaternion) Length() Scalar { return stdmath.Sqrt(q.LengthSquared()) }

// Normalize returns the unit quaternion in q's direction. A (near-)zero quaternion —
// which encodes no orientation — normalizes to the identity rather than producing NaNs.
func (q Quaternion) Normalize() Quaternion {
	n := q.Length()
	if n < 1e-12 {
		return QuaternionIdentity()
	}
	return Quaternion{Scalar(q.W / n), Scalar(q.X / n), Scalar(q.Y / n), Scalar(q.Z / n)}
}

// Mul returns the Hamilton product q·o: the rotation that applies o first, then q.
func (q Quaternion) Mul(o Quaternion) Quaternion {
	return Quaternion{
		W: Scalar(q.W*o.W) - Scalar(q.X*o.X) - Scalar(q.Y*o.Y) - Scalar(q.Z*o.Z),
		X: Scalar(q.W*o.X) + Scalar(q.X*o.W) + Scalar(q.Y*o.Z) - Scalar(q.Z*o.Y),
		Y: Scalar(q.W*o.Y) - Scalar(q.X*o.Z) + Scalar(q.Y*o.W) + Scalar(q.Z*o.X),
		Z: Scalar(q.W*o.Z) + Scalar(q.X*o.Y) - Scalar(q.Y*o.X) + Scalar(q.Z*o.W),
	}
}

// Matrix4 returns q's rotation as an affine transform (no translation). q is
// normalized first so a slightly off-unit solver iterate still yields a valid rotation.
func (q Quaternion) Matrix4() Matrix4 {
	n := q.Normalize()
	w, x, y, z := n.W, n.X, n.Y, n.Z
	return Matrix4FromCells([16]Scalar{
		1 - Scalar(2*(Scalar(y*y)+Scalar(z*z))), Scalar(2 * (Scalar(x*y) - Scalar(w*z))), Scalar(2 * (Scalar(x*z) + Scalar(w*y))), 0,
		Scalar(2 * (Scalar(x*y) + Scalar(w*z))), 1 - Scalar(2*(Scalar(x*x)+Scalar(z*z))), Scalar(2 * (Scalar(y*z) - Scalar(w*x))), 0,
		Scalar(2 * (Scalar(x*z) - Scalar(w*y))), Scalar(2 * (Scalar(y*z) + Scalar(w*x))), 1 - Scalar(2*(Scalar(x*x)+Scalar(y*y))), 0,
		0, 0, 0, 1,
	})
}

// QuaternionFromMatrix extracts the rotation of a rigid transform (Shepperd's method).
// The caller is responsible for m's linear part being a pure rotation (orthonormal,
// determinant +1); any scale/shear is ignored. It is the inverse of [Quaternion.Matrix4]
// up to the usual q/−q double cover, used to warm-start the solver from an occurrence's
// current placement.
func QuaternionFromMatrix(m Matrix4) Quaternion {
	trace := m.At(0, 0) + m.At(1, 1) + m.At(2, 2)
	if trace > 0 {
		s := Scalar(0.5 / stdmath.Sqrt(trace+1))
		return Quaternion{
			W: Scalar(0.25 / s),
			X: Scalar((m.At(2, 1) - m.At(1, 2)) * s),
			Y: Scalar((m.At(0, 2) - m.At(2, 0)) * s),
			Z: Scalar((m.At(1, 0) - m.At(0, 1)) * s),
		}.Normalize()
	}
	return quatFromLargestDiagonal(m).Normalize()
}

// quatFromLargestDiagonal handles the trace ≤ 0 branches of Shepperd's method, pivoting
// on whichever diagonal entry is largest for numerical stability.
func quatFromLargestDiagonal(m Matrix4) Quaternion {
	switch {
	case m.At(0, 0) > m.At(1, 1) && m.At(0, 0) > m.At(2, 2):
		s := Scalar(2 * stdmath.Sqrt(1+m.At(0, 0)-m.At(1, 1)-m.At(2, 2)))
		return Quaternion{
			W: Scalar((m.At(2, 1) - m.At(1, 2)) / s), X: Scalar(0.25 * s),
			Y: Scalar((m.At(0, 1) + m.At(1, 0)) / s), Z: Scalar((m.At(0, 2) + m.At(2, 0)) / s),
		}
	case m.At(1, 1) > m.At(2, 2):
		s := Scalar(2 * stdmath.Sqrt(1+m.At(1, 1)-m.At(0, 0)-m.At(2, 2)))
		return Quaternion{
			W: Scalar((m.At(0, 2) - m.At(2, 0)) / s), X: Scalar((m.At(0, 1) + m.At(1, 0)) / s),
			Y: Scalar(0.25 * s), Z: Scalar((m.At(1, 2) + m.At(2, 1)) / s),
		}
	default:
		s := Scalar(2 * stdmath.Sqrt(1+m.At(2, 2)-m.At(0, 0)-m.At(1, 1)))
		return Quaternion{
			W: Scalar((m.At(1, 0) - m.At(0, 1)) / s), X: Scalar((m.At(0, 2) + m.At(2, 0)) / s),
			Y: Scalar((m.At(1, 2) + m.At(2, 1)) / s), Z: Scalar(0.25 * s),
		}
	}
}

// IsEqualTo reports whether q and o are the same orientation within tol, accounting for
// the quaternion double cover (q and −q rotate identically). Pass tol <= 0 for the
// default.
func (q Quaternion) IsEqualTo(o Quaternion, tol Scalar) bool {
	tt := resolveTolerance(tol)
	same := approxEqual(q.W, o.W, tt) && approxEqual(q.X, o.X, tt) &&
		approxEqual(q.Y, o.Y, tt) && approxEqual(q.Z, o.Z, tt)
	flipped := approxEqual(q.W, -o.W, tt) && approxEqual(q.X, -o.X, tt) &&
		approxEqual(q.Y, -o.Y, tt) && approxEqual(q.Z, -o.Z, tt)
	return same || flipped
}
