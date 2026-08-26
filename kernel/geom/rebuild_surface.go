// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Rebuilding a surface to a clean Class-A NURBS (M36-F02), the tensor-product analogue of
// rebuild_curve.go. A surface least-squares fit is *separable* (The NURBS Book A9.7): sample
// the source on a grid, approximate each v-column as a u-direction curve to get an
// intermediate net, then approximate that net's rows as v-direction curves. Both stages reuse
// the curve fitter (approximate_bspline.go). As with the curve, the rebuild preserves the
// source's parameterization, so it is idempotent on an already-clean surface and the
// deviation can be read off at matching parameters.

// defaultRebuildSurfaceSamples is the per-direction grid density used when a caller passes 0.
const defaultRebuildSurfaceSamples = 40

// RebuildSurface refits src to a clean (uDeg×vDeg) B-spline surface with nu×nv control points,
// returning the rebuilt surface and the maximum geometric deviation from src. It samples a
// (samples+1)² grid (0 ⇒ a default); pass nu = uDeg+1, nv = vDeg+1 for a single-span Class-A
// rebuild. The four corners are interpolated.
//
// Example: clean, dev, _ := RebuildSurface(wavy, 3, 3, 4, 4, 0); if dev <= tol { use(clean) }.
func RebuildSurface(src Surface, uDeg, vDeg, nu, nv, samples int) (BSplineSurface, float64, error) {
	if samples <= 0 {
		samples = defaultRebuildSurfaceSamples
	}
	grid, uu, vv := sampleSurfaceGrid(src, samples)
	ctrl, uKnots, vKnots, err := approximateSurfaceLS(grid, uDeg, vDeg, nu, nv, uu, vv)
	if err != nil {
		return BSplineSurface{}, 0, err
	}
	rebuilt, err := NewBSplineSurface(uDeg, vDeg, ctrl, unitNet(nu, nv), uKnots, vKnots)
	if err != nil {
		return BSplineSurface{}, 0, err
	}
	return rebuilt, surfaceDeviation(src, rebuilt, samples*2), nil
}

// approximateSurfaceLS fits a (uDeg×vDeg) surface with nu×nv control points to the sample grid
// by separable least squares: u-curves per v-column, then v-curves across the intermediate net.
func approximateSurfaceLS(grid [][]math.Point3, uDeg, vDeg, nu, nv int, uu, vv []float64) (ctrl [][]math.Point3, uKnots, vKnots []float64, err error) {
	mid, uKnots, err := fitColumnsU(grid, uDeg, nu, uu)
	if err != nil {
		return nil, nil, nil, err
	}
	ctrl = make([][]math.Point3, nu)
	for k := range nu {
		pctrl, vk, e := approximateLS(mid[k], vDeg, nv, vv)
		if e != nil {
			return nil, nil, nil, e
		}
		vKnots = vk
		ctrl[k] = points3(pctrl)
	}
	return ctrl, uKnots, vKnots, nil
}

// fitColumnsU approximates each v-column of the grid as a u-direction curve, returning the
// intermediate control net mid[k][l] (k over u control points, l over v samples) and the
// shared U knot vector.
func fitColumnsU(grid [][]math.Point3, uDeg, nu int, uu []float64) (mid [][][]float64, uKnots []float64, err error) {
	su, sv := len(grid)-1, len(grid[0])-1
	mid = make([][][]float64, nu)
	for k := range mid {
		mid[k] = make([][]float64, sv+1)
	}
	for l := 0; l <= sv; l++ {
		col := make([]math.Point3, su+1)
		for i := 0; i <= su; i++ {
			col[i] = grid[i][l]
		}
		rctrl, uk, e := approximateLS(coords3(col), uDeg, nu, uu)
		if e != nil {
			return nil, nil, e
		}
		uKnots = uk
		for k := range nu {
			mid[k][l] = rctrl[k]
		}
	}
	return mid, uKnots, nil
}

// sampleSurfaceGrid returns a (samples+1)² grid of src points and the uniform parameters in
// [0,1] for each direction (src's domain mapped linearly), the parameterization rebuild keeps.
func sampleSurfaceGrid(src Surface, samples int) (grid [][]math.Point3, uu, vv []float64) {
	ulo, uhi := src.UDomain()
	vlo, vhi := src.VDomain()
	uu = uniformParams(samples)
	vv = uniformParams(samples)
	grid = make([][]math.Point3, samples+1)
	for i := 0; i <= samples; i++ {
		grid[i] = make([]math.Point3, samples+1)
		u := ulo + (uhi-ulo)*uu[i]
		for j := 0; j <= samples; j++ {
			grid[i][j] = src.PointAt(u, vlo+(vhi-vlo)*vv[j])
		}
	}
	return grid, uu, vv
}

// surfaceDeviation returns the largest distance between src and the rebuilt surface over a
// (samples+1)² grid at matching (parameterization-preserving) parameters — the achieved error.
func surfaceDeviation(src Surface, rebuilt BSplineSurface, samples int) float64 {
	ulo, uhi := src.UDomain()
	vlo, vhi := src.VDomain()
	rulo, ruhi := rebuilt.UDomain()
	rvlo, rvhi := rebuilt.VDomain()
	maxDev := 0.0
	for i := 0; i <= samples; i++ {
		f := float64(i) / float64(samples)
		for j := 0; j <= samples; j++ {
			g := float64(j) / float64(samples)
			p := src.PointAt(ulo+(uhi-ulo)*f, vlo+(vhi-vlo)*g)
			q := rebuilt.PointAt(rulo+(ruhi-rulo)*f, rvlo+(rvhi-rvlo)*g)
			if d := float64(p.DistanceTo(q)); d > maxDev {
				maxDev = d
			}
		}
	}
	return maxDev
}

// uniformParams returns samples+1 parameters evenly spaced over [0,1].
func uniformParams(samples int) []float64 {
	u := make([]float64, samples+1)
	for i := range u {
		u[i] = float64(i) / float64(samples)
	}
	return u
}

// unitNet returns a rows×cols weight net of all ones (a non-rational surface).
func unitNet(rows, cols int) [][]float64 {
	w := make([][]float64, rows)
	for i := range w {
		w[i] = unitWeights(cols)
	}
	return w
}
