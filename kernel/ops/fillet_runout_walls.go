// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// rimSubArc is one oriented boundary arc of a split boss-wall rim (or a host-plane notch): the exact
// footprint sub-arc curve traversed from its own start toward its end. Its interior samples are shared,
// point-for-point, with whichever neighbour face (patch/host) tiles the SAME curve (weld invariant).
type rimSubArc struct {
	curve geom.Curve3
}

// buildSplitBossWall rebuilds a boss's cylinder wall with its closed footprint rim SPLIT into the
// ordered sub-arcs (traced from the parametric seam point all the way around, back to it), so each
// sub-arc welds to exactly one neighbour (a flank/central patch on the dip side, the host-plane notch on
// the host side) and every rim edge is used exactly twice — the watertight invariant. The wall's seam
// and closed top rim are preserved under their original ids so the wall still welds to the boss cap.
// It mirrors buildSplitObstacleWall (fillet_obstacle_wall.go) for the double-interference boss.
func buildSplitBossWall(wall *topo.Face, footprint *topo.Edge, subs []rimSubArc) (filletFace, bool) {
	seam, ok := cylinderWallSeam(wall, footprint)
	if !ok {
		return filletFace{}, false
	}
	loop := traceRimSubArcs(subs)
	loop.addID(seam.bottom, nil, 0, seam.seamEdge)                 // rim close → seamB, then seam up
	loop.addID(seam.top, seam.topCurve, seam.topVID, seam.topEdge) // closed top rim (welds to the cap)
	loop.addID(seam.top, nil, seam.topVID, seam.seamEdge)          // seam down, closes the loop
	return filletFace{surface: wall.Geometry(), loops: []filletLoop{loop}, parent: wall.Lineage()}, true
}

// traceRimSubArcs samples each ordered sub-arc open (excluding its far endpoint, so consecutive sub-arcs
// concatenate without a duplicate vertex), producing the split rim as one point ring starting at the
// seam point and ending just before it (the seam is added by the caller). Segments are STRAIGHT chords
// (nil curve): the samples match the patch/host that tiled the same curve, and a LineSegment edge keeps
// sampleEdgeCurve from re-tracing the full rim arc over each 1/6-span sub-edge (the notch weld, 10b).
func traceRimSubArcs(subs []rimSubArc) filletLoop {
	var loop filletLoop
	for _, s := range subs {
		for _, p := range sampleCurve3Open(s.curve, false) {
			loop.addID(p, nil, 0, 0) // straight chord: the welded edge stays a LineSegment on the cylinder
		}
	}
	return loop
}

// bossWallSubArcs orders the five sub-arcs of a double-interference boss's footprint rim, traced from
// its parametric seam point all the way around and back to it: the host-side arc seam→crossA, the three
// dip arcs (crossA→seamA, seamA→seamB through the boss bottom, seamB→crossB) shared point-for-point with
// the flank/central patches, and the host-side arc crossB→seam. crossA/seamA/seamB/crossB is the fixed
// dip chain (a fillet crossing, its seam top/bottom, the other seam, the other crossing); the two host
// arcs are picked host-side by hostSideArc, so the ordering is correct wherever the seam falls (10b).
func bossWallSubArcs(im runoutImprint, seam, crossA, seamA, seamB, crossB math.Point3) ([]rimSubArc, bool) {
	host0, ok0 := hostSideArc(im, seam, crossA)
	dipA, ok1 := featureSubArc(im, crossA, seamA)
	dipMid, ok2 := featureSubArc(im, seamA, seamB)
	dipB, ok3 := featureSubArc(im, seamB, crossB)
	host1, ok4 := hostSideArc(im, crossB, seam)
	if !ok0 || !ok1 || !ok2 || !ok3 || !ok4 {
		return nil, false
	}
	return []rimSubArc{{host0}, {dipA}, {dipMid}, {dipB}, {host1}}, true
}
