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
	// seamU rotates the (u,v) parameter origin for the arrangement trim only: the artificial azimuth seam
	// (u=0≡2π) is moved to absolute azimuth seamU so it falls clear of the imprint's rim crossings (a
	// section arm grazing the seam otherwise breaks the arrangement, #1405). paramOf reports u relative to
	// seamU; point3/aU/bU add it back. The analytic walk leaves it 0, so its parameterisation is unchanged.
	seamU float64
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

// aU returns a(u) = p + q·cos(u−uN), the v-independent part of the signed distance g(u,v)=a(u)+v·b(u). u is
// relative to the seam origin (seamU), so the absolute azimuth used against uN is u+seamU.
func (c ruledUV) aU(u float64) float64 { return c.p + c.q*stdmath.Cos(u+c.seamU-c.uN) }

// bU returns b(u) = s + t·cos(u−uN), the coefficient of v in the signed distance g(u,v)=a(u)+v·b(u).
func (c ruledUV) bU(u float64) float64 { return c.s + c.t*stdmath.Cos(u+c.seamU-c.uN) }

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

// point3 returns the surface point at (u, v): base + v·â + (radSlope·v+radConst)·r̂(u). u is relative to the
// seam origin (seamU), so the absolute azimuth on the surface frame is u+seamU.
func (c ruledUV) point3(u, v float64) math.Point3 {
	a := u + c.seamU
	radial := c.ref.Scale(math.Scalar(stdmath.Cos(a))).Add(c.binor.Scale(math.Scalar(stdmath.Sin(a))))
	rad := c.radSlope*v + c.radConst
	return c.base.TranslateBy(c.axis.Scale(math.Scalar(v))).TranslateBy(radial.Scale(math.Scalar(rad)))
}

// coneSideUVSplit splits a full periodic frustum side by the general (u,v)-arrangement trimmer (newConeUV +
// trimByImprint), the same path the cylinder side uses — the cone's a(u)+v·b(u) signed distance and its
// conic section (ellipse, hyperbola branch or parabola, windowed to the band by clipParams, the seam moved
// clear of the section by chooseSeamU) flow through it uniformly (Oblikovati#1405).
func coneSideUVSplit(f curvedFace, cone geom.Cone, conic geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	c := newConeUV(cone, band, plane, n)
	return c.trimByImprint(f, cone, []geom.Curve3{conic}, ruledUV.halfSpaceMaterial)
}

// coneApexSideSplit splits a FULL cone side (apex + one rim) by the (u,v) arrangement. The apex is the
// v=0 pole; because a cone has q=0, the apex's signed distance is the constant p, so it is kept exactly
// when p<0. Apex DROPPED → the kept region is a frustum-like band (section + rim) the standard splitSide
// builds (it never references the degenerate apex rim). Apex KEPT → the kept face closes to the apex as a
// single loop (the cut ellipse, or the notched rim), the apex an interior pole (apexCapSide).
func coneApexSideSplit(f curvedFace, cone geom.Cone, conic geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	// Both apex sides go through the general arrangement trim. Apex dropped → a frustum-like band; apex kept
	// → the kept face closes to the apex as a single loop, the apex an interior pole (the degenerate v=0
	// rim loop is dropped inside trimByImprint, dropApexLoop).
	return newConeUV(cone, band, plane, n).trimByImprint(f, cone, []geom.Curve3{conic}, ruledUV.halfSpaceMaterial)
}

// cylinderSideUVSplit splits a full periodic cylinder side by the general (u,v)-arrangement trimmer
// (newCylinderUV + trimByImprint): the axis-parallel flat (a ruling-pair section), the oblique ellipse
// (within-band / clips-rim / tongue), all flow through it uniformly (Oblikovati#1405).
func cylinderSideUVSplit(f curvedFace, cyl geom.Cylinder, curves []geom.Curve3, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	c := newCylinderUV(cyl, band, plane, n)
	return c.trimByImprint(f, cyl, curves, ruledUV.halfSpaceMaterial)
}
