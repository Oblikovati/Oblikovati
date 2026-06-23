// SPDX-License-Identifier: GPL-2.0-only

package geom

import "fmt"

// Least-squares B-spline approximation (The NURBS Book A9.4) — the fitter behind Rebuild
// (M36-F02). Unlike the interpolating fit in fitted_bspline.go (one control point per
// sample point, passing exactly through every point), this fits a *fixed, smaller* number
// of control points of a chosen degree, minimizing the squared distance to the samples
// while interpolating the two endpoints. Fewer, evenly-knotted control points is what makes
// a surface "clean" (Class-A): good reflection lines and predictable milling.

// approximateLS fits a degree-p B-spline with nctrl control points to the rows of pts by
// least squares, holding the first and last points interpolated. ubar are the points'
// parameters (in [0,1]). It returns the control rows (same coordinate shape as pts) and the
// knot vector. nctrl must satisfy degree+1 <= nctrl <= len(pts).
func approximateLS(pts [][]float64, p, nctrl int, ubar []float64) (ctrl [][]float64, knots []float64, err error) {
	if err := validateApprox(len(pts), p, nctrl); err != nil {
		return nil, nil, err
	}
	knots = approximationKnots(ubar, p, nctrl)
	n := nctrl - 1
	ctrl = make([][]float64, nctrl)
	ctrl[0] = append([]float64(nil), pts[0]...)
	ctrl[n] = append([]float64(nil), pts[len(pts)-1]...)
	if n < 2 {
		return ctrl, knots, nil // only the two interpolated endpoints (nctrl == 2)
	}
	ntn, rhs := normalSystem(pts, ubar, p, nctrl, knots)
	if err := gaussSolve(ntn, rhs); err != nil {
		return nil, nil, err
	}
	for i := 1; i <= n-1; i++ {
		ctrl[i] = rhs[i-1]
	}
	return ctrl, knots, nil
}

// validateApprox checks the approximation size relationships.
func validateApprox(npts, p, nctrl int) error {
	if p < 1 {
		return fmt.Errorf("geom: approximation degree %d must be >= 1", p)
	}
	if nctrl < p+1 {
		return fmt.Errorf("geom: approximation needs >= degree+1 (%d) control points, got %d", p+1, nctrl)
	}
	if nctrl > npts {
		return fmt.Errorf("geom: approximation to %d control points needs >= that many sample points, got %d", nctrl, npts)
	}
	return nil
}

// approximationKnots spaces the clamped knot vector so each interior knot sits inside the
// span of the parameters it governs (The NURBS Book eqs. 9.68/9.69), keeping the normal
// system well-conditioned for an arbitrary control-point count.
func approximationKnots(ubar []float64, p, nctrl int) []float64 {
	n, m := nctrl-1, len(ubar)-1
	knots := make([]float64, nctrl+p+1)
	for i := nctrl; i <= nctrl+p; i++ {
		knots[i] = ubar[m]
	}
	d := float64(m+1) / float64(n-p+1)
	for j := 1; j <= n-p; j++ {
		i := int(float64(j) * d)
		alpha := float64(j)*d - float64(i)
		knots[p+j] = (1-alpha)*ubar[i-1] + alpha*ubar[i]
	}
	return knots
}

// normalSystem assembles the least-squares normal equations (NᵀN)·P = R for the interior
// control points: NᵀN is the (n−1)×(n−1) Gram matrix of the interior basis functions over
// the interior sample parameters, R folds in the fixed endpoints (A9.4).
func normalSystem(pts [][]float64, ubar []float64, p, nctrl int, knots []float64) (ntn, rhs [][]float64) {
	n, m, dim := nctrl-1, len(pts)-1, len(pts[0])
	ntn = zeros(n-1, n-1)
	rhs = zeros(n-1, dim)
	for k := 1; k <= m-1; k++ {
		nk := denseBasis(ubar[k], p, nctrl, knots)
		rk := residual(pts[k], pts[0], pts[m], nk[0], nk[n])
		for i := 1; i <= n-1; i++ {
			for c := 0; c < dim; c++ {
				rhs[i-1][c] += nk[i] * rk[c]
			}
			for j := 1; j <= n-1; j++ {
				ntn[i-1][j-1] += nk[i] * nk[j]
			}
		}
	}
	return ntn, rhs
}

// denseBasis returns the full length-nctrl vector of degree-p basis values at u (only the
// p+1 values around the active span are nonzero).
func denseBasis(u float64, p, nctrl int, knots []float64) []float64 {
	row := make([]float64, nctrl)
	span := findSpan(nctrl-1, p, u, knots)
	b := basisFuns(span, p, u, knots)
	for l := 0; l <= p; l++ {
		row[span-p+l] = b[l]
	}
	return row
}

// residual returns Qₖ − N₀·Q₀ − Nₙ·Qₘ, the part of a sample point not explained by the two
// fixed endpoints (A9.4's Rₖ).
func residual(qk, q0, qm []float64, n0, nn float64) []float64 {
	r := make([]float64, len(qk))
	for c := range qk {
		r[c] = qk[c] - n0*q0[c] - nn*qm[c]
	}
	return r
}

// zeros allocates a rows×cols matrix of zeros.
func zeros(rows, cols int) [][]float64 {
	m := make([][]float64, rows)
	for i := range m {
		m[i] = make([]float64, cols)
	}
	return m
}
