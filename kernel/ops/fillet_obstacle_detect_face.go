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
//     CORNER — the patch would need a hole on both rails; the S1/S4/T1/T4 case, deferred to Phase 2);
//   - a non-planar "wall" (patch G1-tangent) neighbour.
//
// Deferred cases fall back to the existing fillet path unchanged (their protruding hole stays a
// diagnostic tripwire, NOT folded into Valid, until the Phase-2 corner engine handles them).
func detectObstacle(ef edgeFillet, res Resolution) (obstacleDetection, bool) {
	if ef.varying || !straightFilletEdge(ef, res) {
		return obstacleDetection{}, false // curved/variable spine ⇒ torus/canal band, not a cylinder blend
	}
	var found obstacleDetection
	qualifying := 0
	for _, hostIsA := range []bool{true, false} {
		host := ef.b
		if hostIsA {
			host = ef.a
		}
		if d, ok := detectObstacleOnHost(ef, host, hostIsA, res); ok {
			found, qualifying = d, qualifying+1
		}
	}
	if qualifying != 1 { // 0 = no obstacle; 2 = dual-host corner pierce (Phase 2)
		return obstacleDetection{}, false
	}
	if _, ok := found.filletWall.Geometry().(geom.Plane); !ok {
		return obstacleDetection{}, false // the patch's G1 wall neighbour must be a clean plane
	}
	if !rebuildableTube(found.obstacleWall) {
		return obstacleDetection{}, false // the obstacle wall must be a tube wallSeamAndTop can split
	}
	return found, true
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
// runout-imprint path's concern, fillet_runout_detect.go), so it discards them here.
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
// sampleHoleRim → project2D → obstacleNodes → filletBandSide → dipsPast; only their gating (dual-host
// reject vs independent-host admit) and result packaging differ, so it is extracted once here
// (CLAUDE.md: no duplication) and each caller adds its own gating around the ok=false/true result.
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
	if !dipsPast(rim2D, nodes[0], nodes[1], boundary, side) {
		return filletLoop{}, [2]crossing{}, flat, back, boundary, side, false
	}
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
	for i := 0; i < n; i++ {
		t0, t1 := float64(i)/float64(n), float64(i+1)/float64(n)
		loop.addID(rim.PointAt(t0), rimSegmentArc(rim, t0, t1), 0, edgeID)
	}
	return loop
}

// rimSegmentArc fits the arc of rim segment [t0,t1] through its start, midpoint and end — exact for a
// circular rim, and a faithful per-segment approximation of an elliptical rim over a small span. nil
// (a straight chord) when the three points are too collinear to define an arc.
func rimSegmentArc(rim geom.Curve3, t0, t1 float64) geom.Curve3 {
	arc, err := geom.Arc3dByThreePoints(rim.PointAt(t0), rim.PointAt((t0+t1)/2), rim.PointAt(t1))
	if err != nil {
		return nil
	}
	return arc
}
