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
	half := angle / 2
	s := stdmath.Sin(half)
	return Quaternion{
		W: stdmath.Cos(half),
		X: axis.X() * s,
		Y: axis.Y() * s,
		Z: axis.Z() * s,
	}
}

// LengthSquared returns w²+x²+y²+z².
func (q Quaternion) LengthSquared() Scalar {
	return q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z
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
	return Quaternion{q.W / n, q.X / n, q.Y / n, q.Z / n}
}

// Mul returns the Hamilton product q·o: the rotation that applies o first, then q.
func (q Quaternion) Mul(o Quaternion) Quaternion {
	return Quaternion{
		W: q.W*o.W - q.X*o.X - q.Y*o.Y - q.Z*o.Z,
		X: q.W*o.X + q.X*o.W + q.Y*o.Z - q.Z*o.Y,
		Y: q.W*o.Y - q.X*o.Z + q.Y*o.W + q.Z*o.X,
		Z: q.W*o.Z + q.X*o.Y - q.Y*o.X + q.Z*o.W,
	}
}

// Matrix4 returns q's rotation as an affine transform (no translation). q is
// normalized first so a slightly off-unit solver iterate still yields a valid rotation.
func (q Quaternion) Matrix4() Matrix4 {
	n := q.Normalize()
	w, x, y, z := n.W, n.X, n.Y, n.Z
	return Matrix4FromCells([16]Scalar{
		1 - 2*(y*y+z*z), 2 * (x*y - w*z), 2 * (x*z + w*y), 0,
		2 * (x*y + w*z), 1 - 2*(x*x+z*z), 2 * (y*z - w*x), 0,
		2 * (x*z - w*y), 2 * (y*z + w*x), 1 - 2*(x*x+y*y), 0,
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
		s := 0.5 / stdmath.Sqrt(trace+1)
		return Quaternion{
			W: 0.25 / s,
			X: (m.At(2, 1) - m.At(1, 2)) * s,
			Y: (m.At(0, 2) - m.At(2, 0)) * s,
			Z: (m.At(1, 0) - m.At(0, 1)) * s,
		}.Normalize()
	}
	return quatFromLargestDiagonal(m).Normalize()
}

// quatFromLargestDiagonal handles the trace ≤ 0 branches of Shepperd's method, pivoting
// on whichever diagonal entry is largest for numerical stability.
func quatFromLargestDiagonal(m Matrix4) Quaternion {
	switch {
	case m.At(0, 0) > m.At(1, 1) && m.At(0, 0) > m.At(2, 2):
		s := 2 * stdmath.Sqrt(1+m.At(0, 0)-m.At(1, 1)-m.At(2, 2))
		return Quaternion{
			W: (m.At(2, 1) - m.At(1, 2)) / s, X: 0.25 * s,
			Y: (m.At(0, 1) + m.At(1, 0)) / s, Z: (m.At(0, 2) + m.At(2, 0)) / s,
		}
	case m.At(1, 1) > m.At(2, 2):
		s := 2 * stdmath.Sqrt(1+m.At(1, 1)-m.At(0, 0)-m.At(2, 2))
		return Quaternion{
			W: (m.At(0, 2) - m.At(2, 0)) / s, X: (m.At(0, 1) + m.At(1, 0)) / s,
			Y: 0.25 * s, Z: (m.At(1, 2) + m.At(2, 1)) / s,
		}
	default:
		s := 2 * stdmath.Sqrt(1+m.At(2, 2)-m.At(0, 0)-m.At(1, 1))
		return Quaternion{
			W: (m.At(1, 0) - m.At(0, 1)) / s, X: (m.At(0, 2) + m.At(2, 0)) / s,
			Y: (m.At(1, 2) + m.At(2, 1)) / s, Z: 0.25 * s,
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
