// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
)

// Profile extraction — LOOP TRACING (M48 #2242 split of profile.go). Traces the closed boundary rings out
// of the welded sketch-edge graph: the directed edge set of a region boundary, the deterministic ring
// walk, and the coordinate welder that keys points to shared vertices. The containment/nesting
// classification lives in profile_nesting.go; the circle-to-polygon sampling in profile_polygon.go.

// weldedLoopEdges returns a loop's undirected edges as sorted welded-index pairs, so an edge two
// abutting cells trace from the same boundary points registers as the one shared edge.
func weldedLoopEdges(w *loopWelder, l Loop) [][2]int {
	idx := make([]int, len(l.polygon))
	for i, p := range l.polygon {
		idx[i] = w.add(p)
	}
	edges := make([][2]int, 0, len(idx))
	for i := range idx {
		a, b := idx[i], idx[(i+1)%len(idx)]
		if a == b {
			continue
		}
		if a > b {
			a, b = b, a
		}
		edges = append(edges, [2]int{a, b})
	}
	return edges
}

// boundaryDirEdges keeps the directed edges whose reverse is absent — the outline of the union
// (an edge shared by two abutting cells appears in both directions and is dropped).
//
// Emitted in sorted order: Go randomizes map iteration, and this slice seeds the adjacency
// chainLoopBoundary walks, whose append order decides how edges pair at a touching vertex.
func boundaryDirEdges(dir map[[2]int]int) [][2]int {
	var keep [][2]int
	for _, e := range sortedDirKeys(dir) {
		if dir[[2]int{e[1], e[0]}] == 0 {
			for k := 0; k < dir[e]; k++ {
				keep = append(keep, e)
			}
		}
	}
	return keep
}

// sortedDirKeys returns the directed edges in ascending (from, to) order.
func sortedDirKeys(dir map[[2]int]int) [][2]int {
	keys := make([][2]int, 0, len(dir))
	for e := range dir {
		keys = append(keys, e)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	return keys
}

// chainLoopBoundary links directed edges head-to-tail into closed rings.
//
// The ring START is taken in sorted order, not map order. A ring's start vertex fixes the point
// SEQUENCE Polygon() hands downstream, and the extrude builds one prism face per polygon edge in
// that order — so a randomly-rotated hole ring produced a prism with the same vertices but its
// faces on different planes, and the boolean consuming it drifted run to run (#23). Where two hole
// rings touch at a vertex the start also decides the edge PAIRING, changing the rings themselves.
func chainLoopBoundary(edges [][2]int, pts []math.Point2) [][]math.Point2 {
	next := make(map[int][]int, len(edges))
	for _, e := range edges {
		next[e[0]] = append(next[e[0]], e[1])
	}
	starts := make([]int, 0, len(next))
	for v := range next {
		starts = append(starts, v)
	}
	sort.Ints(starts)
	var rings [][]math.Point2
	for _, start := range starts {
		for len(next[start]) > 0 {
			if ring := traceRing(start, next, pts); len(ring) >= 3 {
				rings = append(rings, ring)
			}
		}
	}
	return rings
}

// traceRing walks next-edges from start until it returns, consuming each edge once.
func traceRing(start int, next map[int][]int, pts []math.Point2) []math.Point2 {
	ring := []math.Point2{pts[start]}
	for cur := start; ; {
		outs := next[cur]
		if len(outs) == 0 {
			return nil // open chain (degenerate)
		}
		nxt := outs[0]
		next[cur] = outs[1:]
		if nxt == start {
			return ring
		}
		ring = append(ring, pts[nxt])
		cur = nxt
	}
}

// loopWelder merges coincident loop vertices onto a shared index list (a tolerance grid coarser
// than the arrangement weld so computed cell corners coincide).
type loopWelder struct {
	index  map[[2]int64]int
	points []math.Point2
}

func newLoopWelder() *loopWelder { return &loopWelder{index: map[[2]int64]int{}} }

func (w *loopWelder) add(p math.Point2) int {
	const grid = 1e-7 // tol:calibrated — 2D loop-welder coincidence grid (see brep arrange2d)
	k := [2]int64{int64(stdmath.Round(p.X / grid)), int64(stdmath.Round(p.Y / grid))}
	if i, ok := w.index[k]; ok {
		return i
	}
	w.index[k] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}
