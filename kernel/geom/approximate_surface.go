// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Tensor-product least-squares surface approximation (M36-F15) — the fitter behind "fit a Class-A
// NURBS to a scanned region". Given scattered points already carrying surface parameters (u,v) in
// [0,1] (a base-plane projection supplies them; see kernel/fit), it solves for the nu×nv control net
// of a degree du×dv B-spline that minimizes Σ‖S(u_k,v_k)−Q_k‖² (The NURBS Book §9.4.1, generalized to
// a tensor grid). Few, evenly-knotted control points is what makes the result "clean" (Class-A):
// good reflection lines and predictable milling, unlike a point-by-point interpolation of noisy scan
// data. The normal equations (NᵀN)·P = NᵀQ are assembled directly from the separable tensor basis.

// surfaceFitRidge is a tiny diagonal regularization (relative to the largest normal-matrix diagonal)
// that keeps NᵀN invertible against round-off; on a well-covered region it is negligible.
const surfaceFitRidge = 1e-9

// ApproximateSurfaceLS fits a clean degree du×dv B-spline with nu×nv control points to points, whose
// surface parameters are us[k],vs[k] ∈ [0,1]. It needs nu≥du+1, nv≥dv+1 and at least nu*nv points;
// it errors when the parameter coverage leaves the normal system singular (region too sparse for the
// requested control count). Knots are uniform-clamped so the net stays even.
func ApproximateSurfaceLS(points []math.Point3, us, vs []float64, du, dv, nu, nv int) (BSplineSurface, error) {
	if err := validateSurfaceFit(len(points), len(us), len(vs), du, dv, nu, nv); err != nil {
		return BSplineSurface{}, err
	}
	uknots := uniformClampedKnots(nu, du)
	vknots := uniformClampedKnots(nv, dv)
	ntn, rhs := surfaceNormalSystem(points, us, vs, du, dv, nu, nv, uknots, vknots)
	if err := gaussSolve(ntn, rhs); err != nil {
		return BSplineSurface{}, fmt.Errorf("geom.ApproximateSurfaceLS: solve %dx%d normal system: %w", nu*nv, nu*nv, err)
	}
	return NewBSplineSurface(du, dv, reshapeNet(rhs, nu, nv), unitNet(nu, nv), uknots, vknots)
}

// validateSurfaceFit checks the surface-fit size relationships (mirrors validateApprox per direction).
func validateSurfaceFit(npts, nus, nvs, du, dv, nu, nv int) error {
	if nus != npts || nvs != npts {
		return fmt.Errorf("geom.ApproximateSurfaceLS: %d points but %d u-params and %d v-params", npts, nus, nvs)
	}
	if du < 1 || dv < 1 {
		return fmt.Errorf("geom.ApproximateSurfaceLS: degrees (%d,%d) must be >= 1", du, dv)
	}
	if nu < du+1 || nv < dv+1 {
		return fmt.Errorf("geom.ApproximateSurfaceLS: need >= degree+1 control points each way, got %dx%d for degree %dx%d", nu, nv, du, dv)
	}
	if npts < nu*nv {
		return fmt.Errorf("geom.ApproximateSurfaceLS: fitting %d control points needs >= that many points, got %d", nu*nv, npts)
	}
	return nil
}

// surfaceNormalSystem assembles (NᵀN)·P = NᵀQ for all nu*nv control points. Each point contributes
// only through its (du+1)×(dv+1) active basis values (the separable tensor product), so the
// accumulation touches a small block per point. A tiny ridge keeps NᵀN invertible.
func surfaceNormalSystem(points []math.Point3, us, vs []float64, du, dv, nu, nv int, uknots, vknots []float64) (ntn, rhs [][]float64) {
	ncp := nu * nv
	ntn, rhs = zeros(ncp, ncp), zeros(ncp, 3)
	for k, q := range points {
		rows, vals := activeTensorBasis(us[k], vs[k], du, dv, nu, nv, uknots, vknots)
		qc := [3]float64{float64(q.X), float64(q.Y), float64(q.Z)}
		for x, r := range rows {
			for c := range 3 {
				rhs[r][c] += vals[x] * qc[c]
			}
			for y, rr := range rows {
				ntn[r][rr] += vals[x] * vals[y]
			}
		}
	}
	applyRidge(ntn)
	return ntn, rhs
}

// activeTensorBasis returns the flat control indices (i*nv+j) and tensor basis values bu[i]*bv[j] of
// the (du+1)×(dv+1) control points whose support covers (u,v).
func activeTensorBasis(u, v float64, du, dv, nu, nv int, uknots, vknots []float64) (rows []int, vals []float64) {
	su := findSpan(nu-1, du, u, uknots)
	sv := findSpan(nv-1, dv, v, vknots)
	bu := basisFuns(su, du, u, uknots)
	bv := basisFuns(sv, dv, v, vknots)
	rows = make([]int, 0, (du+1)*(dv+1))
	vals = make([]float64, 0, (du+1)*(dv+1))
	for a := 0; a <= du; a++ {
		for c := 0; c <= dv; c++ {
			rows = append(rows, (su-du+a)*nv+(sv-dv+c))
			vals = append(vals, bu[a]*bv[c])
		}
	}
	return rows, vals
}

// applyRidge adds surfaceFitRidge·maxDiag to every diagonal, regularizing only the directions the
// data leaves unconstrained while barely touching the rest.
func applyRidge(ntn [][]float64) {
	maxDiag := 0.0
	for i := range ntn {
		if ntn[i][i] > maxDiag {
			maxDiag = ntn[i][i]
		}
	}
	eps := surfaceFitRidge * maxDiag
	for i := range ntn {
		ntn[i][i] += eps
	}
}

// reshapeNet folds the solved flat control rows (index i*nv+j) into an nu×nv point net.
func reshapeNet(flat [][]float64, nu, nv int) [][]math.Point3 {
	net := make([][]math.Point3, nu)
	for i := range nu {
		net[i] = make([]math.Point3, nv)
		for j := range nv {
			r := flat[i*nv+j]
			net[i][j] = math.P3(math.Scalar(r[0]), math.Scalar(r[1]), math.Scalar(r[2]))
		}
	}
	return net
}

// uniformClampedKnots builds a clamped knot vector over [0,1] with nctrl control points of degree p:
// p+1 zeros, evenly spaced interiors, p+1 ones.
func uniformClampedKnots(nctrl, p int) []float64 {
	knots := make([]float64, nctrl+p+1)
	interior := nctrl - p - 1
	for i := 0; i <= p; i++ {
		knots[nctrl+i] = 1
	}
	for j := 1; j <= interior; j++ {
		knots[p+j] = float64(j) / float64(interior+1)
	}
	return knots
}
