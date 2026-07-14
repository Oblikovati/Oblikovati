// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// buildPatchFace wraps the certified corner-blend surface in a filletFace whose boundary loop is
// rebuilt HERE (not the provider's placeholder sampled ring) so every side is the SAME edge its
// neighbour carries: the wall line A→D (shared with the split wall seam), the two wing section arcs
// (shared with the wing faces), and the dip-side rim samples (shared with the split obstacle wall).
// This is what makes the four-sided patch weld with no T-junction (spec §3, Task 6 weld).
func buildPatchFace(ef edgeFillet, d obstacleDetection, og obstacleGeom, patch CornerBlendPatch) filletFace {
	return filletFace{
		surface: patch.Surface,
		loops:   []filletLoop{patchBoundaryLoop(d, og)},
		parent:  filletEdgeProvenance(ef.edge),
	}
}

// patchBoundaryLoop traces the patch boundary A→D→P+→(dip rim)→P-→A: the wall line, the WingEnd arc,
// the dip-side rim (reversed, so it runs P+→P- antiparallel to the obstacle wall's forward rim), and
// the WingStart arc reversed. The rim segments carry the hole edge's source id so they share one
// identity with the split wall; the wing arcs are shared by value with the wing faces.
func patchBoundaryLoop(d obstacleDetection, og obstacleGeom) filletLoop {
	dip := dipRimSamples(d) // [P-, interior..., P+]
	var loop filletLoop
	loop.addID(og.wallA, nil, 0, 0)       // A -> D (wall seam)
	loop.addID(og.wallD, og.endArc, 0, 0) // D -> P+ (WingEnd, wall->top)
	for i := len(dip) - 1; i >= 1; i-- {  // P+, interior... (down to, excluding, P-)
		loop.addID(dip[i], nil, 0, d.holeEdge.ID())
	}
	startRev, _ := geom.Arc3dByThreePoints(d.pMinus, og.startMid, og.wallA)
	loop.addID(d.pMinus, startRev, 0, 0) // P- -> A (WingStart reversed) closes the ring
	return loop
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
	if seam.bottom.DistanceTo(d.holeSampled.pts[0]) > ResolutionForPoints(d.holeSampled.pts).Weld() {
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
// rim sample, splitting the two crossing segments by inserting the exact nodes P±. The two chord
// segments touching each inserted node carry NO curve (a straight truncated chord, matching the notch
// and patch there); the untouched interior segments keep their exact per-segment ellipse arc. Every
// segment carries the rim's source-edge id so the split arcs share one identity across faces.
func insertNodesIntoRim(d obstacleDetection) filletLoop {
	var out filletLoop
	for i := 0; i < len(d.holeSampled.pts); i++ {
		p := d.holeSampled.pts[i]
		if node, split := insertedNodeAt(d, i); split {
			out.addID(p, nil, 0, d.holeEdge.ID())
			out.addID(node, nil, 0, d.holeEdge.ID())
			continue
		}
		out.addID(p, curveAt(d.holeSampled.curves, i), 0, d.holeEdge.ID())
	}
	return out
}

// insertedNodeAt reports whether rim sample i's outgoing segment carries a boundary crossing and, if
// so, which node (P− at nodes[0].I, P+ at nodes[1].I) must be inserted after it.
func insertedNodeAt(d obstacleDetection, i int) (math.Point3, bool) {
	switch i {
	case d.nodes[0].I:
		return d.pMinus, true
	case d.nodes[1].I:
		return d.pPlus, true
	}
	return math.Point3{}, false
}
