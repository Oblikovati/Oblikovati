// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The genus-1 COMPLEMENT torus face — the full torus minus a single oval cap, kept by an axis-parallel
// half-space cut (Oblikovati/Oblikovati#1375). The cap's small spiric oval is neither a band nor a simple
// patch, and the complement face wraps BOTH periods with the oval as its only boundary (a torus-minus-disk,
// genus 1 with one hole), so the band grid and metricPatchMesh both mis-mesh it (they'd cover the small
// oval interior, the cap, not the complement). torusComplementMesh instead charts the torus on a window
// centred on the oval: a curvature-conforming structured grid (like fullDomainGridMesh) tiles everything
// outside a small rectangular window around the oval, and metricPatchMesh fills the window MINUS the oval
// (a local, well-conditioned rect-with-hole). They share exact grid points on the window edges, so the two
// meshes weld. Validated by area: full-torus − cap, to the meshing deficit (#1375 follow-up).

// torusComplementMesh meshes the complement torus face from its hole loops (the oval cap's boundary). It
// charts the torus on a window centred on the oval (mapping the oval to unwrapped (u,v) via toUVLoop), so q
// sets the grid density the same way the full-domain grid does. Falls back to the full torus grid if the
// oval can't be charted (it wraps the seam) — better an over-filled mesh than a torn one.
func torusComplementMesh(s geom.Torus, holes3D [][]math.Point3, q Quality) *Mesh {
	oval3D := holes3D[0]
	ovalUV, ok := toUVLoop(s, oval3D, isPeriodic(s.UDomain()), isPeriodic(s.VDomain()))
	if !ok || len(holes3D) != 1 {
		return fullDomainGridMesh(s, q)
	}
	uc, vc := centroidUV(ovalUV)
	us := adaptiveParams(func(u float64) math.Point3 { return s.PointAt(u, vc) }, uc-stdmath.Pi, uc+stdmath.Pi, q.tol(), q.angleTol())
	vs := adaptiveParams(func(v float64) math.Point3 { return s.PointAt(uc, v) }, vc-stdmath.Pi, vc+stdmath.Pi, q.tol(), q.angleTol())
	i0, i1, j0, j1 := ovalWindow(us, vs, ovalUV)
	m := gridOutsideWindow(s, us, vs, i0, i1, j0, j1)
	win3D, winUV := windowRectLoop(s, us, vs, i0, i1, j0, j1)
	mergeMesh(m, metricPatchMesh(s, q, win3D, [][]math.Point3{oval3D}, winUV, [][]math.Point2{ovalUV}))
	return m
}

// centroidUV returns the mean (u,v) of a loop — the chart centre, so the oval sits in the middle of the
// [uc±π]×[vc±π] window and the chart seam falls on the far side of the torus, clear of the oval.
func centroidUV(loop []math.Point2) (uc, vc float64) {
	for _, p := range loop {
		uc, vc = uc+float64(p.X), vc+float64(p.Y)
	}
	n := float64(len(loop))
	return uc / n, vc / n
}

// ovalWindow returns the grid-index rectangle [i0,i1]×[j0,j1] that brackets the oval's (u,v) bounding box
// with a one-cell margin. It must always CONTAIN the oval: a near-full-wrap oval (one whose section barely
// clears the seam, at the single/two-oval transition) extends to the chart edges, so the window is allowed
// to reach them — otherwise the clamp would push the window inside the oval, the seam grid cells would tile
// over the oval region, and the complement would double-count (Oblikovati/Oblikovati#1375). The local patch
// then meshes the whole window (up to the full chart minus the oval), which stays well-conditioned.
func ovalWindow(us, vs []float64, ovalUV []math.Point2) (i0, i1, j0, j1 int) {
	uMin, uMax, vMin, vMax := loopUVBounds(ovalUV)
	i0 = math.Clamp(lastBelow(us, uMin)-1, 0, len(us)-2)
	i1 = math.Clamp(firstAbove(us, uMax)+1, i0+1, len(us)-1)
	j0 = math.Clamp(lastBelow(vs, vMin)-1, 0, len(vs)-2)
	j1 = math.Clamp(firstAbove(vs, vMax)+1, j0+1, len(vs)-1)
	return i0, i1, j0, j1
}

// gridOutsideWindow tiles the chart with a structured grid, omitting the cells inside the window
// [i0,i1]×[j0,j1] (those are filled by the window patch). Cells are wound outward by the surface normal.
func gridOutsideWindow(s geom.Torus, us, vs []float64, i0, i1, j0, j1 int) *Mesh {
	return structuredGridMeshSkip(s, us, vs, func(i, j int) bool {
		return i >= i0 && i < i1 && j >= j0 && j < j1 // inside the window
	})
}

// windowRectLoop returns the window's rectangular boundary as a CCW loop of EXACT grid points (3D and
// (u,v)), so the window patch shares those vertices with the structured grid and the two meshes weld.
func windowRectLoop(s geom.Torus, us, vs []float64, i0, i1, j0, j1 int) ([]math.Point3, []math.Point2) {
	var p3 []math.Point3
	var uv []math.Point2
	add := func(u, v float64) {
		p3 = append(p3, s.PointAt(u, v))
		uv = append(uv, math.P2(u, v))
	}
	for i := i0; i < i1; i++ {
		add(us[i], vs[j0])
	}
	for j := j0; j < j1; j++ {
		add(us[i1], vs[j])
	}
	for i := i1; i > i0; i-- {
		add(us[i], vs[j1])
	}
	for j := j1; j > j0; j-- {
		add(us[i0], vs[j])
	}
	return p3, uv
}

// loopUVBounds returns the (u,v) bounding box of a loop.
func loopUVBounds(loop []math.Point2) (uMin, uMax, vMin, vMax float64) {
	uMin, vMin = stdmath.Inf(1), stdmath.Inf(1)
	uMax, vMax = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, p := range loop {
		uMin, uMax = stdmath.Min(uMin, float64(p.X)), stdmath.Max(uMax, float64(p.X))
		vMin, vMax = stdmath.Min(vMin, float64(p.Y)), stdmath.Max(vMax, float64(p.Y))
	}
	return uMin, uMax, vMin, vMax
}

// lastBelow returns the largest index i with xs[i] ≤ x (0 if none), for a strictly increasing xs.
func lastBelow(xs []float64, x float64) int {
	for i, x0 := range slices.Backward(xs) {
		if x0 <= x {
			return i
		}
	}
	return 0
}

// firstAbove returns the smallest index i with xs[i] ≥ x (last index if none), for increasing xs.
func firstAbove(xs []float64, x float64) int {
	for i := range xs {
		if xs[i] >= x {
			return i
		}
	}
	return len(xs) - 1
}
