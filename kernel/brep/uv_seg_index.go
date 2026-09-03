// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/math"
)

// uvSegIndex buckets a chart's tagged (u,v) segments into a uniform grid so recoverEdge tests only the
// segments that can reach the query point, instead of every segment.
//
// The scan it replaces was O(dedges × segments), which is quadratic in the chart's size because both
// grow together. That was tolerable while a wall's frame was two synthetic rims; the loop-framed chart
// (ADR-0060) samples the face's WHOLE boundary into the segment list, and edge recovery became 79% of
// a fine-pitch coil join — 47 s to 253 s on a 7 012-face body (#3459).
//
// The match tolerance is tjTol, orders of magnitude below any cell, so a segment is registered in every
// cell its tjTol-grown box touches and a query visits exactly one cell. A segment whose distance to the
// point is within tjTol necessarily has the point inside that grown box, so the candidate set contains
// every segment the linear scan could have accepted: same answer, bounded cost.
type uvSegIndex struct {
	segs  []uvSeg
	cells map[[2]int][]int
	wide  []int // segments spanning too many cells to register individually
	min   math.Point2
	cw    float64
	ch    float64
}

// uvSegIndexCellCap is how many cells one segment may register in before it is filed as wide instead.
// A seam ruling spans the chart, and registering it cell by cell would cost more than the scan it
// replaces; the wide list is scanned on every query, so it must stay short — which it is, because only
// a chart-spanning segment reaches the cap.
const uvSegIndexCellCap = 16

// newUVSegIndex grids segs. An empty list, or one whose extent is degenerate, yields an index whose
// every query returns the whole list, so the caller needs no special case.
func newUVSegIndex(segs []uvSeg) *uvSegIndex {
	ix := &uvSegIndex{segs: segs, cells: map[[2]int][]int{}}
	if len(segs) == 0 {
		return ix
	}
	lo, hi := uvSegBounds(segs)
	side := stdmath.Ceil(stdmath.Sqrt(float64(len(segs))))
	ix.min = lo
	ix.cw, ix.ch = float64(hi.X-lo.X)/side, float64(hi.Y-lo.Y)/side
	if !(ix.cw > 0) || !(ix.ch > 0) { // a degenerate extent: every query scans the list
		ix.wide = make([]int, len(segs))
		for i := range segs {
			ix.wide[i] = i
		}
		return ix
	}
	for i, s := range segs {
		ix.register(i, s)
	}
	return ix
}

// uvSegBounds is the (u,v) box the segments span.
func uvSegBounds(segs []uvSeg) (lo, hi math.Point2) {
	minX, minY := stdmath.Inf(1), stdmath.Inf(1)
	maxX, maxY := stdmath.Inf(-1), stdmath.Inf(-1)
	for _, s := range segs {
		minX = stdmath.Min(minX, stdmath.Min(float64(s.a.X), float64(s.b.X)))
		maxX = stdmath.Max(maxX, stdmath.Max(float64(s.a.X), float64(s.b.X)))
		minY = stdmath.Min(minY, stdmath.Min(float64(s.a.Y), float64(s.b.Y)))
		maxY = stdmath.Max(maxY, stdmath.Max(float64(s.a.Y), float64(s.b.Y)))
	}
	return math.P2(math.Scalar(minX), math.Scalar(minY)), math.P2(math.Scalar(maxX), math.Scalar(maxY))
}

// register files one segment in every cell its tjTol-grown box touches, or in the wide list when that
// is more cells than the cap allows.
func (ix *uvSegIndex) register(i int, s uvSeg) {
	x0, y0 := ix.cellOf(math.P2(minScalar(s.a.X, s.b.X)-tjTol, minScalar(s.a.Y, s.b.Y)-tjTol))
	x1, y1 := ix.cellOf(math.P2(maxScalar(s.a.X, s.b.X)+tjTol, maxScalar(s.a.Y, s.b.Y)+tjTol))
	if (x1-x0+1)*(y1-y0+1) > uvSegIndexCellCap {
		ix.wide = append(ix.wide, i)
		return
	}
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			ix.cells[[2]int{x, y}] = append(ix.cells[[2]int{x, y}], i)
		}
	}
}

// near lists the segments that can lie within tjTol of p.
func (ix *uvSegIndex) near(p math.Point2) []int {
	if ix.cw == 0 {
		return ix.wide
	}
	x, y := ix.cellOf(p)
	cell := ix.cells[[2]int{x, y}]
	if len(ix.wide) == 0 {
		return cell
	}
	return append(append(make([]int, 0, len(cell)+len(ix.wide)), cell...), ix.wide...)
}

func (ix *uvSegIndex) cellOf(p math.Point2) (int, int) {
	return int(stdmath.Floor(float64(p.X-ix.min.X) / ix.cw)), int(stdmath.Floor(float64(p.Y-ix.min.Y) / ix.ch))
}

func minScalar(a, b math.Scalar) math.Scalar {
	if a < b {
		return a
	}
	return b
}

func maxScalar(a, b math.Scalar) math.Scalar {
	if a > b {
		return a
	}
	return b
}
