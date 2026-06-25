// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
)

// Ruled-side boundary trace (M2 Phase-1, Oblikovati/Oblikovati#1375). The kept-region boundary walk for
// the (u,v) ruled-side split (curved_halfspace_ruled_uv.go): finding the kept azimuth span, tracing each
// of the lo/hi bounds as a chain of whole rim arcs and section sub-arcs split at the exact rim crossings,
// and building those edges (rim arcs, seam-wrap-safe ellipse arcs, open conic arms). Shared by both the
// cone and cylinder sides — it only reads the surface through keptV / sectionV / onRim / point3.

// keptUSpan returns the single azimuth interval [u1, u2] (u2 may exceed 2π when the span wraps the seam)
// where the kept v-interval is non-empty — the span's u-extent. ok=false unless the kept region is EXACTLY
// one such span (the case the non-wrapping arrangements produce; anything else is left to the fallback).
func (c ruledUV) keptUSpan() (u1, u2 float64, ok bool) {
	const twoPi, N = 2 * stdmath.Pi, 1440
	var starts, ends []float64
	prev := c.keptNonEmpty(0)
	for i := 1; i <= N; i++ {
		u := twoPi * float64(i) / N
		cur := c.keptNonEmpty(u)
		switch {
		case cur && !prev:
			starts = append(starts, c.bisectKeptEdge(twoPi*float64(i-1)/N, u, true))
		case !cur && prev:
			ends = append(ends, c.bisectKeptEdge(twoPi*float64(i-1)/N, u, false))
		}
		prev = cur
	}
	if len(starts) != 1 || len(ends) != 1 {
		return 0, 0, false
	}
	u1, u2 = starts[0], ends[0]
	if u2 < u1 {
		u2 += twoPi // the kept span straddles the seam (it was already open at u=0)
	}
	return u1, u2, true
}

// keptNonEmpty reports whether the kept v-interval is non-empty at azimuth u.
func (c ruledUV) keptNonEmpty(u float64) bool { _, _, ok := c.keptV(u); return ok }

// bisectKeptEdge refines the azimuth where the kept interval pinches to empty — the span endpoint where the
// section meets the clamp rim (or the cut line of an axis-parallel flat). rising=true brackets an
// empty→non-empty edge, else non-empty→empty.
func (c ruledUV) bisectKeptEdge(lo, hi float64, rising bool) float64 {
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		if c.keptNonEmpty(mid) == rising {
			hi = mid
		} else {
			lo = mid
		}
	}
	return (lo + hi) / 2
}

// boundarySubChain builds the boundary chain (rim and section edges) of the upper or lower bound over the
// non-wrapping azimuth span [ua, ub], split at the interior rim crossings, with the section sub-arcs
// returned separately (u-increasing) for the lid.
func (c ruledUV) boundarySubChain(conic geom.Curve3, ua, ub float64, upper bool) (edges, section []loopEdge, ok bool) {
	breaks := append([]float64{ua}, c.interiorRimCrossings(ua, ub, upper)...)
	breaks = append(breaks, ub)
	for i := 0; i+1 < len(breaks); i++ {
		e, sec, good := c.boundaryEdge(conic, breaks[i], breaks[i+1], upper)
		if !good {
			return nil, nil, false
		}
		edges = append(edges, e)
		section = append(section, sec...)
	}
	return edges, section, true
}

// interiorRimCrossings returns the azimuths strictly inside (ua, ub) where the bound switches between a
// rim and the section curve, bisected to the exact crossing (where the section reaches the clamp rim).
func (c ruledUV) interiorRimCrossings(ua, ub float64, upper bool) []float64 {
	const N = 1440
	var out []float64
	prev := c.onRim(ua, upper)
	for i := 1; i < N; i++ {
		u := ua + (ub-ua)*float64(i)/N
		if cur := c.onRim(u, upper); cur != prev {
			out = append(out, c.bisectRimBreak(ua+(ub-ua)*float64(i-1)/N, u, upper))
			prev = cur
		}
	}
	return out
}

// loAt and hiAt give the kept interval's lower / upper bound at azimuth u.
func (c ruledUV) loAt(u float64) float64 { lo, _, _ := c.keptV(u); return lo }
func (c ruledUV) hiAt(u float64) float64 { _, hi, _ := c.keptV(u); return hi }

// wrapsAllU reports whether the kept v-interval is non-empty at every azimuth (the band wraps the seam).
func (c ruledUV) wrapsAllU() bool {
	for i := 0; i < 720; i++ {
		if _, _, ok := c.keptV(2 * stdmath.Pi * float64(i) / 720); !ok {
			return false
		}
	}
	return true
}

// boundAt gives the boundary value (lo or hi) at u.
func (c ruledUV) boundAt(u float64, upper bool) float64 {
	if upper {
		return c.hiAt(u)
	}
	return c.loAt(u)
}

// onRim reports whether the boundary at u sits on a rim (constant v) rather than on the section curve.
func (c ruledUV) onRim(u float64, upper bool) bool {
	v := c.boundAt(u, upper)
	if upper {
		return v >= c.band.vMax-1e-7
	}
	return v <= c.band.vMin+1e-7
}

// boundaryLoop builds the upper (upper=true) or lower boundary of the kept band as a closed loop of whole
// rim arcs and section arcs, and the section (conic) sub-edges alone for the lid. A boundary with no rim
// crossing is either the full source rim circle (all rim) or the full re-anchored ellipse (all section).
func (c ruledUV) boundaryLoop(conic geom.Curve3, upper bool) (edges, section []loopEdge, ok bool) {
	breaks := c.rimCrossings(upper)
	if len(breaks) == 0 {
		if c.onRim(0, upper) {
			// A full source rim circle reused whole (built forward here), so it welds to the cap that shares
			// it; splitSide reverses the whole upper loop to give that rim the cap-opposite sense.
			circle := c.band.bottomCirc
			if upper {
				circle = c.band.topCirc
			}
			return []loopEdge{{curve: circle, t0: 0, t1: 1}}, nil, true
		}
		e, sec, good := c.fullSectionEdge(conic) // the within-band ellipse: a closed re-anchored edge
		return []loopEdge{e}, sec, good
	}
	for i := range breaks { // one edge per gap between consecutive crossings, wrapping the last to the first
		ua, ub := breaks[i], breaks[(i+1)%len(breaks)]
		if i == len(breaks)-1 {
			ub += 2 * stdmath.Pi
		}
		e, sec, good := c.boundaryEdge(conic, ua, ub, upper)
		if !good {
			return nil, nil, false
		}
		edges = append(edges, e)
		section = append(section, sec...)
	}
	return edges, section, true
}

// rimCrossings returns the azimuths in [0, 2π) where the boundary switches between a rim and the section
// curve (the section reaches the clamp rim), sorted. Empty when the boundary is wholly rim or wholly
// section.
func (c ruledUV) rimCrossings(upper bool) []float64 {
	const twoPi, N = 2 * stdmath.Pi, 1440
	var out []float64
	prev := c.onRim(0, upper)
	for i := 1; i <= N; i++ {
		u := twoPi * float64(i) / N
		if cur := c.onRim(u, upper); cur != prev {
			out = append(out, c.bisectRimBreak(twoPi*float64(i-1)/N, u, upper))
			prev = cur
		}
	}
	return out
}

// bisectRimBreak refines the azimuth where the section curve EXACTLY reaches the clamp rim (vMax for the
// upper boundary, vMin for the lower) — the crossing shared with the cap, so the section arc welds to the
// cap chord there.
func (c ruledUV) bisectRimBreak(lo, hi float64, upper bool) float64 {
	clamp := c.band.vMin
	if upper {
		clamp = c.band.vMax
	}
	f := func(u float64) float64 { return c.sectionV(u) - clamp }
	flo := f(lo)
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		if (flo < 0) == (f(mid) < 0) {
			lo, flo = mid, f(mid)
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// boundaryEdge builds one boundary edge over [ua, ub]: a rim arc when the boundary is on a rim there, else
// a sub-arc of the section conic (also returned, u-increasing, as the lid's cut edge).
func (c ruledUV) boundaryEdge(conic geom.Curve3, ua, ub float64, upper bool) (loopEdge, []loopEdge, bool) {
	um := (ua + ub) / 2
	if c.onRim(um, upper) {
		center, radius := c.band.bottom, c.band.rBot
		if upper {
			center, radius = c.band.top, c.band.rTop
		}
		arc, err := geom.NewArc3d(center, c.axis, c.ref, radius, ua, ub-ua)
		if err != nil {
			return loopEdge{}, nil, false
		}
		return loopEdge{curve: arc, t0: 0, t1: 1}, nil, true
	}
	e := c.sectionArm(conic, ua, ub) // a partial section arc, seam-wrap-safe for a full ellipse
	return e, []loopEdge{e}, true
}

// sectionArm builds the section conic sub-arc spanning azimuth [ua, ub]. For a full ellipse the endpoint
// parameters alone are ambiguous (the param seam may fall inside the span, so a plain start→end sweep can
// trace the COMPLEMENTARY major arc instead of the kept side); the midpoint azimuth's parameter
// disambiguates by unwrapping start→mid→end onto one monotone run. Other conics (hyperbola/parabola arms)
// are open and need no unwrapping, so conicArm's endpoint parameters suffice.
func (c ruledUV) sectionArm(conic geom.Curve3, ua, ub float64) loopEdge {
	el, ok := conic.(geom.EllipseFull)
	if !ok {
		return conicArm(conic, c.point3(ua, c.sectionV(ua)), c.point3(ub, c.sectionV(ub)))
	}
	t0 := c.ellipseParamAt(el, ua)
	tm := unwrapParamNear(t0, c.ellipseParamAt(el, (ua+ub)/2))
	t1 := unwrapParamNear(tm, c.ellipseParamAt(el, ub))
	return loopEdge{curve: el, t0: t0, t1: t1}
}

// ellipseParamAt returns the full ellipse's parameter (in [0,1)) at the section point of azimuth u.
func (c ruledUV) ellipseParamAt(el geom.EllipseFull, u float64) float64 {
	t, _ := geom.CurveParamAtPoint3(el, c.point3(u, c.sectionV(u)))
	return t
}

// unwrapParamNear shifts x by whole turns so it lands within ±0.5 of ref — turning a wrapped [0,1)
// parameter sequence into a monotone run, so the swept ellipse arc passes through the interior point.
func unwrapParamNear(ref, x float64) float64 {
	for x-ref > 0.5 {
		x--
	}
	for x-ref < -0.5 {
		x++
	}
	return x
}

// fullSectionEdge builds the closed re-anchored ellipse edge when a boundary is the whole within-band
// section (the section never reaches a rim). Only an ellipse encircles the axis, so anything else defers.
func (c ruledUV) fullSectionEdge(conic geom.Curve3) (loopEdge, []loopEdge, bool) {
	el, ok := conic.(geom.EllipseFull)
	if !ok {
		return loopEdge{}, nil, false
	}
	v0 := c.sectionV(0)
	tStart, _ := geom.CurveParamAtPoint3(el, c.point3(0, v0))
	arc := geom.EllipticalArc{Center: el.Center, Normal: el.Normal, MajorAxis: el.MajorAxis,
		MajorRadius: el.MajorRadius, MinorRadius: el.MinorRadius,
		StartAngle: 2 * stdmath.Pi * tStart, SweepAngle: 2 * stdmath.Pi}
	e := loopEdge{curve: arc, t0: 0, t1: 1}
	return e, []loopEdge{e}, true
}

// loopHasRim reports whether a boundary chain carries a rim edge (a full circle or a rim arc, shared with
// a cap) rather than only section conic arcs — the upper boundary is then reversed to the cap-opposite
// sense. A rim edge's curve is a geom.Circle (full rim) or geom.Arc3d (a partial rim between crossings).
func loopHasRim(edges []loopEdge) bool {
	for _, e := range edges {
		switch e.curve.(type) {
		case geom.Circle, geom.Arc3d:
			return true
		}
	}
	return false
}

// reverseEdgeChain reverses a chain of loop edges (order reversed, each reversed).
func reverseEdgeChain(chain []loopEdge) []loopEdge {
	out := make([]loopEdge, len(chain))
	for i, e := range chain {
		out[len(chain)-1-i] = reverseEdge(e)
	}
	return out
}
