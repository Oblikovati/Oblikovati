// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rebuildDissolved reconstructs body with every dropped vertex gone and every chain of collinear
// edges replaced by its single merged edge, preserving each surviving vertex, edge, and face's
// geometry, sense, lineage, and alias keys.
func rebuildDissolved(body *topo.Body, drop map[*topo.Vertex]bool, chains map[*topo.Edge]*edgeChain) *topo.Body {
	bld := topo.NewBuilder(true, body.Lineage())
	vmap := copyVertices(bld, body, drop)
	emap := copyEdges(bld, body, chains, vmap)
	copyFaces(bld, body, chains, emap)
	return bld.Build()
}

// copyVertices adds every surviving (non-dropped) vertex to the builder, mapping old to new.
func copyVertices(bld *topo.Builder, body *topo.Body, drop map[*topo.Vertex]bool) map[*topo.Vertex]*topo.Vertex {
	vmap := map[*topo.Vertex]*topo.Vertex{}
	for _, v := range body.Vertices() {
		if !drop[v] {
			vmap[v] = bld.AddVertex(v.Point(), v.Lineage())
		}
	}
	return vmap
}

// copyEdges adds one edge per chain (the merged segment) and copies every other edge, mapping each old
// edge to the new edge that carries it.
func copyEdges(bld *topo.Builder, body *topo.Body, chains map[*topo.Edge]*edgeChain, vmap map[*topo.Vertex]*topo.Vertex) map[*topo.Edge]*topo.Edge {
	emap := map[*topo.Edge]*topo.Edge{}
	chainEdge := map[*edgeChain]*topo.Edge{}
	for _, e := range body.Edges() {
		ch, ok := chains[e]
		if !ok {
			emap[e] = bld.AddEdge(e.Geometry(), vmap[e.StartVertex()], vmap[e.EndVertex()], e.Lineage())
			continue
		}
		if chainEdge[ch] == nil {
			chainEdge[ch] = bld.AddEdge(ch.merged, vmap[ch.a], vmap[ch.b], chainLineage(ch))
		}
		emap[e] = chainEdge[ch]
	}
	return emap
}

// copyFaces rebuilds every face on the new edges, collapsing chain runs in each loop and preserving
// surface, sense, lineage, and alias keys.
func copyFaces(bld *topo.Builder, body *topo.Body, chains map[*topo.Edge]*edgeChain, emap map[*topo.Edge]*topo.Edge) {
	for _, f := range body.Faces() {
		specs := make([]topo.LoopSpec, 0, len(f.Loops()))
		for _, l := range f.Loops() {
			specs = append(specs, topo.LoopSpec{Outer: l.IsOuter(), Uses: rebuildLoopUses(l, chains, emap)})
		}
		addFace := bld.AddFace
		if f.Reversed() {
			addFace = bld.AddReversedFace
		}
		nf := addFace(f.Geometry(), f.Lineage(), specs...)
		for _, k := range f.AliasKeys() {
			nf.AddAliasKey(k)
		}
	}
}

// rebuildLoopUses maps a source loop's edge-uses onto the rebuilt edges, collapsing each contiguous
// run of uses on one chain into a single use of the merged edge (oriented by where the run enters).
func rebuildLoopUses(l *topo.Loop, chains map[*topo.Edge]*edgeChain, emap map[*topo.Edge]*topo.Edge) []topo.Use {
	uses := l.EdgeUses()
	n := len(uses)
	start := rotateToChainBoundary(uses, chains)
	out := make([]topo.Use, 0, n)
	for seen := 0; seen < n; {
		u := uses[(start+seen)%n]
		ch := chains[u.Edge()]
		if ch == nil {
			out = append(out, topo.Use{Edge: emap[u.Edge()], Reversed: u.Reversed()})
			seen++
			continue
		}
		out = append(out, topo.Use{Edge: emap[u.Edge()], Reversed: traversalStart(u) != ch.a})
		for seen < n && chains[uses[(start+seen)%n].Edge()] == ch {
			seen++ // consume the rest of this chain's contiguous run
		}
	}
	return out
}

// rotateToChainBoundary returns the index to start iterating a loop's uses so that no chain run is
// split across the wrap point (its first edge is not the continuation of the previous edge's chain).
func rotateToChainBoundary(uses []*topo.EdgeUse, chains map[*topo.Edge]*edgeChain) int {
	n := len(uses)
	for i := 0; i < n; i++ {
		prev := chains[uses[(i-1+n)%n].Edge()]
		cur := chains[uses[i].Edge()]
		if cur == nil || cur != prev {
			return i
		}
	}
	return 0 // a whole loop on one chain cannot happen for a closed collinear boundary
}

// traversalStart is the vertex a loop's edge-use is entered from (the edge start, or its end when the
// use is reversed).
func traversalStart(u *topo.EdgeUse) *topo.Vertex {
	if u.Reversed() {
		return u.Edge().EndVertex()
	}
	return u.Edge().StartVertex()
}

// chainLineage returns a stable lineage for a merged edge: the lineage of the member edge incident to
// the chain's ordered a-end (deterministic across runs, since a is chosen by point order), so the
// merged edge inherits a real operand-edge identity for downstream reference keys.
func chainLineage(ch *edgeChain) topo.Lineage {
	for e := range ch.edges {
		if e.StartVertex() == ch.a || e.EndVertex() == ch.a {
			return e.Lineage()
		}
	}
	return topo.Lineage{} // unreachable: ch.a is an endpoint of exactly one chain edge
}

// pointOnLine reports whether p lies on the infinite line through a and b within tol.
func pointOnLine(a, b, p math.Point3, tol float64) bool {
	ab := a.VectorTo(b)
	l2 := float64(ab.Dot(ab))
	if l2 == 0 {
		return a.DistanceTo(p) <= tol
	}
	t := float64(a.VectorTo(p).Dot(ab)) / l2
	return float64(a.TranslateBy(ab.Scale(math.Scalar(t))).DistanceTo(p)) <= tol
}
