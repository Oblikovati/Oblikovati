// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Retrimming a curved host at a CORNER, where several blended arms land on one face (split out of
// fillet_curved_retrim.go for #2220).
//
// The single-arm case above needs one contact rail; a corner needs one per arm plus the connectors
// between them, and the arms have to agree on where the host boundary runs. The chart machinery
// here exists because a ruling's landing point is only well posed in the host's own parameter
// space.

// retrimCornerHost assembles a two-arm host loop: rail A (outer→tHost), rail B (tHost→outer), then
// the surviving far path (outerB→outerA) that avoids the bitten trihedral vertex. cornerFarPath is
// farPathSegs verbatim unless a reflex connector bridges an off-loop interior station (Splice 1, D9's cap).
func retrimCornerHost(host *topo.Face, segs []endSeg, v math.Point3, arms []cornerHostArm, tHost math.Point3, w cornerWeld, res opstol.Resolution, tol float64, conns []cornerConnector) ([]endSeg, bool) {
	railA, outerA, okA := armContactRail(host, arms[0], tHost, v, segs, w, res, tol)
	railB, outerB, okB := armContactRail(host, arms[1], tHost, v, segs, w, res, tol)
	if !okA || !okB {
		return nil, false
	}
	far, ok := cornerFarPath(segs, outerB, outerA, v, tol, conns)
	if !ok {
		return nil, false
	}
	out := append([]endSeg{railA}, reverseEndSegs([]endSeg{railB})...) // outerA→tHost→outerB
	return append(out, far...), true                                   // …→outerA (closed)
}

// armContactRail builds one arm's contact rail on a corner host as an endSeg oriented outer→tHost, plus
// the outer landing point. When the arm carries its weld bundle (ha.hasRail, the production path) it
// consumes that rail VERBATIM (bundleContactRail) — the single-source-of-truth weld with the arm face,
// oblique-aware by construction. Without a bundle (the unit fixtures) it rebuilds: torus arms carve a
// circular arc (curvedHostArc); cylinder arms carve a straight ruling from the far runout to tHost.
func armContactRail(host *topo.Face, ha cornerHostArm, tHost, v math.Point3, segs []endSeg, w cornerWeld, res opstol.Resolution, tol float64) (endSeg, math.Point3, bool) {
	if ha.hasRail {
		return bundleContactRail(ha.rail, tHost, tol)
	}
	switch s := ha.set.arm.(type) {
	case geom.Torus:
		arc, ok := curvedHostArc(host.Geometry(), s, w, res)
		if !ok || float64(arc.PointAt(1).DistanceTo(tHost)) > tol {
			return endSeg{}, math.Point3{}, false // no torus rail here, or it misses the tangent point
		}
		outer := arc.PointAt(0)
		return endSeg{from: outer, to: tHost, curve: arc, mid: arc.PointAt(0.5), arc: true}, outer, true
	case geom.Cylinder:
		outer, ok := armRulingEnd(host, s, ha.set, tHost, v, segs, tol)
		if !ok {
			return endSeg{}, math.Point3{}, false
		}
		return endSeg{from: outer, to: tHost}, outer, true
	}
	return endSeg{}, math.Point3{}, false
}

// bundleContactRail consumes an arm's weld-bundle host rail verbatim as the retrim's corner rail (already
// oriented outer→tHost), so the retrimmed host and the arm face share ONE curve object — watertight by
// construction: the oblique regime's outer end is the foot ON the loop, the perpendicular regime's is the
// untouched full-arc P0. Declines when the bundle rail's inner end misses the shared tangent point beyond
// tol (a mis-paired bundle — do-no-harm rather than weld a crack).
func bundleContactRail(rail endSeg, tHost math.Point3, tol float64) (endSeg, math.Point3, bool) {
	if float64(rail.to.DistanceTo(tHost)) > tol {
		return endSeg{}, math.Point3{}, false
	}
	return rail, rail.from, true
}

// sinFloor is the dimensionless (angular, scale-free) floor for a DEGENERATE ruling: a chart direction
// whose magnitude falls below it means the arm axis is (near-)perpendicular to a planar host — the
// ruling projects to a point and casting it would divide by ~0. Like retrimAxisParallelTol, an angle
// carries no length, so ADR-0042's model-relative rule does not apply; 1e-6 sits far inside the exact
// geometry yet rejects any real misalignment. Never triggers on the corpus (every arm axis lies in / is
// parallel to its host), so it is a defensive decline, not a hot path.
const sinFloor = 1e-6

// armRulingEnd is the far end of a cylinder arm's straight ruling on a host: the FIRST forward crossing
// of the ruling with the (bitten) loop, computed in the host's intrinsic chart (θ,z on a wall; u,v on a
// plane), then cross-checked against the filleted edge's far vertex (the runout authority, R.1a).
// Replaces axialExtremeEnd, which slid to the loop's GLOBAL axial extreme and overshot any intermediate
// trim edge (N7 s_4: global rim z=130, true runout z=80). The accepted crossing (the common convex case)
// is a forward loop crossing whose runout MATCHES the far vertex. When the loop-crossing path produces
// NO accepted outer end — none found, OR the one found has the WRONG runout — it falls back to the
// reflex-corner interior station (rulingStationOuter, D9-T1): a >180° wedge leaves the true outer end
// interior to the host sector, on no loop edge, so the ruling either misses the loop (backward-shadowed)
// or crosses its FAR side at a longer runout than the far vertex (D9's cap rim at t=+145.9 vs 71.58).
// It still honest-declines (false) when the chart ruling is degenerate (rulingChartRay) or the station
// itself is off-face / on-loop (rulingStationOuter), so nothing is fabricated where the ruling genuinely
// runs off the host.
func armRulingEnd(host *topo.Face, cylArm geom.Cylinder, arm armSetback, tHost, v math.Point3, segs []endSeg, tol float64) (math.Point3, bool) {
	ch, o2, d2, ok := rulingChartRay(host, cylArm, tHost, v)
	if !ok {
		return math.Point3{}, false // no host chart, or the ruling projects to a point (grazing) — decline
	}
	if end, ok := chartRulingExit(ch, segs, o2, d2, tol); ok &&
		(!arm.runoutKnown || runoutAgrees(ch, o2, d2, end, arm.farVertex, tol)) {
		return end, true // forward loop crossing whose runout matches the far vertex — the verbatim green path
	}
	return rulingStationOuter(ch, cylArm, arm, tHost, v, segs, o2, d2, tol) // reflex corner: interior far-vertex station
}

// rulingStationOuter is armRulingEnd's reflex-corner (>180° wedge) fallback: when the ruling meets no
// forward loop edge, the arm's true outer end is the far-vertex STATION projected onto the ruling —
// outer = tHost + d̂·((farVertex−tHost)·d̂), the PRE-N2 closed-form terminator. It is correct here because
// a reflex wedge can leave that station INTERIOR to the host sector, on no loop edge (D9's 270° cap:
// station (−10,0,129.9038) at azimuth 180°, radius 10 ≪ 72.27 — forensic §1). Accept it only when the
// runout is stamped and its chart point lies strictly inside the host loop AND the ruling reaches the
// station WITHOUT first leaving the loop (stationRunReached — the D9-T2 overshoot guard: an endpoint-only
// interiority test does not certify the whole ruling span, so a re-entrant loop whose ruling exits before
// the station is rejected); off-face, unknown runout, or an undershoot → decline, so no outer end is
// fabricated where the ruling genuinely runs off the face.
func rulingStationOuter(ch hostChart, cylArm geom.Cylinder, arm armSetback, tHost, v math.Point3, segs []endSeg, o2 math.Point2, d2 math.Vector2, tol float64) (math.Point3, bool) {
	if !arm.runoutKnown {
		return math.Point3{}, false // no far vertex stamped (bare-face unit corner) — nothing to project onto
	}
	d := awayFrom(cylArm.AxisDir.AsVector(), tHost, v) // unit ruling direction, away from the bitten vertex
	outer := tHost.TranslateBy(d.Scale(tHost.VectorTo(arm.farVertex).Dot(d)))
	if !chartPointInLoop(ch, ch.to2(outer), segs) {
		return math.Point3{}, false // station runs off the host face — decline (do not fabricate an end)
	}
	if !stationRunReached(ch, segs, o2, d2, arm.farVertex, tol) {
		return math.Point3{}, false // the ruling exits the loop before the station (undershoot) — off-face span
	}
	return outer, true
}

// chartPointInLoop reports whether chart point q2 lies inside host loop segs, developed into the same
// chart. It samples each seg (arcs included, via segPolyline) so a curved loop edge — D9's 270° cap rim
// — contributes its true shape, not a chord, then ray-casts (pointInLoop2D). Both helpers are reused
// verbatim (segPolyline from the far-runout file, pointInLoop2D from union_holes) — no duplication.
func chartPointInLoop(ch hostChart, q2 math.Point2, segs []endSeg) bool {
	ring3 := segPolyline(segs)
	loop2 := make([]math.Point2, len(ring3))
	for i, p := range ring3 {
		loop2[i] = ch.to2(p)
	}
	return probe.PointInLoop2D(q2, loop2)
}

// rulingChartRay develops the host and returns the arm ruling as a chart ray: origin ch.to2(tHost),
// direction the arm axis oriented away from the bitten vertex, mapped into the chart (θ fixed → vertical
// on a wall; projected axis on a plane). ok=false when the host has no chart or the ruling is degenerate
// (the axis projects to a ~0 chart direction — sinFloor guards the divide chartRulingExit would do).
func rulingChartRay(host *topo.Face, cylArm geom.Cylinder, tHost, v math.Point3) (hostChart, math.Point2, math.Vector2, bool) {
	ch, ok := hostChartFor(host.Geometry())
	if !ok {
		return nil, math.Point2{}, math.Vector2{}, false // host is neither plane nor cylinder
	}
	dir := awayFrom(cylArm.AxisDir.AsVector(), tHost, v)
	o2 := ch.to2(tHost)
	d2 := o2.VectorTo(ch.to2(tHost.TranslateBy(dir)))
	if float64(d2.Length()) < sinFloor { // arm axis ⊥ a planar host: ruling has no chart extent
		return nil, math.Point2{}, math.Vector2{}, false
	}
	return ch, o2, d2, true
}

// runoutAgrees reports whether the crossing end and the far vertex share the same RUNOUT (their extent
// along the ruling direction d2), within tol. It compares only the along-ruling coordinate — NOT the
// full 3D distance — because a fillet's contact ruling is laterally offset from its sharp edge by ~r, so
// the ruling end and the edge far vertex share the runout coordinate (both reach the same rim) but sit
// at different lateral positions; a 3D distance would false-reject by that offset.
func runoutAgrees(ch hostChart, o2 math.Point2, d2 math.Vector2, end, farVertex math.Point3, tol float64) bool {
	unit := d2.Scale(math.Scalar(1 / float64(d2.Length())))
	rEnd := float64(o2.VectorTo(ch.to2(end)).Dot(unit))
	rFar := float64(o2.VectorTo(ch.to2(farVertex)).Dot(unit))
	return stdmath.Abs(rEnd-rFar) <= tol
}

// awayFrom returns axis oriented to point away from the bitten vertex v (the side the ruling runs to).
func awayFrom(axis math.Vector3, tHost, v math.Point3) math.Vector3 {
	if float64(v.VectorTo(tHost).Dot(axis)) >= 0 {
		return axis
	}
	return axis.Scale(-1)
}
