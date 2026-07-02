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

// pivotedQRRank is the rank of src by column-pivoted Householder QR with a threshold RELATIVE
// to the largest pivot (Golub & Van Loan §5.4.2; audit A13 #1609): at step k the remaining
// column of largest norm is swapped in, one Householder reflection zeroes it below the
// diagonal, and the factorization stops when the pivot falls under relTol times the first
// (largest) pivot — scale-invariant where the retired Gauss–Jordan absolute threshold dropped
// rank on large-coordinate sketches.
// rankRelTol is the CPQR pivot threshold RELATIVE to the largest pivot: rows are unit-
// normalized upstream, so 1e-9 marks a pivot ~9 decades below the leading one as numerically
// dependent — scale-free by construction.
const rankRelTol = 1e-9 // tol:numeric — relative rank-revealing threshold

func pivotedQRRank(src [][]float64) int {
	rows := len(src)
	if rows == 0 || len(src[0]) == 0 {
		return 0
	}
	cols := len(src[0])
	a := make([][]float64, rows)
	for i := range a {
		a[i] = append([]float64(nil), src[i]...)
	}
	rank, first := 0, 0.0
	for k := 0; k < min(rows, cols); k++ {
		swapLargestColumn(a, k)
		v, pivot, ok := householderVector(a, k, rows)
		if !ok {
			break
		}
		if k == 0 {
			first = stdmath.Abs(pivot)
		}
		if stdmath.Abs(pivot) < rankRelTol*first {
			break
		}
		applyHouseholder(a, v, k, rows, cols)
		rank++
	}
	return rank
}

// swapLargestColumn swaps the trailing column of largest sub-column norm (rows k..end) into
// position k — the column pivoting that makes the QR factorization rank-revealing.
func swapLargestColumn(a [][]float64, k int) {
	cols := len(a[0])
	best, bestNorm := k, subColumnNorm(a, k, k)
	for c := k + 1; c < cols; c++ {
		if n := subColumnNorm(a, c, k); n > bestNorm {
			best, bestNorm = c, n
		}
	}
	if best == k {
		return
	}
	for r := range a {
		a[r][k], a[r][best] = a[r][best], a[r][k]
	}
}

// subColumnNorm is the Euclidean norm of column c from row k down.
func subColumnNorm(a [][]float64, c, k int) float64 {
	s := 0.0
	for r := k; r < len(a); r++ {
		s += a[r][c] * a[r][c]
	}
	return stdmath.Sqrt(s)
}

// applyHouseholder applies the reflection for Householder vector v (from householderVector at
// step k) to the trailing columns of a.
func applyHouseholder(a [][]float64, v []float64, k, rows, cols int) {
	vTv := 0.0
	for i := k; i < rows; i++ {
		vTv += v[i] * v[i]
	}
	if vTv == 0 {
		return
	}
	for c := k; c < cols; c++ {
		dot := 0.0
		for i := k; i < rows; i++ {
			dot += v[i] * a[i][c]
		}
		f := 2 * dot / vTv
		for i := k; i < rows; i++ {
			a[i][c] -= f * v[i]
		}
	}
}
