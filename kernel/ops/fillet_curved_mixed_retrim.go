// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner-host retrim splice machinery for the mixed-sense curved-host 2r-torus corner weld
// (fillet_curved_mixed_weld.go). Split out of that file for the CLAUDE.md files-under-500 rule — a pure
// move, no behaviour change. These helpers re-clip a two-arm corner host (the two picked edges meeting at
// the trihedral vertex, replaced by the arm rails + optional top-contact arc, flanks re-terminated) and
// grow a concave far-runout cap around its cross-section arc, generalising the single-arm concave grow
// retrim (fillet_arm_concave_retrim.go: concaveRetrimLoop / concaveCapLoop) from one picked edge to two.

// cornerBite is one arm's contribution to a two-arm corner-host retrim: its host contact rail (oriented
// far→near) and the picked edge whose loop segment the rail replaces.
type cornerBite struct {
	rail endSeg
	edge *topo.Edge
}

// mixedCornerHostRetrim re-clips one two-arm corner host: the two picked edges meeting at vCorner are
// removed, replaced by [firstRail(far→near), mid…, secondRail(near→far)], and the two OUTER flanking edges
// are re-terminated onto the chain's extreme feet (grow OR recede, along each flank's own supporting
// line/circle). mid is the top-contact arc (d) on the shared top plane, nil on the two walls (their rails
// meet at a single triple point). It generalises concaveRetrimLoop from one picked edge to the corner's
// two. Declines (ok=false) when a picked edge is not a loop segment, the two are not adjacent at vCorner,
// or a foot leaves a flanking edge's supporting geometry — the do-no-harm floor.
func mixedCornerHostRetrim(host *topo.Face, biteA, biteB cornerBite, mid []endSeg, vCorner math.Point3, tol float64) (filletFace, bool) {
	bitten := hostBittenLoop(host, vCorner, tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	retrim, ok := mixedCornerLoop(segsFromLoop(bitten), biteA, biteB, mid, vCorner, tol)
	if !ok {
		return filletFace{}, false
	}
	return filletFace{surface: host.Geometry(), loops: hostLoopsWithRetrim(host, bitten, outer, retrim), parent: host.Lineage()}, true
}

// mixedCornerLoop rebuilds the bitten loop for a two-arm corner host. It orders the two picked segments so
// segs[first].to == vCorner == segs[second].from (first before second in the ring), builds the chain
// first.rail(far→near) + mid + reverse(second.rail)(near→far), re-terminates the flanking edges before
// first / after second onto the chain's extreme feet, and rebuilds the ring with the two picked segments
// replaced by the chain. Declines on the do-no-harm obstructions named in mixedCornerHostRetrim.
func mixedCornerLoop(segs []endSeg, biteA, biteB cornerBite, mid []endSeg, vCorner math.Point3, tol float64) (filletLoop, bool) {
	n := len(segs)
	iA := indexOfPickedEdge(segs, biteA.edge.StartVertex().Point(), biteA.edge.EndVertex().Point(), tol)
	iB := indexOfPickedEdge(segs, biteB.edge.StartVertex().Point(), biteB.edge.EndVertex().Point(), tol)
	if iA < 0 || iB < 0 || n < 3 {
		return filletLoop{}, false
	}
	first, second, bFirst, bSecond, midSeq := orderCornerPicks(segs, iA, iB, biteA, biteB, mid, vCorner, tol)
	if first < 0 || (second+1)%n == first {
		return filletLoop{}, false // not adjacent, or a loop too small to keep a flank on each side
	}
	chain := append([]endSeg{bFirst.rail}, midSeq...)
	chain = append(chain, reverseEndSegs([]endSeg{bSecond.rail})...)
	return weldCornerChain(segs, first, second, chain, tol)
}

// weldCornerChain splices the corner chain (far foot → … → far foot) into the ring in place of the two
// picked segments first/second, re-terminating the flanking edges onto the chain's extreme feet. When the
// two picks leave a SINGLE shared flank (an n==3 loop) it re-terminates that flank on BOTH ends; otherwise
// it grows/recedes the two distinct flanks. Declines when a foot leaves a flank's supporting geometry.
func weldCornerChain(segs []endSeg, first, second int, chain []endSeg, tol float64) (filletLoop, bool) {
	n := len(segs)
	footFirst, footSecond := chain[0].from, chain[len(chain)-1].to
	before, after := (first-1+n)%n, (second+1)%n
	if before == after {
		flank, ok := reterminateBothEnds(segs[before], footSecond, footFirst, tol)
		if !ok {
			return filletLoop{}, false
		}
		return loopFromSegs(append(append([]endSeg{}, chain...), flank)), true
	}
	flankBefore, okp := reterminateSegTo(segs[before], footFirst, tol)
	flankAfter, okn := reterminateSegFrom(segs[after], footSecond, tol)
	if !okp || !okn {
		return filletLoop{}, false
	}
	return loopFromSegs(rebuildCornerRing(segs, first, second, flankBefore, chain, flankAfter)), true
}

// orderCornerPicks assigns (first, second) so segs[first] enters vCorner and segs[second] leaves it (first
// before second in ring order), with their bites and the mid chain oriented to match. Returns first=−1 when
// neither picked segment touches vCorner as expected.
func orderCornerPicks(segs []endSeg, iA, iB int, biteA, biteB cornerBite, mid []endSeg, vCorner math.Point3, tol float64) (int, int, cornerBite, cornerBite, []endSeg) {
	if float64(segs[iA].to.DistanceTo(vCorner)) <= tol && float64(segs[iB].from.DistanceTo(vCorner)) <= tol {
		return iA, iB, biteA, biteB, mid
	}
	if float64(segs[iB].to.DistanceTo(vCorner)) <= tol && float64(segs[iA].from.DistanceTo(vCorner)) <= tol {
		return iB, iA, biteB, biteA, reverseEndSegs(mid)
	}
	return -1, -1, cornerBite{}, cornerBite{}, nil
}

// rebuildCornerRing rebuilds the ordered ring with picked segments first/second replaced by the chain and
// their outer flanks by the re-terminated forms — positions preserved so the ring stays closed.
func rebuildCornerRing(segs []endSeg, first, second int, flankBefore endSeg, chain []endSeg, flankAfter endSeg) []endSeg {
	n := len(segs)
	out := make([]endSeg, 0, n+len(chain))
	for k := 0; k < n; k++ {
		switch k {
		case (first - 1 + n) % n:
			out = append(out, flankBefore)
		case first:
			out = append(out, chain...)
		case second:
			// folded into the chain — drop
		case (second + 1) % n:
			out = append(out, flankAfter)
		default:
			out = append(out, segs[k])
		}
	}
	return out
}

// reterminateBothEnds re-terminates a single flanking edge on BOTH ends (the n==3 corner-host loop, where
// the two picked edges leave one shared flank): its from→newFrom and to→newTo, both on the flank's own
// supporting line (straight) or circle (arc). Declines when a foot leaves that geometry.
func reterminateBothEnds(s endSeg, newFrom, newTo math.Point3, tol float64) (endSeg, bool) {
	if !s.arc {
		if !pointOnLine(s.from, s.to, newFrom, tol) || !pointOnLine(s.from, s.to, newTo, tol) {
			return endSeg{}, false
		}
		return endSeg{from: newFrom, to: newTo}, true
	}
	return rebuildArcSeg(s.curve.(geom.Arc3d), newFrom, newTo, tol)
}

// growCapArc grows a concave far-runout cap around one cross-section arc: the far-vertex corner is replaced
// by the arc and the two flanking edges meeting there are re-terminated onto the arc's feet (each on its
// own host-shared line/circle) — the segs-level form of concaveCapLoop, so a cap shared by two concave
// arms accumulates both bites. Declines when the far vertex is not a loop corner or a foot is off a flank.
func growCapArc(segs []endSeg, arc endSeg, far math.Point3, tol float64) ([]endSeg, bool) {
	n := len(segs)
	j := indexOfVertex(segs, far, tol)
	if j < 0 || n < 3 {
		return nil, false
	}
	prevIdx := (j - 1 + n) % n
	fFrom, fTo, ok := matchArcFeet(segs[prevIdx], segs[j], arc, tol)
	prev, okp := reterminateSegTo(segs[prevIdx], fFrom, tol)
	next, okn := reterminateSegFrom(segs[j], fTo, tol)
	if !ok || !okp || !okn {
		return nil, false
	}
	arcSeg := endSeg{from: fFrom, to: fTo, curve: arc.curve, mid: arc.mid, arc: arc.arc}
	return spliceCapRing(segs, prevIdx, j, prev, arcSeg, next), true
}
