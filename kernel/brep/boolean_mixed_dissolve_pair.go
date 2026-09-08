// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
)

// Pairing the shared edges, and clearing the seam the pairing orphans (ADR-0061 stage 5).

// sharedEdgeTwins pairs each edge of a with the edge of b that walks the same stretch of the same
// curve the other way round. It names two different refusals: sharing nothing is the ordinary case,
// while an edge that would take two partners means their common boundary is subdivided differently on
// the two sides and re-chaining it would have to choose — a reported decline, not a guess.
func sharedEdgeTwins(a, b curvedFace, res geom.Resolution) (map[edgeAddr]edgeAddr, mergeDecline) {
	twin := map[edgeAddr]edgeAddr{}
	for _, x := range faceEdgeAddrs(a, 0) {
		y, n := onlyPartnerOf(a, x, b, res)
		if n == 0 {
			continue
		}
		if n > 1 || twinTaken(twin, x, y) {
			return nil, declineAmbiguousPairing
		}
		twin[x], twin[y] = y, x
	}
	if len(twin) == 0 {
		return nil, declineUnshared
	}
	return twin, mergeJoined
}

// twinTaken reports whether either end of the pair is already spoken for.
func twinTaken(twin map[edgeAddr]edgeAddr, x, y edgeAddr) bool {
	_, hasX := twin[x]
	_, hasY := twin[y]
	return hasX || hasY
}

// onlyPartnerOf returns b's edge that runs with x, and how many of b's edges do — more than one being
// the ambiguity sharedEdgeTwins declines on.
func onlyPartnerOf(a curvedFace, x edgeAddr, b curvedFace, res geom.Resolution) (edgeAddr, int) {
	var found edgeAddr
	n := 0
	for _, y := range faceEdgeAddrs(b, 1) {
		if !edgesRunTogether(edgeOf(a, x), edgeOf(b, y), res) {
			continue
		}
		if n == 0 {
			found = y
		}
		n++
	}
	return found, n
}

// faceEdgeAddrs lists a face's directed edges under the given side index (0 = a, 1 = b).
func faceEdgeAddrs(f curvedFace, face int) []edgeAddr {
	var out []edgeAddr
	for li, l := range f.loops {
		for ei := range l.edges {
			out = append(out, edgeAddr{face: face, loop: li, edge: ei})
		}
	}
	return out
}

// edgeOf resolves an address against the face it belongs to.
func edgeOf(f curvedFace, at edgeAddr) loopEdge { return f.loops[at.loop].edges[at.edge] }

// dropSeamSlits removes the artificial seams the dissolve orphans. A face that wraps its surface walks
// its seam twice; when the run that seam ended on dissolves, the two traversals become neighbours in
// the merged loop, and a curve walked up and straight back down bounds nothing — it is a slit dangling
// into the merged face's interior, not a boundary. A loop that is nothing but such a slit disappears.
func dropSeamSlits(loops []curvedLoop) []curvedLoop {
	out := make([]curvedLoop, 0, len(loops))
	for _, l := range loops {
		if edges := withoutSlitPairs(l.edges); len(edges) > 0 {
			out = append(out, curvedLoop{edges: edges})
		}
	}
	return out
}

// withoutSlitPairs deletes cyclically adjacent reverse twins until none is left — removing one can
// bring the next pair together, which is how a slit several edges deep unwinds.
func withoutSlitPairs(edges []loopEdge) []loopEdge {
	for {
		k, ok := firstSlitPair(edges)
		if !ok {
			return edges
		}
		edges = withoutCyclicPair(edges, k)
	}
}

// firstSlitPair returns the index of the first edge whose cyclic successor walks it back.
func firstSlitPair(edges []loopEdge) (int, bool) {
	for i := range edges {
		if isReverseTwin(edges[i], edges[(i+1)%len(edges)]) {
			return i, true
		}
	}
	return 0, false
}

// isReverseTwin reports whether two edges are one edge walked both ways. It is EXACT: a slit's two
// sides are the same face's own two traversals of ONE topo edge (the seam), so they carry that edge's
// identity and the swapped parameter span — and a cut of the seam (splitAtSharedRunEnds) cuts both
// traversals at the same parameters, so the pieces pair the same way. Nothing is compared within a
// tolerance here (ADR-0042), and nothing is compared by curve VALUE: `a.curve == b.curve` panicked on
// two value Polylines ("comparing uncomparable type"), and two edges carrying equal curves are still
// two edges. A synthesized edge (no source) is nobody's twin.
func isReverseTwin(a, b loopEdge) bool {
	return a.source != nil && a.source == b.source && a.t0 == b.t1 && a.t1 == b.t0
}

// withoutCyclicPair drops the edges at k and its cyclic successor.
func withoutCyclicPair(edges []loopEdge, k int) []loopEdge {
	if k+1 == len(edges) {
		return append([]loopEdge(nil), edges[1:len(edges)-1]...) // the pair straddles the slice's ends
	}
	return append(append([]loopEdge(nil), edges[:k]...), edges[k+2:]...)
}
