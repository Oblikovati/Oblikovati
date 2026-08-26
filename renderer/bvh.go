// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "sort"

// BVHNode is one node of a flat, compact BVH (M45-F01 PBI-334, ADR-0053): the classic
// "left child at LeftFirst, right child at LeftFirst+1" binary-tree-in-an-array layout
// (Bikker, "How to Build a BVH"), chosen for a trivial, GPU-buffer-friendly upload — no
// pointers, just two contiguous node indices. TriCount==0 marks an internal node
// (LeftFirst is its left child's node index); TriCount>0 marks a leaf (LeftFirst is the
// index of its first triangle in the BVH's own REORDERED triangle order, not the
// caller's original order — see [BuildBVH]'s TriangleOrder).
type BVHNode struct {
	Min, Max  [3]float32
	LeftFirst int32
	TriCount  int32
}

// BVH is a median-split bounding volume hierarchy over a triangle list — the software
// Intersector backend's spatial acceleration structure, built once per body (mirroring
// the hardware backend's BLAS: a preprocessing step, not per-ray work). Traversal is the
// compute shader's job (head/internal/native/shaders/swtrace.comp); this package only
// builds the tree, so construction correctness is CPU-testable without a GPU (ADR-0014).
type BVH struct {
	Nodes []BVHNode
	// TriangleOrder maps a leaf's LeftFirst..LeftFirst+TriCount-1 slots back to indices
	// into the ORIGINAL triangle slice BuildBVH was given — the compute shader looks up
	// hit triangles by this order, then reports the caller's own indices back out.
	TriangleOrder []int32
}

// bvhLeafThreshold caps a leaf at this many triangles before BuildBVH keeps splitting —
// small enough to bound a compute shader's per-leaf linear scan, large enough that a
// median split doesn't produce a degenerate one-triangle-per-node tree for small scenes.
const bvhLeafThreshold = 4

// BuildBVH constructs a median-split BVH over triangles: at each node, split along the
// longest axis of the node's triangle centroids' bounding box, at the median centroid
// (so each split is exactly balanced — simpler and more predictable than a
// surface-area-heuristic split, and explicitly named as an acceptable alternative in
// PBI-334's own scope). Returns nil for an empty input.
func BuildBVH(triangles []Triangle) *BVH {
	if len(triangles) == 0 {
		return nil
	}
	order := make([]int32, len(triangles))
	for i := range order {
		order[i] = int32(i)
	}
	b := &BVH{TriangleOrder: order}
	b.Nodes = append(b.Nodes, BVHNode{})
	b.buildRange(0, triangles, 0, int32(len(triangles)))
	return b
}

// buildRange builds the subtree rooted at node index nodeIdx over
// b.TriangleOrder[first:first+count], splitting recursively until a range is small
// enough to become a leaf.
func (b *BVH) buildRange(nodeIdx int, triangles []Triangle, first, count int32) {
	mn, mx := boundsOf(triangles, b.TriangleOrder[first:first+count])
	b.Nodes[nodeIdx].Min, b.Nodes[nodeIdx].Max = mn, mx

	if count <= bvhLeafThreshold {
		b.Nodes[nodeIdx].LeftFirst = first
		b.Nodes[nodeIdx].TriCount = count
		return
	}

	axis := longestAxis(mn, mx)
	slice := b.TriangleOrder[first : first+count]
	sort.Slice(slice, func(i, j int) bool {
		return centroid(triangles[slice[i]])[axis] < centroid(triangles[slice[j]])[axis]
	})
	mid := count / 2

	leftIdx := len(b.Nodes)
	b.Nodes = append(b.Nodes, BVHNode{}, BVHNode{})
	b.Nodes[nodeIdx].LeftFirst = int32(leftIdx)
	b.Nodes[nodeIdx].TriCount = 0

	b.buildRange(leftIdx, triangles, first, mid)
	b.buildRange(leftIdx+1, triangles, first+mid, count-mid)
}

func centroid(t Triangle) [3]float32 {
	return [3]float32{
		(t.V0[0] + t.V1[0] + t.V2[0]) / 3,
		(t.V0[1] + t.V1[1] + t.V2[1]) / 3,
		(t.V0[2] + t.V1[2] + t.V2[2]) / 3,
	}
}

func boundsOf(triangles []Triangle, indices []int32) (mn, mx [3]float32) {
	mn = [3]float32{maxF32, maxF32, maxF32}
	mx = [3]float32{-maxF32, -maxF32, -maxF32}
	for _, idx := range indices {
		t := triangles[idx]
		for _, v := range [3][3]float32{t.V0, t.V1, t.V2} {
			for a := range 3 {
				if v[a] < mn[a] {
					mn[a] = v[a]
				}
				if v[a] > mx[a] {
					mx[a] = v[a]
				}
			}
		}
	}
	return mn, mx
}

func longestAxis(mn, mx [3]float32) int {
	ext := [3]float32{mx[0] - mn[0], mx[1] - mn[1], mx[2] - mn[2]}
	axis := 0
	for a := 1; a < 3; a++ {
		if ext[a] > ext[axis] {
			axis = a
		}
	}
	return axis
}

const maxF32 = 3.4028235e38
