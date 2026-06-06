// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati/math"
)

// halfEdge is a directed use of an elementary edge (from→to); faces lie to its left.
type halfEdge struct{ from, to int }

// traceCycles walks the half-edge structure of the elementary edges into closed cycles
// (each a loop of points). Counter-clockwise cycles bound faces; clockwise cycles are
// holes or the unbounded outer region — [nestFaces] sorts them out.
func traceCycles(pts []math.Point2, edges [][2]int) [][]math.Point2 {
	hes, heID := buildHalfEdges(edges)
	outgoing, pos := sortAround(pts, hes)
	nextOf := func(id int) int {
		he := hes[id]
		twin := heID[[2]int{he.to, he.from}]
		list := outgoing[he.to]
		return list[(pos[twin]-1+len(list))%len(list)] // clockwise neighbour of the twin
	}
	visited := make([]bool, len(hes))
	var cycles [][]math.Point2
	for start := range hes {
		if visited[start] {
			continue
		}
		var loop []math.Point2
		for id := start; !visited[id]; id = nextOf(id) {
			visited[id] = true
			loop = append(loop, pts[hes[id].from])
		}
		cycles = append(cycles, loop)
	}
	return cycles
}

// buildHalfEdges creates the two directed half-edges of each undirected edge and an index
// from (from,to) to its half-edge id.
func buildHalfEdges(edges [][2]int) ([]halfEdge, map[[2]int]int) {
	hes := make([]halfEdge, 0, 2*len(edges))
	id := map[[2]int]int{}
	add := func(a, b int) {
		id[[2]int{a, b}] = len(hes)
		hes = append(hes, halfEdge{a, b})
	}
	for _, e := range edges {
		add(e[0], e[1])
		add(e[1], e[0])
	}
	return hes, id
}

// sortAround returns, per vertex, its outgoing half-edge ids sorted counter-clockwise by
// direction angle, plus each half-edge's position within its from-vertex's sorted list.
func sortAround(pts []math.Point2, hes []halfEdge) (map[int][]int, []int) {
	outgoing := map[int][]int{}
	for id, he := range hes {
		outgoing[he.from] = append(outgoing[he.from], id)
	}
	angle := func(id int) float64 {
		d := pts[hes[id].from].VectorTo(pts[hes[id].to])
		return stdmath.Atan2(d.Y, d.X)
	}
	pos := make([]int, len(hes))
	for _, list := range outgoing {
		sort.Slice(list, func(a, b int) bool { return angle(list[a]) < angle(list[b]) })
		for idx, id := range list {
			pos[id] = idx
		}
	}
	return outgoing, pos
}

// nestFaces keeps the counter-clockwise cycles as face outer loops and assigns each
// directly-enclosing relationship: a CCW cycle nested one level inside another becomes a
// hole of it (and remains a face in its own right — the arrangement of disjoint loops).
func nestFaces(cycles [][]math.Point2) []Face2D {
	var ccw [][]math.Point2
	for _, c := range cycles {
		if len(c) >= 3 && signedArea2D(c) > arrTol {
			ccw = append(ccw, c)
		}
	}
	faces := make([]Face2D, len(ccw))
	for i, c := range ccw {
		faces[i] = Face2D{Outer: c}
	}
	for i, inner := range ccw {
		if p := smallestContainer(ccw, i); p >= 0 {
			faces[p].Holes = append(faces[p].Holes, reversedLoop(inner))
		}
	}
	return faces
}

// smallestContainer returns the index of the smallest CCW cycle strictly containing
// cycle i (its direct parent in the nesting), or −1 if it is top-level. The probe sits
// just inside cycle i's boundary (an edge midpoint nudged inward), so it is in cycle i's
// own region — not at a shared-boundary vertex (ambiguous) and not near the centre where a
// concentric inner cycle would swallow it (which made a frame look nested in its own hole).
func smallestContainer(ccw [][]math.Point2, i int) int {
	probe := probeJustInside(ccw[i])
	best, bestArea := -1, stdmath.Inf(1)
	for j, c := range ccw {
		if j == i || !pointInPolygon2D(probe, c) {
			continue
		}
		if a := signedArea2D(c); a < bestArea {
			best, bestArea = j, a
		}
	}
	return best
}

// probeJustInside returns a point just inside a CCW loop near its first edge: the edge
// midpoint nudged along the inward (left) normal by a small fraction of the edge length.
func probeJustInside(loop []math.Point2) math.Point2 {
	a, b := loop[0], loop[1]
	e := a.VectorTo(b)
	mid := math.P2((a.X+b.X)/2, (a.Y+b.Y)/2)
	left := math.V2(-e.Y, e.X) // interior side of a CCW loop
	return mid.TranslateBy(left.Scale(1e-4))
}

func reversedLoop(loop []math.Point2) []math.Point2 {
	out := make([]math.Point2, len(loop))
	for i, p := range loop {
		out[len(loop)-1-i] = p
	}
	return out
}
