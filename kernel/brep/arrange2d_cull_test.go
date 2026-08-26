// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdrand "math/rand"
	"reflect"
	"sort"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// randomArrangementSegments builds a reproducible stress mix for the arrangement: random
// chords, grid-snapped segments (exact coincidences and shared endpoints), and axis-aligned
// runs that end ON other segments' interiors (T-junctions) — the configurations the welder,
// splitOne and splitTJunctions each trip on.
func randomArrangementSegments(rng *stdrand.Rand, n int) [][2]math.Point2 {
	segs := make([][2]math.Point2, 0, n)
	snap := func(v float64) float64 { return float64(int(v*2)) / 2 } // 0.5 grid
	for len(segs) < n {
		a := math.P2(rng.Float64()*10, rng.Float64()*10)
		b := math.P2(rng.Float64()*10, rng.Float64()*10)
		switch len(segs) % 3 {
		case 1: // snapped: forces coincident endpoints and exact crossings
			a, b = math.P2(snap(float64(a.X)), snap(float64(a.Y))), math.P2(snap(float64(b.X)), snap(float64(b.Y)))
		case 2: // axis-aligned through the snapped lattice: forces T-junctions
			b = math.P2(float64(a.X), snap(float64(b.Y)))
		}
		if a.DistanceTo(b) > 0 {
			segs = append(segs, [2]math.Point2{a, b})
		}
	}
	return segs
}

// bruteNarrowPhasePairs returns every (i, j) pair the splitOne narrow phase accepts — the set
// the grid candidates must contain.
func bruteNarrowPhasePairs(segments [][2]math.Point2) [][2]int {
	var pairs [][2]int
	for i, seg := range segments {
		si := geom.NewLineSegment2d(seg[0], seg[1])
		for j, other := range segments {
			if j == i {
				continue
			}
			if _, s, _, ok := geom.Segment2dIntersection(si, geom.NewLineSegment2d(other[0], other[1]), arrTol); ok && s > arrTol && s < 1-arrTol {
				pairs = append(pairs, [2]int{i, j})
			}
		}
	}
	return pairs
}

// TestSegmentCullGridSupersetOfNarrowPhase is the culling-soundness gate #1607 requires: on
// randomized fixtures, the grid candidate set must contain every pair the brute narrow phase
// accepts — a miss would silently drop an arrangement crossing.
func TestSegmentCullGridSupersetOfNarrowPhase(t *testing.T) {
	rng := stdrand.New(stdrand.NewSource(7))
	for trial := range 20 {
		segments := randomArrangementSegments(rng, 60)
		cull := newSegmentCullGrid(segments)
		cand := make([][]int, len(segments))
		for i := range segments {
			cand[i] = cull.candidates(i)
		}
		for _, pair := range bruteNarrowPhasePairs(segments) {
			if sort.SearchInts(cand[pair[0]], pair[1]) == len(cand[pair[0]]) ||
				cand[pair[0]][sort.SearchInts(cand[pair[0]], pair[1])] != pair[1] {
				t.Fatalf("trial %d: narrow-phase pair %v missing from grid candidates %v", trial, pair, cand[pair[0]])
			}
		}
	}
}

// brutePlanarize is the retired O(S²) planarize, kept as the oracle: same welder, same
// narrow phase, all-pairs candidacy, brute T-junction scan.
func brutePlanarize(segments [][2]math.Point2) ([]math.Point2, [][2]int) {
	weld := newWelder()
	edges := map[[2]int]bool{}
	all := make([]int, len(segments))
	for i := range all {
		all[i] = i
	}
	for i, seg := range segments {
		cand := append(append([]int{}, all[:i]...), all[i+1:]...)
		for _, e := range splitOne(seg, segments, cand, weld) {
			if e[0] != e[1] {
				edges[canonEdge(e[0], e[1])] = true
			}
		}
	}
	bruteSplitTJunctions(weld.points, edges)
	out := make([][2]int, 0, len(edges))
	for e := range edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return weld.points, out
}

// bruteSplitTJunctions is the retired full-scan T-junction pass (vertexOnEdgeInterior with an
// all-vertices "grid" — one giant cell — degrades to exactly the old scan).
func bruteSplitTJunctions(pts []math.Point2, edges map[[2]int]bool) {
	verts := &vertexCullGrid{cell: 1e30, cells: map[[2]int32][]int32{}} // tol:numeric — one-giant-cell degradation, not a length tolerance
	for i := range pts {
		verts.cells[[2]int32{cellCoord(float64(pts[i].X), verts.cell), cellCoord(float64(pts[i].Y), verts.cell)}] =
			append(verts.cells[[2]int32{cellCoord(float64(pts[i].X), verts.cell), cellCoord(float64(pts[i].Y), verts.cell)}], int32(i))
	}
	for changed := true; changed; {
		changed = false
		for e := range edges {
			c := vertexOnEdgeInterior(pts, e[0], e[1], verts)
			if c < 0 {
				continue
			}
			delete(edges, e)
			edges[canonEdge(e[0], c)] = true
			edges[canonEdge(c, e[1])] = true
			changed = true
		}
	}
}

// TestPlanarizeGridMatchesBrute pins bit-identity of the whole culled arrangement front end
// against the brute oracle on randomized fixtures: same welded points (same order), same
// elementary edge set.
func TestPlanarizeGridMatchesBrute(t *testing.T) {
	rng := stdrand.New(stdrand.NewSource(11))
	for trial := range 20 {
		segments := randomArrangementSegments(rng, 80)
		gotPts, gotEdges := planarize(segments)
		wantPts, wantEdges := brutePlanarize(segments)
		if !reflect.DeepEqual(gotPts, wantPts) {
			t.Fatalf("trial %d: welded points differ (%d vs %d)", trial, len(gotPts), len(wantPts))
		}
		if !reflect.DeepEqual(gotEdges, wantEdges) {
			t.Fatalf("trial %d: elementary edges differ (%d vs %d)", trial, len(gotEdges), len(wantEdges))
		}
	}
}

// TestVertexOnEdgeInteriorGridMatchesFullScan pins the T-junction candidate culling: for
// random welded point sets and edges, the grid-backed lowest-qualifying-vertex answer equals
// the full scan's (via the one-giant-cell degradation).
func TestVertexOnEdgeInteriorGridMatchesFullScan(t *testing.T) {
	rng := stdrand.New(stdrand.NewSource(13))
	for trial := range 20 {
		pts, edges := planarize(randomArrangementSegments(rng, 40))
		grid := newVertexCullGrid(pts)
		full := &vertexCullGrid{cell: 1e30, cells: map[[2]int32][]int32{}} // tol:numeric — one-giant-cell degradation, not a length tolerance
		for i := range pts {
			c := [2]int32{cellCoord(float64(pts[i].X), full.cell), cellCoord(float64(pts[i].Y), full.cell)}
			full.cells[c] = append(full.cells[c], int32(i))
		}
		for _, e := range edges {
			if got, want := vertexOnEdgeInterior(pts, e[0], e[1], grid), vertexOnEdgeInterior(pts, e[0], e[1], full); got != want {
				t.Fatalf("trial %d edge %v: grid found vertex %d, full scan %d", trial, e, got, want)
			}
		}
	}
}
