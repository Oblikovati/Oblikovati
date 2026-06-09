// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rebuildWithPlanes clones a planar solid's topology, replacing each face's surface with
// planeOf(f) and moving each vertex to the meeting point of its adjacent faces' new planes
// (the least-squares intersection — exact for a 3-face corner). The combinatorial structure
// (faces, loops, edges) is preserved, so the result stays a valid solid as long as the moved
// planes don't invert the topology (a modest move). It is the shared engine behind the local
// face operations — shell (offset kept faces inward), move/offset face, draft — which differ
// only in how planeOf changes the selected faces' planes.
func rebuildWithPlanes(solid *topo.Body, tag string, planeOf func(*topo.Face) geom.Plane) *topo.Body {
	vf := vertexFaceMap(solid)
	planes := make(map[uint64]geom.Plane, len(solid.Faces()))
	for _, f := range solid.Faces() {
		planes[f.ID()] = planeOf(f)
	}
	lin := topo.NewLineage(topo.Tok(tag, "body", 0))
	bld := topo.NewBuilder(true, lin)
	nv := make(map[uint64]*topo.Vertex, len(solid.Vertices()))
	for i, v := range solid.Vertices() {
		nv[v.ID()] = bld.AddVertex(vertexAtPlanes(v, vf[v.ID()], planes), topo.NewLineage(topo.Tok(tag, "v", i)))
	}
	ne := make(map[uint64]*topo.Edge, len(solid.Edges()))
	for i, e := range solid.Edges() {
		a, b := nv[e.StartVertex().ID()], nv[e.EndVertex().ID()]
		ne[e.ID()] = bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, topo.NewLineage(topo.Tok(tag, "e", i)))
	}
	for i, f := range solid.Faces() {
		bld.AddFace(planes[f.ID()], topo.NewLineage(topo.Tok(tag, "f", i)), cloneLoops(f, ne)...)
	}
	return bld.Build()
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
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				a[i][j] += nv[i] * nv[j]
			}
			b[i] += nv[i] * d
		}
	}
	x, ok := solve3(a, b)
	if !ok {
		return v.Point()
	}
	return math.P3(x[0], x[1], x[2])
}

// cloneLoops rebuilds a face's loop specs against the rebuilt-body edges, preserving each
// edge use's direction and the outer/inner role.
func cloneLoops(f *topo.Face, ne map[uint64]*topo.Edge) []topo.LoopSpec {
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

// vertexFaceMap returns, per vertex ID, the faces meeting at that vertex.
func vertexFaceMap(solid *topo.Body) map[uint64][]*topo.Face {
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

// singularSolveTol is the magnitude below which a determinant or a ray/line·plane denominator
// is treated as zero — the linear solve is singular or the line is parallel to the plane. It
// is below the linear DefaultTolerance because it bounds a product of (roughly unit) direction
// terms, not a length.
const singularSolveTol = 1e-12

// solve3 solves the 3×3 system a·x = b by Cramer's rule, ok=false when a is singular.
func solve3(a [3][3]float64, b [3]float64) ([3]float64, bool) {
	det := det3(a)
	if det < singularSolveTol && det > -singularSolveTol {
		return [3]float64{}, false
	}
	var x [3]float64
	for c := 0; c < 3; c++ {
		m := a
		for r := 0; r < 3; r++ {
			m[r][c] = b[r]
		}
		x[c] = det3(m) / det
	}
	return x, true
}

// det3 returns the determinant of a 3×3 matrix.
func det3(m [3][3]float64) float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}
