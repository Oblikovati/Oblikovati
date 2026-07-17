// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The per-arm-centre canal arm faces (M6' C4 W2, architecture: canal-armweld-architecture.md
// §"The per-arm-center arm faces"). The reframe: the canal corner is THREE independent single-ball
// sub-problems glued by the canal patch instead of a shared sphere. Each arm rolls on its OWN reflected
// centre (reflectedArmCentres, ADR-C4-3), so the hardened single-ball arm machinery (solveArmSetback /
// armHostContactRail / farCrossSectionArc) works VERBATIM once fed a one-arm LOCAL cornerWeld at that
// centre. The one genuinely new piece is the arm's corner-side rail: NOT a setback great-arc to a sphere
// but the SHARED canal boundary curve (an END ARC for the two wall-sharing arms, the u=1 FOOT-LOCUS for
// the mid arm), sampled the SAME way the corner patch samples it (ADR-C4-2) so the two faces weld
// point-for-point watertight.

// canalArmFaces builds the three per-arm-centre canal arm faces. The arm→corner-rail mapping (from the
// architecture): the two WALL-sharing arms each close on the END ARC whose centre matches that arm's
// reflected centre by PROXIMITY (NOT index order — the boundary ring order [v0,u1,v1,u0] is not tied to
// arm order); the MID (non-wall) arm closes on the u=1 foot-locus (feet[1], on the s_10 host). It
// consumes the ALREADY-resolved boundaries + centres (single source, ADR-C4-2/C4-3), never recomputing
// them. Declines (non-empty reason, carrying the arm index + surface + centre) if any arm cannot build.
func canalArmFaces(arms []edgeFillet, w cornerWeld, b canalBoundaries, centres []math.Point3, scale float64, res Resolution) ([]filletFace, string) {
	wallFace, _, ok := tangentCornerWall(arms)
	if !ok {
		return nil, fmt.Sprintf("canal arm faces: want exactly one cylinder wall + two plane hosts across %d arms", len(arms))
	}
	wa, ok := wallSharingArms(arms, wallFace)
	if !ok {
		return nil, "canal arm faces: want exactly two wall-sharing arms (the canal end-arc neighbours)"
	}
	mid := nonWallArmIndex(arms, wa[0], wa[1])
	if mid < 0 {
		return nil, "canal arm faces: no mid (non-wall) arm for the u=1 foot-locus rail"
	}
	faces := make([]filletFace, 0, len(arms))
	for i := range arms {
		face, reason := canalOneArmFace(arms[i], centres[i], b, i, mid, w, scale, res)
		if reason != "" {
			return nil, reason
		}
		faces = append(faces, face)
	}
	return faces, ""
}

// canalOneArmFace selects arm i's corner-side rail and builds its face, naming WHY it declined.
func canalOneArmFace(arm edgeFillet, centre math.Point3, b canalBoundaries, i, mid int, w cornerWeld, scale float64, res Resolution) (filletFace, string) {
	rail, rev, ok := canalArmCornerRail(b, centre, i, mid)
	if !ok {
		return filletFace{}, fmt.Sprintf("canal arm %d: no corner-side rail (end arc / foot-locus) at centre %v", i, centre)
	}
	face, ok := canalArmFace(arm, centre, rail, rev, w, scale, res)
	if !ok {
		return filletFace{}, fmt.Sprintf("canal arm %d (%T) declined at reflected centre %v", i, arm.armSurface, centre)
	}
	return face, ""
}

// canalArmCornerRail picks arm i's corner-side rail + its patch-sampling direction: the mid arm takes
// the u=1 foot-locus (feet[1]); each wall-sharing arm takes the END ARC nearest its reflected centre.
func canalArmCornerRail(b canalBoundaries, centre math.Point3, i, mid int) (geom.Curve3, bool, bool) {
	if i == mid {
		return b.feet[1], b.feetRev[1], true
	}
	return endArcForCentre(b, centre)
}

// endArcForCentre returns the END ARC whose fitted circle centre is nearest centre (with its sampling
// direction) — the by-proximity match that ties each wall-sharing arm to the cross-section arc about
// its OWN reflected ball centre (arm @ C → the end arc centred at C), robust to the boundary ring order.
func endArcForCentre(b canalBoundaries, centre math.Point3) (geom.Curve3, bool, bool) {
	c0, ok0 := endArcCentre(b.endArcs[0])
	c1, ok1 := endArcCentre(b.endArcs[1])
	if !ok0 || !ok1 {
		return nil, false, false
	}
	if float64(centre.DistanceTo(c0)) <= float64(centre.DistanceTo(c1)) {
		return b.endArcs[0], b.endArcsRev[0], true
	}
	return b.endArcs[1], b.endArcsRev[1], true
}

// endArcCentre fits the circle through an end-arc isocurve's two ends + midpoint and returns its centre;
// the canal v-boundaries are exact radius-r cross-section arcs, so the fit is well posed. A foot-locus
// (mis-tagged into endArcs) is not circular and would not fit — but canalBoundaryRoles tags by
// iso-direction, so this only ever sees the two genuine end arcs.
func endArcCentre(c geom.Curve3) (math.Point3, bool) {
	lo, hi := c.Domain()
	arc, err := geom.Arc3dByThreePoints(c.PointAt(lo), c.PointAt((lo+hi)/2), c.PointAt(hi))
	if err != nil {
		return math.Point3{}, false
	}
	return arc.Center, true
}

// canalArmFace builds ONE canal arm face at its reflected centre (ADR-C4-3): a one-arm LOCAL cornerWeld
// (wi.center = the reflected centre) feeds the REUSED single-ball rail helpers so the arm rolls on its
// own centre — torusArmStation(wi)/endpointOf(wi.center,…) inside armHostContactRail resolve there — and
// the corner-side rail is the shared canal boundary curve. The four-sided loop is
// [cornerRail, hFar, far, hNear] chained head-to-tail. Declines (ok=false) if any sub-step declines
// (a gap at this centre, a host rail that cannot be built, or a loop that will not close). Example:
//
//	f, ok := canalArmFace(arm, centre, endArc, rev, w, scale, res); if !ok { /* do-no-harm floor */ }
func canalArmFace(arm edgeFillet, centre math.Point3, cornerRail geom.Curve3, cornerRev bool, w cornerWeld, scale float64, res Resolution) (filletFace, bool) {
	set, ok := solveArmSetback(arm, centre, w.radius, scale, res)
	if !ok {
		return filletFace{}, false // C not on this arm's spine at its reflected centre — a gap
	}
	wi := cornerWeld{center: centre, radius: w.radius, arms: []armSetback{set}}
	h0, h1, far, ok := canalArmRailsAndTerminal(arm, set, wi, res)
	if !ok {
		return filletFace{}, false // F_far section or reused rail declined at this reflected centre
	}
	loop, reason := canalArmLoop(h0, h1, far, cornerRail, cornerRev, res.Weld()*scale)
	if reason != "" {
		return filletFace{}, false // canalArmLoop already carries the precise junction+gap; see canalOneArmFace
	}
	return filletFace{surface: arm.armSurface, loops: []filletLoop{loop}, parent: armFaceProvenance(arm.edge)}, true
}

// canalArmHostRails builds the arm's two host contact rails at its LOCAL weld wi, both oriented
// outer→tHost. tHost is the host-tangent point endpointOf(wi.center, r, railDir) on each of the arm's
// two hosts — the reflected-centre analogue of armRailBundle's t0/t1.
func canalArmHostRails(arm edgeFillet, set armSetback, wi cornerWeld, res Resolution) (endSeg, endSeg, bool) {
	t0 := endpointOf(wi.center, wi.radius, set.railDir0)
	t1 := endpointOf(wi.center, wi.radius, set.railDir1)
	h0, ok0 := armHostContactRail(arm.a, set, t0, wi, res)
	h1, ok1 := armHostContactRail(arm.b, set, t1, wi, res)
	return h0, h1, ok0 && ok1
}

// canalArmLoop assembles the arm face's closed 4-sided loop. The corner-side rail is sampled the SAME
// way canalPatchLoops samples it (sampleCurve3Open with the patch's own rev) — the point-for-point
// shared-edge identity (ADR-C4-2) that welds the arm face to the corner patch without a crack — and put
// FIRST, so it is loop.pts[:len(corner)]. The remaining three sides (the far host rail, the far
// cross-section arc, the near host rail) are oriented head-to-tail from the corner rail's far end back
// to its start. Declines (non-empty reason, carrying the junction + measured gap) if the host rails do
// not chain onto the corner rail's two ends — checked at EVERY junction (tb→hFar, hFar.to→far,
// far.to→hNear, hNear.to→corner[0]), not just the final one: a break at an EARLIER junction (e.g.
// Mutation C: orientEndSeg reversed into a 75-unit gap) used to survive this gate silently and only get
// caught downstream, in assertArmFaceCloses or assembleBody, with a far less precise reason (W2 review).
func canalArmLoop(h0, h1, far endSeg, cornerRail geom.Curve3, cornerRev bool, tol float64) (filletLoop, string) {
	corner := sampleCurve3Open(cornerRail, cornerRev)
	if len(corner) == 0 {
		return filletLoop{}, "canal arm loop: corner rail sampled to zero points"
	}
	hNear, hFar, tb, ok := orderArmHostRails(h0, h1, corner[0], tol)
	if !ok {
		return filletLoop{}, fmt.Sprintf("canal arm loop: neither host rail ends at the corner rail's start %v (tol %.3e)", corner[0], tol)
	}
	hf, reason := chainOnto("corner rail's far end→far host rail", hFar, tb, tol) // tb → outerFar
	if reason != "" {
		return filletLoop{}, reason
	}
	fr, reason := chainOnto("far host rail→far cross-section arc", far, hf.to, tol) // outerFar → outerNear
	if reason != "" {
		return filletLoop{}, reason
	}
	hn, reason := chainOnto("far cross-section arc→near host rail", hNear, fr.to, tol) // outerNear → corner[0]
	if reason != "" {
		return filletLoop{}, reason
	}
	if gap, ok := chainsWithin(hn.to, corner[0], tol); !ok {
		return filletLoop{}, fmt.Sprintf("canal arm loop: ring does not close — near host rail's end is %.3e from the corner rail's start %v (tol %.3e)", gap, corner[0], tol)
	}
	return buildArmFilletLoop(corner, cornerRail, hf, fr, hn), ""
}

// chainOnto orients seg to start at target (via orientEndSeg) and turns a failed orientation into a
// reason string naming the junction label and the measured gap, so canalArmLoop's decline points at the
// SPECIFIC broken junction instead of a generic "did not close" at the end of the loop.
func chainOnto(junction string, seg endSeg, target math.Point3, tol float64) (endSeg, string) {
	oriented, gap, ok := orientEndSeg(seg, target, tol)
	if ok {
		return oriented, ""
	}
	return endSeg{}, fmt.Sprintf("canal arm loop: junction %s does not chain at %v (gap %.3e > tol %.3e)", junction, target, gap, tol)
}

// buildArmFilletLoop flattens the sampled corner rail + the three chained boundary segments into one
// filletLoop: each corner sample carries cornerRail as its leaving curve; each trailing segment carries
// its own oriented curve. loop.pts[:len(corner)] is exactly the shared corner-rail sampling.
func buildArmFilletLoop(corner []math.Point3, cornerRail geom.Curve3, segs ...endSeg) filletLoop {
	fl := filletLoop{}
	for _, p := range corner {
		fl.pts = append(fl.pts, p)
		fl.curves = append(fl.curves, cornerRail)
	}
	for _, s := range segs {
		fl.pts = append(fl.pts, s.from)
		fl.curves = append(fl.curves, s.curve)
	}
	return fl
}

// orderArmHostRails splits the two host rails into hNear (the one whose tHost is the corner rail's start
// ta) and hFar (the other), returning hFar's tHost tb. Both rails are oriented outer→tHost, so .to is
// the tHost. Declines if neither rail ends at ta (a mis-matched corner rail).
func orderArmHostRails(h0, h1 endSeg, ta math.Point3, tol float64) (endSeg, endSeg, math.Point3, bool) {
	if float64(h0.to.DistanceTo(ta)) <= tol {
		return h0, h1, h1.to, true
	}
	if float64(h1.to.DistanceTo(ta)) <= tol {
		return h1, h0, h0.to, true
	}
	return endSeg{}, endSeg{}, math.Point3{}, false
}

// orientEndSeg returns s (or its reverse) oriented to start at p, so a mixed chain of rails built in
// different directions can be walked head-to-tail. Rejects (ok=false, with the smaller of the two
// measured gaps) when NEITHER orientation starts at p within tol — previously this reversed blindly on
// any mismatch, so a segment that could not actually be made to chain (Mutation C: e.g. a broken
// reversal) was returned anyway and the gap only surfaced downstream. Example:
//
//	oriented, gap, ok := orientEndSeg(hostRail, farHostPoint, res.Weld()*scale)
//	if !ok { /* honest-decline: report gap */ }
func orientEndSeg(s endSeg, p math.Point3, tol float64) (endSeg, float64, bool) {
	if gap, ok := chainsWithin(s.from, p, tol); ok {
		return s, gap, true
	}
	rev := reverseEndSegs([]endSeg{s})[0]
	gap, ok := chainsWithin(rev.from, p, tol)
	return rev, gap, ok
}

// chainsWithin reports whether a and b are within tol of each other — a shared-endpoint chaining
// junction — returning the measured gap either way so a failing caller can report it precisely instead
// of a bare "did not close".
func chainsWithin(a, b math.Point3, tol float64) (float64, bool) {
	gap := float64(a.DistanceTo(b))
	return gap, gap <= tol
}

// armFaceProvenance is the arm face's ADR-0043 lineage: the filleted edge's name when a real edge was
// wired (the whole-body path), else the zero lineage (bare-face corner fixtures carry no edge).
func armFaceProvenance(e *topo.Edge) topo.Lineage {
	if e == nil {
		return topo.Lineage{}
	}
	return filletEdgeProvenance(e)
}
