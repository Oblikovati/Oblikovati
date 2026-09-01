// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Trimmed curved-face tessellation — the (u,v) GRID CONSTRUCTION (M48 #2224 split of tessellate_trim.go).
// When a trim region is an iso-aligned rectangle whose opposite edges sample identically — every analytic
// fillet/blend face, axial cylinder/cone walls, and full-period bands — a STRUCTURED grid of thin iso quads
// tessellates it watertight and with correct curved area. This file builds those grids (rectangle, single-
// and doubly-periodic bands), emits their cells wound outward, and carries the grid-line tolerances/helpers.

// SeamAngularTol snaps periodic parameter samples at the 0/2π seam. It compares RADIANS,
// so it is scale-free by construction; the metric (non-periodic) grid direction derives
// its tolerance from its own span instead (gridTol) so µm and metre parts grid
// identically (#1610 — the old shared 1e-6 was absolute in the metric direction).
const SeamAngularTol = 1e-6 // tol:angular (radians; periodic-seam snap)

// trimGridRelTol is the dimensionless fraction of a grid axis' span within which two
// parameter samples count as the same grid line.
const trimGridRelTol = 1e-6 // tol:numeric (dimensionless fraction of the axis span)

// gridTol is the same-grid-line tolerance for one axis' samples: a fixed fraction of the
// samples' span (zero for a degenerate axis — near() is inclusive, so exact duplicates
// still collapse).
func gridTol(xs []float64) float64 {
	lo, hi := minMax(xs)
	return trimGridRelTol * (hi - lo)
}

// isoRectangleGrid returns the sorted u and v grid lines when the UV boundary is an
// iso-aligned rectangle whose opposite edges carry matching parameter samples (so a
// structured grid is watertight and conforms to the boundary). ok=false otherwise.
func isoRectangleGrid(loop []math.Point2) (us, vs []float64, ok bool) {
	uMin, uMax, vMin, vMax := bounds2D(loop)
	tolU := trimGridRelTol * float64(uMax-uMin)
	tolV := trimGridRelTol * float64(vMax-vMin)
	var bottomU, topU, leftV, rightV []float64
	for _, p := range loop {
		onB, onT := near(p.Y, vMin, tolV), near(p.Y, vMax, tolV)
		onL, onR := near(p.X, uMin, tolU), near(p.X, uMax, tolU)
		if !onB && !onT && !onL && !onR {
			return nil, nil, false // a vertex off the bbox border — not a rectangle
		}
		appendIf(&bottomU, p.X, onB)
		appendIf(&topU, p.X, onT)
		appendIf(&leftV, p.Y, onL)
		appendIf(&rightV, p.Y, onR)
	}
	bottomU, topU = sortUnique(bottomU, tolU), sortUnique(topU, tolU)
	leftV, rightV = sortUnique(leftV, tolV), sortUnique(rightV, tolV)
	if !sameGrid(bottomU, topU, tolU) || !sameGrid(leftV, rightV, tolV) {
		return nil, nil, false // opposite edges sample differently → would leave T-junctions
	}
	return bottomU, leftV, true
}

// structuredGridMesh tessellates the surface over the us×vs parameter grid as thin iso
// quads (two triangles each), wound outward with true per-vertex normals. Border grid
// points reproduce the exact boundary vertices (ParamAt is the inverse of PointAt), so the
// mesh conforms to the shared edge discretization.
func structuredGridMesh(s geom.Surface, us, vs []float64) *Mesh {
	return structuredGridMeshSkip(s, us, vs, nil)
}

// structuredGridMeshSkip is [structuredGridMesh] with an optional per-cell skip predicate (omit a cell
// when skip(i,j) is true) — the torus complement grid omits the window cells the local patch fills.
func structuredGridMeshSkip(s geom.Surface, us, vs []float64, skip func(i, j int) bool) *Mesh {
	m := &Mesh{}
	idx := make([][]int, len(us))
	for i, u := range us {
		idx[i] = make([]int, len(vs))
		for j, v := range vs {
			idx[i][j] = m.AddVertex(s.PointAt(u, v), s.NormalAt(u, v))
		}
	}
	for i := 0; i+1 < len(us); i++ {
		for j := 0; j+1 < len(vs); j++ {
			if skip != nil && skip(i, j) {
				continue
			}
			emitCellOutward(m, s, us[i], us[i+1], vs[j], vs[j+1], idx[i][j], idx[i+1][j], idx[i+1][j+1], idx[i][j+1])
		}
	}
	return m
}

// emitCellOutward adds the two triangles of a grid cell, winding them so their geometric
// normal agrees with the surface normal at the cell centre.
func emitCellOutward(m *Mesh, s geom.Surface, u0, u1, v0, v1 float64, a, b, c, d int) {
	flip := m.CellNormal(a, b, c).Dot(s.NormalAt((u0+u1)/2, (v0+v1)/2)) < 0
	if flip {
		m.AddTriangle(a, c, b)
		m.AddTriangle(a, d, c)
		return
	}
	m.AddTriangle(a, b, c)
	m.AddTriangle(a, c, d)
}

// periodicBandGrid handles a full-period curved band — a complete cylinder/cone side whose
// loop runs up the seam, around a full circle, down the seam and back, so toUVLoops can't
// unwrap it (the loop spans the whole 2π). Its trim is simply the entire period in the
// periodic parameter and the boundary's own range in the other. The grid reuses the
// boundary's parameter samples (the shared circle-edge discretization) so the band stays
// watertight with its caps, and closes the period so the last cell wraps back to the seam.
// ok=false when it isn't a single-periodic band or carries holes (those need a constrained
// triangulation — still the full-domain fallback).
func periodicBandGrid(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (us, vs []float64, ok bool) {
	uPer, vPer := IsPeriodic(s.UDomain()), IsPeriodic(s.VDomain())
	if uPer == vPer || len(holes3D) != 0 {
		return nil, nil, false // need exactly one periodic direction and no holes
	}
	uu := make([]float64, len(outer3D))
	vv := make([]float64, len(outer3D))
	for i, p := range outer3D {
		uu[i], vv[i] = s.ParamAt(p)
	}
	if uPer {
		us, vs = bracketPeriod(uu), sortUnique(vv, gridTol(vv))
	} else {
		us, vs = sortUnique(uu, gridTol(uu)), bracketPeriod(vv)
	}
	if len(us) < 2 || len(vs) < 2 {
		return nil, nil, false // degenerate (e.g. a cone closing to its apex — one rim circle)
	}
	// A genuine band's boundary is two full-period edge circles, so the NON-periodic direction has
	// exactly two distinct values. More than two means the boundary wanders in that direction (a
	// sphere cap straddling the pole, not a latitude band): gridding its full us×vs bounding box
	// would cover non-trim area and tear at the pole, so reject it for the boundary triangulator.
	if (uPer && len(vs) != 2) || (vPer && len(us) != 2) {
		return nil, nil, false
	}
	return us, vs, true
}

// doublyPeriodicBandGrid meshes a band on a DOUBLY-periodic surface (a torus) whose boundary wraps
// exactly ONE parameter direction around its full period — two edge circles joined by a seam — and is
// bounded in the other. The torus rim fillet is such a band: it wraps the axis (u over [0,2π]) and
// spans only a quarter of the tube (v). periodicBandGrid rejects it (a torus is periodic in BOTH
// directions), so the wrap direction is read from the boundary's own parameter span here instead.
func doublyPeriodicBandGrid(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (us, vs []float64, ok bool) {
	if !IsPeriodic(s.UDomain()) || !IsPeriodic(s.VDomain()) || len(holes3D) != 0 {
		return nil, nil, false // need a doubly-periodic surface and no holes
	}
	uu := make([]float64, len(outer3D))
	vv := make([]float64, len(outer3D))
	for i, p := range outer3D {
		uu[i], vv[i] = s.ParamAt(p)
	}
	uWrap, vWrap := spansFullPeriod(uu), spansFullPeriod(vv)
	if uWrap == vWrap {
		return nil, nil, false // need exactly one direction wrapping the full period
	}
	if uWrap {
		us, vs = bracketPeriod(uu), sortUnique(vv, gridTol(vv))
	} else {
		us, vs = sortUnique(uu, gridTol(uu)), bracketPeriod(vv)
	}
	if len(us) < 2 || len(vs) < 2 {
		return nil, nil, false
	}
	return us, vs, true
}

// spansFullPeriod reports whether a periodic parameter's samples cover essentially its whole [0,2π]
// period (a band that wraps the seam), versus a bounded sub-range (the tube of a rim-fillet torus, or a
// perpendicular torus cut's kept arc). It measures the OCCUPIED span as 2π minus the single largest gap —
// so a sub-arc that merely STRADDLES the 0/2π seam (min near 0, max near 2π, but a wide gap in between) is
// correctly seen as bounded, not full (Oblikovati#1375).
func spansFullPeriod(g []float64) bool {
	s := sortUnique(g, SeamAngularTol)
	if len(s) < 2 {
		return false
	}
	maxGap := s[0] + 2*stdmath.Pi - s[len(s)-1] // the wrap gap (last sample → first, across the seam)
	for i := 0; i+1 < len(s); i++ {
		if gap := s[i+1] - s[i]; gap > maxGap {
			maxGap = gap
		}
	}
	return 2*stdmath.Pi-maxGap > 2*stdmath.Pi-0.5 // occupied span (all but the largest gap) is nearly full
}

// bracketPeriod normalises a periodic direction's samples to span a CLOSED [0, 2π]: a seam sample whose
// angle wrapped to just under 2π (a seam vertex at angle 0 read back as 2π−ε from a tiny negative
// coordinate) is snapped to 0 — it is the 0 column — and an explicit 2π closer is appended. So the
// first and last columns are the shared seam and EVERY segment, including the one across the seam, is a
// grid cell. Without this the seam column landed only at ~2π, dropping the [0, first-sample] cell and
// leaving a one-cell crack against the caps. closedDomainMesh then wraps [0, 2π] onto one seam column.
func bracketPeriod(g []float64) []float64 {
	out := make([]float64, 0, len(g)+1)
	for _, x := range g {
		if x > 2*stdmath.Pi-SeamAngularTol {
			x = 0 // a seam sample read back as ~2π is the 0 column
		}
		out = append(out, x)
	}
	out = sortUnique(out, SeamAngularTol)
	return append(out, 2*stdmath.Pi)
}

// bounds2D returns the UV bounding box of the points.
func bounds2D(pts []math.Point2) (uMin, uMax, vMin, vMax float64) {
	uMin, vMin = stdmath.Inf(1), stdmath.Inf(1)
	uMax, vMax = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, p := range pts {
		uMin, uMax = stdmath.Min(uMin, p.X), stdmath.Max(uMax, p.X)
		vMin, vMax = stdmath.Min(vMin, p.Y), stdmath.Max(vMax, p.Y)
	}
	return uMin, uMax, vMin, vMax
}

// near reports whether two parameter samples coincide within tol (inclusive, so exact
// duplicates collapse even on a degenerate zero-span axis).
func near(a, b, tol float64) bool { return stdmath.Abs(a-b) <= tol }

func appendIf(dst *[]float64, x float64, cond bool) {
	if cond {
		*dst = append(*dst, x)
	}
}

// sortUnique returns the values sorted ascending with near-duplicates (within tol) collapsed.
func sortUnique(xs []float64, tol float64) []float64 {
	sort.Float64s(xs)
	out := xs[:0:0]
	for _, x := range xs {
		if len(out) == 0 || !near(out[len(out)-1], x, tol) {
			out = append(out, x)
		}
	}
	return out
}

// sameGrid reports whether two sorted grids have the same lines (within tol).
func sameGrid(a, b []float64, tol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !near(a[i], b[i], tol) {
			return false
		}
	}
	return true
}
