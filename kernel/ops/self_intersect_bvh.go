// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// triBVH is an axis-aligned bounding-volume hierarchy over a mesh's triangles — the broad-phase index
// that replaces the O(Tₐ·T_b) all-pairs triangle test in self-intersection with ~O((Tₐ+T_b)·log T):
// build it over one face's triangles once, then query each of the other face's triangles against it,
// running the exact Möller test only on the few candidates whose boxes overlap (Oblikovati#1411).
// The hierarchy itself was generalized into geom.BoxTree so the planar boolean's face-pair culling
// (kernel/brep, which kernel/ops imports and so cannot be imported back) shares it (#1607); this
// type keeps the triangle soup next to the tree for the narrow phase.
type triBVH struct {
	tris [][3]math.Point3
	tree *geom.BoxTree
}

// newTriBVH builds the hierarchy over tris (recursive median split on the longest centroid axis,
// see geom.NewBoxTree).
func newTriBVH(tris [][3]math.Point3) *triBVH {
	boxes := make([]math.Box, len(tris))
	for i, t := range tris {
		boxes[i] = math.BoxFromPoints(t[0], t[1], t[2])
	}
	return &triBVH{tris: tris, tree: geom.NewBoxTree(boxes)}
}

// query calls hit for each triangle whose box overlaps box, stopping early when hit returns true.
func (b *triBVH) query(box math.Box, hit func(tri int) bool) {
	b.tree.Query(box, hit)
}

// meshTriangles returns a mesh's triangles as explicit point triples, the input the BVH and the Möller
// test both consume.
func meshTriangles(m *Mesh) [][3]math.Point3 {
	out := make([][3]math.Point3, 0, len(m.Indices)/3)
	for i := 0; i+2 < len(m.Indices); i += 3 {
		out = append(out, meshTriangle(m, i))
	}
	return out
}
