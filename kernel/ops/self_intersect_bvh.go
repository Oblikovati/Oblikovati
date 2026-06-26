// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	"oblikovati.org/math"
)

// triBVH is an axis-aligned bounding-volume hierarchy over a mesh's triangles — the broad-phase index
// that replaces the O(Tₐ·T_b) all-pairs triangle test in self-intersection with ~O((Tₐ+T_b)·log T):
// build it over one face's triangles once, then query each of the other face's triangles against it,
// running the exact Möller test only on the few candidates whose boxes overlap (Oblikovati#1411).
type triBVH struct {
	tris  [][3]math.Point3
	boxes []math.Box
	order []int // triangle indices, partitioned into contiguous node ranges
	nodes []bvhNode
	root  int
}

// bvhNode is a box plus EITHER a leaf range [start, start+count) into order (count > 0) or two children.
type bvhNode struct {
	box          math.Box
	start, count int
	left, right  int
}

// bvhLeafSize bounds a leaf's triangle count: small enough to prune well, large enough that the tree
// stays shallow and the per-node overhead does not dominate.
const bvhLeafSize = 8

// newTriBVH builds the hierarchy over tris by recursive median split on the longest centroid axis.
func newTriBVH(tris [][3]math.Point3) *triBVH {
	boxes := make([]math.Box, len(tris))
	order := make([]int, len(tris))
	for i, t := range tris {
		boxes[i] = math.BoxFromPoints(t[0], t[1], t[2])
		order[i] = i
	}
	b := &triBVH{tris: tris, boxes: boxes, order: order}
	if len(tris) > 0 {
		b.root = b.build(0, len(tris))
	}
	return b
}

// build constructs the subtree over order[start:end] and returns its node index.
func (b *triBVH) build(start, end int) int {
	box := math.EmptyBox()
	for _, ti := range b.order[start:end] {
		box = box.Union(b.boxes[ti])
	}
	idx := len(b.nodes)
	b.nodes = append(b.nodes, bvhNode{box: box, left: -1, right: -1})
	if end-start <= bvhLeafSize {
		b.nodes[idx].start, b.nodes[idx].count = start, end-start
		return idx
	}
	axis := longestBoxAxis(box)
	sub := b.order[start:end]
	sort.Slice(sub, func(i, j int) bool {
		return centroidOnAxis(b.boxes[sub[i]], axis) < centroidOnAxis(b.boxes[sub[j]], axis)
	})
	mid := (start + end) / 2
	b.nodes[idx].left = b.build(start, mid)
	b.nodes[idx].right = b.build(mid, end)
	return idx
}

// query calls hit for each triangle whose box overlaps box, stopping early when hit returns true.
func (b *triBVH) query(box math.Box, hit func(tri int) bool) {
	if len(b.nodes) == 0 {
		return
	}
	b.queryNode(b.root, box, hit)
}

func (b *triBVH) queryNode(ni int, box math.Box, hit func(tri int) bool) bool {
	nd := b.nodes[ni]
	if !nd.box.Intersects(box) {
		return false
	}
	if nd.count > 0 {
		for _, ti := range b.order[nd.start : nd.start+nd.count] {
			if b.boxes[ti].Intersects(box) && hit(ti) {
				return true
			}
		}
		return false
	}
	return b.queryNode(nd.left, box, hit) || b.queryNode(nd.right, box, hit)
}

// longestBoxAxis returns 0/1/2 for the box's widest dimension — the split axis with the best separation.
func longestBoxAxis(box math.Box) int {
	dx, dy, dz := box.Max.X-box.Min.X, box.Max.Y-box.Min.Y, box.Max.Z-box.Min.Z
	if dx >= dy && dx >= dz {
		return 0
	}
	if dy >= dz {
		return 1
	}
	return 2
}

// centroidOnAxis returns a triangle box's centre coordinate on the given axis.
func centroidOnAxis(box math.Box, axis int) float64 {
	switch axis {
	case 0:
		return float64(box.Min.X+box.Max.X) / 2
	case 1:
		return float64(box.Min.Y+box.Max.Y) / 2
	default:
		return float64(box.Min.Z+box.Max.Z) / 2
	}
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
