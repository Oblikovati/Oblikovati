// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Chain-aware loop-splice primitives for the corner-weld layer. They generalize the two-arm corner-host
// splice from ONE loop segment per bite to a CONTIGUOUS RUN of segments, and from one rail per bite to a
// rail CHAIN — what a rim continuation needs (its contact rail crosses several host-face spans, so on the
// shared host it replaces several loop edges with several rail pieces). With single-element runs and chains
// they reduce EXACTLY to the previous mixedCornerHostRetrim code path, which is why that function is now a
// thin adapter over cornerChainHostRetrim rather than a second implementation.

// segRun is a CONTIGUOUS run of loop-segment indices in ring order, lo→hi, possibly wrapping past n−1.
// lo<0 marks the empty run.
type segRun struct {
	lo, hi int
}

// has reports whether index k falls inside the run (wrap-aware).
func (r segRun) has(k int) bool {
	if r.lo < 0 {
		return false
	}
	if r.lo <= r.hi {
		return k >= r.lo && k <= r.hi
	}
	return k >= r.lo || k <= r.hi
}

// contiguousSegRun turns an unordered index set into the single ring run covering exactly it, or ok=false when
// the indices are not contiguous in the ring (an arm whose consumed edges are scattered around the loop is
// not something the splice can re-clip — floor honestly).
func contiguousSegRun(idx []int, n int) (segRun, bool) {
	if len(idx) == 0 || n == 0 {
		return segRun{lo: -1, hi: -1}, false
	}
	in := make([]bool, n)
	for _, i := range idx {
		if i < 0 || i >= n || in[i] {
			return segRun{}, false // out of range, or the same segment claimed twice
		}
		in[i] = true
	}
	for _, lo := range idx {
		if in[(lo-1+n)%n] {
			continue // not the run's start
		}
		hi, count := lo, 1
		for in[(hi+1)%n] && count < len(idx) {
			hi, count = (hi+1)%n, count+1
		}
		if count == len(idx) {
			return segRun{lo: lo, hi: hi}, true
		}
	}
	return segRun{}, false
}

// rebuildChainRing rebuilds the ordered ring with runFirst+runSecond replaced by the chain and the two
// outer flanks by their re-terminated forms. Ring positions are otherwise preserved, so closure holds by
// construction. runSecond may be empty (the single-arm splice).
func rebuildChainRing(segs []endSeg, runFirst, runSecond segRun, before, after int, flankBefore endSeg, chain []endSeg, flankAfter endSeg) []endSeg {
	out := make([]endSeg, 0, len(segs)+len(chain))
	for k := range segs {
		switch {
		case k == before:
			out = append(out, flankBefore)
		case k == runFirst.lo:
			out = append(out, chain...)
		case runFirst.has(k) || runSecond.has(k):
			// folded into the chain — drop
		case k == after:
			out = append(out, flankAfter)
		default:
			out = append(out, segs[k])
		}
	}
	return out
}

// cornerChainHostRetrim re-clips one host bitten by TWO arms at the corner vertex: each arm's consumed run
// is replaced by its rail chain, the two chains are joined (directly at a triple point, or through the
// patch's own on-host rail `mid`), and the two outer flanking edges are re-terminated onto the combined
// chain's extreme feet. Declines when a bite's edges are not a contiguous run, the two runs are not
// adjacent at vCorner, or a foot leaves a flank's supporting line/circle — the do-no-harm floor.
func cornerChainHostRetrim(host *topo.Face, biteA, biteB cornerHostChainBite, mid []endSeg, vCorner math.Point3, tol float64) (filletFace, bool) {
	bitten := hostBittenLoop(host, vCorner, tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	retrim, ok := cornerChainLoop(segsFromLoop(bitten), biteA, biteB, mid, vCorner, tol)
	if !ok {
		return filletFace{}, false
	}
	return filletFace{surface: host.Geometry(), loops: hostLoopsWithRetrim(host, bitten, outer, retrim), parent: host.Lineage()}, true
}

// cornerChainLoop rebuilds the bitten loop for a two-arm corner host (see cornerChainHostRetrim).
func cornerChainLoop(segs []endSeg, biteA, biteB cornerHostChainBite, mid []endSeg, vCorner math.Point3, tol float64) (filletLoop, bool) {
	n := len(segs)
	runA, okA := consumedRun(segs, biteA.consumed, tol)
	runB, okB := consumedRun(segs, biteB.consumed, tol)
	if !okA || !okB || n < 3 {
		return filletLoop{}, false
	}
	first, second, bFirst, bSecond := orderChainPicks(segs, runA, runB, biteA, biteB, vCorner, tol)
	if first.lo < 0 || (second.hi+1)%n == first.lo {
		return filletLoop{}, false // not adjacent at vCorner, or no flank left on either side
	}
	midSeq, ok := orientMidToChain(mid, bFirst.rails[len(bFirst.rails)-1].to, tol)
	if !ok {
		return filletLoop{}, false
	}
	chain := append(append([]endSeg{}, bFirst.rails...), midSeq...)
	chain = append(chain, reverseEndSegs(bSecond.rails)...)
	return weldChainRuns(segs, first, second, chain, tol)
}

// orderChainPicks assigns (first, second) so first's run ENDS at vCorner and second's STARTS there (first
// before second in ring order), with their bites oriented to match. first.lo=−1 when neither ordering holds.
func orderChainPicks(segs []endSeg, runA, runB segRun, biteA, biteB cornerHostChainBite, vCorner math.Point3, tol float64) (segRun, segRun, cornerHostChainBite, cornerHostChainBite) {
	if float64(segs[runA.hi].to.DistanceTo(vCorner)) <= tol && float64(segs[runB.lo].from.DistanceTo(vCorner)) <= tol {
		return runA, runB, biteA, biteB
	}
	if float64(segs[runB.hi].to.DistanceTo(vCorner)) <= tol && float64(segs[runA.lo].from.DistanceTo(vCorner)) <= tol {
		return runB, runA, biteB, biteA
	}
	return segRun{lo: -1, hi: -1}, segRun{}, cornerHostChainBite{}, cornerHostChainBite{}
}

// orientMidToChain returns the patch's on-host rail chain traversed so it CONTINUES from `head` — the near
// foot the first bite's rail chain ends at. ★ Slice 2 (O1) found this: the orientation must be decided
// GEOMETRICALLY, never by which bite happens to be registered first. A class builder is free to list its arms
// in any order, and the mid rail is registered once in the patch RING's direction (D→A), so "reverse it iff
// the second bite leads" only holds when the ring's D foot belongs to the first-listed arm. N4's does by
// coincidence; O1's does not, and the mismatch produced a loop whose segments were shifted by one — a body
// that was Valid ∧ Manifold with four 1-incident edges. ok=false when the chain reaches neither foot, which is
// an inconsistent plan and must floor rather than weld a shifted ring.
func orientMidToChain(mid []endSeg, head math.Point3, tol float64) ([]endSeg, bool) {
	if len(mid) == 0 {
		return nil, true // the two rails meet at a triple point; there is nothing to orient
	}
	if float64(mid[0].from.DistanceTo(head)) <= tol {
		return mid, true
	}
	rev := reverseEndSegs(mid)
	if float64(rev[0].from.DistanceTo(head)) <= tol {
		return rev, true
	}
	return nil, false
}

// weldChainRuns splices the corner chain (far foot → … → far foot) into the ring in place of the two runs,
// re-terminating the flanking edges onto the chain's extreme feet. When the two runs leave a SINGLE shared
// flank it is re-terminated on BOTH ends; otherwise the two distinct flanks grow/recede independently.
func weldChainRuns(segs []endSeg, first, second segRun, chain []endSeg, tol float64) (filletLoop, bool) {
	n := len(segs)
	footFirst, footSecond := chain[0].from, chain[len(chain)-1].to
	before, after := (first.lo-1+n)%n, (second.hi+1)%n
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
	return loopFromSegs(rebuildChainRing(segs, first, second, before, after, flankBefore, chain, flankAfter)), true
}
