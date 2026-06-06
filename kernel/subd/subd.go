// SPDX-License-Identifier: GPL-2.0-only

// Package subd is the sub-division-surface kernel for free-form modeling (M10-F03):
// a control mesh (cage) of vertices and polygon faces with optional per-edge creases,
// Catmull–Clark refinement that smooths the cage toward its limit surface (creased and
// boundary edges stay sharp), and conversion of a refined cage to a B-rep body. It is
// decoupled from the analytic kernel and the 2D solver, per the milestone boundary.
package subd

import (
	"oblikovati/math"
)

// Mesh is a sub-D control cage: vertices, polygon faces (ordered vertex indices,
// consistently wound), and per-edge crease sharpness in [0,1] (1 = fully sharp).
type Mesh struct {
	Verts   []math.Point3
	Faces   [][]int
	Creases map[[2]int]float64
}

// edgeKey returns the undirected key (min,max) for an edge between two vertices.
func edgeKey(a, b int) [2]int {
	if a <= b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// sharpness returns the crease sharpness of an edge (0 when smooth/absent).
func (m Mesh) sharpness(a, b int) float64 {
	if m.Creases == nil {
		return 0
	}
	return m.Creases[edgeKey(a, b)]
}

// EdgeList returns the cage's undirected edges in deterministic order — the set the
// free-form UI exposes for selection/creasing.
func (m Mesh) EdgeList() [][2]int { return sortedEdgeKeys(m.edgeFaces()) }

// SetCrease sets an edge's crease sharpness (clamped to [0,1]; 0 removes it),
// allocating the crease map on first use.
func (m *Mesh) SetCrease(a, b int, sharpness float64) {
	if m.Creases == nil {
		m.Creases = map[[2]int]float64{}
	}
	k := edgeKey(a, b)
	if sharpness <= 0 {
		delete(m.Creases, k)
		return
	}
	m.Creases[k] = clamp01(sharpness)
}

// Clone returns a deep copy of the cage so edits do not alias a recomputed body.
func (m Mesh) Clone() Mesh {
	verts := append([]math.Point3(nil), m.Verts...)
	faces := make([][]int, len(m.Faces))
	for i, f := range m.Faces {
		faces[i] = append([]int(nil), f...)
	}
	creases := map[[2]int]float64{}
	for k, v := range m.Creases {
		creases[k] = v
	}
	return Mesh{Verts: verts, Faces: faces, Creases: creases}
}

// edgeFaces maps each undirected edge to the faces using it (one ⇒ boundary).
func (m Mesh) edgeFaces() map[[2]int][]int {
	out := map[[2]int][]int{}
	for fi, f := range m.Faces {
		for i := range f {
			k := edgeKey(f[i], f[(i+1)%len(f)])
			out[k] = append(out[k], fi)
		}
	}
	return out
}

// facePoints returns the centroid of each face.
func (m Mesh) facePoints() []math.Point3 {
	out := make([]math.Point3, len(m.Faces))
	for fi, f := range m.Faces {
		pts := make([]math.Point3, len(f))
		for i, vi := range f {
			pts[i] = m.Verts[vi]
		}
		out[fi] = average(pts)
	}
	return out
}

// average returns the centroid of a set of points.
func average(pts []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range pts {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(pts))
	return math.P3(sx/n, sy/n, sz/n)
}
