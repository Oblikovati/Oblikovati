// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// M5 Slice A (m5-curved-arm-derivation.md §D2/§D3/§D5): the exact analytic arm surfaces that
// classifyCurvedArm dispatches to. A Plane∧Cylinder rolling-ball fillet is a TORUS when the axis is
// ⊥ the plane (config i, circle edge) and a CYLINDER when the axis is ∥ the plane (config ii, line
// edge) — both closed-form primitives, no canal surface — plus the cross-section quarter-arc the
// corner engine consumes. Wired into nothing here (Task 3 does that); unit-pinned to the OCCT B3
// BREP oracle (`5 … 40 10` torus, `2 … 10` cylinder).

// armSpindleBand is k in the existence guard R−r < k·res.Weld() (§Numerical pitfalls): a length band,
// scaled to the model (ADR-0042), below which the convex torus tube reaches the axis and the surface
// self-intersects (spindle/horn torus). k=4 sits at the top of the derivation's k≈2..4 so we reject
// generously rather than emit a near-degenerate torus. Not a bare 1e-6.
const armSpindleBand = 4

// torusArmSurface builds the config-(i) exact torus arm (m5-curved-arm-derivation.md §D2): a
// rolling-ball fillet of radius r on the CIRCLE edge where cylinder cyl meets plane pl with the axis
// perpendicular to the plane. The tube minor radius is r; the major radius is R−r for the convex
// external rim (the corpus case — a protruding boss cap), and the centre is the cylinder axis
// projected onto the plane, offset r into the material (opposite the plane's outward normal). Built
// via NewTorusWithRef so the u=0 seam aligns with the boss wall (Oblikovati#129). Returns false when
// R−r collapses onto the axis (r ≥ R): a self-intersecting spindle torus is never emitted.
//
// Example: torusArmSurface(bossWall{R:50}, topCap{z:100}, 10, res) → torus centre (0,0,90), axis ẑ,
// major 40, minor 10 (the B3 top-rim arm, OCCT BREP `5 0 0 90 0 0 1 … 40 10`).
func torusArmSurface(cyl geom.Cylinder, pl geom.Plane, r float64, res Resolution) (geom.Torus, bool) {
	majorR := cyl.Radius - r
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, false
	}
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return geom.Torus{}, false
	}
	inward := n.Negate().AsVector() // outward cap normal → into the material by r
	center := projectOntoPlane(cyl.Origin, pl).TranslateBy(inward.Scale(r))
	tor, err := geom.NewTorusWithRef(center, n.AsVector(), cyl.Ref.AsVector(), majorR, r)
	return tor, err == nil
}

// cylinderArmSurface builds the config-(ii) exact cylinder arm (m5-curved-arm-derivation.md §D3): a
// rolling-ball fillet of radius r on the LINE edge where cylinder cyl meets plane pl with the axis
// parallel to the plane. The fillet cylinder's axis is the ruling of P_r∩C_ρ (the offset plane meets
// the coaxial offset cylinder of radius ρ=R−r in a pair of rulings; edge selects the near one) and
// its radius is r. Returns false when P_r clears C_ρ (no real ruling — the plane misses the wall).
//
// Example: cylinderArmSurface(wallEdge, bossWall{R:50}, radialPlane, 10) → radius-10 cylinder about
// the wall ruling (the B3 vertical-wall arm, OCCT BREP `2 … 10`).
func cylinderArmSurface(edge *topo.Edge, cyl geom.Cylinder, pl geom.Plane, r float64) (geom.Cylinder, bool) {
	rho := cyl.Radius - r
	if rho <= 0 {
		return geom.Cylinder{}, false // convex spindle: the offset cylinder has collapsed
	}
	n, err := math.UnitVector3FromVector(pl.Normal())
	if err != nil {
		return geom.Cylinder{}, false
	}
	base, ok := armRulingBase(edge, cyl, pl, n, rho, r)
	if !ok {
		return geom.Cylinder{}, false
	}
	arm, err := geom.NewCylinderWithRef(base, cyl.AxisDir.AsVector(), n.AsVector(), r)
	return arm, err == nil
}

// armRulingBase returns a point on the selected ruling of P_r∩C_ρ (m5-curved-arm-derivation.md §D3).
// In the axis frame a ruling centre w satisfies |w|=ρ and w·n̂ = −r − (A−p_P)·n̂ (the offset plane at
// signed distance −r into the material); the two solutions w = m·n̂ ± √(ρ²−m²)·(â×n̂) are the two
// rulings, and the edge midpoint picks the near one. Returns false when the radicand is non-positive
// (P_r grazes or clears C_ρ — no real intersection).
func armRulingBase(edge *topo.Edge, cyl geom.Cylinder, pl geom.Plane, n math.UnitVector3, rho, r float64) (math.Point3, bool) {
	a := cyl.AxisDir.AsVector()
	b := a.Cross(n.AsVector()) // ⟂ both axis and normal (config ii: n̂ ⟂ â), unit length
	m := -r - pl.Origin.VectorTo(cyl.Origin).Dot(n.AsVector())
	disc := rho*rho - m*m
	if disc <= 0 {
		return math.Point3{}, false
	}
	t := stdmath.Sqrt(disc)
	off := n.AsVector().Scale(m)
	plus := cyl.Origin.TranslateBy(off.Add(b.Scale(t)))
	minus := cyl.Origin.TranslateBy(off.Sub(b.Scale(t)))
	return nearerRuling(edge, plus, minus), true
}

// nearerRuling returns whichever ruling base lies closer to the picked edge's midpoint — the physical
// disambiguation of the two P_r∩C_ρ rulings (the edge sits on exactly one of them).
func nearerRuling(edge *topo.Edge, plus, minus math.Point3) math.Point3 {
	lo, hi := edge.StartVertex().Point(), edge.EndVertex().Point()
	mid := lo.TranslateBy(lo.VectorTo(hi).Scale(0.5))
	if float64(mid.DistanceTo(plus)) <= float64(mid.DistanceTo(minus)) {
		return plus
	}
	return minus
}

// curvedArmSectionArc is the corner cross-section quarter-arc on the torus arm
// (m5-curved-arm-derivation.md §D5): at the corner u-station the tube circle v∈[0,π/2] runs from the
// cyl-contact (v=0, on the wall) to the plane-contact (v=π/2, on the cap) — the [φ_P,φ_C] rail the
// corner engine consumes, the sibling of armSectionArc stationed on the tube. Only the torus arm
// needs this new stationed cross-section; the cylinder arm reuses armSectionArc (its section is the
// planar-equivalent one), so a non-torus surface returns nil.
func curvedArmSectionArc(arm geom.Surface, station float64) geom.Curve3 {
	tor, ok := arm.(geom.Torus)
	if !ok {
		return nil
	}
	arc, err := geom.Arc3dByThreePoints(tor.PointAt(station, 0), tor.PointAt(station, armSectionMid), tor.PointAt(station, armSectionEnd))
	if err != nil {
		return nil
	}
	return arc
}

// armSectionMid / armSectionEnd are the tube v-angles of the cross-section quarter: the 45° bisector
// sample (for the through-point) and the plane-contact end. v=0 (the cyl-contact) is implicit.
const (
	armSectionMid = stdmath.Pi / 4
	armSectionEnd = stdmath.Pi / 2
)
