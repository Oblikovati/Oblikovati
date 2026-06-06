// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati/math"
)

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

// eliminateHoles bridges each hole into the outer ring (right-to-left by leftmost vertex).
func (tc *triContext) eliminateHoles(holeIdx []int, end int, outerNode *triNode) *triNode {
	queue := make([]*triNode, 0, len(holeIdx))
	for k, start := range holeIdx {
		stop := end
		if k+1 < len(holeIdx) {
			stop = holeIdx[k+1]
		}
		list := tc.linkedList(start, stop, false)
		if list == list.next {
			list.steiner = true
		}
		queue = append(queue, tc.leftmost(list))
	}
	sortByX(queue, tc.x)
	for _, h := range queue {
		outerNode = tc.eliminateHole(h, outerNode)
	}
	return outerNode
}

// eliminateHole connects hole node h into the outer ring via a bridge and re-filters.
func (tc *triContext) eliminateHole(h, outerNode *triNode) *triNode {
	bridge := tc.findHoleBridge(h, outerNode)
	if bridge == nil {
		return outerNode
	}
	bridgeReverse := tc.splitPolygon(bridge, h)
	tc.filterPoints(bridgeReverse, bridgeReverse.next)
	return tc.filterPoints(bridge, bridge.next)
}

// findHoleBridge finds an outer-ring node mutually visible from hole vertex h (leftmost),
// casting a ray toward −X and resolving reflex blockers (Eberly §4).
func (tc *triContext) findHoleBridge(h, outerNode *triNode) *triNode {
	hx, hy := tc.x[h.i], tc.y[h.i]
	qx, m, exact := tc.bridgeRayHit(hx, hy, outerNode)
	if exact != nil {
		return exact
	}
	if m == nil {
		return nil
	}
	if hx == qx {
		return m
	}
	stop := m
	mx, my := tc.x[m.i], tc.y[m.i]
	tanMin := stdmath.Inf(1)
	p := m
	for {
		px, py := tc.x[p.i], tc.y[p.i]
		// A reflex vertex p inside the candidate region (h, q, m) can block visibility;
		// among such blockers pick the one whose ray to h hugs the +X direction.
		if hx >= px && px >= mx && hx != px &&
			pointInTri(hx, hy, qx, hy, mx, my, px, py) {
			tanCur := stdmath.Abs(hy-py) / (hx - px)
			if tc.locallyInside(p, h) && (tanCur < tanMin ||
				(tanCur == tanMin && (px > mx || (px == mx && tc.sectorContains(m, p))))) {
				m = p
				mx, my = px, py
				tanMin = tanCur
			}
		}
		p = p.next
		if p == stop {
			break
		}
	}
	return m
}

func (tc *triContext) bridgeRayHit(hx, hy float64, outerNode *triNode) (float64, *triNode, *triNode) {
	p := outerNode
	qx := stdmath.Inf(-1)
	var m *triNode
	for {
		px, py := tc.x[p.i], tc.y[p.i]
		nx := tc.x[p.next.i]
		if hy <= py && hy >= tc.y[p.next.i] && tc.y[p.next.i] != py {
			x := px + (hy-py)*(nx-px)/(tc.y[p.next.i]-py)
			if x <= hx && x > qx {
				qx = x
				if x == hx {
					if hy == py {
						return qx, m, p
					}
					if hy == tc.y[p.next.i] {
						return qx, m, p.next
					}
				}
				if px < nx {
					m = p
				} else {
					m = p.next
				}
			}
		}
		p = p.next
		if p == outerNode {
			return qx, m, nil
		}
	}
}

// sectorContains breaks ties in findHoleBridge: whether m lies in p's sector toward the hole.
func (tc *triContext) sectorContains(m, p *triNode) bool {
	return tc.area(m.prev, m, p.prev) < 0 && tc.area(p.next, m, m.next) < 0
}

// splitPolygon links a and b with a doubled bridge, splitting one ring into two.
func (tc *triContext) splitPolygon(a, b *triNode) *triNode {
	a2 := &triNode{i: a.i}
	b2 := &triNode{i: b.i}
	an, bp := a.next, b.prev
	a.next = b
	b.prev = a
	a2.next = an
	an.prev = a2
	b2.next = a2
	a2.prev = b2
	bp.next = b2
	b2.prev = bp
	return b2
}

// leftmost returns the node of a ring with the smallest x (ties by smallest y).
func (tc *triContext) leftmost(start *triNode) *triNode {
	p, left := start, start
	for {
		if tc.x[p.i] < tc.x[left.i] || (tc.x[p.i] == tc.x[left.i] && tc.y[p.i] < tc.y[left.i]) {
			left = p
		}
		p = p.next
		if p == start {
			break
		}
	}
	return left
}

// isValidDiagonal reports whether (a,b) is a valid interior diagonal for splitEarcut.
func (tc *triContext) isValidDiagonal(a, b *triNode) bool {
	return a.next.i != b.i && a.prev.i != b.i && !tc.intersectsPolygon(a, b) &&
		(tc.locallyInside(a, b) && tc.locallyInside(b, a) && tc.middleInside(a, b) &&
			(tc.area(a.prev, a, b.prev) != 0 || tc.area(a, b.prev, b) != 0) ||
			tc.equalsNode(a, b) && tc.area(a.prev, a, a.next) > 0 && tc.area(b.prev, b, b.next) > 0)
}

// area returns twice the signed area of triangle pqr (negative = CW = convex in a CCW ring).
func (tc *triContext) area(p, q, r *triNode) float64 {
	return (tc.y[q.i]-tc.y[p.i])*(tc.x[r.i]-tc.x[q.i]) - (tc.x[q.i]-tc.x[p.i])*(tc.y[r.i]-tc.y[q.i])
}

func (tc *triContext) equalsNode(p, q *triNode) bool {
	return tc.x[p.i] == tc.x[q.i] && tc.y[p.i] == tc.y[q.i]
}

// intersects reports whether segments p1p2 and q1q2 properly intersect.
func (tc *triContext) intersects(p1, p2, q1, q2 *triNode) bool {
	o1 := sign(tc.area(p1, p2, q1))
	o2 := sign(tc.area(p1, p2, q2))
	o3 := sign(tc.area(q1, q2, p1))
	o4 := sign(tc.area(q1, q2, p2))
	if o1 != o2 && o3 != o4 {
		return true
	}
	if o1 == 0 && tc.onSeg(p1, q1, p2) {
		return true
	}
	if o2 == 0 && tc.onSeg(p1, q2, p2) {
		return true
	}
	if o3 == 0 && tc.onSeg(q1, p1, q2) {
		return true
	}
	if o4 == 0 && tc.onSeg(q1, p2, q2) {
		return true
	}
	return false
}

func (tc *triContext) onSeg(p, q, r *triNode) bool {
	return tc.x[q.i] <= stdmath.Max(tc.x[p.i], tc.x[r.i]) && tc.x[q.i] >= stdmath.Min(tc.x[p.i], tc.x[r.i]) &&
		tc.y[q.i] <= stdmath.Max(tc.y[p.i], tc.y[r.i]) && tc.y[q.i] >= stdmath.Min(tc.y[p.i], tc.y[r.i])
}

// intersectsPolygon reports whether the diagonal (a,b) crosses any polygon edge.
func (tc *triContext) intersectsPolygon(a, b *triNode) bool {
	p := a
	for {
		if p.i != a.i && p.next.i != a.i && p.i != b.i && p.next.i != b.i &&
			tc.intersects(p, p.next, a, b) {
			return true
		}
		p = p.next
		if p == a {
			break
		}
	}
	return false
}

// locallyInside reports whether b is inside the polygon in the neighborhood of a.
func (tc *triContext) locallyInside(a, b *triNode) bool {
	if tc.area(a.prev, a, a.next) < 0 {
		return tc.area(a, b, a.next) >= 0 && tc.area(a, a.prev, b) >= 0
	}
	return tc.area(a, b, a.prev) < 0 || tc.area(a, a.next, b) < 0
}

// middleInside reports whether the midpoint of (a,b) lies inside the polygon.
func (tc *triContext) middleInside(a, b *triNode) bool {
	p := a
	inside := false
	px, py := (tc.x[a.i]+tc.x[b.i])/2, (tc.y[a.i]+tc.y[b.i])/2
	for {
		if (tc.y[p.i] > py) != (tc.y[p.next.i] > py) && tc.y[p.next.i] != tc.y[p.i] &&
			px < (tc.x[p.next.i]-tc.x[p.i])*(py-tc.y[p.i])/(tc.y[p.next.i]-tc.y[p.i])+tc.x[p.i] {
			inside = !inside
		}
		p = p.next
		if p == a {
			break
		}
	}
	return inside
}

func (tc *triContext) signedArea(start, end int) float64 {
	var sum float64
	j := end - 1
	for i := start; i < end; i++ {
		sum += (tc.x[j] - tc.x[i]) * (tc.y[i] + tc.y[j])
		j = i
	}
	return sum
}

func sign(x float64) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

// pointInTri reports whether (px,py) lies inside triangle (ax,ay)(bx,by)(cx,cy).
func pointInTri(ax, ay, bx, by, cx, cy, px, py float64) bool {
	return (cx-px)*(ay-py)-(ax-px)*(cy-py) >= 0 &&
		(ax-px)*(by-py)-(bx-px)*(ay-py) >= 0 &&
		(bx-px)*(cy-py)-(cx-px)*(by-py) >= 0
}

// sortByX stable-sorts hole bridge nodes by their leftmost x.
func sortByX(nodes []*triNode, x []float64) {
	for i := 1; i < len(nodes); i++ {
		for j := i; j > 0 && x[nodes[j-1].i] > x[nodes[j].i]; j-- {
			nodes[j-1], nodes[j] = nodes[j], nodes[j-1]
		}
	}
}
