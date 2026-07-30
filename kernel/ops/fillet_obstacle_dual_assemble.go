// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// stationSnapTol is the bit-identical-modulo-reassociation tolerance for matching an axis station back
// to the boss node / interior-seam z it came from — the same convention outerDetection/sliverStations
// already use (axisParam evaluated on the SAME node point differs only by float re-association, never by
// measurement, fillet_obstacle_dual.go / _panel_sliver.go).
const stationSnapTol = 1e-9

// ringSeg is one traced side of a dual-host panel boundary as an ordered polyline: pts[i]→pts[i+1] runs
// along curves[i] (nil = straight chord), and every chord carries srcE so a side shared with a neighbour
// face (a boss rim shared with the wall, a wing arc shared with the wing face) welds by identity.
type ringSeg struct {
	pts    []math.Point3
	curves []geom.Curve3
	srcE   uint64
}

// reverseRingSeg reverses a traced side (used for the B-side rim and the low-z end rail, which are traced
// high-z→low-z in the boundary's forward winding). Chord i of the reverse runs from the original chord
// (len-2-i)'s far end, so the per-chord curve is re-derived through its recovered midpoint in the new
// direction (mirroring reverseOpenArc for the arc case; straight chords stay nil).
func reverseRingSeg(s ringSeg) ringSeg {
	n := len(s.pts)
	out := ringSeg{pts: make([]math.Point3, n), curves: make([]geom.Curve3, n-1), srcE: s.srcE}
	for i := 0; i < n; i++ {
		out.pts[i] = s.pts[n-1-i]
	}
	for i := 0; i+1 < n; i++ {
		seg := n - 2 - i
		if c := s.curves[seg]; c != nil {
			mid := c.PointAt((domainMid(c)))
			out.curves[i], _ = geom.Arc3dByThreePoints(out.pts[i], mid, out.pts[i+1])
		}
	}
	return out
}

// domainMid returns the parameter at the middle of c's domain (the arc's recovered midpoint anchor).
func domainMid(c geom.Curve3) float64 {
	lo, hi := c.Domain()
	return (lo + hi) / 2
}

// flattenRing concatenates the traced sides into one closed filletLoop: each side contributes all its
// points EXCEPT the last (which equals the next side's first, the shared corner), each carrying its
// outgoing chord's curve and rim/seam identity. The sides are pre-built so side k's last point equals
// side k+1's first point (and the final side's last point equals the first side's first point), so the
// result is a single continuous ring (assembleBody's point-welder then shares each corner across faces).
func flattenRing(segs []ringSeg) filletLoop {
	var fl filletLoop
	for _, s := range segs {
		for i := 0; i+1 < len(s.pts); i++ {
			fl.addID(s.pts[i], curveAt(s.curves, i), 0, s.srcE)
		}
	}
	return fl
}

// injectWingTangentInserts subdivides the INACTIVE host's receded front seam at the two wing-tangent
// points (the ±outer-node stations where the plain fillet cylinder gives way to a sliver — boss B reaches
// there, boss A does not, derivation §1.1 "B ⊃ A"), so both the outer wing's A-tangent line and the
// sliver's A-tangent line each weld to a real straight segment of the notch front rather than to a
// mid-air point (the derivation §3.2 orphan-seam caveat, pinned by U4-1's WingsWeldNoOrphanSeam test). It
// mutates the shared edgeInserts map BEFORE the notches are built, keyed by the filleted edge on the
// inactive host — the same edge-insert mechanism variable fillets already use to weld their ruling strips
// (#695). A no-op when the shape does not present the expected outer/inner detection split.
func injectWingTangentInserts(ef edgeFillet, dets []obstacleDetection, spans []panelSpan, maps filletRebuildMaps) {
	outer, ok := outerDetection(ef, dets, spans)
	if !ok {
		return
	}
	inner, ok := innerHost(dets, outer)
	if !ok {
		return
	}
	var pts []math.Point3
	for _, z := range []float64{axisParam(ef, outer.pMinus), axisParam(ef, outer.pPlus)} {
		if _, _, wallP, _, wok := sliverWingEnd(ef, outer, z); wok {
			pts = append(pts, wallP)
		}
	}
	if len(pts) != 2 {
		return
	}
	start := ef.edge.StartVertex().Point()
	if start.DistanceTo(pts[0]) > start.DistanceTo(pts[1]) {
		pts[0], pts[1] = pts[1], pts[0]
	}
	if maps.edgeInserts[inner.host] == nil {
		maps.edgeInserts[inner.host] = map[uint64][]math.Point3{}
	}
	maps.edgeInserts[inner.host][ef.edge.ID()] = pts
}

// innerHost returns the detection that is NOT the outer (bracketing) one — the host whose footprint the
// other fully brackets, so its front seam is the one that carries the sliver/wing tangent split.
func innerHost(dets []obstacleDetection, outer obstacleDetection) (obstacleDetection, bool) {
	for _, d := range dets {
		if d.host != outer.host {
			return d, true
		}
	}
	return obstacleDetection{}, false
}

// clearWingTangentInserts removes the front-seam inserts injectWingTangentInserts placed on the inner
// host, restoring the shared map to pristine when the dual assembly declines after injecting — so the
// edge's fallback path (single-host / runout / setback) transforms the host face exactly as it would
// have without the dual attempt (do-no-harm).
func clearWingTangentInserts(ef edgeFillet, dets []obstacleDetection, maps filletRebuildMaps) {
	for _, d := range dets {
		if m := maps.edgeInserts[d.host]; m != nil {
			delete(m, ef.edge.ID())
		}
	}
}

// buildDualWallsAndWings builds the two split obstacle walls and the two outer wings — the closure pieces
// that need NO front-seam inserts (the walls carry their own rims, the wings weld to the outer nodes). It
// validates the U4-class shape (both walls are rebuildable tubes split at every panel seam on their dip
// rim; the wings bracket at the outer nodes) so the caller can reject before mutating the shared map.
func buildDualWallsAndWings(ef edgeFillet, dets []obstacleDetection, spans, panels []panelSpan, res Resolution) (wallA, wallB filletFace, wings []filletFace, ok bool) {
	detA, detB, hok := hostDetections(dets)
	if !hok {
		return filletFace{}, filletFace{}, nil, false
	}
	wallA, okWA := buildDualSplitWall(detA, ef, interiorSeamStations(ef, panels, detA, res.Weld()), res.Weld())
	wallB, okWB := buildDualSplitWall(detB, ef, interiorSeamStations(ef, panels, detB, res.Weld()), res.Weld())
	if !okWA || !okWB {
		return filletFace{}, filletFace{}, nil, false
	}
	wings, wok := buildOuterWings(ef, dets, spans)
	if !wok {
		return filletFace{}, filletFace{}, nil, false
	}
	return wallA, wallB, wings, true
}

// interiorSeamStations returns the panel-boundary stations strictly inside det's own dip band that are NOT
// weld-coincident with one of det's rim samples — exactly the stations det's obstacle wall must be split
// at (beyond its own two nodes) so its dip rim welds edge-for-edge to the panels. A station that lands on
// an existing rim sample (boss A's z=0) needs no split: the sample already breaks the rim there.
func interiorSeamStations(ef edgeFillet, panels []panelSpan, det obstacleDetection, weld float64) []float64 {
	zLo, zHi := axisParam(ef, det.pMinus), axisParam(ef, det.pPlus)
	if zLo > zHi {
		zLo, zHi = zHi, zLo
	}
	var out []float64
	for _, z := range panelBoundaryStations(panels) {
		if z <= zLo+stationSnapTol || z >= zHi-stationSnapTol {
			continue // det's own nodes / outside its dip band
		}
		ps, ok := dipRimPointAtStation(det, ef, z)
		if !ok || nearestDipSampleDist(det, ps) < weld {
			continue // no crossing, or already split by a coincident rim sample
		}
		out = append(out, z)
	}
	return out
}

// panelBoundaryStations returns the sorted distinct z stations bounding the four panels (each panel's
// zLo/zHi), the full set of seam/wing stations the closure and panels must agree on.
func panelBoundaryStations(panels []panelSpan) []float64 {
	var zs []float64
	for _, p := range panels {
		zs = append(zs, p.zLo, p.zHi)
	}
	var out []float64
	for _, z := range zs {
		dup := false
		for _, o := range out {
			if stdmath.Abs(o-z) <= stationSnapTol {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, z)
		}
	}
	return out
}

// nearestDipSampleDist returns the distance from p to the nearest of det's dip-rim samples (dipRimSamples)
// — the test for whether a station already coincides with a rim break.
func nearestDipSampleDist(det obstacleDetection, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, s := range dipRimSamples(det) {
		if d := p.DistanceTo(s); d < best {
			best = d
		}
	}
	return best
}

// buildDualSplitWall rebuilds the obstacle wall (buildSplitObstacleWall's structure) with its bottom rim
// split not only at the two boundary nodes but at each interior seam station (spliceRimPoint), so a
// dual-host wall welds edge-for-edge to the sliver/core panels that share its dip rim. With no interior
// stations it is byte-identical to buildSplitObstacleWall (insertNodesIntoRim + preserved seam/top).
func buildDualSplitWall(d obstacleDetection, ef edgeFillet, stations []float64, weld float64) (filletFace, bool) {
	seam, ok := wallSeamAndTop(d)
	if !ok {
		return filletFace{}, false
	}
	if seam.bottom.DistanceTo(d.holeSampled.pts[0]) > ResolutionForPoints(d.holeSampled.pts).Weld() {
		return filletFace{}, false
	}
	loop := insertNodesIntoRim(d)
	for _, z := range stations {
		ps, okP := dipRimPointAtStation(d, ef, z)
		if !okP {
			return filletFace{}, false
		}
		loop, ok = spliceRimPoint(loop, d.holeEdge.Geometry(), ps, d.holeEdge.ID(), weld)
		if !ok {
			return filletFace{}, false
		}
	}
	loop.addID(seam.bottom, nil, 0, seam.seamEdge)
	loop.addID(seam.top, seam.topCurve, seam.topVID, seam.topEdge)
	loop.addID(seam.top, nil, seam.topVID, seam.seamEdge)
	return filletFace{surface: d.obstacleWall.Geometry(), loops: []filletLoop{loop}, parent: d.obstacleWall.Lineage()}, true
}

// spliceRimPoint inserts the exact station point ps into the rim polyline on whichever segment it lies
// (nearest by point-to-segment distance — ps is on the true rim, the bracketing chord is its own secant),
// replacing that segment with two sub-segments carrying the rim's srcE. The panels' rim sides use the
// SAME station endpoint (dipRimPointAtStation) and the SAME interior samples, so the wall's two new
// sub-segments weld station-for-station to them. A ps already at a segment endpoint (already split there)
// is left untouched.
//
// The split segment's CURVE is now split with it (splitRimSegmentCurve) rather than dropped. Dropping it
// silently discarded, on U4, two of the four trimmed node sub-arcs insertNodesIntoRim had just built —
// segments terminating exactly on the receded boundary y = −15 — plus an interior rim arc, which is why
// the rim-trim slice measured as fully cancelled on that case.
func spliceRimPoint(loop filletLoop, rim geom.Curve3, ps math.Point3, srcE uint64, weld float64) (filletLoop, bool) {
	n := len(loop.pts)
	best, bestD := -1, stdmath.Inf(1)
	for i := 0; i < n; i++ {
		seg := geom.NewLineSegment(loop.pts[i], loop.pts[(i+1)%n])
		if d := geom.DistancePointToSegment(seg, ps); d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return loop, false
	}
	if ps.DistanceTo(loop.pts[best]) < weld || ps.DistanceTo(loop.pts[(best+1)%n]) < weld {
		return loop, true // already effectively split at this chord's endpoint
	}
	var out filletLoop
	in, off := splitRimSegmentCurve(rim, curveAt(loop.curves, best), loop.pts[best], ps, loop.pts[(best+1)%n], weld)
	for i := 0; i < n; i++ {
		if i == best {
			out.addID(loop.pts[i], in, loop.srcV[i], srcE)
			out.addID(ps, off, 0, srcE)
			continue
		}
		out.addID(loop.pts[i], loop.curves[i], loop.srcV[i], loop.srcE[i])
	}
	return out, true
}

// splitRimSegmentCurve trims a rim segment at the interior station ps into the two sub-arcs a→ps and
// ps→b, each read off the RIM's own parameterisation (rimSubArcBetween) — the identical construction
// nodeSubArcs applies at a boundary node and rimSegmentArc gives every untouched interior segment.
//
// It used to recover the split parameter by inverting ps onto the SEGMENT's own conic and then require
// that conic to reproduce ps to within the model weld. On the only path that ships it that test can never
// pass: U4's boss-B rim is an imported b-spline whose per-segment circular fit sits ~1e-7 off it, while
// the station is solved on the exact rim to ~1e-12, so the inversion measured 8.723146e-08 against a
// 3.394e-08 weld and BOTH halves fell back to straight chords — the rim-trim slice's own fix, cancelled
// on U4's four node-adjacent halves exactly as spliceRimPoint had cancelled the slice before it. (The
// regression test that was supposed to cover this split each segment at a point ON ITS ARC, which is the
// one input for which the old inversion could succeed and the shipped path never supplies.)
//
// nil in ⇒ nil out — a segment with no curve (an unrefined node's honest chord) splits into two chords —
// and so does a station that is not strictly inside its own rim span (rimStationSplitsSpan).
func splitRimSegmentCurve(rim, c geom.Curve3, a, ps, b math.Point3, weld float64) (geom.Curve3, geom.Curve3) {
	if c == nil || !rimStationSplitsSpan(rim, a, ps, b, weld) {
		return nil, nil
	}
	return rimSubArcBetween(rim, a, ps, weld), rimSubArcBetween(rim, ps, b, weld)
}

// dipRimSideCurves returns the rim sub-arc of every segment of a traced dip-rim side. Each is
// rimSubArcBetween of that segment's OWN two endpoints — the same pure function of the same two values
// the split obstacle wall reads for the same segment — so the shared dip-rim edge carries one
// parameterisation instead of whichever face the first-writer-wins edge catalog happened to reach first.
func dipRimSideCurves(rim geom.Curve3, pts []math.Point3, weld float64) []geom.Curve3 {
	out := make([]geom.Curve3, len(pts)-1)
	for i := range out {
		out[i] = rimSubArcBetween(rim, pts[i], pts[i+1], weld)
	}
	return out
}

// buildDualPanelFaces resolves each of the four panels' fill surface (extractPanelLoop → resolveBlend:
// coons4 for the slivers, the exact-station canal loft for the cores) and wraps it in a filletFace whose
// boundary loop is traced HERE (dualPanelBoundary) — NOT the RailLoop's fitted rails — so every side is
// the SAME edge its neighbour carries (a boss rim shared with the wall/notch, a wing arc shared with the
// wing, an interior seam shared with the adjacent panel), the discipline buildPatchFace already applies
// for the single-host patch. ok=false honest-rejects the whole set if any panel declines.
func buildDualPanelFaces(ef edgeFillet, dets []obstacleDetection, panels []panelSpan, res Resolution) ([]filletFace, bool) {
	detA, detB, ok := hostDetections(dets)
	if !ok {
		return nil, false
	}
	faces := make([]filletFace, 0, len(panels))
	for _, span := range panels {
		loop, lok := extractPanelLoop(span, dets, ef, res)
		if !lok {
			return nil, false
		}
		patch, pok := resolveBlend(loop, res)
		if !pok {
			return nil, false
		}
		bnd, bok := dualPanelBoundary(span, dets, detA, detB, ef, res)
		if !bok {
			return nil, false
		}
		faces = append(faces, filletFace{surface: patch.Surface, loops: []filletLoop{bnd}, parent: filletEdgeProvenance(ef.edge)})
	}
	return faces, true
}

// dualPanelBoundary traces ONE panel's watertight boundary loop as four welded sides (derivation §1.4):
// the A-side (boss-A rim sub-arc when boss A dips there, else the plain fillet A-tangent line over a
// sliver), the zHi end (a wing quarter-arc at the outer B node, else an interior setback seam), the
// B-side (boss-B rim sub-arc, reversed), and the zLo end (wing arc / seam, reversed). Every corner is a
// node / exact rim point / seam point BOTH the panel and its neighbour compute from the same pure
// function, so the shell welds with no drift (ADR-0042). ok=false when any side/corner declines.
func dualPanelBoundary(span panelSpan, dets []obstacleDetection, detA, detB obstacleDetection, ef edgeFillet, res Resolution) (filletLoop, bool) {
	aSeg, ok := panelASideSeg(span, detA, detB, ef, res)
	if !ok {
		return filletLoop{}, false
	}
	hiSeg, ok := panelEndSeg(span, span.zHi, dets, detA, detB, ef, res, false)
	if !ok {
		return filletLoop{}, false
	}
	bReg, ok := panelBSideSeg(span, detB, ef, res)
	if !ok {
		return filletLoop{}, false
	}
	bSeg := reverseRingSeg(bReg)
	loSeg, ok := panelEndSeg(span, span.zLo, dets, detA, detB, ef, res, true)
	if !ok {
		return filletLoop{}, false
	}
	return flattenRing([]ringSeg{aSeg, hiSeg, bSeg, loSeg}), true
}

// panelASideSeg builds the A-side (from zLo to zHi): a boss-A dip-rim sub-arc when boss A dips over the
// span (the core panels), else the plain fillet A-tangent line over a B-only sliver (from the A-node seam
// corner to the wing-tangent corner). The rim sub-arc carries boss A's hole-edge identity so it welds to
// wallA / notchA; the tangent line is op-generated (srcE 0) and welds to the notch front the wing-tangent
// insert already split.
func panelASideSeg(span panelSpan, detA, detB obstacleDetection, ef edgeFillet, res Resolution) (ringSeg, bool) {
	if span.hostA {
		pts, ok := dipRimSubArcSnapped(detA, ef, span.zLo, span.zHi, res)
		if !ok {
			return ringSeg{}, false
		}
		return ringSeg{pts: pts, curves: dipRimSideCurves(detA.holeEdge.Geometry(), pts, res.Weld()), srcE: detA.holeEdge.ID()}, true
	}
	a0, ok := aCornerInactive(detA, detB, ef, span.zLo)
	if !ok {
		return ringSeg{}, false
	}
	a1, ok := aCornerInactive(detA, detB, ef, span.zHi)
	if !ok {
		return ringSeg{}, false
	}
	return ringSeg{pts: []math.Point3{a0, a1}, curves: []geom.Curve3{nil}, srcE: 0}, true
}

// panelBSideSeg builds the B-side as boss B's dip-rim sub-arc from zLo to zHi (boss B dips over every
// panel), carrying boss B's hole-edge identity so it welds to wallB / notchB. dualPanelBoundary reverses
// it into the loop's forward winding.
func panelBSideSeg(span panelSpan, detB obstacleDetection, ef edgeFillet, res Resolution) (ringSeg, bool) {
	pts, ok := dipRimSubArcSnapped(detB, ef, span.zLo, span.zHi, res)
	if !ok {
		return ringSeg{}, false
	}
	return ringSeg{pts: pts, curves: dipRimSideCurves(detB.holeEdge.Geometry(), pts, res.Weld()), srcE: detB.holeEdge.ID()}, true
}

// panelEndSeg builds a z-end rail (A-corner→B-corner at station z): the full fillet quarter arc when z is
// the outer B node (the wing weld — the SAME wingSection arc the wing face carries), else the interior
// setback seam (setbackSection sampled into chords, endpoints snapped to the panel's A/B corners). reverse
// flips it for the low-z end (traced B→A in the loop's winding).
func panelEndSeg(span panelSpan, z float64, dets []obstacleDetection, detA, detB obstacleDetection, ef edgeFillet, res Resolution, reverse bool) (ringSeg, bool) {
	aC, ok := aCorner(span, detA, detB, ef, z)
	if !ok {
		return ringSeg{}, false
	}
	bC, ok := rimCorner(detB, ef, z)
	if !ok {
		return ringSeg{}, false
	}
	var seg ringSeg
	if isNodeStation(detB, ef, z) {
		arc, _, wallP, node, wok := sliverWingEnd(ef, detB, z)
		if !wok {
			return ringSeg{}, false
		}
		seg = ringSeg{pts: []math.Point3{wallP, node}, curves: []geom.Curve3{arc}, srcE: 0}
		_ = aC
	} else {
		s, sok := seamSeg(z, dets, ef, res, aC, bC)
		if !sok {
			return ringSeg{}, false
		}
		seg = s
	}
	if reverse {
		seg = reverseRingSeg(seg)
	}
	return seg, true
}

// seamSeg builds an interior setback seam rail as a SINGLE arc segment pinned to the panel's two welded
// corners (aC/bC): the radius-r arc through [aC, setbackSection(z)'s own outward apex, bC]. Emitting ONE
// clean curved rail (not a chord fan) keeps the sliver panel's boundary the same 4-rail shape the
// production tessellator's aspect-aware densification (#2009) was tuned for — a chord-fan long side folds
// the extreme-aspect sliver even though the equal-area core is fine. Both neighbouring panels call this
// with the SAME z and corners, so the shared seam edge is bit-identical (its endpoints ARE the welded
// corners, and Arc3dByThreePoints is a pure function) and welds with no drift. Falls back to a straight
// chord when the three points are collinear (a vanishingly thin section, §2.4).
func seamSeg(z float64, dets []obstacleDetection, ef edgeFillet, res Resolution, aC, bC math.Point3) (ringSeg, bool) {
	sec, ok := setbackSection(z, dets, ef, res)
	if !ok {
		return ringSeg{}, false
	}
	apex := sec.PointAt(domainMid(sec))
	arc, err := geom.Arc3dByThreePoints(aC, apex, bC)
	if err != nil {
		return ringSeg{pts: []math.Point3{aC, bC}, curves: []geom.Curve3{nil}, srcE: 0}, true
	}
	return ringSeg{pts: []math.Point3{aC, bC}, curves: []geom.Curve3{arc}, srcE: 0}, true
}

// aCorner returns the panel's A-side corner at station z: the boss-A rim/node point when boss A dips over
// the span (core), else the inactive-host corner (the A-node seam corner or the wing-tangent corner) over
// a sliver.
func aCorner(span panelSpan, detA, detB obstacleDetection, ef edgeFillet, z float64) (math.Point3, bool) {
	if span.hostA {
		return rimCorner(detA, ef, z)
	}
	return aCornerInactive(detA, detB, ef, z)
}

// aCornerInactive returns a B-only sliver's A-side corner: boss A's own node where the sliver borders the
// core (the seam end), else the wing-tangent point on host A where the sliver borders the outer wing.
func aCornerInactive(detA, detB obstacleDetection, ef edgeFillet, z float64) (math.Point3, bool) {
	if p, ok := nodePointAt(detA, ef, z); ok {
		return p, true
	}
	_, _, wallP, _, ok := sliverWingEnd(ef, detB, z)
	return wallP, ok
}

// rimCorner returns boss det's rim corner at station z: det's exact boundary node when z is one of det's
// own two nodes (so the panel welds to the notch/wall node point, NOT the axis-plane rim crossing which is
// a sagitta off it — measured ~6.5e-3 at U4's boss A nodes), else the exact dip-rim crossing at z (the
// same point the wall is split at and setbackSection ends at).
func rimCorner(det obstacleDetection, ef edgeFillet, z float64) (math.Point3, bool) {
	if p, ok := nodePointAt(det, ef, z); ok {
		return p, true
	}
	return dipRimPointAtStation(det, ef, z)
}

// nodePointAt returns det's own boundary node (pMinus/pPlus) when z matches its axis station, else false.
func nodePointAt(det obstacleDetection, ef edgeFillet, z float64) (math.Point3, bool) {
	if stdmath.Abs(axisParam(ef, det.pMinus)-z) <= stationSnapTol {
		return det.pMinus, true
	}
	if stdmath.Abs(axisParam(ef, det.pPlus)-z) <= stationSnapTol {
		return det.pPlus, true
	}
	return math.Point3{}, false
}

// isNodeStation reports whether z is one of det's own two boundary-node axis stations.
func isNodeStation(det obstacleDetection, ef edgeFillet, z float64) bool {
	_, ok := nodePointAt(det, ef, z)
	return ok
}

// dipRimSubArcSnapped returns boss det's dip-rim points from zLo to zHi: the two exact corners (rimCorner
// — det's node at a node station, else the axis-plane crossing) and the interior dip samples strictly
// between them (dipRimSamples), skipping any sample weld-coincident with a corner. The wall's matching
// sub-arc (spliceRimPoint into the same dip samples) and the neighbouring panels' corners are all the SAME
// points, so the boss rim welds edge-for-edge across wall + notch + panels.
func dipRimSubArcSnapped(det obstacleDetection, ef edgeFillet, zLo, zHi float64, res Resolution) ([]math.Point3, bool) {
	lo, ok := rimCorner(det, ef, zLo)
	if !ok {
		return nil, false
	}
	hi, ok := rimCorner(det, ef, zHi)
	if !ok {
		return nil, false
	}
	weld := res.Weld()
	pts := []math.Point3{lo}
	for _, p := range dipRimSamples(det) {
		z := axisParam(ef, p)
		if z <= zLo+stationSnapTol || z >= zHi-stationSnapTol {
			continue
		}
		if p.DistanceTo(lo) < weld || p.DistanceTo(hi) < weld {
			continue
		}
		pts = append(pts, p)
	}
	return append(pts, hi), true
}
