// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"sort"

	"oblikovati.org/math"
)

// trimScan answers the interior grid's five-probe clear-of-trim test ((u,v), (u±m,v), (u,v±m) all
// strictly inside) from per-row crossing lists instead of a per-probe polygon walk. The walk is
// O(boundary) per probe, which was fine under the old 80×80 grid cap but is quadratic-ish once the
// grid honours the chord tolerance (a giant patch's PropertyQuality grid probes ~10^5–10^6 candidates
// against ~10^3 boundary points — patchgridcap-report.md §cost). Crossings are computed with
// pointInUVPoly's exact per-edge arithmetic and the same strict u < x parity, so the accepted point
// set is bit-identical to the walk's — proven by every unsaturated face's fingerprint pin holding.
type trimScan struct {
	outer  []math.Point2
	holes  [][]math.Point2
	margin float64
	rows   map[float64]trimScanRow // lazily built: the grid revisits each row v once per column
}

// trimScanRow is one horizontal line's sorted polygon crossings: the outer loop's and each hole's.
type trimScanRow struct {
	outer []float64
	holes [][]float64
}

// newTrimScan builds the scan for one patch's chart-space trim (outer loop + holes) at probe margin m.
func newTrimScan(outer []math.Point2, holes [][]math.Point2, margin float64) *trimScan {
	return &trimScan{outer: outer, holes: holes, margin: margin, rows: map[float64]trimScanRow{}}
}

// clear reports whether p and its four margin neighbours are all inside the trim — the same five
// probes clearOfTrim makes, so no accepted node sits on or just past the boundary.
func (s *trimScan) clear(p [2]float64) bool {
	m := s.margin
	return s.inside(p[0], p[1]) && s.inside(p[0]+m, p[1]) && s.inside(p[0]-m, p[1]) &&
		s.inside(p[0], p[1]+m) && s.inside(p[0], p[1]-m)
}

// inside is insideUVTrim's predicate (inside the outer loop, outside every hole) answered from the
// row's crossing lists.
func (s *trimScan) inside(u, v float64) bool {
	r := s.row(v)
	if !oddCrossingsRightOf(r.outer, u) {
		return false
	}
	for _, h := range r.holes {
		if oddCrossingsRightOf(h, u) {
			return false
		}
	}
	return true
}

// row returns (building on first use) the crossing lists for the horizontal line at v. Rows are keyed
// by the exact float64 v: the grid computes each row's v identically for every column, so a row is
// built once and reused laidU times.
func (s *trimScan) row(v float64) trimScanRow {
	if r, ok := s.rows[v]; ok {
		return r
	}
	r := trimScanRow{outer: rowCrossings(s.outer, v)}
	for _, h := range s.holes {
		r.holes = append(r.holes, rowCrossings(h, v))
	}
	s.rows[v] = r
	return r
}

// rowCrossings returns the u-coordinates where poly's edges cross the horizontal line at v — the same
// edge selection ((yi > v) != (yj > v)) and the same crossing expression pointInUVPoly evaluates, so
// parity classification is bit-identical to the walk it replaces.
func rowCrossings(poly []math.Point2, v float64) []float64 {
	var xs []float64
	n := len(poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		yi, yj := float64(poly[i].Y), float64(poly[j].Y)
		if (yi > v) != (yj > v) {
			xi, xj := float64(poly[i].X), float64(poly[j].X)
			xs = append(xs, (xj-xi)*(v-yi)/(yj-yi)+xi)
		}
	}
	sort.Float64s(xs)
	return xs
}

// oddCrossingsRightOf reports whether an odd number of crossings lie STRICTLY right of u — exactly
// pointInUVPoly's parity, whose per-edge toggle fires on u < x.
func oddCrossingsRightOf(xs []float64, u float64) bool {
	i := sort.SearchFloat64s(xs, u)
	for i < len(xs) && xs[i] == u {
		i++ // x == u does not toggle (the walk's comparison is strict)
	}
	return (len(xs)-i)%2 == 1
}
