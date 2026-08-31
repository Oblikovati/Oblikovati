// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/brep"
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
// does this face own?
//
// It was once ALSO confirmed against the solid, by stepping a hair either way along the normal and
// asking which side held material. That step is gone. It was there because the trim test used to be
// wrong — brep's closed-surface classifier projected loops orthographically onto the tangent plane, a
// 2-to-1 map that read a region larger than a hemisphere as outside itself — and once that
// classifier began answering from the nearest boundary FOOT, the material step stopped adding
// anything and started subtracting: a probe a Sew() away from a curved face is a knife-edge query on
// a boundary built by marching, and it answered "no material" on the correct side of a ball drilled
// by a stud, so a 15.708 sphere zone integrated as its 298.451 complement. Dropping it moved four
// corpus cases into the analytic regime with nothing else changed (Oblikovati/Oblikovati#3489).
func faceHoldsEnclosedRegion(f *topo.Face, loops []faceLoop) (holds, certain bool) {
	u, v, ok := regionProbeUV(loops)
	if !ok {
		return false, false
	}
	if !brep.PointInFaceTrim(f, f.Geometry().PointAt(u, v)) {
		return false, true // the face owns the other region: its terms are the complement's
	}
	return true, true
}

// regionProbeUV returns a parameter point in the enclosed region for the side test. A loop that
// WRAPS a periodic seam is not a closed polygon in the plane, so the even-odd search cannot be
// asked about it — every torus band and every bore wall would get a meaningless answer. For those
// the probe steps inward from the boundary instead, which is well defined for any loop.
func regionProbeUV(loops []faceLoop) (u, v float64, ok bool) {
	if loopsWrapASeam(loops) {
		return bandInteriorUV(loops)
	}
	return regionInteriorUV(loops)
}

// loopsWrapASeam reports whether any loop travels a WHOLE PERIOD in a parameter instead of returning
// to where it started, which is what makes it an open polyline in the covering space.
//
// The question is asked of the period, not of zero. Net travel is accumulated from ParamAt round
// trips, so a loop that closes perfectly still reports a residue — measured, 3.0e-8 in u on a torus
// section and 1.8e-7 in v on its planar cap. Comparing that against a bare absolute epsilon called
// every one of those loops seam-wrapping and sent ordinary bounded faces down the BAND path, which
// reasons about a band's two rims and has no meaning for them. A wrap is one period (6.283 here) or
// none; rounding the net travel to whole periods cannot be fooled by the residue, and a parameter
// that does not wrap at all (a plane's) has period 0 and can never report one.
func loopsWrapASeam(loops []faceLoop) bool {
	uPeriod, vPeriod := loopsPeriod(loops, bandAxis{}), loopsPeriod(loops, bandAxis{alongV: true})
	for _, fl := range loops {
		if wholePeriodOffset(fl.netU, uPeriod) != 0 || wholePeriodOffset(fl.netV, vPeriod) != 0 {
			return true
		}
	}
	return false
}

// bandInteriorUV returns a point deep inside a band whose loops WRAP the parameter seam. Such a
// band is bounded in the parameter that closes and unbounded in the one that wraps, so at any fixed
// station of the wrapping parameter its boundary curves sit above and below: the midpoint between
// them is interior by construction, and it is far from the boundary rather than a hair off it.
//
// Stepping inward off a boundary chord instead — the previous construction — could land OUTSIDE the
// true region and go undetected, because the check available at that point compares against the
// loops' SAMPLED polygon, which does not track the true trim curve at that scale. A probe 5.2e-3
// outside the operand read as in-trim, the per-face gate then correctly refused a correct body, and
// a blind hole fell to a 1830-face rescue (Oblikovati/Oblikovati#2247).
func bandInteriorUV(loops []faceLoop) (u, v float64, ok bool) {
	axis := bandAxisOf(loops)
	along, across, found := bandInterior(loops, axis)
	if !found {
		return 0, 0, false
	}
	u, v = axis.pointOf(along, across)
	return u, v, true
}

// bandInterior is bandInteriorUV in the band's own (along, across) frame.
func bandInterior(loops []faceLoop, axis bandAxis) (along, across float64, ok bool) {
	samples := allLoopSamples(loops, axis)
	if len(samples) < 2 {
		return 0, 0, false
	}
	holes := nonWrappingPolygons(loops)
	for _, i := range bandStationOrder(len(samples)) {
		station := samples[i].u
		lo, hi, found := vSpanAt(samples, station)
		if !found {
			continue
		}
		mid := (lo + hi) / 2
		hu, hv := axis.pointOf(station, mid)
		if !uvCrossingsOdd(holes, hu, hv) {
			return station, mid, true
		}
	}
	return 0, 0, false // every station tried put the midpoint in a hole
}

// nonWrappingPolygons are the loops that close in the plane — the face's HOLES on a band, since a
// band's own bounding curves are the ones that wrap. The midpoint of the boundary's span can fall
// inside one of these, which is outside the face, so a station that does is rejected rather than
// trusted.
func nonWrappingPolygons(loops []faceLoop) [][]arcSample {
	var out [][]arcSample
	for i, fl := range loops {
		if closeUV(fl.netU, fl.netV, 0, 0) {
			out = append(out, loopUVPolygons(loops)[i])
		}
	}
	return out
}

// bandStationOrder walks candidate stations from the middle of the sample range outward, so an
// ordinary band answers on the first try and a holed one still gets alternatives to try.
func bandStationOrder(n int) []int {
	out := make([]int, 0, bandStationTries)
	for k := range bandStationTries {
		out = append(out, n*(2*k+1)/(2*bandStationTries))
	}
	return out
}

// bandStationTries is how many stations across the band are tried before declining. A band with
// holes needs more than one; a plain band answers on the first.
const bandStationTries = 9

// allLoopSamples flattens every loop's uv samples, ordered by the wrapping parameter so the median
// is a station the band actually spans. The u values are folded into ONE period first: the two rims
// of a band are unwrapped onto different branches of the covering space — one walks 0→2π, the next
// 2π→4π — so their raw parameters never meet even though the rims sit directly above one another,
// and a station window over raw u would see only one of them.
func allLoopSamples(loops []faceLoop, axis bandAxis) []arcSample {
	period := loopsPeriod(loops, axis)
	var out []arcSample
	for _, fl := range loops {
		for _, le := range fl.edges {
			for _, sp := range le.samples {
				a, x := axis.coordsOf(sp)
				out = append(out, arcSample{t: sp.t, u: foldU(a, period), v: x})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].u < out[j].u })
	return out
}

// loopsPeriod is the face's period in the wrapping parameter, or 0 when it does not wrap.
func loopsPeriod(loops []faceLoop, axis bandAxis) float64 {
	for _, fl := range loops {
		if p := axis.periodOf(fl); p > 0 {
			return p
		}
	}
	return 0
}

// vSpanAt returns the lowest and highest v the boundary reaches near the given u station. found is
// false when the boundary has no thickness there — a station at the very end of the band, where the
// midpoint would sit on the boundary itself.
func vSpanAt(samples []arcSample, station float64) (lo, hi float64, found bool) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	window := bandStationWindow * uSpanOf(samples)
	for _, s := range samples {
		if stdmath.Abs(s.u-station) <= window {
			lo, hi = stdmath.Min(lo, s.v), stdmath.Max(hi, s.v)
		}
	}
	return lo, hi, hi-lo > 0
}

// bandStationWindow is how much of the band's wrapping extent counts as "at this station" when
// reading the boundary's v span. Wide enough to catch both boundary curves through their sampling,
// narrow enough that the span is local rather than the band's whole height.
const bandStationWindow = 0.02 // tol:parametric — station window, relative to the band's u extent

// uSpanOf is the total extent the samples cover in the wrapping parameter.
func uSpanOf(samples []arcSample) float64 {
	return samples[len(samples)-1].u - samples[0].u
}

// FaceInteriorPoint returns one point strictly inside face f's trimmed region, taken from the
// analytic surface and its uv loops — never from a tessellation. It is the representative point a
// per-face gate classifies (M48/C3, Oblikovati/Oblikovati#3447).
//
// It DECLINES for a face whose loops wrap the parameter seam, and that is deliberate. A gate exists
// to disprove a result; a probe it had to guess at can disprove a CORRECT one, and the cost of that
// is not a weaker gate but a right answer thrown away — a five-face blind hole demoted to a
// 1830-face faceted rescue. On a wrapping band no probe here has proved trustworthy: an even-odd
// grid returns a point in the band the operation discards, and a step inward from the boundary can
// land outside the true region while the loops' sampled polygon still calls it inside. Until a
// wrapping band's interior can be certified exactly, the gate skips those faces under its own
// "skipped rather than failed" rule and the volume bracket carries them.
//
// (The integrator's own side test does probe a wrapping band, through bandInteriorUV. That is sound
// there for a different reason: a wrong answer is caught by the vector-area closure post-condition,
// which declines the body rather than shipping it.)
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
	if !ok || loopsWrapASeam(loops) {
		return math.Point3{}, false // see the seam-wrapping note above
	}
	u, v, found := regionInteriorUV(loops)
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
//
// Every loop is unwrapped from its OWN first sample, so two loops of one face can come back on
// different branches of the covering space — a cylinder wall's outer loop at u ∈ [−2π, 0] with its
// window holes at [0, 2π). Nothing is wrong with either, but a nesting test comparing them across
// branches reads a hole as OUTSIDE the loop that contains it, and the hole is then ADDED to the
// region: a drilled cylinder wall measured 240.81 where 211.57 is right, and the body's volume came
// out 9.8% over (Oblikovati/Oblikovati#3489). Each polygon is shifted onto the first one's branch as
// a WHOLE, by whole periods, which preserves its shape — folding each point separately would cut any
// polygon that straddles the fold.
func loopUVPolygons(loops []faceLoop) [][]arcSample {
	out := make([][]arcSample, len(loops))
	for i, fl := range loops {
		var poly []arcSample
		for _, le := range fl.edges {
			poly = append(poly, le.samples...)
		}
		out[i] = poly
	}
	return alignPolygonBranches(out, loopsPeriod(loops, bandAxis{}), loopsPeriod(loops, bandAxis{alongV: true}))
}

// alignPolygonBranches brings each inner loop onto the branch of the outer one, per parameter.
//
// The shift is SELF-VERIFYING: a whole period is applied only when it lands the inner loop inside the
// outer loop's interval and the loop is not already there. Choosing the NEAREST branch instead is
// wrong, and wrong in a way a real body reaches — a sphere zone's two rims sit exactly half a period
// apart in their chart, which is the tie point of "nearest", so rounding moved a loop that was
// already placed correctly and the belt went on to name the caps and report four times the area.
// Refusing to move anything a shift cannot justify leaves that case alone and still repairs the one
// this exists for.
func alignPolygonBranches(polys [][]arcSample, uPeriod, vPeriod float64) [][]arcSample {
	if len(polys) < 2 {
		return polys
	}
	alignAlongAxis(polys, uPeriod, bandAxis{})
	alignAlongAxis(polys, vPeriod, bandAxis{alongV: true})
	return polys
}

// alignAlongAxis places the inner loops inside the outer loop's interval in ONE parameter. loops[0]
// is the outer loop; a face with holes only (a closed surface's outerless face) has no interval that
// should contain the rest, and the containment test then justifies no shift, which is correct.
func alignAlongAxis(polys [][]arcSample, period float64, axis bandAxis) {
	if period <= 0 {
		return
	}
	outer, ok := polygonRangeAlong(polys[0], axis)
	if !ok {
		return
	}
	for i := 1; i < len(polys); i++ {
		inner, ok := polygonRangeAlong(polys[i], axis)
		if !ok || outer.holds(inner) {
			continue
		}
		shift := wholePeriodOffset(outer.mid()-inner.mid(), period)
		if shift == 0 || !outer.holds(inner.shifted(shift)) {
			continue // no whole number of periods puts this loop inside the outer one
		}
		polys[i] = shiftPolygonAlong(polys[i], shift, axis)
	}
}

// paramRange is a closed interval in one surface parameter.
type paramRange struct{ lo, hi float64 }

// holds reports whether the whole of r lies within p.
func (p paramRange) holds(r paramRange) bool { return r.lo >= p.lo && r.hi <= p.hi }

// mid is the interval's centre, the anchor a branch is chosen by.
func (p paramRange) mid() float64 { return (p.lo + p.hi) / 2 }

// shifted moves the whole interval.
func (p paramRange) shifted(d float64) paramRange { return paramRange{p.lo + d, p.hi + d} }

// polygonRangeAlong is the extent a polygon covers in one parameter.
func polygonRangeAlong(poly []arcSample, axis bandAxis) (paramRange, bool) {
	if len(poly) == 0 {
		return paramRange{}, false
	}
	at := func(sp arcSample) float64 {
		if axis.alongV {
			return sp.v
		}
		return sp.u
	}
	out := paramRange{at(poly[0]), at(poly[0])}
	for _, sp := range poly {
		out.lo, out.hi = stdmath.Min(out.lo, at(sp)), stdmath.Max(out.hi, at(sp))
	}
	return out, true
}

// shiftPolygonAlong moves a whole polygon in one parameter, which preserves its shape — folding each
// point separately would cut any polygon that straddles the fold.
func shiftPolygonAlong(poly []arcSample, shift float64, axis bandAxis) []arcSample {
	out := make([]arcSample, len(poly))
	for i, sp := range poly {
		out[i] = sp
		if axis.alongV {
			out[i].v += shift
			continue
		}
		out[i].u += shift
	}
	return out
}

// wholePeriodOffset is the whole number of periods that best closes a gap, or zero when the
// parameter does not wrap or the gap is already under half a period. (Distinct from
// holed_cylinder_wall.go's branchShift, which brings one angle INTO a given range.)
func wholePeriodOffset(gap, period float64) float64 {
	if period <= 0 {
		return 0
	}
	return period * stdmath.Round(gap/period)
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

// bandLoopSigns signs each loop of a face whose loops WRAP the parameter seam, where the
// enclosed-measure convention a closed polygon uses does not apply.
//
// Green's conjugate identity for a band is a DIFFERENCE of its two boundary curves taken the same
// way round: ∫∫ g du dv = ∫ [P(u, v_hi) − P(u, v_lo)] du. So a rim is normalised to a +u traversal by
// the sign of its own net travel, then signed by its ROLE — the rim BELOW the region adds, the one
// above subtracts. Summing them instead, which is what a stored-orientation sum does whenever a
// producer winds both rims the same way, reports a band larger than the whole surface it lies on:
// a cone ∩ box band read a lateral area of 337.86 against a frustum of 295.19
// (Oblikovati/Oblikovati#3489).
//
// ONE wrapping loop is a different shape and keeps its stored traversal: it carries both rims joined
// by seam edges, the seam contributes nothing because du = 0 along it, and the antiderivative's base
// sits on the lower rim, so the walk already telescopes to the region. That is how a drilled bore
// wall integrates exactly, and this must not disturb it.
//
// A loop that does NOT wrap is a hole in the band, and subtracts its own enclosed measure.
func bandLoopSigns(loops []faceLoop, form greenAxis) []float64 {
	signs := make([]float64, len(loops))
	axis := bandAxisOf(loops)
	rims := wrappingLoopCount(loops)
	station, interiorAcross, haveInterior := bandInterior(loops, axis)
	for i, fl := range loops {
		switch {
		case !loopWraps(fl):
			signs[i] = loopRegionSign(false, loopSignedArea(fl))
		case rims == 1 || !haveInterior:
			signs[i] = 1
		default:
			signs[i] = bandRimSign(fl, axis, form, station, interiorAcross)
		}
	}
	return signs
}

// bandRimSign is one rim's multiplier: its travel direction, so every rim reads as a +u traversal,
// times its role, which is +1 below the region and −1 above it.
func bandRimSign(fl faceLoop, axis bandAxis, form greenAxis, station, interiorAcross float64) float64 {
	direction := 1.0
	if axis.netOf(fl) < 0 {
		direction = -1
	}
	above := false
	if a, ok := loopAcrossNear(fl, axis, station); ok {
		above = a > interiorAcross
	}
	return direction * bandRimRole(above, form)
}

// bandRimRole is +1 for the rim whose side of the region ADDS under the Green form in use, −1 for
// the other. Which side that is depends on the form, because the two are conjugates: the
// u-antiderivative form integrates ∮ Q dv and reconstructs the region as Q(u_hi) − Q(u_lo), so the
// rim at the LARGER across-coordinate adds; the v-antiderivative form integrates −∮ P du and gives
// P(v_hi) − P(v_lo), which makes the SMALLER one add. Reading the role without the form flips every
// band whose wrapping parameter is the other one — a torus band that wraps in v came out at
// −173.99 for an area.
func bandRimRole(above bool, form greenAxis) float64 {
	if above == form.dv {
		return 1
	}
	return -1
}

// bandAxis names which parameter a band wraps. Everything about a band is stated in ALONG (the one
// that wraps, where the rims travel) and ACROSS (the one that closes, where the region has
// thickness): a torus band can wrap either way, and reading a v-wrapping band's rims off u gives
// both the direction and the role of every rim wrongly.
type bandAxis struct{ alongV bool }

// netOf is the loop's net travel in the wrapping parameter.
func (a bandAxis) netOf(fl faceLoop) float64 {
	if a.alongV {
		return fl.netV
	}
	return fl.netU
}

// coordsOf splits one sample into (along, across).
func (a bandAxis) coordsOf(sp arcSample) (along, across float64) {
	if a.alongV {
		return sp.v, sp.u
	}
	return sp.u, sp.v
}

// pointOf rebuilds surface parameters from (along, across).
func (a bandAxis) pointOf(along, across float64) (u, v float64) {
	if a.alongV {
		return across, along
	}
	return along, across
}

// periodOf is the wrapping parameter's period on this loop, or 0 when it does not wrap.
func (a bandAxis) periodOf(fl faceLoop) float64 {
	for _, le := range fl.edges {
		if a.alongV {
			if le.vPeriod > 0 {
				return le.vPeriod
			}
			continue
		}
		if le.uPeriod > 0 {
			return le.uPeriod
		}
	}
	return 0
}

// bandAxisOf reports which parameter the loops wrap. A face where none of them wraps is not a band
// and never reaches here through bandLoopSigns; it reads as u-wrapping, which is the identity frame.
func bandAxisOf(loops []faceLoop) bandAxis {
	for _, fl := range loops {
		if !closeUV(fl.netU, 0, 0, 0) {
			return bandAxis{}
		}
		if !closeUV(fl.netV, 0, 0, 0) {
			return bandAxis{alongV: true}
		}
	}
	return bandAxis{}
}

// loopWraps reports whether this loop alone fails to return to its starting parameters.
func loopWraps(fl faceLoop) bool { return !closeUV(fl.netU, fl.netV, 0, 0) }

// wrappingLoopCount is how many of the face's loops wrap the seam.
func wrappingLoopCount(loops []faceLoop) int {
	n := 0
	for _, fl := range loops {
		if loopWraps(fl) {
			n++
		}
	}
	return n
}

// loopAcrossNear returns the loop's ACROSS coordinate where it passes closest to the given ALONG
// station, comparing stations MODULO the wrapping period: two rims of one band are unwrapped onto
// different branches (one walks 0→2π, the next 2π→4π), so their raw parameters never meet even
// though the rims sit directly one side of the other.
func loopAcrossNear(fl faceLoop, axis bandAxis, station float64) (float64, bool) {
	period := axis.periodOf(fl)
	target, nearest, across, found := foldU(station, period), stdmath.Inf(1), 0.0, false
	for _, le := range fl.edges {
		for _, sp := range le.samples {
			a, x := axis.coordsOf(sp)
			if d := stdmath.Abs(foldU(a, period) - target); d < nearest {
				nearest, across, found = d, x, true
			}
		}
	}
	return across, found
}

// foldU brings a parameter into one period, so branches of the covering space compare.
func foldU(u, period float64) float64 {
	if period <= 0 {
		return u
	}
	return u - period*stdmath.Floor(u/period)
}

// singleCycleBand reports a face bounded by exactly ONE seam-wrapping loop — a cycle carrying both
// rims joined by seam edges, as opposed to a pair of separate rims.
func singleCycleBand(loops []faceLoop) bool {
	return loopsWrapASeam(loops) && wrappingLoopCount(loops) == 1
}
