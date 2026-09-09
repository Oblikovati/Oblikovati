// SPDX-License-Identifier: GPL-2.0-only

package topo

// TopologySignature is everything about a body that a TOPOLOGICAL validity verdict reads: whether it
// is a solid, how many vertices, edges, faces and loops it has, and how its edges are used. Two
// bodies whose signatures agree cannot differ in any way ops.ValidateTopology can see, and two
// readings of ONE body that disagree prove it changed between them.
//
// It exists because a body is documented as immutable after construction and is not enforced to be:
// a builder that reuses an existing edge appends an edge-use to it, which leaves the body that edge
// came from non-manifold without changing the pointer to it. The feature engine's Validate
// post-condition skipped exactly on that pointer, so such a body escaped it (#3524).
type TopologySignature struct {
	Solid                         bool
	Vertices, Edges, Faces, Loops int
	// EdgeUses folds every edge's use count and use orientations, in the body's own edge order. The
	// counts alone miss a mutation that rewires an edge without adding or removing an entity.
	EdgeUses uint64
}

// TopologySignature computes this body's signature. It is O(vertices + edges + faces) and allocates
// nothing, which is what lets a caller take one before every operation.
//
// Example:
//
//	before := body.TopologySignature()
//	run(body)
//	if body.TopologySignature() != before { /* the operation changed it in place */ }
func (b *Body) TopologySignature() TopologySignature {
	faces, edges, verts := b.entityLists()
	sig := TopologySignature{
		Solid: b.solid, Faces: len(faces), Edges: len(edges), Vertices: len(verts),
		EdgeUses: fnvOffset64,
	}
	for _, f := range faces {
		sig.Loops += len(f.loops)
	}
	for _, e := range edges {
		sig.EdgeUses = foldEdgeUses(sig.EdgeUses, e)
	}
	return sig
}

// entityLists returns the body's face, edge and vertex lists WITHOUT the defensive copies the public
// accessors make. The caller must only read them.
func (b *Body) entityLists() ([]*Face, []*Edge, []*Vertex) {
	if b.derived {
		return b.cachedFaces, b.cachedEdges, b.cachedVertices
	}
	faces := b.deriveFaces()
	edges := deriveEdgesFrom(faces)
	return faces, edges, deriveVerticesFrom(edges)
}

// foldEdgeUses mixes one edge's use count and its uses' orientations into the running fold.
func foldEdgeUses(h uint64, e *Edge) uint64 {
	h = fnvMix(h, uint64(len(e.uses)))
	for _, u := range e.uses {
		h = fnvMix(h, useOrientationBit(u))
	}
	return h
}

// useOrientationBit is the one bit an edge-use contributes beyond its existence.
func useOrientationBit(u *EdgeUse) uint64 {
	if u.reversed {
		return 1
	}
	return 0
}

// FNV-1a over 64-bit words: a fold with no ordering surprises and no allocation, so the signature is
// byte-identical on every platform.
const (
	fnvOffset64 uint64 = 14695981039346656037
	fnvPrime64  uint64 = 1099511628211
)

func fnvMix(h, v uint64) uint64 { return (h ^ v) * fnvPrime64 }
