// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Curved-fillet weld — the ARM case (M48 #2222 split of fillet_curved_weld.go). One trimmed analytic
// arm face (torus/cylinder) and the four boundary rails that close it: the two host contact rails, the
// arm↔sphere setback great-arc, and the far cross-section runout trim. The host rails are built by the
// SAME constructors the corner-host retrim uses (curvedHostArc / armRulingEnd via cylinderRulingOuterOnHost),
// so the arm face and its neighbour host land on byte-identical curves and weld without a crack.

// armRails is one arm's four boundary edges as a closed loop (host rail on ef.a, setback great-arc,
// host rail on ef.b reversed, far runout trim), plus that far trim alone (outer0→outer1) — the rail
// shared with the far-runout host, kept separate so both faces reuse the identical curve — and the
// runout object (regime + capping identity) the far-runout host bite router reads (FR3).
//
// hostA/hostB are the SAME (possibly oblique-re-terminated) host contact rails, un-reversed and oriented
// outer→tHost — hostA on ef.a, hostB on ef.b. They are kept as their own fields (not re-extracted from
// segs) so the CORNER-HOST retrim (fillet_curved_retrim.go, FR4) consumes the identical curve OBJECT the
// arm face carries, welding the two sides by construction: for a perpendicular runout hostA/hostB are the
// untouched full curvedHostArc arcs (existing greens byte-identical), for an oblique runout their outer
// ends are the closed-form feet ON the host loop (D5/E4). Re-extracting from segs would double-reverse
// segs[2] and re-derive its arc — a congruent but NOT bit-identical curve, breaking the weld identity.
type armRails struct {
	segs   []endSeg
	far    endSeg
	runout armRunout
	hostA  endSeg // ef.a host contact rail, oriented outer→tHost (== segs[0])
	hostB  endSeg // ef.b host contact rail, oriented outer→tHost (segs[2] is this, reversed)
	// armSurface overrides ef.armSurface for the trimmed arm FACE when non-nil — the cone-ruling canal arm
	// (CN4b-2) re-lofts over the CORNER span [x_f,far, x_f,C] (NOT the edge span), so the setback boundary is
	// the exact v-edge of the surface rather than a curve interior to it (which meshes the sliver past the
	// corner, D1 +119). nil for every torus/cylinder arm (its analytic surface already stops at the corner).
	armSurface geom.Surface
}

// armRailBundle assembles one arm's four boundary rails (§B.5). t0/t1 are the arm's two host-tangent
// points (the setback rail endpoints); the far terminus runs THROUGH the general far-runout engine
// (armFarRunout, FR3) — perpendicular caps take the existing farCrossSectionArc verbatim (byte-identity
// by call-graph, ADR-2) and pass the host rails through untouched; an oblique cap builds the analytic
// section trim (intersectArmCapping) and RE-TERMINATES the two host rails on the feet (ADR-4). Returns a
// non-empty reason (the exact obstruction) on any host-rail / setback / far-runout decline — do-no-harm.
func armRailBundle(set armSetback, ef edgeFillet, w cornerWeld, filletedEdges map[uint64]bool, res opstol.Resolution) (armRails, string) {
	if set.canalSpine != nil {
		return canalArmRailBundle(set, ef, w, filletedEdges, res) // CN4b-2: cone-ruling canal arm (spring rails + oblique/snout cap)
	}
	t0 := endpointOf(w.center, w.radius, set.railDir0) // host-tangent point on ef.a
	t1 := endpointOf(w.center, w.radius, set.railDir1) // host-tangent point on ef.b
	h0, ok0 := armHostContactRail(ef.a, set, t0, w, res)
	h1, ok1 := armHostContactRail(ef.b, set, t1, w, res)
	setback, ok2 := setbackEndSeg(w, set, t0, t1)
	if !ok0 || !ok1 || !ok2 {
		return armRails{}, "host contact rail or setback rail could not be built"
	}
	h0, h1, run, ok3, reason := armFarRunout(ef, w, h0, h1, filletedEdges, res)
	if !ok3 {
		return armRails{}, reason
	}
	return closeArmRails(h0, h1, setback, run), ""
}

// closeArmRails assembles one arm's four-rail boundary loop from its (possibly re-terminated) host rails,
// the setback great-arc, and the far runout trim reversed (outer1→outer0, closing to h0.from).
// reverseEndSegs reverses ANY curve — an Arc3d by its three points (a perpendicular cross-section arc,
// byte-identical to the pre-FR3 loop) or an analytic section curve via ReverseCurve3 (an oblique spiric/
// ellipse trim) — so both regimes close the loop the same way.
func closeArmRails(h0, h1, setback endSeg, run armRunout) armRails {
	segs := []endSeg{h0, setback}
	segs = append(segs, reverseEndSegs([]endSeg{h1})...)       // t1 → outer1
	segs = append(segs, reverseEndSegs([]endSeg{run.trim})...) // outer1 → outer0 (closes to h0.from)
	return armRails{segs: segs, far: run.trim, runout: run, hostA: h0, hostB: h1}
}

// filletedEdgeSet is the set of picked (filleted) edge ids at the welded corner — the arm edges gathered
// by cornerArms. The far-runout admission gate reads it to decline when a SECOND picked edge ends at an
// arm's far vertex (fillet-fillet interference, out of scope; architecture Q5).
func filletedEdgeSet(arms []edgeFillet) map[uint64]bool {
	set := make(map[uint64]bool, len(arms))
	for _, ef := range arms {
		set[ef.edge.ID()] = true
	}
	return set
}

// armHostContactRail builds one arm's contact rail on host, oriented outer→tHost: a torus arm carves a
// circular arc (curvedHostArc — the SAME constructor T5.3's retrim uses, so the two sides weld), a
// cylinder arm a straight ruling whose outer end is armRulingEnd (the SAME landing the retrim uses).
// Declines when neither applies.
func armHostContactRail(host *topo.Face, set armSetback, tHost math.Point3, w cornerWeld, res opstol.Resolution) (endSeg, bool) {
	tol := res.Weld() * w.radius
	switch s := set.arm.(type) {
	case geom.Torus:
		arc, ok := curvedHostArc(host.Geometry(), s, w, res)
		if !ok || float64(arc.PointAt(1).DistanceTo(tHost)) > tol {
			return endSeg{}, false // no torus rail on this host, or it misses the tangent point
		}
		return endSeg{from: arc.PointAt(0), to: tHost, curve: arc, mid: arc.PointAt(0.5), arc: true}, true
	case geom.Cylinder:
		outer, ok := cylinderRulingOuterOnHost(host, s, set, tHost, w, tol)
		if !ok {
			return endSeg{}, false
		}
		return endSeg{from: outer, to: tHost}, true
	}
	return endSeg{}, false
}

// cylinderRulingOuterOnHost is the SINGLE source of truth for a cylinder arm ruling's outer end on a
// host: it calls the SAME armRulingEnd (N2) the host retrim uses, with the host's OWN original loop and
// bitten vertex AND the arm's setback (its far-vertex authority), so the arm face and the retrimmed host
// land on a byte-identical point and weld without a crack. This replaced the former closed-form
// cylinderRulingOuter, which slid tHost to the arm FAR VERTEX's axial station: that equalled the host
// loop's extreme only when the far vertex happened to sit on the host rim (true for B3, a coincidence —
// NOT the identity the old docstring claimed). Routing both sides through one helper makes the coupling
// correct by construction, or declines to the floor.
func cylinderRulingOuterOnHost(host *topo.Face, arm geom.Cylinder, set armSetback, tHost math.Point3, w cornerWeld, tol float64) (math.Point3, bool) {
	segs := originalHostSegs(host)
	if len(segs) == 0 {
		return math.Point3{}, false // a host with no readable outer loop cannot anchor a ruling
	}
	v := bittenVertex(segs, w.center)
	return armRulingEnd(host, arm, set, tHost, v, segs, tol)
}

// setbackEndSeg wraps the arm↔sphere great-circle weld rail (T5.2) as an endSeg oriented t0→t1.
func setbackEndSeg(w cornerWeld, set armSetback, t0, t1 math.Point3) (endSeg, bool) {
	rail, ok := curvedSetbackRail(w, set)
	if !ok {
		return endSeg{}, false
	}
	return endSeg{from: t0, to: t1, curve: rail, mid: rail.PointAt(0.5), arc: true}, true
}

// farCrossSectionArc is the arm's terminal cross-section arc (radius r about the spine point at the far
// runout) joining the two host rails' outer ends — the torus tube quarter at the y=0 cut, or the
// through-cylinder foot arc on the cap it exits (§B.5). Shared verbatim with the far-runout host face.
func farCrossSectionArc(arm geom.Surface, r float64, outer0, outer1 math.Point3) (endSeg, bool) {
	spine, ok := armBallCenter(arm, outer0)
	if !ok {
		return endSeg{}, false
	}
	mid := arcMidBetween(spine, r, outer0, outer1)
	arc, err := geom.Arc3dByThreePoints(outer0, mid, outer1)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: outer0, to: outer1, curve: arc, mid: mid, arc: true}, true
}

// curvedArmTrimmedFace emits one trimmed arm face: the analytic arm surface bounded by its four rails
// (two host contact rails + the setback great-arc + the far cross-section arc). Its parent lineage is
// the generating filleted edge (ADR-0043, via filletEdgeProvenance — the same helper the planar cyl
// faces use) so the blend's edges/faces inherit a stable topological name that survives an upstream
// edit, rather than a build-order name that renumbers.
func curvedArmTrimmedFace(rails armRails, ef edgeFillet) filletFace {
	surface := ef.armSurface
	if rails.armSurface != nil {
		surface = rails.armSurface // CN4b-2: the canal arm re-lofted over the corner span [x_f,far, x_f,C]
	}
	return filletFace{
		surface: surface,
		loops:   []filletLoop{loopFromSegs(rails.segs)},
		parent:  filletEdgeProvenance(ef.edge),
	}
}
