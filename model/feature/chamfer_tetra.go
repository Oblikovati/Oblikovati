// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Chamfer — the TETRA TOOL BUILDER (M48 #2232 split of chamfer.go). Builds the four-vertex, four-face
// tetrahedron body used as a corner-cut tool: its six edges are keyed by ascending vertex-index pair
// so the four triangular faces share them with consistent traversal (C6-a: guarded edge reuse). The
// flat-corner blend that drives it lives in chamfer_corner.go.

// tetraEdges holds the six edges of a tetrahedron keyed by their ascending vertex-index
// pair, so the four triangular faces can share them with the right traversal direction.
type tetraEdges map[[2]int]*topo.Edge

// use returns the oriented use that traverses the tetra edge from vertex i to vertex j.
func (te tetraEdges) use(i, j int) topo.Use {
	if i < j {
		return topo.Fwd(te[[2]int{i, j}])
	}
	return topo.Rev(te[[2]int{j, i}])
}

// buildTetra assembles a solid tetrahedron from four points as a boolean cut tool. Each
// triangular face is planar with its plane oriented outward (away from the opposite
// vertex) and its loop wound to match, so the body is a valid closed solid the boolean can
// subtract. Vertex 0 is the apex; 1..3 the base.
func buildTetra(p [4]math.Point3, feat string) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	var v [4]*topo.Vertex
	for i := range p {
		v[i] = bld.AddVertex(p[i], topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	edges := newTetraEdges(bld, p, v, feat)
	faces := [4][3]int{{1, 2, 3}, {0, 2, 3}, {0, 1, 3}, {0, 1, 2}}
	opposite := [4]int{0, 1, 2, 3}
	for fi, tri := range faces {
		addTetraFace(bld, p, edges, tri, opposite[fi], feat, fi)
	}
	return bld.Build()
}

// newTetraEdges builds the six line-segment edges of the tetrahedron, each stored under
// its ascending vertex-index pair.
func newTetraEdges(bld *topo.Builder, p [4]math.Point3, v [4]*topo.Vertex, feat string) tetraEdges {
	pairs := [6][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}
	te := make(tetraEdges, 6)
	for k, pr := range pairs {
		te[pr] = bld.AddEdge(geom.NewLineSegment(p[pr[0]], p[pr[1]]), v[pr[0]], v[pr[1]], topo.NewLineage(topo.Tok(feat, "edge", k)))
	}
	return te
}

// addTetraFace adds the triangular face through corner triple tri, flipping the traversal
// so its plane normal points away from the opposite vertex (outward) and winding the loop
// to match. A near-degenerate triangle is dropped (cornerTetra already filters flat
// tetras).
func addTetraFace(bld *topo.Builder, p [4]math.Point3, te tetraEdges, tri [3]int, opp int, feat string, fi int) {
	a, b, c := tri[0], tri[1], tri[2]
	n := p[a].VectorTo(p[b]).Cross(p[a].VectorTo(p[c]))
	if n.Dot(p[a].VectorTo(p[opp])) > 0 { // points toward the interior vertex → flip
		b, c = c, b
		n = n.Negate()
	}
	unit, err := math.UnitVector3FromVector(n)
	if err != nil {
		return
	}
	surf, _ := geom.NewPlane(p[a], unit.AsVector())
	loop := topo.OuterLoop(te.use(a, b), te.use(b, c), te.use(c, a))
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", fi)), loop)
}
