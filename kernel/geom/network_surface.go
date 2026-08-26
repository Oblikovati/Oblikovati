// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// Network (Gordon) surface (M36-F10): a single NURBS interpolating a grid of intersecting U- and
// V-direction curves — the primary Class-A skinning tool for complex panels. The curves must
// approximately intersect at a regular grid; we sample the grid nodes (averaging each U/V crossing),
// validate the crossings close within tolerance, then globally tensor-interpolate a bicubic B-spline
// through the node grid (Piegl-Tiller A9.7: interpolate each row in v, then those control points in
// u). With dense curves this passes through the curves within tolerance; degenerate / non-intersecting
// input is a clear error (CLAUDE.md exception rule).

// NetworkSurface interpolates the grid formed by uCurves (each spanning the u-direction at a v-station)
// and vCurves (each spanning v at a u-station). It needs ≥2 curves each way. It errors when a U/V
// crossing gap exceeds networkGapTol (the curves do not form a usable grid).
func NetworkSurface(uCurves, vCurves []BSplineCurve) (BSplineSurface, error) {
	nu := len(vCurves) // u-stations (one per v-curve)
	nv := len(uCurves) // v-stations (one per u-curve)
	if nu < 2 || nv < 2 {
		return BSplineSurface{}, fmt.Errorf("geom.NetworkSurface: need ≥2 curves each way, got %d u-curves and %d v-curves", nv, nu)
	}
	grid, err := networkGrid(uCurves, vCurves)
	if err != nil {
		return BSplineSurface{}, err
	}
	return globalInterpSurface(grid)
}

// networkGrid builds the nu×nv node grid: node (a,b) is the crossing of v-curve a (u-station a) and
// u-curve b (v-station b), taken as the midpoint of the two curves' points there. It errors if any
// crossing is farther apart than networkGapTol.
func networkGrid(uCurves, vCurves []BSplineCurve) ([][]math.Point3, error) {
	nu, nv := len(vCurves), len(uCurves)
	gapTol := networkGapTol(uCurves, vCurves)
	grid := make([][]math.Point3, nu)
	for a := range nu {
		grid[a] = make([]math.Point3, nv)
		us := float64(a) / float64(nu-1)
		for b := range nv {
			vs := float64(b) / float64(nv-1)
			pu := uCurves[b].PointAt(us) // u-curve b at u-station a
			pv := vCurves[a].PointAt(vs) // v-curve a at v-station b
			if d := float64(pu.DistanceTo(pv)); d > gapTol {
				return nil, fmt.Errorf("geom.NetworkSurface: curves do not intersect at grid node (%d,%d): gap %.4g > %.4g", a, b, d, gapTol)
			}
			grid[a][b] = pu.Midpoint(pv)
		}
	}
	return grid, nil
}

// networkGapTol is the model-relative gap (#1399) a U- and V-curve crossing may span before the
// network is rejected as non-intersecting: derived from the network's own extent (the curve
// endpoints) so the generous "close enough to form a grid" threshold scales with the part instead
// of a cm-anchored 1e-3.
func networkGapTol(uCurves, vCurves []BSplineCurve) float64 {
	pts := make([]math.Point3, 0, 2*(len(uCurves)+len(vCurves)))
	for _, c := range uCurves {
		pts = append(pts, c.PointAt(0), c.PointAt(1))
	}
	for _, c := range vCurves {
		pts = append(pts, c.PointAt(0), c.PointAt(1))
	}
	return ResolutionForPoints(pts).Sew()
}

// globalInterpSurface tensor-interpolates a bicubic B-spline through the nu×nv point grid (A9.7):
// interpolate each row across v, then interpolate those control points across u.
func globalInterpSurface(grid [][]math.Point3) (BSplineSurface, error) {
	nu, nv := len(grid), len(grid[0])
	pv, err := fitDegree(nv)
	if err != nil {
		return BSplineSurface{}, fmt.Errorf("geom.NetworkSurface: v-direction: %w", err)
	}
	pu, err := fitDegree(nu)
	if err != nil {
		return BSplineSurface{}, fmt.Errorf("geom.NetworkSurface: u-direction: %w", err)
	}
	vparams := avgChordParams(grid, false)
	vknots := averagedKnots(vparams, pv)
	rows, err := interpRows(grid, pv, vparams, vknots) // each grid row → its v-curve control points
	if err != nil {
		return BSplineSurface{}, err
	}
	uparams := avgChordParamsRows(rows)
	uknots := averagedKnots(uparams, pu)
	net, err := interpCols(rows, pu, uparams, uknots) // interpolate those control points across u
	if err != nil {
		return BSplineSurface{}, err
	}
	return NewBSplineSurface(pu, pv, net, unitNet(len(net), len(net[0])), uknots, vknots)
}

// interpRows interpolates each grid row across v (shared v-params/knots), returning per-row control
// points (coords).
func interpRows(grid [][]math.Point3, pv int, vparams, vknots []float64) ([][][]float64, error) {
	rows := make([][][]float64, len(grid))
	for a := range grid {
		ctrl, err := solveInterp(vparams, pv, vknots, coords3(grid[a]))
		if err != nil {
			return nil, err
		}
		rows[a] = ctrl
	}
	return rows, nil
}

// interpCols interpolates each column of the per-row control points across u, returning the final
// control net.
func interpCols(rows [][][]float64, pu int, uparams, uknots []float64) ([][]math.Point3, error) {
	nu, nv := len(rows), len(rows[0])
	net := make([][]math.Point3, nu)
	for a := range net {
		net[a] = make([]math.Point3, nv)
	}
	for b := range nv {
		col := make([][]float64, nu)
		for a := range nu {
			col[a] = rows[a][b]
		}
		ctrl, err := solveInterp(uparams, pu, uknots, col)
		if err != nil {
			return nil, err
		}
		for a := range nu {
			net[a][b] = math.P3(math.Scalar(ctrl[a][0]), math.Scalar(ctrl[a][1]), math.Scalar(ctrl[a][2]))
		}
	}
	return net, nil
}

// solveInterp solves the global-interpolation collocation system A·ctrl = pts for the control points,
// given parameters, degree and knots (each point a [x,y,z] coord row).
func solveInterp(params []float64, p int, knots []float64, pts [][]float64) ([][]float64, error) {
	a := collocation(params, p, knots)
	b := cloneRows(pts)
	if err := gaussSolve(a, b); err != nil {
		return nil, fmt.Errorf("geom.NetworkSurface: interpolation solve: %w", err)
	}
	return b, nil
}

// avgChordParams averages chord-length parameters across the grid's rows (the v-direction), so every
// row shares one v-parameterization.
func avgChordParams(grid [][]math.Point3, _ bool) []float64 {
	nv := len(grid[0])
	sum := make([]float64, nv)
	for _, row := range grid {
		ub, err := chordParams(coords3(row))
		if err != nil {
			continue
		}
		for j := range ub {
			sum[j] += ub[j]
		}
	}
	for j := range sum {
		sum[j] /= float64(len(grid))
	}
	return sum
}

// avgChordParamsRows averages chord params down the columns of the per-row control points (the
// u-direction).
func avgChordParamsRows(rows [][][]float64) []float64 {
	nu, nv := len(rows), len(rows[0])
	sum := make([]float64, nu)
	for b := range nv {
		col := make([][]float64, nu)
		for a := range nu {
			col[a] = rows[a][b]
		}
		ub, err := chordParams(col)
		if err != nil {
			continue
		}
		for a := range ub {
			sum[a] += ub[a]
		}
	}
	for a := range sum {
		sum[a] /= float64(nv)
	}
	return sum
}
