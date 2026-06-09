// SPDX-License-Identifier: GPL-2.0-only

package subd

import (
	"sort"

	"oblikovati.org/math"
)

// SubdivideN applies n Catmull–Clark refinement steps (n ≤ 0 returns the cage).
func SubdivideN(m Mesh, n int) Mesh {
	for i := 0; i < n; i++ {
		m = Subdivide(m)
	}
	return m
}

// Subdivide performs one Catmull–Clark step: each face yields a face point, each edge
// an edge point (the midpoint when boundary/sharp, else the smooth average), each
// original vertex is repositioned (smooth, crease, or fixed corner), and every face is
// split into one quad per corner. Creases propagate to both halves of a sharp edge.
func Subdivide(m Mesh) Mesh {
	fp := m.facePoints()
	ef := m.edgeFaces()
	V, F := len(m.Verts), len(m.Faces)
	keys := sortedEdgeKeys(ef)
	epIdx := make(map[[2]int]int, len(keys))
	for i, k := range keys {
		epIdx[k] = V + F + i
	}
	verts := m.refinedVerts(fp, ef, keys)
	faces := splitFaces(m.Faces, V, epIdx)
	return Mesh{Verts: verts, Faces: faces, Creases: propagateCreases(m, keys, epIdx)}
}

// refinedVerts lays out the new vertex array: repositioned originals, then face
// points, then edge points (in sorted-key order).
func (m Mesh) refinedVerts(fp []math.Point3, ef map[[2]int][]int, keys [][2]int) []math.Point3 {
	verts := make([]math.Point3, 0, len(m.Verts)+len(fp)+len(keys))
	verts = append(verts, m.vertexPoints(fp, ef)...)
	verts = append(verts, fp...)
	for _, k := range keys {
		verts = append(verts, m.edgePoint(k, ef[k], fp))
	}
	return verts
}

// splitFaces replaces each n-gon with n quads [vertex, next-edge, face, prev-edge].
func splitFaces(faces [][]int, vBase int, epIdx map[[2]int]int) [][]int {
	var out [][]int
	for fi, f := range faces {
		n := len(f)
		for i := 0; i < n; i++ {
			out = append(out, []int{
				f[i],
				epIdx[edgeKey(f[i], f[(i+1)%n])],
				vBase + fi,
				epIdx[edgeKey(f[(i-1+n)%n], f[i])],
			})
		}
	}
	return out
}

// propagateCreases carries each sharp edge's sharpness onto its two subdivided halves.
func propagateCreases(m Mesh, keys [][2]int, epIdx map[[2]int]int) map[[2]int]float64 {
	out := map[[2]int]float64{}
	for _, k := range keys {
		s := m.sharpness(k[0], k[1])
		if s <= 0 {
			continue
		}
		mid := epIdx[k]
		out[edgeKey(k[0], mid)] = s
		out[edgeKey(mid, k[1])] = s
	}
	return out
}

// edgePoint is the smooth average of the edge's endpoints and adjacent face points,
// blended toward the midpoint by the edge's crease sharpness (midpoint on a boundary).
func (m Mesh) edgePoint(k [2]int, faces []int, fp []math.Point3) math.Point3 {
	a, b := m.Verts[k[0]], m.Verts[k[1]]
	mid := a.Midpoint(b)
	if len(faces) < 2 {
		return mid
	}
	smooth := average([]math.Point3{a, b, fp[faces[0]], fp[faces[1]]})
	return lerpP(smooth, mid, clamp01(m.sharpness(k[0], k[1])))
}

// vertexPoints repositions every original vertex per its incidence and creasing.
func (m Mesh) vertexPoints(fp []math.Point3, ef map[[2]int][]int) []math.Point3 {
	inc := m.incidence(ef)
	out := make([]math.Point3, len(m.Verts))
	for v := range m.Verts {
		out[v] = m.vertexPoint(v, inc[v], fp, ef)
	}
	return out
}

// vertexPoint applies the corner (≥3 sharp ⇒ fixed), crease (exactly 2 sharp), or
// smooth (Catmull–Clark) rule depending on how many incident edges are sharp.
func (m Mesh) vertexPoint(v int, in vinc, fp []math.Point3, ef map[[2]int][]int) math.Point3 {
	sharp := m.sharpEdges(in.edges, ef)
	switch {
	case len(sharp) >= 3:
		return m.Verts[v]
	case len(sharp) == 2:
		return creaseVertex(m.Verts[v], m.Verts[other(sharp[0], v)], m.Verts[other(sharp[1], v)])
	default:
		return m.smoothVertex(v, in, fp)
	}
}

// smoothVertex is the Catmull–Clark interior rule (F + 2R + (n−3)P)/n.
func (m Mesh) smoothVertex(v int, in vinc, fp []math.Point3) math.Point3 {
	n := float64(len(in.edges))
	facePts := make([]math.Point3, len(in.faces))
	for i, fi := range in.faces {
		facePts[i] = fp[fi]
	}
	mids := make([]math.Point3, len(in.edges))
	for i, e := range in.edges {
		mids[i] = m.Verts[e[0]].Midpoint(m.Verts[e[1]])
	}
	f, r, p := average(facePts), average(mids), m.Verts[v]
	return math.P3(
		(f.X+2*r.X+(n-3)*p.X)/n,
		(f.Y+2*r.Y+(n-3)*p.Y)/n,
		(f.Z+2*r.Z+(n-3)*p.Z)/n,
	)
}

// creaseVertex is the crease/boundary vertex rule (6P + q1 + q2)/8.
func creaseVertex(p, q1, q2 math.Point3) math.Point3 {
	return math.P3((6*p.X+q1.X+q2.X)/8, (6*p.Y+q1.Y+q2.Y)/8, (6*p.Z+q1.Z+q2.Z)/8)
}

// sharpEdges returns the incident edges that are boundary or fully creased (s ≥ 1).
func (m Mesh) sharpEdges(edges [][2]int, ef map[[2]int][]int) [][2]int {
	var out [][2]int
	for _, e := range edges {
		if len(ef[e]) < 2 || m.sharpness(e[0], e[1]) >= 1 {
			out = append(out, e)
		}
	}
	return out
}

// vinc is a vertex's incidence: the edges and faces touching it.
type vinc struct {
	edges [][2]int
	faces []int
}

// incidence builds per-vertex edge and face incidence lists.
func (m Mesh) incidence(ef map[[2]int][]int) []vinc {
	inc := make([]vinc, len(m.Verts))
	for fi, f := range m.Faces {
		for _, vi := range f {
			inc[vi].faces = append(inc[vi].faces, fi)
		}
	}
	for _, k := range sortedEdgeKeys(ef) {
		inc[k[0]].edges = append(inc[k[0]].edges, k)
		inc[k[1]].edges = append(inc[k[1]].edges, k)
	}
	return inc
}

// sortedEdgeKeys returns the edge keys in deterministic (a,b) order.
func sortedEdgeKeys(ef map[[2]int][]int) [][2]int {
	keys := make([][2]int, 0, len(ef))
	for k := range ef {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	return keys
}

func other(e [2]int, v int) int {
	if e[0] == v {
		return e[1]
	}
	return e[0]
}

func lerpP(a, b math.Point3, t float64) math.Point3 {
	return math.P3(a.X+(b.X-a.X)*t, a.Y+(b.Y-a.Y)*t, a.Z+(b.Z-a.Z)*t)
}

func clamp01(x float64) float64 {
	switch {
	case x < 0:
		return 0
	case x > 1:
		return 1
	default:
		return x
	}
}
