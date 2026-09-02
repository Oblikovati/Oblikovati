// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/math"
)

// General intersecting boolean (PBI-171). Our solids are planar-faceted, so the robust
// path is a BSP-tree CSG (the Thibault–Naylor / csg.js algorithm) over the triangles of
// each body's tessellation: each operand becomes a BSP tree, the trees clip each other,
// and the kept triangles are welded back into a watertight B-rep. Correctness depends on
// the tessellation being consistently outward-wound (see planeProjector).

// csgUnion / csgSubtract / csgIntersect implement A∪B, A−B and A∩B over triangle sets
// via BSP clipping (the csg.js operation sequences). planeTol is the model-relative
// on-plane resolution (ADR-0042) the BSP uses to classify a point coplanar.
func csgUnion(a, b []mesh.Tri, planeTol float64) []mesh.Tri {
	na, nb := newBSP(a, planeTol), newBSP(b, planeTol)
	na.clipTo(nb)
	nb.clipTo(na)
	nb.invert()
	nb.clipTo(na)
	nb.invert()
	na.build(nb.all())
	return na.all()
}

func csgSubtract(a, b []mesh.Tri, planeTol float64) []mesh.Tri {
	na, nb := newBSP(a, planeTol), newBSP(b, planeTol)
	na.invert()
	na.clipTo(nb)
	nb.clipTo(na)
	nb.invert()
	nb.clipTo(na)
	nb.invert()
	na.build(nb.all())
	na.invert()
	return na.all()
}

func csgIntersect(a, b []mesh.Tri, planeTol float64) []mesh.Tri {
	na, nb := newBSP(a, planeTol), newBSP(b, planeTol)
	na.invert()
	nb.clipTo(na)
	nb.invert()
	na.clipTo(nb)
	nb.clipTo(na)
	na.build(nb.all())
	na.invert()
	return na.all()
}

// bspNode is a node of a BSP tree: a partition plane (the first inserted triangle's
// plane), the triangles coplanar with it, and the front/back subtrees.
type bspNode struct {
	n           math.Vector3
	w           float64
	hasPlane    bool
	tris        []mesh.Tri
	front, back *bspNode
	planeTol    float64 // model-relative on-plane resolution, propagated to every subtree
}

func newBSP(tris []mesh.Tri, planeTol float64) *bspNode {
	node := &bspNode{planeTol: planeTol}
	node.build(tris)
	return node
}

// invert flips the solid this tree represents (every triangle reversed, front/back
// swapped) — turning keep-inside into keep-outside.
func (node *bspNode) invert() {
	for i := range node.tris {
		node.tris[i] = node.tris[i].Flipped()
	}
	node.n = node.n.Scale(-1)
	node.w = -node.w
	if node.front != nil {
		node.front.invert()
	}
	if node.back != nil {
		node.back.invert()
	}
	node.front, node.back = node.back, node.front
}

// clip returns the parts of tris that fall outside the solid this tree represents.
func (node *bspNode) clip(tris []mesh.Tri) []mesh.Tri {
	if !node.hasPlane {
		return append([]mesh.Tri(nil), tris...)
	}
	var front, back []mesh.Tri
	for _, t := range tris {
		splitTri(node.n, node.w, t, &front, &back, &front, &back, node.planeTol)
	}
	if node.front != nil {
		front = node.front.clip(front)
	}
	if node.back != nil {
		back = node.back.clip(back)
	} else {
		back = nil // no back subtree ⇒ everything behind is inside ⇒ discarded
	}
	return append(front, back...)
}

// clipTo removes the parts of this tree's triangles that lie inside other.
func (node *bspNode) clipTo(other *bspNode) {
	node.tris = other.clip(node.tris)
	if node.front != nil {
		node.front.clipTo(other)
	}
	if node.back != nil {
		node.back.clipTo(other)
	}
}

// all returns every triangle in the tree.
func (node *bspNode) all() []mesh.Tri {
	out := append([]mesh.Tri(nil), node.tris...)
	if node.front != nil {
		out = append(out, node.front.all()...)
	}
	if node.back != nil {
		out = append(out, node.back.all()...)
	}
	return out
}

// build inserts triangles, using this node's plane (the first triangle's) to partition
// the rest into the coplanar set and the front/back subtrees.
func (node *bspNode) build(tris []mesh.Tri) {
	if len(tris) == 0 {
		return
	}
	if !node.hasPlane {
		node.n, node.w, node.hasPlane = tris[0].N, tris[0].W, true
	}
	var front, back []mesh.Tri
	for _, t := range tris {
		splitTri(node.n, node.w, t, &node.tris, &node.tris, &front, &back, node.planeTol)
	}
	if len(front) > 0 {
		if node.front == nil {
			node.front = &bspNode{planeTol: node.planeTol}
		}
		node.front.build(front)
	}
	if len(back) > 0 {
		if node.back == nil {
			node.back = &bspNode{planeTol: node.planeTol}
		}
		node.back.build(back)
	}
}
