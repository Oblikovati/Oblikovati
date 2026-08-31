// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The exact boundary ring a face-loop VALIDITY query is taken on (M48/C3,
// Oblikovati/Oblikovati#3476, #3475).
//
// The loop detectors used to build their ring with loopBoundary, the tessellator's chord polyline:
// every edge sampled to the caller's Quality. That made a topological verdict — is this boundary a
// simple polygon — a function of facet density, so the same body could be reported malformed at
// property quality and clean at display quality. The ring here is derived from TOPOLOGY alone: one
// point per edge use, the vertex that use starts at, in traversal order. It is the same ring at every
// quality, it reads no tessellation, and its vertices are the exact B-rep points rather than samples
// of a curve.
//
// THE ONE REFINEMENT, AND WHY IT IS NOT TESSELLATION. On a PERIODIC surface the chart is only defined
// up to whole periods, and the unwrap that develops a loop assumes each step is the SHORT way round
// (wrapPi). One point per edge breaks that assumption the moment an edge spans more than half a period
// — a 270° rim arc unwraps as −90° and the developed polygon is nonsense. So a step is split until it
// AGREES WITH ITS OWN HALVES (stepAgreesWithHalves), the standard "does refining change the answer"
// test. The split count is driven by the surface's periodicity, never by a chord tolerance: a plane
// never splits, and neither does a wide edge that already unwraps correctly.
//
// ★ WHAT THE CORNER RING CANNOT SEE, NAMED. One segment per edge develops a curved edge as the CHORD
// between its ends, so two edges that bow INTO each other strictly between their endpoints — while
// their corner chords stay clear — are not reported. Seeing those needs exact curve-curve intersection
// in the surface's chart, which geom does not yet offer (it has no curve-curve intersector; the
// closest primitive, brep.EntityDistance, answers distance, not the crossing parameters an Area
// measurement needs).
//
// A chord also errs the OTHER way — it can cut across ground the curve keeps clear of, and invent a
// crossing. Measured on the OCCT blend-parity corpus that is not hypothetical: six planar faces
// reported one. So the ring carries, per point, the EDGE its outgoing segment lies on, and a chart
// crossing is corroborated against those two edges' own curves before it is reported
// (crossingIsOnTheCurves). What is left is one-sided: every report is a real crossing of the exact
// boundary, and only the bow-into-each-other case is missed.

// chartStepsCap bounds the periodicity refinement (a COUNT, not a tolerance): an edge still leaping a
// half period after 32 pieces winds the axis several times over, and unwrap declines such a loop
// anyway. chartStepAgreement is the residual below which a step and the sum of its halves are the same
// angle — many decades above double-precision noise on a parameter and many below a half period.
const (
	chartStepsCap      = 32
	chartStepAgreement = 1e-6 // tol:angular — radians of disagreement between a step and its halves
)

// cornerRing is one face loop's exact boundary ring: its points, for each point the EDGE whose own
// curve the segment LEAVING that point lies on, and that segment's own MID-parameter point on the edge.
// The owner and the mid are what let a chart verdict be corroborated against the geometry the chart
// came from: developed, the mid must sit on the straight chart segment its two ends span, or the chart
// is not rendering that piece of boundary at all.
type cornerRing struct {
	pts    []math.Point3
	mids   []math.Point3 // the edge-curve point halfway along each segment, aligned with pts
	owners []*topo.Edge
}

// faceCornerRings returns f's boundary rings — outer first, then the holes — as edge corners, the
// same order [developedFaceLoops] develops them in, so a developed loop can be checked against its
// own 3D points index for index.
func faceCornerRings(f *topo.Face) []cornerRing {
	outer := faceOuterCornerRing(f)
	if len(outer.pts) == 0 {
		return nil
	}
	return append([]cornerRing{outer}, faceHoleCornerRings(f)...)
}

// faceOuterCornerRing returns the corner ring of a face's outer loop.
func faceOuterCornerRing(f *topo.Face) cornerRing {
	for _, l := range f.Loops() {
		if l.IsOuter() {
			return loopCornerRing(l, f.Geometry())
		}
	}
	return cornerRing{}
}

// faceHoleCornerRings returns the corner ring of each inner (hole) loop.
func faceHoleCornerRings(f *topo.Face) []cornerRing {
	var holes []cornerRing
	for _, l := range f.Loops() {
		if !l.IsOuter() {
			holes = append(holes, loopCornerRing(l, f.Geometry()))
		}
	}
	return holes
}

// loopCornerRing returns one point per edge use — the vertex that use starts at — in traversal order,
// refined where s's periodicity demands it, each point tagged with the edge it belongs to. The closing
// point is not repeated, matching what loopBoundary returned for a closed ring.
func loopCornerRing(l *topo.Loop, s geom.Surface) cornerRing {
	var out cornerRing
	for _, u := range l.EdgeUses() {
		pts, mids := edgeChartPoints(u, s)
		out.pts, out.mids = append(out.pts, pts...), append(out.mids, mids...)
		for range pts {
			out.owners = append(out.owners, u.Edge())
		}
	}
	return out
}

// edgeChartPoints is the use's own ring points in traversal order — the vertex it enters at, then any
// interior points the periodicity refinement asks for — together with each step's mid-parameter point
// on the edge curve. It stops BEFORE the vertex it leaves at, which is the next use's first point.
func edgeChartPoints(u *topo.EdgeUse, s geom.Surface) (pts, mids []math.Point3) {
	n := chartStepsFor(u, s)
	pts, mids = make([]math.Point3, n), make([]math.Point3, n)
	pts[0] = edgeUseStartPoint(u) // the exact vertex, not the curve's end evaluation
	for k := 1; k < n; k++ {
		pts[k] = edgeUsePointAt(u, float64(k)/float64(n))
	}
	for k := range n {
		mids[k] = edgeUsePointAt(u, (float64(k)+0.5)/float64(n))
	}
	return pts, mids
}

// edgeUseStartPoint is the vertex a use enters its edge at: the edge's start vertex when the use runs
// forward, its end vertex when reversed.
func edgeUseStartPoint(u *topo.EdgeUse) math.Point3 {
	if u.Reversed() {
		return u.Edge().EndVertex().Point()
	}
	return u.Edge().StartVertex().Point()
}

// edgeUsePointAt evaluates the use's edge curve at the traversal fraction tau ∈ [0,1], where 0 is the
// vertex the use enters at — so the ring is built in the loop's own direction.
func edgeUsePointAt(u *topo.EdgeUse, tau float64) math.Point3 {
	c := u.Edge().Geometry()
	lo, hi := c.Domain()
	if u.Reversed() {
		tau = 1 - tau
	}
	return c.PointAt(lo + tau*(hi-lo))
}

// chartStepsFor is how many pieces a use's edge must be split into so every ring step unwraps the short
// way in s's periodic parameters. A non-periodic surface, an unbounded edge curve, and any edge whose
// single step already agrees with its halves all take one piece — so a planar boundary is exactly its
// corners.
func chartStepsFor(u *topo.EdgeUse, s geom.Surface) int {
	lo, hi := u.Edge().Geometry().Domain()
	if s == nil || stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) || !surfaceIsPeriodic(s) {
		return 1
	}
	n := 1
	for n < chartStepsCap && !chartStepsAgree(u, s, n) {
		n *= 2
	}
	return n
}

// surfaceIsPeriodic reports whether s wraps in either parameter direction, the only case in which a
// ring step can be unwrapped the wrong way.
func surfaceIsPeriodic(s geom.Surface) bool {
	return isPeriodic(s.UDomain()) || isPeriodic(s.VDomain())
}

// chartStepsAgree reports whether splitting the edge into n pieces already renders it faithfully: every
// piece must agree with its own two halves.
func chartStepsAgree(u *topo.EdgeUse, s geom.Surface, n int) bool {
	for k := range n {
		a, b := float64(k)/float64(n), float64(k+1)/float64(n)
		if !stepAgreesWithHalves(s, edgeUsePointAt(u, a), edgeUsePointAt(u, (a+b)/2), edgeUsePointAt(u, b)) {
			return false
		}
	}
	return true
}

// stepAgreesWithHalves reports whether the unwrapped step pa→pb equals the sum of pa→pm and pm→pb in
// every periodic direction. It fails exactly when the whole step leaps more than half a period, where
// wrapPi takes it the wrong way round while the halves take it the right way.
func stepAgreesWithHalves(s geom.Surface, pa, pm, pb math.Point3) bool {
	ua, va := periodicParams(s, pa)
	um, vm := periodicParams(s, pm)
	ub, vb := periodicParams(s, pb)
	return halvesAgree(ua, um, ub) && halvesAgree(va, vm, vb)
}

// periodicParams returns p's surface parameters in the PERIODIC directions only; a non-periodic
// direction reports 0, which agrees trivially and so never forces a split.
func periodicParams(s geom.Surface, p math.Point3) (u, v float64) {
	u, v = s.ParamAt(p)
	if !isPeriodic(s.UDomain()) {
		u = 0
	}
	if !isPeriodic(s.VDomain()) {
		v = 0
	}
	return u, v
}

// halvesAgree compares the unwrapped whole step with the sum of its two halves.
func halvesAgree(a, m, b float64) bool {
	return stdmath.Abs(wrapPi(b-a)-(wrapPi(m-a)+wrapPi(b-m))) <= chartStepAgreement
}
