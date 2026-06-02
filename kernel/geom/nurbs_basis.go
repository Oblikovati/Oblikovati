// SPDX-License-Identifier: GPL-2.0-only

package geom

// B-spline basis-function machinery (Piegl & Tiller, "The NURBS Book").
// Shared by BSplineCurve and BSplineSurface in each parametric direction.

// findSpan returns the knot-span index containing u, where n is the index of
// the last control point (len(ctrl)−1) and p the degree. Clamps to the valid
// span at the domain ends.
func findSpan(n, p int, u float64, knots []float64) int {
	if u >= knots[n+1] {
		return n
	}
	if u <= knots[p] {
		return p
	}
	low, high := p, n+1
	mid := (low + high) / 2
	for u < knots[mid] || u >= knots[mid+1] {
		if u < knots[mid] {
			high = mid
		} else {
			low = mid
		}
		mid = (low + high) / 2
	}
	return mid
}

// basisFuns returns the p+1 nonzero degree-p basis functions at u (global
// indices span−p … span), via the triangular Cox–de Boor recurrence.
func basisFuns(span, p int, u float64, knots []float64) []float64 {
	n := make([]float64, p+1)
	left := make([]float64, p+1)
	right := make([]float64, p+1)
	n[0] = 1
	for j := 1; j <= p; j++ {
		left[j], right[j] = u-knots[span+1-j], knots[span+j]-u
		saved := 0.0
		for r := 0; r < j; r++ {
			temp := n[r] / (right[r+1] + left[j-r])
			n[r] = saved + right[r+1]*temp
			saved = left[j-r] * temp
		}
		n[j] = saved
	}
	return n
}

// basisAndFirstDerivs returns the p+1 nonzero degree-p basis values and their
// first derivatives at u (global indices span−p … span). Derivatives use the
// standard lower-degree identity
//
//	N'_{i,p} = p·( N_{i,p−1}/(U[i+p]−U[i]) − N_{i+1,p−1}/(U[i+p+1]−U[i+1]) ),
//
// with out-of-range / repeated-knot terms taken as zero.
func basisAndFirstDerivs(span, p int, u float64, knots []float64) (values, derivs []float64) {
	values = basisFuns(span, p, u, knots)
	low := basisFuns(span, p-1, u, knots) // degree p−1, indices span−(p−1) … span
	derivs = make([]float64, p+1)
	for k := 0; k <= p; k++ {
		i := span - p + k
		derivs[k] = float64(p) * (lowBasis(low, knots, span, p, i) - lowBasis(low, knots, span, p, i+1))
	}
	return values, derivs
}

// lowBasis returns N_{i,p−1}/(U[i+p]−U[i]) for the precomputed degree-(p−1)
// values, treating out-of-support indices and zero-width (repeated) knot spans
// as 0.
func lowBasis(low, knots []float64, span, p, i int) float64 {
	idx := i - (span - (p - 1))
	if idx < 0 || idx >= len(low) {
		return 0
	}
	den := knots[i+p] - knots[i]
	if den == 0 {
		return 0
	}
	return low[idx] / den
}
