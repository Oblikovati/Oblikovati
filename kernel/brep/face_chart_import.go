// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Deriving the chart of a face NOBODY's arrangement built (Oblikovati/Oblikovati#3550).
//
// ADR-0063 put a face's parametric trim on the face, and `kernel/brep` records one on every face its
// (u, v) arrangement winds. Nothing else does. A body that arrives through `kernel/exchange`, or one a
// feature rebuild re-winds, therefore reaches the tessellator with chart = nil on every face — measured
// on occtparity simple/J3 and bfuseblend/A4, all four and all nine faces — and the general chart-driven
// mesher declines such a face outright. That is why the bespoke arms behind the curved-trim
// classification are still alive: they exist to serve exactly the faces the general path cannot see.
//
// This is the producer side of that gap. It derives a face's chart from the face's own loops, and the
// derivation is legitimate for the reason ADR-0063 gives: a ring that CLOSES in the covering space
// already bounds one region there. What was missing is the LIFT.
//
// THE LIFT, and why the obvious one is wrong. A loop is sampled edge by edge and each sample is carried
// onto the branch of the one before it (unwrapLoopRing does exactly this). That is right INSIDE an
// edge and wrong ACROSS one, because a face on a periodic surface bridges its two rims with an
// ARTIFICIAL SEAM that the loop walks TWICE — up one side of the slit and back down the other. The two
// traversals are the same 3-D curve, so a point-to-point unwrap snaps the second onto the first's
// branch, the keyhole collapses, and the ring comes back with a net turn of two periods instead of
// closing. Measured on J3's host torus: a contour spanning v ∈ [−2π, +2π] where the region is one
// period tall, which the boundary-side classification then refuses as naming both sides as material
// (127 segments left, 126 right).
//
// So each EDGE USE is lifted on its own — continuous within itself and nowhere else — and the uses are
// then placed by whole-period junction shifts. No rule about seams is needed: the second traversal of a
// seam arrives with its own branch, the junction shift that meets the running chain is the one a whole
// period along, and the keyhole falls out. What the derivation asks for at the end is the thing that
// makes it not a guess — that the ring CLOSE. It declines when it does not.

// ChartOfFace derives the closed (u, v) contours of a face on a periodic surface from its own loops,
// for a producer that has no arrangement to record one from (an importer, a rebuild). ok=false when the
// face's loops do not determine a region — an aperiodic surface needs no chart, and a ring that does
// not close is not one.
//
// Example:
//
//	if chart, ok := brep.ChartOfFace(f); ok { f.SetChart(chart) }
func ChartOfFace(f *topo.Face) ([][]math.Point2, bool) {
	s := f.Geometry()
	uPer, vPer := surfacePeriodic(s)
	if !uPer && !vPer {
		return nil, false // ToUVLoops charts an aperiodic trim; there is no branch to choose
	}
	rings := make([][]math.Point2, 0, len(f.Loops()))
	for _, l := range f.Loops() {
		ring, ok := liftLoopByUse(s, l, uPer, vPer)
		if !ok {
			return nil, false
		}
		rings = append(rings, ring)
	}
	if len(rings) == 0 {
		return nil, false // boundaryless: the face is its whole surface and needs no chart
	}
	return closeRingsIntoChart(s, rings, isOuterlessFace(f), uPer, vPer)
}

// liftLoopByUse lifts one loop into the covering space, each edge use continuous within itself and
// placed against the previous use by a whole-period shift. ok=false when the result does not close
// even after re-sensing a rim (below).
func liftLoopByUse(s geom.Surface, l *topo.Loop, uPer, vPer bool) ([]math.Point2, bool) {
	var ring []math.Point2
	for _, u := range l.EdgeUses() {
		trace := useTrace(s, u, uPer, vPer)
		if len(trace) < 2 {
			return nil, false
		}
		ring = appendPlacedTrace(ring, trace, uPer, vPer)
	}
	if !ringTravelsAWholePeriodAtMost(ring) {
		return nil, false
	}
	return ring, true
}

// ringTravelsAWholePeriodAtMost is the lift's post-condition, and the thing it REFUSES is worth naming
// because it is a real producer defect rather than a shape this derivation is too weak for.
//
// A lifted loop may end where it started (a circuit) or one whole period along (a band RIM, which
// closeRingsIntoChart pairs with its partner across the seam). It may not end TWO periods along. That
// happens when the loop walks its two tube-wrapping rims the SAME way, which is not a consistently
// wound boundary: measured on occtparity simple/J3's host torus, −4π in v, because the fillet's rim
// rebuild takes the replacement rim's use flag from the blend's CONVEXITY (fillet_rim_build.go's
// `Reversed: !g.concave`, chosen to mirror the band face under Validate's 2-incidence rule) rather
// than from the rim it replaces. simple/J3's IMPORTED torus face closes and charts; the rebuilt one
// does not. bfuseblend/A4, the concave case, comes out consistent by the same rule and charts.
//
// Re-sensing one rim here would close the ring and would even give the right REGION — the seam the loop
// carries pins which of the two bands the face is, and a chart is orientation-free (chartContains
// counts even-odd; ADR-0063 reads handedness against the region, not from the contour's area). It was
// built and measured, and it buys nothing: the tessellator's boundary-side classification then refuses
// the same face for the same underlying reason, "its boundary chain 0 names both sides as material
// (127 segments say left, 126 say right)" — the two rims again. The invariant that is broken is the
// loop's winding, and the ground rules put the fix there, not in a branch here.
func ringTravelsAWholePeriodAtMost(ring []math.Point2) bool {
	if len(ring) < 3 {
		return false
	}
	a, b := ring[0], ring[len(ring)-1]
	tol := ringClosureTol * ringExtent(ring)
	return withinOnePeriod(float64(b.X-a.X), tol) && withinOnePeriod(float64(b.Y-a.Y), tol)
}

// withinOnePeriod reports whether a closure defect is zero or one whole period, to tolerance.
func withinOnePeriod(d, tol float64) bool {
	return stdmath.Abs(d) <= tol || stdmath.Abs(stdmath.Abs(d)-twoPi) <= tol
}

// appendPlacedTrace places a use's trace so its first sample meets the chain's last, and appends it
// without repeating that shared sample. The FIRST use is placed as it stands.
func appendPlacedTrace(ring, trace []math.Point2, uPer, vPer bool) []math.Point2 {
	if len(ring) == 0 {
		return append(ring, trace...)
	}
	du, dv := junctionShift(ring[len(ring)-1], trace[0], uPer, vPer)
	for _, p := range trace[1:] {
		ring = append(ring, math.P2(float64(p.X)+du, float64(p.Y)+dv))
	}
	return ring
}

// junctionShift is the whole-period offset carrying a trace's first sample onto the chain's last. The
// two are the SAME 3-D point (a shared loop vertex), so their parameters differ by whole periods only,
// and rounding the difference is exact rather than a nearest-match heuristic.
func junctionShift(end, start math.Point2, uPer, vPer bool) (du, dv float64) {
	if uPer {
		du = wholePeriods(float64(end.X) - float64(start.X))
	}
	if vPer {
		dv = wholePeriods(float64(end.Y) - float64(start.Y))
	}
	return du, dv
}

// wholePeriods rounds a parameter difference to the nearest whole number of periods.
func wholePeriods(d float64) float64 { return twoPi * stdmath.Round(d/twoPi) }

// ringClosureTol is how far, as a fraction of the ring's own (u, v) extent, the lift may land from its
// start and still count as closed. The lift's only inexactness is the surface inversion at the shared
// vertex, which is a parameter read of a point both edges pass through exactly; a thousandth of the
// ring's extent is orders above that and still far below a whole period, which is the thing it must
// not admit.
const ringClosureTol = 1e-3 // tol:parametric (relative to the ring's own extent)

// ringExtent is the larger side of a ring's (u, v) bounding box, the scale its closure is judged at.
func ringExtent(ring []math.Point2) float64 {
	u0, u1, v0, v1 := polyBoundsOf(ring)
	return stdmath.Max(stdmath.Max(u1-u0, v1-v0), twoPi)
}

// polyBoundsOf is one ring's (u, v) bounding box.
func polyBoundsOf(ring []math.Point2) (u0, u1, v0, v1 float64) {
	u0, v0 = stdmath.Inf(1), stdmath.Inf(1)
	u1, v1 = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, p := range ring {
		u0, u1 = stdmath.Min(u0, float64(p.X)), stdmath.Max(u1, float64(p.X))
		v0, v1 = stdmath.Min(v0, float64(p.Y)), stdmath.Max(v1, float64(p.Y))
	}
	return u0, u1, v0, v1
}

// useTrace samples one edge use in loop order and makes its own parameters continuous — and ONLY its
// own. Continuity stops at the edge because that is where a seam's two traversals have to be allowed
// to differ (see the file doc).
func useTrace(s geom.Surface, u *topo.EdgeUse, uPer, vPer bool) []math.Point2 {
	e := orientedLoopEdge(u)
	out := make([]math.Point2, 0, trimUVSamples+1)
	for k := 0; k <= trimUVSamples; k++ {
		t := e.t0 + (e.t1-e.t0)*float64(k)/trimUVSamples
		cu, cv := s.ParamAt(e.curve.PointAt(t))
		if n := len(out); n > 0 {
			cu, cv = continueUV(out[:n], cu, cv, uPer, vPer)
		}
		out = append(out, math.P2(cu, cv))
	}
	return out
}
