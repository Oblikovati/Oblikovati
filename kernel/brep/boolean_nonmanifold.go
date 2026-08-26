// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/topo"
)

// Edge-use collection and loop construction for the planar-boolean stitch. The radial resolution of
// a non-manifold (tangent-contact) edge into manifold edge-groups lives in boolean_radial_edge.go;
// naming and construction of the resulting topo entities live in boolean_mint.go. This file holds
// the two ends the stitch talks to directly: gathering the directed half-edge uses that feed the
// sew, and building each face's loops from the minted edges.

// loopEdgeUse is one directed use of an undirected vertex pair by a face loop: the face/ring/pos
// that locate it, and whether the ring traverses the pair high→low (reversed, vs the canonical
// low→high orientation used to key the edge).
type loopEdgeUse struct {
	face, ring, pos int
	reversed        bool
	fromB           bool // operand of the using face, for cross-operand fusion of tangent contacts
}

// collectEdgeUses groups every directed loop edge by its canonical (low,high) vertex pair, so
// the stitch can resolve each pair's uses into shared topo edges.
func collectEdgeUses(faces []builtFace) map[[2]int][]loopEdgeUse {
	uses := map[[2]int][]loopEdgeUse{}
	for fi, f := range faces {
		for ri, r := range f.rings {
			n := len(r)
			for i := range n {
				a, b := r[i], r[(i+1)%n]
				key := canonEdge(a, b)
				uses[key] = append(uses[key], loopEdgeUse{face: fi, ring: ri, pos: i, reversed: a > b, fromB: f.fromB})
			}
		}
	}
	return uses
}

// allUsesPaired reports whether every vertex pair is used an even, non-zero number of times —
// the combinatorial closed-shell test. Exactly two is the manifold norm; more (a tangent
// contact) still closes once resolveEdgeUses splits it into twice-used edges, so even is the
// right test; an odd or zero count cannot close.
func allUsesPaired(uses map[[2]int][]loopEdgeUse) bool {
	for _, u := range uses {
		if len(u) == 0 || len(u)%2 != 0 {
			return false
		}
	}
	return true
}

// loopSpecResolved builds a face loop from a ring of vertex indices, resolving each directed
// edge to the topo edge assigned to that exact use (so coincident, tangent-contact edges stay
// distinct), reversed when the ring traverses the pair high→low.
func loopSpecResolved(outer bool, ring []int, fi, ri int, useEdge map[[3]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, len(ring))
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		uses[i] = topo.Use{Edge: useEdge[[3]int{fi, ri, i}], Reversed: a > b}
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}
