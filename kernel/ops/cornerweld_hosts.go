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
	grouped := groupHostBites(plan, welds)
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

// cornerHostRetrim dispatches one host. A two-arm corner host is the only shape this slice serves; any
// other count is out of scope and floors honestly (do-no-harm).
func cornerHostRetrim(plan cornerWeldPlan, g hostBiteGroup, tol float64) (filletFace, string) {
	if len(g.bites) != 2 {
		return filletFace{}, fmt.Sprintf("corner weld: host %d is bitten by %d arms (want exactly 2)", g.face.ID(), len(g.bites))
	}
	ff, ok := cornerChainHostRetrim(g.face, g.bites[0], g.bites[1], g.mid, plan.vertex, tol)
	if !ok {
		return filletFace{}, fmt.Sprintf("corner weld: two-arm host %d retrim declined", g.face.ID())
	}
	return ff, ""
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
// first-seen order so the retrim sequence is deterministic.
func groupHostBites(plan cornerWeldPlan, welds []cornerArmWeld) []hostBiteGroup {
	index := map[*topo.Face]int{}
	var out []hostBiteGroup
	for _, w := range welds {
		for _, b := range w.hosts {
			bite := cornerHostChainBite{rails: plan.ledger.chain(b.rails, railForward), consumed: b.consumed}
			k, seen := index[b.face]
			if !seen {
				index[b.face] = len(out)
				out = append(out, hostBiteGroup{face: b.face, bites: []cornerHostChainBite{bite}, mid: midRailsFor(plan, b.face)})
				continue
			}
			out[k].bites = append(out[k].bites, bite)
		}
	}
	return out
}

// midRailsFor resolves the patch side that rides on this host (design Axis A3), or nil when none does.
func midRailsFor(plan cornerWeldPlan, face *topo.Face) []endSeg {
	for _, m := range plan.mids {
		if m.face == face {
			return plan.ledger.chain(m.rails, railForward)
		}
	}
	return nil
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
		segs, reason := applyCapBite(work[c.face], plan.ledger.seg(c.trim, railForward), c, tol)
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
