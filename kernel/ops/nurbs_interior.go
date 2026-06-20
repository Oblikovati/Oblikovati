// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// maxInteriorCells bounds the grid so a pathological curvature estimate cannot explode the node
// count; minInteriorCells keeps a curved face from collapsing to a single coarse quad.
const (
	maxInteriorCells = 64
	minInteriorCells = 2
)

// adaptiveInteriorNodes returns interior (u,v) nodes for a trimmed NURBS face: a staggered grid
// whose spacing follows the surface's curvature (a flat face gets none, a curved face gets dense
// nodes at the same Quality), kept STRICTLY inside the trim — inside the outer pcurve, outside every
// hole, and at least a margin off the boundary so no node spills past the trim or sits on it (the
// over-enclosure that inflated the volume in every naive attempt — ADR-0030 / M24 F02). The pcurves
// are the smooth march-projected boundary (F01); their point-in-polygon test is therefore reliable.
func adaptiveInteriorNodes(s geom.Surface, outer []math.Point2, holes [][]math.Point2, q Quality, refine float64) [][2]float64 {
	umin, umax, vmin, vmax := uvBBox(outer)
	stepU, stepV := adaptiveStep(s, umin, umax, vmin, vmax, q)
	if refine > 0 && refine < 1 {
		// Fold-driven refinement asks for a denser grid (#585), floored at maxInteriorCells so a
		// re-mesh cannot explode the node count even on a face that was already finely sampled.
		stepU = stdmath.Max(stepU*refine, (umax-umin)/maxInteriorCells)
		stepV = stdmath.Max(stepV*refine, (vmax-vmin)/maxInteriorCells)
	}
	if stepU <= 0 || stepV <= 0 {
		return nil
	}
	margin := 0.3 * stdmath.Min(stepU, stepV)
	var pts [][2]float64
	row := 0
	for v := vmin + stepV/2; v < vmax; v += stepV {
		off := 0.0
		if row%2 == 1 {
			off = stepU / 2 // stagger alternate rows for better-shaped triangles
		}
		row++
		for u := umin + stepU/2 + off; u < umax; u += stepU {
			if p := [2]float64{u, v}; clearOfTrim(outer, holes, p, margin) {
				pts = append(pts, p)
			}
		}
	}
	return pts
}

// adaptiveStep picks the (u,v) grid spacing so the surface's chord deviation over a cell stays
// within q.ChordTolerance. The whole-region sagitta D scales ~quadratically with cell size, so a
// cell fraction f ≈ sqrt(tol/D) hits the tolerance; the result is clamped to [min,max] cells.
func adaptiveStep(s geom.Surface, umin, umax, vmin, vmax float64, q Quality) (stepU, stepV float64) {
	uExt, vExt := umax-umin, vmax-vmin
	if uExt <= 0 || vExt <= 0 {
		return 0, 0
	}
	tol := q.ChordTolerance
	if tol <= 0 {
		tol = DefaultQuality().ChordTolerance
	}
	d := regionSagitta(s, umin, umax, vmin, vmax)
	frac := 1.0
	if d > tol {
		frac = stdmath.Sqrt(tol / d)
	}
	return clampStep(uExt*frac, uExt), clampStep(vExt*frac, vExt)
}

func clampStep(step, ext float64) float64 {
	return stdmath.Max(ext/maxInteriorCells, stdmath.Min(step, ext/minInteriorCells))
}

// regionSagitta estimates how far the surface bulges from the flat bilinear quad spanning the (u,v)
// region — a curvature proxy: 0 for a planar patch, large for a tightly-curved one.
func regionSagitta(s geom.Surface, umin, umax, vmin, vmax float64) float64 {
	c00, c10 := s.PointAt(umin, vmin), s.PointAt(umax, vmin)
	c01, c11 := s.PointAt(umin, vmax), s.PointAt(umax, vmax)
	var max float64
	for _, f := range [][2]float64{{0.25, 0.25}, {0.5, 0.5}, {0.75, 0.75}, {0.25, 0.75}, {0.75, 0.25}} {
		u, v := umin+f[0]*(umax-umin), vmin+f[1]*(vmax-vmin)
		flat := bilerp(c00, c10, c01, c11, f[0], f[1])
		if d := float64(s.PointAt(u, v).DistanceTo(flat)); d > max {
			max = d
		}
	}
	return max
}

// bilerp bilinearly interpolates the four corners at (fu, fv) ∈ [0,1]².
func bilerp(c00, c10, c01, c11 math.Point3, fu, fv float64) math.Point3 {
	x := lerp2(c00.X, c10.X, c01.X, c11.X, fu, fv)
	y := lerp2(c00.Y, c10.Y, c01.Y, c11.Y, fu, fv)
	z := lerp2(c00.Z, c10.Z, c01.Z, c11.Z, fu, fv)
	return math.P3(x, y, z)
}

func lerp2(a00, a10, a01, a11 math.Scalar, fu, fv float64) math.Scalar {
	bottom := float64(a00) + (float64(a10)-float64(a00))*fu
	top := float64(a01) + (float64(a11)-float64(a01))*fu
	return math.Scalar(bottom + (top-bottom)*fv)
}

// clearOfTrim reports whether p is inside the trim AND at least `margin` (in u and v) from the
// boundary, by requiring p and its four axis neighbours to be inside — so no node sits on or just
// past the boundary, the seam that lets a node spill and over-enclose.
func clearOfTrim(outer []math.Point2, holes [][]math.Point2, p [2]float64, margin float64) bool {
	for _, off := range [][2]float64{{0, 0}, {margin, 0}, {-margin, 0}, {0, margin}, {0, -margin}} {
		if !insideUVTrim(outer, holes, [2]float64{p[0] + off[0], p[1] + off[1]}) {
			return false
		}
	}
	return true
}

// uvBBox returns the (u,v) bounding box of a pcurve loop.
func uvBBox(loop []math.Point2) (umin, umax, vmin, vmax float64) {
	umin, vmin = stdmath.Inf(1), stdmath.Inf(1)
	umax, vmax = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, p := range loop {
		umin, umax = stdmath.Min(umin, float64(p.X)), stdmath.Max(umax, float64(p.X))
		vmin, vmax = stdmath.Min(vmin, float64(p.Y)), stdmath.Max(vmax, float64(p.Y))
	}
	return umin, umax, vmin, vmax
}
