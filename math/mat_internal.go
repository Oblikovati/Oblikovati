// SPDX-License-Identifier: GPL-2.0-only

package math

// Shared small-matrix kernels used by both Matrix4 (its 3×3 linear block) and
// Matrix3 (its full 3×3 homogeneous form). Row-major throughout: index r*n+c.

// det3 returns the determinant of a row-major 3×3 matrix.
func det3(m [9]Scalar) Scalar {
	return m[0]*(m[4]*m[8]-m[5]*m[7]) -
		m[1]*(m[3]*m[8]-m[5]*m[6]) +
		m[2]*(m[3]*m[7]-m[4]*m[6])
}

// invert3x3 returns the inverse of a row-major 3×3 matrix and true, or the zero
// matrix and false when it is singular (|det| <= DefaultTolerance).
func invert3x3(m [9]Scalar) ([9]Scalar, bool) {
	det := det3(m)
	if det <= DefaultTolerance && det >= -DefaultTolerance {
		return [9]Scalar{}, false
	}
	inv := adjugate3(m)
	for i := range inv {
		inv[i] /= det
	}
	return inv, true
}

// adjugate3 returns the adjugate (transposed cofactor matrix) of a 3×3 matrix.
func adjugate3(m [9]Scalar) [9]Scalar {
	return [9]Scalar{
		m[4]*m[8] - m[5]*m[7], m[2]*m[7] - m[1]*m[8], m[1]*m[5] - m[2]*m[4],
		m[5]*m[6] - m[3]*m[8], m[0]*m[8] - m[2]*m[6], m[2]*m[3] - m[0]*m[5],
		m[3]*m[7] - m[4]*m[6], m[1]*m[6] - m[0]*m[7], m[0]*m[4] - m[1]*m[3],
	}
}

// mul3x3 returns the matrix product a·b of two row-major 3×3 matrices.
func mul3x3(a, b [9]Scalar) [9]Scalar {
	var out [9]Scalar
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			out[r*3+c] = a[r*3]*b[c] + a[r*3+1]*b[3+c] + a[r*3+2]*b[6+c]
		}
	}
	return out
}
