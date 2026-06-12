// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// This file traces the zero set of a scalar field over a surface's parameter domain by
// marching squares + per-edge bisection (M22-F10 PBI-244). It is the shared machinery
// behind surface↔surface intersection (the field is the signed distance to the other
// surface) and silhouette extraction (the field is the surface normal · view direction).
// Each emitted point lies on the base surface exactly (via PointAt) and on the field's
// zero to the bisection tolerance.

// SurfaceGrid bounds the (u, v) window and resolution for tracing a field's zero over a
// surface. A non-positive step count defaults to traceDefaultSteps; bounds that are left
// zero are filled from the surface's finite parameter domain (an infinite direction must
// be given an explicit window).
type SurfaceGrid struct {
	UMin, UMax, VMin, VMax float64
	USteps, VSteps         int
}

const (
	traceDefaultSteps = 96
	traceBisectIter   = 48
	traceChainTol     = 1e-7
)

// resolveGrid fills a SurfaceGrid's zero/absent fields from the surface's finite domain
// and the default resolution. An unbounded direction with no explicit window collapses to
// a degenerate range, yielding no crossings (the caller must supply a window for it).
func resolveGrid(s Surface, g SurfaceGrid) SurfaceGrid {
	uLo, uHi := s.UDomain()
	vLo, vHi := s.VDomain()
	if g.UMin == 0 && g.UMax == 0 {
		g.UMin, g.UMax = finiteOr(uLo, 0), finiteOr(uHi, 0)
	}
	if g.VMin == 0 && g.VMax == 0 {
		g.VMin, g.VMax = finiteOr(vLo, 0), finiteOr(vHi, 0)
	}
	if g.USteps <= 0 {
		g.USteps = traceDefaultSteps
	}
	if g.VSteps <= 0 {
		g.VSteps = traceDefaultSteps
	}
	return g
}

// finiteOr returns x when finite, else the fallback (so an unbounded domain bound does not
// poison the grid window).
func finiteOr(x, fallback float64) float64 {
	if stdmath.IsInf(x, 0) {
		return fallback
	}
	return x
}

// scalarField is a function of surface parameters whose zero set is traced.
type scalarField func(u, v float64) float64

// traceZeroOnSurface returns polylines approximating the zero set of f over the surface's
// parameter window, by marching squares with per-edge bisection refinement.
func traceZeroOnSurface(s Surface, f scalarField, grid SurfaceGrid) [][]math.Point3 {
	g := resolveGrid(s, grid)
	if g.UMax <= g.UMin || g.VMax <= g.VMin {
		return nil
	}
	segs := marchSquares(s, f, g)
	return chainSegments3D(segs)
}

// gridSeg is one marching-squares segment (two 3D points on the zero contour).
type gridSeg struct{ a, b math.Point3 }

// marchSquares samples f on the grid and emits the zero-contour segment(s) of each cell.
func marchSquares(s Surface, f scalarField, g SurfaceGrid) []gridSeg {
	du := (g.UMax - g.UMin) / float64(g.USteps)
	dv := (g.VMax - g.VMin) / float64(g.VSteps)
	var segs []gridSeg
	for i := 0; i < g.USteps; i++ {
		for j := 0; j < g.VSteps; j++ {
			u0, v0 := g.UMin+float64(i)*du, g.VMin+float64(j)*dv
			segs = appendCellSegments(segs, s, f, u0, v0, u0+du, v0+dv)
		}
	}
	return segs
}

// appendCellSegments emits the zero crossing(s) of one grid cell [u0,u1]×[v0,v1]. It
// gathers each edge's sign-change crossing and pairs them (two crossings ⇒ one segment;
// the four-crossing saddle ⇒ two segments).
func appendCellSegments(segs []gridSeg, s Surface, f scalarField, u0, v0, u1, v1 float64) []gridSeg {
	f00, f10, f11, f01 := f(u0, v0), f(u1, v0), f(u1, v1), f(u0, v1)
	var pts []math.Point3
	pts = appendCross(pts, s, f, u0, v0, f00, u1, v0, f10) // bottom
	pts = appendCross(pts, s, f, u1, v0, f10, u1, v1, f11) // right
	pts = appendCross(pts, s, f, u1, v1, f11, u0, v1, f01) // top
	pts = appendCross(pts, s, f, u0, v1, f01, u0, v0, f00) // left
	switch len(pts) {
	case 2:
		return append(segs, gridSeg{pts[0], pts[1]})
	case 4:
		return append(segs, gridSeg{pts[0], pts[1]}, gridSeg{pts[2], pts[3]})
	default:
		return segs
	}
}

// appendCross adds the bisected zero crossing of one cell edge when its endpoints straddle
// the field's zero.
func appendCross(pts []math.Point3, s Surface, f scalarField, ua, va, fa, ub, vb, fb float64) []math.Point3 {
	if !straddlesZero(fa, fb) {
		return pts
	}
	return append(pts, bisectEdge(s, f, ua, va, fa, ub, vb))
}

// straddlesZero reports a sign change between two field samples, classifying an exact
// zero with the negative side (a <= 0). This consistent tie-break is what lets the tracer
// capture a contour that lies exactly on grid lines (e.g. an equator on a parameter row),
// the classic marching-squares degeneracy, without double-counting it. Samples within
// floating noise of zero snap to exact zero first: a node sitting ON the contour evaluates
// to ±1e-16 with a sign that varies row to row (numerically-derived normals), and without
// the snap adjacent columns alternate crossings, doubling the contour as a zigzag
// (M07-F07 silhouette wires, Oblikovati/Oblikovati#630).
func straddlesZero(a, b float64) bool {
	return (snapTinyToZero(a) <= 0) != (snapTinyToZero(b) <= 0)
}

// traceFieldEps is the field magnitude treated as exactly zero (fields here are
// unit-vector dots, so 1e-12 is far below any genuine sample).
const traceFieldEps = 1e-12

func snapTinyToZero(v float64) float64 {
	if stdmath.Abs(v) < traceFieldEps {
		return 0
	}
	return v
}

// bisectEdge refines the zero crossing along a cell edge in parameter space and returns
// the 3D point on the base surface there.
func bisectEdge(s Surface, f scalarField, ua, va, fa, ub, vb float64) math.Point3 {
	lo, hi := 0.0, 1.0
	for k := 0; k < traceBisectIter; k++ {
		mid := (lo + hi) / 2
		um, vm := lerp(ua, ub, mid), lerp(va, vb, mid)
		fm := f(um, vm)
		if fm == 0 {
			return s.PointAt(um, vm)
		}
		if straddlesZero(fa, fm) {
			hi = mid
		} else {
			lo, fa = mid, fm
		}
	}
	m := (lo + hi) / 2
	return s.PointAt(lerp(ua, ub, m), lerp(va, vb, m))
}

// lerp linearly interpolates between a and b at t.
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// chainSegments3D links marching-squares segments into ordered polylines by matching
// shared endpoints within tolerance.
func chainSegments3D(segs []gridSeg) [][]math.Point3 {
	used := make([]bool, len(segs))
	var out [][]math.Point3
	for i := range segs {
		if used[i] {
			continue
		}
		used[i] = true
		poly := []math.Point3{segs[i].a, segs[i].b}
		growPolyline(segs, used, &poly)
		out = append(out, poly)
	}
	return out
}

// growPolyline extends a polyline from its tail by attaching unused segments that share
// the free endpoint (forward only; marching-squares contours are locally 1-manifold).
func growPolyline(segs []gridSeg, used []bool, poly *[]math.Point3) {
	for {
		tail := (*poly)[len(*poly)-1]
		j, far, ok := nextSegment(segs, used, tail)
		if !ok {
			return
		}
		used[j] = true
		*poly = append(*poly, far)
	}
}

// nextSegment finds an unused segment with an endpoint coincident with tail.
func nextSegment(segs []gridSeg, used []bool, tail math.Point3) (int, math.Point3, bool) {
	for j := range segs {
		if used[j] {
			continue
		}
		switch {
		case tail.IsEqualTo(segs[j].a, traceChainTol):
			return j, segs[j].b, true
		case tail.IsEqualTo(segs[j].b, traceChainTol):
			return j, segs[j].a, true
		}
	}
	return 0, math.Point3{}, false
}
