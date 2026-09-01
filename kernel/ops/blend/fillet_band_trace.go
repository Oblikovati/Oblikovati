// SPDX-License-Identifier: GPL-2.0-only

package blend

// Tracing the surviving region's boundary — the MARCH of the band-imprint walk.
//
// Once the obstacles' cuts have arranged the band's ideal box into a grid and every cell has been
// classified (fillet_band_imprint.go), the band the fillet keeps is the union of the surviving cells.
// Its boundary is what every consumer needs: the band's own rebuilt loop, and — run by run — the
// contact chain each face the band runs into must absorb.
//
// The march is deliberately the textbook one for a cell union: emit one directed grid edge per
// solid/not-solid cell wall, wound so the traversal runs +u along the v = 0 side, +v up the u = uMax
// side, −u back along v = vMax and −v down u = 0 — i.e. the SAME cycle cylinderFace already emits for
// the ideal box, so a region with no obstacle in it retraces the existing loop exactly. The walk then
// chains those edges and merges collinear neighbours, so a side crossed by a grid line the region does
// not actually turn at stays ONE run (and the face under it keeps the single edge it has today).
//
// Anything that is not one simple closed cycle — a region in two pieces, a hole in the middle of the
// band, a pinch where two corners of the region meet at a point — is DECLINED. Those are real shapes a
// blend can have, but each needs a loop structure this slice does not build, and a guessed one is a
// silently wrong solid.

// bandGridCorner is a corner of the cut arrangement, as indices into the u and v grid lines.
type bandGridCorner struct{ i, j int }

// bandGridEdge is one directed wall of the surviving region, corner to corner.
type bandGridEdge struct{ a, b bandGridCorner }

// bandTraceRegion walks the surviving cells' boundary into an ordered, closed list of runs.
// ok=false when the boundary is not exactly one simple cycle (§ file comment).
func bandTraceRegion(solid [][]bool, us, vs []float64) ([]bandRun, bool) {
	edges := bandRegionWalls(solid)
	if len(edges) < 4 {
		return nil, false
	}
	cycle, ok := bandChainWalls(edges)
	if !ok {
		return nil, false
	}
	return bandMergeRuns(cycle, us, vs), true
}

// bandRegionWalls emits one directed edge per wall between a surviving cell and a non-surviving
// neighbour (or the outside), wound counter-clockwise in the chart.
func bandRegionWalls(solid [][]bool) []bandGridEdge {
	var out []bandGridEdge
	for i := range solid {
		for j := range solid[i] {
			if !solid[i][j] {
				continue
			}
			out = append(out, bandCellWalls(solid, i, j)...)
		}
	}
	return out
}

// bandCellWalls is one surviving cell's contribution: each of its four walls that faces a
// non-surviving neighbour, directed so the cell lies to the walk's left.
func bandCellWalls(solid [][]bool, i, j int) []bandGridEdge {
	var out []bandGridEdge
	if !bandCellSolid(solid, i, j-1) {
		out = append(out, bandGridEdge{bandGridCorner{i, j}, bandGridCorner{i + 1, j}})
	}
	if !bandCellSolid(solid, i+1, j) {
		out = append(out, bandGridEdge{bandGridCorner{i + 1, j}, bandGridCorner{i + 1, j + 1}})
	}
	if !bandCellSolid(solid, i, j+1) {
		out = append(out, bandGridEdge{bandGridCorner{i + 1, j + 1}, bandGridCorner{i, j + 1}})
	}
	if !bandCellSolid(solid, i-1, j) {
		out = append(out, bandGridEdge{bandGridCorner{i, j + 1}, bandGridCorner{i, j}})
	}
	return out
}

// bandCellSolid is solid[i][j] with the grid's outside reading as not-solid.
func bandCellSolid(solid [][]bool, i, j int) bool {
	if i < 0 || i >= len(solid) || j < 0 || j >= len(solid[i]) {
		return false
	}
	return solid[i][j]
}

// bandChainWalls orders the walls into one closed cycle. ok=false when a corner has more than one
// wall leaving it (a pinch) or when the cycle does not consume every wall (two loops, or a hole).
func bandChainWalls(edges []bandGridEdge) ([]bandGridEdge, bool) {
	next := make(map[bandGridCorner]bandGridEdge, len(edges))
	for _, e := range edges {
		if _, dup := next[e.a]; dup {
			return nil, false
		}
		next[e.a] = e
	}
	cycle := []bandGridEdge{edges[0]}
	for at := edges[0].b; at != edges[0].a; at = cycle[len(cycle)-1].b {
		e, ok := next[at]
		if !ok || len(cycle) > len(edges) {
			return nil, false
		}
		cycle = append(cycle, e)
	}
	return cycle, len(cycle) == len(edges)
}

// bandMergeRuns turns the ordered walls into chart runs, merging every collinear neighbour so a side
// the region does not actually turn at stays a single run.
func bandMergeRuns(cycle []bandGridEdge, us, vs []float64) []bandRun {
	var out []bandRun
	for _, e := range cycle {
		r := bandRunOf(e, us, vs)
		if n := len(out); n > 0 && out[n-1].constU == r.constU && out[n-1].at == r.at {
			out[n-1].to = r.to
			continue
		}
		out = append(out, r)
	}
	return bandCloseRunCycle(out)
}

// bandRunOf is one grid wall as a chart run.
func bandRunOf(e bandGridEdge, us, vs []float64) bandRun {
	if e.a.i == e.b.i {
		return bandRun{constU: true, at: us[e.a.i], from: vs[e.a.j], to: vs[e.b.j]}
	}
	return bandRun{at: vs[e.a.j], from: us[e.a.i], to: us[e.b.i]}
}

// bandCloseRunCycle folds the cycle's last run into its first when the two are collinear — the merge
// above cannot see across the list's own seam.
func bandCloseRunCycle(runs []bandRun) []bandRun {
	n := len(runs)
	if n < 2 || runs[n-1].constU != runs[0].constU || runs[n-1].at != runs[0].at {
		return runs
	}
	runs[0].from = runs[n-1].from
	return runs[:n-1]
}
