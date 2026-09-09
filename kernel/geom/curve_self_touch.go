// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Where two curves meet without crossing, and where one comes back to a point it has already visited
// (ADR-0061 stage 2, ADR-0062).
//
// A section may TOUCH. A plane grazing a tilted torus's inner equator cuts a spiric figure-eight whose
// two lobes meet at one point — delivered as one self-touching curve or as two curves that touch,
// depending on which path produced it. Either way the meeting is an incidence like every other, and an
// arrangement has to carry a vertex there. Without one the boundary walks straight through and the two
// lobes come out as a single circuit wound oppositely, whose boundary integral cancels: measured, a lid
// of two 25.267 lobes integrating to 1.41e-06.
//
// It cannot be found by sampling. These touches are TANGENTIAL — the branches meet with equal tangents,
// nothing crosses — so a sampled polyline does not intersect there, it only comes close: on the
// corpus's oblique figure-eight the nearest approach between two non-adjacent samples is 1.03e-07
// against a weld grid of 1e-07, just wide enough to miss. Widening the grid would be an epsilon
// standing in for a solve. This solves it.

// curveTouchScan is how many stations the coarse pass walks per curve. It only has to bracket each
// touch in one sampling interval, so it is a bracketing density rather than an accuracy parameter —
// the refinement supplies the accuracy.
const curveTouchScan = 128

// curveTouchGap is the smallest parameter separation a pair on ONE curve may have and still count as a
// return rather than as the curve's own neighbourhood. Below it two stations are close because the
// curve is continuous, not because it comes back.
const curveTouchGap = 4.0 / curveTouchScan

// CurveSelfTouch returns the parameters of one point the curve visits twice, within tol. ok is false
// when it does not come back to itself, which is the ordinary case.
//
// Example:
//
//	if a, b, ok := geom.CurveSelfTouch(section, res.Weld()); ok {
//		arcs = []span{{lo, a}, {a, b}, {b, hi}} // the touch is now an endpoint
//	}
func CurveSelfTouch(c Curve3, tol float64) (a, b float64, ok bool) {
	hits := CurveTouches(c, c, true, tol)
	if len(hits) == 0 {
		return 0, 0, false
	}
	return hits[0][0], hits[0][1], true
}

// CurveTouches returns every point at which two curves meet within tol, as the parameter on each.
//
// Set same when both arguments are the SAME curve: the pair is then required to be a genuine return,
// which is what excludes a curve's own neighbourhood and, on a closed curve, its closure. Between
// DIFFERENT curves every meeting counts, including one at an endpoint they already share — a caller
// that has made that a vertex already can drop it.
//
// Example:
//
//	for _, hit := range geom.CurveTouches(ovalA, ovalB, false, res.Weld()) { … }
func CurveTouches(a, b Curve3, same bool, tol float64) [][2]float64 {
	aLo, aHi := a.Domain()
	bLo, bHi := b.Domain()
	if !(aHi > aLo) || !(bHi > bLo) || tol <= 0 {
		return nil
	}
	if !same && curvesAreApart(a, b, tol) {
		return nil // their extents are apart: no sampling, no refinement
	}
	pa, pb := scanCurveStations(a, aLo, aHi), scanCurveStations(b, bLo, bHi)
	// A candidate can only close by what one sampling interval of each curve spans, so anything farther
	// apart than that cannot refine to within tol however hard it is searched. Discarding those first is
	// exact — no candidate that could succeed is dropped — and it is what keeps the refinement off the
	// overwhelming majority of pairs, which do not meet at all.
	reach := tol + widestStation(pa) + widestStation(pb)
	var out [][2]float64
	for _, cell := range distinctCells(touchCandidateCells(pa, pb, same, CurveIsClosed(a)), pa, pb, reach) {
		ta, tb := refineTouch(a, b,
			aLo+(aHi-aLo)*float64(cell[0])/curveTouchScan, bLo+(bHi-bLo)*float64(cell[1])/curveTouchScan,
			(aHi-aLo)/curveTouchScan, (bHi-bLo)/curveTouchScan)
		if float64(a.PointAt(ta).DistanceTo(b.PointAt(tb))) <= tol {
			out = appendDistinctTouch(out, ta, tb, (aHi-aLo)/curveTouchScan, (bHi-bLo)/curveTouchScan)
		}
	}
	return out
}

// appendDistinctTouch adds a meeting unless one already found is the SAME meeting.
//
// Neighbouring scan cells bracket one meeting between them and each refines onto it, so without this
// every meeting is reported once per cell that saw it. Two curves that run close — the two lobes of a
// figure-eight either side of their pinch — produce a whole ridge of such cells, and the caller that
// splits an arc at every reported parameter split ONE island into 130 000 arcs and walked 8.3 million
// samples of it (ADR-0061 stage 2). Same meeting means same parameters, so the test is the scan
// interval each was bracketed in.
func appendDistinctTouch(out [][2]float64, ta, tb, aStep, bStep float64) [][2]float64 {
	for _, had := range out {
		if stdmath.Abs(had[0]-ta) <= aStep && stdmath.Abs(had[1]-tb) <= bStep {
			return out
		}
	}
	return append(out, [2]float64{ta, tb})
}

// widestStation is the largest gap between consecutive scan stations — how far a point can travel
// inside one sampling interval.
func widestStation(pts []math.Point3) float64 {
	widest := 0.0
	for i := 1; i < len(pts); i++ {
		widest = stdmath.Max(widest, float64(pts[i-1].DistanceTo(pts[i])))
	}
	return widest
}

// curveCullStations is the coarse sample the extent cull is taken from. It only has to bound the curve,
// which the inflation below makes safe, so it is far cheaper than the scan it saves.
const curveCullStations = 16

// curvesAreApart reports two curves whose EXTENTS are far enough apart that no point of one can be
// within tol of the other. It is the cull that keeps the scan and the refinement off the pairs that do
// not meet, which is nearly all of them: without it kernel/brep went from 33 s to a 1800 s timeout.
//
// The bound is conservative. Each box is taken from curveCullStations samples and inflated by the
// widest gap between them, which covers any excursion the curve makes between two samples — for a
// chord the deviation is at most half that.
func curvesAreApart(a, b Curve3, tol float64) bool {
	aBox, aPad := curveExtent(a)
	bBox, bPad := curveExtent(b)
	pad := math.Scalar(aPad + bPad + tol)
	grown := aBox.ExtendPoint(aBox.Min.TranslateBy(math.V3(-1, -1, -1).Scale(pad))).
		ExtendPoint(aBox.Max.TranslateBy(math.V3(1, 1, 1).Scale(pad)))
	return !grown.Intersects(bBox)
}

// curveExtent boxes a curve from a coarse sample, with the widest gap between samples alongside.
func curveExtent(c Curve3) (math.Box, float64) {
	lo, hi := c.Domain()
	box := math.EmptyBox()
	widest, prev := 0.0, c.PointAt(lo)
	for i := 0; i <= curveCullStations; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/curveCullStations)
		box = box.ExtendPoint(p)
		widest = stdmath.Max(widest, float64(prev.DistanceTo(p)))
		prev = p
	}
	return box, widest
}

// scanCurveStations evaluates the coarse pass's stations.
func scanCurveStations(c Curve3, lo, hi float64) []math.Point3 {
	pts := make([]math.Point3, curveTouchScan+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/curveTouchScan)
	}
	return pts
}

// distinctCells keeps the candidates that are within reach and drops those adjacent to one already
// kept, BEFORE any of them is refined.
//
// Neighbouring cells bracket the same meeting, and where two curves run close — the lobes of a
// figure-eight either side of their pinch — a whole ridge of them qualifies. Refining each costs
// thousands of curve evaluations and they all converge on the same point, so collapsing the ridge here
// rather than deduplicating the answers afterwards is what makes the search affordable: measured, the
// refinement was 34% of a 51 s figure-eight cut with the trig and allocation it drives making up most
// of the rest.
func distinctCells(cells [][2]int, pa, pb []math.Point3, reach float64) [][2]int {
	out := make([][2]int, 0, len(cells))
	for _, cell := range cells {
		if float64(pa[cell[0]].DistanceTo(pb[cell[1]])) > reach || adjacentToKept(out, cell) {
			continue
		}
		out = append(out, cell)
	}
	return out
}

// adjacentToKept reports a cell neighbouring one already kept, which brackets the same meeting.
func adjacentToKept(kept [][2]int, cell [2]int) bool {
	for _, had := range kept {
		if abs(had[0]-cell[0]) <= 1 && abs(had[1]-cell[1]) <= 1 {
			return true
		}
	}
	return false
}

// abs is the integer magnitude, for the cell adjacency test.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// touchCandidateCells returns the station pairs that are a local minimum of the separation — each one
// brackets at most one touch. A minimum is taken over the eight neighbouring cells, so two curves that
// run close for a while contribute one candidate rather than a run of them.
func touchCandidateCells(pa, pb []math.Point3, same, closed bool) [][2]int {
	n := len(pa) - 1
	var out [][2]int
	for i := 0; i <= n; i++ {
		for j := 0; j <= n; j++ {
			if same && (j <= i || !stationsReturn(i, j, n, closed)) {
				continue
			}
			if isSeparationMinimum(pa, pb, i, j, n) {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// isSeparationMinimum reports whether cell (i, j) is no farther apart than any of its neighbours.
func isSeparationMinimum(pa, pb []math.Point3, i, j, n int) bool {
	here := float64(pa[i].DistanceTo(pb[j]))
	for di := -1; di <= 1; di++ {
		for dj := -1; dj <= 1; dj++ {
			x, y := i+di, j+dj
			if x < 0 || y < 0 || x > n || y > n || (di == 0 && dj == 0) {
				continue
			}
			if float64(pa[x].DistanceTo(pb[y])) < here {
				return false
			}
		}
	}
	return true
}

// stationsReturn reports whether two stations of ONE curve are separated enough in parameter that
// meeting is a return rather than continuity. On a CLOSED curve the separation is the shorter way
// round, which is what excludes the closure itself — its first and last stations are the same point.
func stationsReturn(i, j, n int, closed bool) bool {
	gap := float64(j-i) / float64(n)
	if closed && 1-gap < gap {
		gap = 1 - gap
	}
	return gap >= curveTouchGap
}

// curveTouchGoldenSteps is how far each golden-section search narrows its bracket. Each step shrinks it
// by the golden ratio, so this takes a one-scan-interval bracket far below any modelling tolerance.
const curveTouchGoldenSteps = 60

// refineTouch minimises |A(ta) − B(tb)| from the scan's bracket, by golden section on ta over the
// closest approach to B — an outer 1-D search whose objective is an inner 1-D search.
//
// Nested rather than a Newton step or a walk over the pair, because the two shapes of meeting defeat
// both. A TANGENTIAL touch makes the separation vanish quadratically and the Jacobian with it, which is
// where Newton is worst. A TRANSVERSAL crossing makes a V-shaped valley, and every method that steps
// the two parameters together drifts along its wall: an alternating descent stalled 3.1 mm from the
// crossing of a circle with a chord, and shrinking a box around the best of a joint grid stalled at
// 1.2 mm — both reported "no meeting" where there is one. Each 1-D slice is unimodal over a
// one-interval bracket, so golden section on each is safe for either shape.
//
// Each parameter stays inside its OWN scan interval. Without that, a curve searched against itself
// finds the trivial minimum — |C(x) − C(y)| is zero at x = y for every curve — and every arc reads as
// self-touching. The brackets cannot meet there, because curveTouchGap keeps the stations four
// intervals apart and each bracket is one wide.
func refineTouch(a, b Curve3, ta, tb, aStep, bStep float64) (float64, float64) {
	closest := func(x float64) (float64, float64) {
		pa := a.PointAt(x)
		return goldenMin(func(y float64) float64 { return float64(pa.DistanceTo(b.PointAt(y))) }, tb-bStep, tb+bStep)
	}
	bestA, _ := goldenMin(func(x float64) float64 { _, d := closest(x); return d }, ta-aStep, ta+aStep)
	bestB, _ := closest(bestA)
	return bestA, bestB
}

// goldenRatio is the golden-section shrink factor, (√5 − 1)/2.
const goldenRatio = 0.6180339887498949

// goldenMin narrows [lo, hi] onto the minimum of a unimodal f, returning the argument and the value.
func goldenMin(f func(float64) float64, lo, hi float64) (float64, float64) {
	c, d := hi-goldenRatio*(hi-lo), lo+goldenRatio*(hi-lo)
	fc, fd := f(c), f(d)
	for range curveTouchGoldenSteps {
		if fc < fd {
			hi, d, fd = d, c, fc
			c = hi - goldenRatio*(hi-lo)
			fc = f(c)
			continue
		}
		lo, c, fc = c, d, fd
		d = lo + goldenRatio*(hi-lo)
		fd = f(d)
	}
	if fc < fd {
		return c, fc
	}
	return d, fd
}
