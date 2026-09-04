// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Where a curve comes back to a point it has already visited (ADR-0061 stage 2, ADR-0062).
//
// A closed section curve may TOUCH itself: a plane grazing a tilted torus's inner equator cuts a
// spiric figure-eight whose two lobes meet at one point. That meeting is an incidence like every
// other — an arrangement has to carry a vertex there, or the circuit walks straight through its own
// touch and the region it bounds comes out as one self-touching wire whose lobes wind oppositely and
// whose boundary integral cancels (measured: a lid of two 25.267 lobes integrating to 1.41e-06).
//
// It cannot be found by sampling. The touch this exists for is TANGENTIAL — the two branches meet
// with equal tangents, nothing crosses — so a sampled polyline does not self-intersect there, it only
// comes close: on the corpus's oblique figure-eight the nearest approach between two non-adjacent
// samples is 1.03e-07 against a weld grid of 1e-07, just wide enough to miss. Widening the grid would
// be an epsilon standing in for a solve. This solves it.

// curveTouchScan is how many stations the coarse pass walks. It only has to bracket each touch in one
// sampling interval, and a section curve's lobes are far apart everywhere else, so this is a
// bracketing density rather than an accuracy parameter — the refinement supplies the accuracy.
const curveTouchScan = 256

// curveTouchGap is the smallest parameter separation a pair may have and still count as a touch
// rather than as the curve's own neighbourhood. Below it the two stations are simply adjacent samples
// of one smooth arc, which are close because the curve is continuous, not because it returns.
const curveTouchGap = 4.0 / curveTouchScan

// CurveSelfTouch returns the parameters of one point the curve visits twice, within tol.
//
// The pair is ordered (a < b) and is a genuine return, not the curve's own closure: a closed curve's
// two ends meet by construction and are excluded. ok is false when the curve does not touch itself,
// which is the ordinary case.
//
// Example:
//
//	if a, b, ok := geom.CurveSelfTouch(section, res.Weld()); ok {
//		lobes := []geom.Curve3{
//			geom.TrimmedCurve3{Base: section, Lo: a, Hi: b},
//			geom.TrimmedCurve3{Base: section, Lo: b, Hi: a + span},
//		}
//	}
func CurveSelfTouch(c Curve3, tol float64) (a, b float64, ok bool) {
	lo, hi := c.Domain()
	if !(hi > lo) || tol <= 0 {
		return 0, 0, false
	}
	pts := scanCurveStations(c, lo, hi)
	ia, ib, found := nearestReturningPair(pts, CurveIsClosed(c))
	if !found {
		return 0, 0, false
	}
	span := hi - lo
	a, b = refineCurveTouch(c, lo+span*float64(ia)/curveTouchScan, lo+span*float64(ib)/curveTouchScan, span)
	if float64(c.PointAt(a).DistanceTo(c.PointAt(b))) > tol {
		return 0, 0, false
	}
	return a, b, true
}

// scanCurveStations evaluates the coarse pass's stations.
func scanCurveStations(c Curve3, lo, hi float64) []math.Point3 {
	pts := make([]math.Point3, curveTouchScan+1)
	for i := range pts {
		pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/curveTouchScan)
	}
	return pts
}

// nearestReturningPair is the closest pair of stations that are far enough apart IN PARAMETER to be a
// return rather than neighbours, measured on the closed curve where the parameter itself wraps.
func nearestReturningPair(pts []math.Point3, closed bool) (int, int, bool) {
	best, bi, bj := stdmath.Inf(1), -1, -1
	n := len(pts) - 1
	for i := 0; i <= n; i++ {
		for j := i + 1; j <= n; j++ {
			if !stationsReturn(i, j, n, closed) {
				continue
			}
			if d := float64(pts[i].DistanceTo(pts[j])); d < best {
				best, bi, bj = d, i, j
			}
		}
	}
	return bi, bj, bi >= 0
}

// stationsReturn reports whether two stations are separated enough in parameter that meeting is a
// return rather than continuity. On a CLOSED curve the separation is the shorter way round, which is
// what excludes the closure itself — the first and last stations are the same point.
func stationsReturn(i, j, n int, closed bool) bool {
	gap := float64(j-i) / float64(n)
	if closed && 1-gap < gap {
		gap = 1 - gap
	}
	return gap >= curveTouchGap
}

// curveTouchRefineSteps is the number of halvings the refinement takes. Each step shrinks the bracket
// by half in both parameters, so this converges the touch far below any modelling tolerance from the
// one-interval bracket the scan hands over.
const curveTouchRefineSteps = 60

// refineCurveTouch minimises |C(a) − C(b)| from the scan's bracket by shrinking a box around the pair.
//
// A coordinate walk rather than a Newton step, because at a TANGENTIAL touch the distance vanishes
// quadratically and the Jacobian with it — the very case this exists for is the one Newton is worst
// at. Halving a bracket does not care.
//
// Each parameter stays inside its OWN scan interval. Without that the walk finds the trivial minimum:
// |C(a) − C(b)| is zero at a = b for every curve, and on a smooth arc the distance falls all the way
// down to it, so an unclamped descent reports every arc as touching itself. The brackets cannot meet,
// because curveTouchGap keeps the stations four intervals apart and each bracket is one wide.
func refineCurveTouch(c Curve3, a, b, span float64) (float64, float64) {
	width := span / curveTouchScan
	aLo, aHi, bLo, bHi := a-width, a+width, b-width, b+width
	step := width
	for range curveTouchRefineSteps {
		a = clampParam(descendTouchParam(c, a, b, step, true), aLo, aHi)
		b = clampParam(descendTouchParam(c, a, b, step, false), bLo, bHi)
		step /= 2
	}
	return a, b
}

// clampParam holds a refined parameter inside its own bracket.
func clampParam(x, lo, hi float64) float64 {
	return stdmath.Max(lo, stdmath.Min(hi, x))
}

// descendTouchParam moves one of the two parameters to whichever of {−step, 0, +step} brings the pair
// closest together.
func descendTouchParam(c Curve3, a, b, step float64, moveA bool) float64 {
	best, at := touchDistance(c, a, b), a
	if !moveA {
		at = b
	}
	for _, d := range [2]float64{-step, step} {
		x, y := a+d, b
		if !moveA {
			x, y = a, b+d
		}
		if dist := touchDistance(c, x, y); dist < best {
			best, at = dist, x
			if !moveA {
				at = y
			}
		}
	}
	return at
}

// touchDistance is the separation of the curve at two parameters.
func touchDistance(c Curve3, a, b float64) float64 {
	return float64(c.PointAt(a).DistanceTo(c.PointAt(b)))
}
