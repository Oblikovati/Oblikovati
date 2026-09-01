// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The four rails of the tangent-degenerate corner (T-N7.2), ordered head-to-tail into the valence-4
// loop coons4 fills. The cycle is  W0 →[wallArm0 cross-section]→ P0 →[non-wall arm]→ P1 →[wallArm1
// cross-section]→ W1 →[on-wall bridge E2]→ W0, where W0/W1 are the two distinct wall feet, P0/P1 the
// shared plane feet. Each rail's Adjacent is its arm surface (the wall for E2), Cont=G1.

// tangentDegenerateSides assembles the four ordered Sides. wa holds the two wall-sharing arm indices;
// the remaining arm is the non-wall (two-plane) one. Declines if the plane feet do not chain.
func tangentDegenerateSides(w cornerWeld, arms []edgeFillet, centres []math.Point3, wallFace *topo.Face, wall geom.Cylinder, wa [2]int) ([]Side, bool) {
	i0, i1 := wa[0], wa[1]
	mid := nonWallArmIndex(arms, i0, i1)
	if mid < 0 {
		return nil, false
	}
	w0 := cylinderWallPoint(wall, centres[i0])
	w1 := cylinderWallPoint(wall, centres[i1])
	p0, ok0 := nonWallPlaneFoot(arms[i0], wallFace, centres[i0])
	p1, ok1 := nonWallPlaneFoot(arms[i1], wallFace, centres[i1])
	if !ok0 || !ok1 {
		return nil, false
	}
	return orderTangentSides(arms, centres, wall, w, sideEnds{i0, i1, mid, w0, w1, p0, p1})
}

// sideEnds carries the resolved corner vertices threaded into the ordered-side builder.
type sideEnds struct {
	i0, i1, mid int
	w0, w1      math.Point3 // the two distinct wall feet (wallArm0, wallArm1)
	p0, p1      math.Point3 // the plane feet of wallArm0, wallArm1 (shared with the non-wall arm)
}

// orderTangentSides materialises the four rails in cyclic head-to-tail order (W0→P0→P1→W1→W0), each
// arc oriented by construction (crossSectionArc controls its endpoints) and E2 built W1→W0 on the wall.
func orderTangentSides(arms []edgeFillet, centres []math.Point3, wall geom.Cylinder, w cornerWeld, e sideEnds) ([]Side, bool) {
	a0, ok0 := crossSectionArc(centres[e.i0], e.w0, e.p0, w.radius)
	amid, ok1 := crossSectionArc(centres[e.mid], e.p0, e.p1, w.radius)
	a1, ok2 := crossSectionArc(centres[e.i1], e.p1, e.w1, w.radius)
	e2, ok3 := wallBridgeRail(wall, arms[e.i1].armSurface, arms[e.i0].armSurface, e.w1, e.w0)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return nil, false
	}
	return []Side{
		{Curve: a0, Adjacent: arms[e.i0].armSurface, Cont: G1},
		{Curve: amid, Adjacent: arms[e.mid].armSurface, Cont: G1},
		{Curve: a1, Adjacent: arms[e.i1].armSurface, Cont: G1},
		{Curve: e2, Adjacent: wall, Cont: G1},
	}, true
}

// nonWallArmIndex returns the index of the one arm that is neither wall-sharing arm, or −1.
func nonWallArmIndex(arms []edgeFillet, i0, i1 int) int {
	for i := range arms {
		if i != i0 && i != i1 {
			return i
		}
	}
	return -1
}

// nonWallPlaneFoot is the tangent foot of centre m on the arm's PLANE host (its non-wall host) — the
// shared plane vertex (V1/V2). Declines if the arm has no plane host (a malformed corner).
func nonWallPlaneFoot(a edgeFillet, wallFace *topo.Face, m math.Point3) (math.Point3, bool) {
	for _, f := range [2]*topo.Face{a.a, a.b} {
		if f == wallFace {
			continue
		}
		if _, isPl := f.Geometry().(geom.Plane); isPl {
			return planeFootPoint(f, m), true
		}
	}
	return math.Point3{}, false
}

// crossSectionArc is one arm's cross-section circle (centre m, radius r) between its two host tangent
// feet, oriented from→to. The short (fillet-side) arc is pinned by its mid = m + r·bisector of the two
// endpoint radials, so Arc3dByThreePoints recovers exactly the OCCT edge (validated: E1/E4 mids).
func crossSectionArc(m, from, to math.Point3, r float64) (geom.Arc3d, bool) {
	r0, err0 := math.UnitVector3FromVector(m.VectorTo(from))
	r1, err1 := math.UnitVector3FromVector(m.VectorTo(to))
	if err0 != nil || err1 != nil {
		return geom.Arc3d{}, false
	}
	bis, err := math.UnitVector3FromVector(r0.AsVector().Add(r1.AsVector()))
	if err != nil {
		return geom.Arc3d{}, false // antipodal feet (a half-turn): no short arc
	}
	arc, err := geom.Arc3dByThreePoints(from, m.TranslateBy(bis.AsVector().Scale(r)), to)
	return arc, err == nil
}

// wallBridgeRail builds the on-wall bridge E2 from wStart to wEnd. Its shape is a cubic Hermite whose
// two END-TANGENT DIRECTIONS are exact — each is the wall-contact ruling tangent of the adjacent arm
// (a cylinder arm touches the wall along a ruling → wall axis; a torus arm along a circle → azimuthal)
// — oriented into the span, at magnitude wallBridgeFullness·chord. Every sample is radially projected
// to the wall (projectWall), so the rail is on-wall by construction; the interpolating fit's between-
// sample residual is verified ≤ res.Weld·R by the test. Declines if a wall-contact tangent is undefined.
func wallBridgeRail(wall geom.Cylinder, armStart, armEnd geom.Surface, wStart, wEnd math.Point3) (geom.BSplineCurve, bool) {
	chord := wEnd.VectorTo(wStart).Length() // = wStart..wEnd distance
	tStart, ok0 := wallContactTangent(armStart, wall, wStart, wEnd)
	tEnd, ok1 := wallContactTangent(armEnd, wall, wEnd, wStart)
	if !ok0 || !ok1 || float64(chord) == 0 {
		return geom.BSplineCurve{}, false
	}
	mag := wallBridgeFullness * float64(chord)
	pts := sampleWallHermite(wall, wStart, wEnd, tStart.Scale(mag), tEnd.Scale(-mag), wallBridgeSamples)
	c, err := geom.NewFittedBSplineCurve(pts)
	return c, err == nil
}

// sampleWallHermite samples the cubic Hermite P(0)=wStart, P(1)=wEnd with end tangents m0,m1, radially
// projecting each point onto the wall so the sampled rail lies exactly on the wall (radius R).
func sampleWallHermite(wall geom.Cylinder, wStart, wEnd math.Point3, m0, m1 math.Vector3, n int) []math.Point3 {
	pts := make([]math.Point3, n)
	for i := range n {
		t := float64(i) / float64(n-1)
		h00 := 2*t*t*t - 3*t*t + 1
		h10 := t*t*t - 2*t*t + t
		h01 := -2*t*t*t + 3*t*t
		h11 := t*t*t - t*t
		raw := wStart.AsVector().Scale(h00).Add(m0.Scale(h10)).Add(wEnd.AsVector().Scale(h01)).Add(m1.Scale(h11))
		pts[i] = cylinderWallPoint(wall, math.P3(float64(raw.X), float64(raw.Y), float64(raw.Z)))
	}
	return pts
}

// wallContactTangent is the unit tangent of arm's wall-contact curve at wall foot f, oriented so it
// points into the bridge span (positive component toward `toward`). A cylinder arm (axis ∥ wall axis)
// touches along a ruling → the wall axis direction; a torus arm (axis ∥ wall axis) along a latitude
// circle → the azimuthal direction (axis × radial). Declines for any other adjacent surface.
func wallContactTangent(arm geom.Surface, wall geom.Cylinder, f, toward math.Point3) (math.Vector3, bool) {
	axis := wall.AxisDir.AsVector()
	var dir math.Vector3
	switch arm.(type) {
	case geom.Cylinder:
		dir = axis
	case geom.Torus:
		radial, err := math.UnitVector3FromVector(perpToAxis(wall, f))
		if err != nil {
			return math.Vector3{}, false
		}
		dir = axis.Cross(radial.AsVector())
	default:
		return math.Vector3{}, false
	}
	return orientInto(dir, f, toward), true
}

// orientInto returns d or −d, whichever points from f toward `toward` (positive dot with f→toward).
func orientInto(d math.Vector3, f, toward math.Point3) math.Vector3 {
	if d.Dot(f.VectorTo(toward)) < 0 {
		return d.Scale(-1)
	}
	return d
}

// perpToAxis is f minus its projection onto the wall axis line — the radial vector from the axis to f.
func perpToAxis(wall geom.Cylinder, f math.Point3) math.Vector3 {
	axis := wall.AxisDir.AsVector()
	w := wall.Origin.VectorTo(f)
	return w.Sub(axis.Scale(w.Dot(axis)))
}
