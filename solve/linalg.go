// SPDX-License-Identifier: GPL-2.0-only

package solve

import stdmath "math"

// Minimal dense linear algebra for the constraint solver — pure Go, no
// dependencies (ADR-0009). Matrices are [][]float64 (row-major).

// solveLinear solves A·x = b for a square system by Gaussian elimination with
// partial pivoting. It returns ok=false if A is singular to working precision.
func solveLinear(a [][]float64, b []float64) ([]float64, bool) {
	n := len(b)
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n+1)
		copy(m[i], a[i])
		m[i][n] = b[i]
	}
	for col := 0; col < n; col++ {
		piv := col
		for r := col + 1; r < n; r++ {
			if stdmath.Abs(m[r][col]) > stdmath.Abs(m[piv][col]) {
				piv = r
			}
		}
		if stdmath.Abs(m[piv][col]) < 1e-14 {
			return nil, false
		}
		m[col], m[piv] = m[piv], m[col]
		eliminateBelow(m, col, n)
	}
	return backSubstitute(m, n), true
}

func eliminateBelow(m [][]float64, col, n int) {
	for r := col + 1; r < n; r++ {
		f := m[r][col] / m[col][col]
		for c := col; c <= n; c++ {
			m[r][c] -= f * m[col][c]
		}
	}
}

func backSubstitute(m [][]float64, n int) []float64 {
	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		s := m[i][n]
		for c := i + 1; c < n; c++ {
			s -= m[i][c] * x[c]
		}
		x[i] = s / m[i][i]
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
