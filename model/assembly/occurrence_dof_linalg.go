// SPDX-License-Identifier: GPL-2.0-only

package assembly

import stdmath "math"

// Small dense linear algebra for the DOF split (#1980): the null space of a (small-column) matrix by
// Gauss–Jordan elimination. The twist Jacobian has at most six columns, so a direct RREF is both
// simplest and exact enough; a relative pivot tolerance keeps the rank classification scale-invariant.

// columnRange returns the submatrix of j with columns [lo, hi).
func columnRange(j [][]float64, lo, hi int) [][]float64 {
	out := make([][]float64, len(j))
	for i, row := range j {
		out[i] = append([]float64(nil), row[lo:hi]...)
	}
	return out
}

// nullSpaceBasis returns a basis of the null space of matrix a (rows = equations, columns =
// variables): one vector per free (non-pivot) column, in column order.
func nullSpaceBasis(a [][]float64) [][]float64 {
	if len(a) == 0 || len(a[0]) == 0 {
		return nil
	}
	r := cloneRows(a)
	tol := relTol(r)
	pivotCols := reduceToRREF(r, tol)
	return freeColumnVectors(r, pivotCols, len(a[0]))
}

// reduceToRREF row-reduces r in place and returns the pivot column of each pivot row, in order.
func reduceToRREF(r [][]float64, tol float64) []int {
	var pivotCols []int
	row := 0
	n := len(r[0])
	for col := 0; col < n && row < len(r); col++ {
		piv := pivotRow(r, row, col, tol)
		if piv < 0 {
			continue
		}
		r[row], r[piv] = r[piv], r[row]
		scaleRow(r[row], 1/r[row][col])
		eliminateColumn(r, row, col)
		pivotCols = append(pivotCols, col)
		row++
	}
	return pivotCols
}

// pivotRow returns the row (≥ start) with the largest magnitude in col above tol, or -1.
func pivotRow(r [][]float64, start, col int, tol float64) int {
	best, piv := tol, -1
	for i := start; i < len(r); i++ {
		if v := stdmath.Abs(r[i][col]); v > best {
			best, piv = v, i
		}
	}
	return piv
}

// scaleRow multiplies every entry of row by f.
func scaleRow(row []float64, f float64) {
	for j := range row {
		row[j] *= f
	}
}

// eliminateColumn zeroes col in every row but pivot, subtracting a multiple of the pivot row.
func eliminateColumn(r [][]float64, pivot, col int) {
	for i := range r {
		if i == pivot || r[i][col] == 0 {
			continue
		}
		g := r[i][col]
		for j := range r[i] {
			r[i][j] -= g * r[pivot][j]
		}
	}
}

// freeColumnVectors builds a null-space basis vector for each free column: 1 in the free column and
// −(pivot-row coefficient) in each pivot column.
func freeColumnVectors(r [][]float64, pivotCols []int, n int) [][]float64 {
	isPivot := make([]bool, n)
	for _, c := range pivotCols {
		isPivot[c] = true
	}
	var basis [][]float64
	for f := 0; f < n; f++ {
		if isPivot[f] {
			continue
		}
		vec := make([]float64, n)
		vec[f] = 1
		for pr, pc := range pivotCols {
			vec[pc] = -r[pr][f]
		}
		basis = append(basis, vec)
	}
	return basis
}

// cloneRows deep-copies a matrix.
func cloneRows(a [][]float64) [][]float64 {
	out := make([][]float64, len(a))
	for i := range a {
		out[i] = append([]float64(nil), a[i]...)
	}
	return out
}

// relTol is a pivot threshold relative to the matrix's largest magnitude entry, so rank
// classification is scale-invariant.
func relTol(a [][]float64) float64 {
	maxAbs := 0.0
	for _, row := range a {
		for _, v := range row {
			if av := stdmath.Abs(v); av > maxAbs {
				maxAbs = av
			}
		}
	}
	if maxAbs == 0 {
		return 1e-12
	}
	return 1e-9 * maxAbs
}
