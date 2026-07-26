// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

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
// isoparms (the ball's contact loci on the lateral arm and on the mid host), each a CHAIN so a class may
// register them subdivided (see cornerCanalRailRing's pieces argument).
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
// ★ `pieces` is how many sub-segs each RAIL is registered as (1 = one seg end to end). It exists because
// the shell-orientation pass reverses a flipped face's loop by RE-FITTING each segment through three points
// (reverseSegmentCurve, fillet_orient.go), which is exact for an arc and WRONG for a canal isoparm: it
// replaces the exact contact locus with a circle. Measured on the O1 corner, whose patch and lateral-arm
// faces are both flipped: a single-seg rail lands 0.092 and 0.292 off its own surface at r = 5, and the
// NURBS mesher then tiles slivers against a boundary that is not on the patch and FOLDS. Subdividing makes
// the three-point fit's error fall as h³ — it is the same cure canalPatchLoops applies to the N7 canal
// patch, and N7's boundary sub-edges measure ≤1e-12 off their surface where O1's single segs measured 0.29.
// A class that keeps pieces = 1 is byte-identical to the pre-subdivision ledger (N4 does, so its pin holds).
// Example:
//
//	ring := cornerCanalRailRing(led, "o1", corner.pts, corner.railBC, corner.railDA, o1CanalRailPieces)
//	plan.patch = cornerPatchSpec{surface: corner.patch.Surface, sides: ring.sides}
func cornerCanalRailRing(led *cornerWeldLedger, class string, p cornerCanalPts, railBC, railDA geom.Curve3, pieces int) cornerPatchRailRing {
	ab := led.add(class+"/arcAB", endSeg{from: p.a, to: p.b, curve: p.arcAB, mid: p.arcAB.PointAt(0.5), arc: true})
	bc := addRailChain(led, class+"/railBC", railBC, pieces)
	cd := led.add(class+"/arcCD", endSeg{from: p.c, to: p.d, curve: p.arcCD, mid: p.arcCD.PointAt(0.5), arc: true})
	da := addRailChain(led, class+"/railDA", railDA, pieces)
	sides := append([]railID{ab}, bc...)
	sides = append(append(sides, cd), da...)
	return cornerPatchRailRing{arcAB: ab, railBC: bc, arcCD: cd, railDA: da, sides: sides}
}

// addRailChain registers one on-host rail as `pieces` consecutive sub-segs, each carrying the rail
// RESTRICTED to its own sub-span (a geom.TrimmedCurve3) so both faces sharing it read one curve object per
// piece. pieces <= 1 registers the whole rail as one seg, which is byte-identical to the pre-subdivision
// ledger.
func addRailChain(led *cornerWeldLedger, name string, rail geom.Curve3, pieces int) []railID {
	if pieces <= 1 {
		return []railID{led.add(name, endSeg{from: curveStart(rail), to: curveEnd(rail), curve: rail, mid: curveMidPoint(rail)})}
	}
	lo, hi := rail.Domain()
	out := make([]railID, 0, pieces)
	at := func(k int) float64 { return lo + (hi-lo)*float64(k)/float64(pieces) }
	for k := 0; k < pieces; k++ {
		t0, t1 := at(k), at(k+1)
		sub := geom.TrimmedCurve3{Base: rail, Lo: t0, Hi: t1}
		out = append(out, led.add(fmt.Sprintf("%s[%d]", name, k),
			endSeg{from: rail.PointAt(t0), to: rail.PointAt(t1), curve: sub, mid: rail.PointAt((t0 + t1) / 2)}))
	}
	return out
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
