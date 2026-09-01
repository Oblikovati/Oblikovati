// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Elliptical-cylinder-host arm fillets — the elliptic-prism vein (F4/T5/U2 …). Rounding a CONVEX
// straight RULING edge where a plane P meets a right geom.EllipticalCylinder wall E (both translation-
// invariant along the extrusion axis â, the edge ∥ â) with a constant radius r is — DESPITE E's
// non-constant curvature — an EXACT right CIRCULAR cylinder of radius r. The rolling-ball spine is
// {P offset r into material} ∩ {E offset r into material}: the intersection of two â-invariant
// surfaces is a straight line ∥ â, and the envelope of equal-radius spheres on a straight line is a
// right circular cylinder of radius r — independent of E's cross-section shape (geometry-math-advisor,
// f4-elliptical-canal-report.md; Rossignac & Requicha 1986 offset-of-translation-surface; do Carmo
// envelope of a sphere family over a straight centre-locus). OCCT stores this blend as a rational
// BSpline, but geometrically it is a cylinder, so we build it analytically and route it through the
// EXISTING single-arm curved runout weld (fillet_curved_single_runout.go) — no corner, no canal loft.
//
// Before this, such an edge FLAT-REFUSED at curvedAdjacentError ("edge bordering a curved
// (geom.EllipticalCylinder) face … not yet supported"). Gated to the EllipticalCylinder∧Plane RULING
// edge so it CANNOT touch the circular-cylinder / analytic-torus greens: a non-elliptic host, a
// non-ruling (curved/oblique) edge, a concave or varying-radius pick, or an arm that cannot be built
// all return handled=false and fall through to the byte-identical curvedAdjacentError refusal.

// rulingParallelTol is the direction slack (dimensionless, on unit vectors) for "the edge is a ruling
// of the elliptic cylinder" — the edge tangent ∥ the extrusion axis. Only a ruling gives a straight
// spine (hence a right circular cylinder); an oblique or curved elliptic edge is a genuine canal this
// slice does not build, so it must fall through. A generous 1e-6 admits imported-geometry drift.
const rulingParallelTol = 1e-6

// ellipticArmBracketDoublings caps the outward bracket search for the tangency root (each doubling
// grows the reach by ×2 from an initial r); a corner whose ball tangency is not bracketed within
// r·2^n has no reachable arm — floor honestly rather than loop.
const ellipticArmBracketDoublings = 40

// ellipticArmBisectIters is the bisection budget for the 1-D tangency root g(s)=dist(C(s),E)−r=0.
// Bisection halves the bracket each step, so 60 iterations drive |g| far below any model-relative
// weld tolerance (2^-60 of an r-scaled bracket) — machine-precise for the downstream weld's foot test.
const ellipticArmBisectIters = 60

// ellipticalCylinderPlaneEdge reports an edge bounded by one EllipticalCylinder face and one plane
// face, returning both surfaces and both faces (the plane face fixes the material-outward offset sign).
// The sibling of cylinderPlaneEdge / spherePlaneEdge for the elliptic-cylinder host.
func ellipticalCylinderPlaneEdge(e *topo.Edge) (ec geom.EllipticalCylinder, pl geom.Plane, ecFace, plFace *topo.Face, ok bool) {
	faces := e.Faces()
	if len(faces) != 2 {
		return geom.EllipticalCylinder{}, geom.Plane{}, nil, nil, false
	}
	for i := range 2 {
		c, okc := faces[i].Geometry().(geom.EllipticalCylinder)
		p, okp := faces[1-i].Geometry().(geom.Plane)
		if okc && okp {
			return c, p, faces[i], faces[1-i], true
		}
	}
	return geom.EllipticalCylinder{}, geom.Plane{}, nil, nil, false
}

// ellipticalCylinderArmEdge dispatches an EllipticalCylinder∧Plane edge to the analytic circular-
// cylinder arm builder. handled=true ONLY when the arm was built (the edge is diverted from the flat
// refusal); every decline returns handled=false so the edge falls through to the byte-identical
// curvedAdjacentError path — the maximally do-no-harm gate (an elliptic edge is stolen only when it
// can actually be greened). The sibling of sphereArmEdge / coneArmEdge.
func ellipticalCylinderArmEdge(body *topo.Body, e *topo.Edge, p filletPick) (edgeFillet, bool) {
	ec, pl, _, _, ok := ellipticalCylinderPlaneEdge(e)
	if !ok {
		return edgeFillet{}, false
	}
	arm, built := ellipticCylinderArmSurface(e, ec, pl, p, tol.ForBody(body))
	if !built {
		return edgeFillet{}, false // fall through to curvedAdjacentError, byte-identical
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: arm}, true
}

// ellipticCylinderArmSurface builds the exact right circular cylinder arm (radius r, axis ∥ the
// extrusion direction) for a CONVEX constant-radius ruling edge of the elliptic wall. Returns false —
// so ellipticalCylinderArmEdge falls through to the flat refusal (do-no-harm) — for a varying/concave
// pick, a non-ruling edge (curved/oblique elliptic edge, a genuine canal this slice does not build), an
// unreadable plane host normal, or a corner whose rolling-ball tangency has no reachable root.
func ellipticCylinderArmSurface(e *topo.Edge, ec geom.EllipticalCylinder, pl geom.Plane, p filletPick, res tol.Resolution) (geom.Cylinder, bool) {
	// NOT gated on ClassifyEdgeConvexity: the imported oblique-extrusion elliptic face carries an
	// unreliable Reversed flag (the STEP extrusion→EllipticalCylinder orientation defect), so the
	// dihedral classifier mis-calls this convex edge concave. Convexity/material-side is instead read
	// from the GEOM normal (insideEllipticCylinder, away-from-axis) when picking the spine branch.
	//
	// TODO(f4-convexity): that substitution is only HALF done — reading the material side off the geometric
	// normal answers "which side of the wall is solid", not "is this dihedral convex or concave", so a
	// genuinely CONCAVE ruling edge of the same wall would still be routed here as if it were convex (the
	// doc above claims a concave pick is declined; nothing actually declines it). PRE-EXISTING, out of the
	// canal band's scope. The sound fix is the same solid-probe the closed rim already uses —
	// ellipticRimConvexitySide's quadrant probe (fillet_elliptic_rim_spine.go) — lifted to serve both.
	if p.varying() {
		return geom.Cylinder{}, false // constant-radius only
	}
	edgeDir, err := math.UnitVector3FromVector(e.StartVertex().Point().VectorTo(e.EndVertex().Point()))
	if err != nil || !edgeDir.IsParallelTo(ec.AxisDir, rulingParallelTol) {
		return geom.Cylinder{}, false // only a straight ruling ∥ the axis is a right-cylinder fillet
	}
	nP, ok := planeHostNormal(e, pl)
	if !ok {
		return geom.Cylinder{}, false
	}
	center, ok := ellipticArmAxisPoint(e, ec, nP, p.r0, res)
	if !ok {
		return geom.Cylinder{}, false
	}
	arm, err := geom.NewCylinderWithRef(center, ec.AxisDir.AsVector(), nP.AsVector(), p.r0)
	return arm, err == nil
}

// ellipticArmAxisPoint solves for a point on the fillet cylinder's axis (the rolling-ball spine): the
// ball centre C on the plane offset r into the material (base − s·b, b ⊥ axis and ⊥ nP so C stays on
// that offset plane) that is ALSO at distance r from the elliptic wall (tangent). It roots the tangency
// in BOTH in-plane directions and keeps the root whose ball centre lies INSIDE E (the material side —
// the lens is interior to the wall), so the branch selection needs no material query on the solid.
func ellipticArmAxisPoint(e *topo.Edge, ec geom.EllipticalCylinder, nP math.UnitVector3, r float64, res tol.Resolution) (math.Point3, bool) {
	mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
	base := mid.TranslateBy(nP.AsVector().Scale(-r)) // r into the material, on the offset plane P'
	b, err := math.UnitVector3FromVector(ec.AxisDir.Cross(nP))
	if err != nil {
		return math.Point3{}, false // axis ∥ nP — impossible for a translation face, but guard
	}
	cPlus, okP := ellipticTangentRoot(ec, base, b, r, res)
	cMinus, okM := ellipticTangentRoot(ec, base, b.Negate(), r, res)
	return pickInteriorArmCentre(ec, base, cPlus, okP, cMinus, okM)
}

// pickInteriorArmCentre chooses between the two tangency roots (one per in-plane direction) the one
// whose ball centre sits INSIDE the elliptic wall (the material side); on a tie (both or neither
// interior) it keeps the candidate nearer the edge base. false when neither direction rooted.
func pickInteriorArmCentre(ec geom.EllipticalCylinder, base, cPlus math.Point3, okP bool, cMinus math.Point3, okM bool) (math.Point3, bool) {
	inP := okP && insideEllipticCylinder(ec, cPlus)
	inM := okM && insideEllipticCylinder(ec, cMinus)
	switch {
	case inP && !inM:
		return cPlus, true
	case inM && !inP:
		return cMinus, true
	case okP && okM:
		if base.DistanceTo(cPlus) <= base.DistanceTo(cMinus) {
			return cPlus, true
		}
		return cMinus, true
	case okP:
		return cPlus, true
	case okM:
		return cMinus, true
	}
	return math.Point3{}, false
}

// ellipticTangentRoot brackets and bisects the 1-D tangency g(s)=dist(base+s·dir, E)−r=0 for s≥0 (dir
// is one in-plane direction along the offset plane). It requires the ball to START inside the wall
// (g(0)≤0: the offset-plane point under the edge is within r of E) and to REACH r as it slides into the
// wedge; the outward doubling brackets the crossing, bisection lands it. false when g(0)>0 (this
// direction leaves E immediately — the other direction owns the corner) or no crossing is bracketed.
func ellipticTangentRoot(ec geom.EllipticalCylinder, base math.Point3, dir math.UnitVector3, r float64, res tol.Resolution) (math.Point3, bool) {
	g := func(s float64) float64 { return distToEllipticCylinder(ec, cAt(base, dir, s)) - r }
	if g(0) > 0 {
		return math.Point3{}, false
	}
	hi := r
	for i := 0; i < ellipticArmBracketDoublings && g(hi) < 0; i++ {
		hi *= 2
	}
	if g(hi) < 0 {
		return math.Point3{}, false // tangency never reached — no arm this direction
	}
	lo := 0.0
	for range ellipticArmBisectIters {
		m := 0.5 * (lo + hi)
		if g(m) < 0 {
			lo = m
		} else {
			hi = m
		}
	}
	c := cAt(base, dir, 0.5*(lo+hi))
	return c, stdmath.Abs(distToEllipticCylinder(ec, c)-r) <= tangencyTol(r, res)
}

// cAt is the ball-centre candidate base + s·dir.
func cAt(base math.Point3, dir math.UnitVector3, s float64) math.Point3 {
	return base.TranslateBy(dir.AsVector().Scale(s))
}

// tangencyTol is the model-relative acceptance band for |dist(C,E)−r|: the downstream single-runout
// weld tests each ball foot at res.Weld()·r (armRunoutFoot), so the arm centre must land inside that.
func tangencyTol(r float64, res tol.Resolution) float64 { return res.Weld() * r }

// distToEllipticCylinder is the true distance from C to the elliptic wall via the shared generic
// point-inversion (the SAME routine the weld's armRunoutFoot uses, so a root accepted here is tangent
// there by construction) — robust where an inner-offset parametrisation would be invalid (r above the
// wall's local curvature radius, e.g. F4's major-axis-vertex edge).
func distToEllipticCylinder(ec geom.EllipticalCylinder, c math.Point3) float64 {
	_, _, foot := geom.ClosestPointOnSurface(ec, c)
	return float64(foot.DistanceTo(c))
}

// insideEllipticCylinder reports whether C is on the material (interior) side of the elliptic wall:
// the foot-to-C vector opposes the wall's outward normal. Interior selects the rolling-ball spine
// branch that nestles in the material corner (the lens is interior to E), not its mirror in the void.
func insideEllipticCylinder(ec geom.EllipticalCylinder, c math.Point3) bool {
	u, v, foot := geom.ClosestPointOnSurface(ec, c)
	return foot.VectorTo(c).Dot(ec.NormalAt(u, v)) < 0
}
