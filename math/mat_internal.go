// SPDX-License-Identifier: GPL-2.0-only

package math

import stdmath "math"

// Shared small-matrix kernels used by both Matrix4 (its 3×3 linear block) and
// Matrix3 (its full 3×3 homogeneous form). Row-major throughout: index r*n+c.

// det3 returns the determinant of a row-major 3×3 matrix.
func det3(m [9]Scalar) Scalar {
	return Scalar(m[0]*(Scalar(m[4]*m[8])-Scalar(m[5]*m[7]))) -
		Scalar(m[1]*(Scalar(m[3]*m[8])-Scalar(m[5]*m[6]))) +
		Scalar(m[2]*(Scalar(m[3]*m[7])-Scalar(m[4]*m[6])))
}

// invert3x3 returns the inverse of a row-major 3×3 matrix and true, or the zero matrix and
// false when it is singular. Singularity is judged by the determinant NORMALIZED by the
// Hadamard bound (the product of row norms, |det| ≤ Πᵢ‖rᵢ‖, Golub & Van Loan): the ratio is a
// scale-invariant conditioning measure, where the retired absolute |det| ≤ DefaultTolerance
// falsely rejected a perfectly conditioned uniform-scale-1e-3 matrix (det = 1e-9; audit A15,
// #1611).
func invert3x3(m [9]Scalar) ([9]Scalar, bool) {
	det := det3(m)
	h := hadamardBound(m)
	if h == 0 || stdmath.Abs(float64(det)) <= 1e-12*float64(h) { // tol:numeric — relative singularity ratio
		return [9]Scalar{}, false
	}
	inv := adjugate3(m)
	for i := range inv {
		inv[i] = Scalar(inv[i] / det)
	}
	return inv, true
}

// adjugate3 returns the adjugate (transposed cofactor matrix) of a 3×3 matrix.
func adjugate3(m [9]Scalar) [9]Scalar {
	return [9]Scalar{
		Scalar(m[4]*m[8]) - Scalar(m[5]*m[7]), Scalar(m[2]*m[7]) - Scalar(m[1]*m[8]), Scalar(m[1]*m[5]) - Scalar(m[2]*m[4]),
		Scalar(m[5]*m[6]) - Scalar(m[3]*m[8]), Scalar(m[0]*m[8]) - Scalar(m[2]*m[6]), Scalar(m[2]*m[3]) - Scalar(m[0]*m[5]),
		Scalar(m[3]*m[7]) - Scalar(m[4]*m[6]), Scalar(m[1]*m[6]) - Scalar(m[0]*m[7]), Scalar(m[0]*m[4]) - Scalar(m[1]*m[3]),
	}
}

// mul3x3 returns the matrix product a·b of two row-major 3×3 matrices.
func mul3x3(a, b [9]Scalar) [9]Scalar {
	var out [9]Scalar
	for r := range 3 {
		for c := range 3 {
			out[r*3+c] = Scalar(a[r*3]*b[c]) + Scalar(a[r*3+1]*b[3+c]) + Scalar(a[r*3+2]*b[6+c])
		}
	}
	return out
}

// hadamardBound is the product of the row norms — the Hadamard upper bound on |det|, used to
// normalize the singularity test so it is invariant under uniform scaling.
func hadamardBound(m [9]Scalar) Scalar {
	row := func(i int) float64 {
		return stdmath.Sqrt(float64(Scalar(m[3*i]*m[3*i]) + Scalar(m[3*i+1]*m[3*i+1]) + Scalar(m[3*i+2]*m[3*i+2])))
	}
	return Scalar(row(0) * row(1) * row(2))
}
