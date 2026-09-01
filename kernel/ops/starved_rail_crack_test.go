// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestStarvedRailAspectGateOff is the do-no-harm sanity check at the unit level (the CLAUDE.md "every
// new function gets a test" rule): a LOW-aspect cylindrical strip (arc length comparable to height,
// A well under tessellate.AspectDensifyThreshold) must leave discretizeEdge's straight-rail output at exactly 2
// points — byte-identical to the pre-#2009 path — even though the rail is still perfectly straight.
// Complements the fingerprint-pin byte-identity sweep (T6 A=2.13 and friends) with a minimal,
// self-contained repro that does not depend on the corpus fixtures.
func TestStarvedRailAspectGateOff(t *testing.T) {
	t.Parallel()
	const r, sweep, h = 2.0, 1.0, 2.5 // arc length r*sweep=2, height 2.5 → aspect 1.25 (<= 4)
	s := cylindricalStripSurface(t, r, sweep, h)
	face, rails := cylindricalStripFace(t, s, r, sweep, h)
	su, sv := tessellate.MetricScale(s)
	if a := tessellate.FaceAspect(s, su, sv); a > tessellate.AspectDensifyThreshold {
		t.Fatalf("low-aspect fixture measured aspect=%.2f, want <= %.1f", a, tessellate.AspectDensifyThreshold)
	}
	_ = face
	for i, e := range rails {
		pts := tessellate.DiscretizeEdge(e, DefaultQuality())
		if len(pts) != 2 {
			t.Errorf("rail %d: discretizeEdge returned %d points at aspect<=threshold, want 2 (byte-identical do-no-harm path)", i, len(pts))
		}
	}
}

// TestStarvedRailNoCrackAgainstNeighbourFace is hard gate 3 (the "no-crack" regression): a
// high-aspect B-spline panel's starved rail is shared with a plain, non-B-spline PLANAR neighbour
// face (the exact TestFilletRunOutToZero shape — a B-spline cap sharing a straight edge with an
// analytic neighbour). Proves densifyStarvedRail is CALLER-INDEPENDENT (starvedEdgeTarget scans
// e.Faces(), not a caller-supplied face — nurbs_pcurve_mesh.go): the shared edge's discretization is
// densified, and BOTH faces' own tessellations — the B-spline panel via nurbsPcurveMesh AND the plane
// via the ordinary earclip/loopBoundary path — end up with EXACTLY the same finer polyline (checked
// both by direct point-set comparison and by per-triangle-edge conformance ALONG THE SHARED RAIL —
// TRUE conformance, not merely a "no gap" collinearity argument).
func TestStarvedRailNoCrackAgainstNeighbourFace(t *testing.T) {
	t.Parallel()
	const r, sweep, h = bunchedR, bunchedSweep, bunchedH
	s := cylindricalStripSurfaceBunched(t, r, sweep, h, bunchedK)

	lin := topo.NewLineage(topo.Tok("test", "crack", 0))
	bld := topo.NewBuilder(false, lin)
	c00, c10, c11, c01 := s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)
	v00 := bld.AddVertex(c00, topo.NewLineage(topo.Tok("test", "v", 0)))
	v10 := bld.AddVertex(c10, topo.NewLineage(topo.Tok("test", "v", 1)))
	v11 := bld.AddVertex(c11, topo.NewLineage(topo.Tok("test", "v", 2)))
	v01 := bld.AddVertex(c01, topo.NewLineage(topo.Tok("test", "v", 3)))
	alpha := sweep / 2
	arcLo, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, -alpha, sweep)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	arcHi, err := geom.NewArc3d(math.P3(0, 0, h), math.V3(0, 0, 1), math.V3(1, 0, 0), r, -alpha, sweep)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	railV0 := bld.AddEdge(geom.NewLineSegment(c00, c10), v00, v10, topo.NewLineage(topo.Tok("test", "e", 0))) // the SHARED starved rail
	arcU1 := bld.AddEdge(arcHi, v10, v11, topo.NewLineage(topo.Tok("test", "e", 1)))
	railV1 := bld.AddEdge(geom.NewLineSegment(c11, c01), v11, v01, topo.NewLineage(topo.Tok("test", "e", 2)))
	arcU0 := bld.AddEdge(arcLo, v00, v01, topo.NewLineage(topo.Tok("test", "e", 3)))
	faceA := bld.AddFace(s, topo.NewLineage(topo.Tok("test", "faceA", 0)),
		topo.OuterLoop(topo.Fwd(railV0), topo.Fwd(arcU1), topo.Fwd(railV1), topo.Rev(arcU0)))

	// The neighbour: a plain rectangular PLANE hanging off the SAME railV0 edge (reversed, the
	// standard two-face-per-edge B-rep convention), width w in an arbitrary outward direction. A
	// low-aspect, non-B-spline face — the ordinary earclip path never gates on tessellate.FaceAspect at all.
	const w = 5.0
	outward := math.V3(0, -1, 0)
	c20 := c10.TranslateBy(outward.Scale(w))
	c30 := c00.TranslateBy(outward.Scale(w))
	v20 := bld.AddVertex(c20, topo.NewLineage(topo.Tok("test", "v", 4)))
	v30 := bld.AddVertex(c30, topo.NewLineage(topo.Tok("test", "v", 5)))
	eB1 := bld.AddEdge(geom.NewLineSegment(c10, c20), v10, v20, topo.NewLineage(topo.Tok("test", "e", 4)))
	eB2 := bld.AddEdge(geom.NewLineSegment(c20, c30), v20, v30, topo.NewLineage(topo.Tok("test", "e", 5)))
	eB3 := bld.AddEdge(geom.NewLineSegment(c30, c00), v30, v00, topo.NewLineage(topo.Tok("test", "e", 6)))
	plane, err := geom.NewPlane(c00, c00.VectorTo(c10).Cross(c00.VectorTo(c30)))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	// Fwd(railV0): faceB's loop must be a topologically CONNECTED cycle (v00→v10→v20→v30→v00) —
	// eB1/eB2/eB3 start where the previous edge-use ends, so railV0 must run v00→v10 here too (the
	// Builder does not validate edge-use connectivity, so a Rev/Fwd mismatch silently feeds
	// loopBoundary a disconnected "polygon", which is what a first draft of this test did — earclip
	// degenerated to a single wrong triangle on the garbage input, not a #2009 defect).
	faceB := bld.AddFace(plane, topo.NewLineage(topo.Tok("test", "faceB", 0)),
		topo.OuterLoop(topo.Fwd(railV0), topo.Fwd(eB1), topo.Fwd(eB2), topo.Fwd(eB3)))

	for _, gq := range gateQualities() {
		canonical := tessellate.DiscretizeEdge(railV0, gq.q)
		if len(canonical) <= 2 {
			t.Fatalf("%s quality: shared rail did not densify (len=%d); high-aspect neighbour faceA should have "+
				"triggered it", gq.name, len(canonical))
		}

		meshA := tessellate.TessellateFace(faceA, gq.q)
		meshB := tessellate.TessellateFace(faceB, gq.q)
		if meshA == nil || meshB == nil {
			t.Fatalf("%s quality: TessellateFace returned nil (faceA=%v faceB=%v)", gq.name, meshA == nil, meshB == nil)
		}
		for i, p := range canonical {
			if !meshContainsPoint(meshB, p, 1e-6*r) {
				t.Errorf("%s quality: neighbour plane's own mesh is missing shared-rail vertex %d (%v) — a T-junction crack",
					gq.name, i, p)
			}
			if !meshContainsPoint(meshA, p, 1e-6*r) {
				t.Errorf("%s quality: panel's own mesh is missing shared-rail vertex %d (%v)", gq.name, i, p)
			}
		}

		// A whole-mesh freeEdgeCount is the WRONG bar here: this synthetic 2-face body is deliberately
		// open (faceA's arc sides and faceB's other 3 sides have no third/fourth face closing them, so
		// they are LEGITIMATELY free — not a crack). The targeted check is per-triangle-edge
		// conformance ALONG THE SHARED RAIL specifically: each of its N-1 segments must be used by
		// EXACTLY 2 triangles (one from meshA, one from meshB) after welding.
		merged := &Mesh{}
		tessellate.MergeMesh(merged, meshA)
		tessellate.MergeMesh(merged, meshB)
		for i, deg := range railEdgeDegrees(t, merged, canonical) {
			if deg != 2 {
				t.Errorf("%s quality: rail segment %d (%v→%v) has %d incident triangles, want 2 — a crack",
					gq.name, i, canonical[i], canonical[i+1], deg)
			}
		}
	}
}

// railEdgeDegrees welds m's coincident vertices (model-relative grid) and returns, for each
// consecutive pair in rail, how many triangles use that exact edge — the TRUE per-triangle-edge
// conformance check (not merely "no gap"): 2 means the edge is shared cleanly by one triangle on
// each side of the rail; anything else is a crack or an overlap.
func railEdgeDegrees(t *testing.T, m *Mesh, rail []math.Point3) []int {
	t.Helper()
	grid := ResolutionForPoints(m.Positions).Weld()
	canon := map[[3]int64]int{}
	for i, p := range m.Positions {
		if _, ok := canon[tessellate.WeldKey(p, grid)]; !ok {
			canon[tessellate.WeldKey(p, grid)] = i
		}
	}
	weld := make([]int, len(m.Positions))
	for i, p := range m.Positions {
		weld[i] = canon[tessellate.WeldKey(p, grid)]
	}
	deg := map[[2]int]int{}
	for tr := 0; 3*tr+2 < len(m.Indices); tr++ {
		v := [3]int{weld[m.Indices[3*tr]], weld[m.Indices[3*tr+1]], weld[m.Indices[3*tr+2]]}
		for k := range 3 {
			a, b := v[k], v[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			deg[[2]int{a, b}]++
		}
	}
	degrees := make([]int, len(rail)-1)
	for i := 0; i+1 < len(rail); i++ {
		a, aok := canon[tessellate.WeldKey(rail[i], grid)]
		b, bok := canon[tessellate.WeldKey(rail[i+1], grid)]
		if !aok || !bok {
			t.Fatalf("rail point %d or %d not found in merged mesh", i, i+1)
		}
		if a > b {
			a, b = b, a
		}
		degrees[i] = deg[[2]int{a, b}]
	}
	return degrees
}

// meshContainsPoint reports whether m has a vertex within tol of p.
func meshContainsPoint(m *Mesh, p math.Point3, tol float64) bool {
	for _, q := range m.Positions {
		if q.DistanceTo(p) < tol {
			return true
		}
	}
	return false
}
