// SPDX-License-Identifier: GPL-2.0-only

package analysis

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// cm2ToMm2 converts the per-unit-density inertia (database units, cm⁵) once multiplied by density
// (g/cm³ → g·cm²) into g·mm² (1 cm² = 100 mm²).
const cm2ToMm2 = 100.0

// applyInertia fills mp's inertia fields: each body's inertia about its own centroid (per unit
// density) is shifted to the combined centroid by the parallel-axis theorem and summed, scaled to
// g·mm² by the density, then diagonalised for the principal moments and axes.
func applyInertia(mp *MassProperties, bodies []*topo.Body, q ops.Quality, density float64, centroidCm [3]float64) {
	var total sym3
	for _, b := range bodies {
		gp := ops.BodyGeometryProperties(b, q)
		it := ops.BodyInertia(b, q) // about the body's own centroid, per unit density
		body := sym3{it.Ixx, it.Iyy, it.Izz, it.Ixy, it.Iyz, it.Izx}
		d := [3]float64{centroidCm[0] - float64(gp.Centroid.X), centroidCm[1] - float64(gp.Centroid.Y), centroidCm[2] - float64(gp.Centroid.Z)}
		total = total.add(body).add(parallelAxis(gp.Volume, d))
	}
	scale := density * cm2ToMm2 // per-unit-density cm⁵ → g·mm²
	mp.InertiaXxGmm2, mp.InertiaYyGmm2, mp.InertiaZzGmm2 = total.xx*scale, total.yy*scale, total.zz*scale
	mp.InertiaXyGmm2, mp.InertiaYzGmm2, mp.InertiaZxGmm2 = total.xy*scale, total.yz*scale, total.zx*scale
	vals, vecs := jacobiEigenSym3(total)
	for i := 0; i < 3; i++ {
		mp.PrincipalMomentsGmm2[i] = vals[i] * scale
		mp.PrincipalAxes[i] = vecs[i]
	}
}

// sym3 is a symmetric 3×3 matrix (the inertia tensor): diagonal xx/yy/zz and products xy/yz/zx.
type sym3 struct{ xx, yy, zz, xy, yz, zx float64 }

func (a sym3) add(b sym3) sym3 {
	return sym3{a.xx + b.xx, a.yy + b.yy, a.zz + b.zz, a.xy + b.xy, a.yz + b.yz, a.zx + b.zx}
}

// parallelAxis is the inertia (per unit density) added when shifting a body's inertia from its
// centroid to a point offset by d: vol·(|d|²·Id − d⊗d), with products following Ixy = −∫xy.
func parallelAxis(vol float64, d [3]float64) sym3 {
	d2 := d[0]*d[0] + d[1]*d[1] + d[2]*d[2]
	return sym3{
		xx: vol * (d2 - d[0]*d[0]), yy: vol * (d2 - d[1]*d[1]), zz: vol * (d2 - d[2]*d[2]),
		xy: vol * (-d[0] * d[1]), yz: vol * (-d[1] * d[2]), zx: vol * (-d[2] * d[0]),
	}
}

// jacobiEigenSym3 diagonalises a symmetric 3×3 matrix by cyclic Jacobi rotation, returning the
// eigenvalues sorted ascending and their unit eigenvectors (vecs[i] pairs with vals[i]).
func jacobiEigenSym3(s sym3) (vals [3]float64, vecs [3][3]float64) {
	a := [3][3]float64{{s.xx, s.xy, s.zx}, {s.xy, s.yy, s.yz}, {s.zx, s.yz, s.zz}}
	v := [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}} // accumulated rotations (columns = eigenvectors)
	for sweep := 0; sweep < 50; sweep++ {
		if offDiagNorm(a) < 1e-18 {
			break
		}
		for _, pq := range [3][2]int{{0, 1}, {0, 2}, {1, 2}} {
			jacobiRotate(&a, &v, pq[0], pq[1])
		}
	}
	idx := [3]int{0, 1, 2} // sort eigenvalues ascending
	for i := 0; i < 3; i++ {
		for j := i + 1; j < 3; j++ {
			if a[idx[j]][idx[j]] < a[idx[i]][idx[i]] {
				idx[i], idx[j] = idx[j], idx[i]
			}
		}
	}
	for i := 0; i < 3; i++ {
		k := idx[i]
		vals[i] = a[k][k]
		vecs[i] = [3]float64{v[0][k], v[1][k], v[2][k]} // column k is the eigenvector
	}
	return vals, vecs
}

// offDiagNorm is the sum of squared off-diagonal entries (the Jacobi convergence measure).
func offDiagNorm(a [3][3]float64) float64 {
	return a[0][1]*a[0][1] + a[0][2]*a[0][2] + a[1][2]*a[1][2]
}

// jacobiRotate zeroes the (p,q) off-diagonal entry of a with a Givens rotation, accumulating it
// into the eigenvector matrix v.
func jacobiRotate(a *[3][3]float64, v *[3][3]float64, p, q int) {
	if stdmath.Abs(a[p][q]) < 1e-300 {
		return
	}
	theta := (a[q][q] - a[p][p]) / (2 * a[p][q])
	t := sign(theta) / (stdmath.Abs(theta) + stdmath.Sqrt(theta*theta+1))
	if theta == 0 {
		t = 1
	}
	c := 1 / stdmath.Sqrt(t*t+1)
	sn := t * c
	for k := 0; k < 3; k++ {
		akp, akq := a[k][p], a[k][q]
		a[k][p] = c*akp - sn*akq
		a[k][q] = sn*akp + c*akq
	}
	for k := 0; k < 3; k++ {
		apk, aqk := a[p][k], a[q][k]
		a[p][k] = c*apk - sn*aqk
		a[q][k] = sn*apk + c*aqk
	}
	for k := 0; k < 3; k++ {
		vkp, vkq := v[k][p], v[k][q]
		v[k][p] = c*vkp - sn*vkq
		v[k][q] = sn*vkp + c*vkq
	}
}

// sign returns ±1 (with +1 for zero), for the Jacobi rotation angle.
func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}
