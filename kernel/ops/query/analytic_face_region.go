// SPDX-License-Identifier: GPL-2.0-only

package query

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
// measures[i] is loop i's OWN boundary integral (∮), which is the signed enclosed measure the rule
// needs — see enclosedTerms for why the sampled polyline's shoelace cannot carry that sign.
func loopRegionSigns(loops []faceLoop, measures []float64) []float64 {
	polys, per := loopUVPolygons(loops), loopsUVPeriod(loops)
	signs := make([]float64, len(loops))
	for i := range loops {
		signs[i] = loopRegionSign(loopDepthIsEven(polys, i, per), measures[i])
	}
	return signs
}

// loopDepthIsEven reports whether loop i sits at an even nesting depth, counting how many of the
// other loops contain one of its points.
func loopDepthIsEven(polys [][]arcSample, i int, per uvPeriod) bool {
	if i >= len(polys) || len(polys[i]) == 0 {
		return true
	}
	p := polys[i][0]
	depth := 0
	for j, poly := range polys {
		if j != i && pointInLoopPolygon(poly, p.u, p.v, per) {
			depth++
		}
	}
	return depth%2 == 0
}

// loopRegionSign multiplies one loop's stored boundary integral so it contributes +|enclosed| at
// even depth and −|enclosed| at odd depth. ∮_stored IS the signed enclosed measure, so the
// multiplier is the depth's sign times the sign of that measure.
func loopRegionSign(depthEven bool, signedMeasure float64) float64 {
	role := -1.0
	if depthEven {
		role = 1
	}
	if signedMeasure < 0 {
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
	if !loopsWrapASeam(loops) {
		u, v, ok := regionProbeUV(f.Geometry(), loops)
		if !ok {
			return false, false
		}
		return brep.PointInFaceTrim(f, f.Geometry().PointAt(u, v)), true
	}
	return bandSideOfEnclosedRegion(f, loops)
}

// regionProbeUV returns a parameter point in the enclosed region for the side test. A loop that
// WRAPS a periodic seam is not a closed polygon in the plane, so the even-odd search cannot be
// asked about it — every torus band and every bore wall would get a meaningless answer. For those
// the probe steps inward from the boundary instead, which is well defined for any loop.
func regionProbeUV(s geom.Surface, loops []faceLoop) (u, v float64, ok bool) {
	if !loopsWrapASeam(loops) {
		return regionInteriorUV(loops)
	}
	// A CAP is not a band: its one rim has no v-span for the band probe to read, so every station
	// declined and the side could not be certified at all (ADR-0062). Its interior lies between the
	// rim and the pole its contour closes at, which capPoleContour names.
	if u, v, ok = capInteriorUV(s, loops); ok {
		return u, v, true
	}
	return bandInteriorUV(loops)
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

// loopsCloseTheirWalk reports whether every loop's unwrapped uv walk RETURNS to where it started —
// Green's theorem's own precondition, since ∮ over an open polyline measures nothing. A whole number
// of periods is a return: that is a band crossing the parameter seam, and its conjugate form closes it.
//
// The walk can fail to close even though the loop is a closed circuit in 3-D, and two causes were
// measured on the blend-parity corpus. A loop's edge uses are taken in stored order, and that order is
// not always a connected traversal: a cylinder arm's loop came back as (wall line, far rim, wall line,
// near rim) — every edge present, none adjacent to its neighbour — so the walk jumped the arm's whole
// height twice and read a lateral area of −12075.67 where 9629.06 is right. And a surface with a POLE
// collapses a whole isoparametric edge onto one 3-D point, so ParamAt cannot recover the parameter
// along it and the walk restarts on the wrong branch of that edge.
//
// Neither is recoverable HERE — one is a topology-ordering question, the other needs the pcurve the
// producer discarded — so the face is refused before any integral is built and the tessellated
// fallback measures the body. The vector-area closure post-condition cannot substitute for this: a
// full band's outward vector area is zero whichever region it takes, and the two pole-degenerate
// flanks above were mirror images whose residuals CANCELLED, so both bodies passed the closure with a
// 1.7% wrong volume (Oblikovati/Oblikovati#3453).
func loopsCloseTheirWalk(loops []faceLoop) bool {
	uPeriod, vPeriod := loopsPeriod(loops, bandAxis{}), loopsPeriod(loops, bandAxis{alongV: true})
	for _, fl := range loops {
		extent := loopUVExtent(fl)
		if !closesUpToPeriod(fl.netU, uPeriod, extent) || !closesUpToPeriod(fl.netV, vPeriod, extent) {
			return false
		}
	}
	return true
}

// closesUpToPeriod reports whether a net travel is a return: zero, or a whole number of periods in a
// parameter that HAS one. A non-periodic parameter has period 0, where only zero is a return.
//
// What is left after the whole periods is judged RELATIVE TO THE CONTOUR'S OWN SIZE, not against an
// absolute parametric epsilon. A closed curve can carry a residue that is real and irreducible: a
// spiric section TURNS at each end, where the branches meet at w = 1 ± 1e-16, and acos of that puts
// the shared endpoint 2.98e-8 away in u — double rounding amplified by a square root, on a loop that
// closes exactly. An absolute 1e-9 called every torus oval an open contour and refused faces the
// integrator computes exactly. The breaks this gate exists to catch are nothing like that size: a
// disconnected edge order jumped 57.57 of a face's height, a pole-degenerate flank jumped its whole
// unit domain. Relative to the contour they are 1e0; the spiric residue is 1e-8.
func closesUpToPeriod(net, period, extent float64) bool {
	residue := stdmath.Abs(net - wholePeriodOffset(net, period))
	return residue <= closedWalkRelTol*extent
}

// closedWalkRelTol is how much of its own parametric extent a contour may fail to close by. It is set
// at the vector-area closure post-condition's tolerance, because that is what the gate feeds: a gap
// this size perturbs the boundary integral by about as much, so anything it admits is still caught
// downstream, and anything larger is a break rather than a rounding residue.
const closedWalkRelTol = 1e-6 // tol:parametric — relative to the loop's own uv extent

// loopUVExtent is how far a loop reaches in uv, the natural scale for judging its closure residue.
func loopUVExtent(fl faceLoop) float64 {
	uLo, uHi, vLo, vHi := 0.0, 0.0, 0.0, 0.0
	seen := false
	for _, le := range fl.edges {
		for _, sp := range le.samples {
			if !seen {
				uLo, uHi, vLo, vHi, seen = sp.u, sp.u, sp.v, sp.v, true
				continue
			}
			uLo, uHi = stdmath.Min(uLo, sp.u), stdmath.Max(uHi, sp.u)
			vLo, vHi = stdmath.Min(vLo, sp.v), stdmath.Max(vHi, sp.v)
		}
	}
	return stdmath.Max(uHi-uLo, vHi-vLo)
}

// FaceInteriorPoint returns one point strictly inside face f's trimmed region, taken from the
// analytic surface and its uv loops — never from a tessellation. It is the representative point a
// per-face gate classifies (M48/C3, Oblikovati/Oblikovati#3447).
//
// A face whose loops WRAP the parameter seam is probed through regionProbeUV — the band and cap
// probes the integrator's own side test uses — and not, as it once was, declined outright. The
// decline was the right answer while the probe was a guess: a gate exists to disprove a result, and a
// probe it had to guess at can disprove a CORRECT one, at the cost of a five-face blind hole demoted
// to a 1830-face faceted rescue. What makes the probe safe is not the probe but the CERTIFICATION
// below it: whatever uv the band or cap rule proposes, the point is returned only when
// brep.PointInFaceTrim — an independent classifier, not these loops' polygon — agrees it is on the
// face. A probe that lands in the band the operation discards fails that test and still declines, so
// the gate never gains a probe it cannot stand behind, and it stops skipping every ordinary bore wall
// and rod tunnel the general pipeline builds (ADR-0061 stage 4).
//
// THE POINT THIS RETURNS MOVED, for every seam-wrapping face (Oblikovati/Oblikovati#3553). It used to
// be the MIDDLE of the boundary's span at a station — every torus band, every bore wall, every rod
// tunnel the boolean certificate probes — and it is now a THIRD of that span. This is a public query
// and the change is API-visible across a whole class of faces, not a tuning of one shape: any caller
// that assumed the mid-band point, or that placed a fixture around it, reads a different point now.
//
// Why a third is the right point and the middle was not: the middle is the one fraction a charted
// region's artificial SLIT occupies at every station at once, because a slit is a constant-parameter
// line and a region symmetric about it puts it at exactly 1/2 of every span. A probe there asks the
// classifier a question it cannot answer — an even-odd count ON a contour answers by which side its ray
// was cast from — and the answer then tracks the model's last bit. The #2247 invariant this probe
// carries is DEPTH, not the midpoint: a third of the band is depth, and it is the same depth at every
// discretisation. bandInteriorCandidates has the construction and the measurement.
//
// What has NOT changed is the certification: whatever uv the band or cap rule proposes, the point is
// returned only when brep.PointInFaceTrim agrees it is on the face.
//
// Example: p, ok := query.FaceInteriorPoint(f) // ok ⇒ brep.PointInFaceTrim(f, p)
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
	if u, v, found := regionProbeUV(s, loops); found {
		if p := s.PointAt(u, v); brep.PointInFaceTrim(f, p) {
			return p, true
		}
	}
	return faceComplementPoint(f, loops)
}

// faceComplementPoint probes the side of the chart OPPOSITE the region the loops enclose, for the
// face that holds it. On a closed surface a loop set bounds two regions and the face may own either;
// the enclosed-region probe answers only for the near one, so a torus with a bore through it — the
// face carrying 296.088 of a bored ring's 296.107 of area — had NO representative point at any bore
// radius, and certifyBooleanFaces, which skips a face it cannot probe, therefore never examined it
// (Oblikovati/Oblikovati#3516). The probe is certified the same way the near one is: it is returned
// only when brep.PointInFaceTrim, an independent classifier, agrees the point is on the face.
func faceComplementPoint(f *topo.Face, loops []faceLoop) (math.Point3, bool) {
	s := f.Geometry()
	u, v, found := faceComplementUV(s, loops)
	if !found {
		return math.Point3{}, false
	}
	p := s.PointAt(u, v)
	if !brep.PointInFaceTrim(f, p) {
		return math.Point3{}, false
	}
	return p, true
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
func bandLoopSigns(loops []faceLoop, measures []float64, form greenAxis) []float64 {
	signs := make([]float64, len(loops))
	axis := bandAxisOf(loops)
	rims := wrappingLoopCount(loops)
	station, interiorAcross, haveInterior := bandInterior(loops, axis)
	for i, fl := range loops {
		switch {
		case !loopWraps(fl):
			signs[i] = loopRegionSign(false, measures[i])
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
