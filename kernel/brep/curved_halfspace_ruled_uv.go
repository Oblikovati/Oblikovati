// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Ruled-side split in PARAMETER SPACE (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). The OCCT-style
// arrangement approach for a full periodic ruled side (a cone frustum side OR a cylinder side) cut by a
// plane: instead of hand-building the kept loop and deriving its winding per topological case, split the
// side in its own (u, v) = (azimuth, axial-distance) space. A ruled side is straight along v, so the
// signed distance is LINEAR in v:
//
//	g(u, v) = a(u) + v·b(u),   a(u) = p + q·cos(u − u_n),   b(u) = s + t·cos(u − u_n)
//
// because the surface point is P(u,v) = base + v·â + rad(v)·r̂(u) with rad(v) = radSlope·v + radConst,
// so g = n·(base − O) + v·n·â + rad(v)·(n·r̂(u)) and n·r̂(u) = |n_r|·cos(u − u_n). A CONE has rad(v)=v·tanα
// (radConst 0 → q 0, t=tanα·|n_r|); a CYLINDER has rad(v)=R (radSlope 0 → t 0, q=R·|n_r|). The section is
// the single-valued v(u) = −a(u)/b(u) and the kept region {g<0} is a v-INTERVAL [lo(u), hi(u)] per u —
// every arrangement (arc-band, vertex-inside, oblique, within-band, clips-rim, tongue, the cylinder's
// axis-parallel flat) is one of two boundary walks: WRAPPING (the interval is non-empty at every azimuth
// → a two-loop band) or NON-WRAPPING (it empties over part of the seam → a single span), with the
// surface's own orientation inherited, no per-case winding. This is the sole splitter for both ruled
// sides; the bespoke cone and cylinder split families it replaced are gone.

// ruledUV is a periodic ruled side (cone or cylinder) expressed in (u, v): the surface frame and the
// linear coefficients of the signed distance g(u,v) = (p + q·cos(u−uN)) + v·(s + t·cos(u−uN)).
type ruledUV struct {
	base               math.Point3 // surface point at v=0 on the axis (cone apex, cylinder bottom centre)
	axis, ref, binor   math.Vector3
	radSlope, radConst float64 // rad(v) = radSlope·v + radConst (cone: tanα, 0; cylinder: 0, R)
	band               coneSideBand_
	p, q, s, t, uN     float64
}

// newConeUV builds the (u, v) model of a frustum side cut by a plane (n the unit plane normal).
func newConeUV(cone geom.Cone, band coneSideBand_, plane geom.Plane, n math.Vector3) ruledUV {
	tanA := stdmath.Tan(cone.HalfAngle)
	return newRuledUV(cone.Apex, cone.AxisDir.AsVector(), cone.Ref.AsVector(), tanA, 0, band, plane, n)
}

// newCylinderUV builds the (u, v) model of a cylinder side cut by a plane. The cylinder is the degenerate
// cone — constant radius R, so rad(v)=R (radSlope 0); v is the axial distance from the bottom rim centre.
func newCylinderUV(cyl geom.Cylinder, band coneSideBand_, plane geom.Plane, n math.Vector3) ruledUV {
	return newRuledUV(band.bottom, cyl.AxisDir.AsVector(), cyl.Ref.AsVector(), 0, cyl.Radius, band, plane, n)
}

// newRuledUV reduces a ruled side and a cut plane to the (u,v) signed-distance coefficients. base is the
// surface point at v=0 (cone apex / cylinder bottom centre), rad(v)=radSlope·v+radConst the cross-section
// radius, so p=n·(base−O), q=radConst·|n_r|, s=n·â, t=radSlope·|n_r|, uN the azimuth of n's radial part.
func newRuledUV(base math.Point3, axis, ref math.Vector3, radSlope, radConst float64, band coneSideBand_, plane geom.Plane, n math.Vector3) ruledUV {
	binor := axis.Cross(ref)
	nAxis := float64(n.Dot(axis))
	nr := n.Sub(axis.Scale(math.Scalar(nAxis))) // radial part of n
	nRad := float64(nr.Length())
	uN := stdmath.Atan2(float64(nr.Dot(binor)), float64(nr.Dot(ref)))
	return ruledUV{
		base: base, axis: axis, ref: ref, binor: binor,
		radSlope: radSlope, radConst: radConst, band: band,
		p:  float64(plane.Origin.VectorTo(base).Dot(n)),
		q:  radConst * nRad,
		s:  nAxis,
		t:  radSlope * nRad,
		uN: uN,
	}
}

// aU returns a(u) = p + q·cos(u−uN), the v-independent part of the signed distance g(u,v)=a(u)+v·b(u).
func (c ruledUV) aU(u float64) float64 { return c.p + c.q*stdmath.Cos(u-c.uN) }

// bU returns b(u) = s + t·cos(u−uN), the coefficient of v in the signed distance g(u,v)=a(u)+v·b(u).
func (c ruledUV) bU(u float64) float64 { return c.s + c.t*stdmath.Cos(u-c.uN) }

// sectionV returns the axial distance v where the cut plane meets the side at azimuth u — the section
// curve v(u) = −a(u)/b(u). It returns 0 where b(u)≈0 (the plane is parallel to the ruling at u, no finite
// section there); that azimuth's section is a vertical ruling handled by the span-end edge, not sampled here.
func (c ruledUV) sectionV(u float64) float64 {
	b := c.bU(u)
	if stdmath.Abs(b) < 1e-12 {
		return 0
	}
	return -c.aU(u) / b
}

// vPinchTol is the axial-distance margin below which a kept interval counts as PINCHED (empty). A tongue
// pinches where the section meets a clamp rim (lo≈hi); when that azimuth lands exactly on a sample (a
// symmetric cut puts the pinch on u=0/π/2π) the section value equals the rim to within rounding, so a
// strict lo<hi flickers and breaks the span pairing. The margin makes the pinch read as empty either way.
// It sits well ABOVE the ~1e-14 rounding flicker yet two orders below the 1e-7 weld tolerance, so the span
// endpoint the bisection lands on (where the section sits vPinchTol inside the rim) still welds to the rim.
//
// tol:calibrated — this (u,v) arrangement margin is OCC-validated for the delicate tangent-limit cases
// (the parabola cone∩box, the symmetric pinch). It is NOT model-relativised under #1399: at a part scale
// of ~10 a size-scaled weld is an order of magnitude off and misclassifies the rim crossing, breaking the
// validated volumes. The split operates on the cone's own (azimuth, axial) frame, which is already ~O(R).
const vPinchTol = 1e-9

// keptV returns the kept (g<0) axial-distance interval [lo, hi] at azimuth u, clamped to the band, plus
// whether it is non-empty (thicker than vPinchTol). g=a(u)+v·b(u) is linear in v: when b>0 the kept side
// is v<v(u), when b<0 it is v>v(u), and when b≈0 the whole ruling is kept (a<0) or dropped (a≥0).
func (c ruledUV) keptV(u float64) (lo, hi float64, ok bool) {
	a, b := c.aU(u), c.bU(u)
	switch {
	case b > 1e-12:
		hi = c.band.vMax
		if v := -a / b; v < hi {
			hi = v
		}
		return c.band.vMin, hi, hi > c.band.vMin+vPinchTol
	case b < -1e-12:
		lo = c.band.vMin
		if v := -a / b; v > lo {
			lo = v
		}
		return lo, c.band.vMax, lo < c.band.vMax-vPinchTol
	default:
		return c.band.vMin, c.band.vMax, a < 0 // the plane is parallel to the ruling: whole column kept or dropped
	}
}

// point3 returns the surface point at (u, v): base + v·â + (radSlope·v+radConst)·r̂(u).
func (c ruledUV) point3(u, v float64) math.Point3 {
	radial := c.ref.Scale(math.Scalar(stdmath.Cos(u))).Add(c.binor.Scale(math.Scalar(stdmath.Sin(u))))
	rad := c.radSlope*v + c.radConst
	return c.base.TranslateBy(c.axis.Scale(math.Scalar(v))).TranslateBy(radial.Scale(math.Scalar(rad)))
}

// coneSideUVSplit splits a full periodic frustum side by the (u,v) arrangement (newConeUV + splitSide).
func coneSideUVSplit(f curvedFace, cone geom.Cone, conic geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	return newConeUV(cone, band, plane, n).splitSide(f, cone, conic)
}

// coneApexSideSplit splits a FULL cone side (apex + one rim) by the (u,v) arrangement. The apex is the
// v=0 pole; because a cone has q=0, the apex's signed distance is the constant p, so it is kept exactly
// when p<0. Apex DROPPED → the kept region is a frustum-like band (section + rim) the standard splitSide
// builds (it never references the degenerate apex rim). Apex KEPT → the kept face closes to the apex as a
// single loop (the cut ellipse, or the notched rim), the apex an interior pole (apexCapSide).
func coneApexSideSplit(f curvedFace, cone geom.Cone, conic geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	uv := newConeUV(cone, band, plane, n)
	if !uv.apexKept() {
		return uv.splitSide(f, cone, conic) // apex on the dropped side: a frustum-like band, no apex pole
	}
	return uv.apexCapSide(f, cone, conic)
}

// apexKept reports whether the cone apex (the v=0 pole) is on the kept (negative) side. A cone has q=0, so
// a(0)=p is constant in u and the apex's signed distance g(0)=p; the apex is kept exactly when p<0.
func (c ruledUV) apexKept() bool { return c.p < 0 }

// apexCapSide builds the kept face when the apex is KEPT: the cone closes to its apex pole capped by the
// cut, so the face is a SINGLE loop — the hi boundary (the full cut ellipse, or the rim notched by the
// section) — with the apex an interior pole and no lower loop. The hi boundary is oriented like splitSide's
// upper loop (reversed when it carries a rim shared with the base cap); the section caps the lid.
func (c ruledUV) apexCapSide(f curvedFace, surface geom.Surface, conic geom.Curve3) ([]curvedFace, []loopEdge, error) {
	hiEdges, hiSec, ok := c.boundaryLoop(conic, true)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	hiLoop, lidSec := hiEdges, reverseEdgeChain(hiSec)
	if loopHasRim(hiEdges) && c.band.topRimReversed {
		hiLoop, lidSec = reverseEdgeChain(hiEdges), hiSec
	}
	kept := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: hiLoop}}}
	return []curvedFace{kept}, lidSec, nil
}

// cylinderSideUVSplit splits a full periodic cylinder side by the (u,v) arrangement (newCylinderUV +
// splitSide). It handles both the axis-parallel flat (b≡0 → a vertical-edged span) and an oblique ellipse
// cut (within-band / clips-rim / tongue), the latter the case the line-only cylinder split deferred to CSG.
func cylinderSideUVSplit(f curvedFace, cyl geom.Cylinder, curves []geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	c := newCylinderUV(cyl, band, plane, n)
	return c.trimByImprint(f, cyl, curves, c.halfSpaceMaterial())
}

// splitSide builds the kept region {g<0} of a ruled side in (u,v). The WRAPPING case (kept v-interval
// non-empty for every azimuth) is a band represented as a face with two boundary loops (no seam, like the
// vertex-inside annulus #1374): the upper loop is the kept hi(u) curve and the lower loop the kept lo(u)
// curve, each a closed chain of WHOLE rim arcs and section arcs split only at the rim crossings — so each
// rim arc welds with the cap that shares it. A NON-WRAPPING arrangement — the interval empties at some
// azimuths — is built by tongueSide as one span. surface is the kept face's analytic surface (cone/cylinder).
func (c ruledUV) splitSide(f curvedFace, surface geom.Surface, conic geom.Curve3) ([]curvedFace, []loopEdge, error) {
	if !c.wrapsAllU() {
		return c.tongueSide(f, surface, conic) // a non-wrapping arrangement: one kept azimuth span
	}
	hiEdges, hiSec, ok1 := c.boundaryLoop(conic, true)
	loEdges, loSec, ok2 := c.boundaryLoop(conic, false)
	if !ok1 || !ok2 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	// A ruled side's two rims run oppositely so the side stays consistent with both caps. The lo boundary is
	// built CCW (forward); the UPPER boundary is reversed whenever it carries a rim that the source face
	// traversed reversed (band.topRimReversed) — so the rebuilt rim keeps the sense opposite its kept cap.
	// A frustum/cylinder traverses the top rim reversed; an apex-at-top full cone traverses its rim forward,
	// so this is NOT a fixed flip. The lid uses each section sub-arc OPPOSITE to the band's final use of it.
	hiLoop, lidHiSec := hiEdges, reverseEdgeChain(hiSec)
	if loopHasRim(hiEdges) && c.band.topRimReversed {
		hiLoop, lidHiSec = reverseEdgeChain(hiEdges), hiSec
	}
	kept := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage,
		loops: []curvedLoop{{edges: hiLoop}, {edges: loEdges}}}
	lidSection := append(append([]loopEdge{}, lidHiSec...), reverseEdgeChain(loSec)...)
	return []curvedFace{kept}, lidSection, nil
}

// tongueSide builds the kept face of a NON-WRAPPING arrangement: the kept v-interval is non-empty only
// over a single azimuth span [u1, u2]. The kept region is one loop — the LOWER bound forward over [u1, u2],
// the span-end ruling UP (lo→hi at u2), the UPPER bound reversed, the start ruling DOWN (hi→lo at u1).
// At a PINCH end (cone tongue: section meets the clamp rim, lo≈hi) the ruling is degenerate and dropped;
// at a FULL end (cylinder axis-parallel flat: the plane parallels the ruling, lo=vMin hi=vMax) the ruling
// is the vertical cut line that bounds the lid. The section sub-arcs and the cut-line rulings cap the
// planar lid together with the cap chords the same plane carves on the end caps (Oblikovati#1375).
func (c ruledUV) tongueSide(f curvedFace, surface geom.Surface, conic geom.Curve3) ([]curvedFace, []loopEdge, error) {
	u1, u2, ok := c.keptUSpan()
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace // not a single span
	}
	loEdges, loSec, ok1 := c.boundarySubChain(conic, u1, u2, false)
	hiEdges, hiSec, ok2 := c.boundarySubChain(conic, u1, u2, true)
	if !ok1 || !ok2 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	endHi, hasHi := c.spanEndEdge(u2) // the ruling at u2, oriented lo→hi
	endLo, hasLo := c.spanEndEdge(u1) // the ruling at u1, oriented lo→hi
	loop := tongueLoop(loEdges, hiEdges, endHi, hasHi, endLo, hasLo)
	section := tongueSection(loSec, hiSec, endHi, hasHi, endLo, hasLo)
	kept := curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}}
	return []curvedFace{kept}, section, nil
}

// tongueLoop assembles the single kept loop of a non-wrapping span: the lower bound forward, the u2 ruling
// up (lo→hi), the upper bound reversed, the u1 ruling down (hi→lo). A degenerate (pinch) ruling is dropped.
func tongueLoop(loEdges, hiEdges []loopEdge, endHi loopEdge, hasHi bool, endLo loopEdge, hasLo bool) []loopEdge {
	loop := append([]loopEdge{}, loEdges...)
	if hasHi {
		loop = append(loop, endHi)
	}
	loop = append(loop, reverseEdgeChain(hiEdges)...)
	if hasLo {
		loop = append(loop, reverseEdge(endLo))
	}
	return loop
}

// tongueSection assembles the span's lid section edges, each OPPOSITE to the band's final use of it: the
// lo section runs forward in the band so the lid reverses it, the hi section runs reversed so the lid uses
// it forward, and each cut-line ruling likewise (the u2 ruling reversed, the u1 ruling forward).
func tongueSection(loSec, hiSec []loopEdge, endHi loopEdge, hasHi bool, endLo loopEdge, hasLo bool) []loopEdge {
	section := append(reverseEdgeChain(loSec), hiSec...)
	if hasHi {
		section = append(section, reverseEdge(endHi))
	}
	if hasLo {
		section = append(section, endLo)
	}
	return section
}

// spanEndEdge returns the vertical ruling at azimuth u from the kept lo to hi, oriented lo→hi, and whether
// it is non-degenerate. At a pinch end (lo≈hi) it returns false (the loop closes at the pinch vertex with
// no edge); at a full end (the cut line of an axis-parallel flat) it returns the straight cut ruling.
func (c ruledUV) spanEndEdge(u float64) (loopEdge, bool) {
	lo, hi, _ := c.keptV(u)
	if hi-lo < 1e-6 {
		return loopEdge{}, false // pinched: no span-end ruling
	}
	seg := geom.NewLineSegment(c.point3(u, lo), c.point3(u, hi))
	return loopEdge{curve: seg, t0: 0, t1: 1}, true
}
