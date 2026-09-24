// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/math"
)

// A slab index answers chartRegion.covers without walking every edge of every contour.
//
// covers runs at the centroid of every triangle the covering proposes, and each call walked every edge
// of every contour under every period shift: O(triangles × contour edges). Profiled on occtparity's
// TestWaveETorusRimWatertight it was 12% of the run, all of it inside pointInUVPoly.
//
// The even-odd rule only counts an edge that STRADDLES the query's horizontal line, and it combines
// the counts by parity — an XOR, which no visiting order can change. An edge straddles y exactly when
// min(yi,yj) <= y < max(yi,yj). So each edge is filed under every horizontal slab its [min,max] touches,
// and a query reads only its own slab. The slab of a value is floor((y-y0)·inv) clamped — a monotone
// function of y computed the same way for edges and queries — so a straddling edge's slab range always
// contains the query's slab, and the edges read are a superset of the edges that count. Each is then
// asked edgeCrossesRightOf, the same function the unindexed rule asks, so the answer is bit-identical.

// contourSlabs files one contour's edges by horizontal slab. Edge i runs poly[i-1] → poly[i], wrapping,
// which is the order pointInUVPoly walks them in.
type contourSlabs struct {
	y0, inv float64
	edges   [][]int32
}

// newContourSlabs indexes one contour. ok is false for a contour it cannot index exactly — fewer than
// three points, or a non-finite coordinate, for which floor(·) has no defined slab — and covers then
// asks that contour the unindexed way.
func newContourSlabs(poly []math.Point2) (contourSlabs, bool) {
	yLo, yHi, ok := finiteYRange(poly)
	if !ok || len(poly) < 3 {
		return contourSlabs{}, false
	}
	n := len(poly) // one slab per edge keeps a slab's share of a fine contour to a handful of edges
	s := contourSlabs{y0: yLo, edges: make([][]int32, n)}
	if yHi > yLo {
		s.inv = float64(n) / (yHi - yLo)
	}
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		a, b := float64(poly[j].Y), float64(poly[i].Y)
		for k := s.slab(min(a, b)); k <= s.slab(max(a, b)); k++ {
			s.edges[k] = append(s.edges[k], int32(i))
		}
	}
	return s, true
}

// finiteYRange is the contour's y extent, and false when any coordinate is not finite.
func finiteYRange(poly []math.Point2) (lo, hi float64, ok bool) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, q := range poly {
		x, y := float64(q.X), float64(q.Y)
		if stdmath.IsNaN(x) || stdmath.IsInf(x, 0) || stdmath.IsNaN(y) || stdmath.IsInf(y, 0) {
			return 0, 0, false
		}
		lo, hi = min(lo, y), max(hi, y)
	}
	return lo, hi, true
}

// slab is floor((y-y0)·inv) clamped into the index. It is monotone in y, which is the whole of the
// exactness argument above, and it is only ever called with a finite y.
func (s contourSlabs) slab(y float64) int {
	f := stdmath.Floor((y - s.y0) * s.inv)
	if f <= 0 {
		return 0
	}
	if last := float64(len(s.edges) - 1); f >= last {
		return len(s.edges) - 1
	}
	return int(f)
}

// contains is pointInUVPoly(poly, p), answered from the slab p falls in.
//
//	s, _ := newContourSlabs(poly); in := s.contains(poly, p) // == pointInUVPoly(poly, p)
func (s contourSlabs) contains(poly []math.Point2, p [2]float64) bool {
	// A non-finite query straddles nothing — every comparison against it agrees — so the unindexed
	// rule says "outside"; saying so here keeps floor(·) away from a value with no slab.
	if stdmath.IsNaN(p[1]) || stdmath.IsInf(p[1], 0) {
		return false
	}
	in := false
	n := len(poly)
	for _, i := range s.edges[s.slab(p[1])] {
		if edgeCrossesRightOf(poly, int(i), (int(i)+n-1)%n, p) {
			in = !in
		}
	}
	return in
}
