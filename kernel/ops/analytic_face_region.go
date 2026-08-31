// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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

// faceHoldsEnclosedRegion reports whether the face covers the region its loops enclose, or the
// complement of it on a closed surface. certain is false when the question cannot be settled, so the
// caller declines instead of integrating a guess.
//
// The loops' WINDING cannot answer it: a producer may wind a closed-surface face's loops either way
// and lean on Reversed to place the material, and every torus band in the corpus comes out clockwise
// whichever side it covers. So the question is put to the face's own TRIM — which of the two regions
// does this face own? — and confirmed against the SOLID.
//
// Both halves are needed, and each fails alone. The trim test alone was wrong while brep's
// closed-surface classifier projected loops orthographically onto the tangent plane, a 2-to-1 map
// that read a region larger than a hemisphere as outside itself. The material test alone cannot tell
// WHICH face owns a surface point: for the big cap of a ball joined to a rod, the deepest point of
// the enclosed region is the sphere's far pole, where the solid really does separate material from
// air — but that pole belongs to the TIP cap, so the material test answered "holds" for both faces
// and the big cap integrated as the small one.
func faceHoldsEnclosedRegion(f *topo.Face, loops []faceLoop) (holds, certain bool) {
	body := faceBody(f)
	if body == nil {
		return false, false // no assembled solid to ask
	}
	u, v, ok := regionProbeUV(loops)
	if !ok {
		return false, false
	}
	if !brep.PointInFaceTrim(f, f.Geometry().PointAt(u, v)) {
		return false, true // the face owns the other region: its terms are the complement's
	}
	return faceSpansMaterial(body, f, u, v)
}

// regionProbeUV returns a parameter point in the enclosed region for the side test. A loop that
// WRAPS a periodic seam is not a closed polygon in the plane, so the even-odd search cannot be
// asked about it — every torus band and every bore wall would get a meaningless answer. For those
// the probe steps inward from the boundary instead, which is well defined for any loop.
func regionProbeUV(loops []faceLoop) (u, v float64, ok bool) {
	if loopsWrapASeam(loops) {
		return regionInwardOfBoundary(loops)
	}
	return regionInteriorUV(loops)
}

// loopsWrapASeam reports whether any loop fails to return to its starting parameters, which is what
// makes it an open polyline in the covering space.
func loopsWrapASeam(loops []faceLoop) bool {
	for _, fl := range loops {
		if !closeUV(fl.netU, fl.netV, 0, 0) {
			return true
		}
	}
	return false
}

// regionInwardOfBoundary returns a point just inside the enclosed region, found by stepping off the
// longest boundary segment along its inward normal. Inward is the LEFT of the traversal taken in the
// loop's positive-measure direction, which loopRegionSigns already establishes, so this needs no
// containment test and works on a loop that wraps a seam.
func regionInwardOfBoundary(loops []faceLoop) (u, v float64, ok bool) {
	fl, sign, found := widestPositiveLoop(loops)
	if !found {
		return 0, 0, false
	}
	a, b, found := longestSegment(loopPolyline(fl))
	if !found {
		return 0, 0, false
	}
	du, dv := (b.u-a.u)*sign, (b.v-a.v)*sign
	n := stdmath.Hypot(du, dv)
	step := regionInwardStep * n
	return (a.u+b.u)/2 - dv/n*step, (a.v+b.v)/2 + du/n*step, true
}

// regionInwardStep is how far inward of the boundary the probe sits, as a fraction of the segment it
// steps off. Small enough to stay inside a slender band, large enough that the point is not on the
// boundary itself.
const regionInwardStep = 0.05 // tol:parametric — inward probe offset, relative to the local boundary

// widestPositiveLoop returns the top-level loop of greatest enclosed measure and the direction sign
// that walks it the positive-measure way.
func widestPositiveLoop(loops []faceLoop) (faceLoop, float64, bool) {
	signs := loopRegionSigns(loops)
	best, bestArea, found := faceLoop{}, 0.0, false
	for i, fl := range loops {
		area := stdmath.Abs(loopSignedArea(fl))
		if signs[i] > 0 && area >= bestArea {
			best, bestArea, found = fl, area, true
		}
	}
	if !found {
		return faceLoop{}, 0, false
	}
	if loopSignedArea(best) < 0 {
		return best, -1, true
	}
	return best, 1, true
}

// longestSegment returns the polyline's longest edge — the one least likely to be a corner sliver,
// so the inward step off its midpoint lands cleanly inside. The closing pair is NOT considered: on a
// seam-wrapping loop the last sample sits a whole period from the first, and that phantom segment
// would otherwise win every time.
func longestSegment(pts []arcSample) (a, b arcSample, ok bool) {
	best := 0.0
	for i := 0; i+1 < len(pts); i++ {
		p, q := pts[i], pts[i+1]
		if d := stdmath.Hypot(q.u-p.u, q.v-p.v); d > best {
			a, b, best, ok = p, q, d, true
		}
	}
	return a, b, ok
}

// faceSpansMaterial reports whether the surface point at (u, v) separates material from air along
// the face's outward normal — the test that decides which region the face covers. certain is false
// when the normal is degenerate there (a parametric pole), where no side can be read.
func faceSpansMaterial(body *topo.Body, f *topo.Face, u, v float64) (holds, certain bool) {
	s := f.Geometry()
	n := s.NormalAt(u, v)
	if f.Reversed() {
		n = n.Scale(-1)
	}
	if n.LengthSquared() == 0 {
		return false, false
	}
	step := n.AsUnit().AsVector().Scale(math.Scalar(geom.ResolutionForBox(body.RangeBox()).Sew()))
	p := s.PointAt(u, v)
	return brep.PointInside(body, p.TranslateBy(step.Scale(-1))) &&
		!brep.PointInside(body, p.TranslateBy(step)), true
}

// faceBody returns the solid the face bounds, or nil while the body is still being assembled.
func faceBody(f *topo.Face) *topo.Body {
	sh := f.Shell()
	if sh == nil {
		return nil
	}
	return sh.Body()
}

// FaceInteriorPoint returns one point strictly inside face f's trimmed region, taken from the
// analytic surface and its uv loops — never from a tessellation. It is the representative point a
// per-face gate classifies (M48/C3, Oblikovati/Oblikovati#3447). ok is false for a face whose loops
// cannot be reconstructed in uv, or whose region is too slender for the probe to land inside.
//
// The probe goes through regionProbeUV, so a face whose loops WRAP the parameter seam — a cone or
// cylinder band bounded by two full-turn loops — takes the inward-of-boundary route. Running the
// plain even-odd grid on such loops returned a point in the band the operation DISCARDED, and the
// per-face boolean certificate then refused a correct analytic result (Oblikovati/Oblikovati#3447).
//
// Example: p, ok := ops.FaceInteriorPoint(f) // ok ⇒ brep.PointInFaceTrim(f, p)
func FaceInteriorPoint(f *topo.Face) (math.Point3, bool) {
	s := f.Geometry()
	if len(f.Loops()) == 0 {
		uLo, uHi := s.UDomain()
		vLo, vHi := s.VDomain()
		if !allFinite(uLo, uHi, vLo, vHi) {
			return math.Point3{}, false
		}
		return s.PointAt((uLo+uHi)/2, (vLo+vHi)/2), true
	}
	loops, ok := buildFaceLoops(s, f)
	if !ok {
		return math.Point3{}, false
	}
	u, v, found := regionProbeUV(loops)
	if !found {
		return math.Point3{}, false
	}
	p := s.PointAt(u, v)
	if !brep.PointInFaceTrim(f, p) {
		return math.Point3{}, false // the enclosed side is the face's complement; no probe here
	}
	return p, true
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
	best, deepEnough := -1.0, regionProbeDeepEnough*stdmath.Hypot(uHi-uLo, vHi-vLo)
	for i := 1; i < regionProbeGrid && best < deepEnough; i++ {
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

// regionProbeDeepEnough is the depth, as a fraction of the region's parameter box diagonal, past
// which a probe is unambiguous and the search stops. The scan is over a grid of points each measured
// against every boundary sample, so on an ordinary face — where the first row already lands well
// inside — this turns a full sweep into a few rows. A slender region never reaches it and falls back
// to the full sweep, which is the case that needs one.
const regionProbeDeepEnough = 0.1 // tol:parametric — probe depth that ends the search, relative

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
