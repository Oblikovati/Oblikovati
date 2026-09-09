// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
)

// Dissolving the boundary two faces on one surface share (ADR-0061 stage 5).
//
// The merged boundary is the two faces' loops with every shared edge removed, re-chained. Re-chaining
// them by endpoint is AMBIGUOUS — four seam ends meet at the point where two coaxial bands' seams
// join — so nothing here reads a coordinate to decide a successor. It follows each loop's own order,
// and at a dissolved edge it crosses to the edge after that edge's twin. That is the whole of "the two
// faces are now one", and it is exact.
//
// What the crossing leaves behind is the artificial seam. A face that wraps the surface carries the
// seam twice, and when the run that seam ended on dissolves, the two traversals become adjacent: a
// slit dangling into the merged face's interior, which is not a boundary at all. dropSeamSlits removes
// exactly those adjacent reverse twins, and a loop that is nothing else disappears with them.

// edgeAddr addresses one directed boundary edge of the merging pair: which face (0 = a, 1 = b), which
// of its loops, which edge of that loop.
type edgeAddr struct{ face, loop, edge int }

// boundaryDissolve is the merge's combinatorial state: both faces' loops, the twin of every dissolved
// edge, and the edges already walked.
type boundaryDissolve struct {
	loops [2][]curvedLoop
	twin  map[edgeAddr]edgeAddr
	seen  map[edgeAddr]bool
	total int
}

// dissolveSharedEdges is the merged face's boundary: both faces' loops fused across every edge they
// share, with the seam the fusion orphans removed. It names its reason when it does not complete, so
// every exit but the ordinary "they share no boundary" reaches the result as a diagnostic.
func dissolveSharedEdges(a, b curvedFace, res geom.Resolution) ([]curvedLoop, mergeDecline) {
	a, b = splitAtSharedRunEnds(a, b, res)
	twin, why := sharedEdgeTwins(a, b, res)
	if why != mergeJoined {
		return nil, why
	}
	d := &boundaryDissolve{loops: [2][]curvedLoop{a.loops, b.loops}, twin: twin,
		seen: map[edgeAddr]bool{}, total: countLoopEdges(a) + countLoopEdges(b)}
	loops, closed := d.mergedLoops()
	if !closed {
		return nil, declineOpenWalk
	}
	return dropSeamSlits(loops), mergeJoined
}

// countLoopEdges is how many directed boundary edges a face has, which bounds every walk here.
func countLoopEdges(f curvedFace) int {
	n := 0
	for _, l := range f.loops {
		n += len(l.edges)
	}
	return n
}

// mergedLoops walks every surviving edge exactly once, in face/loop/edge index order so the merged
// loops come out in one order on every run.
func (d *boundaryDissolve) mergedLoops() ([]curvedLoop, bool) {
	var out []curvedLoop
	for _, start := range d.addresses() {
		if d.seen[start] || d.dissolved(start) {
			continue
		}
		loop, ok := d.loopFrom(start)
		if !ok {
			return nil, false
		}
		out = append(out, loop)
	}
	return out, true
}

// addresses lists both faces' directed edges in index order.
func (d *boundaryDissolve) addresses() []edgeAddr {
	var out []edgeAddr
	for face, loops := range d.loops {
		for li, l := range loops {
			for ei := range l.edges {
				out = append(out, edgeAddr{face: face, loop: li, edge: ei})
			}
		}
	}
	return out
}

// loopFrom walks one merged loop from a surviving edge back round to it.
func (d *boundaryDissolve) loopFrom(start edgeAddr) (curvedLoop, bool) {
	edges := make([]loopEdge, 0, d.total)
	for at := start; len(edges) <= d.total; {
		d.seen[at] = true
		edges = append(edges, d.edgeAt(at))
		next, ok := d.nextKept(at)
		if !ok {
			return curvedLoop{}, false
		}
		if next == start {
			return curvedLoop{edges: edges}, true
		}
		at = next
	}
	return curvedLoop{}, false // more edges than the two faces have: the successor was not a bijection
}

// nextKept is the merged boundary's successor: the next edge of the same loop, and — a dissolved edge
// being no boundary any more — the edge after its twin whenever the step lands on one. The bound is
// the two faces' edge count: a run that never leaves the dissolved set is not a boundary, and declines.
func (d *boundaryDissolve) nextKept(at edgeAddr) (edgeAddr, bool) {
	for i := 0; i <= d.total; i++ {
		at = d.successor(at)
		if !d.dissolved(at) {
			return at, true
		}
		at = d.twin[at]
	}
	return edgeAddr{}, false
}

// successor is the next edge of the loop the address lies in.
func (d *boundaryDissolve) successor(at edgeAddr) edgeAddr {
	n := len(d.loops[at.face][at.loop].edges)
	return edgeAddr{face: at.face, loop: at.loop, edge: (at.edge + 1) % n}
}

// dissolved reports whether the addressed edge is one of the shared pair.
func (d *boundaryDissolve) dissolved(at edgeAddr) bool {
	_, ok := d.twin[at]
	return ok
}

// edgeAt is the addressed edge.
func (d *boundaryDissolve) edgeAt(at edgeAddr) loopEdge {
	return d.loops[at.face][at.loop].edges[at.edge]
}
