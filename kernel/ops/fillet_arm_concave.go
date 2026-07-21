// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The CONCAVE Cylinder∧Plane single-arm runout fillet (group-a-concave-arm-derivation.md, corpus
// N3/M4/N9). On a REENTRANT axis-parallel Cylinder∧Plane LINE edge the rolling ball sits in the void
// wedge and the fillet ADDS material. The ball-centre locus is a straight line ∥ the axis (â ∥ P), so
// the arm is an exact geom.Cylinder of radius r — the concave DUAL of the convex cylinderArmSurface,
// with BOTH r-offsets flipped OUTWARD (ball into the void): the plane offset −r → +r and the radial
// offset R−r → R+ε·r (ε = n_C·r̂ ∈ {+1 boss, −1 bore}). It is routed via a SEPARATE dispatch branch
// (concaveCurvedArmFillet) guarded on a single concave Cylinder∧Plane line edge, disambiguated from the
// spurious material-side (convex mirror) root by the void gate PointInsideBody(centre)==false plus
// contact-foot tangency. The 3-pick concave corners and the concave torus/sphere arms are later slices.

// concaveCurvedArmFillet builds the exact concave cylinder arm on a reentrant axis-parallel
// Cylinder∧Plane LINE edge (N3/M4/N9) and packs it into an edgeFillet marked armConcave so the
// single-arm runout weld winds the arm band into the material. Returns (false, nil) — so cylinderArmEdge
// keeps the do-no-harm floor — for a varying pick, an inward-fill request, a convex/tangent edge, a
// non-line (torus/oblique) concave edge (a later slice), or an ambiguous void/foot gate. Returns
// (false, err) — an HONEST reject carrying the offending r and host R — for a constructor decline
// (spindle/clearance/degenerate frame, derivation §4).
func concaveCurvedArmFillet(body *topo.Body, e *topo.Edge, cyl geom.Cylinder, pl geom.Plane, p filletPick, res Resolution, concave ConcaveFill) (edgeFillet, bool, error) {
	if p.varying() || concave != FillConcaveOutward || ClassifyEdgeConvexity(e) != EdgeConcave {
		return edgeFillet{}, false, nil // convex-external, inward recess, and tangent edges are not this path
	}
	if classifyCurvedArm(cyl, pl, res) != armCylinder {
		return edgeFillet{}, false, nil // concave torus (circle edge) / oblique ellipse edge — later slice
	}
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
	cylFace, planeFace := cylinderPlaneHostFaces(e, cyl, pl)
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
		if !pointInLoop2D(f2, loopUVPolygon(l, uv)) {
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

// cylinderPlaneHostFaces returns the edge's two host faces split by kind: the cylinder host (geometry ==
// cyl) and the plane host (geometry == pl). Either is nil when no bordering face matches — a defensive
// guard (cylinderPlaneEdge already established both exist). Shared by the concave arm root gate and the
// convex torus-arm side-selection (torusArmSurface), so its name is host-kind, not convexity-specific.
func cylinderPlaneHostFaces(e *topo.Edge, cyl geom.Cylinder, pl geom.Plane) (cylFace, planeFace *topo.Face) {
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

// concaveArmHostRetrim re-clips one CONCAVE arm host (ef.a or ef.b) to the contact rail. Unlike the
// convex recede-and-splice (feet land INTERIOR to the host loop), a concave fillet ADDS the fill wedge:
// the host GROWS, its two contact feet landing on the flanking rim edges EXTENDED past the picked
// vertex (N9's rim arcs; N3's plane top edge). So the retrim replaces the picked-edge segment with the
// straight contact rail and RE-TERMINATES the two flanking rim edges onto the rail's feet (a straight
// ruling or a rim arc grown/receded to the foot on its OWN line/circle). Declines when the picked
// segment is absent from the loop or a foot is off a flanking edge's supporting line/circle (do-no-harm).
func concaveArmHostRetrim(host *topo.Face, rail endSeg, edge *topo.Edge, tol float64) (filletFace, bool) {
	v0, v1 := edge.StartVertex().Point(), edge.EndVertex().Point()
	bitten := hostBittenLoop(host, v0, tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false // malformed host (no loops) — do-no-harm
	}
	retrim, ok := concaveRetrimLoop(bitten, rail, v0, v1, tol)
	if !ok {
		return filletFace{}, false
	}
	loops := hostLoopsWithRetrim(host, bitten, outer, retrim)
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// concaveRetrimLoop rebuilds the bitten loop for a concave arm host: it finds the picked-edge segment
// (endpoints {v0,v1}), replaces it with the straight contact rail, and re-terminates the two flanking
// segments onto the rail's feet (the rail was built foot0↔v0 → foot1↔v1). Every other segment is
// carried through verbatim, so the ring stays closed by construction. Declines when the picked segment
// is not on the loop or a foot leaves a flanking edge's supporting line/circle.
func concaveRetrimLoop(bitten *topo.Loop, rail endSeg, v0, v1 math.Point3, tol float64) (filletLoop, bool) {
	segs := segsFromLoop(bitten)
	n := len(segs)
	i := indexOfPickedEdge(segs, v0, v1, tol)
	if i < 0 || n < 3 {
		return filletLoop{}, false
	}
	fFrom, fTo := railFeetForPicked(segs[i], rail, v0, tol)
	prev, okp := reterminateSegTo(segs[(i-1+n)%n], fFrom, tol)
	next, okn := reterminateSegFrom(segs[(i+1)%n], fTo, tol)
	if !okp || !okn {
		return filletLoop{}, false
	}
	return loopFromSegs(spliceConcaveRing(segs, i, prev, endSeg{from: fFrom, to: fTo}, next)), true
}

// indexOfPickedEdge returns the loop segment whose endpoints are the picked edge's vertices {v0,v1}
// (either orientation), or −1 when the picked edge is not a single segment of this loop.
func indexOfPickedEdge(segs []endSeg, v0, v1 math.Point3, tol float64) int {
	for i, s := range segs {
		if pointsMatch(s.from, s.to, v0, v1, tol) || pointsMatch(s.from, s.to, v1, v0, tol) {
			return i
		}
	}
	return -1
}

// pointsMatch reports whether (a,b) coincide with (c,d) within tol, respectively.
func pointsMatch(a, b, c, d math.Point3, tol float64) bool {
	return float64(a.DistanceTo(c)) <= tol && float64(b.DistanceTo(d)) <= tol
}

// railFeetForPicked orders the rail's feet to the picked segment's own traversal: the rail runs
// foot0↔v0 → foot1↔v1, so a picked segment traversed v0→v1 keeps (foot0,foot1), v1→v0 swaps them.
func railFeetForPicked(picked, rail endSeg, v0 math.Point3, tol float64) (math.Point3, math.Point3) {
	if float64(picked.from.DistanceTo(v0)) <= tol {
		return rail.from, rail.to
	}
	return rail.to, rail.from
}

// spliceConcaveRing rebuilds the ordered ring with the picked segment i replaced by the straight rail
// and its two neighbours replaced by their re-terminated forms — positions preserved so the ring stays
// closed (prev.to == rail.from, rail.to == next.from by construction).
func spliceConcaveRing(segs []endSeg, i int, prev, rail, next endSeg) []endSeg {
	n := len(segs)
	out := make([]endSeg, n)
	for k := 0; k < n; k++ {
		switch k {
		case (i - 1 + n) % n:
			out[k] = prev
		case i:
			out[k] = rail
		case (i + 1) % n:
			out[k] = next
		default:
			out[k] = segs[k]
		}
	}
	return out
}

// reterminateSegTo moves segment s's FAR endpoint to newTo along its own supporting line (straight) or
// circle (arc) — growing or receding the flanking rim edge to the contact foot. Declines when newTo
// leaves that line/circle (the foot is not co-linear/co-circular — not a valid concave grow).
func reterminateSegTo(s endSeg, newTo math.Point3, tol float64) (endSeg, bool) {
	if !s.arc {
		if !pointOnLine(s.from, s.to, newTo, tol) {
			return endSeg{}, false
		}
		return endSeg{from: s.from, to: newTo}, true
	}
	return rebuildArcSeg(s.curve.(geom.Arc3d), s.from, newTo, tol)
}

// reterminateSegFrom moves segment s's NEAR endpoint to newFrom along its own line/circle — the mirror
// of reterminateSegTo for the flanking edge on the picked segment's far side.
func reterminateSegFrom(s endSeg, newFrom math.Point3, tol float64) (endSeg, bool) {
	if !s.arc {
		if !pointOnLine(s.from, s.to, newFrom, tol) {
			return endSeg{}, false
		}
		return endSeg{from: newFrom, to: s.to}, true
	}
	return rebuildArcSeg(s.curve.(geom.Arc3d), newFrom, s.to, tol)
}

// rebuildArcSeg builds the sub-arc of arc's own circle between from and to (both required on that
// circle within tol) through the shorter-arc midpoint — the arc analogue of extending a straight rim
// edge to the foot. Declines when either endpoint is off the circle or the three-point fit fails.
func rebuildArcSeg(arc geom.Arc3d, from, to math.Point3, tol float64) (endSeg, bool) {
	if !onCircle(arc, from, tol) || !onCircle(arc, to, tol) {
		return endSeg{}, false
	}
	mid := arcMidBetween(arc.Center, arc.Radius, from, to)
	sub, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: from, to: to, curve: sub, mid: mid, arc: true}, true
}

// onCircle reports whether p lies on arc's circle (centre distance ≈ radius within tol).
func onCircle(arc geom.Arc3d, p math.Point3, tol float64) bool {
	return stdmath.Abs(float64(p.DistanceTo(arc.Center))-arc.Radius) <= tol
}

// pointOnLine reports whether p lies on the INFINITE line through a→b within tol (co-linear, so an
// extension past b is admitted — the concave grow).
func pointOnLine(a, b, p math.Point3, tol float64) bool {
	d := a.VectorTo(b)
	l2 := float64(d.Dot(d))
	if l2 == 0 {
		return false
	}
	t := float64(a.VectorTo(p).Dot(d)) / l2
	return float64(a.TranslateBy(d.Scale(math.Scalar(t))).DistanceTo(p)) <= tol
}

// concaveCapRetrim re-clips one CONCAVE end cap (run0/run1.capping): for a reentrant fillet the cap
// GAINS the fill wedge instead of losing a corner (variant (b), opposite sign to the convex cap bite),
// so the cap's far-vertex corner is REPLACED by the cross-section arc and the two edges meeting there
// are re-terminated onto the arc's feet (each on its own host-shared line/circle). Declines when the
// far vertex is not a loop corner or an arc foot is off a flanking edge — the do-no-harm floor.
func concaveCapRetrim(cap *topo.Face, arc endSeg, far math.Point3, tol float64) (filletFace, bool) {
	bitten := hostBittenLoop(cap, far, tol)
	outer := outerHostLoop(cap)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	retrim, ok := concaveCapLoop(bitten, arc, far, tol)
	if !ok {
		return filletFace{}, false
	}
	loops := hostLoopsWithRetrim(cap, bitten, outer, retrim)
	return filletFace{surface: cap.Geometry(), loops: loops, parent: cap.Lineage()}, true
}

// concaveCapLoop rebuilds a cap's bitten loop by replacing the far-vertex CORNER with the cross-section
// arc: the two flanking edges (each shared with one host) are re-terminated onto the arc's feet and the
// arc is spliced between them, growing the loop by one segment. Declines when the far vertex is not a
// loop corner or the arc feet do not land on the two flanking edges' supporting line/circle.
func concaveCapLoop(bitten *topo.Loop, arc endSeg, far math.Point3, tol float64) (filletLoop, bool) {
	segs := segsFromLoop(bitten)
	n := len(segs)
	j := indexOfVertex(segs, far, tol)
	if j < 0 || n < 3 {
		return filletLoop{}, false
	}
	prevIdx := (j - 1 + n) % n
	fFrom, fTo, ok := matchArcFeet(segs[prevIdx], segs[j], arc, tol)
	prev, okp := reterminateSegTo(segs[prevIdx], fFrom, tol)
	next, okn := reterminateSegFrom(segs[j], fTo, tol)
	if !ok || !okp || !okn {
		return filletLoop{}, false
	}
	arcSeg := endSeg{from: fFrom, to: fTo, curve: arc.curve, mid: arc.mid, arc: arc.arc}
	return loopFromSegs(spliceCapRing(segs, prevIdx, j, prev, arcSeg, next)), true
}

// indexOfVertex returns the segment index whose FROM endpoint coincides with p (the loop corner p
// starts), or −1 when no segment leaves p.
func indexOfVertex(segs []endSeg, p math.Point3, tol float64) int {
	for j, s := range segs {
		if float64(s.from.DistanceTo(p)) <= tol {
			return j
		}
	}
	return -1
}

// matchArcFeet assigns the cross-section arc's two endpoints to the two flanking cap edges: fFrom lands
// on prev's supporting line/circle, fTo on next's (the arc runs prev-side → next-side). Tries both
// endpoint orderings; ok=false when neither pairing lands each foot on its flanking edge.
func matchArcFeet(prev, next, arc endSeg, tol float64) (fFrom, fTo math.Point3, ok bool) {
	if segSupportsPoint(prev, arc.from, tol) && segSupportsPoint(next, arc.to, tol) {
		return arc.from, arc.to, true
	}
	if segSupportsPoint(prev, arc.to, tol) && segSupportsPoint(next, arc.from, tol) {
		return arc.to, arc.from, true
	}
	return math.Point3{}, math.Point3{}, false
}

// segSupportsPoint reports whether p lies on segment s's supporting geometry — its infinite line
// (straight) or its circle (arc) — the co-linear/co-circular test the concave grow re-terminates onto.
func segSupportsPoint(s endSeg, p math.Point3, tol float64) bool {
	if !s.arc {
		return pointOnLine(s.from, s.to, p, tol)
	}
	return onCircle(s.curve.(geom.Arc3d), p, tol)
}

// spliceCapRing rebuilds the ordered ring with the far-vertex corner (edges prevIdx→j) replaced by
// [prev, arc, next] — the arc INSERTED between the re-terminated flanking edges (the loop grows by one
// segment). prevIdx and j are adjacent (j = prevIdx+1 mod n), so order and closure are preserved.
func spliceCapRing(segs []endSeg, prevIdx, j int, prev, arc, next endSeg) []endSeg {
	out := make([]endSeg, 0, len(segs)+1)
	for k := range segs {
		switch k {
		case prevIdx:
			out = append(out, prev, arc)
		case j:
			out = append(out, next)
		default:
			out = append(out, segs[k])
		}
	}
	return out
}
