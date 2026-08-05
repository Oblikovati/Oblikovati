// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
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

// nestFaces turns the traced cycles into faces: every COUNTER-CLOCKWISE cycle bounds a face,
// and every CLOCKWISE cycle is an INNER boundary — it becomes a hole of the face it encloses.
//
// The holes must come from the clockwise cycles, not from re-nesting the counter-clockwise ones by
// containment (Oblikovati#2030). Both describe the same nesting only while the regions inside a
// face are pairwise DISJOINT. When two of them are ADJACENT — they share an edge, so the
// complement component they form is one connected region — containment nesting makes BOTH a hole
// of the parent, and the boundary they share is then emitted twice, once per hole loop. Downstream
// that lands as an edge whose two uses are on the SAME face: the face is no longer a disk-with-
// holes, so its χ = V−E+2F−L is off by one per shared edge and the solid is rejected as
// topologically inadmissible (a camera PCB whose mounting hole straddled the footprint of a block
// joined on top of it: 31 doubled edges, χ = −31).
//
// The half-edge walk already answers this exactly. A face lies to the LEFT of every half-edge on
// its boundary, so its outer boundary traces counter-clockwise and each of its inner boundaries
// traces CLOCKWISE — and an inner boundary that runs around several adjacent regions is traced as
// the ONE cycle it is. Taking the clockwise cycles as the holes therefore yields ∂(A∪B) once
// instead of ∂A and ∂B twice, and is identical to the old behaviour on disjoint nesting.
func nestFaces(cycles [][]math.Point2) []Face2D {
	var ccw, cw [][]math.Point2
	for _, c := range cycles {
		switch a := signedArea2D(c); {
		case len(c) < 3:
		case a > arrTol:
			ccw = append(ccw, c)
		case a < -arrTol:
			cw = append(cw, c)
		}
	}
	faces := make([]Face2D, len(ccw))
	for i, c := range ccw {
		faces[i] = Face2D{Outer: c}
	}
	for _, hole := range cw {
		if p := enclosingFace(ccw, hole); p >= 0 {
			faces[p].Holes = append(faces[p].Holes, hole)
		}
	}
	return faces
}

// enclosingFace returns the index of the smallest counter-clockwise cycle containing the clockwise
// cycle `hole` — the face `hole` is an inner boundary of — or −1 when there is none (the
// arrangement's outermost clockwise cycle, which bounds the unbounded region).
//
// The probe sits just to the LEFT of the hole's first edge. A face lies to the left of its boundary
// half-edges, so for a clockwise inner boundary that is a point in the enclosing face itself —
// OUTSIDE the area the hole encloses. That is what makes the containment test unambiguous: it never
// samples a shared-boundary vertex, and never lands inside the hole where a nested cycle could
// swallow it.
func enclosingFace(ccw [][]math.Point2, hole []math.Point2) int {
	probe := probeLeftOfFirstEdge(hole)
	best, bestArea := -1, stdmath.Inf(1)
	for j, c := range ccw {
		if !pointInPolygon2D(probe, c) {
			continue
		}
		if a := signedArea2D(c); a < bestArea {
			best, bestArea = j, a
		}
	}
	return best
}

// probeLeftOfFirstEdge returns a point just to the left of a loop's first edge: the edge midpoint
// nudged along the left normal by a small fraction of the edge length. "Left" is the face side of
// any boundary half-edge — the loop's interior for a CCW loop, its exterior for a CW one.
func probeLeftOfFirstEdge(loop []math.Point2) math.Point2 {
	a, b := loop[0], loop[1]
	e := a.VectorTo(b)
	mid := math.P2((a.X+b.X)/2, (a.Y+b.Y)/2)
	left := math.V2(-e.Y, e.X)               // face side of a boundary half-edge
	return mid.TranslateBy(left.Scale(1e-4)) // tol:calibrated — interior-probe nudge (see arrange2d arrTol)
}
