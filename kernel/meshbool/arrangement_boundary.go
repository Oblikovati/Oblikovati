// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Boundary tracing for one arrangement face (ADR-0054 Layer 2a). A face's boundary is
// every directed facet edge whose reverse leaves the face's component. Tracing chains
// those into loops by pivoting around each shared vertex through the face interior —
// which separates loops correctly even where several meet at one vertex, unlike a
// naive greedy walk. Each loop is then split into runs at neighbour-face changes.

// halfEdge is one boundary directed edge with the tag of the face on its far side.
type halfEdge struct {
	from, to, ntag int
}

// arrangementFace traces one component's boundary into loops and runs and names it by
// its provenance tag.
func arrangementFace(verts []Point, tris [][3]int, tags []int, owner map[[2]int]int, comp, group []int) ArrangementFace {
	c := comp[group[0]]
	edges := boundaryHalfEdges(tris, tags, owner, comp, group, c)
	rawLoops := loopsFromBoundary(edges, tris, owner, comp, c)
	axis := planeAxis(triPoints(verts, tris[group[0]]))
	return ArrangementFace{Tag: tags[group[0]], Loops: classifyArrangementLoops(rawLoops, verts, axis)}
}

// boundaryHalfEdges collects the component's boundary directed edges — those whose
// reverse is owned by a triangle outside the component — each tagged with the
// neighbour face's tag.
func boundaryHalfEdges(tris [][3]int, tags []int, owner map[[2]int]int, comp, group []int, c int) []halfEdge {
	var out []halfEdge
	for _, ti := range group {
		t := tris[ti]
		for e := 0; e < 3; e++ {
			a, b := t[e], t[(e+1)%3]
			if tj, ok := owner[[2]int{b, a}]; !ok || comp[tj] != c {
				out = append(out, halfEdge{a, b, neighborTag(tags, tj, ok)})
			}
		}
	}
	return out
}

// neighborTag is the far-side face's tag, or -1 for an open boundary (which a
// watertight result never has).
func neighborTag(tags []int, tj int, ok bool) int {
	if !ok {
		return -1
	}
	return tags[tj]
}

// loopsFromBoundary chains the boundary half-edges into closed loops by fan-pivoting
// around each vertex; every boundary edge is consumed exactly once.
func loopsFromBoundary(edges []halfEdge, tris [][3]int, owner map[[2]int]int, comp []int, c int) [][]halfEdge {
	idx := make(map[[2]int]halfEdge, len(edges))
	for _, e := range edges {
		idx[[2]int{e.from, e.to}] = e
	}
	used := make(map[[2]int]bool, len(edges))
	var loops [][]halfEdge
	for _, s := range edges {
		if used[[2]int{s.from, s.to}] {
			continue
		}
		loops = append(loops, walkBoundaryLoop(s, idx, used, tris, owner, comp, c))
	}
	return loops
}

// walkBoundaryLoop follows one boundary loop from s, using nextBoundaryEdge to pick
// the correct successor at each vertex, until it closes.
func walkBoundaryLoop(s halfEdge, idx map[[2]int]halfEdge, used map[[2]int]bool, tris [][3]int, owner map[[2]int]int, comp []int, c int) []halfEdge {
	var loop []halfEdge
	for cur := s; ; {
		used[[2]int{cur.from, cur.to}] = true
		loop = append(loop, cur)
		w, ok := nextBoundaryEdge(cur.from, cur.to, tris, owner, comp, c)
		if !ok || used[[2]int{cur.to, w}] {
			return loop
		}
		cur = idx[[2]int{cur.to, w}]
	}
}

// nextBoundaryEdge pivots around vertex b (arrived via a→b) through the face interior
// and returns the next boundary edge b→w — the first fan edge whose reverse leaves the
// component. This is the manifold boundary-trace rotation, so a vertex where two loops
// touch is walked without crossing loops.
func nextBoundaryEdge(a, b int, tris [][3]int, owner map[[2]int]int, comp []int, c int) (int, bool) {
	for from := a; ; {
		ti, ok := owner[[2]int{from, b}]
		if !ok || comp[ti] != c {
			return 0, false // not reachable on a watertight component
		}
		w := apexVertex(tris[ti], from, b)
		if tj, ok := owner[[2]int{w, b}]; !ok || comp[tj] != c {
			return w, true // b→w's reverse leaves the component — a boundary edge
		}
		from = w
	}
}

// apexVertex returns the triangle vertex that is neither a nor b.
func apexVertex(t [3]int, a, b int) int {
	for _, v := range t {
		if v != a && v != b {
			return v
		}
	}
	return t[0] // degenerate triangle (a==b): callers never reach this on valid input
}

// classifyArrangementLoops marks the largest-projected-area loop as the outer boundary
// and splits every loop into neighbour-runs.
func classifyArrangementLoops(rawLoops [][]halfEdge, verts []Point, axis int) []ArrangementLoop {
	best := -1
	bestArea := new(big.Rat)
	for i, l := range rawLoops {
		a := new(big.Rat).Abs(loopArea2(loopVertexCycle(l), verts, axis))
		if best < 0 || a.Cmp(bestArea) > 0 {
			best, bestArea = i, a
		}
	}
	out := make([]ArrangementLoop, len(rawLoops))
	for i, l := range rawLoops {
		out[i] = ArrangementLoop{Outer: i == best, Runs: runsOfLoop(l)}
	}
	return out
}

// loopVertexCycle is a loop's ordered vertex indices (each edge's start), for the area
// test.
func loopVertexCycle(loop []halfEdge) []int {
	cycle := make([]int, len(loop))
	for i, e := range loop {
		cycle[i] = e.from
	}
	return cycle
}

// runsOfLoop splits a loop into maximal same-neighbour runs. It rotates the loop so a
// neighbour change sits at the start, so a run never wraps the seam; a loop bordering a
// single neighbour becomes one closed run (Verts[0] == Verts[last]).
func runsOfLoop(loop []halfEdge) []ArrangementRun {
	n := len(loop)
	start := 0
	for i := 0; i < n; i++ {
		if loop[(i-1+n)%n].ntag != loop[i].ntag {
			start = i
			break
		}
	}
	var runs []ArrangementRun
	for i := 0; i < n; {
		e := loop[(start+i)%n]
		run := ArrangementRun{NeighborTag: e.ntag, Verts: []int{e.from}}
		for i < n && loop[(start+i)%n].ntag == e.ntag {
			run.Verts = append(run.Verts, loop[(start+i)%n].to)
			i++
		}
		runs = append(runs, run)
	}
	return runs
}
