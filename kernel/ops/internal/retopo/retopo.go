// SPDX-License-Identifier: GPL-2.0-only

package retopo

import (
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// RebuildWithPlanes clones a planar solid's topology, replacing each face's surface with
// planeOf(f) and moving each vertex to the meeting point of its adjacent faces' new planes
// (the least-squares intersection — exact for a 3-face corner). The combinatorial structure
// (faces, loops, edges) is preserved, so the result stays a valid solid as long as the moved
// planes don't invert the topology (a modest move). It is the shared engine behind the local
// face operations — shell (offset kept faces inward), move/offset face, draft — which differ
// only in how planeOf changes the selected faces' planes.
// keepIdentity (ADR-0043): because the rebuild is a 1:1 clone of the combinatorial structure, a
// direct-result op (draft, move/offset/rotate-face, replace-face) preserves each face/edge/vertex's
// ORIGINAL lineage — the geometry moves but identity does not, so a selection on a drafted face
// survives. A throwaway tool (shell's cavity, fed to the boolean) instead takes fresh ordinal names
// under tag, so its faces don't share lineages with the target and confuse the boolean's face-pair
// edge naming.
func RebuildWithPlanes(solid *topo.Body, tag string, keepIdentity bool, planeOf func(*topo.Face) geom.Plane) *topo.Body {
	vf := VertexFaceMap(solid)
	planes := make(map[uint64]geom.Plane, len(solid.Faces()))
	for _, f := range solid.Faces() {
		planes[f.ID()] = planeOf(f)
	}
	lin := topo.NewLineage(topo.Tok(tag, "body", 0))
	bld := topo.NewBuilder(true, lin)
	nv := make(map[uint64]*topo.Vertex, len(solid.Vertices()))
	for i, v := range solid.Vertices() {
		nv[v.ID()] = bld.AddVertex(vertexAtPlanes(v, vf[v.ID()], planes), CloneName(keepIdentity, v.Lineage(), tag, "v", i))
	}
	ne := make(map[uint64]*topo.Edge, len(solid.Edges()))
	for i, e := range solid.Edges() {
		a, b := nv[e.StartVertex().ID()], nv[e.EndVertex().ID()]
		ne[e.ID()] = bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, CloneName(keepIdentity, e.Lineage(), tag, "e", i))
	}
	for i, f := range solid.Faces() {
		bld.AddFace(planes[f.ID()], CloneName(keepIdentity, f.Lineage(), tag, "f", i), CloneLoops(f, ne)...)
	}
	return bld.Build()
}

// CloneName returns orig when a 1:1 rebuild keeps identity, else a fresh build-order ordinal under
// tag — the per-entity naming choice RebuildWithPlanes makes (ADR-0043).
func CloneName(keepIdentity bool, orig topo.Lineage, tag, role string, i int) topo.Lineage {
	if keepIdentity {
		return orig
	}
	return topo.NewLineage(topo.Tok(tag, role, i))
}

// vertexAtPlanes returns where a vertex lands as the least-squares meeting point of its
// adjacent faces' planes (exact for a 3-face corner). Falls back to the original position if
// the planes are degenerate (parallel normals).
func vertexAtPlanes(v *topo.Vertex, faces []*topo.Face, planes map[uint64]geom.Plane) math.Point3 {
	var a [3][3]float64
	var b [3]float64
	for _, f := range faces {
		pl := planes[f.ID()]
		n := pl.Normal()
		d := n.Dot(pl.Origin.AsVector())
		nv := [3]float64{n.X, n.Y, n.Z}
		for i := range 3 {
			for j := range 3 {
				a[i][j] += nv[i] * nv[j]
			}
			b[i] += nv[i] * d
		}
	}
	x, ok := probe.Solve3(a, b)
	if !ok {
		return v.Point()
	}
	return math.P3(x[0], x[1], x[2])
}

// CloneLoops rebuilds a face's loop specs against the rebuilt-body edges, preserving each
// edge use's direction and the outer/inner role.
func CloneLoops(f *topo.Face, ne map[uint64]*topo.Edge) []topo.LoopSpec {
	specs := make([]topo.LoopSpec, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		uses := make([]topo.Use, 0, len(l.EdgeUses()))
		for _, u := range l.EdgeUses() {
			uses = append(uses, topo.Use{Edge: ne[u.Edge().ID()], Reversed: u.Reversed()})
		}
		if l.IsOuter() {
			specs = append(specs, topo.OuterLoop(uses...))
		} else {
			specs = append(specs, topo.InnerLoop(uses...))
		}
	}
	return specs
}

// VertexFaceMap returns, per vertex ID, the faces meeting at that vertex.
func VertexFaceMap(solid *topo.Body) map[uint64][]*topo.Face {
	m := map[uint64][]*topo.Face{}
	seen := map[[2]uint64]bool{}
	for _, f := range solid.Faces() {
		for _, e := range f.Edges() {
			for _, v := range e.Vertices() {
				if key := [2]uint64{v.ID(), f.ID()}; !seen[key] {
					seen[key] = true
					m[v.ID()] = append(m[v.ID()], f)
				}
			}
		}
	}
	return m
}

// FullDomainBody builds a one-face surface body over a B-spline surface's whole domain, bounded by
// its four boundary iso-curves.
func FullDomainBody(s geom.BSplineSurface, feat string) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	c00 := bld.AddVertex(s.PointAt(0, 0), topo.NewLineage(topo.Tok(feat, "v", 0)))
	c10 := bld.AddVertex(s.PointAt(1, 0), topo.NewLineage(topo.Tok(feat, "v", 1)))
	c11 := bld.AddVertex(s.PointAt(1, 1), topo.NewLineage(topo.Tok(feat, "v", 2)))
	c01 := bld.AddVertex(s.PointAt(0, 1), topo.NewLineage(topo.Tok(feat, "v", 3)))
	eBottom := bld.AddEdge(s.BoundaryVIso(false), c00, c10, topo.NewLineage(topo.Tok(feat, "e", 0))) // v=0, along u
	eRight := bld.AddEdge(s.BoundaryUIso(true), c10, c11, topo.NewLineage(topo.Tok(feat, "e", 1)))   // u=1, along v
	eTop := bld.AddEdge(s.BoundaryVIso(true), c01, c11, topo.NewLineage(topo.Tok(feat, "e", 2)))     // v=1, along u
	eLeft := bld.AddEdge(s.BoundaryUIso(false), c00, c01, topo.NewLineage(topo.Tok(feat, "e", 3)))   // u=0, along v
	loop := topo.OuterLoop(topo.Fwd(eBottom), topo.Fwd(eRight), topo.Rev(eTop), topo.Rev(eLeft))
	bld.AddFace(s, topo.NewLineage(topo.Tok(feat, "face", 0)), loop)
	return bld.Build()
}

// The planar-loop soup builder: a set of point rings with their plane normals welded into a
// solid. Face deletion, hole capping and thickening all produce that soup and all need the
// same body out of it, so the builder lives here rather than in whichever operation reached
// for it first.

// PlanarLoop is a surviving face as 3D point rings (outer first) plus its normal, ready for
// welding into a body.
type PlanarLoop struct {
	Normal  math.Vector3
	Rings   [][]math.Point3
	Lineage topo.Lineage // provenance: the surviving original face this loop came from (ADR-0043)
}

// BuildSolidFromLoops welds coincident loop points into a body, dropping the degenerate
// (zero-length) edges that a heal collapses. One shared edge per undirected vertex pair; a
// closed body (every edge used twice) is a solid.
func BuildSolidFromLoops(faces []PlanarLoop) *topo.Body {
	var pts []math.Point3
	for _, f := range faces {
		for _, r := range f.Rings {
			pts = append(pts, r...)
		}
	}
	w := mesh.NewPointWelder(tol.ForPoints(pts).Weld())
	rings := make([][][]int, len(faces))
	for i, f := range faces {
		for _, r := range f.Rings {
			rings[i] = append(rings[i], DropRepeats(w.WeldRing(r)))
		}
	}
	edgeUse := map[[2]int]int{}
	for _, fr := range rings {
		for _, r := range fr {
			for k := range r {
				edgeUse[probe.Canon2(r[k], r[(k+1)%len(r)])]++
			}
		}
	}
	return assembleLoops(w.Points, faces, rings, edgeUse)
}

// assembleLoops builds the topo body from welded points, per-face loop index rings, and the
// edge-use counts (every edge twice ⇒ solid).
func assembleLoops(verts []math.Point3, faces []PlanarLoop, rings [][][]int, edgeUse map[[2]int]int) *topo.Body {
	solid := true
	for _, c := range edgeUse {
		if c != 2 {
			solid = false
		}
	}
	bld := topo.NewBuilder(solid, topo.NewLineage(topo.Tok("delface", "body", 0)))
	tv := make([]*topo.Vertex, len(verts))
	for i, p := range verts {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("delface", "v", i)))
	}
	edges := buildSharedEdges(bld, verts, tv, edgeUse)
	for fi, fr := range rings {
		specs := make([]topo.LoopSpec, 0, len(fr))
		for ri, r := range fr {
			specs = append(specs, indexLoop(ri == 0, r, edges))
		}
		surf, _ := geom.NewPlane(probe.CentroidPts(faces[fi].Rings[0]), faces[fi].Normal)
		// Provenance (ADR-0043): a surviving face keeps its original identity; fall back to the
		// build-order name only if a loop arrived without one.
		lin := faces[fi].Lineage
		if len(lin.Key()) == 0 {
			lin = topo.NewLineage(topo.Tok("delface", "f", fi))
		}
		bld.AddFace(surf, lin, specs...)
	}
	return bld.Build()
}

// dropRepeats removes consecutive (and wrap-around) duplicate indices — the degenerate edges
// a heal collapses.
func DropRepeats(r []int) []int {
	var out []int
	for _, x := range r {
		if len(out) == 0 || out[len(out)-1] != x {
			out = append(out, x)
		}
	}
	for len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

// buildSharedEdges creates one shared edge per undirected vertex pair (sorted for stable
// lineage).
func buildSharedEdges(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, edgeUse map[[2]int]int) map[[2]int]*topo.Edge {
	keys := make([][2]int, 0, len(edgeUse))
	for k := range edgeUse {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	edges := make(map[[2]int]*topo.Edge, len(keys))
	for i, k := range keys {
		edges[k] = bld.AddEdge(geom.NewLineSegment(verts[k[0]], verts[k[1]]), tv[k[0]], tv[k[1]], topo.NewLineage(topo.Tok("delface", "e", i)))
	}
	return edges
}

// indexLoop builds a face loop from a ring of welded vertex indices.
func indexLoop(outer bool, ring []int, edges map[[2]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, len(ring))
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		uses[i] = topo.Use{Edge: edges[probe.Canon2(a, b)], Reversed: a > b}
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}
