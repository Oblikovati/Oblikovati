// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/topo"
)

// Stage 5 of the shared corner-weld executor: re-clip every bitten host, retrim every far cap, and carry
// every untouched face through. A host touched by TWO arms is the corner host (its two picked edges at the
// vertex are replaced by the two rail chains, optionally joined by the patch's own on-host rail); a host
// touched by ONE arm is a single-arm bite. Caps accumulate their bites so a face capping several arms —
// N4's box wall, which BOTH grows around the concave band's cross-section AND recedes around the convex
// rim arm's — gets all of them.

// cornerPlanHostFaces builds the retrimmed hosts + caps + passthrough faces for the whole plan.
func cornerPlanHostFaces(body *topo.Body, plan cornerWeldPlan, welds []cornerArmWeld, res Resolution) ([]filletFace, string) {
	tol := res.Weld() * plan.radius
	retrims, reason := cornerHostRetrims(plan, welds, tol)
	if reason != "" {
		return nil, reason
	}
	caps, reason := cornerCapRetrims(plan, welds, tol)
	if reason != "" {
		return nil, reason
	}
	for f, ff := range caps {
		if _, dup := retrims[f]; dup {
			// TODO(slice 3 — the multi-corner ring, N2/N8/O2/Y9): a ring's shared boss-base face IS both an
			// arm host and another arm's far cap, so it needs ONE retrim carrying both the contact-rail
			// splice and the cap bite. Declining honestly until that composition exists.
			return nil, fmt.Sprintf("corner weld: face %d is both an arm host and a far cap (unsupported)", f.ID())
		}
		retrims[f] = ff
	}
	out := make([]filletFace, 0, len(body.Faces()))
	for _, f := range body.Faces() {
		if ff, bitten := retrims[f]; bitten {
			out = append(out, ff)
			continue
		}
		out = append(out, passthroughFace(f))
	}
	return out, ""
}

// cornerHostRetrims re-clips each bitten host once, dispatching on how many arms bite it.
func cornerHostRetrims(plan cornerWeldPlan, welds []cornerArmWeld, tol float64) (map[*topo.Face]filletFace, string) {
	grouped, reason := groupHostBites(plan, welds)
	if reason != "" {
		return nil, reason
	}
	out := map[*topo.Face]filletFace{}
	for _, g := range grouped {
		ff, reason := cornerHostRetrim(plan, g, tol)
		if reason != "" {
			return nil, reason
		}
		out[g.face] = ff
	}
	return out, ""
}

// cornerHostRetrim dispatches one host: two bites → the corner-host splice (the two picked edges at the
// vertex), one bite → the single-arm splice (a host the arm merely runs across, which a rim continuation
// reaches). Any other count is out of the layer's scope and floors honestly.
func cornerHostRetrim(plan cornerWeldPlan, g hostBiteGroup, tol float64) (filletFace, string) {
	if len(g.bites) == 2 {
		ff, ok := cornerChainHostRetrim(g.face, g.bites[0], g.bites[1], g.mid, plan.vertex, tol)
		if !ok {
			return filletFace{}, fmt.Sprintf("corner weld: two-arm host %d retrim declined", g.face.ID())
		}
		return ff, ""
	}
	if len(g.bites) == 1 {
		ff, ok := cornerSingleHostRetrim(g.face, g.bites[0], tol)
		if !ok {
			return filletFace{}, fmt.Sprintf("corner weld: single-arm host %d retrim declined", g.face.ID())
		}
		return ff, ""
	}
	// TODO(slice 3 — rings/loops, M3/M9/Q2 and N2/N8/O2/Y9): a host wrapped by a closed pick loop takes
	// THREE OR MORE bites in one retrim, which needs the splice generalized to an ordered set of runs
	// rather than the 1-or-2 dispatch here. Honest decline until then.
	return filletFace{}, fmt.Sprintf("corner weld: host %d is bitten by %d arms (want 1 or 2)", g.face.ID(), len(g.bites))
}

// hostBiteGroup is one host face's resolved bites plus the patch rail chain riding on it (nil when the two
// arm rails meet at a triple point).
type hostBiteGroup struct {
	face  *topo.Face
	bites []cornerHostChainBite
	mid   []endSeg
}

// cornerHostChainBite is one arm's resolved contribution to a host retrim: its contact rail chain
// (FAR→NEAR) and the loop edges the chain replaces.
type cornerHostChainBite struct {
	rails    []endSeg
	consumed []*topo.Edge
}

// groupHostBites resolves every arm's host bites from the ledger and groups them by face, preserving
// first-seen order so the retrim sequence is deterministic. Each bite is claimed under its HOST face, so
// two arms biting one host read as one claimant per rail rather than two anonymous references.
func groupHostBites(plan cornerWeldPlan, welds []cornerArmWeld) ([]hostBiteGroup, string) {
	index := map[*topo.Face]int{}
	var out []hostBiteGroup
	for _, w := range welds {
		for _, b := range w.hosts {
			rails, ok := plan.ledger.chain(b.rails, railForward, hostClaimant(b.face))
			if !ok {
				return nil, fmt.Sprintf("corner weld: host %d bite claims an unregistered rail (an unset handle)", b.face.ID())
			}
			bite := cornerHostChainBite{rails: rails, consumed: b.consumed}
			if k, seen := index[b.face]; seen {
				out[k].bites = append(out[k].bites, bite)
				continue
			}
			mid, ok := midRailsFor(plan, b.face)
			if !ok {
				return nil, fmt.Sprintf("corner weld: host %d patch rail claims an unregistered rail (an unset handle)", b.face.ID())
			}
			index[b.face] = len(out)
			out = append(out, hostBiteGroup{face: b.face, bites: []cornerHostChainBite{bite}, mid: mid})
		}
	}
	return out, ""
}

// midRailsFor resolves the patch side that rides on this host (design Axis A3), or nil when none does.
// ok=false only when a registered mid chain carries an unset handle.
func midRailsFor(plan cornerWeldPlan, face *topo.Face) ([]endSeg, bool) {
	for _, m := range plan.mids {
		if m.face == face {
			return plan.ledger.chain(m.rails, railForward, hostClaimant(face))
		}
	}
	return nil, true
}

// cornerCapRetrims accumulates every arm's far-cap bite onto its capping face: a convex arm's cap RECEDES
// around the trim (spliceCornerBite), a concave arm's cap GROWS around it (growCapArc).
func cornerCapRetrims(plan cornerWeldPlan, welds []cornerArmWeld, tol float64) (map[*topo.Face]filletFace, string) {
	work := map[*topo.Face][]endSeg{}
	for _, w := range welds {
		c := w.cap
		if c.face == nil {
			return nil, "corner weld: an arm far termination produced no capping face"
		}
		if _, seen := work[c.face]; !seen {
			work[c.face] = segsFromLoop(outerHostLoop(c.face))
		}
		trim, ok := plan.ledger.seg(c.trim, railForward, hostClaimant(c.face))
		if !ok {
			return nil, fmt.Sprintf("corner weld: far-cap %d claims an unregistered runout trim (an unset handle)", c.face.ID())
		}
		segs, reason := applyCapBite(work[c.face], trim, c, tol)
		if reason != "" {
			return nil, reason
		}
		work[c.face] = segs
	}
	out := map[*topo.Face]filletFace{}
	for face, segs := range work {
		out[face] = capFaceFromSegs(face, segs)
	}
	return out, ""
}

// applyCapBite recedes or grows one cap around one arm's runout trim.
func applyCapBite(segs []endSeg, trim endSeg, c cornerCapBite, tol float64) ([]endSeg, string) {
	if c.sense == biteInward {
		spliced, ok := spliceCornerBite(segs, trim, tol)
		if !ok {
			return nil, fmt.Sprintf("corner weld: far-cap %d recede declined", c.face.ID())
		}
		return spliced, ""
	}
	grown, ok := growCapArc(segs, trim, c.far, tol)
	if !ok {
		return nil, fmt.Sprintf("corner weld: far-cap %d grow declined at %v", c.face.ID(), c.far)
	}
	return grown, ""
}

// cornerSingleHostRetrim re-clips a host bitten by ONE arm: the consumed loop edges are replaced by the
// arm's contact rail chain and the two flanking edges are re-terminated onto the chain's feet, each on its
// own supporting line/circle. This is the shape concaveRetrimLoop has for a single picked edge, lifted to a
// rail CHAIN over a contiguous RUN so a rim continuation can re-clip a host it merely crosses (N4's second
// boss-wall face, receded over its whole 180° span). Declines when the consumed edges are not a contiguous
// run or a foot leaves a flank's geometry — the do-no-harm floor.
func cornerSingleHostRetrim(host *topo.Face, bite cornerHostChainBite, tol float64) (filletFace, bool) {
	if len(bite.consumed) == 0 {
		return filletFace{}, false
	}
	bitten := hostBittenLoop(host, bite.consumed[0].StartVertex().Point(), tol)
	outer := outerHostLoop(host)
	if bitten == nil || outer == nil {
		return filletFace{}, false
	}
	segs := segsFromLoop(bitten)
	run, ok := consumedRun(segs, bite.consumed, tol)
	if !ok || len(segs) < 3 {
		return filletFace{}, false
	}
	retrim, ok := spliceSingleRun(segs, run, bite.rails, tol)
	if !ok {
		return filletFace{}, false
	}
	return filletFace{surface: host.Geometry(), loops: hostLoopsWithRetrim(host, bitten, outer, retrim), parent: host.Lineage()}, true
}

// spliceSingleRun replaces the consumed run with the rail chain — oriented to the loop's own traversal by
// matching each chain end to the flanking edge's supporting geometry — and re-terminates the two flanks.
func spliceSingleRun(segs []endSeg, run segRun, rails []endSeg, tol float64) (filletLoop, bool) {
	n := len(segs)
	before, after := (run.lo-1+n)%n, (run.hi+1)%n
	if before == after {
		return filletLoop{}, false // one shared flank leaves nothing to re-terminate independently
	}
	chain, ok := orientChainToFlanks(segs[before], segs[after], rails, tol)
	if !ok {
		return filletLoop{}, false
	}
	prev, okp := reterminateSegTo(segs[before], chain[0].from, tol)
	next, okn := reterminateSegFrom(segs[after], chain[len(chain)-1].to, tol)
	if !okp || !okn {
		return filletLoop{}, false
	}
	return loopFromSegs(rebuildChainRing(segs, run, segRun{lo: -1, hi: -1}, before, after, prev, chain, next)), true
}

// orientChainToFlanks returns the rail chain traversed so its FIRST foot lies on prev's supporting geometry
// and its LAST on next's — the loop's own direction. ok=false when neither orientation matches.
func orientChainToFlanks(prev, next endSeg, rails []endSeg, tol float64) ([]endSeg, bool) {
	if len(rails) == 0 {
		return nil, false
	}
	if segSupportsPoint(prev, rails[0].from, tol) && segSupportsPoint(next, rails[len(rails)-1].to, tol) {
		return rails, true
	}
	rev := reverseEndSegs(rails)
	if segSupportsPoint(prev, rev[0].from, tol) && segSupportsPoint(next, rev[len(rev)-1].to, tol) {
		return rev, true
	}
	return nil, false
}

// consumedRun locates the CONTIGUOUS ring run of loop segments the bite's consumed edges occupy. A rim
// continuation consumes several adjacent edges; a plain arm consumes one.
func consumedRun(segs []endSeg, consumed []*topo.Edge, tol float64) (segRun, bool) {
	idx := make([]int, 0, len(consumed))
	for _, e := range consumed {
		i := indexOfPickedEdge(segs, e.StartVertex().Point(), e.EndVertex().Point(), tol)
		if i < 0 {
			return segRun{}, false // this loop does not carry that edge as one segment
		}
		idx = append(idx, i)
	}
	return contiguousSegRun(idx, len(segs))
}
