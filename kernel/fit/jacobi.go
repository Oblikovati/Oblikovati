// SPDX-License-Identifier: GPL-2.0-only

package fit

import stdmath "math"

// mat3 is a 3×3 matrix in row-major order.
type mat3 [3][3]float64

// jacobiSweeps bounds the cyclic Jacobi iterations; a symmetric 3×3 converges in a handful, so 50
// is a generous ceiling that also guarantees termination.
const jacobiSweeps = 50

// jacobiEigen3 diagonalises a symmetric 3×3 matrix by the cyclic Jacobi method, returning the
// eigenvalues and the matrix of eigenvectors (column j is the eigenvector for vals[j]). It is used
// for principal-component analysis, where the eigenvector of the smallest eigenvalue is the
// best-fit plane normal. The input is assumed symmetric (a[i][j] == a[j][i]).
func jacobiEigen3(a mat3) (vals [3]float64, vecs mat3) {
	vecs = identity3()
	for range jacobiSweeps {
		if offDiagonalNorm(a) == 0 {
			break
		}
		for _, pq := range [3][2]int{{0, 1}, {0, 2}, {1, 2}} {
			a, vecs = rotate(a, vecs, pq[0], pq[1])
		}
	}
	return [3]float64{a[0][0], a[1][1], a[2][2]}, vecs
}

// rotate applies the Givens rotation that zeroes a[p][q] to both a (similarity transform) and the
// accumulated eigenvector matrix v.
func rotate(a, v mat3, p, q int) (mat3, mat3) {
	if a[p][q] == 0 {
		return a, v
	}
	c, s := givens(a[p][p], a[q][q], a[p][q])
	a = applySimilarity(a, p, q, c, s)
	v = applyColumns(v, p, q, c, s)
	return a, v
}

// givens returns the cosine and sine of the rotation that annihilates the off-diagonal app/aqq/apq.
func givens(app, aqq, apq float64) (c, s float64) {
	theta := (aqq - app) / (2 * apq)
	t := sign(theta) / (stdmath.Abs(theta) + stdmath.Sqrt(theta*theta+1))
	c = 1 / stdmath.Sqrt(t*t+1)
	return c, t * c
}

// applySimilarity computes Jᵀ·a·J for the rotation in the (p,q) plane, keeping the matrix symmetric.
func applySimilarity(a mat3, p, q int, c, s float64) mat3 {
	app, aqq, apq := a[p][p], a[q][q], a[p][q]
	a[p][p] = c*c*app - 2*s*c*apq + s*s*aqq
	a[q][q] = s*s*app + 2*s*c*apq + c*c*aqq
	a[p][q], a[q][p] = 0, 0
	for k := range 3 {
		if k == p || k == q {
			continue
		}
		akp, akq := a[k][p], a[k][q]
		a[k][p] = c*akp - s*akq
		a[p][k] = a[k][p]
		a[k][q] = s*akp + c*akq
		a[q][k] = a[k][q]
	}
	return a
}

// applyColumns post-multiplies the eigenvector matrix by the rotation (updates columns p and q).
func applyColumns(v mat3, p, q int, c, s float64) mat3 {
	for k := range 3 {
		vkp, vkq := v[k][p], v[k][q]
		v[k][p] = c*vkp - s*vkq
		v[k][q] = s*vkp + c*vkq
	}
	return v
}

// offDiagonalNorm is the sum of absolute off-diagonal entries (zero ⇒ already diagonal).
func offDiagonalNorm(a mat3) float64 {
	return stdmath.Abs(a[0][1]) + stdmath.Abs(a[0][2]) + stdmath.Abs(a[1][2])
}

func identity3() mat3 { return mat3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} }

func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}
