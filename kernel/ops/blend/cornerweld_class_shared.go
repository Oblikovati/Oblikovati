// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The vocabulary every ROLLING-BALL CANAL corner class builds its plan out of — shared by the N4 class
// (cornerweld_class_n4.go) and the O1 class (cornerweld_class_o1.go), and by any later class of the same
// shape (H7). It exists because slice 2 measured the prediction in corner-weld-layer-design.md ADR-1: a new
// corner class should be one ~120-line CONFIGURATION, not a new weld. Everything in this file is the part of
// that configuration that is NOT per-case — the patch-ring registration, the two arm-spec shapes, the
// trihedral vertex, the interference edge set — so a class builder declares only what actually differs:
// which arm plays which role, each arm's far termination and retrim sense, and which host carries the mid
// rail.
//
// The shape all these classes share (design Axes A+C):
//
//	two arms TERMINATE at the corner on radius-r cross-section arcs (A1),
//	one arm runs PAST it (A2) and the patch rails on that arm's own surface,
//	the patch's fourth side rails on a shared HOST (A3 — the `mid`),
//	each arm's far cap grows (concave) or recedes (convex) per its own sense.

// cornerPatchRailRing is a canal patch's four registered boundary handles plus the ordered ring the patch
// reads. The two arcs are the terminating arms' cross-sections; the two rails are the canal's own boundary
// isoparms (the ball's contact loci on the lateral arm and on the mid host), each a one-element CHAIN
// because the arm/mid specs take []railID (a rail is free to become a chain when a class needs one).
type cornerPatchRailRing struct {
	arcAB, arcCD   railID
	railBC, railDA []railID
	sides          []railID
}

// cornerCanalRailRing registers a canal corner's four patch sides in the ledger — arc A→B (the first
// terminating arm's cross-section), rail B→C (the ball's contact locus on the LATERAL arm), arc C→D (the
// second terminating arm's cross-section), rail D→A (the contact locus on the MID host) — each as SINGLE
// curve-segs, never a sampled polyline (a sampled side would crack against the arm faces' single-seg
// curves). The two rails are boundary isoparms of a lofted canal, whose chord-length knots make Domain()
// the only safe source for the mid witness, hence curveMidPoint.
//
// Registering the ring ONCE here is what makes the patch and its three welded neighbours read the identical
// curve objects (design Axis F), for every class rather than once per class.
//
// Example:
//
//	ring := cornerCanalRailRing(led, "o1", corner.pts, corner.railBC, corner.railDA)
//	plan.patch = cornerPatchSpec{surface: corner.patch.Surface, sides: ring.sides}
func cornerCanalRailRing(led *cornerWeldLedger, class string, p cornerCanalPts, railBC, railDA geom.Curve3) cornerPatchRailRing {
	ab := led.add(class+"/arcAB", endSeg{from: p.a, to: p.b, curve: p.arcAB, mid: p.arcAB.PointAt(0.5), arc: true})
	bc := addOnHostRail(led, class+"/railBC", railBC)
	cd := led.add(class+"/arcCD", endSeg{from: p.c, to: p.d, curve: p.arcCD, mid: p.arcCD.PointAt(0.5), arc: true})
	da := addOnHostRail(led, class+"/railDA", railDA)
	sides := append([]railID{ab}, bc...)
	sides = append(append(sides, cd), da...)
	return cornerPatchRailRing{arcAB: ab, railBC: bc, arcCD: cd, railDA: da, sides: sides}
}

// addOnHostRail registers one on-host canal rail as a SINGLE curve-seg, returned as the one-element chain
// the arm/mid specs take. It is one seg because the shell-orientation pass now reverses a flipped face's
// non-circular loop segment by RE-PARAMETERISING it (reverseCurveExactly, fillet_orient.go) instead of
// re-fitting it through three points: O1 used to register these rails as 8 sub-segs so the re-fit's h³
// error stayed under the weld tolerance (0.092/0.292 off its own surface at one seg → 6.2e-4 at eight), and
// with the re-fit gone that subdivision is dead — a whole rail now measures ≤1.5e-8 off its surface after
// reversal, i.e. its own construction residual.
func addOnHostRail(led *cornerWeldLedger, name string, rail geom.Curve3) []railID {
	return []railID{led.add(name, endSeg{from: curveStart(rail), to: curveEnd(rail), curve: rail, mid: curveMidPoint(rail)})}
}

// concaveTerminatingArmSpec declares a CONCAVE arm that ENDS at the corner on its radius-r cross-section arc
// (A1). Both of its remaining role choices are fixed by that description rather than passed in, and the name
// says which:
//   - far = farCappedVertex, because an arm that terminates at the corner reaches its other end at a vertex
//     carrying a transverse cap; the rim-CONTINUATION variant (B3) is by construction a property of an arm
//     that runs PAST a corner, which is lateralArmSpec — where far IS a parameter;
//   - sense = growOutward, because a reentrant fillet ADDS material, so its far cap grows out to the contact
//     rail rather than receding around it.
//
// Every class built on the layer so far (N4, O1) has exactly two such arms. A class with a CONVEX terminating
// arm adds a sibling (or re-introduces the parameter) together with the case that exercises it — declaring an
// unused role choice here would be an untested branch.
func concaveTerminatingArmSpec(role string, ef edgeFillet, arc geom.Arc3d, near railID) cornerArmSpec {
	return cornerArmSpec{
		role: role, ef: ef, surface: ef.armSurface,
		nearKind: armTerminatesAtArc, nearArc: arc, near: []railID{near},
		far: farCappedVertex, sense: growOutward,
	}
}

// lateralArmSpec declares the arm that runs PAST the corner (A2): its near boundary is the general on-arm
// contact rail the patch shares, not a cross-section arc.
func lateralArmSpec(role string, ef edgeFillet, near []railID, far farTermKind, sense retrimSense) cornerArmSpec {
	return cornerArmSpec{
		role: role, ef: ef, surface: ef.armSurface,
		nearKind: armPassesLaterally, near: near,
		far: far, sense: sense,
	}
}

// trihedralArmVertex returns the point where all three arm edges meet — the corner site. It reads the
// candidates off `anchor` and requires BOTH other edges to carry the same endpoint, so a corner whose arms
// do not actually share a vertex cannot be silently welded at one arm's end: ok=false instead.
func trihedralArmVertex(anchor, x, y edgeFillet, tol float64) (math.Point3, bool) {
	e := anchor.edge
	for _, p := range [2]math.Point3{e.StartVertex().Point(), e.EndVertex().Point()} {
		if edgeHasEndpoint(x.edge, p, tol) && edgeHasEndpoint(y.edge, p, tol) {
			return p, true
		}
	}
	return math.Point3{}, false
}

// filletedArmEdgeIDs is the picked-edge id set at one corner site — armFarRunout's fillet-fillet
// interference guard, which must see every pick at the corner so a run-out cannot terminate ON another
// fillet's edge.
func filletedArmEdgeIDs(arms ...edgeFillet) map[uint64]bool {
	out := make(map[uint64]bool, len(arms))
	for _, ef := range arms {
		out[ef.edge.ID()] = true
	}
	return out
}
