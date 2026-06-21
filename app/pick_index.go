// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"sort"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/scene"
)

// pickLeafSize is the most placements a BVH leaf holds. A small leaf keeps the broad-phase
// candidate set tight (fewer false-positive boxes to ray-cast) at the cost of a deeper tree;
// 4 is the usual sweet spot for ray queries over scattered AABBs.
const pickLeafSize = 4

// pickPlacement is one component placement the index can hit: its component-LOCAL source body
// (shared across sibling placements — the flyweight, ADR-0038), the world transform, the
// world-space AABB of that transformed body, and the occurrence it resolves to (#769).
type pickPlacement struct {
	source    *topo.Body
	transform math.Matrix4
	box       math.Box
	occ       *occurrence.Occurrence
	index     int
}

// bvhNode is one node of the placement BVH. Interior nodes have left >= 0 (and ignore
// first/count); leaves have left == -1 and own the placements order[first:first+count].
type bvhNode struct {
	box          math.Box
	left, right  int
	first, count int
}

// assemblyPickIndex is a bounding-volume hierarchy over an assembly's placement AABBs so a pick
// ray visits O(log N + hits) boxes instead of materializing all N world-space bodies — the F5
// bottleneck, where worldAssemblyBodies ran one ops.TransformBody per occurrence (2 GB / 1.5 s
// for a single 30k selection). World geometry is built lazily and ONLY for the placements the
// ray actually crosses, then cached so OccurrenceOfBody still resolves a hit body to its
// component (#769). See M34-F5.
type assemblyPickIndex struct {
	asm        *compdef.AssemblyComponentDefinition
	revision   uint64
	placements []pickPlacement
	order      []int
	nodes      []bvhNode
	world      map[int]*topo.Body
	owner      map[*topo.Body]*occurrence.Occurrence
}

// newAssemblyPickIndex flattens asm's placed bodies into world-AABB placements and builds the
// BVH over them. The AABB is the source body's local RangeBox transformed by the placement
// matrix (eight corners) — cheap, and it never touches the body's faces.
//
//	idx := newAssemblyPickIndex(asm)
//	bodies := idx.rayBodies(origin, dir) // only the placements the ray crosses
func newAssemblyPickIndex(asm *compdef.AssemblyComponentDefinition) *assemblyPickIndex {
	placed := asm.PlacedBodies()
	idx := &assemblyPickIndex{
		asm:        asm,
		revision:   asm.Occurrences().Revision(),
		placements: make([]pickPlacement, 0, len(placed)),
		world:      map[int]*topo.Body{},
		owner:      map[*topo.Body]*occurrence.Occurrence{},
	}
	for i, pb := range placed {
		box := pb.Body.RangeBox().Transform(pb.Transform)
		idx.placements = append(idx.placements, pickPlacement{
			source: pb.Body, transform: pb.Transform, box: box, occ: pb.Source, index: i,
		})
	}
	idx.build()
	return idx
}

// build constructs the BVH over idx.placements, populating order (the leaf permutation of
// placement indices) and nodes. An empty assembly still yields one empty leaf, so a query
// returns nothing without a nil check.
func (idx *assemblyPickIndex) build() {
	idx.order = make([]int, len(idx.placements))
	for i := range idx.order {
		idx.order[i] = i
	}
	idx.nodes = idx.nodes[:0]
	idx.buildNode(0, len(idx.order))
}

// buildNode builds the subtree over order[first:first+count] and returns its node index,
// splitting at the median centroid on the widest centroid axis until a leaf fits pickLeafSize.
func (idx *assemblyPickIndex) buildNode(first, count int) int {
	node := bvhNode{box: idx.rangeBox(first, count), left: -1, first: first, count: count}
	if count <= pickLeafSize {
		return idx.appendNode(node)
	}
	idx.sortByCentroid(first, count, idx.widestCentroidAxis(first, count))
	mid := count / 2
	self := idx.appendNode(node)
	left := idx.buildNode(first, mid)
	right := idx.buildNode(first+mid, count-mid)
	idx.nodes[self].left, idx.nodes[self].right = left, right
	return self
}

// appendNode stores n and returns its index.
func (idx *assemblyPickIndex) appendNode(n bvhNode) int {
	idx.nodes = append(idx.nodes, n)
	return len(idx.nodes) - 1
}

// rangeBox returns the union AABB of the placements in order[first:first+count].
func (idx *assemblyPickIndex) rangeBox(first, count int) math.Box {
	box := math.EmptyBox()
	for _, p := range idx.order[first : first+count] {
		box = box.Union(idx.placements[p].box)
	}
	return box
}

// widestCentroidAxis returns the axis (0=X,1=Y,2=Z) with the largest spread of placement-box
// centroids in the range — the axis to split on so the children separate cleanly.
func (idx *assemblyPickIndex) widestCentroidAxis(first, count int) int {
	cb := math.EmptyBox()
	for _, p := range idx.order[first : first+count] {
		cb = cb.ExtendPoint(idx.placements[p].box.Center())
	}
	d := cb.Diagonal()
	if d.X >= d.Y && d.X >= d.Z {
		return 0
	}
	if d.Y >= d.Z {
		return 1
	}
	return 2
}

// sortByCentroid orders order[first:first+count] by centroid coordinate on axis so the lower
// half holds the smaller coordinates. A full sort of the range is used for clarity; the index
// is rebuilt only on an occurrence-revision change, so build cost is amortized across frames.
func (idx *assemblyPickIndex) sortByCentroid(first, count, axis int) {
	sub := idx.order[first : first+count]
	sort.Slice(sub, func(a, b int) bool {
		return centroidOnAxis(idx.placements[sub[a]].box, axis) < centroidOnAxis(idx.placements[sub[b]].box, axis)
	})
}

// centroidOnAxis returns the box center's coordinate on the given axis (0=X,1=Y,2=Z).
func centroidOnAxis(b math.Box, axis int) float64 {
	c := b.Center()
	switch axis {
	case 0:
		return c.X
	case 1:
		return c.Y
	default:
		return c.Z
	}
}

// rayCandidates returns the placement indices whose world AABB the forward ray crosses, found
// by descending the BVH and skipping any subtree the ray misses. Order is unspecified; the
// caller depth-sorts by the true face hit.
func (idx *assemblyPickIndex) rayCandidates(origin math.Point3, dir math.Vector3) []int {
	if len(idx.nodes) == 0 {
		return nil
	}
	var out []int
	stack := []int{0}
	for len(stack) > 0 {
		n := idx.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		if _, ok := n.box.IntersectsRay(origin, dir); !ok {
			continue
		}
		if n.left < 0 {
			out = idx.appendLeafHits(out, n, origin, dir)
			continue
		}
		stack = append(stack, n.left, n.right)
	}
	return out
}

// appendLeafHits adds the leaf's placements whose own AABB the ray crosses — the leaf box is a
// union, so a leaf hit does not imply every member is hit.
func (idx *assemblyPickIndex) appendLeafHits(out []int, n bvhNode, origin math.Point3, dir math.Vector3) []int {
	for _, p := range idx.order[n.first : n.first+n.count] {
		if _, ok := idx.placements[p].box.IntersectsRay(origin, dir); ok {
			out = append(out, p)
		}
	}
	return out
}

// rayBodies returns the world-space bodies for the placements the ray crosses, materializing
// each (ops.TransformBody) at most once and caching it so OccurrenceOfBody resolves the hit and
// a repeat pick on the same revision is free. Only ray-crossed placements are built — the whole
// point of F5 — so a deep selection costs a handful of transforms, not N.
func (idx *assemblyPickIndex) rayBodies(origin math.Point3, dir math.Vector3) []*topo.Body {
	cands := idx.rayCandidates(origin, dir)
	out := make([]*topo.Body, 0, len(cands))
	for _, p := range cands {
		if body, ok := idx.materialize(p); ok {
			out = append(out, body)
		}
	}
	return out
}

// materialize returns the cached world body for placement p, building it on first use. The
// lineage prefix matches worldAssemblyBodies, so a body's reference keys are identical whether
// it came from the index or the legacy full-flatten path.
func (idx *assemblyPickIndex) materialize(p int) (*topo.Body, bool) {
	if body, ok := idx.world[p]; ok {
		return body, true
	}
	pl := idx.placements[p]
	body, err := ops.TransformBody(pl.source, pl.transform, occurrenceBodyLineage(pl.index))
	if err != nil {
		return nil, false
	}
	idx.world[p] = body
	idx.owner[body] = pl.occ
	return body, true
}

// occurrenceOf resolves a world body the index materialized back to its occurrence.
func (idx *assemblyPickIndex) occurrenceOf(b *topo.Body) (*occurrence.Occurrence, bool) {
	o, ok := idx.owner[b]
	return o, ok
}

// frustumPlacements returns the placement indices whose world AABB the frustum keeps, pruning any
// BVH subtree wholly outside the view — the broad phase of F1 per-instance culling. The result is
// in ascending placement order so grouping is deterministic and matches VisibleInstances'
// first-seen order (the head's per-source mesh cache then keeps hitting).
func (idx *assemblyPickIndex) frustumPlacements(f scene.Frustum) []int {
	if len(idx.nodes) == 0 {
		return nil
	}
	var out []int
	stack := []int{0}
	for len(stack) > 0 {
		n := idx.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		if !f.IntersectsBox(n.box) {
			continue
		}
		if n.left < 0 {
			out = idx.appendFrustumLeaf(out, n, f)
			continue
		}
		stack = append(stack, n.left, n.right)
	}
	sort.Ints(out)
	return out
}

// appendFrustumLeaf adds the leaf's placements whose own AABB the frustum keeps (the leaf box is a
// union, so a kept leaf does not mean every member is in view).
func (idx *assemblyPickIndex) appendFrustumLeaf(out []int, n bvhNode, f scene.Frustum) []int {
	for _, p := range idx.order[n.first : n.first+n.count] {
		if f.IntersectsBox(idx.placements[p].box) {
			out = append(out, p)
		}
	}
	return out
}

// groupPlacements collapses the given placements into render instance groups by shared source body,
// preserving the placements' (ascending) order so the result is deterministic and identical in
// shape to assemblyInstances when every placement is included.
func (idx *assemblyPickIndex) groupPlacements(placements []int) []InstanceGroup {
	bySource := make(map[*topo.Body]int, len(placements))
	out := make([]InstanceGroup, 0, len(placements))
	for _, p := range placements {
		pl := idx.placements[p]
		i, ok := bySource[pl.source]
		if !ok {
			i = len(out)
			bySource[pl.source] = i
			out = append(out, InstanceGroup{Source: pl.source})
		}
		out[i].Transforms = append(out[i].Transforms, pl.transform)
	}
	return out
}
