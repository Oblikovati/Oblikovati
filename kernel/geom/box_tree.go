// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"sort"

	"oblikovati.org/math"
)

// BoxTree is an axis-aligned bounding-volume hierarchy over a list of boxes — the shared
// broad-phase pair-culling index of the boolean pipeline (#1607). It generalizes the triangle
// BVH of kernel/ops/self_intersect_bvh.go (#1411) — which now delegates here — so kernel/brep
// (imported BY kernel/ops, hence unable to import it back) can cull face pairs with the same
// structure: build once over one side's boxes, then Query each of the other side's boxes,
// visiting only the items whose boxes overlap.
//
// Example: t := geom.NewBoxTree(bBoxes); t.Query(aBox, func(j int) bool { narrowPhase(j); return false })
type BoxTree struct {
	boxes []math.Box
	order []int // item indices, partitioned into contiguous leaf ranges
	nodes []boxTreeNode
	root  int
}

// boxTreeNode is a box plus EITHER a leaf range [start, start+count) into order (count > 0)
// or two children.
type boxTreeNode struct {
	box          math.Box
	start, count int
	left, right  int
}

// boxTreeLeafSize bounds a leaf's item count: small enough to prune well, large enough that
// the tree stays shallow and the per-node overhead does not dominate.
const boxTreeLeafSize = 8

// NewBoxTree builds the hierarchy over boxes by recursive median split on the longest
// centroid axis. The boxes slice is retained (not copied); items keep their input indices.
func NewBoxTree(boxes []math.Box) *BoxTree {
	order := make([]int, len(boxes))
	for i := range boxes {
		order[i] = i
	}
	t := &BoxTree{boxes: boxes, order: order}
	if len(boxes) > 0 {
		t.root = t.build(0, len(boxes))
	}
	return t
}

// build constructs the subtree over order[start:end] and returns its node index.
func (t *BoxTree) build(start, end int) int {
	box := math.EmptyBox()
	for _, bi := range t.order[start:end] {
		box = box.Union(t.boxes[bi])
	}
	idx := len(t.nodes)
	t.nodes = append(t.nodes, boxTreeNode{box: box, left: -1, right: -1})
	if end-start <= boxTreeLeafSize {
		t.nodes[idx].start, t.nodes[idx].count = start, end-start
		return idx
	}
	axis := widestBoxAxis(box)
	sub := t.order[start:end]
	sort.Slice(sub, func(i, j int) bool {
		return boxCentroidOnAxis(t.boxes[sub[i]], axis) < boxCentroidOnAxis(t.boxes[sub[j]], axis)
	})
	mid := (start + end) / 2
	t.nodes[idx].left = t.build(start, mid)
	t.nodes[idx].right = t.build(mid, end)
	return idx
}

// Query calls hit for each item whose box overlaps box, stopping early when hit returns true.
func (t *BoxTree) Query(box math.Box, hit func(item int) bool) {
	if len(t.nodes) == 0 {
		return
	}
	t.queryNode(t.root, box, hit)
}

func (t *BoxTree) queryNode(ni int, box math.Box, hit func(item int) bool) bool {
	nd := t.nodes[ni]
	if !nd.box.Intersects(box) {
		return false
	}
	if nd.count > 0 {
		for _, bi := range t.order[nd.start : nd.start+nd.count] {
			if t.boxes[bi].Intersects(box) && hit(bi) {
				return true
			}
		}
		return false
	}
	return t.queryNode(nd.left, box, hit) || t.queryNode(nd.right, box, hit)
}

// widestBoxAxis returns 0/1/2 for the box's widest dimension — the split axis with the best
// separation.
func widestBoxAxis(box math.Box) int {
	dx, dy, dz := box.Max.X-box.Min.X, box.Max.Y-box.Min.Y, box.Max.Z-box.Min.Z
	if dx >= dy && dx >= dz {
		return 0
	}
	if dy >= dz {
		return 1
	}
	return 2
}

// boxCentroidOnAxis returns a box's centre coordinate on the given axis.
func boxCentroidOnAxis(box math.Box, axis int) float64 {
	switch axis {
	case 0:
		return float64(box.Min.X+box.Max.X) / 2
	case 1:
		return float64(box.Min.Y+box.Max.Y) / 2
	default:
		return float64(box.Min.Z+box.Max.Z) / 2
	}
}
