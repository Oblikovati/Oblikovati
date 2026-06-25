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
// interval, with the cone's own orientation inherited, no per-case winding. This file builds the (u,v)
// model and the kept-interval structure; the boundary trace and edge build follow.

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

// keptV returns the kept (g<0) apex-distance interval [lo, hi] at azimuth u, clamped to the band, plus
// whether it is non-empty. g=A+v·C(u) is linear in v: when C>0 the kept side is v<v(u), when C<0 it is
// v>v(u), and when C≈0 the whole column is kept (A<0) or dropped (A≥0).
func (c coneUV) keptV(u float64) (lo, hi float64, ok bool) {
	cu := c.coeffC(u)
	switch {
	case cu > 1e-12:
		hi = c.vMax
		if v := -c.a / cu; v < hi {
			hi = v
		}
		return c.vMin, hi, hi > c.vMin
	case cu < -1e-12:
		lo = c.vMin
		if v := -c.a / cu; v > lo {
			lo = v
		}
		return lo, c.vMax, lo < c.vMax
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
// only at the rim crossings — so each rim arc welds with the cap that shares it. A non-wrapping (tongue)
// arrangement defers to the caller for now.
func coneSideUVSplit(f curvedFace, cone geom.Cone, conic geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	uv := newConeUV(cone, band, plane, n)
	if !uv.wrapsAllU() {
		return nil, nil, ErrUnsupportedHalfSpace // a tongue (kept interval empties somewhere): not yet built here
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
	va := c.sectionV(ua)
	vb := c.sectionV(ub)
	e := conicArm(conic, c.point3(ua, va), c.point3(ub, vb)) // a partial section arc
	return e, []loopEdge{e}, true
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
