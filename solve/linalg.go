// SPDX-License-Identifier: GPL-2.0-only

package solve

import stdmath "math"

// Minimal dense linear algebra for the constraint solver — pure Go, no
// dependencies (ADR-0009). Matrices are [][]float64 (row-major).

// leastSquares solves min ‖A·x − b‖₂ for an over-determined (rows ≥ cols) or square
// system by Householder QR — the numerically stable factorisation that avoids forming the
// normal equations AᵀA (which squares the condition number, ADR §1420/Nocedal–Wright §10).
// It returns ok=false if A is rank-deficient to working precision (a vanishing pivot on
// R's diagonal); the Levenberg–Marquardt caller appends √λ·I so the augmented matrix is
// always full column rank. A and b are not modified.
func leastSquares(src [][]float64, b []float64) ([]float64, bool) {
	rows := len(src)
	if rows == 0 || len(src[0]) == 0 {
		return nil, false
	}
	cols := len(src[0])
	a := make([][]float64, rows)
	for i := range a {
		a[i] = append([]float64(nil), src[i]...)
	}
	y := append([]float64(nil), b...)
	for k := 0; k < cols; k++ {
		if !householderColumn(a, y, k, rows, cols) {
			return nil, false
		}
	}
	return triangularSolve(a, y, cols), true
}

// householderColumn reflects column k so its sub-diagonal entries vanish, applying the same
// reflector to the remaining columns and to y. It returns false when the column is zero to
// working precision (rank-deficient).
func householderColumn(a [][]float64, y []float64, k, rows, cols int) bool {
	v, pivot, ok := householderVector(a, k, rows)
	if !ok {
		return false
	}
	beta := 0.0
	for i := k; i < rows; i++ {
		beta += v[i] * v[i]
	}
	for j := k; j < cols; j++ {
		applyReflector(a, v, beta, k, j, rows)
	}
	applyReflectorVec(y, v, beta, k, rows)
	a[k][k] = pivot
	for i := k + 1; i < rows; i++ {
		a[i][k] = 0
	}
	return true
}

// householderVector builds the reflector vector v = a[k:,k] − pivot·e_k for column k (pivot
// is the resulting R diagonal entry, signed to avoid subtractive cancellation), or ok=false
// when the column is zero to working precision (rank-deficient).
func householderVector(a [][]float64, k, rows int) (v []float64, pivot float64, ok bool) {
	norm := 0.0
	for i := k; i < rows; i++ {
		norm += a[i][k] * a[i][k]
	}
	norm = stdmath.Sqrt(norm)
	if norm < 1e-14 { // tol:numeric — vanishing column ⇒ rank-deficient (LM damping prevents this)
		return nil, 0, false
	}
	if a[k][k] > 0 {
		norm = -norm
	}
	v = make([]float64, rows)
	for i := k; i < rows; i++ {
		v[i] = a[i][k]
	}
	v[k] -= norm
	return v, norm, true
}

// applyReflector applies the Householder reflector (I − 2vvᵀ/β) to column j of a.
func applyReflector(a [][]float64, v []float64, beta float64, k, j, rows int) {
	dot := 0.0
	for i := k; i < rows; i++ {
		dot += v[i] * a[i][j]
	}
	f := 2 * dot / beta
	for i := k; i < rows; i++ {
		a[i][j] -= f * v[i]
	}
}

// applyReflectorVec applies the same reflector to the right-hand side y.
func applyReflectorVec(y, v []float64, beta float64, k, rows int) {
	dot := 0.0
	for i := k; i < rows; i++ {
		dot += v[i] * y[i]
	}
	f := 2 * dot / beta
	for i := k; i < rows; i++ {
		y[i] -= f * v[i]
	}
}

// triangularSolve back-substitutes the upper-triangular R·x = y₀ (R is the top cols×cols of
// the reflected a; y₀ the first cols entries of the reflected rhs).
func triangularSolve(a [][]float64, y []float64, cols int) []float64 {
	x := make([]float64, cols)
	for i := cols - 1; i >= 0; i-- {
		s := y[i]
		for j := i + 1; j < cols; j++ {
			s -= a[i][j] * x[j]
		}
		x[i] = s / a[i][i]
	}
	return x
}

// matrixRank returns the numerical rank of an m×n matrix via Gauss–Jordan
// elimination, counting pivots whose magnitude exceeds tol.
func matrixRank(src [][]float64, tol float64) int {
	rows := len(src)
	if rows == 0 || len(src[0]) == 0 {
		return 0
	}
	cols := len(src[0])
	a := make([][]float64, rows)
	for i := range a {
		a[i] = append([]float64(nil), src[i]...)
	}
	rank, pivotRow := 0, 0
	for col := 0; col < cols && pivotRow < rows; col++ {
		if !pivotAt(a, pivotRow, col, tol) {
			continue
		}
		clearColumn(a, pivotRow, col, cols)
		pivotRow++
		rank++
	}
	return rank
}

// pivotAt finds the largest-magnitude entry in col at or below pivotRow and swaps
// it up; it reports whether a usable pivot (> tol) was found.
func pivotAt(a [][]float64, pivotRow, col int, tol float64) bool {
	sel := pivotRow
	for r := pivotRow + 1; r < len(a); r++ {
		if stdmath.Abs(a[r][col]) > stdmath.Abs(a[sel][col]) {
			sel = r
		}
	}
	if stdmath.Abs(a[sel][col]) < tol {
		return false
	}
	a[pivotRow], a[sel] = a[sel], a[pivotRow]
	return true
}

// clearColumn eliminates col from every other row using the pivot row.
func clearColumn(a [][]float64, pivotRow, col, cols int) {
	for r := 0; r < len(a); r++ {
		if r == pivotRow {
			continue
		}
		f := a[r][col] / a[pivotRow][col]
		for c := col; c < cols; c++ {
			a[r][c] -= f * a[pivotRow][c]
		}
	}
}

// infNorm returns the maximum absolute component of v.
func infNorm(v []float64) float64 {
	max := 0.0
	for _, x := range v {
		if a := stdmath.Abs(x); a > max {
			max = a
		}
	}
	return max
}
