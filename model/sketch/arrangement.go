// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Region detection is a planar-subdivision problem: the regions a feature can extrude
// are the minimal closed cells the sketch curves carve the plane into, so a curve that
// divides a shape (an arc splitting a rectangle, two crossing lines, overlapping
// circles) yields several regions, not one. The naive degree-2 chain walk could not do
// this. Instead every curve is faceted into straight segments (the same faceting the
// extrude prism uses — see curvesample.go), every crossing reduces to a segment
// crossing (geom.Segment2dIntersection), and the faceted segments form a planar graph
// (an "arrangement"); faces.go then extracts the cells.

// arrMergeTol is the distance under which two arrangement endpoints are treated as the
// same node. It is coarser than DefaultTolerance so a computed crossing point and a
// curve endpoint that land on it snap together despite float error.
const arrMergeTol = 1e-6 // tol:calibrated — 2D sketch arrangement node merge (exact 2D intersections; see brep arrange2d)

// arrEdge is one straight sub-segment of a faceted curve, between two merged nodes,
// remembering the sketch entity it came from (for the loop's entity list).
type arrEdge struct {
	u, v   int
	entity Entity
}

// arrangement is the planar graph: merged nodes (unique positions) and the straight
// sub-segments connecting them, with a coordinate index for node merging and a set to
// drop duplicate edges (which would break the rotational face walk).
type arrangement struct {
	nodes []math.Point2
	edges []arrEdge
	index map[[2]int64]int
	seen  map[[2]int]bool
}

// buildArrangement facets the entities, splits every facet at its crossings with other
// entities, and merges coincident endpoints into a planar graph.
func buildArrangement(entities []Entity) *arrangement {
	segs := collectSegments(entities)
	cuts := intersectionCuts(segs)
	a := &arrangement{index: map[[2]int64]int{}, seen: map[[2]int]bool{}}
	for i, s := range segs {
		pts := splitPoints(s, cuts[i])
		for k := 0; k+1 < len(pts); k++ {
			a.addEdge(pts[k], pts[k+1], s.entity)
		}
	}
	return a
}

// taggedSeg is a single faceted segment of an entity (the unit the arrangement is
// built from).
type taggedSeg struct {
	a, b   math.Point2
	entity Entity
}

// collectSegments facets every entity into straight segments (closed curves wrap back
// to their first point), each tagged with its source entity.
func collectSegments(entities []Entity) []taggedSeg {
	var out []taggedSeg
	for _, e := range entities {
		pts, closed := entityPolyline(e)
		for k := 0; k+1 < len(pts); k++ {
			out = append(out, taggedSeg{pts[k], pts[k+1], e})
		}
		if closed && len(pts) >= 2 {
			out = append(out, taggedSeg{pts[len(pts)-1], pts[0], e})
		}
	}
	return out
}

// entityPolyline returns an entity's faceted polyline and whether it is inherently
// closed (so the caller wraps it). Reuses the curve samplers in curvesample.go.
func entityPolyline(e Entity) (pts []math.Point2, closed bool) {
	switch t := e.(type) {
	case *Circle:
		return sampleCircle(t), true
	case *Ellipse:
		return sampleEllipseEntity(t), true
	case *Spline:
		if t.Closed {
			return sampleSplineEntity(t), true
		}
		return naturalPolyline(t), false
	default:
		return naturalPolyline(e), false // Line, Arc, open Spline
	}
}

// cut is a crossing along a segment: the parameter t∈[0,1] where it occurs and the
// canonical crossing point (shared by both segments so they merge to one node).
type cut struct {
	param float64
	pt    math.Point2
}

// arrBruteMax is the segment count below which intersectionCuts compares every pair
// directly; above it a grid broad-phase is used, since all-pairs is O(n²) and a dense
// imported drawing has hundreds of thousands of segments.
const arrBruteMax = 1024

// intersectionCuts finds, for every segment, the points where other entities' segments
// cross it. Facets of the same entity are skipped (they only share endpoints, they do
// not cross). A grid broad-phase keeps it tractable on large sketches: two segments
// can only intersect if their bounding boxes overlap, and overlapping boxes always
// share a grid cell, so testing only same-cell pairs is exact (no crossing is missed).
func intersectionCuts(segs []taggedSeg) [][]cut {
	cuts := make([][]cut, len(segs))
	if len(segs) <= arrBruteMax {
		for i := 0; i < len(segs); i++ {
			for j := i + 1; j < len(segs); j++ {
				cutPair(segs, i, j, cuts)
			}
		}
		return cuts
	}
	gridCuts(segs, cuts)
	return cuts
}

// cutPair tests one segment pair and records the crossing (if any) on both.
func cutPair(segs []taggedSeg, i, j int, cuts [][]cut) {
	if segs[i].entity == segs[j].entity {
		return
	}
	pt, s, t, ok := geom.Segment2dIntersection(
		geom.NewLineSegment2d(segs[i].a, segs[i].b),
		geom.NewLineSegment2d(segs[j].a, segs[j].b), arrMergeTol,
	)
	if !ok {
		return
	}
	cuts[i] = append(cuts[i], cut{s, pt})
	cuts[j] = append(cuts[j], cut{t, pt})
}

// gridCuts bins segments into a ~sqrt(n)×sqrt(n) grid by their bounding box and tests
// only pairs that share a cell — each pair once (deduped, since a long segment spans
// several cells). Exact: AABB-overlapping (hence possibly-intersecting) segments
// always co-occupy a cell.
//
//nolint:funlen // spatial-grid binning of segment AABBs; length is the binning, not logic.
func gridCuts(segs []taggedSeg, cuts [][]cut) {
	minX, minY, maxX, maxY := segsBounds(segs)
	dim := gridDim(len(segs))
	cell := stdmath.Max(stdmath.Max(maxX-minX, maxY-minY)/float64(dim), 1e-9) // tol:numeric — degenerate spatial-grid cell floor
	cols := int((maxX-minX)/cell) + 1
	rows := int((maxY-minY)/cell) + 1
	col := func(x float64) int { return math.Clamp(int((x-minX)/cell), 0, cols-1) }
	row := func(y float64) int { return math.Clamp(int((y-minY)/cell), 0, rows-1) }
	bins := binSegments(segs, col, row, cols, rows)
	testBinnedPairs(segs, bins, cuts)
}

// segsBounds returns the axis-aligned extent of the segment endpoints.
func segsBounds(segs []taggedSeg) (minX, minY, maxX, maxY float64) {
	minX, minY = stdmath.Inf(1), stdmath.Inf(1)
	maxX, maxY = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, s := range segs {
		minX, minY = stdmath.Min(minX, stdmath.Min(s.a.X, s.b.X)), stdmath.Min(minY, stdmath.Min(s.a.Y, s.b.Y))
		maxX, maxY = stdmath.Max(maxX, stdmath.Max(s.a.X, s.b.X)), stdmath.Max(maxY, stdmath.Max(s.a.Y, s.b.Y))
	}
	return minX, minY, maxX, maxY
}

// gridDim picks the uniform-grid resolution from the segment count, clamped to [1, 2048].
func gridDim(n int) int {
	dim := int(stdmath.Sqrt(float64(n)))
	if dim < 1 {
		return 1
	}
	if dim > 2048 {
		return 2048
	}
	return dim
}

// binSegments buckets each segment into every grid cell its bounding box touches.
func binSegments(segs []taggedSeg, col, row func(float64) int, cols, rows int) [][]int32 {
	bins := make([][]int32, cols*rows)
	for i, s := range segs {
		for cj := row(stdmath.Min(s.a.Y, s.b.Y)); cj <= row(stdmath.Max(s.a.Y, s.b.Y)); cj++ {
			for ci := col(stdmath.Min(s.a.X, s.b.X)); ci <= col(stdmath.Max(s.a.X, s.b.X)); ci++ {
				bins[cj*cols+ci] = append(bins[cj*cols+ci], int32(i))
			}
		}
	}
	return bins
}

// testBinnedPairs cuts every distinct co-binned segment pair exactly once (a pair can share
// several cells, so a tested-set deduplicates).
func testBinnedPairs(segs []taggedSeg, bins [][]int32, cuts [][]cut) {
	tested := map[[2]int32]struct{}{}
	for _, bin := range bins {
		for x := 0; x < len(bin); x++ {
			for y := x + 1; y < len(bin); y++ {
				i, j := bin[x], bin[y]
				if i > j {
					i, j = j, i
				}
				if _, seen := tested[[2]int32{i, j}]; seen {
					continue
				}
				tested[[2]int32{i, j}] = struct{}{}
				cutPair(segs, int(i), int(j), cuts)
			}
		}
	}
}

// splitPoints returns a segment's points in order: its start, the interior crossing
// points (sorted, endpoint-coincident ones dropped), and its end.
func splitPoints(s taggedSeg, cuts []cut) []math.Point2 {
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].param < cuts[j].param })
	pts := []math.Point2{s.a}
	for _, c := range cuts {
		if c.param <= arrMergeTol || c.param >= 1-arrMergeTol {
			continue // a crossing at an endpoint splits nothing
		}
		if pts[len(pts)-1].IsEqualTo(c.pt, arrMergeTol) {
			continue
		}
		pts = append(pts, c.pt)
	}
	return append(pts, s.b)
}

// node returns the index of the merged node at p, creating it on first sight. Points
// within arrMergeTol quantize to the same grid cell and so share a node.
func (a *arrangement) node(p math.Point2) int {
	key := [2]int64{
		int64(stdmath.Round(p.X / arrMergeTol)),
		int64(stdmath.Round(p.Y / arrMergeTol)),
	}
	if id, ok := a.index[key]; ok {
		return id
	}
	id := len(a.nodes)
	a.nodes = append(a.nodes, p)
	a.index[key] = id
	return id
}

// addEdge adds an undirected edge between the nodes at p and q, dropping zero-length
// and duplicate edges (a repeated node pair would give the face walk two identical
// rotational directions).
func (a *arrangement) addEdge(p, q math.Point2, e Entity) {
	u, v := a.node(p), a.node(q)
	if u == v {
		return
	}
	key := [2]int{u, v}
	if u > v {
		key = [2]int{v, u}
	}
	if a.seen[key] {
		return
	}
	a.seen[key] = true
	a.edges = append(a.edges, arrEdge{u, v, e})
}
