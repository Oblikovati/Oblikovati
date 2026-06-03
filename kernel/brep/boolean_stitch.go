// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// stitch welds the kept sub-faces into a watertight B-rep: coincident vertices merge, one
// shared edge per undirected vertex pair, and a face per sub-face (outer loop oriented CCW
// about its normal, holes CW). The body is a solid when every edge is used exactly twice.
// Lineage is freshly synthesized (operand reference-key survival is a follow-up).
func stitch(faces []subFace) (*topo.Body, error) {
	if len(faces) == 0 {
		return nil, nil
	}
	w := newWelder3()
	var out []builtFace
	edgeUse := map[[2]int]int{}
	for _, sf := range faces {
		rings := [][]int{w.ring(orientRing(sf.outer, sf.normal, true))}
		for _, h := range sf.holes {
			rings = append(rings, w.ring(orientRing(h, sf.normal, false)))
		}
		for _, r := range rings {
			for i := 0; i < len(r); i++ {
				edgeUse[canonEdge(r[i], r[(i+1)%len(r)])]++
			}
		}
		surf, _ := geom.NewPlane(centroid3(sf.outer), sf.normal)
		out = append(out, builtFace{rings, surf})
	}
	return assemble(w.points, out, edgeUse), nil
}

// builtFace is a welded sub-face ready for assembly: its loop rings (vertex indices, outer
// first) and its planar surface.
type builtFace struct {
	rings [][]int
	surf  geom.Plane
}

// assemble builds the topo body from welded vertices, per-face loop rings, and the edge
// use counts (to decide solid vs. surface).
func assemble(verts []math.Point3, faces []builtFace, edgeUse map[[2]int]int) *topo.Body {
	solid := true
	for _, c := range edgeUse {
		if c != 2 {
			solid = false
			break
		}
	}
	bld := topo.NewBuilder(solid, topo.NewLineage(topo.Tok("brep", "body", 0)))
	tv := make([]*topo.Vertex, len(verts))
	for i, p := range verts {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("brep", "vertex", i)))
	}
	edges := buildEdges(bld, verts, tv, edgeUse)
	for fi, f := range faces {
		specs := make([]topo.LoopSpec, len(f.rings))
		for ri, r := range f.rings {
			uses := make([]topo.Use, len(r))
			for i := range r {
				a, b := r[i], r[(i+1)%len(r)]
				uses[i] = topo.Use{Edge: edges[canonEdge(a, b)], Reversed: a > b}
			}
			if ri == 0 {
				specs[ri] = topo.OuterLoop(uses...)
			} else {
				specs[ri] = topo.InnerLoop(uses...)
			}
		}
		bld.AddFace(f.surf, topo.NewLineage(topo.Tok("brep", "face", fi)), specs...)
	}
	return bld.Build()
}

// buildEdges creates one shared topo edge per undirected vertex pair (sorted for stable
// lineage).
func buildEdges(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, edgeUse map[[2]int]int) map[[2]int]*topo.Edge {
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
		edges[k] = bld.AddEdge(geom.NewLineSegment(verts[k[0]], verts[k[1]]), tv[k[0]], tv[k[1]], topo.NewLineage(topo.Tok("brep", "edge", i)))
	}
	return edges
}

// welder3 merges coincident 3D points onto a shared index list.
type welder3 struct {
	index  map[[3]int64]int
	points []math.Point3
}

func newWelder3() *welder3 { return &welder3{index: map[[3]int64]int{}} }

func (w *welder3) add(p math.Point3) int {
	const grid = 1e-6
	k := [3]int64{int64(stdmath.Round(p.X / grid)), int64(stdmath.Round(p.Y / grid)), int64(stdmath.Round(p.Z / grid))}
	if i, ok := w.index[k]; ok {
		return i
	}
	w.index[k] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}

// ring welds a 3D loop to vertex indices, dropping consecutive duplicates.
func (w *welder3) ring(loop []math.Point3) []int {
	var out []int
	for _, p := range loop {
		i := w.add(p)
		if len(out) == 0 || out[len(out)-1] != i {
			out = append(out, i)
		}
	}
	if len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

// orientRing returns the loop wound so its Newell normal points along (outer) or against
// (hole) the face normal.
func orientRing(loop []math.Point3, normal math.Vector3, outer bool) []math.Point3 {
	aligned := newell3(loop).Dot(normal) > 0
	if aligned == outer {
		return loop
	}
	return reverseRing(loop)
}

// newell3 returns a 3D loop's (unnormalized) Newell normal.
func newell3(loop []math.Point3) math.Vector3 {
	var nx, ny, nz float64
	n := len(loop)
	for i := 0; i < n; i++ {
		c, d := loop[i], loop[(i+1)%n]
		nx += (c.Y - d.Y) * (c.Z + d.Z)
		ny += (c.Z - d.Z) * (c.X + d.X)
		nz += (c.X - d.X) * (c.Y + d.Y)
	}
	return math.V3(nx, ny, nz)
}

func centroid3(loop []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range loop {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(loop))
	return math.P3(sx/n, sy/n, sz/n)
}
