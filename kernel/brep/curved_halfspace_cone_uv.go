// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone-side split in PARAMETER SPACE (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). The OCCT-style
// arrangement approach: instead of hand-building the kept loop and deriving its winding per topological
// case (the arc-band / annulus / tongue / clip families), split the full periodic cone side in its own
// (u, v) = (azimuth, apex-distance) space, where the section a plane carves is a SINGLE-VALUED function
//
//	v(u) = −A / C(u),   C(u) = n·â + tanα·|n_r|·cos(u − u_n),   A = n·(apex − O)
//
// because a cone point is P(u,v) = apex + v·â + v·tanα·r̂(u), so the signed distance is g(u,v)=A+v·C(u),
// linear in v. The kept region {g<0} is then just a v-INTERVAL [lo(u), hi(u)] per u, bounded below/above
// by the rims (v=vMin / vMax) and the section curve — tongue, annulus and rim-clip all fall out of that
// interval, with the cone's own orientation inherited, no per-case winding.
//
// This is the SOLE cone-side splitter: it replaced the bespoke per-section family (the axis-parallel
// arc-band / vertex-inside #1372/#1374, the oblique ellipse, the root-finding oblique hyperbola/parabola),
// each of which hand-derived its own loop and winding. The (u,v) model subsumes them all — every conic
// section (ellipse / hyperbola branch / parabola) and every arrangement is one of two boundary walks:
// wrapping (the kept interval is non-empty at every azimuth → a two-loop band) or non-wrapping (the
// interval empties over part of the seam → a single tongue span). coneSideUVSplit picks between them.

// coneUV is the cone side expressed in (u, v): the apex frame plus the cut plane reduced to A and the
// cosine coefficient C(u) of the signed distance g(u,v) = A + v·C(u).
type coneUV struct {
	apex             math.Point3
	axis, ref, binor math.Vector3
	tanA             float64
	vMin, vMax       float64
	rBot, rTop       float64
	a                float64 // A = n·(apex − O)
	nAxis            float64 // n·â
	nRad             float64 // |radial component of n|
	uN               float64 // azimuth of n's radial component
}

// newConeUV builds the (u, v) model of a frustum side cut by a plane (n the unit plane normal).
func newConeUV(cone geom.Cone, band coneSideBand_, plane geom.Plane, n math.Vector3) coneUV {
	axis := cone.AxisDir.AsVector()
	ref := cone.Ref.AsVector()
	binor := axis.Cross(ref)
	nAxis := float64(n.Dot(axis))
	nr := n.Sub(axis.Scale(math.Scalar(nAxis))) // radial part of n
	nRad := float64(nr.Length())
	uN := stdmath.Atan2(float64(nr.Dot(binor)), float64(nr.Dot(ref)))
	return coneUV{
		apex: cone.Apex, axis: axis, ref: ref, binor: binor, tanA: stdmath.Tan(cone.HalfAngle),
		vMin: band.vMin, vMax: band.vMax, rBot: band.rBot, rTop: band.rTop,
		a:     float64(plane.Origin.VectorTo(cone.Apex).Dot(n)),
		nAxis: nAxis, nRad: nRad, uN: uN,
	}
}

// coeffC returns C(u) = n·â + tanα·|n_r|·cos(u − u_n); the signed distance is g(u,v) = A + v·C(u).
func (c coneUV) coeffC(u float64) float64 {
	return c.nAxis + c.tanA*c.nRad*stdmath.Cos(u-c.uN)
}

// sectionV returns the apex distance v where the cut plane meets the cone at azimuth u — the section
// curve v(u) = −A/C(u). It returns 0 where C(u)≈0 (the plane is parallel to the generator at u, the
// section's asymptote); the wrapping-band caller never samples there, so the degenerate value is unused.
func (c coneUV) sectionV(u float64) float64 {
	cu := c.coeffC(u)
	if stdmath.Abs(cu) < 1e-12 {
		return 0
	}
	return -c.a / cu
}

// vPinchTol is the apex-distance margin below which a kept interval counts as PINCHED (empty). A tongue
// pinches where the section meets a clamp rim (lo≈hi); when that azimuth lands exactly on a sample (a
// symmetric cut puts the pinch on u=0/π/2π) the section value equals the rim to within rounding, so a
// strict lo<hi flickers and breaks the span pairing. The margin makes the pinch read as empty either way.
// It sits well ABOVE the ~1e-14 rounding flicker yet two orders below the 1e-7 weld tolerance, so the span
// endpoint the bisection lands on (where the section sits vPinchTol inside the rim) still welds to the rim.
const vPinchTol = 1e-9

// keptV returns the kept (g<0) apex-distance interval [lo, hi] at azimuth u, clamped to the band, plus
// whether it is non-empty (thicker than vPinchTol). g=A+v·C(u) is linear in v: when C>0 the kept side is
// v<v(u), when C<0 it is v>v(u), and when C≈0 the whole column is kept (A<0) or dropped (A≥0).
func (c coneUV) keptV(u float64) (lo, hi float64, ok bool) {
	cu := c.coeffC(u)
	switch {
	case cu > 1e-12:
		hi = c.vMax
		if v := -c.a / cu; v < hi {
			hi = v
		}
		return c.vMin, hi, hi > c.vMin+vPinchTol
	case cu < -1e-12:
		lo = c.vMin
		if v := -c.a / cu; v > lo {
			lo = v
		}
		return lo, c.vMax, lo < c.vMax-vPinchTol
	default:
		return c.vMin, c.vMax, c.a < 0 // the plane is parallel to the generator: whole column kept or dropped
	}
}

// point3 returns the cone point at (u, v): apex + v·â + v·tanα·r̂(u).
func (c coneUV) point3(u, v float64) math.Point3 {
	radial := c.ref.Scale(math.Scalar(stdmath.Cos(u))).Add(c.binor.Scale(math.Scalar(stdmath.Sin(u))))
	return c.apex.TranslateBy(c.axis.Scale(math.Scalar(v))).TranslateBy(radial.Scale(math.Scalar(v * c.tanA)))
}

// coneSideUVSplit splits a full periodic frustum side by building the kept region {g<0} in (u,v). The
// WRAPPING case (kept v-interval non-empty for every azimuth) is a band represented as a face with two
// boundary loops (no seam, like the vertex-inside annulus #1374): the upper loop is the kept hi(u) curve
// and the lower loop the kept lo(u) curve, each a closed chain of WHOLE rim arcs and section arcs split
// only at the rim crossings — so each rim arc welds with the cap that shares it. A NON-WRAPPING (tongue)
// arrangement — the kept interval empties at some azimuths because the section straddles a rim on its
// kept side — is built by coneSideUVTongue as one loop over the single surviving azimuth span.
func coneSideUVSplit(f curvedFace, cone geom.Cone, conic geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	uv := newConeUV(cone, band, plane, n)
	if !uv.wrapsAllU() {
		return uv.coneSideUVTongue(f, cone, conic) // a non-wrapping arrangement: one kept azimuth span
	}
	hiEdges, hiSec, ok1 := uv.boundaryLoop(conic, band, true)
	loEdges, loSec, ok2 := uv.boundaryLoop(conic, band, false)
	if !ok1 || !ok2 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	// A frustum side's two rims run oppositely (lower CCW, upper CW) so the side stays consistent with both
	// caps. The lo boundary is built CCW (forward); the UPPER boundary is reversed to CW whenever it carries
	// a rim shared with a kept cap, so that rim welds to the cap with opposite sense. The lid then uses each
	// section sub-arc OPPOSITE to the band's (final) use of it, so the shared section edge is consistent too.
	hiLoop, lidHiSec := hiEdges, reverseEdgeChain(hiSec)
	if loopHasRim(hiEdges) {
		hiLoop, lidHiSec = reverseEdgeChain(hiEdges), hiSec
	}
	kept := curvedFace{surface: cone, reversed: f.reversed, lineage: f.lineage,
		loops: []curvedLoop{{edges: hiLoop}, {edges: loEdges}}}
	lidSection := append(append([]loopEdge{}, lidHiSec...), reverseEdgeChain(loSec)...)
	return []curvedFace{kept}, lidSection, nil
}

// coneSideUVTongue builds the kept face of a NON-WRAPPING arrangement: the section straddles the rim on
// its kept side (the section ellipse dips below vMin, or rises above vMax), so the kept v-interval is
// non-empty only over a single azimuth span [u1, u2] and pinches to a point at each end (where the
// section meets the clamp rim). The kept region is one loop — the LOWER bound forward over [u1, u2] then
// the UPPER bound reversed — closing at the two pinch vertices; the section sub-arcs cap the planar lid
// together with the cap chords the same plane carves on the frustum's end caps (Oblikovati#1375).
func (c coneUV) coneSideUVTongue(f curvedFace, cone geom.Cone, conic geom.Curve3) ([]curvedFace, []loopEdge, error) {
	u1, u2, ok := c.keptUSpan()
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace // not a single span (only an ellipse-section tongue is built)
	}
	loEdges, loSec, ok1 := c.boundarySubChain(conic, u1, u2, false)
	hiEdges, hiSec, ok2 := c.boundarySubChain(conic, u1, u2, true)
	if !ok1 || !ok2 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	loop := append(append([]loopEdge{}, loEdges...), reverseEdgeChain(hiEdges)...)
	kept := curvedFace{surface: cone, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}}
	// The lid uses each section sub-arc OPPOSITE to the band's final use: the lo section runs forward in the
	// band so the lid reverses it; the hi section runs reversed in the band so the lid uses it forward.
	section := append(reverseEdgeChain(loSec), hiSec...)
	return []curvedFace{kept}, section, nil
}

// keptUSpan returns the single azimuth interval [u1, u2] (u2 may exceed 2π when the span wraps the seam)
// where the kept v-interval is non-empty — the tongue's u-extent, pinching at u1 and u2 where the section
// reaches the clamp rim. ok=false unless the kept region is EXACTLY one such span (an ellipse section
// straddling a rim makes exactly one; anything else is left to the caller's fallback).
func (c coneUV) keptUSpan() (u1, u2 float64, ok bool) {
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
func (c coneUV) keptNonEmpty(u float64) bool { _, _, ok := c.keptV(u); return ok }

// bisectKeptEdge refines the azimuth where the kept interval pinches to empty — the tongue endpoint where
// the section curve meets the clamp rim. rising=true brackets an empty→non-empty edge, else non-empty→empty.
func (c coneUV) bisectKeptEdge(lo, hi float64, rising bool) float64 {
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
func (c coneUV) boundarySubChain(conic geom.Curve3, ua, ub float64, upper bool) (edges, section []loopEdge, ok bool) {
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
func (c coneUV) interiorRimCrossings(ua, ub float64, upper bool) []float64 {
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
func (c coneUV) loAt(u float64) float64 { lo, _, _ := c.keptV(u); return lo }
func (c coneUV) hiAt(u float64) float64 { _, hi, _ := c.keptV(u); return hi }

// wrapsAllU reports whether the kept v-interval is non-empty at every azimuth (the band wraps the seam).
func (c coneUV) wrapsAllU() bool {
	for i := 0; i < 720; i++ {
		if _, _, ok := c.keptV(2 * stdmath.Pi * float64(i) / 720); !ok {
			return false
		}
	}
	return true
}

// boundAt gives the boundary value (lo or hi) at u.
func (c coneUV) boundAt(u float64, upper bool) float64 {
	if upper {
		return c.hiAt(u)
	}
	return c.loAt(u)
}

// onRim reports whether the boundary at u sits on a rim (constant v) rather than on the section curve.
func (c coneUV) onRim(u float64, upper bool) bool {
	v := c.boundAt(u, upper)
	if upper {
		return v >= c.vMax-1e-7
	}
	return v <= c.vMin+1e-7
}

// boundaryLoop builds the upper (upper=true) or lower boundary of the kept band as a closed loop of whole
// rim arcs and section arcs, and the section (conic) sub-edges alone for the lid. A boundary with no rim
// crossing is either the full source rim circle (all rim) or the full re-anchored ellipse (all section).
func (c coneUV) boundaryLoop(conic geom.Curve3, band coneSideBand_, upper bool) (edges, section []loopEdge, ok bool) {
	breaks := c.rimCrossings(upper)
	if len(breaks) == 0 {
		if c.onRim(0, upper) {
			// A full source rim circle reused whole (built forward here), so it welds to the cap that shares
			// it; coneSideUVSplit reverses the whole upper loop to give that rim the cap-opposite sense.
			circle := band.bottomCirc
			if upper {
				circle = band.topCirc
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
func (c coneUV) rimCrossings(upper bool) []float64 {
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
func (c coneUV) bisectRimBreak(lo, hi float64, upper bool) float64 {
	clamp := c.vMin
	if upper {
		clamp = c.vMax
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
func (c coneUV) boundaryEdge(conic geom.Curve3, ua, ub float64, upper bool) (loopEdge, []loopEdge, bool) {
	um := (ua + ub) / 2
	if c.onRim(um, upper) {
		center, radius := c.apex.TranslateBy(c.axis.Scale(math.Scalar(c.vMin))), c.rBot
		if upper {
			center, radius = c.apex.TranslateBy(c.axis.Scale(math.Scalar(c.vMax))), c.rTop
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
// trace the COMPLEMENTARY major arc instead of the tongue side); the midpoint azimuth's parameter
// disambiguates by unwrapping start→mid→end onto one monotone run. Other conics (hyperbola/parabola arms)
// are open and need no unwrapping, so conicArm's endpoint parameters suffice.
func (c coneUV) sectionArm(conic geom.Curve3, ua, ub float64) loopEdge {
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
func (c coneUV) ellipseParamAt(el geom.EllipseFull, u float64) float64 {
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
func (c coneUV) fullSectionEdge(conic geom.Curve3) (loopEdge, []loopEdge, bool) {
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
