// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
)

// Which side of its loops a face lies on (M48/C3, Oblikovati/Oblikovati#3453).
//
// Green's theorem gives the measure of the region ENCLOSED by a face's loops. On an open surface
// (a plane, a cylinder, a cone) that is the face and there is nothing to decide: the complement is
// unbounded, so a bounded trim is always the enclosed side. On a CLOSED surface it is a genuine
// branch — one circle on a sphere bounds two caps, and `OuterLoop(Rev(edge))` names the far one —
// and the loops' enclosed measure alone cannot tell them apart. The branch is CERTIFIED against the
// geometry rather than assumed, and when the face turns out to be the complement its terms are the
// whole surface's minus the enclosed region's, which is exact; a sign flip would not be, because the
// complement of a region is not its negation.

// regionProbeGrid is the resolution of the search for one point strictly inside the enclosed region.
// It only has to find A point, not a particular one, so the count is a robustness margin for a
// slender region, not an accuracy parameter: nothing downstream depends on which point it returns.
const regionProbeGrid = 33

// loopRegionSigns orients every loop's boundary integral so their sum is the measure of the region
// the loops ENCLOSE: a loop at EVEN nesting depth is a top-level boundary and adds its own enclosed
// measure, one at odd depth is a hole and subtracts it.
//
// The role comes from the loops' own uv nesting, never from topo.Loop.IsOuter, and never from the
// largest loop. "Outer" is not well defined on a closed surface: the zone between two coaxial
// circles on a sphere is bounded by two NESTED loops, either of which a producer may hand over
// first, and naming the smaller one the outer integrates the region to a NEGATIVE area and flips
// that face's whole flux (Oblikovati/Oblikovati#3453 — a rod−ball cut read 55.63 against OCC's
// 3.27). A rod passing through a ball leaves the opposite shape: a face bounded by two DISJOINT
// loops and no outer one at all, where both are top-level and both must ADD. Nor can the stored
// winding carry the role alone — a boolean may leave a hole wound the SAME way as its enclosing
// loop, and a winding-only sum would then add that hole. Depth for the role and |∮| for the
// magnitude is right in all three.
func loopRegionSigns(loops []faceLoop) []float64 {
	polys := loopUVPolygons(loops)
	signs := make([]float64, len(loops))
	for i, fl := range loops {
		signs[i] = loopRegionSign(loopDepthIsEven(polys, i), loopSignedArea(fl))
	}
	return signs
}

// loopDepthIsEven reports whether loop i sits at an even nesting depth, counting how many of the
// other loops contain one of its points.
func loopDepthIsEven(polys [][]arcSample, i int) bool {
	if i >= len(polys) || len(polys[i]) == 0 {
		return true
	}
	p := polys[i][0]
	depth := 0
	for j, poly := range polys {
		if j != i && len(poly) >= 3 && polygonCrossings(poly, p.u, p.v)%2 == 1 {
			depth++
		}
	}
	return depth%2 == 0
}

// loopRegionSign multiplies one loop's stored boundary integral so it contributes +|enclosed| at
// even depth and −|enclosed| at odd depth. ∮_stored IS the signed enclosed measure, so the
// multiplier is the depth's sign times the sign of that measure.
func loopRegionSign(depthEven bool, signedArea float64) float64 {
	role := -1.0
	if depthEven {
		role = 1
	}
	if signedArea < 0 {
		return -role
	}
	return role
}

// regionIsFace reports whether the region enclosed by the face's loops IS the face (true) or its
// complement on a closed surface (false). ok is false when no interior probe point could be found,
// so the caller declines the whole body to the mesh path rather than guess the side.
func regionIsFace(f *topo.Face, loops []faceLoop) (isFace, ok bool) {
	u, v, found := regionInteriorUV(loops)
	if !found {
		return false, false
	}
	return brep.PointInFaceTrim(f, f.Geometry().PointAt(u, v)), true
}

// regionInteriorUV returns one parameter point strictly inside the region the loops enclose: of the
// grid points with an ODD even-odd crossing count — inside the outer loop and outside every hole —
// it takes the one FARTHEST from the boundary. Depth matters: a probe a hair inside the trim is a
// point where two independent classifiers may legitimately disagree, and the answer here selects a
// branch, so the point must be unambiguous rather than merely inside. The samples are the loops'
// unwrapped uv polylines, so a seam-crossing loop stays a simple polygon here.
func regionInteriorUV(loops []faceLoop) (u, v float64, ok bool) {
	polys := loopUVPolygons(loops)
	uLo, uHi, vLo, vHi := uvPolygonBounds(polys)
	if !(uLo < uHi && vLo < vHi) {
		return 0, 0, false
	}
	best := -1.0
	for i := 1; i < regionProbeGrid; i++ {
		for j := 1; j < regionProbeGrid; j++ {
			pu := uLo + (uHi-uLo)*float64(i)/regionProbeGrid
			pv := vLo + (vHi-vLo)*float64(j)/regionProbeGrid
			if d := uvDepthOf(polys, pu, pv); d > best {
				best, u, v = d, pu, pv
			}
		}
	}
	return u, v, best > 0
}

// uvDepthOf is how far (u, v) sits inside the region: its distance to the nearest boundary sample,
// or −1 when it is outside the region altogether.
func uvDepthOf(polys [][]arcSample, u, v float64) float64 {
	if !uvCrossingsOdd(polys, u, v) {
		return -1
	}
	nearest := stdmath.Inf(1)
	for _, poly := range polys {
		for _, s := range poly {
			nearest = stdmath.Min(nearest, stdmath.Hypot(s.u-u, s.v-v))
		}
	}
	return nearest
}

// loopUVPolygons flattens each loop's edge samples into one closed uv polyline per loop, INDEX-
// ALIGNED with loops: a degenerate loop keeps its (short) slot so a caller can address loop i.
func loopUVPolygons(loops []faceLoop) [][]arcSample {
	out := make([][]arcSample, len(loops))
	for i, fl := range loops {
		var poly []arcSample
		for _, le := range fl.edges {
			poly = append(poly, le.samples...)
		}
		out[i] = poly
	}
	return out
}

// uvPolygonBounds is the parameter-space box of every polygon.
func uvPolygonBounds(polys [][]arcSample) (uLo, uHi, vLo, vHi float64) {
	uLo, vLo = stdmath.Inf(1), stdmath.Inf(1)
	uHi, vHi = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, poly := range polys {
		for _, s := range poly {
			uLo, uHi = stdmath.Min(uLo, s.u), stdmath.Max(uHi, s.u)
			vLo, vHi = stdmath.Min(vLo, s.v), stdmath.Max(vHi, s.v)
		}
	}
	return uLo, uHi, vLo, vHi
}

// uvCrossingsOdd is the even-odd point-in-polygons test at (u, v), casting along +u and counting
// crossings over every loop: odd means inside the outer loop and outside its holes.
func uvCrossingsOdd(polys [][]arcSample, u, v float64) bool {
	crossings := 0
	for _, poly := range polys {
		if len(poly) >= 3 {
			crossings += polygonCrossings(poly, u, v)
		}
	}
	return crossings%2 == 1
}

// polygonCrossings counts how often the +u ray from (u, v) crosses one closed uv polyline.
func polygonCrossings(poly []arcSample, u, v float64) int {
	n := 0
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		if (a.v > v) == (b.v > v) {
			continue
		}
		if u < a.u+(v-a.v)/(b.v-a.v)*(b.u-a.u) {
			n++
		}
	}
	return n
}
