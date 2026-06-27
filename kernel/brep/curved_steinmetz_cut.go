// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Equal-radius Steinmetz cut and join (M2 Phase 2, Oblikovati/Oblikovati#1335). Subtracting and uniting two
// equal-radius perpendicular cylinders, built from the same analytic ellipses as the Steinmetz intersection
// (curved_steinmetz.go). Where the intersection kept the four lobes, the cut and join keep the cylinders'
// OUTSIDE: each cylinder's side wall, with its two lobes removed, splits into two full-period BANDS — a top
// band between the top cap circle and the saddle rim, and a bottom band between the saddle rim and the
// bottom cap — and the saddle rim is two ellipse arcs meeting at the pinch points (so the band mesher pools
// them into one rim). The cut adds the other cylinder's two lobes back, REVERSED, as the bite walls; the
// join adds the other cylinder's two bands instead (its own outside).

// EqualRadiusSteinmetzCut builds target − tool for two equal-radius perpendicular cylinders (the target
// cylinder with the tool's bite removed), or ok=false to defer. The result is the target's two caps, its
// side split into a top and a bottom band (each a strip outside the tool), and the tool's two lobes inside
// the target, reversed as the bite walls.
//
// Example — a radius-3 cylinder on x with a radius-3 cylinder on z subtracted:
//
//	cx, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 3, 12)
//	cz, _ := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	res, ok := brep.EqualRadiusSteinmetzCut(cx, cz) // cx with a saddle bite
func EqualRadiusSteinmetzCut(target, tool *topo.Body) (*topo.Body, bool) {
	o, dirA, dirB, r, ok := steinmetzFrame(target, tool)
	if !ok {
		return nil, false
	}
	_, baseA, hA, _ := cylinderSolidParams(facesOfAny(target))
	return buildSteinmetzCut(o, dirA, dirB, r, baseA, hA), true
}

// EqualRadiusSteinmetzJoin builds a ∪ b for two equal-radius perpendicular cylinders, or ok=false to defer.
// The result is each cylinder's outside — two bands and two caps per cylinder — with the two cylinders'
// bands meeting directly along the shared saddle rims (the intersection ellipses), so no lobes appear.
//
// Example — two radius-3 cylinders on x and z united:
//
//	cx, _ := brep.SolidCylinder(math.P3(-6,0,0), math.V3(1,0,0), 3, 12)
//	cz, _ := brep.SolidCylinder(math.P3(0,0,-6), math.V3(0,0,1), 3, 12)
//	res, ok := brep.EqualRadiusSteinmetzJoin(cx, cz)
func EqualRadiusSteinmetzJoin(a, b *topo.Body) (*topo.Body, bool) {
	o, dirA, dirB, r, ok := steinmetzFrame(a, b)
	if !ok {
		return nil, false
	}
	_, baseA, hA, _ := cylinderSolidParams(facesOfAny(a))
	_, baseB, hB, _ := cylinderSolidParams(facesOfAny(b))
	return buildSteinmetzJoin(o, dirA, dirB, r, baseA, hA, baseB, hB), true
}

// buildSteinmetzJoin welds a ∪ b: each cylinder's two bands and caps. The two cylinders' bands share the
// four ellipse arcs (each arc used by one A-band and one B-band in opposite orientation), so the union is a
// closed (pinched) manifold solid with no interior faces.
func buildSteinmetzJoin(o math.Point3, dirA, dirB math.Vector3, r float64, baseA math.Point3, hA float64, baseB math.Point3, hB float64) *topo.Body {
	n := dirA.Cross(dirB)
	cA, _ := geom.NewCylinderWithRef(o, dirA, dirB, r)
	cB, _ := geom.NewCylinderWithRef(o, dirB, dirA, r)
	pPlus := o.TranslateBy(n.Scale(math.Scalar(r)))
	pMinus := o.TranslateBy(n.Scale(math.Scalar(-r)))
	topA := baseA.TranslateBy(dirA.Scale(math.Scalar(hA)))
	topB := baseB.TranslateBy(dirB.Scale(math.Scalar(hB)))

	bld := topo.NewBuilder(true, steinLin("body"))
	vP := bld.AddVertex(pPlus, steinLin("ph"))
	vM := bld.AddVertex(pMinus, steinLin("pl"))
	e1, e2, e3, e4 := addSteinmetzArcEdges(bld, o, dirA, dirB, r, vM, vP)

	// Cylinder A's bands (top saddle {e1,e3}, bottom {e2,e4}) and cylinder B's bands (the +b band's saddle is
	// the +b-side arcs {e1,e4}, the −b band's {e2,e3}). Each arc is shared by one A-band and one B-band in
	// opposite orientation — the two cylinders meet along the intersection ellipses.
	addCylBandCap(bld, cA, topA, pMinus, vM, o, dirA, []topo.Use{topo.Fwd(e1), topo.Rev(e3)}, "atop")
	addCylBandCap(bld, cA, baseA, pPlus, vP, o, dirA.Scale(-1), []topo.Use{topo.Fwd(e4), topo.Rev(e2)}, "abot")
	addCylBandCap(bld, cB, topB, pPlus, vP, o, dirB, []topo.Use{topo.Rev(e1), topo.Rev(e4)}, "btop")
	addCylBandCap(bld, cB, baseB, pPlus, vP, o, dirB.Scale(-1), []topo.Use{topo.Fwd(e2), topo.Fwd(e3)}, "bbot")
	return bld.Build()
}

// buildSteinmetzCut welds target − tool: the target cylinder's two bands and caps (its side outside the
// tool), plus the tool's two lobes reversed inward as the bite walls. Every ellipse arc is shared by one
// target band (its saddle rim) and one reversed tool lobe in opposite orientation, so the result is a closed
// (pinched) manifold solid.
func buildSteinmetzCut(o math.Point3, dirA, dirB math.Vector3, r float64, baseA math.Point3, hA float64) *topo.Body {
	n := dirA.Cross(dirB)
	cA, _ := geom.NewCylinderWithRef(o, dirA, dirB, r)
	cB1, _ := geom.NewCylinderWithRef(o, dirB, dirA, r)
	cB2, _ := geom.NewCylinderWithRef(o, dirB, dirA.Scale(-1), r)
	pPlus := o.TranslateBy(n.Scale(math.Scalar(r)))
	pMinus := o.TranslateBy(n.Scale(math.Scalar(-r)))
	topCenter := baseA.TranslateBy(dirA.Scale(math.Scalar(hA)))

	bld := topo.NewBuilder(true, steinLin("body"))
	vP := bld.AddVertex(pPlus, steinLin("ph"))
	vM := bld.AddVertex(pMinus, steinLin("pl"))
	e1, e2, e3, e4 := addSteinmetzArcEdges(bld, o, dirA, dirB, r, vM, vP)

	// The top band's saddle rim is {e1,e3}, the bottom band's is {e2,e4}; the tool lobes B1{e1,e3} and
	// B2{e2,e4} take the same arcs in the opposite orientation (see curved_steinmetz.go for the lobe arcs).
	addCylBandCap(bld, cA, topCenter, pPlus, vP, o, dirA, []topo.Use{topo.Rev(e1), topo.Fwd(e3)}, "top")
	addCylBandCap(bld, cA, baseA, pMinus, vM, o, dirA.Scale(-1), []topo.Use{topo.Rev(e2), topo.Fwd(e4)}, "bot")
	bld.AddReversedFace(cB1, steinLin("b1"), topo.OuterLoop(topo.Fwd(e1), topo.Rev(e3)))
	bld.AddReversedFace(cB2, steinLin("b2"), topo.OuterLoop(topo.Fwd(e2), topo.Rev(e4)))
	return bld.Build()
}

// addSteinmetzArcEdges creates the four shared ellipse-arc edges between the two pinch vertices (the E+ and
// E− ellipses each split front/back), returning them in the e1,e2,e3,e4 order the assemblers use.
func addSteinmetzArcEdges(bld *topo.Builder, o math.Point3, dirA, dirB math.Vector3, r float64, vM, vP *topo.Vertex) (e1, e2, e3, e4 *topo.Edge) {
	ePF, ePB, eMF, eMB := steinmetzArcs(o, dirA, dirB, r)
	e1 = bld.AddEdge(ePF, vM, vP, steinLin("e1")) // E+ front: P− → P+
	e2 = bld.AddEdge(ePB, vP, vM, steinLin("e2")) // E+ back:  P+ → P−
	e3 = bld.AddEdge(eMF, vM, vP, steinLin("e3")) // E− front: P− → P+
	e4 = bld.AddEdge(eMB, vP, vM, steinLin("e4")) // E− back:  P+ → P−
	return e1, e2, e3, e4
}

// addCylBandCap adds one full-period band of a cylinder's side (the strip between a cap circle and the
// saddle rim) and its planar cap. The seam runs from the saddle's pinch vertex straight up the cylinder to
// the cap circle (a hole-free ruling), so the band has one closed cap rim and one saddle rim (the arcs in
// `saddle`); the band keeps the cylinder's natural outward normal, the cap faces along capOutward.
func addCylBandCap(bld *topo.Builder, cyl geom.Cylinder, capCenter, pinchPt math.Point3, vPinch *topo.Vertex, o math.Point3, capOutward math.Vector3, saddle []topo.Use, tag string) {
	capSeamPt := capCenter.TranslateBy(o.VectorTo(pinchPt)) // directly along the axis above the pinch
	capCircle := seamedCircle(capCenter, cyl.AxisDir, capSeamPt, cyl.Radius)
	vCap := bld.AddVertex(capSeamPt, steinLin(tag+"vc"))
	eCap := bld.AddEdge(capCircle, vCap, vCap, steinLin(tag+"ec"))
	eSeam := bld.AddEdge(geom.NewLineSegment(pinchPt, capSeamPt), vPinch, vCap, steinLin(tag+"es"))

	loop := append([]topo.Use{topo.Rev(eSeam)}, saddle...)
	loop = append(loop, topo.Fwd(eSeam), topo.Fwd(eCap))
	bld.AddFace(cyl, steinLin(tag+"band"), topo.OuterLoop(loop...))
	capPlane, _ := geom.NewPlane(capCenter, capOutward)
	bld.AddFace(capPlane, steinLin(tag+"cap"), topo.OuterLoop(topo.Rev(eCap)))
}

// seamedCircle builds a cap circle whose angle-zero seam vertex is at seamPt (radius and centre on the
// cap), so a cylinder wall sharing it can seam there. (Retained from the bespoke crossing handlers; the
// equal-radius Steinmetz cut is now its only caller.)
func seamedCircle(center math.Point3, axis math.UnitVector3, seamPt math.Point3, radius float64) geom.Circle {
	ref, err := math.UnitVector3FromVector(center.VectorTo(seamPt))
	if err != nil {
		return geom.Circle{Center: center, Normal: axis, RefDir: axis, Radius: radius}
	}
	return geom.Circle{Center: center, Normal: axis, RefDir: ref, Radius: radius}
}
