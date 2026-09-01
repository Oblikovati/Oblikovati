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
// and weld with its cap), and their radii. topRimReversed/botRimReversed record how the SOURCE side face
// traverses each rim — opposite to the cap that shares it; a kept band reuses those senses so its rims stay
// opposite those caps. Neither is a fixed convention: a frustum/cylinder side (apex below, axis "up")
// traverses the top rim REVERSED and the bottom rim forward, while an apex-at-top cone — whose v runs from
// the APEX, so the model's top circle is the vMin rim — has them the other way round (#3460). A band with a
// SYNTHETIC end (the recovered vMax of an already-cut side) leaves that end's flag false; nothing emits it.
type coneSideBand_ struct {
	bottom, top         math.Point3
	bottomCirc, topCirc geom.Circle
	vMin, vMax          float64
	rBot, rTop          float64
	topRimReversed      bool
	botRimReversed      bool
}

// coneSideBand recovers the frustum side's two full-circle rims, ordered by apex distance. ok=false
// unless the face is the expected two-circle periodic side (a full cone, with one rim and an apex,
// is not the band case and defers).
func coneSideBand(f curvedFace, cone geom.Cone) (coneSideBand_, bool) {
	axis := cone.AxisDir.AsVector()
	tanA := stdmath.Tan(cone.HalfAngle)
	var circles []geom.Circle
	var revs []bool
	for _, le := range f.loops[0].edges {
		if c, isCircle := le.curve.(geom.Circle); isCircle && isFullDomain(le.t0, le.t1) {
			circles, revs = append(circles, c), append(revs, le.t1 < le.t0)
		}
	}
	if len(circles) != 2 {
		return coneSideBand_{}, false
	}
	if float64(cone.Apex.VectorTo(circles[0].Center).Dot(axis)) > float64(cone.Apex.VectorTo(circles[1].Center).Dot(axis)) {
		circles[0], circles[1] = circles[1], circles[0]
		revs[0], revs[1] = revs[1], revs[0]
	}
	vMin := float64(cone.Apex.VectorTo(circles[0].Center).Dot(axis))
	vMax := float64(cone.Apex.VectorTo(circles[1].Center).Dot(axis))
	return coneSideBand_{
		bottom: circles[0].Center, top: circles[1].Center,
		bottomCirc: circles[0], topCirc: circles[1],
		vMin: vMin, vMax: vMax, rBot: vMin * tanA, rTop: vMax * tanA,
		topRimReversed: revs[1], botRimReversed: revs[0],
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

// fullConeApexSideBand reports whether f is a FULL cone side closing to its APEX — a geom.Cone whose
// single boundary loop is exactly one full rim circle, the apex its v=0 pole (no second rim). It returns
// the cone and a band whose bottom "rim" is the degenerate apex (vMin=0, rBot=0, a zero-radius circle) and
// whose top rim is the circle, so the (u,v) split treats the apex as the v=0 pole. A frustum (two circles)
// or an already-trimmed band (a circle plus section arcs) fails the one-edge test and is handled elsewhere.
func fullConeApexSideBand(f curvedFace, res geom.Resolution) (geom.Cone, coneSideBand_, bool) {
	cone, ok := f.surface.(geom.Cone)
	if !ok || len(f.loops) != 1 {
		return geom.Cone{}, coneSideBand_{}, false
	}
	var circle geom.Circle
	circles, rimReversed, touchesApex := 0, false, false
	for _, le := range f.loops[0].edges {
		if c, isCircle := le.curve.(geom.Circle); isCircle && isFullDomain(le.t0, le.t1) {
			circle, circles, rimReversed = c, circles+1, le.t1 < le.t0
		}
		if samePoint(le.start(), cone.Apex, res) || samePoint(le.end(), cone.Apex, res) {
			touchesApex = true // a seam ruling ends at the apex pole
		}
	}
	if circles != 1 || !touchesApex { // a frustum has two circles; a trimmed band reaches no apex vertex
		return geom.Cone{}, coneSideBand_{}, false
	}
	axis := cone.AxisDir.AsVector()
	vRim := float64(cone.Apex.VectorTo(circle.Center).Dot(axis))
	apexCirc, err := geom.NewCircle(cone.Apex, axis, 0) // the degenerate zero-radius rim at the apex pole
	if err != nil {
		return geom.Cone{}, coneSideBand_{}, false
	}
	band := coneSideBand_{
		bottom: cone.Apex, top: circle.Center,
		bottomCirc: apexCirc, topCirc: circle,
		vMin: 0, vMax: vRim, rBot: 0, rTop: vRim * stdmath.Tan(cone.HalfAngle),
		topRimReversed: rimReversed,
	}
	return cone, band, true
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
