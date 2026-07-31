// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Chain-capable host retrims for the curved-miter weld and the single-arm runout — the W-C wave's
// "host retrim variants" capability (OCCT blend/simple P4, P5, O8; blend-parity corpus).
//
// The prior host retrim spliced exactly ONE bite segment whose feet both lie on the host's boundary
// ring. Three real geometries break that shape (all DRAWEXE-grounded on the two-offset-cylinder
// fixture CFI_i1234fis, r5):
//   - a rail foot lands MID-FACE at a capped end (P4's wall: the fillet band stops on the pocket
//     floor plane; the wall keeps its material below, so a fresh latitude edge — wall ∩ floor plane
//     extended past the old seam vertex — must bridge the foot back to the ring);
//   - the miter seam's sBot lies on the OTHER arm's outer host, so this arm's rail cannot end at
//     sBot; it ends where the arm tube meets that host's plane, and the tube∩plane TRANSIT arc
//     carries the boundary from the rail's end to sBot (P5's cylinder arm);
//   - one host is bitten TWICE (P5's pocket wall: the cylinder arm's rail AND the torus arm's far
//     trim), so bites accumulate per host and splice sequentially.
// All splices go through the shipped chain primitive chainRetrimLoop (fillet_retrim_chain.go),
// which owns landing clipping and the smaller-developed-area span pick — extended here, not forked.

// miterHostChain is one connected bite chain on a bitten host: the ordered segments (each seg's `to`
// is the next seg's `from`; both extremes land on the host's ring), plus `consumed` — a vertex the
// splice must REMOVE from the ring when hasConsumed (the do-no-harm guard against a wrong-span pick).
type miterHostChain struct {
	chain       []endSeg
	consumed    math.Point3
	hasConsumed bool
}

// chainedHostRetrim re-clips one bitten host against its accumulated bite chains, splicing them
// sequentially through chainRetrimLoop. The bitten loop is the one carrying the first chain's ring
// landing; every other loop is carried through unchanged with the outer loop kept at index 0
// (assembleBody keys outer-ness on that). Declines honestly when any splice fails or a consumed-
// vertex guard is violated — never a partial or wrong-span solid.
func chainedHostRetrim(host *topo.Face, chains []miterHostChain, tol float64) (filletFace, bool) {
	if len(chains) == 0 {
		return filletFace{}, false
	}
	bitten := chainBittenLoop(host, chains[0], tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	ring, ok := splicedChainRing(host.Geometry(), segsFromLoop(bitten), chains, tol)
	if !ok || ringHasRepeatedPoint(ring, tol) {
		return filletFace{}, false
	}
	loops := hostLoopsWithRetrim(host, bitten, outer, loopFromSegs(ring))
	return filletFace{surface: host.Geometry(), loops: loops, parent: host.Lineage()}, true
}

// ringHasRepeatedPoint reports whether the spliced ring revisits an earlier vertex — the signature of
// splicing across a self-touching or degenerate (zero-length) segment already present in the host's
// OWN boundary before any retrim runs. simple/P4 and P5's raw STEP-imported base solid carries exactly
// this defect: the wall's own loop uses one pre-existing edge TWICE (once each direction) bracketing a
// truly zero-length edge at (100,50,0) — present verbatim in the un-filleted body, so it predates and
// is outside the fillet capability. A clean splice never revisits a ring vertex; a repeat means the
// chain landed on or crossed untrustworthy material and the tessellator silently drops the pinched-off
// lobe (measured on P4/P5: the wall face comes out ~10.6% short and 10-28 free edges appear in the weld).
// Declining here is the do-no-harm floor: never ship a solid whose face has a self-touching slit.
func ringHasRepeatedPoint(ring []endSeg, tol float64) bool {
	pts := make([]math.Point3, len(ring))
	for i, s := range ring {
		pts[i] = s.from
	}
	return pointsHaveDuplicate(pts, tol)
}

// pointsHaveDuplicate reports whether any two points in pts coincide within tol — the shared core
// ringHasRepeatedPoint (chain-splice rings) and sharedRetrimIsSound (the shared-face two-rail splice,
// fillet_miter_curved_weld.go) both drive, so the ONE degenerate-boundary detector serves every miter
// retrim result rather than two near-duplicate loops.
func pointsHaveDuplicate(pts []math.Point3, tol float64) bool {
	for i, p := range pts {
		for _, q := range pts[i+1:] {
			if p.DistanceTo(q) <= tol {
				return true
			}
		}
	}
	return false
}

// splicedChainRing applies each bite chain to the ring in turn, enforcing every chain's consumed-
// vertex guard: after all splices, a vertex a chain marked consumed must be GONE from the ring —
// the smaller-area pick removed the intended span, not its complement.
func splicedChainRing(surf geom.Surface, ring []endSeg, chains []miterHostChain, tol float64) ([]endSeg, bool) {
	for _, c := range chains {
		// Pre-split the ring at the chain's landings: a two-arc "lens" ring (intersecting-cylinder
		// caps, P5/O8) is a legitimate bitten wire but chainRetrimLoop guards on len(ring) ≥ 3;
		// splitting at interior landings makes the wire explicit without touching the primitive.
		ring = insertSplits(ring, []math.Point3{c.chain[0].from, c.chain[len(c.chain)-1].to}, tol)
		next, ok := chainRetrimLoop(surf, ring, c.chain, tol)
		if !ok {
			return nil, false
		}
		ring = next
	}
	for _, c := range chains {
		if c.hasConsumed && pathHasVertex(ring, c.consumed, tol) {
			return nil, false // the splice kept the span it was meant to remove — do-no-harm
		}
	}
	return ring, true
}

// chainBittenLoop selects the loop the chain actually bites: the loop carrying the chain's first
// ring landing (either extreme). Falls back to the outer loop when neither extreme is a loop point
// yet (both feet bridged) — the retrim then fails honestly downstream if that was wrong.
func chainBittenLoop(host *topo.Face, c miterHostChain, tol float64) *topo.Loop {
	head, tail := c.chain[0].from, c.chain[len(c.chain)-1].to
	for _, l := range host.Loops() {
		segs := segsFromLoop(l)
		if pointOnRing(segs, head, tol) || pointOnRing(segs, tail, tol) {
			return l
		}
	}
	return outerHostLoop(host)
}

// miterChainEndBridgeSeg is the fresh edge that carries a mid-face rail/trim foot back to the boundary
// at a capped arm end: the sub-curve of wall ∩ (cap plane through both points) from the picked
// edge's end vertex vEnd to the foot. On a CYLINDER wall it is the capBridgeArc latitude arc (the
// same constructor the shared-face retrim uses, so the wall-side and cap-side splices carry the
// SAME curve and weld watertight); on a PLANE wall it is the straight segment. ok=false for any
// other wall geometry — the do-no-harm floor.
func miterChainEndBridgeSeg(wall geom.Surface, foot, vEnd math.Point3, tol float64) (endSeg, bool) {
	switch wall.(type) {
	case geom.Cylinder:
		seg, ok := capBridgeArc(wall, foot, vEnd, tol)
		if !ok {
			return endSeg{}, false
		}
		return reverseEndSegs([]endSeg{seg})[0], true // capBridgeArc runs foot→vEnd; the chain needs vEnd→foot
	case geom.Plane:
		return endSeg{from: vEnd, to: foot}, true
	}
	return endSeg{}, false
}

// tubeCapTransitArc is the transit edge closing a miter arm's boundary across the OTHER arm's outer
// plane: the cylinder-arm tube's cross-section circle in that plane (centre = the spine point on the
// plane, radius = the fillet radius r), taken as the exact minor arc from the rail's junction to the
// seam endpoint sBot. Both points must lie on that circle within tol (they do by construction: the
// junction is the tube∩plane rail end, sBot the seam's tube∩tube∩plane vertex). ok=false otherwise.
func tubeCapTransitArc(spineAtCap math.Point3, r float64, junction, sBot math.Point3, tol float64) (endSeg, bool) {
	if stdmath.Abs(float64(junction.DistanceTo(spineAtCap))-r) > tol ||
		stdmath.Abs(float64(sBot.DistanceTo(spineAtCap))-r) > tol {
		return endSeg{}, false // a transit endpoint is off the tube's cap circle — not this geometry
	}
	mid := arcMidBetween(spineAtCap, r, junction, sBot)
	arc, err := geom.Arc3dByThreePoints(junction, mid, sBot)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: junction, to: sBot, curve: arc, mid: mid, arc: true}, true
}

// cylArmCapJunction is where a cylinder arm's outer-host rail truly ends when the miter seam's sBot
// lies on the OTHER arm's outer plane instead of on this arm's own outer host: the last arm ball
// sits where the spine pierces that plane, and the junction is its tangency foot on this arm's
// outer host. Requires the spine to actually cross the plane (not run parallel to it).
func cylArmCapJunction(outer *topo.Face, arm geom.Cylinder, capPl geom.Plane, r, tol float64) (spineAtCap, junction math.Point3, ok bool) {
	n := planeUnitNormal(capPl)
	denom := float64(arm.AxisDir.AsVector().Dot(n))
	if stdmath.Abs(denom) < sinFloor {
		return math.Point3{}, math.Point3{}, false // spine parallel to the cap plane — no last ball
	}
	t := float64(arm.Origin.VectorTo(capPl.Origin).Dot(n)) / denom
	spineAtCap = arm.Origin.TranslateBy(arm.AxisDir.AsVector().Scale(math.Scalar(t)))
	junction, ok = armRunoutFoot(outer, spineAtCap, r, tol)
	return spineAtCap, junction, ok
}

// planeUnitNormal is the plane's unit normal vector (geometry-side, orientation as stored).
func planeUnitNormal(pl geom.Plane) math.Vector3 {
	return pl.Normal()
}

// bridgedRunoutHostFace is the single-arm runout's chain fallback for a CONVEX arm whose bite has
// exactly one MID-FACE foot at a capped end (blend/simple/P4: the fillet band stops on the pocket
// floor, so the wall's rail foot and the floor's trim foot both land on a fresh latitude cut that
// is on no original edge). The foot is bridged back to the picked edge's end vertex along the
// wall ∩ cap-plane curve — the SAME capBridgeArc construction on both bitten hosts, so the fresh
// edge welds watertight — and the chain splices through the sequential chain retrim.
func bridgedRunoutHostFace(f *topo.Face, ef edgeFillet, bite endSeg, tol float64) (filletFace, bool) {
	if ef.armConcave {
		return filletFace{}, false // the concave grow-retrim family has its own splice
	}
	oriented, foot, ok := runoutBiteOffFoot(f, bite, tol)
	if !ok {
		return filletFace{}, false
	}
	vEnd, vOther := miterChainEndVertices(ef, foot)
	wall, ok := runoutBridgeWall(f, ef, foot, tol)
	if !ok {
		return filletFace{}, false
	}
	bridge, ok := miterChainEndBridgeSeg(wall, foot, vEnd, tol)
	if !ok {
		return filletFace{}, false
	}
	chain := miterHostChain{chain: []endSeg{bridge, oriented}, consumed: vOther, hasConsumed: true}
	return chainedHostRetrim(f, []miterHostChain{chain}, tol)
}

// runoutBiteOffFoot finds the bite's single mid-face foot on host f and returns the bite oriented
// to START there. ok=false when both feet already lie on f's loops (not this fallback's shape) or
// both are mid-face (outside the single-bridge scope).
func runoutBiteOffFoot(f *topo.Face, bite endSeg, tol float64) (endSeg, math.Point3, bool) {
	fromOn := miterCapPointOnLoops(f, bite.from, tol)
	toOn := miterCapPointOnLoops(f, bite.to, tol)
	if fromOn == toOn {
		return endSeg{}, math.Point3{}, false
	}
	if !fromOn {
		return bite, bite.from, true
	}
	return reverseEndSegs([]endSeg{bite})[0], bite.to, true
}

// miterChainEndVertices splits the picked edge's two end vertices by the off-loop foot: vEnd is the
// bridge anchor (the vertex at the foot's own capped end), vOther the far one — the vertex the
// splice must consume when it lies on the bitten ring (the wrong-span guard).
func miterChainEndVertices(ef edgeFillet, foot math.Point3) (vEnd, vOther math.Point3) {
	v0, v1 := ef.edge.StartVertex().Point(), ef.edge.EndVertex().Point()
	if float64(foot.DistanceTo(v0)) <= float64(foot.DistanceTo(v1)) {
		return v0, v1
	}
	return v1, v0
}

// runoutBridgeWall picks the surface the bridge curve lives on besides the cap plane: for a bitten
// ARM host the wall is that host itself; for a bitten CAP the wall is whichever arm host the foot
// lies on. Both sides construct the bridge from the SAME wall surface, so the fresh edge is the
// same curve on both and the weld is watertight by construction.
func runoutBridgeWall(f *topo.Face, ef edgeFillet, foot math.Point3, tol float64) (geom.Surface, bool) {
	if f == ef.a || f == ef.b {
		return f.Geometry(), true
	}
	if distanceToSurface(ef.a, foot) <= tol {
		return ef.a.Geometry(), true
	}
	if distanceToSurface(ef.b, foot) <= tol {
		return ef.b.Geometry(), true
	}
	return nil, false // the foot is on neither arm host — not a capped-end fresh cut
}

// trimEndNearerSurface orients a far-runout trim by its endpoints' distance to a host surface and
// returns the endpoint lying on (nearer to) that surface — the trim's foot on that host. Used to
// reconcile an outer rail with its far trim when the capping face is curved: the rail must end at
// the trim's actual triple point on the outer host, not at the far ball's tangency foot.
func trimEndNearerSurface(trim endSeg, host *topo.Face) math.Point3 {
	if distanceToSurface(host, trim.from) <= distanceToSurface(host, trim.to) {
		return trim.from
	}
	return trim.to
}
