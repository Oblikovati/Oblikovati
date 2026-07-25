// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The N4-class corner PLAN BUILDER — the first customer of the general corner-weld layer, and the shape
// every later class (O1, H7, …) copies. It replaces the bespoke weld that used to live in
// fillet_curved_mixed_bspline_weld.go: the geometry half of that file (classifyN4MixedArms, solveN4Corner,
// the corner patch) is unchanged and still owns the solve; only the ORCHESTRATION — 200 lines that were
// ~60% a line-for-line copy of M8's — becomes the ~90 lines of declaration below.
//
// The class: a trihedral vertex where a CONCAVE cylinder arm (boss wall ∧ box wall) + a CONVEX torus arm
// (the boss cap rim) + a planar concave band meet. Its two cylinder arms TERMINATE at the corner on
// radius-r cross-section arcs (A1); the torus arm runs PAST it, bordering the patch along the on-torus rail
// B→C (A2). The patch's fourth side rides on the shared vertical plane (A3, the `mid` rail D→A).
//
// The role signature is DISJOINT from M8's (convex-cyl + concave-cove-torus + planar), asserted by
// TestClassifyN4DeclinesM8Roles, so the two paths can never both fire.

// n4CornerPlanBuilder recognises the N4 class and declares its plan. Nothing is mutated before took=true,
// so a non-N4 corner falls through the ladder with no trace (layer invariant #5).
type n4CornerPlanBuilder struct{}

// Plan declares the N4 corner site. took=false leaves the corner to the untouched legacy sphere path.
func (n4CornerPlanBuilder) Plan(_ *topo.Body, arms []edgeFillet, res Resolution) (cornerWeldPlan, bool, string) {
	roles, ok := classifyN4MixedArms(arms)
	if !ok {
		return cornerWeldPlan{}, false, ""
	}
	r, ok := armTubeRadius(roles.torus.armSurface)
	if !ok {
		return cornerWeldPlan{}, false, ""
	}
	corner, ok := solveN4Corner(roles, r, res)
	if !ok {
		return cornerWeldPlan{}, false, "" // not the N4 BSpline corner — keep the sphere path
	}
	vFace, ok := sharedPlaneHost(roles.ccyl, roles.band)
	if !ok {
		return cornerWeldPlan{}, true, "n4 corner: the two concave arms share no vertical plane host"
	}
	return n4Plan(roles, corner, vFace, r, res), true, ""
}

// n4Plan assembles the declarative plan: one ledger holding the four patch-ring rails, three arm specs, and
// the one host mid rail. Registering the ring ONCE is what makes the patch and its three welded neighbours
// read the identical curve objects (the property n4PatchFace used to maintain by hand).
func n4Plan(roles n4MixedArms, corner n4Corner, vFace *topo.Face, r float64, res Resolution) cornerWeldPlan {
	led := newCornerWeldLedger()
	ring := n4PatchRing(led, corner)
	return cornerWeldPlan{
		ledger: led,
		patch:  cornerPatchSpec{surface: corner.patch.Surface, sides: ring.sides},
		arms: []cornerArmSpec{
			n4TerminatingArm("ccyl", roles.ccyl, corner.pts.arcCD, ring.arcCD),
			n4TerminatingArm("band", roles.band, corner.pts.arcAB, ring.arcAB),
			n4LateralTorusArm(roles.torus, ring.railBC),
		},
		mids:     []cornerHostMid{{face: vFace, rails: []railID{ring.railDA}}},
		vertex:   n4CornerVertex(roles, res.Weld()*r),
		radius:   r,
		filleted: n4FilletedEdges(roles),
	}
}

// n4PatchRailRing is the four registered patch-boundary handles plus the ordered ring the patch reads.
type n4PatchRailRing struct {
	arcAB, railBC, arcCD, railDA railID
	sides                        []railID
}

// n4PatchRing registers the canal patch's four sides — arc A→B (band cross-section), rail B→C (the ball's
// contact locus on the torus arm), arc C→D (ccyl cross-section), rail D→A (its contact locus on the
// vertical plane) — each as a SINGLE curve-seg, never a sampled polyline (a sampled side would crack
// against the arm faces' single-seg curves). The two rails are the canal's own boundary isoparms, whose
// chord-length knots make Domain() the only safe source for the mid witness.
func n4PatchRing(led *cornerWeldLedger, c n4Corner) n4PatchRailRing {
	p := c.pts
	ab := led.add("n4/arcAB", endSeg{from: p.a, to: p.b, curve: p.arcAB, mid: p.arcAB.PointAt(0.5), arc: true})
	bc := led.add("n4/railBC", endSeg{from: p.b, to: p.c, curve: c.railBC, mid: curveMidPoint(c.railBC)})
	cd := led.add("n4/arcCD", endSeg{from: p.c, to: p.d, curve: p.arcCD, mid: p.arcCD.PointAt(0.5), arc: true})
	da := led.add("n4/railDA", endSeg{from: p.d, to: p.a, curve: c.railDA, mid: curveMidPoint(c.railDA)})
	return n4PatchRailRing{arcAB: ab, railBC: bc, arcCD: cd, railDA: da, sides: []railID{ab, bc, cd, da}}
}

// n4TerminatingArm declares one of the two CONCAVE cylinder arms: it ends at the corner on its radius-r
// cross-section arc (A1), runs out onto a plane cap at its far vertex (B1/B2), and its hosts GROW to its
// contact rails (the reentrant fillet ADDS material).
func n4TerminatingArm(role string, ef edgeFillet, arc geom.Arc3d, near railID) cornerArmSpec {
	return cornerArmSpec{
		role: role, ef: ef, surface: ef.armSurface,
		nearKind: armTerminatesAtArc, nearArc: arc, near: []railID{near},
		far: farCappedVertex, sense: growOutward,
	}
}

// n4LateralTorusArm declares the CONVEX torus arm (the boss cap-rim fillet). Two things distinguish it from
// a plain arm, and both are layer stage variants rather than N4 code:
//   - it runs PAST the corner, so its near boundary is the general on-torus rail B→C rather than a
//     cross-section arc (A2);
//   - its far end is a G1 SEAM on the boss wall, not a cap: only the 90° piece of the 270° cap rim was
//     picked, and the rim continues tangentially across the wall's second face. It therefore terminates by
//     rim CONTINUATION (B3) — the arm runs through the seam to the end of the tangent chain, splitting into
//     one face per host-face span (C4). OCCT's oracle confirms this is what a blend does here: its result
//     carries the band over the whole 270° as two faces and recedes both wall faces.
//
// Its far cap RECEDES around the runout trim (a convex arm bites material away).
func n4LateralTorusArm(ef edgeFillet, near railID) cornerArmSpec {
	return cornerArmSpec{
		role: "torus", ef: ef, surface: ef.armSurface,
		nearKind: armPassesLaterally, near: []railID{near},
		far: farRimContinuation, sense: biteInward,
	}
}

// n4FilletedEdges is the picked-edge id set at this corner — armFarRunout's fillet-fillet interference guard.
func n4FilletedEdges(roles n4MixedArms) map[uint64]bool {
	return map[uint64]bool{
		roles.ccyl.edge.ID(): true, roles.torus.edge.ID(): true, roles.band.edge.ID(): true,
	}
}

// n4CornerVertex returns the shared trihedral vertex point the three arm edges meet at.
func n4CornerVertex(roles n4MixedArms, tol float64) math.Point3 {
	e := roles.ccyl.edge
	for _, p := range [2]math.Point3{e.StartVertex().Point(), e.EndVertex().Point()} {
		if edgeHasEndpoint(roles.torus.edge, p, tol) && edgeHasEndpoint(roles.band.edge, p, tol) {
			return p
		}
	}
	return e.StartVertex().Point()
}
