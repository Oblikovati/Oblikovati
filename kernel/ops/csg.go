// SPDX-License-Identifier: GPL-2.0-only

package ops

import "oblikovati/math"

// General intersecting boolean (PBI-171). Our solids are planar-faceted, so the robust
// path is a BSP-tree CSG (the Thibault–Naylor / csg.js algorithm) over the triangles of
// each body's tessellation: each operand becomes a BSP tree, the trees clip each other,
// and the kept triangles are welded back into a watertight B-rep. Correctness depends on
// the tessellation being consistently outward-wound (see planeProjector).

// csgEps is the on-plane tolerance (database units, cm); below it a point is coplanar.
const csgEps = 1e-7

// tri is a triangle with its supporting plane (unit normal n, offset w: n·p = w).
type tri struct {
	a, b, c math.Point3
	n       math.Vector3
	w       float64
}

func newTri(a, b, c math.Point3) (tri, bool) {
	n, err := math.UnitVector3FromVector(a.VectorTo(b).Cross(a.VectorTo(c)))
	if err != nil {
		return tri{}, false // degenerate (zero-area) triangle: drop it
	}
	nv := n.AsVector()
	return tri{a: a, b: b, c: c, n: nv, w: nv.Dot(a.AsVector())}, true
}

func (t tri) flipped() tri {
	return tri{a: t.a, b: t.c, c: t.b, n: t.n.Scale(-1), w: -t.w}
}

func (t tri) points() [3]math.Point3 { return [3]math.Point3{t.a, t.b, t.c} }

// csgUnion / csgSubtract / csgIntersect implement A∪B, A−B and A∩B over triangle sets
// via BSP clipping (the csg.js operation sequences).
func csgUnion(a, b []tri) []tri {
	na, nb := newBSP(a), newBSP(b)
	na.clipTo(nb)
	nb.clipTo(na)
	nb.invert()
	nb.clipTo(na)
	nb.invert()
	na.build(nb.all())
	return na.all()
}

func csgSubtract(a, b []tri) []tri {
	na, nb := newBSP(a), newBSP(b)
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

func csgIntersect(a, b []tri) []tri {
	na, nb := newBSP(a), newBSP(b)
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
	tris        []tri
	front, back *bspNode
}

func newBSP(tris []tri) *bspNode {
	node := &bspNode{}
	node.build(tris)
	return node
}

// invert flips the solid this tree represents (every triangle reversed, front/back
// swapped) — turning keep-inside into keep-outside.
func (node *bspNode) invert() {
	for i := range node.tris {
		node.tris[i] = node.tris[i].flipped()
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
func (node *bspNode) clip(tris []tri) []tri {
	if !node.hasPlane {
		return append([]tri(nil), tris...)
	}
	var front, back []tri
	for _, t := range tris {
		splitTri(node.n, node.w, t, &front, &back, &front, &back)
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
func (node *bspNode) all() []tri {
	out := append([]tri(nil), node.tris...)
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
func (node *bspNode) build(tris []tri) {
	if len(tris) == 0 {
		return
	}
	if !node.hasPlane {
		node.n, node.w, node.hasPlane = tris[0].n, tris[0].w, true
	}
	var front, back []tri
	for _, t := range tris {
		splitTri(node.n, node.w, t, &node.tris, &node.tris, &front, &back)
	}
	if len(front) > 0 {
		if node.front == nil {
			node.front = &bspNode{}
		}
		node.front.build(front)
	}
	if len(back) > 0 {
		if node.back == nil {
			node.back = &bspNode{}
		}
		node.back.build(back)
	}
}
