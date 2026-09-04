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
	pa, pb := scanCurveStations(a, aLo, aHi), scanCurveStations(b, bLo, bHi)
	var out [][2]float64
	for _, cell := range touchCandidateCells(pa, pb, same, CurveIsClosed(a)) {
		ta, tb := refineTouch(a, b,
			aLo+(aHi-aLo)*float64(cell[0])/curveTouchScan, bLo+(bHi-bLo)*float64(cell[1])/curveTouchScan,
			(aHi-aLo)/curveTouchScan, (bHi-bLo)/curveTouchScan)
		if float64(a.PointAt(ta).DistanceTo(b.PointAt(tb))) <= tol {
			out = append(out, [2]float64{ta, tb})
		}
	}
	return out
}

// scanCurveStations evaluates the coarse pass's stations.
func scanCurveStations(c Curve3, lo, hi float64) []math.Point3 {
	pts := make([]math.Point3, curveTouchScan+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/curveTouchScan)
	}
	return pts
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

// curveTouchRefineSteps is the number of halvings the refinement takes, which converges the touch far
// below any modelling tolerance from the one-interval bracket the scan hands over.
const curveTouchRefineSteps = 60

// refineTouch minimises |A(ta) − B(tb)| from the scan's bracket by shrinking a box around the pair.
//
// A coordinate walk rather than a Newton step, because at a TANGENTIAL touch the separation vanishes
// quadratically and the Jacobian with it — the very case this exists for is the one Newton is worst at.
// Halving a bracket does not care.
//
// Each parameter stays inside its OWN scan interval. Without that, a walk on one curve against itself
// finds the trivial minimum — |C(x) − C(y)| is zero at x = y for every curve — and reports every arc as
// touching itself. The brackets cannot meet there, because curveTouchGap keeps the stations four
// intervals apart and each bracket is one wide.
func refineTouch(a, b Curve3, ta, tb, aStep, bStep float64) (float64, float64) {
	aLo, aHi, bLo, bHi := ta-aStep, ta+aStep, tb-bStep, tb+bStep
	for range curveTouchRefineSteps {
		ta = clampParam(descendTouchParam(a, b, ta, tb, aStep, true), aLo, aHi)
		tb = clampParam(descendTouchParam(a, b, ta, tb, bStep, false), bLo, bHi)
		aStep, bStep = aStep/2, bStep/2
	}
	return ta, tb
}

// clampParam holds a refined parameter inside its own bracket.
func clampParam(x, lo, hi float64) float64 {
	return stdmath.Max(lo, stdmath.Min(hi, x))
}

// descendTouchParam moves one of the two parameters to whichever of {−step, 0, +step} brings the pair
// closest together.
func descendTouchParam(a, b Curve3, ta, tb, step float64, moveA bool) float64 {
	best, at := touchSeparation(a, b, ta, tb), ta
	if !moveA {
		at = tb
	}
	for _, d := range [2]float64{-step, step} {
		x, y := ta+d, tb
		if !moveA {
			x, y = ta, tb+d
		}
		if s := touchSeparation(a, b, x, y); s < best {
			best, at = s, x
			if !moveA {
				at = y
			}
		}
	}
	return at
}

// touchSeparation is the distance between the two curves at the given parameters.
func touchSeparation(a, b Curve3, ta, tb float64) float64 {
	return float64(a.PointAt(ta).DistanceTo(b.PointAt(tb)))
}
