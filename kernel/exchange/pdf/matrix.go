// SPDX-License-Identifier: GPL-2.0-only

package pdf

// matrix is a 2-D affine transform stored as the six PDF operands [a b c d e f], which
// denote the 3×3 matrix [[a b 0] [c d 0] [e f 1]]. A point row-vector [x y 1] maps as
// [x y 1]·M, i.e. (a·x + c·y + e, b·x + d·y + f) — the PDF current-transformation-matrix
// convention (spec §8.3.3).
type matrix [6]float64

// translateMatrix shifts by (dx, dy).
func translateMatrix(dx, dy float64) matrix { return matrix{1, 0, 0, 1, dx, dy} }

// apply transforms a user-space point to the space this matrix maps into.
func (m matrix) apply(x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

// concat returns a·b: the transform that applies a first, then b (row-vector order
// p·a·b). The cm operator uses concat(Mcm, CTM) so new user space maps through Mcm into
// the previous user space and on to device space.
func concat(a, b matrix) matrix {
	return matrix{
		a[0]*b[0] + a[1]*b[2],
		a[0]*b[1] + a[1]*b[3],
		a[2]*b[0] + a[3]*b[2],
		a[2]*b[1] + a[3]*b[3],
		a[4]*b[0] + a[5]*b[2] + b[4],
		a[4]*b[1] + a[5]*b[3] + b[5],
	}
}
