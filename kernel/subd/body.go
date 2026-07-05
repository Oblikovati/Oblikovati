// SPDX-License-Identifier: GPL-2.0-only

package subd

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// ToBody converts a (refined) control cage into a B-rep body: one shared vertex per
// cage vertex, one shared edge per cage edge, and one planar face per cage face
// (fitted through the face centroid with the Newell normal). A closed cage — every
// edge used by two faces — becomes a solid; an open cage a surface body. Faces are
// approximated as planar in phase A; the exact bicubic limit surface is a NURBS phase.
func ToBody(m Mesh, feat string) *topo.Body {
	// edgeFaces is O(edges) to build; compute the adjacency once and share it between the
	// closed-cage test and the shared-edge build rather than rebuilding it per call — the
	// second-largest allocation source on a dense mesh import (#1765).
	ef := m.edgeFaces()
	bld := topo.NewBuilder(cageClosed(ef), topo.NewLineage(topo.Tok(feat, "body", 0)))
	verts := make([]*topo.Vertex, len(m.Verts))
	for i, p := range m.Verts {
		verts[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	edges := buildBodyEdges(m, ef, bld, verts, feat)
	for fi, f := range m.Faces {
		bld.AddFace(facePlane(m, f), topo.NewLineage(topo.Tok(feat, "face", fi)), faceLoop(f, edges))
	}
	return bld.Build()
}

// buildBodyEdges creates one shared topo edge per undirected cage edge (sorted-key
// order for stable lineage).
func buildBodyEdges(m Mesh, ef map[[2]int][]int, bld *topo.Builder, verts []*topo.Vertex, feat string) map[[2]int]*topo.Edge {
	keys := sortedEdgeKeys(ef)
	edges := make(map[[2]int]*topo.Edge, len(keys))
	for i, k := range keys {
		seg := geom.NewLineSegment(m.Verts[k[0]], m.Verts[k[1]])
		edges[k] = bld.AddEdge(seg, verts[k[0]], verts[k[1]], topo.NewLineage(topo.Tok(feat, "edge", i)))
	}
	return edges
}

// faceLoop builds a face's outer loop, marking a use reversed when its directed cage
// edge runs against the canonical (min,max) stored edge.
func faceLoop(f []int, edges map[[2]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, len(f))
	for i := range f {
		a, b := f[i], f[(i+1)%len(f)]
		uses[i] = topo.Use{Edge: edges[edgeKey(a, b)], Reversed: a > b}
	}
	return topo.OuterLoop(uses...)
}

// cageClosed reports whether every cage edge is shared by exactly two faces (a watertight
// solid), from a precomputed edge→faces adjacency so ToBody builds it only once (#1765).
// An empty cage (no edges) is not closed.
func cageClosed(ef map[[2]int][]int) bool {
	if len(ef) == 0 {
		return false
	}
	for _, fs := range ef {
		if len(fs) != 2 {
			return false
		}
	}
	return true
}

// facePlane fits a plane through a face's centroid with its Newell normal, falling
// back to +Z for a degenerate (zero-area) face.
func facePlane(m Mesh, f []int) geom.Surface {
	pts := make([]math.Point3, len(f))
	for i, vi := range f {
		pts[i] = m.Verts[vi]
	}
	p, err := geom.NewPlane(average(pts), newellNormal(m, f))
	if err != nil {
		p, _ = geom.NewPlane(average(pts), math.V3(0, 0, 1))
	}
	return p
}

// newellNormal computes a face's normal via Newell's method (robust for the
// non-planar quads that subdivision produces).
func newellNormal(m Mesh, f []int) math.Vector3 {
	var nx, ny, nz float64
	n := len(f)
	for i := 0; i < n; i++ {
		cur, nxt := m.Verts[f[i]], m.Verts[f[(i+1)%n]]
		nx += (cur.Y - nxt.Y) * (cur.Z + nxt.Z)
		ny += (cur.Z - nxt.Z) * (cur.X + nxt.X)
		nz += (cur.X - nxt.X) * (cur.Y + nxt.Y)
	}
	return math.V3(nx, ny, nz)
}
