// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CONCAVE Cylinder∧Plane single-arm runout fillet (group-a-concave-arm-derivation.md, corpus
// N3/M4/N9, plus the concave-torus-arm-derivation.md torus sibling, corpus H7/M8/O1/M3/M9/O2). On a
// REENTRANT axis-parallel Cylinder∧Plane LINE edge the rolling ball sits in the void wedge and the
// fillet ADDS material. The ball-centre locus is a straight line ∥ the axis (â ∥ P), so the arm is an
// exact geom.Cylinder of radius r — the concave DUAL of the convex cylinderArmSurface, with BOTH
// r-offsets flipped OUTWARD (ball into the void): the plane offset −r → +r and the radial offset
// R−r → R+ε·r (ε = n_C·r̂ ∈ {+1 boss, −1 bore}). It is routed via a SEPARATE dispatch branch
// (concaveCurvedArmFillet) guarded on a single concave Cylinder∧Plane line edge, disambiguated from the
// spurious material-side (convex mirror) root by the void gate PointInsideBody(centre)==false plus
// contact-foot tangency. On a REENTRANT circle (armTorus) edge the ball-centre locus is coaxial with
// the host, so the arm is an exact geom.Torus, major R+r, UNIQUE (no ± ruling disambiguation —
// concave-torus-arm-derivation.md §3) — concaveTorusArmSurface, already shipped for the miter weld
// (fillet_miter_concave.go), reused here verbatim. The 3-pick concave corners and the concave sphere
// arm are later slices.

// concaveCurvedArmFillet dispatches a reentrant Cylinder∧Plane edge to its exact concave arm — the
// LINE edge's cylinder arm (N3/M4/N9, concaveCylinderArmEdge) or the CIRCLE edge's torus arm (H7/M8/
// O1/M3/M9/O2, concaveTorusArmEdge) — and packs it into an edgeFillet marked armConcave so the
// single-arm runout weld winds the arm band into the material. Returns (false, nil) — so cylinderArmEdge
// keeps the do-no-harm floor — for a varying pick, an inward-fill request, a convex/tangent edge, an
// oblique (armRejected) concave edge (a later slice), or either arm builder's own do-no-harm decline.
// Returns (false, err) — an HONEST reject carrying the offending r and host R — for a line-arm
// constructor decline (spindle/clearance/degenerate frame, derivation §4); the torus arm has no
// analogous reject (major R+r > 0 always, concave-torus-arm-derivation.md §4).
func concaveCurvedArmFillet(body *topo.Body, e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, p filletPick, res Resolution, concave ConcaveFill) (edgeFillet, bool, error) {
	if p.varying() || concave != FillConcaveOutward || ClassifyEdgeConvexity(e) != EdgeConcave {
		return edgeFillet{}, false, nil // convex-external, inward recess, and tangent edges are not this path
	}
	switch classifyCurvedArm(cyl, pl, res) {
	case armTorus:
		return concaveTorusArmEdge(e, cyl, pl, p.r0, res)
	case armCylinder:
		return concaveCylinderArmEdge(body, e, cyl, pl, p, res)
	default:
		return edgeFillet{}, false, nil // armRejected: oblique ellipse edge (config iii) — later slice
	}
}

// concaveTorusArmEdge builds the UNIQUE concave torus arm (R+r, no plus/minus root selection —
// concave-torus-arm-derivation.md §3) on a reentrant circle Cylinder∧Plane edge (H7/M8/O1 single-corner,
// M3/M9/O2 miter) and packs it into an edgeFillet marked armConcave. planeHostNormal is the SAME
// material-outward-normal helper the convex curvedArmFillet uses; concaveTorusArmSurface then offsets
// both the major radius (R+r) and the centre (+r·outwardN) into the void, the exact mirror of the
// shipped convex torusArmSurface (derivation §3). Returns (false, nil) — do-no-harm — when the plane
// host normal is unreadable or the constructor declines a degenerate torus frame; there is no honest
// error here because major R+r > 0 always holds (no spindle case, derivation §4).
func concaveTorusArmEdge(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, r float64, res Resolution) (edgeFillet, bool, error) {
	outwardN, ok := planeHostNormal(e, pl)
	if !ok {
		return edgeFillet{}, false, nil // no readable plane host normal — cannot offset the ball into the void
	}
	tor, ok := concaveTorusArmSurface(cyl, pl, outwardN, r, concaveTorusWallSign(e, cyl), res)
	if !ok {
		return edgeFillet{}, false, nil // degenerate torus frame (axis ∥ ref) or a notch spindle — do-no-harm
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: tor, armConcave: true}, true, nil
}

// concaveTorusWallSign is ε ∈ {+1,−1} for a concave circle Cylinder∧Plane edge's cove torus: +1 BOSS
// (material inside the wall, ball-centre circle at R+r outside it — the historical M8/O1/N4 cove) and
// −1 NOTCH/BORE (material outside the wall, ball rolls INSIDE it at R−r — DRAWEXE M5 ground truth,
// oracle cove torus maj=25=R−r on the notch ceiling). Reuses cylinderHostRadialSign (the concave line
// arm's exact n_C·r̂ read) and DEFAULTS to +1 on an unreadable sign (an on-axis edge), so every boss
// cove stays byte-identical — the sibling of convexArmWallSign for the concave circle edge.
func concaveTorusWallSign(e *topo.Edge, cyl geom.Cylinder) float64 {
	if eps, ok := cylinderHostRadialSign(e, cyl); ok {
		return eps
	}
	return 1
}

// concaveCylinderArmEdge builds the exact concave LINE arm (N3/M4/N9) via planeHostNormal →
// concaveCylinderArmCandidates → selectConcaveArmRoot — extracted verbatim from
// concaveCurvedArmFillet's former body (behaviour BYTE-IDENTICAL) so the armTorus dispatch above could
// be added without pushing concaveCurvedArmFillet over funlen.
func concaveCylinderArmEdge(body *topo.Body, e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, p filletPick, res Resolution) (edgeFillet, bool, error) {
	planeN, ok := planeHostNormal(e, pl)
	if !ok {
		return edgeFillet{}, false, nil // no readable plane host normal — cannot offset the ball into the void
	}
	plus, minus, err := concaveCylinderArmCandidates(e, cyl, pl, planeN, p.r0, res)
	if err != nil {
		return edgeFillet{}, false, err // spindle/clearance/degenerate — honest reject with r, R
	}
	arm, ok := selectConcaveArmRoot(body, e, cyl, pl, plus, minus, p.r0, res)
	if !ok {
		return edgeFillet{}, false, nil // 0 or 2 candidates pass the void+foot gate — do-no-harm floor
	}
	faces := e.Faces()
	return edgeFillet{a: faces[0], b: faces[1], edge: e, armSurface: arm, armConcave: true}, true, nil
}

// concaveCylinderArmCandidates builds BOTH config-(ii) CONCAVE cylinder arms (derivation §1) — the two
// P_r∩C_ρ rulings as exact radius-r cylinders whose axis ∥ the host axis. The offset plane is pushed +r
// into the VOID and the coaxial offset cylinder sits at ρ = R + ε·r (ε = n_C·r̂: +1 boss / −1 bore, so
// centre = wall + r·n_C). The two are NOT yet disambiguated — selectConcaveArmRoot's void+foot gate picks
// the physical one. Returns an HONEST error (carrying r and R) on a bore spindle (ρ reaches the axis),
// a clearance (P_r clears C_ρ, no real ruling), or a degenerate arm frame.
//
// Example: concaveCylinderArmCandidates(bossWallEdge{R:20}, boss, radialPlane, planeN, 5, res) → the two
// radius-5 rulings on the void-side offset cylinder at distance R+r=25 from the boss axis (N3's arm pair).
func concaveCylinderArmCandidates(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, planeN math.UnitVector3, r float64, res Resolution) (plus, minus geom.Cylinder, err error) {
	rho, err := concaveArmOffsetRadius(e, cyl, r, res)
	if err != nil {
		return geom.Cylinder{}, geom.Cylinder{}, err
	}
	pBase, mBase, ok := concaveArmRulingBases(cyl, pl, planeN, rho, r)
	if !ok {
		return geom.Cylinder{}, geom.Cylinder{}, concaveClearanceErr(cyl, pl, planeN, rho, r)
	}
	axis, ref := cyl.AxisDir.AsVector(), planeN.AsVector()
	plus, ep := geom.NewCylinderWithRef(pBase, axis, ref, r)
	minus, em := geom.NewCylinderWithRef(mBase, axis, ref, r)
	if ep != nil || em != nil {
		return geom.Cylinder{}, geom.Cylinder{}, fmt.Errorf("concave cylinder arm: degenerate arm frame (axis ∥ plane normal) for r=%g on host R=%g", r, cyl.Radius)
	}
	return plus, minus, nil
}

// concaveArmOffsetRadius is ρ = R + ε·r (derivation §1): the ball-centre line's perpendicular distance to
// the host axis, ε = n_C·r̂ ∈ {+1 boss, −1 bore}. The bore spindle reject (ρ = R−r reaches the axis when
// r ≥ R) carries the offending r and R per §4 / CLAUDE.md. Errors on an on-axis edge (ε undefined).
func concaveArmOffsetRadius(e *topo.Edge, cyl geom.Cylinder, r float64, res Resolution) (float64, error) {
	eps, ok := cylinderHostRadialSign(e, cyl)
	if !ok {
		return 0, fmt.Errorf("concave cylinder arm: radial sign undefined at an on-axis edge (host R=%g)", cyl.Radius)
	}
	rho := cyl.Radius + eps*r
	if rho < armSpindleBand*res.Weld() {
		return 0, fmt.Errorf("concave cylinder arm: rolling radius r=%g must be < host radius R=%g (bore spindle: ρ=R−r=%g reaches the axis)", r, cyl.Radius, rho)
	}
	return rho, nil
}

// concaveClearanceErr reports the P_r∩C_ρ clearance reject (disc = ρ²−m² ≤ 0: the offset plane clears
// the offset cylinder, so there is no real ruling — derivation §4) with the offending ρ, |m|, r, and R.
func concaveClearanceErr(cyl geom.Cylinder, pl geom.Plane, planeN math.UnitVector3, rho, r float64) error {
	m := r - pl.Origin.VectorTo(cyl.Origin).Dot(planeN.AsVector())
	return fmt.Errorf("concave cylinder arm: offset plane clears the offset cylinder (ρ=%g ≤ |m|=%g); no real ruling for r=%g on host R=%g", rho, stdmath.Abs(m), r, cyl.Radius)
}

// selectConcaveArmRoot picks the PHYSICAL concave arm from the two candidate rulings by the void+foot
// gate (concaveArmRootValid), NOT by nearest-to-edge-midpoint (derivation §2 / §Numerical pitfalls):
// it validates BOTH candidates and returns the SOLE one that passes. ok=false when zero or both pass —
// an ambiguous/degenerate config the caller then floors do-no-harm — so a symmetric config that would
// tie nearest-midpoint to the spurious mirror ruling can never ship or needlessly floor the wrong root.
func selectConcaveArmRoot(body *topo.Body, e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, plus, minus geom.Cylinder, r float64, res Resolution) (geom.Cylinder, bool) {
	okPlus := concaveArmRootValid(body, e, plus, cyl, pl, r, res)
	okMinus := concaveArmRootValid(body, e, minus, cyl, pl, r, res)
	if okPlus == okMinus {
		return geom.Cylinder{}, false // 0 or 2 candidates satisfy the gate — cannot disambiguate
	}
	if okPlus {
		return plus, true
	}
	return minus, true
}

// cylinderHostRadialSign is ε = n_C·r̂ ∈ {+1,−1}: the sign that selects ρ = R + ε·r, from the cylinder
// FACE's material-outward normal n_C at the edge and the outward radial r̂ = unit(edge_mid − axis_foot).
// ε=+1 (boss: material inside the wall, void outside → centre at R+r) or −1 (bore: void inside the wall
// → centre at R−r). Because n_C is exactly radial, ε is exact. False when r̂ is undefined (edge on axis)
// or the cylinder host face carries no readable outward normal.
func cylinderHostRadialSign(e *topo.Edge, cyl geom.Cylinder) (float64, bool) {
	mid := edgeMidpoint(e)
	rhat, err := math.UnitVector3FromVector(cylinderBallCenter(cyl, mid).VectorTo(mid))
	if err != nil {
		return 0, false
	}
	nC, ok := cylinderHostOutwardNormal(e, cyl, mid)
	if !ok {
		return 0, false
	}
	if nC.Dot(rhat.AsVector()) < 0 {
		return -1, true
	}
	return 1, true
}

// cylinderHostOutwardNormal is the material-outward unit normal of the CYLINDER host face of e at p —
// the Reversed-aware outwardFaceNormal of the face whose geometry is cyl (the sibling of planeHostNormal
// for the curved wall). False when the edge borders no matching cylinder face (a defensive guard).
func cylinderHostOutwardNormal(e *topo.Edge, cyl geom.Cylinder, p math.Point3) (math.Vector3, bool) {
	for _, f := range e.Faces() {
		if c, isCyl := f.Geometry().(geom.Cylinder); isCyl && c == cyl {
			return outwardFaceNormal(f, p)
		}
	}
	return math.Vector3{}, false
}

// concaveArmRulingBases returns BOTH rulings of P_r∩C_ρ for the CONCAVE arm (derivation §1) —
// armRulingBase's dual with the plane offset flipped into the void: in the axis frame a ruling centre w
// satisfies |w|=ρ and w·n̂_P = +r − (A−p_P)·n̂_P (the plane pushed +r into the void), so
// w = m·n̂_P ± √(ρ²−m²)·(â×n̂_P) are the two rulings. ok=false when the radicand is non-positive (P_r
// grazes or clears C_ρ). The two are handed to selectConcaveArmRoot, which disambiguates them by the
// void+foot gate — NOT by nearest-to-midpoint, so a symmetric config cannot pick the spurious mirror.
func concaveArmRulingBases(cyl geom.Cylinder, pl geom.Plane, planeN math.UnitVector3, rho, r float64) (plus, minus math.Point3, ok bool) {
	a := cyl.AxisDir.AsVector()
	b := a.Cross(planeN.AsVector()) // ⟂ both axis and plane normal (config ii: n̂_P ⟂ â), unit length
	m := r - pl.Origin.VectorTo(cyl.Origin).Dot(planeN.AsVector())
	disc := rho*rho - m*m
	if disc <= 0 {
		return math.Point3{}, math.Point3{}, false
	}
	t := stdmath.Sqrt(disc)
	off := planeN.AsVector().Scale(m)
	plus = cyl.Origin.TranslateBy(off.Add(b.Scale(t)))
	minus = cyl.Origin.TranslateBy(off.Sub(b.Scale(t)))
	return plus, minus, true
}

// concaveArmRootValid disambiguates the physical concave root from the spurious mirror ruling
// (derivation §2): the ball centre at the edge midpoint's axial station must (1) sit in the VOID
// (PointInsideBody == false), (2) be internally tangent (distance ≈ r) to BOTH host surfaces, AND
// (3) land its PLANE contact foot inside the plane host's REAL trimmed loop. Condition (3) is the exact
// discriminator §2.2/§3 mandate: for these finite bodies BOTH candidate centres sit outside the solid
// (so the void gate alone accepts both) and BOTH are tangent to the infinite host surfaces — only the
// physical root's plane foot RECEDES into the pre-existing plane loop; the mirror root's foot lands
// outside it. Tangency uses the model-relative tol res.Weld()·r (ADR-0042), as armRunoutFoot does.
func concaveArmRootValid(body *topo.Body, e *topo.Edge, arm, cyl geom.Cylinder, pl geom.Plane, r float64, res Resolution) bool {
	centre, ok := armBallCenter(arm, edgeMidpoint(e))
	if !ok || PointInsideBody(body, centre) {
		return false // undefined spine, or a root sitting in the material
	}
	cylFace, planeFace := concaveHostFaces(e, cyl, pl)
	if cylFace == nil || planeFace == nil {
		return false
	}
	tol := res.Weld() * r
	if _, okC := armRunoutFoot(cylFace, centre, r, tol); !okC {
		return false
	}
	footP, okP := armRunoutFoot(planeFace, centre, r, tol)
	return okP && planeFootOnTrimmedFace(planeFace, pl, footP)
}

// planeFootOnTrimmedFace reports whether the plane contact foot lands on the plane host's REAL trimmed
// region — inside its outer loop and outside every hole — in exact in-plane (u,v) coordinates. It is the
// planar discriminator the void gate alone cannot supply (both candidate centres sit outside the finite
// body): the physical concave root's plane foot recedes INTO the pre-existing loop (derivation §3), the
// spurious mirror root's foot falls OUTSIDE it.
func planeFootOnTrimmedFace(planeFace *topo.Face, pl geom.Plane, foot math.Point3) bool {
	uv := func(p math.Point3) math.Point2 {
		d := pl.Origin.VectorTo(p)
		return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
	}
	f2, inOuter := uv(foot), false
	for _, l := range planeFace.Loops() {
		if !probe.PointInLoop2D(f2, loopUVPolygon(l, uv)) {
			continue
		}
		if !l.IsOuter() {
			return false // inside a hole — off the trimmed region
		}
		inOuter = true
	}
	return inOuter
}

// loopUVPolygon projects a loop's boundary to the plane's (u,v) frame, sampling each arc segment's
// midpoint so a curved rim keeps its bulge for the ray-cast containment test.
func loopUVPolygon(l *topo.Loop, uv func(math.Point3) math.Point2) []math.Point2 {
	var poly []math.Point2
	for _, s := range segsFromLoop(l) {
		poly = append(poly, uv(s.from))
		if s.arc {
			poly = append(poly, uv(s.mid))
		}
	}
	return poly
}

// concaveHostFaces returns the edge's two host faces split by kind: the cylinder host (geometry == cyl)
// and the plane host (geometry == pl). Either is nil when no bordering face matches — a defensive guard
// (cylinderPlaneEdge already established both exist).
func concaveHostFaces(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane) (cylFace, planeFace *topo.Face) {
	for _, f := range e.Faces() {
		switch g := f.Geometry().(type) {
		case geom.Cylinder:
			if g == cyl {
				cylFace = f
			}
		case geom.Plane:
			if g == pl {
				planeFace = f
			}
		}
	}
	return cylFace, planeFace
}
