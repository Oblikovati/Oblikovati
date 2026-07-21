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
// rolling-ball fillet of radius r on the CONVEX CIRCLE edge where cylinder cyl meets plane pl with the
// axis perpendicular to the plane. The tube minor radius is r; the major radius and the centre offset
// SIDE are SELECTED per case, because a convex Cylinder∧Plane rim can admit either of TWO exact tori:
// the BOSS-CAP side (major R−r, centre offset r into the material along −n̂, the ball tucked in the
// interior corner — B3 and every prior caller) or the EXTERNAL-SHOULDER side (major R+r, centre offset
// r into the VOID along +n̂, the ball in the outside wedge — a convex shaft-shelf rim). The physical
// side is disambiguated by the contact-foot gate (torusArmFeetValid): the tube's tangent feet must
// land on the REAL trimmed host faces, mirroring the concave-arm void+foot discipline
// (fillet_arm_concave.go). When the faces are absent (bare-geometry callers/tests) or the gate cannot
// single out the external-shoulder side, the boss-cap side is kept — so every current caller stays
// byte-identical (the external-shoulder branch fires ONLY when the outer feet uniquely validate).
// Built via NewTorusWithRef so the u=0 seam aligns with the boss wall (Oblikovati#129). Returns false
// when the boss-cap major R−r collapses onto the axis (r ≥ R) and no external-shoulder side is chosen.
//
// NOTE: OCCT blend/simple H6 (a shaft-shelf rim) does NOT route here — its rim edge is CONCAVE and is
// dispatched to the legacy arc-rim path (FilletCylinderArc, fillet_arc.go) before this M5 builder; see
// h6-root2-torus-side-report.md. This side-selection hardens the CONVEX external-shoulder case that
// DOES route through the M5 arm path (a fillet-of-a-fillet leaving a convex shaft-shelf).
//
// Example: torusArmSurface(bossWall{R:50}, topCap{z:100}, capFace, wallFace, rimMid, +ẑ, 10, res) →
// boss-cap torus centre (0,0,90), major 40, minor 10 (B3, OCCT BREP `5 0 0 90 0 0 1 … 40 10`); a convex
// external-shoulder rim (R=50 wall, shelf annulus below) instead selects R+r=60 into the void.
func torusArmSurface(cyl geom.Cylinder, pl geom.Plane, cylFace, planeFace *topo.Face, edgeMid math.Point3, outwardN math.UnitVector3, r float64, res Resolution) (geom.Torus, bool) {
	inner, innerOK := torusArmCandidate(cyl, pl, outwardN, -1, r, res) // boss-cap: R−r, centre into the material
	outer, outerOK := torusArmCandidate(cyl, pl, outwardN, +1, r, res) // external-shoulder: R+r, centre into the void
	if outerOK &&
		torusArmFeetValid(cylFace, planeFace, pl, outer, true, edgeMid, r, res) &&
		!torusArmFeetValid(cylFace, planeFace, pl, inner, innerOK, edgeMid, r, res) {
		return outer, true // external-shoulder is the SOLE physically valid side (its feet land on the real hosts)
	}
	return inner, innerOK // boss-cap side, or an ambiguous/unresolvable gate — keep the byte-identical R−r result
}

// torusArmCandidate builds ONE of the two config-(i) torus arms: sigma=−1 is the BOSS-CAP side (major
// R−r, centre = axis∩plane offset r into the MATERIAL along −n̂) and sigma=+1 the EXTERNAL-SHOULDER
// side (major R+r, centre offset r into the VOID along +n̂). The R−r branch reproduces the pre-H6-root2
// construction byte-for-byte (same centre, same NewTorusWithRef frame). Returns false only when the
// major radius collapses onto the axis (the spindle guard bites sigma=−1 at r ≥ R; sigma=+1's R+r never
// collapses) or the torus frame is degenerate. outwardN is the plane FACE's material-outward normal
// (Reversed-aware) — NOT the raw geom normal, which on an imported cap can point into the material.
func torusArmCandidate(cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, sigma, r float64, res Resolution) (geom.Torus, bool) {
	majorR := cyl.Radius + sigma*r
	if majorR < armSpindleBand*res.Weld() {
		return geom.Torus{}, false // spindle: R−r reaches the axis (self-intersecting) — only bites sigma=−1
	}
	n := outwardN.AsVector()
	center := projectOntoPlane(cyl.Origin, pl).TranslateBy(n.Scale(sigma * r))
	tor, err := geom.NewTorusWithRef(center, n, cyl.Ref.AsVector(), majorR, r)
	return tor, err == nil
}

// torusArmFeetValid reports whether the candidate torus arm's rolling ball is the PHYSICAL fillet for
// the real trimmed body: the ball centre (the spine point at the edge midpoint's azimuth) must be
// internally tangent-r to the cylinder host AND its plane contact foot must land inside the plane
// host's REAL trimmed loop (planeFootOnTrimmedFace). This is the exact discriminator between the two
// convex sides — both tori are tangent to the INFINITE cylinder+plane, but only the physical side's
// plane foot recedes into the pre-existing cap/shelf loop; the wrong side's foot lands off it (B3: the
// R+r foot spills off the cap disk; H6: the R−r foot falls in the shelf's central hole). It mirrors
// concaveArmRootValid's plane-foot gate. False when a face is missing or the ball spine is undefined.
func torusArmFeetValid(cylFace, planeFace *topo.Face, pl geom.Plane, tor geom.Torus, built bool, edgeMid math.Point3, r float64, res Resolution) bool {
	if !built || cylFace == nil || planeFace == nil {
		return false
	}
	centre, ok := torusBallCenter(tor, edgeMid)
	if !ok {
		return false
	}
	tol := res.Weld() * r
	if _, okC := armRunoutFoot(cylFace, centre, r, tol); !okC {
		return false // ball not internally tangent-r to the cylinder host — not a contact foot
	}
	footP, okP := armRunoutFoot(planeFace, centre, r, tol)
	return okP && planeFootOnTrimmedFace(planeFace, pl, footP)
}

// cylinderArmSurface builds the config-(ii) exact cylinder arm (m5-curved-arm-derivation.md §D3): a
// rolling-ball fillet of radius r on the LINE edge where cylinder cyl meets plane pl with the axis
// parallel to the plane. The fillet cylinder's axis is the ruling of P_r∩C_ρ (the offset plane meets
// the coaxial offset cylinder of radius ρ=R−r in a pair of rulings; edge selects the near one) and
// its radius is r. Returns false when P_r clears C_ρ (no real ruling — the plane misses the wall).
//
// Example: cylinderArmSurface(wallEdge, bossWall{R:50}, radialPlane, 10) → radius-10 cylinder about
// the wall ruling (the B3 vertical-wall arm, OCCT BREP `2 … 10`).
func cylinderArmSurface(edge *topo.Edge, cyl geom.Cylinder, pl geom.Plane, outwardN math.UnitVector3, r float64) (geom.Cylinder, bool) {
	rho := cyl.Radius - r
	if rho <= 0 {
		return geom.Cylinder{}, false // convex spindle: the offset cylinder has collapsed
	}
	n := outwardN                                       // the plane FACE's material-outward normal (Reversed-aware): armRulingBase offsets the
	base, ok := armRulingBase(edge, cyl, pl, n, rho, r) // ruling −r into the MATERIAL, so its sign must be right
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
