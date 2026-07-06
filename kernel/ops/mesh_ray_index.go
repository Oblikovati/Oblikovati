// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/math"
)

// MeshRayIndex is a triangle BVH over a static mesh for hover-safe ray picking: built once
// (O(n log n)), it answers Nearest in ~O(log n) instead of testing every triangle, so the viewport
// can ray-test a dense placed mesh (e.g. 1.88M triangles, #1776) on hover every frame. Immutable
// after build; safe to cache and share for concurrent reads.
//
//	idx := NewMeshRayIndex(positions, triangles)
//	if tri, t, ok := idx.Nearest(origin, dir); ok { /* triangles[tri] hit at distance t */ }
type MeshRayIndex struct {
	pos   []math.Point3
	tris  [][3]int32
	order []int32 // triangle permutation grouped so each leaf owns a contiguous span
	nodes []meshRayNode
}

// meshRayNode is one BVH node: an AABB plus either two children (left ≥ 0) or a leaf span into order.
type meshRayNode struct {
	box          math.Box
	left, right  int32 // children node indices; left < 0 marks a leaf
	first, count int32 // leaf triangle span order[first:first+count]
}

// meshRayLeafSize is the most triangles a leaf holds; small leaves keep the per-node box test paying.
const meshRayLeafSize = 8

// NewMeshRayIndex builds a BVH over triangles (index triples into positions). The caller owns index
// validity (the placed-mesh path feeds welded, in-range triples); the returned triangle index maps
// 1:1 to the input slice. Returns nil when there are no triangles.
func NewMeshRayIndex(positions []math.Point3, triangles [][3]int32) *MeshRayIndex {
	if len(triangles) == 0 {
		return nil
	}
	idx := &MeshRayIndex{pos: positions, tris: triangles, order: make([]int32, len(triangles))}
	for i := range idx.order {
		idx.order[i] = int32(i)
	}
	idx.build(0, int32(len(idx.order)))
	return idx
}

// build recursively constructs the subtree over order[first:first+count], returning its node index.
func (idx *MeshRayIndex) build(first, count int32) int32 {
	ni := int32(len(idx.nodes))
	idx.nodes = append(idx.nodes, meshRayNode{}) // reserve this node's slot before recursing
	node := meshRayNode{box: idx.spanBox(first, count), left: -1, first: first, count: count}
	if count > meshRayLeafSize {
		if mid := idx.partition(first, count, node.box); mid > first && mid < first+count {
			node.left = idx.build(first, mid-first)
			node.right = idx.build(mid, first+count-mid)
		}
	}
	idx.nodes[ni] = node
	return ni
}

// spanBox is the union AABB of the triangles in order[first:first+count].
func (idx *MeshRayIndex) spanBox(first, count int32) math.Box {
	box := math.EmptyBox()
	for k := first; k < first+count; k++ {
		t := idx.tris[idx.order[k]]
		box = box.ExtendPoint(idx.pos[t[0]]).ExtendPoint(idx.pos[t[1]]).ExtendPoint(idx.pos[t[2]])
	}
	return box
}

// partition splits order[first:first+count] about the centre on the box's widest axis (spatial
// median), returning the split index. A fully one-sided split leaves the caller to make a leaf.
func (idx *MeshRayIndex) partition(first, count int32, box math.Box) int32 {
	axis := meshBoxAxis(box)
	pivot := axisVal(box.Center(), axis)
	i, j := first, first+count-1
	for i <= j {
		for i <= j && idx.centroidAxis(idx.order[i], axis) < pivot {
			i++
		}
		for i <= j && idx.centroidAxis(idx.order[j], axis) >= pivot {
			j--
		}
		if i < j {
			idx.order[i], idx.order[j] = idx.order[j], idx.order[i]
			i++
			j--
		}
	}
	return i
}

// centroidAxis returns triangle tr's centroid coordinate on the given axis.
func (idx *MeshRayIndex) centroidAxis(tr int32, axis int) float64 {
	t := idx.tris[tr]
	a, b, c := idx.pos[t[0]], idx.pos[t[1]], idx.pos[t[2]]
	switch axis {
	case 0:
		return float64(a.X+b.X+c.X) / 3
	case 1:
		return float64(a.Y+b.Y+c.Y) / 3
	default:
		return float64(a.Z+b.Z+c.Z) / 3
	}
}

// Nearest returns the index of the nearest forward-hit triangle along the ray, its distance, and
// whether any was hit — descending the BVH and pruning subtrees the ray enters beyond the best hit.
func (idx *MeshRayIndex) Nearest(origin math.Point3, dir math.Vector3) (tri int, t float64, ok bool) {
	best := stdmath.Inf(1)
	hit := int32(-1)
	idx.nearest(0, origin, dir, &best, &hit)
	if hit < 0 {
		return -1, 0, false
	}
	return int(hit), best, true
}

// nearest walks the subtree at ni, updating best/hit with any closer triangle intersection.
func (idx *MeshRayIndex) nearest(ni int32, origin math.Point3, dir math.Vector3, best *float64, hit *int32) {
	n := &idx.nodes[ni]
	if tEnter, ok := n.box.IntersectsRay(origin, dir); !ok || float64(tEnter) > *best {
		return
	}
	if n.left < 0 {
		idx.testLeaf(n, origin, dir, best, hit)
		return
	}
	idx.nearest(n.left, origin, dir, best, hit)
	idx.nearest(n.right, origin, dir, best, hit)
}

// testLeaf tests every triangle in a leaf, keeping the closest forward hit.
func (idx *MeshRayIndex) testLeaf(n *meshRayNode, origin math.Point3, dir math.Vector3, best *float64, hit *int32) {
	for k := n.first; k < n.first+n.count; k++ {
		tr := idx.tris[idx.order[k]]
		if t, ok := rayTriangleDist(origin, dir, idx.pos[tr[0]], idx.pos[tr[1]], idx.pos[tr[2]]); ok && t < *best {
			*best, *hit = t, idx.order[k]
		}
	}
}

// meshBoxAxis returns the index (0=X,1=Y,2=Z) of the box's longest side.
func meshBoxAxis(b math.Box) int {
	d := b.Diagonal()
	if d.X >= d.Y && d.X >= d.Z {
		return 0
	}
	if d.Y >= d.Z {
		return 1
	}
	return 2
}

// axisVal returns p's coordinate on the given axis.
func axisVal(p math.Point3, axis int) float64 {
	switch axis {
	case 0:
		return float64(p.X)
	case 1:
		return float64(p.Y)
	default:
		return float64(p.Z)
	}
}
