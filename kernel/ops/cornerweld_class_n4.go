// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
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
	vertex, ok := trihedralArmVertex(roles.ccyl, roles.torus, roles.band, res.Weld()*r)
	if !ok {
		return cornerWeldPlan{}, true, "n4 corner: the three arm edges do not meet at one vertex"
	}
	return n4Plan(roles, corner, vFace, vertex, r), true, ""
}

// n4Plan assembles the declarative plan: one ledger holding the four patch-ring rails (registered by the
// shared cornerCanalRailRing, so the patch and its three welded neighbours read the identical curve objects),
// three arm specs, and the one host mid rail. Every line below is a role DECLARATION — there is no N4-specific
// assembly left in it.
func n4Plan(roles n4MixedArms, corner n4Corner, vFace *topo.Face, vertex math.Point3, r float64) cornerWeldPlan {
	led := newCornerWeldLedger()
	ring := cornerCanalRailRing(led, "n4", corner.pts, corner.railBC, corner.railDA)
	return cornerWeldPlan{
		ledger: led,
		patch:  cornerPatchSpec{surface: corner.patch.Surface, sides: ring.sides},
		arms: []cornerArmSpec{
			// The two CONCAVE cylinder arms terminate at the corner and their caps GROW outward to the
			// contact rail (a reentrant fillet ADDS material).
			concaveTerminatingArmSpec("ccyl", roles.ccyl, corner.pts.arcCD, ring.arcCD),
			concaveTerminatingArmSpec("band", roles.band, corner.pts.arcAB, ring.arcAB),
			n4LateralTorusArm(roles.torus, ring.railBC),
		},
		mids:     []cornerHostMid{{face: vFace, rails: ring.railDA}},
		vertex:   vertex,
		radius:   r,
		filleted: filletedArmEdgeIDs(roles.ccyl, roles.torus, roles.band),
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
func n4LateralTorusArm(ef edgeFillet, near []railID) cornerArmSpec {
	return lateralArmSpec("torus", ef, near, farRimContinuation, biteInward)
}
