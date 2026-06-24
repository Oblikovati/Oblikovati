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
// and the kept band's two cross-section arcs have different radii. coneArcBand handles the ARC-BAND
// arrangement where the plane cuts EVERY cross-section in the band (the vertex sits at or below the
// bottom rim, |D| < bottom radius) — a flat milled the full length of a frustum's side. coneSideVertex-
// Inside handles the VERTEX-INSIDE-BAND arrangement where the flat fades out before the small rim
// (bottom radius ≤ |D| < top radius), the hyperbola turning through its vertex inside the side
// (Oblikovati/Oblikovati#1374). A full cone reaching its apex still defers to ErrUnsupportedHalfSpace,
// so the CSG fallback covers it with no regression.

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
	if absD >= band.rBot-cylinderAxisTol { // vertex sits inside the band: the flat fades before the small rim
		return coneSideVertexInside(f, cone, hyper, band, n, d)
	}
	return coneArcBand(f, cone, hyper, band, n, d)
}

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
// is not the arc-band case and defers).
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

// coneSideVertexInside builds the kept sub-face(s) when the hyperbola vertex (apex distance |d|/tanα)
// lies strictly inside the band — the flat fades out before reaching the small rim
// (band.rBot ≤ |d| < band.rTop, Oblikovati/Oblikovati#1374). The imprint is one continuous hyperbola
// branch through its vertex (an interior cone point), so the kept top boundary is NOTCHED down to that
// vertex. Two arrangements by which side the apex is on:
//   - d > 0 (apex on the dropped side): the small rim is wholly dropped and the kept region is a single
//     tongue around u_n+π narrowing to the vertex — one loop.
//   - d < 0 (apex on the kept side): the kept region is the whole side MINUS that tongue — an annulus
//     whose outer loop is the notched top and whose inner loop is the full small-rim circle.
//
// Both arrangements share the same notched-top loop and the same through-vertex hyperbola section
// (reversed for the lid).
func coneSideVertexInside(f curvedFace, cone geom.Cone, hyper geom.Hyperbola, band coneSideBand_, n math.Vector3, d float64) ([]curvedFace, []loopEdge, error) {
	axis := cone.AxisDir.AsVector()
	ref := cone.Ref.AsVector()
	uN := coneAngleOf(cone, n)
	phiT := stdmath.Acos(clampUnit(-d / band.rTop))
	topArc, err := geom.NewArc3d(band.top, axis, ref, band.rTop, uN-phiT, -(2*stdmath.Pi - 2*phiT))
	if err != nil {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	vertex := hyper.PointAt(0)                                // the branch vertex sits on the cone at apex distance |d|/tanα
	armDown := hyperbolaArm(hyper, topArc.PointAt(1), vertex) // top rim → vertex
	armUp := hyperbolaArm(hyper, vertex, topArc.PointAt(0))   // vertex → top rim
	notched := []loopEdge{{curve: topArc, t0: 0, t1: 1}, armDown, armUp}
	section := []loopEdge{reverseEdge(armDown), reverseEdge(armUp)}
	return coneVertexInsideFaces(f, cone, band, d, notched), section, nil
}

// coneVertexInsideFaces wraps the notched-top loop into the kept face: a lone tongue when the apex is
// dropped (d>0), or an annulus closed by the intact small-rim circle as an inner loop when the apex is
// kept (d<0). The small-rim circle is the source side's own edge so it welds with the small cap.
func coneVertexInsideFaces(f curvedFace, cone geom.Cone, band coneSideBand_, d float64, notched []loopEdge) []curvedFace {
	loops := []curvedLoop{{edges: notched}}
	if d < 0 { // apex kept: close the annulus with the intact small rim
		loops = append(loops, curvedLoop{edges: []loopEdge{{curve: band.bottomCirc, t0: 0, t1: 1}}})
	}
	return []curvedFace{{surface: cone, reversed: f.reversed, lineage: f.lineage, loops: loops}}
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

// coneSideBandSplit routes a genuine full cone side by the section type the cut plane produces: a
// closed ellipse (oblique tilt steeper than the generators) to coneSideEllipseSplit, a hyperbola
// branch (axis-parallel or shallow tilt) to coneSideSplit. A perpendicular circle is handled by the
// fast cone path before the arrangement, so anything else here defers.
func coneSideBandSplit(f curvedFace, curves []geom.Curve3, cone geom.Cone, band coneSideBand_, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	if allEllipses(curves) {
		return coneSideEllipseSplit(f, curves, cone, plane, n)
	}
	if !allHyperbolas(curves) {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	if isAxisParallel(n, cone) {
		return coneSideSplit(f, curves, plane, n) // the symmetric constant-chord hyperbola (#1372/#1374)
	}
	if hyper, ok := curves[0].(geom.Hyperbola); ok && len(curves) == 1 {
		return coneSideObliqueHyperbolaSplit(f, cone, hyper, band, plane, n) // an oblique (tilted) hyperbola
	}
	return nil, nil, ErrUnsupportedHalfSpace
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
