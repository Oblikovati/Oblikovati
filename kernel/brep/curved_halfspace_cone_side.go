// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone-side split (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1372). The cone analogue of
// cylinderSideSplit: a FULL periodic frustum side cut by a plane PARALLEL to the axis. Like the
// cylinder, the seam tangles the general looped split, so this gives a dedicated closed-form split —
// but the imprint is one hyperbola branch (the conic a plane ∥ axis cuts from a cone), not two lines,
// and the kept band's two cross-section arcs have different radii. This slice handles the ARC-BAND
// arrangement where the plane cuts EVERY cross-section in the band (the vertex sits at or below the
// bottom rim, |D| < bottom radius) — a flat milled the full length of a frustum's side. The vertex-
// inside-band arrangement (the flat fades out before the bottom rim) and a full cone reaching its
// apex defer to ErrUnsupportedHalfSpace, so the CSG fallback still covers them with no regression.

// coneSideSplit splits a full periodic frustum side f by a plane parallel to the axis, given the
// hyperbola branch the plane cuts. It returns the kept arc-band sub-face and the two hyperbola arms
// (reversed, for the lid). Defers the vertex-inside and through-apex arrangements.
func coneSideSplit(f curvedFace, curves []geom.Curve3, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	cone, ok := f.surface.(geom.Cone)
	if !ok || len(curves) != 1 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	hyper, ok := curves[0].(geom.Hyperbola)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	band, ok := coneSideBand(f, cone)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	d := float64(plane.Origin.VectorTo(cone.Apex).Dot(n)) // signed: every cross-section centre sits at distance d
	absD := stdmath.Abs(d)
	if absD >= band.rTop-cylinderAxisTol { // plane clears every cross-section: whole side, or none
		return coneSideWholeOrEmpty(f, d), nil, nil
	}
	if absD >= band.rBot-cylinderAxisTol { // vertex sits inside the band: deferred to CSG for now
		return nil, nil, ErrUnsupportedHalfSpace
	}
	return coneArcBand(f, cone, hyper, band, n, d)
}

// coneSideBand carries a frustum side's two cross-section circles (centres on the axis, ordered
// low→high in apex distance) and their radii.
type coneSideBand_ struct {
	bottom, top math.Point3
	vMin, vMax  float64
	rBot, rTop  float64
}

// coneSideBand recovers the frustum side's two full-circle rims, ordered by apex distance. ok=false
// unless the face is the expected two-circle periodic side (a full cone, with one rim and an apex,
// is not the arc-band case and defers).
func coneSideBand(f curvedFace, cone geom.Cone) (coneSideBand_, bool) {
	axis := cone.AxisDir.AsVector()
	tanA := stdmath.Tan(cone.HalfAngle)
	var centers []math.Point3
	for _, le := range f.loops[0].edges {
		if c, isCircle := le.curve.(geom.Circle); isCircle && isFullDomain(le.t0, le.t1) {
			centers = append(centers, c.Center)
		}
	}
	if len(centers) != 2 {
		return coneSideBand_{}, false
	}
	if float64(cone.Apex.VectorTo(centers[0]).Dot(axis)) > float64(cone.Apex.VectorTo(centers[1]).Dot(axis)) {
		centers[0], centers[1] = centers[1], centers[0]
	}
	vMin := float64(cone.Apex.VectorTo(centers[0]).Dot(axis))
	vMax := float64(cone.Apex.VectorTo(centers[1]).Dot(axis))
	return coneSideBand_{bottom: centers[0], top: centers[1], vMin: vMin, vMax: vMax, rBot: vMin * tanA, rTop: vMax * tanA}, true
}

// coneSideWholeOrEmpty handles a plane that clears the whole band: the cone's axis sits at signed
// distance d from the plane, so d<0 keeps the entire side (negative side) and d≥0 drops it.
func coneSideWholeOrEmpty(f curvedFace, d float64) []curvedFace {
	if d < 0 {
		return []curvedFace{f}
	}
	return nil
}

// coneArcBand builds the kept arc-band sub-face and the two hyperbola section arms. The kept arc of
// each cross-section is centred on u_c = u_n+π (the rim point farthest from the plane) and spans the
// angle the plane leaves; the two arms are the hyperbola feet climbing from the bottom rim to the top.
func coneArcBand(f curvedFace, cone geom.Cone, hyper geom.Hyperbola, band coneSideBand_, n math.Vector3, d float64) ([]curvedFace, []loopEdge, error) {
	axis := cone.AxisDir.AsVector()
	ref := cone.Ref.AsVector()
	uN := coneAngleOf(cone, n)
	// Kept where cos(u−u_n) < c*(v) = −d/(v·tanα): the arc centred on u_n+π spanning 2π−2φ, φ=arccos(c*).
	phiB := stdmath.Acos(clampUnit(-d / band.rBot))
	phiT := stdmath.Acos(clampUnit(-d / band.rTop))
	bottomArc, err := geom.NewArc3d(band.bottom, axis, ref, band.rBot, uN+phiB, 2*stdmath.Pi-2*phiB)
	if err != nil {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	topArc, err := geom.NewArc3d(band.top, axis, ref, band.rTop, uN-phiT, -(2*stdmath.Pi - 2*phiT))
	if err != nil {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	armA := hyperbolaArm(hyper, bottomArc.PointAt(1), topArc.PointAt(0)) // −φ side, bottom→top
	armB := hyperbolaArm(hyper, topArc.PointAt(1), bottomArc.PointAt(0)) // +φ side, top→bottom
	loop := []loopEdge{{curve: bottomArc, t0: 0, t1: 1}, armA, {curve: topArc, t0: 0, t1: 1}, armB}
	kept := curvedFace{surface: cone, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}}
	section := []loopEdge{reverseEdge(armA), reverseEdge(armB)}
	return []curvedFace{kept}, section, nil
}

// hyperbolaArm builds the loop edge along the hyperbola branch from start to end, its parameters the
// θ values the branch inverse gives at the two rim feet (a hyperbola loop edge's t0/t1 are θ).
func hyperbolaArm(hyper geom.Hyperbola, start, end math.Point3) loopEdge {
	t0, _ := geom.CurveParamAtPoint3(hyper, start)
	t1, _ := geom.CurveParamAtPoint3(hyper, end)
	return loopEdge{curve: hyper, t0: t0, t1: t1}
}

// coneAngleOf returns the angle of a direction (here the cut-plane normal, which lies in the cone's
// cross-section plane since the plane is parallel to the axis) in the cone's Ref/binormal frame.
func coneAngleOf(cone geom.Cone, dir math.Vector3) float64 {
	axis := cone.AxisDir.AsVector()
	r := dir.Sub(axis.Scale(dir.Dot(axis))) // drop the axial component
	binormal := axis.Cross(cone.Ref.AsVector())
	return stdmath.Atan2(float64(r.Dot(binormal)), float64(r.Dot(cone.Ref.AsVector())))
}

// isFullConeSide reports whether f is a full periodic cone side (a geom.Cone bounded by a closed-
// circle rim and the seam) — the case the general looped split tangles on, routed to coneSideSplit.
func isFullConeSide(f curvedFace) bool {
	if _, ok := f.surface.(geom.Cone); !ok || len(f.loops) == 0 {
		return false
	}
	for _, le := range f.loops[0].edges {
		if _, isCircle := le.curve.(geom.Circle); isCircle && isFullDomain(le.t0, le.t1) {
			return true
		}
	}
	return false
}

// allHyperbolas reports whether every imprint curve is a hyperbola branch (the axis-parallel cut of a
// cone), distinguishing it from a perpendicular circle the dedicated cone split cannot consume.
func allHyperbolas(curves []geom.Curve3) bool {
	for _, c := range curves {
		if _, ok := c.(geom.Hyperbola); !ok {
			return false
		}
	}
	return len(curves) > 0
}

// clampUnit clamps x into [-1, 1] so arccos stays defined against rounding at the band rims.
func clampUnit(x float64) float64 {
	return stdmath.Max(-1, stdmath.Min(1, x))
}
