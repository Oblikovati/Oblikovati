// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
)

// The loops' uv POLYGONS and point-in-polygon on a periodic parameter plane.
//
// A face's loops are sampled curves, and several of the region rules need them as plain polygons in one
// branch of the covering space: the nesting rule that gives each loop its sign, the hole rejection the
// band probe runs, and the complement probe. Developing them, carrying them onto one branch and
// counting crossings against them is one responsibility, and it lives here because
// analytic_face_region.go was over this repo's 500-line file rule (review 2, M1). Nothing here changed
// in the move.

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

// uvCrossingsOdd is the even-odd point-in-polygons test at (u, v) over every loop: an ODD number of
// loops contains it, which means inside the outer loop and outside its holes.
func uvCrossingsOdd(polys [][]arcSample, u, v float64, per uvPeriod) bool {
	inside := 0
	for _, poly := range polys {
		if pointInLoopPolygon(poly, u, v, per) {
			inside++
		}
	}
	return inside%2 == 1
}

// pointInLoopPolygon is the even-odd point-in-polygon test asked in the SURFACE's space rather than
// in one branch of its chart: on a periodic parameter u and u±period name the same point, so the
// probe is tried at every whole-period translate the polygon's own extent can hold, and any hit
// counts. A parameter that does not wrap has period 0 and only the untranslated probe.
//
// Testing a single branch is wrong for a loop that STRADDLES the branch cut. A wrapped emboss
// footprint on a cone came back on u ∈ [−0.034, 0.034] while the face's outer loop occupied
// [−2π, 0]. No whole period puts the footprint inside that interval — it hangs off both ends — so
// alignPolygonBranches rightly declined to move it, the one-branch test then read the footprint as
// OUTSIDE the loop enclosing it, and the hole was ADDED to the face instead of subtracted: 1333.86
// cm² on a face whose undrilled band measures 1332.86 (Oblikovati/Oblikovati#3505; #3489 fixed the
// sibling case where a whole loop sat on the wrong branch — this is the one no shift can repair).
func pointInLoopPolygon(poly []arcSample, u, v float64, per uvPeriod) bool {
	if len(poly) < 3 {
		return false
	}
	uRange, _ := polygonRangeAlong(poly, bandAxis{})
	vRange, _ := polygonRangeAlong(poly, bandAxis{alongV: true})
	for _, du := range branchOffsets(u, uRange, per.u) {
		for _, dv := range branchOffsets(v, vRange, per.v) {
			if polygonCrossings(poly, u+du, v+dv)%2 == 1 {
				return true
			}
		}
	}
	return false
}

// branchOffsets lists the whole-period offsets that bring x into the polygon's own extent — the only
// translates that can possibly be inside it. A non-periodic parameter (period 0) has the single
// offset 0, which makes the test the ordinary planar one.
func branchOffsets(x float64, r paramRange, period float64) []float64 {
	if period <= 0 {
		return []float64{0}
	}
	lo := stdmath.Ceil((r.lo - x) / period)
	hi := stdmath.Floor((r.hi - x) / period)
	out := make([]float64, 0, 3)
	for k := lo; k <= hi && len(out) < branchOffsetCap; k++ {
		out = append(out, k*period)
	}
	return out
}

// branchOffsetCap bounds the translate search. A face's loop spans one fundamental domain, so two or
// three branches always suffice; the cap keeps a malformed loop with a huge parametric extent from
// turning an O(1) predicate into an unbounded scan.
const branchOffsetCap = 4

// uvPeriod is a face's period in each surface parameter, 0 where that parameter does not wrap.
type uvPeriod struct{ u, v float64 }

// loopsUVPeriod reads both periods off the face's loops.
func loopsUVPeriod(loops []faceLoop) uvPeriod {
	return uvPeriod{u: loopsPeriod(loops, bandAxis{}), v: loopsPeriod(loops, bandAxis{alongV: true})}
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
