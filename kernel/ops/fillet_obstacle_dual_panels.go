// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// This file is the DUAL-host corner-blend PANEL half of the dual obstacle assembly: the four panel faces
// buildDualPanelFaces raises between the two bosses, the ring segments their boundaries are traced from,
// and the dip-rim sub-arc sampling those sides read. Its sibling fillet_obstacle_dual_assemble.go keeps
// the split obstacle WALLS and the wing-tangent inserts. They were one file until it passed the 500-line
// budget; the seam between them is exactly the seam assembleDualObstacleSet already calls across
// (buildDualWallsAndWings vs buildDualPanelFaces).

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
		return ringSeg{pts: pts, srcE: detA.holeEdge.ID(),
			curves: dipRimSideCurves(detA.holeEdge.Geometry(), pts, res.Weld())}, true
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
	return ringSeg{pts: pts, srcE: detB.holeEdge.ID(),
		curves: dipRimSideCurves(detB.holeEdge.Geometry(), pts, res.Weld())}, true
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
