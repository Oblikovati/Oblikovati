// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/ops/internal/probe"
)

// Turning a polygon WITH holes into a simple one, and the predicates that decide where the
// bridge may go (split out of earcut.go for #2216).
//
// Ear clipping only works on a simple polygon, so each hole is first bridged into the outer
// ring by a diagonal: find the hole's leftmost vertex, ray-cast right to the ring, and pick a
// visible partner. The diagonal is only valid when it stays inside the polygon and crosses no
// edge — which is what the geometric predicates below decide.

// eliminateHoles bridges each hole into the outer ring (right-to-left by leftmost vertex).
func (tc *triContext) eliminateHoles(holeIdx []int, end int, outerNode *triNode) *triNode {
	queue := make([]*triNode, 0, len(holeIdx))
	for k, start := range holeIdx {
		stop := end
		if k+1 < len(holeIdx) {
			stop = holeIdx[k+1]
		}
		list := tc.linkedList(start, stop, false)
		if list == nil {
			continue // a degenerate (empty/collinear) hole range contributes no ring (#873)
		}
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
	o1 := probe.Sign(tc.area(p1, p2, q1))
	o2 := probe.Sign(tc.area(p1, p2, q2))
	o3 := probe.Sign(tc.area(q1, q2, p1))
	o4 := probe.Sign(tc.area(q1, q2, p2))
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
