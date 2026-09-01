// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// buildPatchFace wraps the certified corner-blend surface in a filletFace whose boundary loop is
// rebuilt HERE (not the provider's placeholder sampled ring) so every side is the SAME edge its
// neighbour carries: the wall line A→D (shared with the split wall seam), the two wing section arcs
// (shared with the wing faces), and the dip-side rim samples (shared with the split obstacle wall).
// This is what makes the four-sided patch weld with no T-junction (spec §3, Task 6 weld).
func buildPatchFace(ef edgeFillet, d obstacleDetection, og obstacleGeom, of *ObstacleFeature, patch CornerBlendPatch) filletFace {
	return filletFace{
		surface: patch.Surface,
		loops:   []filletLoop{patchBoundaryLoop(d, og, of)},
		parent:  filletEdgeProvenance(ef.edge),
	}
}

// patchBoundaryLoop traces the patch boundary A→(wall front)→D→P+→(dip rim)→P-→A: the wall front, the
// WingEnd arc, the dip-side rim (reversed, so it runs P+→P- antiparallel to the obstacle wall's forward
// rim), and the WingStart arc reversed. The rim segments carry the hole edge's source id so they share
// one identity with the split wall; the wing arcs are shared by value with the wing faces.
//
// The wall front is of.Canal.wallFront() when the exact surf-rst canal was accepted — the SAME slice
// orderedWallInserts subdivides the wall face's own tangent seam with, so the shared front is one list of
// points rather than two agreeing computations — and the bare straight A→D seam otherwise.
func patchBoundaryLoop(d obstacleDetection, og obstacleGeom, of *ObstacleFeature) filletLoop {
	dip := dipRimSamples(d) // [P-, interior..., P+]
	var loop filletLoop
	for _, p := range wallFrontInterior(og, of) { // A, interior wall-foot stations...
		loop.addID(p, nil, 0, 0)
	}
	loop.addID(og.wallD, og.endArc, 0, 0) // D -> P+ (WingEnd, wall->top)
	for i := len(dip) - 1; i >= 1; i-- {  // P+, interior... (down to, excluding, P-)
		loop.addID(dip[i], dipRimReverseCurve(d, dip, i), 0, d.holeEdge.ID())
	}
	startRev, _ := geom.Arc3dByThreePoints(d.pMinus, og.startMid, og.wallA)
	loop.addID(d.pMinus, startRev, 0, 0) // P- -> A (WingStart reversed) closes the ring
	return loop
}

// dipRimReverseCurve returns the curve for the patch boundary's REVERSED dip-rim segment dip[i]→dip[i-1]:
// whatever the split obstacle wall traces on that same segment forwards (a trimmed node sub-arc at the
// two ends, the rim's per-segment arc in between), re-derived in the reverse direction through its own
// recovered midpoint.
//
// The patch used to hand nil — a chord — for every dip-rim segment, and the interior segments still came
// out as arcs only because edgeCatalog.use took the wall face's curve and the wall face happens to be
// built first (it REPLACES a body face; the patch is appended as an extra). Agreement by build order is
// not agreement — even now that the catalog ADOPTS a later curve over an earlier nil, that adoption is
// a repair recorded as debt, not the design. Handing the same curve
// by value makes the shared segment's geometry independent of who reaches it first.
//
// A dip of exactly two points (both nodes bracketed by ONE rim segment) has no unambiguous forward
// segment to mirror, so it keeps the straight chord — the same answer insertNodesIntoRim gives that rim.
func dipRimReverseCurve(d obstacleDetection, dip []math.Point3, i int) geom.Curve3 {
	if len(dip) < 3 {
		return nil
	}
	return reversedArcThroughMid(dipRimForwardCurve(d, len(dip), i), dip[i], dip[i-1])
}

// dipRimForwardCurve is the curve the split obstacle wall traces FORWARD on the dip segment
// dip[i-1] → dip[i]: a node's trimmed sub-arc at either end of the dip, and the rim's own per-segment arc
// in between. The interior branch reads dipRimSampleIndex rather than re-deriving dipRimSamples' start
// convention here — the segment leaving dip[i-1] is the rim segment leaving dip[i-1]'s own sample.
func dipRimForwardCurve(d obstacleDetection, dipLen, i int) geom.Curve3 {
	switch i {
	case dipLen - 1:
		return d.rimTrims.in[1] // sample nodes[1].I -> P+
	case 1:
		return d.rimTrims.out[0] // P- -> sample nodes[0].I+1
	}
	return curveAt(d.holeSampled.curves, dipRimSampleIndex(d, i-1))
}

// wallFrontInterior returns the wall front from A up to (excluding) D — the points patchBoundaryLoop adds
// before it reaches D, so a straight seam contributes A alone and a canal contributes A plus every
// interior wall-foot station.
func wallFrontInterior(og obstacleGeom, of *ObstacleFeature) []math.Point3 {
	if of.Canal == nil {
		return []math.Point3{og.wallA}
	}
	front := of.Canal.wallFront()
	return front[:len(front)-1]
}

// buildSplitObstacleWall rebuilds the face behind the hole rim (e.g. the elliptical tube wall) with
// its bottom rim SPLIT at P±: the dip-side sub-arc welds to the patch, the host-side sub-arc to the
// notch, meeting only at P±. Each rim arc is thereby used exactly twice — the invariant that makes the
// mid-span obstacle body watertight (splitting only the base plane would leave the wall's closed rim a
// boundary edge). The wall's original seam and closed top rim are preserved. ok=false honest-rejects a
// wall whose seam does not sit on a rim sample (outside the single-dip slice).
func buildSplitObstacleWall(d obstacleDetection) (filletFace, bool) {
	seam, ok := wallSeamAndTop(d)
	if !ok {
		return filletFace{}, false
	}
	if seam.bottom.DistanceTo(d.holeSampled.pts[0]) > tol.ForPoints(d.holeSampled.pts).Weld() {
		return filletFace{}, false // the wall seam must coincide with rim sample 0 (single-dip slice)
	}
	loop := insertNodesIntoRim(d)
	// The seam.top vertex is SHARED with the untouched top-cap face, which carries it under its
	// original source-vertex id. The point-welder merges an id-carrying vertex only BY id (a fresh id
	// never coordinate-welds to an already-claimed cell, #1600 pinch-split rule), so the rebuilt wall
	// must present seam.top under that SAME id — else the two coincident closed top-rim edges weld to
	// different indices and the shell opens at the column top. seam.bottom stays id 0: it is shared
	// only with the rebuilt notch/patch rim (all id 0), which coordinate-weld among themselves.
	loop.addID(seam.bottom, nil, 0, seam.seamEdge)                 // seamB (2nd) -> seamT (seam up)
	loop.addID(seam.top, seam.topCurve, seam.topVID, seam.topEdge) // seamT -> seamT (closed top rim)
	loop.addID(seam.top, nil, seam.topVID, seam.seamEdge)          // seamT (2nd) -> seamB (seam down, closes)
	return filletFace{surface: d.obstacleWall.Geometry(), loops: []filletLoop{loop}, parent: d.obstacleWall.Lineage()}, true
}

// wallSeam bundles the obstacle wall's preserved rim/seam pieces: the bottom and top seam vertices,
// the seam line's edge id, and the closed top rim's edge id + curve.
type obstacleWallSeam struct {
	bottom, top math.Point3
	seamEdge    uint64
	topEdge     uint64
	topVID      uint64 // the top-rim seam vertex's source id — carried so the rebuilt wall welds to the cap
	topCurve    geom.Curve3
}

// wallSeamAndTop extracts the obstacle wall's seam line and its closed top rim from the wall's single
// loop: the bottom rim is the shared hole edge (skipped), the top rim is the other closed edge
// (StartVertex==EndVertex), the seam is the remaining open edge. ok=false on any unexpected shape.
func wallSeamAndTop(d obstacleDetection) (obstacleWallSeam, bool) {
	return cylinderWallSeam(d.obstacleWall, d.holeEdge)
}

// cylinderWallSeam extracts a swept wall face's preserved pieces from its single loop given its bottom
// rim edge: the bottom rim (holeEdge) is skipped, the closed top rim (StartVertex==EndVertex) yields
// top/topEdge/topCurve/topVID, and the remaining open edge is the parametric seam. Shared by the mid-
// span obstacle wall (elliptical tube) and the runout boss wall (circular cylinder), both of which
// preserve seam + closed top rim while their bottom rim is re-split (Task 10b, no duplication).
func cylinderWallSeam(wall *topo.Face, holeEdge *topo.Edge) (obstacleWallSeam, bool) {
	loops := wall.Loops()
	if len(loops) != 1 {
		return obstacleWallSeam{}, false
	}
	var s obstacleWallSeam
	for _, u := range loops[0].EdgeUses() {
		e := u.Edge()
		switch {
		case e.ID() == holeEdge.ID():
			continue
		case e.StartVertex() == e.EndVertex():
			s.topEdge, s.topCurve, s.top = e.ID(), e.Geometry(), e.StartVertex().Point()
			s.topVID = e.StartVertex().ID()
		default:
			s.seamEdge = e.ID()
		}
	}
	s.bottom = holeEdge.StartVertex().Point()
	return s, s.topEdge != 0 && s.seamEdge != 0
}

// insertNodesIntoRim traces the obstacle wall's bottom rim from seam sample 0 forward through every
// rim sample, splitting the two crossing segments by inserting the exact nodes P±. Each of the four
// chord segments touching an inserted node carries its TRIMMED sub-arc of that segment's own conic
// (d.rimTrims, the node's rim parameter as the trim point) — the untouched interior segments keep their
// exact per-segment arc, so the whole rim is traced on the rim. Every segment carries the rim's
// source-edge id so the split arcs share one identity across faces.
//
// Those four segments used to carry NO curve at all — a straight truncated chord — and that was the
// largest single off-surface residual in the obstacle class: measured on the shipped bodies, the chord
// sat its own sagitta off the very faces it bounds (R9 7.00e-03 off its r=8 boss cylinder, S3 9.43e-03
// off its cone, T6 1.17e-02 off its elliptical cylinder, U3 1.56e-02 / X3 2.96e-02 off the corner-blend
// patch), each matching the closed-form chord sagitta to a few tenths of a percent.
func insertNodesIntoRim(d obstacleDetection) filletLoop {
	var out filletLoop
	for i := 0; i < len(d.holeSampled.pts); i++ {
		p := d.holeSampled.pts[i]
		if node, split := insertedNodeAt(d, i); split {
			out.addID(p, node.in, 0, d.holeEdge.ID())
			out.addID(node.p, node.out, 0, d.holeEdge.ID())
			continue
		}
		out.addID(p, curveAt(d.holeSampled.curves, i), 0, d.holeEdge.ID())
	}
	return out
}

// insertedRimNode is one boundary node as the rim tracers consume it: the exact node point plus the two
// trimmed sub-arcs of the single rim segment it splits (in = sample→node, out = node→next sample).
type insertedRimNode struct {
	p       math.Point3
	in, out geom.Curve3
}

// insertedNodeAt reports whether rim sample i's outgoing segment carries a boundary crossing and, if
// so, which node (P− at nodes[0].I, P+ at nodes[1].I) must be inserted after it, with that segment's
// two trimmed sub-arcs. nodes[0] wins a tie (both nodes bracketed by ONE segment), preserving the
// single-insert behaviour this function has always had for that degenerate rim.
func insertedNodeAt(d obstacleDetection, i int) (insertedRimNode, bool) {
	switch i {
	case d.nodes[0].I:
		return insertedRimNode{p: d.pMinus, in: d.rimTrims.in[0], out: d.rimTrims.out[0]}, true
	case d.nodes[1].I:
		return insertedRimNode{p: d.pPlus, in: d.rimTrims.in[1], out: d.rimTrims.out[1]}, true
	}
	return insertedRimNode{}, false
}
