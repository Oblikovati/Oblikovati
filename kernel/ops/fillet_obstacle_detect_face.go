// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// detectObstacle looks for a mid-span obstacle on exactly ONE of the fillet's planar faces: a single
// through-hole whose rim genuinely dips PAST the receded fillet boundary (ADR-4, spec §3). The slice
// STRICTLY scopes itself to the single-host, straight-axis, constant-radius cylinder blend the rebuild
// can make watertight, and honest-rejects (ADR-3) everything else so the corpus stays green:
//   - a CURVED filleted edge (its rolling ball sweeps a torus/canal band, not a cylinder — the wing
//     split is invalid there; the S9 torus-band regression);
//   - a DUAL-HOST edge where BOTH fillet faces carry a dipping obstacle (a column piercing the fillet
//     CORNER — the patch would need a hole on both rails). obstacleFacesFor's dualObstacleRoute sees
//     this case via detectObstacles below and routes it to assembleDualObstacleSet — the U4 multi-rail
//     corner-blend build (derivation §3.1/§3.3, #2007 Group C); its U4-0 slice is a stub that still
//     honest-rejects, so this function's own qualifying!=1 gate is unchanged for now;
//   - a non-planar "wall" (patch G1-tangent) neighbour.
//
// Deferred cases fall back to the existing fillet path unchanged (their protruding hole stays a
// diagnostic tripwire, NOT folded into Valid, until the dual-host rebuild lands, U4-1..U4-5).
func detectObstacle(ef edgeFillet, res Resolution) (obstacleDetection, bool) {
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 1 { // 0 = no obstacle; 2 = dual-host (obstacleFacesFor's dualObstacleRoute)
		return obstacleDetection{}, false
	}
	found := dets[0]
	if _, ok := found.filletWall.Geometry().(geom.Plane); !ok {
		return obstacleDetection{}, false // the patch's G1 wall neighbour must be a clean plane
	}
	if !rebuildableTube(found.obstacleWall) {
		return obstacleDetection{}, false // the obstacle wall must be a tube wallSeamAndTop can split
	}
	return found, true
}

// detectObstacles runs the crossing/dip test (detectObstacleOnHost) against BOTH candidate hosts and
// keeps EVERY qualifying detection, where detectObstacle above keeps only the winner of a "found,
// qualifying" loop and then requires qualifying==1. Nothing new is DETECTED here — the two per-host
// results already existed — this is purely a packaging change: it is the seam that lets
// obstacleFacesFor's dualObstacleRoute see the qualifying==2 (dual-host) case, instead of it silently
// collapsing at detectObstacle's own single-host gate (derivation §3.1, U4-0, #2007 Group C). ok=true
// when at least one host qualifies (0 detections ⇒ ok=false, matching "no obstacle at all").
func detectObstacles(ef edgeFillet, res Resolution) ([]obstacleDetection, bool) {
	if ef.varying || !straightFilletEdge(ef, res) {
		return nil, false // curved/variable spine ⇒ torus/canal band, not a cylinder blend
	}
	var dets []obstacleDetection
	for _, hostIsA := range []bool{true, false} {
		host := ef.b
		if hostIsA {
			host = ef.a
		}
		if d, ok := detectObstacleOnHost(ef, host, hostIsA, res); ok {
			dets = append(dets, d)
		}
	}
	return analyticNodeDetections(dets, ef), len(dets) >= 1
}

// analyticNodeDetections re-solves every detection's two boundary nodes on its own EXACT rim curve
// (analyticNode) and re-lifts pMinus/pPlus from them. It runs HERE, once both hosts have been detected,
// rather than inside crossingDetection, because whether a node's exact answer is the fixed-tangent
// crossing at all depends on the OTHER host (coupledNodeStation). Detections are read from the input
// slice and written to a fresh one so a node's classification never sees a partially refined pair.
func analyticNodeDetections(dets []obstacleDetection, ef edgeFillet) []obstacleDetection {
	out := make([]obstacleDetection, len(dets))
	for i := range dets {
		out[i] = analyticNodeDetection(dets[i], otherHostDetection(dets, i), ef)
	}
	return out
}

// analyticNodeDetection re-solves one detection's nodes, skipping any node whose station the other
// host's boss already governs, and re-lifts the two 3D nodes through the host plane's exact inverse.
func analyticNodeDetection(d obstacleDetection, other *obstacleDetection, ef edgeFillet) obstacleDetection {
	boundary, ok := boundaryFromTangents(ef, d.hostIsA, d.flat)
	if !ok {
		return d // the tangent pair degenerated; keep the sampled nodes rather than guess
	}
	for i := range d.nodes {
		if coupledNodeStation(other, ef, d.back(d.nodes[i].P)) {
			continue
		}
		d.nodes[i] = analyticNode(d.nodes[i], d.holeEdge.Geometry(), len(d.holeSampled.pts), d.flat, boundary)
	}
	d.pMinus, d.pPlus = d.back(d.nodes[0].P), d.back(d.nodes[1].P)
	d.rimTrims = rimNodeTrimsOf(d) // one trim, three consumers — see rimNodeTrims
	return d
}

// otherHostDetection returns the OTHER host's detection of a dual-host pair, or nil when this filleted
// edge has only one qualifying host (then no node can be coupled — the ball is tangent to the other
// host all along the span).
func otherHostDetection(dets []obstacleDetection, i int) *obstacleDetection {
	if len(dets) != 2 {
		return nil
	}
	return &dets[1-i]
}

// coupledNodeStation reports whether the OTHER host's boss is ALREADY setting the rolling ball back at
// this node's axis station. Where it is, the node is NOT this rim's crossing of the fillet's own FIXED
// tangent line: the ball centre has migrated off the plain fillet axis under the other boss's setback,
// so the true node is where this rim meets the MIGRATING tangency foot — a coupled solve this slice does
// not model. U4's host-A node is the live instance: its exact coupled station is z = −6.2399856
// (DRAWEXE 8.0.0's own sliver pole (5.00625411, −20, −6.23998556) lies on boss A's r=8 rim to 1.4e-05),
// while the fixed-tangent closed form is −√39 = −6.2449980. Refining that node would move it 5.0e-03
// AWAY from the truth, so it is left exactly as the sampled polyline had it (do-no-harm) and the coupled
// solve stays a tracked follow-up — the honest-reject discipline the rest of this path uses (ADR-3).
// U4's host-B node is NOT coupled (at z = ±√44 boss A's rim has not yet reached the A-tangent), so it
// IS refined, and lands on OCCT's own patch end to 1.2e-11.
func coupledNodeStation(other *obstacleDetection, ef edgeFillet, node math.Point3) bool {
	if other == nil {
		return false
	}
	_, active := activeDipRimAt(*other, ef, axisParam(ef, node))
	return active
}

// rebuildableTube reports whether the obstacle wall is one of the tube surfaces buildSplitObstacleWall
// can split at the two nodes and re-weld: a straight elliptical/circular cylinder or a cone. A torus,
// b-spline, or other surface is a different band whose seam/rim structure wallSeamAndTop does not model,
// so the rebuild honest-rejects it (the S9/T9 mis-identified-wall case).
func rebuildableTube(wall *topo.Face) bool {
	switch wall.Geometry().(type) {
	case geom.Cylinder, geom.Cone, geom.EllipticalCylinder:
		return true
	}
	return false
}

// straightFilletEdge reports whether the filleted edge is geometrically straight, so the rolling-ball
// blend is a constant-radius CYLINDER whose cross-section is translation-invariant along the axis (the
// invariant computeObstacleGeom relies on). A curved edge sweeps a torus/canal band the wing-split
// cannot model. Tested by deviation from the start→end chord rather than curve type, since an imported
// STEP edge may be a b-spline that is geometrically straight. Model-relative tolerance (ADR-0042).
func straightFilletEdge(ef edgeFillet, res Resolution) bool {
	c := ef.edge.Geometry()
	a, b := ef.edge.StartVertex().Point(), ef.edge.EndVertex().Point()
	chord, err := geom.LineThrough(a, b)
	if err != nil {
		return false // degenerate (zero-length) edge
	}
	tol := res.Weld()
	for i := 1; i < 8; i++ {
		p := c.PointAt(float64(i) / 8)
		if chord.PointAt(a.VectorTo(p).Dot(chord.Dir.AsVector())).DistanceTo(p) > tol {
			return false
		}
	}
	return true
}

// detectObstacleOnHost runs the crossing/dip test for one candidate host face and, on success,
// packages the confirmed detection (host + neighbour faces, the sampled rim, the two nodes, the
// plane's flat/back frame). The hole must be ONE closed rim seam (a circular or elliptical
// through-feature); a multi-edge or non-closed hole is outside this single-dip slice.
func detectObstacleOnHost(ef edgeFillet, host *topo.Face, hostIsA bool, res Resolution) (obstacleDetection, bool) {
	pl, ok := host.Geometry().(geom.Plane)
	if !ok {
		return obstacleDetection{}, false
	}
	holeEdge, ok := singleHoleEdge(host)
	if !ok || holeEdge.StartVertex() != holeEdge.EndVertex() {
		return obstacleDetection{}, false
	}
	return crossingDetection(ef, host, hostIsA, pl, holeEdge, res)
}

// crossingDetection runs the shared band-crossing test (bandCrossings) against holeEdge already
// confirmed as host's single closed hole rim, then packages the obstacle only when it passes. The
// obstacle path has no use for the band line/side bandCrossings now also returns (that's the
// runout-imprint path's concern, fillet_runout_detect.go), so it discards them here. The two crossings
// are still the SAMPLED polyline's brackets at this point; detectObstacles re-solves them on the exact
// rim once both hosts are known (analyticNodeDetections).
func crossingDetection(ef edgeFillet, host *topo.Face, hostIsA bool, pl geom.Plane,
	holeEdge *topo.Edge, res Resolution) (obstacleDetection, bool) {
	sampled, nodes, flat, back, _, _, ok := bandCrossings(ef, hostIsA, pl, holeEdge, res)
	if !ok {
		return obstacleDetection{}, false
	}
	return packDetection(ef, host, hostIsA, holeEdge, sampled, nodes, flat, back), true
}

// bandCrossings builds the receded fillet boundary in host plane's 2D frame, samples the footprint
// rim, and admits it only when the rim crosses the boundary exactly twice AND the enclosed arc dips
// onto the fillet side (a genuine crossing, not a rim bulging away). Both detectObstacle's mid-span
// obstacle path (crossingDetection above) and detectRunouts' runout-imprint path (runoutOnHost,
// fillet_runout_detect.go) need this identical sequence — planeFrame → boundaryFromTangents →
// sampleHoleRim → project2D → obstacleNodes → filletBandSide → dipArcOrder → dipsPast; only their
// gating (dual-host reject vs independent-host admit) and result packaging differ, so it is extracted
// once here (CLAUDE.md: no duplication) and each caller adds its own gating around the ok=false/true
// result. dipArcOrder picks which of the two crossings-bounded arcs is the true dip — the two crossings
// split the closed rim, so testing the WRONG one is a silent false-reject (#2007 U3) — and REPLACES
// nodes with that order before returning: every downstream consumer (packDetection's pMinus/pPlus,
// buildObstacleFeature's wallA/wallD start/end arcs, buildNotchedHost, mergeObstacleRim's "Task 2
// dip-range convention" dip=nodes[0].I+1..nodes[1].I) hard-codes nodes[0]→nodes[1] AS the dip arc, so
// dipsPast's verdict and the rebuild's own notion of "the dip" must agree on the SAME arc, not just the
// boolean (an earlier version of this fix reordered only the dipsPast call and left nodes ascending —
// dipsPast then correctly said "yes, a dip", but the rebuild still notched around the ORIGINAL
// (majority, non-dip) arc: watertight, but ~21% off area — a self-consistent but wrong rebuild).
// It also returns the band line and its host/fillet side sign: the runout-imprint path (solveImprint,
// fillet_runout_imprint.go) needs both to pick the true outboard sub-arc by a signed test, rather than
// re-deriving (wrongly, for a deep dip) an arc-size heuristic — see outboardArc.
func bandCrossings(ef edgeFillet, hostIsA bool, pl geom.Plane, fp *topo.Edge, res Resolution) (
	sampled filletLoop, nodes [2]crossing, flat func(math.Point3) math.Point2, back func(math.Point2) math.Point3,
	boundary boundaryLine2, side float64, ok bool) {
	flat, back = planeFrame(pl)
	boundary, ok = boundaryFromTangents(ef, hostIsA, flat)
	if !ok {
		return filletLoop{}, [2]crossing{}, flat, back, boundary, 0, false
	}
	side = filletBandSide(ef, boundary, flat)
	sampled = sampleHoleRim(fp.Geometry(), fp.ID())
	rim2D := project2D(sampled.pts, flat)
	nodes, ok = obstacleNodes(rim2D, boundary, res)
	if !ok {
		return filletLoop{}, [2]crossing{}, flat, back, boundary, side, false
	}
	n0, n1 := dipArcOrder(nodes, len(rim2D))
	if !dipsPast(rim2D, n0, n1, boundary, side) {
		return filletLoop{}, [2]crossing{}, flat, back, boundary, side, false
	}
	nodes = [2]crossing{n0, n1}
	return sampled, nodes, flat, back, boundary, side, true
}

// packDetection fills the detection record, resolving the obstacle wall (the face sharing the hole
// rim with the host) and the fillet's other face (the wall the patch is G1-tangent to), and lifting
// the two crossings onto the host plane via the plane's exact inverse (back).
func packDetection(ef edgeFillet, host *topo.Face, hostIsA bool, holeEdge *topo.Edge,
	sampled filletLoop, nodes [2]crossing, flat func(math.Point3) math.Point2, back func(math.Point2) math.Point3) obstacleDetection {
	filletWall := ef.b
	if !hostIsA {
		filletWall = ef.a
	}
	return obstacleDetection{
		host:         host,
		filletWall:   filletWall,
		obstacleWall: otherFace(holeEdge, host),
		hostIsA:      hostIsA,
		holeEdge:     holeEdge,
		holeSampled:  sampled,
		nodes:        nodes,
		pMinus:       back(nodes[0].P),
		pPlus:        back(nodes[1].P),
		flat:         flat,
		back:         back,
	}
}

// singleHoleEdge returns the face's one interior (hole) loop's single closed edge, or ok=false when
// the face has no hole, more than one hole, or a hole made of several edges — all outside this
// single-dip slice, so the caller honest-rejects.
func singleHoleEdge(f *topo.Face) (*topo.Edge, bool) {
	var hole *topo.Loop
	for _, l := range f.Loops() {
		if l.IsOuter() {
			continue
		}
		if hole != nil {
			return nil, false
		}
		hole = l
	}
	if hole == nil || len(hole.EdgeUses()) != 1 {
		return nil, false
	}
	return hole.EdgeUses()[0].Edge(), true
}

// hostTangents returns the fillet's two tangent points ON the host plane (the receded boundary's
// endpoints): the c*.ta when the host is face a, the c*.tb when it is face b.
func hostTangents(ef edgeFillet, hostIsA bool) (math.Point3, math.Point3) {
	if hostIsA {
		return ef.c0.ta, ef.c1.ta
	}
	return ef.c0.tb, ef.c1.tb
}

// boundaryFromTangents builds the receded boundary line in the host plane's 2D frame: origin at the
// first host tangent point, unit direction toward the second. ok=false on a degenerate (coincident)
// tangent pair.
func boundaryFromTangents(ef edgeFillet, hostIsA bool, flat func(math.Point3) math.Point2) (boundaryLine2, bool) {
	t0, t1 := hostTangents(ef, hostIsA)
	dir, err := math.UnitVector2FromVector(flat(t0).VectorTo(flat(t1)))
	if err != nil {
		return boundaryLine2{}, false
	}
	return boundaryLine2{origin: flat(t0), dir: dir.AsVector()}, true
}

// filletBandSide returns the dipsPast side sign: +1 when the removed fillet band lies on the
// boundary's negative-signed-distance side. The filleted edge itself sits in that band, so the sign
// of its projected midpoint's signed distance is the negation of the side (spec §Numerical pitfalls).
func filletBandSide(ef edgeFillet, b boundaryLine2, flat func(math.Point3) math.Point2) float64 {
	mid := ef.edge.StartVertex().Point().Midpoint(ef.edge.EndVertex().Point())
	if b.signedDist(flat(mid)) < 0 {
		return 1
	}
	return -1
}

// planeFrame returns the flat/back pair for a plane: flat drops a point to the plane's own
// orthonormal (UAxis,VAxis) coordinates, back is its exact inverse. Unlike planeProjector (which
// drops an axis and has no general inverse), this pair round-trips any plane, which the notch merge
// and the node lift require (back(flat(p)) == p for p on the plane, to machine precision).
func planeFrame(pl geom.Plane) (func(math.Point3) math.Point2, func(math.Point2) math.Point3) {
	o, u, v := pl.Origin, pl.UAxis.AsVector(), pl.VAxis.AsVector()
	flat := func(p math.Point3) math.Point2 {
		d := o.VectorTo(p)
		return math.P2(d.Dot(u), d.Dot(v))
	}
	back := func(q math.Point2) math.Point3 {
		return o.TranslateBy(u.Scale(q.X)).TranslateBy(v.Scale(q.Y))
	}
	return flat, back
}

// sampleHoleRim discretizes the obstacle rim (a closed circle or ellipse) into obstacleRimSamples
// segments, each carrying a per-segment arc through its own three points and the rim edge's source id
// (so the notch, patch and split obstacle wall weld on one rim identity). The per-segment arc is exact
// for a circle and — for an ellipse over a 1/64 span — within the model weld, so the mesher samples the
// true rim rather than a chord. Sample 0 is the rim curve's t=0 point, which the closed-seam vertex
// sits on; the tube rebuild relies on that coincidence (else it honest-rejects).
func sampleHoleRim(rim geom.Curve3, edgeID uint64) filletLoop {
	var loop filletLoop
	n := obstacleRimSamples
	for i := range n {
		t0, t1 := float64(i)/float64(n), float64(i+1)/float64(n)
		loop.addID(rim.PointAt(t0), rimSegmentArc(rim, t0, t1), 0, edgeID)
	}
	return loop
}

// rimSegmentArc fits the arc of rim segment [t0,t1] through its start, midpoint and end — exact for a
// circular rim, and a faithful per-segment approximation of an elliptical rim over a small span. nil
// (a straight chord) when the three points are too collinear to define an arc. nodeSubArcs applies the
// same construction to a SUB-span of one segment, with the node pinned as an endpoint.
func rimSegmentArc(rim geom.Curve3, t0, t1 float64) geom.Curve3 {
	return rimArcThrough(rim.PointAt(t0), rim.PointAt((t0+t1)/2), rim.PointAt(t1))
}
