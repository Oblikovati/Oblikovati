// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati/math"

// Region/chain detection walks the sketch's segment connectivity. It assumes the
// well-formed degree-2 case (each junction joins two segments), which covers normal
// profiles; ambiguous high-degree junctions yield whatever simple chain is found
// rather than failing. Standalone closed curves (circles) are loops by themselves.

type segment struct {
	entity Entity
	a, b   *Point
}

// detectLoops returns the closed loops and the leftover open chains found among the
// given entities.
func detectLoops(entities []Entity) (closed []Loop, open []Loop) {
	var segs []segment
	for _, e := range entities {
		if loop, ok := standaloneLoop(e); ok {
			closed = append(closed, loop)
			continue
		}
		if a, b, ok := segmentEnds(e); ok {
			segs = append(segs, segment{entity: e, a: a, b: b})
		}
	}
	c, o := walkAll(segs)
	return append(closed, c...), o
}

// standaloneLoop returns a one-entity closed loop for an inherently closed curve: a
// circle, a full ellipse, or a spline flagged closed.
func standaloneLoop(e Entity) (Loop, bool) {
	switch t := e.(type) {
	case *Circle:
		return oneEntityLoop(t, sampleCircle(t)), true
	case *Ellipse:
		return oneEntityLoop(t, sampleEllipseEntity(t)), true
	case *Spline:
		if t.Closed {
			return oneEntityLoop(t, sampleSplineEntity(t)), true
		}
	}
	return Loop{}, false
}

// oneEntityLoop wraps a single closed entity and its sampled polygon as a loop.
func oneEntityLoop(e Entity, polygon []math.Point2) Loop {
	return Loop{entities: []ProfileEntity{{Entity: e}}, polygon: polygon, closed: true}
}

// segmentEnds returns the endpoints of an open segment entity (line, arc, or an open
// spline — joined to its neighbours by its first/last defining point).
func segmentEnds(e Entity) (a, b *Point, ok bool) {
	switch t := e.(type) {
	case *Line:
		return t.A, t.B, true
	case *Arc:
		return t.Start, t.End, true
	case *Spline:
		if !t.Closed && len(t.Points) >= 2 {
			return t.Points[0], t.Points[len(t.Points)-1], true
		}
		return nil, nil, false
	default:
		return nil, nil, false
	}
}

// walkAll traverses every segment exactly once, grouping them into closed loops and
// open chains.
func walkAll(segs []segment) (closed, open []Loop) {
	adj := map[*Point][]int{}
	for i, s := range segs {
		adj[s.a] = append(adj[s.a], i)
		adj[s.b] = append(adj[s.b], i)
	}
	used := make([]bool, len(segs))
	for i := range segs {
		if used[i] {
			continue
		}
		loop := walkChain(i, segs, adj, used)
		if loop.closed {
			closed = append(closed, loop)
		} else {
			open = append(open, loop)
		}
	}
	return closed, open
}

// walkChain follows connectivity from segment start until it closes (returns to the
// start point) or dead-ends.
func walkChain(start int, segs []segment, adj map[*Point][]int, used []bool) Loop {
	var (
		entities   []ProfileEntity
		polygon    = []math.Point2{}
		cur        = segs[start].a
		startPoint = cur
		idx        = start
	)
	for {
		used[idx] = true
		s := segs[idx]
		next, reversed := otherEnd(s, cur)
		entities = append(entities, ProfileEntity{Entity: s.entity, reversed: reversed})
		polygon = append(polygon, traversalPolyline(s.entity, reversed)...)
		cur = next
		if cur == startPoint {
			return Loop{entities: entities, polygon: polygon, closed: true}
		}
		nidx := nextUnused(adj[cur], used)
		if nidx < 0 {
			return Loop{entities: entities, polygon: polygon, closed: false}
		}
		idx = nidx
	}
}

// otherEnd returns the far endpoint of s relative to cur and whether traversal
// reverses the segment's natural direction.
func otherEnd(s segment, cur *Point) (*Point, bool) {
	if s.a == cur {
		return s.b, false
	}
	return s.a, true
}

// nextUnused returns the first unused segment index in candidates, or -1.
func nextUnused(candidates []int, used []bool) int {
	for _, k := range candidates {
		if !used[k] {
			return k
		}
	}
	return -1
}
