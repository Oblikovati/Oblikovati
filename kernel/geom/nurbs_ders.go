// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Higher-order B-spline derivative machinery (Piegl & Tiller A2.3 / eq. 4.8 /
// eq. 4.20), powering the member-level evaluators (M01-F06, #603). The earlier
// basisAndFirstDerivs identity covers only order 1; the evaluators need orders
// 2 and 3 for curvature and torsion-grade queries.

// dersBasisFuns returns the nonzero degree-p basis functions and their
// derivatives up to the given order at u: ders[k][j] is the k-th derivative of
// basis function span−p+j. Derivatives beyond the degree are zero (A2.3).
func dersBasisFuns(span, p int, u float64, order int, knots []float64) [][]float64 {
	ndu := basisTriangle(span, p, u, knots)
	ders := make([][]float64, order+1)
	for k := range ders {
		ders[k] = make([]float64, p+1)
	}
	for j := 0; j <= p; j++ {
		ders[0][j] = ndu[j][p]
	}
	for r := 0; r <= p; r++ {
		derivativeRow(ndu, ders, r, p, order)
	}
	scaleDerivativeRows(ders, p, order)
	return ders
}

// basisTriangle builds A2.3's ndu table: ndu[r][j] holds the degree-j basis
// value of function r, and ndu[j][r] (lower triangle) the knot differences the
// derivative pass divides by.
func basisTriangle(span, p int, u float64, knots []float64) [][]float64 {
	ndu := make([][]float64, p+1)
	for i := range ndu {
		ndu[i] = make([]float64, p+1)
	}
	left := make([]float64, p+1)
	right := make([]float64, p+1)
	ndu[0][0] = 1
	for j := 1; j <= p; j++ {
		left[j], right[j] = u-knots[span+1-j], knots[span+j]-u
		saved := 0.0
		for r := 0; r < j; r++ {
			ndu[j][r] = right[r+1] + left[j-r] // knot difference, reused below
			temp := ndu[r][j-1] / ndu[j][r]
			ndu[r][j] = saved + right[r+1]*temp
			saved = left[j-r] * temp
		}
		ndu[j][j] = saved
	}
	return ndu
}

// derivativeRow fills ders[k][r] for one basis function r over every requested
// order k, using the a-coefficient recurrence of A2.3.
func derivativeRow(ndu [][]float64, ders [][]float64, r, p, order int) {
	a := [2][]float64{make([]float64, p+1), make([]float64, p+1)}
	a[0][0] = 1
	s1, s2 := 0, 1
	for k := 1; k <= order && k <= p; k++ {
		d := 0.0
		rk, pk := r-k, p-k
		if r >= k {
			a[s2][0] = a[s1][0] / ndu[pk+1][rk]
			d = a[s2][0] * ndu[rk][pk]
		}
		d += derivativeInner(ndu, a, s1, s2, r, k, p)
		ders[k][r] = d
		s1, s2 = s2, s1
	}
}

// derivativeInner accumulates the middle and upper terms of the a-recurrence.
func derivativeInner(ndu [][]float64, a [2][]float64, s1, s2, r, k, p int) float64 {
	rk, pk := r-k, p-k
	j1, j2 := 1, k-1
	if rk < -1 {
		j1 = -rk
	}
	if r-1 > pk {
		j2 = p - r
	}
	d := 0.0
	for j := j1; j <= j2; j++ {
		a[s2][j] = (a[s1][j] - a[s1][j-1]) / ndu[pk+1][rk+j]
		d += a[s2][j] * ndu[rk+j][pk]
	}
	if r <= pk {
		a[s2][k] = -a[s1][k-1] / ndu[pk+1][r]
		d += a[s2][k] * ndu[r][pk]
	}
	return d
}

// scaleDerivativeRows applies the p!/(p−k)! factor that turns the recurrence
// output into true derivatives.
func scaleDerivativeRows(ders [][]float64, p, order int) {
	factor := float64(p)
	for k := 1; k <= order && k <= p; k++ {
		for j := range ders[k] {
			ders[k][j] *= factor
		}
		factor *= float64(p - k)
	}
}

// binomialRow returns Pascal-triangle row k (k ≤ 3 is all the evaluators need).
func binomialRow(k int) []float64 {
	rows := [4][]float64{{1}, {1, 1}, {1, 2, 1}, {1, 3, 3, 1}}
	return rows[k]
}

// DersAt returns the curve position and parametric derivatives [P, P′, P″, …]
// up to the given order at t, applying the rational quotient rule (P&T eq. 4.8).
func (c BSplineCurve) DersAt(t float64, order int) []math.Vector3 {
	span := findSpan(len(c.Ctrl)-1, c.Degree, t, c.Knots)
	basis := dersBasisFuns(span, c.Degree, t, order, c.Knots)
	num := make([]math.Vector3, order+1) // homogeneous numerators A⁽ᵏ⁾
	den := make([]float64, order+1)      // weight derivatives w⁽ᵏ⁾
	for k := 0; k <= order; k++ {
		for j := 0; j <= c.Degree; j++ {
			i := span - c.Degree + j
			bw := basis[k][j] * c.Weights[i]
			num[k] = num[k].Add(c.Ctrl[i].AsVector().Scale(bw))
			den[k] += bw
		}
	}
	return rationalDers3(num, den, order)
}

// rationalDers3 converts homogeneous derivatives to rational ones via Leibniz:
// C⁽ᵏ⁾ = (A⁽ᵏ⁾ − Σᵢ₌₁ᵏ C(k,i)·w⁽ⁱ⁾·C⁽ᵏ⁻ⁱ⁾) / w.
func rationalDers3(num []math.Vector3, den []float64, order int) []math.Vector3 {
	out := make([]math.Vector3, order+1)
	for k := 0; k <= order; k++ {
		v := num[k]
		bin := binomialRow(k)
		for i := 1; i <= k; i++ {
			v = v.Sub(out[k-i].Scale(bin[i] * den[i]))
		}
		out[k] = v.Scale(1 / den[0])
	}
	return out
}

// DersAt returns the 2D curve position and derivatives up to order at t.
func (c BSplineCurve2d) DersAt(t float64, order int) []math.Vector2 {
	span := findSpan(len(c.Ctrl)-1, c.Degree, t, c.Knots)
	basis := dersBasisFuns(span, c.Degree, t, order, c.Knots)
	num := make([]math.Vector2, order+1)
	den := make([]float64, order+1)
	for k := 0; k <= order; k++ {
		for j := 0; j <= c.Degree; j++ {
			i := span - c.Degree + j
			bw := basis[k][j] * c.Weights[i]
			num[k] = num[k].Add(c.Ctrl[i].AsVector().Scale(bw))
			den[k] += bw
		}
	}
	return rationalDers2(num, den, order)
}

// rationalDers2 is the 2D analogue of [rationalDers3].
func rationalDers2(num []math.Vector2, den []float64, order int) []math.Vector2 {
	out := make([]math.Vector2, order+1)
	for k := 0; k <= order; k++ {
		v := num[k]
		bin := binomialRow(k)
		for i := 1; i <= k; i++ {
			v = v.Sub(out[k-i].Scale(bin[i] * den[i]))
		}
		out[k] = v.Scale(1 / den[0])
	}
	return out
}

// SurfaceDersAt returns the rational surface partials S⁽ᵏ˒ˡ⁾ for every k ≤ du,
// l ≤ dv at (u, v) — out[k][l] is ∂ᵏ⁺ˡS/∂uᵏ∂vˡ (P&T eq. 4.20).
func (s BSplineSurface) SurfaceDersAt(u, v float64, du, dv int) [][]math.Vector3 {
	num, den := s.homogeneousDers(u, v, du, dv)
	out := make([][]math.Vector3, du+1)
	for k := range out {
		out[k] = make([]math.Vector3, dv+1)
	}
	for k := 0; k <= du; k++ {
		for l := 0; l <= dv; l++ {
			out[k][l] = rationalSurfaceDer(out, num, den, k, l)
		}
	}
	return out
}

// homogeneousDers accumulates the tensor-product homogeneous partials A⁽ᵏ˒ˡ⁾
// and weight partials w⁽ᵏ˒ˡ⁾ over the control net.
func (s BSplineSurface) homogeneousDers(u, v float64, du, dv int) (num [][]math.Vector3, den [][]float64) {
	us, vs := s.spans(u, v)
	ubasis := dersBasisFuns(us, s.UDegree, u, du, s.UKnots)
	vbasis := dersBasisFuns(vs, s.VDegree, v, dv, s.VKnots)
	num = make([][]math.Vector3, du+1)
	den = make([][]float64, du+1)
	for k := 0; k <= du; k++ {
		num[k] = make([]math.Vector3, dv+1)
		den[k] = make([]float64, dv+1)
		for l := 0; l <= dv; l++ {
			num[k][l], den[k][l] = s.homogeneousDer(us, vs, ubasis[k], vbasis[l])
		}
	}
	return num, den
}

// homogeneousDer accumulates one (k, l) homogeneous partial over the net.
func (s BSplineSurface) homogeneousDer(us, vs int, ub, vb []float64) (math.Vector3, float64) {
	var a math.Vector3
	w := 0.0
	for i := 0; i <= s.UDegree; i++ {
		for j := 0; j <= s.VDegree; j++ {
			ci, cj := us-s.UDegree+i, vs-s.VDegree+j
			bw := ub[i] * vb[j] * s.Weights[ci][cj]
			a = a.Add(s.Ctrl[ci][cj].AsVector().Scale(bw))
			w += bw
		}
	}
	return a, w
}

// rationalSurfaceDer applies the bivariate Leibniz rule for S⁽ᵏ˒ˡ⁾, consuming
// the already-computed lower-order rational partials in out.
func rationalSurfaceDer(out [][]math.Vector3, num [][]math.Vector3, den [][]float64, k, l int) math.Vector3 {
	v := num[k][l]
	bk, bl := binomialRow(k), binomialRow(l)
	for i := 0; i <= k; i++ {
		for j := 0; j <= l; j++ {
			if i == 0 && j == 0 {
				continue
			}
			v = v.Sub(out[k-i][l-j].Scale(bk[i] * bl[j] * den[i][j]))
		}
	}
	return v.Scale(1 / den[0][0])
}
