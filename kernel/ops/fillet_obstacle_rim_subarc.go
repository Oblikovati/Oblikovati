// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// rimNodeTrims holds the four sub-arcs the two boundary nodes split their rim segments into: for node k,
// in[k] runs rim sample nodes[k].I → node k and out[k] runs node k → rim sample nodes[k].I+1. Each is the
// segment's OWN conic trimmed at the node's rim parameter, not a re-fit and not the straight truncated
// chord the rim's three consumers used to carry there.
//
// It is computed ONCE per detection (analyticNodeDetection) and read by all three consumers — the notched
// host (hostSideSubArc), the split obstacle wall (insertNodesIntoRim) and the corner-blend patch
// (patchBoundaryLoop). That single computation is the by-value agreement the weld needs: assembleBody's
// edge catalog is FIRST-WRITER-WINS (edgeCatalog.use keeps the curve of whichever face reaches the
// segment first and silently discards the others), so two consumers agreeing "within tolerance", or
// agreeing only because of face build order, is not agreement at all — it is a latent T-junction of
// exactly the class that produced the A2/J1, canal-band, bandWrapRings and canalRailRow leaks.
type rimNodeTrims struct {
	in, out [2]geom.Curve3
}

// rimNodeTrimsOf trims each node's rim segment at that node's own rim parameter. Every entry is nil when
// the rim carries no curve, and a node's pair is nil when analyticNode could not refine it ONTO the rim
// (it kept the chord's lerped point, so there is no parameter to trim at — U4's coupled host-A node) —
// the straight chord stays the only honest answer there.
func rimNodeTrimsOf(d obstacleDetection) rimNodeTrims {
	rim := d.holeEdge.Geometry()
	n := len(d.holeSampled.pts)
	nodePts := [2]math.Point3{d.pMinus, d.pPlus}
	var t rimNodeTrims
	for k := range d.nodes {
		t.in[k], t.out[k] = nodeSubArcs(rim, n, d.nodes[k], nodePts[k])
	}
	return t
}

// nodeSubArcs trims rim segment [c.I/n, (c.I+1)/n] at the node's rim parameter c.T into two sub-arcs
// oriented FORWARD along the rim: sample→node, then node→sample. Each is built through the rim curve's
// own midpoint of its sub-span — the identical construction rimSegmentArc gives every interior segment,
// so a circular rim gets its exact trimmed arc and an elliptical rim the same per-segment fit over a
// strictly SHORTER span. The node point is passed in (rather than re-evaluated as rim.PointAt(c.T)) so
// the sub-arcs' shared endpoint is bit-identical to the vertex every consumer's loop carries there.
func nodeSubArcs(rim geom.Curve3, n int, c crossing, node math.Point3) (geom.Curve3, geom.Curve3) {
	if rim == nil || !c.onRim || n <= 0 {
		return nil, nil
	}
	t0, t1 := float64(c.I)/float64(n), float64(c.I+1)/float64(n)
	if c.T <= t0 || c.T >= t1 {
		return nil, nil // the refinement left its own bracket: do not trim on a parameter we cannot trust
	}
	return rimArcThrough(rim.PointAt(t0), rim.PointAt((t0+c.T)/2), node),
		rimArcThrough(node, rim.PointAt((c.T+t1)/2), rim.PointAt(t1))
}

// rimArcThrough is Arc3dByThreePoints with the kernel's straight-chord fallback: nil when the three points
// are too collinear to define an arc, which every filletLoop consumer already reads as "a line".
func rimArcThrough(a, mid, b math.Point3) geom.Curve3 {
	arc, err := geom.Arc3dByThreePoints(a, mid, b)
	if err != nil {
		return nil
	}
	return arc
}

// rimSubArcBetween returns the rim's OWN arc from a to b: the three-point fit through the rim curve's
// midpoint of that sub-span — the identical construction rimSegmentArc gives an interior segment and
// nodeSubArcs gives a node's half. Both endpoints are inverted onto the rim, so the span is read off the
// RIM's parameterisation and never off a per-segment fit that only approximates it.
//
//	arc := rimSubArcBetween(d.holeEdge.Geometry(), rimSample, seamStation) // one dip-rim segment
//
// It is the ONE parameterisation every dual dip-rim consumer reads — the split wall's spliced halves
// (splitRimSegmentCurve) and both panels' traced rim sides (dipRimSideCurves) — which is what makes the
// shared segment's curve a pure function of its own two endpoint VALUES rather than of which face reaches
// the edge catalog first (edgeCatalog.use is first-writer-wins and silently discards every later curve).
//
// nil when the rim carries no curve, when either endpoint does not invert uniquely onto it, or when the
// three points are too collinear to define an arc — the straight chord is the honest answer in each.
func rimSubArcBetween(rim geom.Curve3, a, b math.Point3, weld float64) geom.Curve3 {
	ta, tb, ok := rimParamsOf(rim, a, b, weld)
	if !ok {
		return nil
	}
	lo, hi := rim.Domain()
	return rimArcThrough(a, rim.PointAt(wrapRimParam(ta+foldRimStep(tb-ta, hi-lo)/2, lo, hi)), b)
}

// rimStationSplitsSpan reports whether ps lies strictly INSIDE the rim span a→b, both steps measured the
// short way round from a. It is the extent guard a trim needs on a CLOSED rim: "both endpoints on the rim"
// alone also admits the complementary arc the long way round, and a station outside its own bracket would
// produce two sub-arcs that do not tile the segment they replace.
func rimStationSplitsSpan(rim geom.Curve3, a, ps, b math.Point3, weld float64) bool {
	ta, tps, okP := rimParamsOf(rim, a, ps, weld)
	_, tb, okB := rimParamsOf(rim, a, b, weld)
	if !okP || !okB {
		return false
	}
	lo, hi := rim.Domain()
	dps, db := foldRimStep(tps-ta, hi-lo), foldRimStep(tb-ta, hi-lo)
	return db != 0 && dps/db > 0 && dps/db < 1
}

// rimParamsOf inverts both endpoints onto the rim curve, ok=false unless BOTH lie on it (onRimParam).
func rimParamsOf(rim geom.Curve3, a, b math.Point3, weld float64) (float64, float64, bool) {
	if rim == nil {
		return 0, 0, false
	}
	ta, oka := onRimParam(rim, a, weld)
	tb, okb := onRimParam(rim, b, weld)
	return ta, tb, oka && okb
}

// onRimParam returns p's parameter on the rim, ok=false unless the inversion is unique AND the rim
// reproduces p there to within the model weld. An endpoint that is not ON the rim has no rim sub-arc and
// must keep the straight chord: U4's coupled host-A node is 4.04e-03 off its own r=8 rim circle
// (x²+z² = 63.935, not 64) because analyticNode never refined it, and the split wall honestly carries a
// chord on both of its segments — so every other consumer of those two segments must carry one too.
func onRimParam(rim geom.Curve3, p math.Point3, weld float64) (float64, bool) {
	t, nature := geom.CurveParamAtPoint3(rim, p)
	if nature != geom.UniqueSolution || rim.PointAt(t).DistanceTo(p) > weld {
		return 0, false
	}
	return t, true
}

// foldRimStep folds a parameter step on a CLOSED rim of the given period to the short way round, so a
// sub-span straddling the rim's seam (parameter 0) stays its own short arc.
func foldRimStep(d, period float64) float64 {
	if period <= 0 {
		return d
	}
	switch {
	case d > period/2:
		return d - period
	case d < -period/2:
		return d + period
	}
	return d
}

// wrapRimParam folds a parameter back into the rim's own [lo,hi) domain (foldRimStep can walk one step
// past either end).
func wrapRimParam(t, lo, hi float64) float64 {
	switch {
	case t < lo:
		return t + (hi - lo)
	case t >= hi:
		return t - (hi - lo)
	}
	return t
}

// reversedArcThroughMid re-derives an arc in the OPPOSITE direction through its own recovered midpoint,
// so a consumer traversing a shared segment backwards hands the edge catalog the same trimmed conic the
// forward consumer does (reverseOpenArc's and reverseRingSeg's convention). nil in ⇒ nil out: a straight
// chord reverses to a straight chord.
func reversedArcThroughMid(c geom.Curve3, from, to math.Point3) geom.Curve3 {
	if c == nil {
		return nil
	}
	return rimArcThrough(from, c.PointAt(domainMid(c)), to)
}
