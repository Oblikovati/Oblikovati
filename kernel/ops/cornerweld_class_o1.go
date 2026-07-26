// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The O1-class corner PLAN BUILDER — slice 2 of the general corner-weld layer, and the FALSIFICATION TEST of
// corner-weld-layer-design.md ADR-1: a second class built on the layer must be a CONFIGURATION, not a second
// weld. It is exactly that. Nothing below emits a face, splices a loop, retrims a host, terminates an arm or
// touches the executor; every line names a role, a far termination, a retrim sense, a mid rail or a rail
// handle. The geometry the configuration consumes is solved outside the layer (fillet_curved_mixed_o1.go +
// fillet_curved_mixed_canal_o1.go), exactly where the design puts corner SOLVING.
//
// The class (fillet_curved_mixed_o1.go documents the geometry): two CONCAVE arms — a cylinder arm and a cove
// torus arm — terminate at the corner on radius-r cross-section arcs (A1) and their far caps GROW outward
// (a reentrant fillet ADDS material); one CONVEX planar band runs PAST the corner (A2), so the patch rails on
// the band's own cylinder, and its far cap RECEDES (a convex fillet bites material away); the patch's fourth
// side rails on the shared MID host, which for this class is the boss CYLINDER (A3).
//
// DRAWEXE 8.0.0 target, re-verified live (`restore data/CFI_f5678fin.rle s ; tscale s 0 0 0 10 ; explode s e ;
// blend result s 5 s_7 5 s_6 5 s_14 ; nbshapes ; sprops ; vprops ; checkshape`): a valid SOLID, 1 shell,
// 12 faces / 13 wires / 18 vertices / 27 edges, area 65104.9, volume 1.11166e6. Twelve faces is the input's
// eight plus three arm bands and the corner patch — no rim continuation and no host-seam split here, so the
// layer's B3/C4 variants stay dormant and this case exercises the plain single-link path.

// o1CanalRailPieces is how many sub-segs O1 registers each on-host canal rail as. It is > 1 because the
// shell-orientation pass re-fits a FLIPPED face's loop segments through three points, which is exact for an
// arc and replaces a canal isoparm with a circle — and O1's patch AND lateral-arm faces are both flipped
// (N4's patch is not). Measured on this corner: a single seg lands the two rails 0.092 and 0.292 off their
// own surface at r = 5 and the NURBS mesher folds on four edges; at 8 pieces the worst is 6.2e-4 (0.012% of
// r, 3 orders below the weld tolerance) and the body is fold-free. See cornerCanalRailRing's `pieces` note —
// this is the cure canalPatchLoops already applies to the N7 canal patch, whose sub-edges measure ≤1e-12.
const o1CanalRailPieces = 8

// o1CornerPlanBuilder recognises the O1 class and declares its plan. Nothing is mutated before took=true, so
// a non-O1 corner falls through the ladder with no trace (layer invariant #5).
type o1CornerPlanBuilder struct{}

// Plan declares the O1 corner site. took=false leaves the corner to the untouched legacy sphere path.
func (o1CornerPlanBuilder) Plan(_ *topo.Body, arms []edgeFillet, res Resolution) (cornerWeldPlan, bool, string) {
	roles, ok := classifyO1MixedArms(arms)
	if !ok {
		return cornerWeldPlan{}, false, ""
	}
	r, ok := armTubeRadius(roles.lateral.armSurface)
	if !ok {
		return cornerWeldPlan{}, false, ""
	}
	corner, ok := solveO1Corner(roles, r, res)
	if !ok {
		return cornerWeldPlan{}, false, "" // not the O1 cylinder-on-cylinder canal corner — keep the prior path
	}
	vertex, ok := trihedralArmVertex(roles.ccyl, roles.cove, roles.lateral, res.Weld()*r)
	if !ok {
		return cornerWeldPlan{}, true, "o1 corner: the three arm edges do not meet at one vertex"
	}
	return o1Plan(roles, corner, vertex, r), true, ""
}

// o1Plan is the whole per-case cost of this class: one ledger holding the four patch-ring rails, three arm
// role declarations, and the one host mid rail. Compare n4Plan — the two differ only in which arm fills which
// role, which far termination the lateral arm takes, and whether the mid host is a plane or a cylinder.
func o1Plan(roles o1MixedArms, corner o1Corner, vertex math.Point3, r float64) cornerWeldPlan {
	led := newCornerWeldLedger()
	ring := cornerCanalRailRing(led, "o1", corner.pts, corner.railBC, corner.railDA, o1CanalRailPieces)
	return cornerWeldPlan{
		ledger: led,
		patch:  cornerPatchSpec{surface: corner.patch.Surface, sides: ring.sides},
		arms: []cornerArmSpec{
			// Both concave arms terminate at the corner; each runs out onto a plane cap at its far vertex and
			// that cap GROWS around the runout trim.
			concaveTerminatingArmSpec("ccyl", roles.ccyl, corner.pts.arcAB, ring.arcAB),
			concaveTerminatingArmSpec("cove", roles.cove, corner.pts.arcCD, ring.arcCD),
			// The convex band runs PAST the corner on rail B→C, ends on an ordinary transverse cap, and that
			// cap RECEDES around the trim.
			lateralArmSpec("band", roles.lateral, ring.railBC, farCappedVertex, biteInward),
		},
		mids:     []cornerHostMid{{face: corner.midFace, rails: ring.railDA}},
		vertex:   vertex,
		radius:   r,
		filleted: filletedArmEdgeIDs(roles.ccyl, roles.cove, roles.lateral),
	}
}
