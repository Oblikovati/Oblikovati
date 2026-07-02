// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// maxInteriorCells bounds the grid so a pathological curvature estimate cannot explode the node
// count; minInteriorCells keeps a curved face from collapsing to a single coarse quad.
const (
	maxInteriorCells = 64
	minInteriorCells = 2
)

// CodeTessellateCapSaturated marks a face whose interior refinement hit the cell-size floor with the
// surface still bulging past the chord tolerance — the mesh may not meet tolerance there, recorded
// instead of silently exceeding it (Oblikovati#1412).
const CodeTessellateCapSaturated diag.Code = "tessellate.cap-saturated"

// localRefineThreshold is how far above the global step's per-cell deviation budget a base cell must
// bulge before it is LOCALLY subdivided. It is > 1 so a uniformly-curved face — whose cells already sit
// ≈ at the budget by the global step's construction — is left exactly as before (no regression, #1412);
// only a cell markedly more curved than the face's average (an off-centre bump, a torus tube against
// its flatter ring) crosses it and gets concentrated refinement.
const localRefineThreshold = 1.5

// adaptiveInteriorNodes returns interior (u,v) nodes for a trimmed NURBS face, and whether the local
// refinement SATURATED (hit the cell-size floor with the surface still over tolerance). The base is the
// curvature-scaled staggered grid as before — a flat face gets none, a uniformly-curved one a uniform
// grid — kept STRICTLY inside the trim (inside the outer pcurve, outside every hole, a margin off the
// boundary, so no node spills past the trim, ADR-0030 / M24 F02). ON TOP of it, any base cell that
// bulges well past the global step's average is recursively subdivided, so the refinement CONCENTRATES
// where curvature varies (a bump, a torus tube) without densifying the rest — fixing the under/over-
// tessellation of the single global step from one diagonal sagitta sample (#1412).
// localRefine asks adaptiveInteriorNodes to ADD concentrated curvature nodes on top of the base grid.
// Only the robust all-points constrainedDelaunay path (metricCDTPatch) passes it: the holed-wall path
// uses constrainedDelaunayRefined, whose post-constraint insertion can fold, and extra interior nodes
// there tear the mesh — so it takes the base grid only and relies on the saturation diagnostic instead.
func adaptiveInteriorNodes(s geom.Surface, outer []math.Point2, holes [][]math.Point2, q Quality, refine float64, localRefine bool) ([][2]float64, bool) {
	umin, umax, vmin, vmax := uvBBox(outer)
	stepU, stepV := adaptiveStep(s, umin, umax, vmin, vmax, q)
	if refine > 0 && refine < 1 {
		// Fold-driven refinement asks for a denser grid (#585), floored at maxInteriorCells so a
		// re-mesh cannot explode the node count even on a face that was already finely sampled.
		stepU = stdmath.Max(stepU*refine, (umax-umin)/maxInteriorCells)
		stepV = stdmath.Max(stepV*refine, (vmax-vmin)/maxInteriorCells)
	}
	if stepU <= 0 || stepV <= 0 {
		return nil, false
	}
	margin := 0.3 * stdmath.Min(stepU, stepV)
	pts := baseStaggeredGrid(umin, umax, vmin, vmax, stepU, stepV, outer, holes, margin)
	saturated := stepSaturates(s, umin, umax, vmin, vmax, stepU, stepV, q.tol())
	if localRefine && refineLocalCurvature(s, umin, umax, vmin, vmax, stepU, stepV, q, outer, holes, margin, &pts) {
		saturated = true
	}
	return pts, saturated
}

// stepSaturates reports whether the base step is clamped to the cell-size floor with a floor-sized cell
// STILL bulging past the chord tolerance — a genuinely large high-curvature face the cap cannot resolve,
// the case #1412 surfaces as a diagnostic instead of silently under-tessellating. It samples cells at
// the centre and quarter positions (not just a corner, which on a centred dome is flat) and reports true
// if the most-curved of them exceeds tol.
func stepSaturates(s geom.Surface, umin, umax, vmin, vmax, stepU, stepV, tol float64) bool {
	floorU, floorV := (umax-umin)/maxInteriorCells, (vmax-vmin)/maxInteriorCells
	if stepU > floorU*1.001 && stepV > floorV*1.001 {
		return false // step is not at the floor → the grid already meets the tolerance
	}
	for _, c := range [][2]float64{{0.5, 0.5}, {0.25, 0.25}, {0.75, 0.75}, {0.25, 0.75}, {0.75, 0.25}} {
		u0 := umin + c[0]*(umax-umin-stepU)
		v0 := vmin + c[1]*(vmax-vmin-stepV)
		if regionSagitta(s, u0, u0+stepU, v0, v0+stepV) > tol {
			return true
		}
	}
	return false
}

// baseStaggeredGrid is the curvature-scaled staggered interior grid (alternate rows offset by half a
// step for better-shaped triangles), every node clear of the trim by margin.
func baseStaggeredGrid(umin, umax, vmin, vmax, stepU, stepV float64, outer []math.Point2, holes [][]math.Point2, margin float64) [][2]float64 {
	var pts [][2]float64
	row := 0
	for v := vmin + stepV/2; v < vmax; v += stepV {
		off := 0.0
		if row%2 == 1 {
			off = stepU / 2
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

// refineLocalCurvature walks the base cells and recursively subdivides any whose surface sagitta exceeds
// localRefineThreshold·tol, appending the new finer nodes to pts. It returns whether any cell saturated
// the cell-size floor while still over tolerance.
func refineLocalCurvature(s geom.Surface, umin, umax, vmin, vmax, stepU, stepV float64, q Quality, outer []math.Point2, holes [][]math.Point2, margin float64, pts *[][2]float64) bool {
	tol := q.tol() * localRefineThreshold
	minU, minV := (umax-umin)/maxInteriorCells, (vmax-vmin)/maxInteriorCells
	saturated := false
	for u0 := umin; u0 < umax-1e-12; u0 += stepU {
		u1 := stdmath.Min(u0+stepU, umax)
		for v0 := vmin; v0 < vmax-1e-12; v0 += stepV {
			refineCell(s, u0, u1, v0, stdmath.Min(v0+stepV, vmax), tol, minU, minV, outer, holes, margin, pts, &saturated)
		}
	}
	return saturated
}

// refineCell subdivides a (u,v) cell while its surface bulges past tol, emitting the four sub-cell
// centres (the new finer nodes that are clear of the trim) at each split, down to the cell-size floor —
// where, if still over tol, it flags saturation for the caller to record.
func refineCell(s geom.Surface, u0, u1, v0, v1, tol, minU, minV float64, outer []math.Point2, holes [][]math.Point2, margin float64, pts *[][2]float64, saturated *bool) {
	if regionSagitta(s, u0, u1, v0, v1) <= tol {
		return
	}
	if u1-u0 <= minU && v1-v0 <= minV {
		*saturated = true
		return
	}
	um, vm := (u0+u1)/2, (v0+v1)/2
	for _, c := range [4][2]float64{{(u0 + um) / 2, (v0 + vm) / 2}, {(um + u1) / 2, (v0 + vm) / 2}, {(u0 + um) / 2, (vm + v1) / 2}, {(um + u1) / 2, (vm + v1) / 2}} {
		if clearOfTrim(outer, holes, c, margin) {
			*pts = append(*pts, c)
		}
	}
	refineCell(s, u0, um, v0, vm, tol, minU, minV, outer, holes, margin, pts, saturated)
	refineCell(s, um, u1, v0, vm, tol, minU, minV, outer, holes, margin, pts, saturated)
	refineCell(s, u0, um, vm, v1, tol, minU, minV, outer, holes, margin, pts, saturated)
	refineCell(s, um, u1, vm, v1, tol, minU, minV, outer, holes, margin, pts, saturated)
}

// recordCapSaturation records a cap-saturation diagnostic on the mesh when the interior refinement
// could not meet the chord tolerance within the cell-size floor — so a face that under-tessellates is
// visible to the caller, not silently shipped (#1412).
func recordCapSaturation(m *Mesh, saturated bool, q Quality) {
	if m == nil || !saturated {
		return
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeTessellateCapSaturated,
		Severity: diag.Warning,
		Detail:   fmt.Sprintf("interior refinement saturated the %d-cell floor still above chord tol %g", maxInteriorCells, q.tol()),
	})
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

// clampStep stays a wrapper over math.Clamp (#1652): it owns the derived
// per-axis bounds [ext/maxInteriorCells, ext/minInteriorCells], not new arithmetic.
func clampStep(step, ext float64) float64 {
	return math.Clamp(step, ext/maxInteriorCells, ext/minInteriorCells)
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

// bilerp bilinearly interpolates the four corners at (fu, fv) ∈ [0,1]²:
// two nested [math.Point3.Lerp] passes (#1654), no bespoke arithmetic.
func bilerp(c00, c10, c01, c11 math.Point3, fu, fv float64) math.Point3 {
	return c00.Lerp(c10, fu).Lerp(c01.Lerp(c11, fu), fv)
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
