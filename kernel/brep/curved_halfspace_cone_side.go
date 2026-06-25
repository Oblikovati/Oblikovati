// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone-side split (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). A FULL periodic frustum side cut by
// a plane. Like the cylinder, the seam tangles the general looped split, so a dedicated split is needed —
// but unlike the cylinder the section is one of the whole conic family (ellipse / hyperbola branch /
// parabola) and the kept band's cross-section radius varies with apex distance. Every one of those
// arrangements — arc-band, vertex-inside, oblique, within-band, clips-rim, tongue — is now built UNIFORMLY
// by the (u,v) arrangement split (coneSideUVSplit, curved_halfspace_cone_uv.go): on a cone the signed
// distance g(u,v)=A+v·C(u) is linear in v, so the section is single-valued and the kept region is a
// per-azimuth v-interval whose boundary the split traces with the cone's own orientation inherited. This
// file keeps the shared band/edge helpers the unified split builds on. A full cone reaching its apex, or a
// plane through the apex (a degenerate two-line section), still defers to ErrUnsupportedHalfSpace so the
// CSG fallback covers it with no regression.

// coneSideBand carries a frustum side's two cross-section circles (centres on the axis, ordered
// low→high in apex distance), the source rim circles themselves (so a kept face can reuse a rim edge
// and weld with its cap), and their radii.
type coneSideBand_ struct {
	bottom, top         math.Point3
	bottomCirc, topCirc geom.Circle
	vMin, vMax          float64
	rBot, rTop          float64
}

// coneSideBand recovers the frustum side's two full-circle rims, ordered by apex distance. ok=false
// unless the face is the expected two-circle periodic side (a full cone, with one rim and an apex,
// is not the band case and defers).
func coneSideBand(f curvedFace, cone geom.Cone) (coneSideBand_, bool) {
	axis := cone.AxisDir.AsVector()
	tanA := stdmath.Tan(cone.HalfAngle)
	var circles []geom.Circle
	for _, le := range f.loops[0].edges {
		if c, isCircle := le.curve.(geom.Circle); isCircle && isFullDomain(le.t0, le.t1) {
			circles = append(circles, c)
		}
	}
	if len(circles) != 2 {
		return coneSideBand_{}, false
	}
	if float64(cone.Apex.VectorTo(circles[0].Center).Dot(axis)) > float64(cone.Apex.VectorTo(circles[1].Center).Dot(axis)) {
		circles[0], circles[1] = circles[1], circles[0]
	}
	vMin := float64(cone.Apex.VectorTo(circles[0].Center).Dot(axis))
	vMax := float64(cone.Apex.VectorTo(circles[1].Center).Dot(axis))
	return coneSideBand_{
		bottom: circles[0].Center, top: circles[1].Center,
		bottomCirc: circles[0], topCirc: circles[1],
		vMin: vMin, vMax: vMax, rBot: vMin * tanA, rTop: vMax * tanA,
	}, true
}

// coneSideBandSplit splits a genuine full cone side by the (u,v) arrangement, which builds every conic
// section uniformly — ellipse, hyperbola branch or parabola; within-band, clips-rim, tongue and
// vertex-inside all fall out of the kept v-interval. A section that is not a single conic (a plane through
// the apex carves two lines) defers to CSG.
func coneSideBandSplit(f curvedFace, curves []geom.Curve3, cone geom.Cone, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	if len(curves) != 1 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	return coneSideUVSplit(f, cone, curves[0], band, plane, n)
}

// conicArm builds the loop edge along an open conic section (a hyperbola branch or a parabola) from
// start to end, its parameters the conic's own parameter values at the two rim feet (the loop edge's
// t0/t1 are the curve parameter — θ for a hyperbola, the cross coordinate t for a parabola).
func conicArm(conic geom.Curve3, start, end math.Point3) loopEdge {
	t0, _ := geom.CurveParamAtPoint3(conic, start)
	t1, _ := geom.CurveParamAtPoint3(conic, end)
	return loopEdge{curve: conic, t0: t0, t1: t1}
}

// fullConeSideBand reports whether f is the GENUINE full periodic cone side — a geom.Cone bounded by
// TWO closed-circle rims and the seam, the untrimmed frustum side the general looped split tangles on
// — and returns its band. An already-trimmed band (one circle plus an ellipse/hyperbola rim, after a
// first oblique cut) fails the two-circle test here and falls through to loopedSplit, so a later
// clearing plane composes instead of re-entering the dedicated split (which needs both rim circles).
func fullConeSideBand(f curvedFace) (geom.Cone, coneSideBand_, bool) {
	cone, ok := f.surface.(geom.Cone)
	if !ok || len(f.loops) == 0 {
		return geom.Cone{}, coneSideBand_{}, false
	}
	band, ok := coneSideBand(f, cone)
	if !ok {
		return geom.Cone{}, coneSideBand_{}, false
	}
	return cone, band, true
}
