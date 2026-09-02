// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/math"
)

// FillTriangles triangulates a planar region — an outer boundary with optional holes — into triangles
// whose vertex indices address outer[i] for i<len(outer), then the holes concatenated after it. It is
// the exported entry to the ear-clipping triangulator, for callers that need a filled overlay of a 2D
// region (a sketch profile/area highlight) rather than a B-rep face's tessellation.
//
//	tris := ops.FillTriangles(profile.OuterLoop().Polygon(), holePolys)
func FillTriangles(outer []math.Point2, holes [][]math.Point2) [][3]int {
	return earcut(outer, holes)
}

// earcut triangulates a planar polygon with holes, returning triangles as index triples into
// a combined vertex list (the outer loop followed by each hole, in order). It is a faithful
// port of Mapbox's "earcut" ear-clipping algorithm (ISC-licensed; Eberly-style hole bridging
// + a coincident/collinear-robust ear test + a self-intersection cure and a brute-force
// fallback). We adopt it because a hand-rolled single-bridge ear-clipper mis-triangulates
// faces with several holes whose bridges interact — e.g. a cap face with a hole grid coming
// out at half its true area. earcut handles those degeneracies, so a multi-hole face both
// measures (mass properties) and renders correctly.
//
// outer is the CCW boundary; holes are the (CW or CW-or-CCW, orientation-normalized
// internally) inner loops. The returned indices address outer[i] for i<len(outer), then the
// holes concatenated after it.
func earcut(outer []math.Point2, holes [][]math.Point2) [][3]int {
	verts := append([]math.Point2(nil), outer...)
	holeIdx := make([]int, 0, len(holes))
	for _, h := range holes {
		holeIdx = append(holeIdx, len(verts))
		verts = append(verts, h...)
	}
	n := len(verts)
	tc := &triContext{x: make([]float64, n), y: make([]float64, n)}
	for i, p := range verts {
		tc.x[i], tc.y[i] = p.X, p.Y
	}

	outerNode := tc.linkedList(0, len(outer), true)
	if outerNode == nil || outerNode.next == outerNode.prev {
		return nil
	}
	if len(holeIdx) > 0 {
		outerNode = tc.eliminateHoles(holeIdx, n, outerNode)
	}

	var tris [][3]int
	tc.earcutLinked(outerNode, &tris, 0)
	return tris
}

// triNode is one vertex in the doubly-linked polygon ring used by earcut.
type triNode struct {
	i          int // index into triContext.x / .y
	prev, next *triNode
	steiner    bool
}

// triContext carries the coordinate arrays and node pool for one triangulation.
type triContext struct {
	x, y []float64
}

func (tc *triContext) newNode(i int, last *triNode) *triNode {
	p := &triNode{i: i}
	if last == nil {
		p.prev, p.next = p, p
	} else {
		p.next = last.next
		p.prev = last
		last.next.prev = p
		last.next = p
	}
	return p
}

func removeTriNode(p *triNode) {
	p.next.prev = p.prev
	p.prev.next = p.next
}

// linkedList builds a circular doubly-linked list from a vertex range, oriented as requested.
func (tc *triContext) linkedList(start, end int, clockwise bool) *triNode {
	var last *triNode
	if clockwise == (tc.signedArea(start, end) > 0) {
		for i := start; i < end; i++ {
			last = tc.newNode(i, last)
		}
	} else {
		for i := end - 1; i >= start; i-- {
			last = tc.newNode(i, last)
		}
	}
	if last != nil && tc.equalsNode(last, last.next) {
		removeTriNode(last)
		last = last.next
	}
	return last
}

// filterPoints removes collinear or duplicate points from the ring starting at start.
func (tc *triContext) filterPoints(start, end *triNode) *triNode {
	if start == nil {
		return nil
	}
	if end == nil {
		end = start
	}
	p, again := start, true
	for again || p != end {
		again = false
		if !p.steiner && (tc.equalsNode(p, p.next) || tc.area(p.prev, p, p.next) == 0) {
			removeTriNode(p)
			p = p.prev
			end = p
			if p == p.next {
				return nil
			}
			again = true
		} else {
			p = p.next
		}
	}
	return end
}

// earcutLinked clips ears off the ring, recursing into the harder passes (cure
// self-intersections, then brute-force split) when a simple pass stalls.
func (tc *triContext) earcutLinked(ear *triNode, tris *[][3]int, pass int) {
	if ear == nil {
		return
	}
	stop := ear
	for ear.prev != ear.next {
		prev, next := ear.prev, ear.next
		if tc.isEar(ear) {
			*tris = append(*tris, [3]int{prev.i, ear.i, next.i})
			removeTriNode(ear)
			ear = next.next
			stop = next.next
			continue
		}
		ear = next
		if ear == stop {
			switch pass {
			case 0:
				tc.earcutLinked(tc.filterPoints(ear, nil), tris, 1)
			case 1:
				ear = tc.cureLocalIntersections(tc.filterPoints(ear, nil), tris)
				tc.earcutLinked(ear, tris, 2)
			case 2:
				tc.splitEarcut(ear, tris)
			}
			return
		}
	}
}

// isEar reports whether the triangle at ear is a valid (convex, empty) ear.
func (tc *triContext) isEar(ear *triNode) bool {
	a, b, c := ear.prev, ear, ear.next
	if tc.area(a, b, c) >= 0 {
		return false // reflex or collinear
	}
	ax, ay, bx, by, cx, cy := tc.x[a.i], tc.y[a.i], tc.x[b.i], tc.y[b.i], tc.x[c.i], tc.y[c.i]
	x0, y0 := stdmath.Min(ax, stdmath.Min(bx, cx)), stdmath.Min(ay, stdmath.Min(by, cy))
	x1, y1 := stdmath.Max(ax, stdmath.Max(bx, cx)), stdmath.Max(ay, stdmath.Max(by, cy))
	for p := c.next; p != a; p = p.next {
		px, py := tc.x[p.i], tc.y[p.i]
		if px >= x0 && px <= x1 && py >= y0 && py <= y1 &&
			pointInTri(ax, ay, bx, by, cx, cy, px, py) && tc.area(p.prev, p, p.next) >= 0 {
			return false
		}
	}
	return true
}

// cureLocalIntersections removes self-intersections by clipping any diagonal whose endpoints'
// neighbors form a valid triangle, emitting that triangle.
func (tc *triContext) cureLocalIntersections(start *triNode, tris *[][3]int) *triNode {
	if start == nil {
		return nil // filterPoints collapsed the ring (e.g. degenerate/overlapping holes) — #873
	}
	p := start
	for {
		a, b := p.prev, p.next.next
		if !tc.equalsNode(a, b) && tc.intersects(a, p, p.next, b) &&
			tc.locallyInside(a, b) && tc.locallyInside(b, a) {
			*tris = append(*tris, [3]int{a.i, p.i, b.i})
			removeTriNode(p)
			removeTriNode(p.next)
			p = b
			start = b
		}
		p = p.next
		if p == start {
			break
		}
	}
	return tc.filterPoints(p, nil)
}

// splitEarcut is the last-resort fallback: find a valid diagonal that splits the polygon in
// two and triangulate each half independently.
func (tc *triContext) splitEarcut(start *triNode, tris *[][3]int) {
	if start == nil {
		return // a collapsed ring has nothing left to split (#873)
	}
	a := start
	for {
		b := a.next.next
		for b != a.prev {
			if a.i != b.i && tc.isValidDiagonal(a, b) {
				c := tc.splitPolygon(a, b)
				a = tc.filterPoints(a, a.next)
				c = tc.filterPoints(c, c.next)
				tc.earcutLinked(a, tris, 0)
				tc.earcutLinked(c, tris, 0)
				return
			}
			b = b.next
		}
		a = a.next
		if a == start {
			return
		}
	}
}
