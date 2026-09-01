// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// buildObstacleFeature turns a confirmed detection into the ObstacleFeature the corner-blend provider
// consumes plus the shared cross-section geometry (obstacleGeom) the wings/patch/wall rebuild reuse
// BY VALUE. The wing section arcs are the true fillet-cylinder cross-sections at the two nodes, so
// they weld to the wing faces by identity; the two nodes are the exact rim∩boundary crossings, shared
// with RimArcPts' endpoints to machine precision (spec §3, the corner-weld invariant, ADR-0042).
func buildObstacleFeature(ef edgeFillet, d obstacleDetection, res tol.Resolution) (*ObstacleFeature, obstacleGeom, bool) {
	og, ok := computeObstacleGeom(ef, d)
	if !ok {
		return nil, obstacleGeom{}, false
	}
	wallInto, ok := wallIntoDir(ef, d, og)
	if !ok {
		return nil, obstacleGeom{}, false
	}
	of := &ObstacleFeature{
		RimCurve:  d.holeEdge.Geometry(),
		RimArcPts: dipRimSamples(d),
		Nodes:     [2]math.Point3{d.pMinus, d.pPlus},
		WingStart: og.startArc,
		WingEnd:   og.endArc,
		WallLine:  geom.NewLineSegment(og.wallA, og.wallD),
		HostPlane: d.host.Geometry().(geom.Plane),
		Radius:    ef.cyl.Radius,
		BlendAxis: ef.cyl.AxisDir.AsVector(),
		WallInto:  wallInto,
	}
	of.Canal = buildObstacleCanal(ef, d, og, of, res) // nil ⇒ the straight-seam Coons model, wall front included
	return of, og, true
}

// computeObstacleGeom builds each node's cylinder cross-section (wall-tangent point, arc midpoint, and
// the wall→top quarter arc) from the constant radial vectors of corner c0 — the cross-section is
// translation-invariant along a straight fillet's axis, so c0's radials evaluated at each node give
// the exact section there. The section's TOP endpoint is pinned to the node itself (not recomputed),
// so it is bit-identical to the notch/patch node (the weld invariant).
func computeObstacleGeom(ef edgeFillet, d obstacleDetection) (obstacleGeom, bool) {
	hostRadial, wallRadial, midRadial := cornerRadials(ef, d.hostIsA)
	startArc, start, ok0 := wingSection(d.pMinus, hostRadial, wallRadial, midRadial)
	endArc, end, ok1 := wingSection(d.pPlus, hostRadial, wallRadial, midRadial)
	if !ok0 || !ok1 {
		return obstacleGeom{}, false
	}
	return obstacleGeom{
		wallA: start.wall, wallD: end.wall,
		startArc: startArc, endArc: endArc, startMid: start.mid, endMid: end.mid,
		startCen: start.cen, endCen: end.cen,
	}, true
}

// cornerRadials returns the constant radial vectors from the fillet cylinder axis (corner c0's centre)
// to the host-tangent point, the wall-tangent point, and the arc midpoint. On the host face the
// tangent is c0.ta (host==a) or c0.tb (host==b); the wall tangent is the other one.
func cornerRadials(ef edgeFillet, hostIsA bool) (hostRadial, wallRadial, midRadial math.Vector3) {
	c := ef.c0
	hostTan, wallTan := c.tb, c.ta
	if hostIsA {
		hostTan, wallTan = c.ta, c.tb
	}
	return c.cen.VectorTo(hostTan), c.cen.VectorTo(wallTan), c.cen.VectorTo(c.mid)
}

// wingSectionPts is one node's cylinder cross-section: its ball centre on the fillet axis, the
// wall-tangent point, and the arc midpoint.
type wingSectionPts struct {
	cen, wall, mid math.Point3
}

// wingSection places the cylinder cross-section at a node: the axis centre is node−hostRadial, from
// which the wall-tangent point and arc midpoint follow by the constant radials. The section is the
// wall→top quarter arc (Arc3dByThreePoints through the midpoint), matching Task 3's WingStart
// convention (its wall point → top point).
func wingSection(node math.Point3, hostRadial, wallRadial, midRadial math.Vector3) (geom.Arc3d, wingSectionPts, bool) {
	cen := node.TranslateBy(hostRadial.Scale(-1))
	pts := wingSectionPts{cen: cen, wall: cen.TranslateBy(wallRadial), mid: cen.TranslateBy(midRadial)}
	arc, err := geom.Arc3dByThreePoints(pts.wall, pts.mid, node)
	return arc, pts, err == nil
}

// dipRimSamples returns the obstacle rim's DIP-side arc from P− to P+ inclusive (the samples between
// the two crossings that poke onto the fillet side), with the two endpoints OVERWRITTEN to the exact
// nodes so RimArcPts' ends equal ObstacleFeature.Nodes to machine precision (the rail-fit weld gate).
func dipRimSamples(d obstacleDetection) []math.Point3 {
	n := len(d.holeSampled.pts)
	pts := []math.Point3{d.pMinus}
	for i := (d.nodes[0].I + 1) % n; i != (d.nodes[1].I+1)%n; i = (i + 1) % n {
		pts = append(pts, d.holeSampled.pts[i])
	}
	return append(pts, d.pPlus)
}

// dipRimSampleIndex maps an INTERIOR dipRimSamples position (1 <= i <= len(dip)-2) back to the rim sample
// it came from. dipRimSamples' start convention — dip[1] is rim sample nodes[0].I+1 — is written down
// HERE and read from here (dipRimForwardCurve), rather than re-derived at each reader: an independently
// re-derived copy of an index convention is the shape of defect that has already cost this vein four
// T-junction leaks.
func dipRimSampleIndex(d obstacleDetection, i int) int {
	return (d.nodes[0].I + i) % len(d.holeSampled.pts)
}

// wallIntoDir returns the unit WallInto vector: in the fillet wall's plane, perpendicular to the
// blend axis (the seam), pointing AWAY from the host plane into the wall material. The sign is anchored
// to the wall-tangent point's signed offset from the host plane, so it is robust to whichever way the
// stored plane normals face (spec §3 / corner_blend_obstacle Build's orientInward guard).
func wallIntoDir(ef edgeFillet, d obstacleDetection, og obstacleGeom) (math.Vector3, bool) {
	wallPl, ok := d.filletWall.Geometry().(geom.Plane)
	if !ok {
		return math.Vector3{}, false
	}
	hostPl := d.host.Geometry().(geom.Plane)
	cand, err := math.UnitVector3FromVector(ef.cyl.AxisDir.AsVector().Cross(wallPl.Normal()))
	if err != nil {
		return math.Vector3{}, false
	}
	c := cand.AsVector()
	offset := hostPl.Normal().Dot(hostPl.Origin.VectorTo(og.wallA))
	if c.Dot(hostPl.Normal())*offset < 0 {
		c = c.Scale(-1)
	}
	return c, true
}

// buildNotchedHost rebuilds the host plane face by transformFace (its receded outer loop, identical to
// the non-obstacle path) and then merging the sampled hole rim into that outer loop (mergeHoleIntoNotch,
// Task 5) — the outer boundary detours up around the host-side rim arc, the dip-side rim goes to the
// patch, and the two meet only at the new split vertices P±. ok=false honest-rejects a malformed merge.
func buildNotchedHost(d obstacleDetection, maps filletRebuildMaps) (filletFace, bool) {
	base := transformFace(d.host, maps.forFace(d.host, 0))
	outerIdx, ok := outerLoopIndex(d.host)
	if !ok {
		return filletFace{}, false
	}
	notch, ok := mergeHoleIntoNotch(base.loops[outerIdx], d.holeSampled, d.nodes, d.flat, d.back, d.rimTrims)
	if !ok {
		return filletFace{}, false
	}
	return filletFace{surface: base.surface, loops: []filletLoop{notch}, parent: base.parent}, true
}

// outerLoopIndex returns the index of the face's outer loop (transformFace preserves loop order, so the
// same index selects the receded outer loop of the transformed face).
func outerLoopIndex(f *topo.Face) (int, bool) {
	for i, l := range f.Loops() {
		if l.IsOuter() {
			return i, true
		}
	}
	return 0, false
}

// buildObstacleWings splits the fillet cylinder into two wing faces at the obstacle band: the left wing
// runs from corner c0 to the node nearer c0, the right wing from the node nearer c1 to corner c1. Each
// wing keeps cylinderFace's winding logic, and its node-end section arc is shared (by value) with the
// patch rail so the two weld with no T-junction.
func buildObstacleWings(ef edgeFillet, d obstacleDetection, og obstacleGeom) []filletFace {
	nearIdx, farIdx := nearFarNodes(ef, d)
	left := buildWingFace(ef, nodeSection(d, og, nearIdx), true)
	right := buildWingFace(ef, nodeSection(d, og, farIdx), false)
	return []filletFace{left, right}
}

// nearFarNodes returns which node index (0 or 1) lies nearer corner c0 along the cylinder axis, so the
// left wing (c0 side) and right wing (c1 side) attach to the correct node.
func nearFarNodes(ef edgeFillet, d obstacleDetection) (int, int) {
	axis := ef.cyl.AxisDir.AsVector()
	if ef.c0.cen.VectorTo(d.pMinus).Dot(axis) <= ef.c0.cen.VectorTo(d.pPlus).Dot(axis) {
		return 0, 1
	}
	return 1, 0
}

// wingCut is one node's cross-section split points: the tangent points on faces a and b (nodeTa,
// nodeTb) and the arc midpoint, resolved for the host orientation so the wing loop reads them exactly
// as cylinderFace reads a corner's ta/tb/mid.
type wingCut struct {
	nodeTa, nodeTb, mid math.Point3
}

// nodeSection resolves the split points at node idx into face-a/face-b tangent points: on the host
// face the tangent is the node itself, on the wall face it is the section's wall point.
func nodeSection(d obstacleDetection, og obstacleGeom, idx int) wingCut {
	node, wallP, mid := d.pMinus, og.wallA, og.startMid
	if idx == 1 {
		node, wallP, mid = d.pPlus, og.wallD, og.endMid
	}
	if d.hostIsA {
		return wingCut{nodeTa: node, nodeTb: wallP, mid: mid}
	}
	return wingCut{nodeTa: wallP, nodeTb: node, mid: mid}
}

// buildWingFace assembles one wing's cylinder face with the mid-span obstacle path's single analytic
// quarter-arc cut section (cutSegs nil).
func buildWingFace(ef edgeFillet, cut wingCut, leftWing bool) filletFace {
	return buildWingFaceCut(ef, cut, leftWing, nil)
}

// buildWingFaceCut assembles one wing's cylinder face: the truncated tangent lines, the node cut
// section, and the untouched c0/c1 rounded end (cornerEndSegs) — then applies cylinderFace's winding
// flip. cutSegs is the cut section oriented nodeTa→nodeTb; nil ⇒ the single analytic quarter-arc. The
// runout path (Task 10b) passes the flank patch's arm arc sampled into ringSegSamples chords so the
// wing and patch share those vertices (class 1, no T-junction).
func buildWingFaceCut(ef edgeFillet, cut wingCut, leftWing bool, cutSegs []endSeg) filletFace {
	segs := leftWingSegs(ef, cut, cutSegs)
	if !leftWing {
		segs = rightWingSegs(ef, cut, cutSegs)
	}
	if cylinderSegsFlipped(ef, segs) != ef.flip {
		segs = reverseEndSegs(segs)
	}
	return filletFace{surface: ef.cyl, loops: []filletLoop{loopFromSegs(segs)}, parent: filletEdgeProvenance(ef.edge)}
}

// cutSectionSegs returns the wing's cut section oriented nodeTa→nodeTb: the caller's sampled chain when
// present (already nodeTa→nodeTb), else the single analytic quarter-arc through cut.mid.
func cutSectionSegs(cut wingCut, cutSegs []endSeg) []endSeg {
	if cutSegs != nil {
		return cutSegs
	}
	arc, _ := geom.Arc3dByThreePoints(cut.nodeTa, cut.mid, cut.nodeTb)
	return []endSeg{{from: cut.nodeTa, to: cut.nodeTb, curve: arc, mid: cut.mid, arc: true}}
}

// leftWingSegs is the c0→node wing boundary: A-tangent c0.ta→nodeTa, the node cut section nodeTa→nodeTb,
// B-tangent nodeTb→c0.tb, and the reversed c0 rounded end — cylinderFace's chain with the c1 end
// replaced by the node cut.
func leftWingSegs(ef edgeFillet, cut wingCut, cutSegs []endSeg) []endSeg {
	segs := []endSeg{{from: ef.c0.ta, to: cut.nodeTa}}
	segs = append(segs, cutSectionSegs(cut, cutSegs)...)
	segs = append(segs, endSeg{from: cut.nodeTb, to: ef.c0.tb})
	return append(segs, reverseEndSegs(cornerEndSegs(ef.c0, nil, ef.splitEnds))...)
}

// rightWingSegs is the node→c1 wing boundary: A-tangent nodeTa→c1.ta, the c1 rounded end, B-tangent
// c1.tb→nodeTb, and the reversed node cut section nodeTb→nodeTa — cylinderFace's chain with the c0 end
// replaced by the node cut.
func rightWingSegs(ef edgeFillet, cut wingCut, cutSegs []endSeg) []endSeg {
	segs := []endSeg{{from: cut.nodeTa, to: ef.c1.ta}}
	segs = append(segs, cornerEndSegs(ef.c1, nil, ef.splitEnds)...)
	segs = append(segs, endSeg{from: ef.c1.tb, to: cut.nodeTb})
	return append(segs, reverseEndSegs(cutSectionSegs(cut, cutSegs))...)
}
