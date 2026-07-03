// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
)

// Uniform-grid spatial hashing for the 2D arrangement (Ericson, Real-Time Collision
// Detection §7.1) — the broad phase that retires planarize's O(S²) segment-pair scan and
// splitTJunctions' O(E·V) vertex-on-edge scan (#1607). Only candidacy moves here: the narrow
// phase (Segment2dIntersection at arrTol, the tjTol distance test) is untouched, and every
// pair it can accept lies within its tolerance of both participants — far inside the padded
// boxes hashed below — so the culled arrangement finds identical intersections and
// T-junctions (pinned by the randomized equivalence tests in arrange2d_cull_test.go).

// segCullPad inflates every segment's AABB before hashing and overlap-testing: 10× the
// arrangement tolerance under which the narrow phase can still accept a crossing between
// near-touching segments.
const segCullPad = 10 * arrTol // tol:calibrated — box-cull slack over the arrangement tolerance (see arrTol)

// tjCullPad inflates an edge's query box in the T-junction pass: 10× the on-edge distance
// tolerance, so every vertex within tjTol of the edge is guaranteed to be visited.
const tjCullPad = 10 * tjTol // tol:calibrated — box-cull slack over the T-junction tolerance (see tjTol)

// segmentCullGrid hashes padded segment AABBs into square cells.
type segmentCullGrid struct {
	cell  float64
	boxes [][4]float64 // per segment: padded [minX, minY, maxX, maxY]
	cells map[[2]int32][]int32
}

// newSegmentCullGrid builds the hash over all segments' padded boxes.
func newSegmentCullGrid(segments [][2]math.Point2) *segmentCullGrid {
	boxes := make([][4]float64, len(segments))
	for i, s := range segments {
		boxes[i] = [4]float64{
			min(float64(s[0].X), float64(s[1].X)) - segCullPad,
			min(float64(s[0].Y), float64(s[1].Y)) - segCullPad,
			max(float64(s[0].X), float64(s[1].X)) + segCullPad,
			max(float64(s[0].Y), float64(s[1].Y)) + segCullPad,
		}
	}
	g := &segmentCullGrid{cell: cullCellEdge(boxes), boxes: boxes, cells: map[[2]int32][]int32{}}
	for i := range boxes {
		g.eachCoveredCell(boxes[i], func(c [2]int32) { g.cells[c] = append(g.cells[c], int32(i)) })
	}
	return g
}

// cullCellEdge picks the hash cell edge as the mean padded-box extent — Ericson's guidance of
// sizing cells to the average object, so a segment covers O(1) cells and a cell holds
// O(local density) segments. Any positive value is CORRECT (candidacy is decided by the
// padded-box overlap test, never by the cell); it only tunes bucket occupancy, so the
// degenerate all-degenerate input just falls back to 1.
func cullCellEdge(boxes [][4]float64) float64 {
	if len(boxes) == 0 {
		return 1
	}
	sum := 0.0
	for _, b := range boxes {
		sum += (b[2] - b[0]) + (b[3] - b[1])
	}
	if cell := sum / float64(2*len(boxes)); cell > 0 && !stdmath.IsInf(cell, 1) {
		return cell
	}
	return 1
}

// candidates returns the ascending indices j ≠ i whose padded boxes overlap segment i's — a
// superset of every pair the narrow phase can accept, iterated in the retired brute scan's
// ascending-j order (the cut list feeds an unstable sort, so insertion order must not drift).
func (g *segmentCullGrid) candidates(i int) []int {
	bi := g.boxes[i]
	var out []int
	g.eachCoveredCell(bi, func(c [2]int32) {
		for _, j := range g.cells[c] {
			if int(j) != i && boxesOverlap2D(bi, g.boxes[j]) {
				out = append(out, int(j))
			}
		}
	})
	sort.Ints(out)
	return dedupSortedInts(out)
}

// eachCoveredCell visits every cell the box overlaps.
func (g *segmentCullGrid) eachCoveredCell(b [4]float64, visit func(c [2]int32)) {
	x0, y0 := cellCoord(b[0], g.cell), cellCoord(b[1], g.cell)
	x1, y1 := cellCoord(b[2], g.cell), cellCoord(b[3], g.cell)
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			visit([2]int32{x, y})
		}
	}
}

func cellCoord(v, cell float64) int32 {
	return int32(stdmath.Floor(v / cell))
}

func boxesOverlap2D(a, b [4]float64) bool {
	return a[0] <= b[2] && a[2] >= b[0] && a[1] <= b[3] && a[3] >= b[1]
}

// dedupSortedInts compacts consecutive duplicates in-place (a segment straddling several
// cells is hashed into each of them).
func dedupSortedInts(v []int) []int {
	out := v[:0]
	for k, x := range v {
		if k == 0 || x != out[len(out)-1] {
			out = append(out, x)
		}
	}
	return out
}

// vertexCullGrid hashes the welded arrangement vertices for the T-junction pass; each vertex
// occupies exactly one cell (so queries need no dedup) and an edge queries the cells its
// padded box covers.
type vertexCullGrid struct {
	cell  float64
	cells map[[2]int32][]int32
}

// newVertexCullGrid builds the hash once per splitTJunctions run — the welded point set is
// fixed there; only the edge set evolves.
func newVertexCullGrid(pts []math.Point2) *vertexCullGrid {
	g := &vertexCullGrid{cell: vertexCellEdge(pts), cells: map[[2]int32][]int32{}}
	for i, p := range pts {
		c := [2]int32{cellCoord(float64(p.X), g.cell), cellCoord(float64(p.Y), g.cell)}
		g.cells[c] = append(g.cells[c], int32(i))
	}
	return g
}

// vertexCellEdge sizes vertex cells to the point-cloud extent over √n — ~O(1) vertices per
// cell for a roughly uniform arrangement. Any positive value is correct (see cullCellEdge).
func vertexCellEdge(pts []math.Point2) float64 {
	if len(pts) == 0 {
		return 1
	}
	lo := [2]float64{stdmath.Inf(1), stdmath.Inf(1)}
	hi := [2]float64{stdmath.Inf(-1), stdmath.Inf(-1)}
	for _, p := range pts {
		lo[0], lo[1] = min(lo[0], float64(p.X)), min(lo[1], float64(p.Y))
		hi[0], hi[1] = max(hi[0], float64(p.X)), max(hi[1], float64(p.Y))
	}
	if cell := max(hi[0]-lo[0], hi[1]-lo[1]) / stdmath.Sqrt(float64(len(pts))); cell > 0 {
		return cell
	}
	return 1
}

// eachInBox visits every vertex hashed into a cell the box [x0,y0]–[x1,y1] overlaps.
func (g *vertexCullGrid) eachInBox(x0, y0, x1, y1 float64, visit func(v int)) {
	cx0, cy0 := cellCoord(x0, g.cell), cellCoord(y0, g.cell)
	cx1, cy1 := cellCoord(x1, g.cell), cellCoord(y1, g.cell)
	for x := cx0; x <= cx1; x++ {
		for y := cy0; y <= cy1; y++ {
			g.visitCell([2]int32{x, y}, visit)
		}
	}
}

func (g *vertexCullGrid) visitCell(c [2]int32, visit func(v int)) {
	for _, v := range g.cells[c] {
		visit(int(v))
	}
}
