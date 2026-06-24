// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone-side oblique-hyperbolic split (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). An OBLIQUE box
// face tilted SHALLOWER than the cone's generators (but not parallel to the axis) cuts the frustum side
// in a HYPERBOLA — like the axis-parallel case (#1372) but tilted, so the section is symmetric about the
// {axis, normal} meridian rather than a constant-distance chord. Its two arms still climb from one rim to
// the other, so the kept band is an ARC-BAND bounded by the two arms plus the kept arcs of the rims.
//
// coneArcBand (#1372) builds that band from a CONSTANT chord half-angle φ=arccos(−d/r); here the chord
// distance varies with apex distance, so this finds the rim crossings by ROOT-FINDING (the apex distance
// along a hyperbola arm is monotonic, so each arm crosses each rim once) and reuses coneArcBand's winding
// with the per-rim φ taken from the ACTUAL crossing azimuths. The vertex-below-band arrangement (the arms
// cross BOTH rims) is handled; a vertex inside the band, or arms that miss a rim, defer to CSG.

// coneSideObliqueHyperbolaSplit splits a full periodic frustum side by an oblique plane whose section is
// one hyperbola branch whose vertex sits below the band (both arms cross both rims). It returns the kept
// arc-band sub-face and the two hyperbola arms (reversed, for the lid). A plane that clears the band
// keeps the side whole or drops it; a vertex inside the band or arms missing a rim defer.
func coneSideObliqueHyperbolaSplit(f curvedFace, cone geom.Cone, hyper geom.Hyperbola, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	if side := bandSideOfPlane(cone, band, plane, n); side != 0 {
		return coneSideWholeOrEmpty(f, float64(side)), nil, nil // the plane clears the side: whole or empty
	}
	if apexDistOf(cone, hyper.PointAt(0)) >= band.vMin-cylinderAxisTol {
		return nil, nil, ErrUnsupportedHalfSpace // vertex inside the band (oblique #1374 analogue): deferred
	}
	thetaBot, okB := hyperThetaAtApexDist(hyper, cone, band.vMin)
	thetaTop, okT := hyperThetaAtApexDist(hyper, cone, band.vMax)
	if !okB || !okT {
		return nil, nil, ErrUnsupportedHalfSpace // an arm does not reach a rim: an unhandled arrangement
	}
	loop, section := obliqueHyperbolaBand(cone, hyper, band, thetaBot, thetaTop, n)
	if loop == nil {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	kept := curvedFace{surface: cone, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}}
	return []curvedFace{kept}, section, nil
}

// obliqueHyperbolaBand builds the kept band the SAME way coneArcBand (#1372) does — bottom arc CCW from
// u_n+φ_B over the major arc, top arc CW, the two hyperbola arms between them — but the per-rim half-angle
// φ comes from the ACTUAL rim crossing (root-found) instead of the constant-chord arccos(−d/r), because
// an oblique plane's chord distance varies with apex distance. u_n is the meridian azimuth toward the
// plane; the crossings sit symmetrically at u_n±φ, so the kept (negative-side) band is the major arc away
// from u_n, identical in winding to the axis-parallel band so it composes with the cap chords.
func obliqueHyperbolaBand(cone geom.Cone, hyper geom.Hyperbola, band coneSideBand_, thetaBot, thetaTop float64, n math.Vector3) ([]loopEdge, []loopEdge) {
	axis, ref := cone.AxisDir.AsVector(), cone.Ref.AsVector()
	uN := coneAngleOf(cone, n)
	phiB := crossingHalfAngle(cone, hyper.PointAt(thetaBot), uN)
	phiT := crossingHalfAngle(cone, hyper.PointAt(thetaTop), uN)
	bottomArc, errB := geom.NewArc3d(band.bottom, axis, ref, band.rBot, uN+phiB, 2*stdmath.Pi-2*phiB)
	topArc, errT := geom.NewArc3d(band.top, axis, ref, band.rTop, uN-phiT, -(2*stdmath.Pi - 2*phiT))
	if errB != nil || errT != nil {
		return nil, nil
	}
	armA := hyperbolaArm(hyper, bottomArc.PointAt(1), topArc.PointAt(0)) // −φ side, bottom→top
	armB := hyperbolaArm(hyper, topArc.PointAt(1), bottomArc.PointAt(0)) // +φ side, top→bottom
	loop := []loopEdge{{curve: bottomArc, t0: 0, t1: 1}, armA, {curve: topArc, t0: 0, t1: 1}, armB}
	return loop, []loopEdge{reverseEdge(armA), reverseEdge(armB)}
}

// crossingHalfAngle returns the unsigned azimuthal angle between a rim crossing point and the meridian
// u_n — the φ the kept major arc (u_n+φ … u_n−φ the long way) leaves on that rim.
func crossingHalfAngle(cone geom.Cone, crossing math.Point3, uN float64) float64 {
	u := coneAngleOf(cone, cone.Apex.VectorTo(crossing))
	return stdmath.Abs(wrapToPi(u - uN))
}

// bandSideOfPlane samples the frustum side over a (u, v) grid and reports −1 if every sample is on the
// plane's negative (kept) side (the plane clears the band → keep whole), +1 if every sample is positive
// (→ empty), or 0 if the plane crosses the band (→ split). It guards the split against a plane whose
// section hyperbola lies on the infinite cone OUTSIDE the frustum (a far box face), which must clear the
// side, not defer.
func bandSideOfPlane(cone geom.Cone, band coneSideBand_, plane geom.Plane, n math.Vector3) int {
	axis := cone.AxisDir.AsVector()
	tanA := stdmath.Tan(cone.HalfAngle)
	pos, neg := false, false
	for vi := 0; vi <= 4; vi++ {
		v := band.vMin + (band.vMax-band.vMin)*float64(vi)/4
		center := cone.Apex.TranslateBy(axis.Scale(math.Scalar(v)))
		for ui := 0; ui < 16; ui++ {
			p := coneRimPoint(cone, center, v*tanA, 2*stdmath.Pi*float64(ui)/16)
			if signedDistance(p, plane, n) < 0 {
				neg = true
			} else {
				pos = true
			}
		}
	}
	switch {
	case neg && !pos:
		return -1
	case pos && !neg:
		return 1
	default:
		return 0
	}
}

// coneRimPoint returns the point at azimuth u on a cross-section circle (centre on the axis, given
// radius) in the cone's Ref/binormal frame.
func coneRimPoint(cone geom.Cone, center math.Point3, radius, u float64) math.Point3 {
	ref := cone.Ref.AsVector()
	binormal := cone.AxisDir.AsVector().Cross(ref)
	radial := ref.Scale(math.Scalar(stdmath.Cos(u))).Add(binormal.Scale(math.Scalar(stdmath.Sin(u))))
	return center.TranslateBy(radial.Scale(math.Scalar(radius)))
}

// apexDistOf returns a point's distance from the apex measured along the cone axis.
func apexDistOf(cone geom.Cone, p math.Point3) float64 {
	return float64(cone.Apex.VectorTo(p).Dot(cone.AxisDir.AsVector()))
}

// hyperThetaAtApexDist returns the positive hyperbola parameter θ where the arm reaches apex distance
// target. Apex distance grows monotonically with θ along a +nappe arm, so it brackets then bisects.
// ok=false if no θ up to a large bound reaches the target (the arm never gets that far up the cone).
func hyperThetaAtApexDist(hyper geom.Hyperbola, cone geom.Cone, target float64) (float64, bool) {
	lo, hi := 0.0, 0.5
	for apexDistOf(cone, hyper.PointAt(hi)) < target {
		hi *= 2
		if hi > 1e4 {
			return 0, false
		}
	}
	for i := 0; i < 80; i++ {
		mid := (lo + hi) / 2
		if apexDistOf(cone, hyper.PointAt(mid)) < target {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2, true
}

// isAxisParallel reports whether the cut plane is (essentially) parallel to the cone axis — the
// symmetric constant-chord hyperbola of #1372/#1374, routed to coneSideSplit rather than the oblique
// root-finding split here.
func isAxisParallel(n math.Vector3, cone geom.Cone) bool {
	return stdmath.Abs(float64(n.Dot(cone.AxisDir.AsVector()))) < 1e-6
}
