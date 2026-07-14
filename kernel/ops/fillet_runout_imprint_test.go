// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// hostPlaneZ0 is the z=0 plane with UAxis=X, VAxis=Y exactly, so its flat/back frame
// (planeFrame) is the identity on (x, y) — fixtures below can place points in the plane's
// 2D frame by literal (x, y) without a separate projection step.
func hostPlaneZ0() geom.Plane {
	pl, _ := geom.NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 1, 0))
	return pl
}

// squareHostFace builds a minimal planar face on pl, big enough to bound a footprint of the
// fixtures' size — only its vertices matter here (solveImprint's grazing-guard scale reads the
// host's bounding-box diagonal, not the face's actual shape).
func squareHostFace(pl geom.Plane, half float64) *topo.Face {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t3", "body", 0)))
	a := bld.AddVertex(math.P3(-half, -half, 0), topo.NewLineage(topo.Tok("t3", "v", 0)))
	b := bld.AddVertex(math.P3(half, -half, 0), topo.NewLineage(topo.Tok("t3", "v", 1)))
	c := bld.AddVertex(math.P3(0, half, 0), topo.NewLineage(topo.Tok("t3", "v", 2)))
	seg := func(p, q *topo.Vertex) geom.LineSegment { return geom.NewLineSegment(p.Point(), q.Point()) }
	edge := func(p, q *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(seg(p, q), p, q, topo.NewLineage(topo.Tok("t3", "e", i)))
	}
	ab, bc, ca := edge(a, b, 0), edge(b, c, 1), edge(c, a, 2)
	return bld.AddFace(pl, topo.NewLineage(topo.Tok("t3", "f", 0)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
}

// footprintCircleEdge wraps a 3D circle as a closed topo.Edge (start==end), matching how a real
// feature base curve is represented (kernel/topo.TestCircularEdgeReturnsCircle).
func footprintCircleEdge(c geom.Circle) *topo.Edge {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t3", "body", 1)))
	v := bld.AddVertex(c.PointAt(0), topo.NewLineage(topo.Tok("t3", "v", 3)))
	return bld.AddEdge(c, v, v, topo.NewLineage(topo.Tok("t3", "e", 3)))
}

// bandNodes returns two node points spanning the horizontal band line y=y0 in host-plane 2D,
// far apart so bandLineFromNodes reconstructs the line direction well-conditioned. solveImprint
// only reads their 2D position (see runoutImprint.nodes), so they need not sit on the circle.
func bandNodes(y0 float64) [2]crossing {
	return [2]crossing{{P: math.P2(-100, y0)}, {P: math.P2(100, y0)}}
}

// runoutImprintCircle is the named test fixture: a runoutImprint whose footprint is a circle of
// radius r centered at center (host-plane 2D), crossing the receded band described by nodes.
// side is bandCrossings' host/fillet sign (filletBandSide): every fixture here builds nodes via
// bandNodes, a line through y=y0 directed toward +x, whose signedDist is p.y−y0 (positive above
// the line) — so side=1 means "the fillet band is below the line, host is above" and side=-1 the
// reverse, matching what a real detection would have found for an edge sitting on that side.
func runoutImprintCircle(center math.Point2, r float64, nodes [2]crossing, side float64) runoutImprint {
	pl := hostPlaneZ0()
	flat, back := planeFrame(pl)
	circle, _ := geom.NewCircle(back(center), pl.Normal(), r)
	return runoutImprint{
		host:          squareHostFace(pl, 4*r),
		hostIsA:       true,
		plane:         pl,
		footprintEdge: footprintCircleEdge(circle),
		nodes:         nodes,
		boundary:      bandLineFromNodes(nodes),
		side:          side,
		flat:          flat,
		back:          back,
	}
}

// unitRes is a Resolution large enough not to interfere with the fixtures' scale (their model
// size is ~8-16 units); the grazing guard here is driven by the host bounding diagonal, not res.
func unitRes() Resolution { return ResolutionForSize(1) }

func TestSolveImprint_CircleCrossing(t *testing.T) {
	im := runoutImprintCircle(math.P2(0, 0), 8, bandNodes(-4), 1)
	cut, ok := solveImprint(im, unitRes())
	if !ok {
		t.Fatal("want crossing, got tangential")
	}
	wantX := stdmath.Sqrt(48)
	if stdmath.Abs(stdmath.Abs(cut.pMinus.X)-wantX) > 1e-9 {
		t.Fatalf("P- x=%v want ±%v", cut.pMinus.X, wantX)
	}
	if stdmath.Abs(stdmath.Abs(cut.pPlus.X)-wantX) > 1e-9 {
		t.Fatalf("P+ x=%v want ±%v", cut.pPlus.X, wantX)
	}
	if stdmath.Abs(cut.pMinus.Y+4) > 1e-9 || stdmath.Abs(cut.pPlus.Y+4) > 1e-9 {
		t.Fatalf("P± y = %v/%v, want -4", cut.pMinus.Y, cut.pPlus.Y)
	}
}

func TestSolveImprint_TangentRejected(t *testing.T) {
	im := runoutImprintCircle(math.P2(0, 0), 8, bandNodes(-8), 1)
	if _, ok := solveImprint(im, unitRes()); ok {
		t.Fatal("tangential line must not imprint")
	}
}

// TestSolveImprint_ArcIsOutboardMajorArc pins the sub-arc selection: for the circle∩y=-4 case
// the band clips a small cap off the BOTTOM of the circle (center above the line), so the
// outboard (away-from-band) arc is the major arc and must stay entirely at y > -4 (except at
// its endpoints, which sit exactly on the band).
func TestSolveImprint_ArcIsOutboardMajorArc(t *testing.T) {
	im := runoutImprintCircle(math.P2(0, 0), 8, bandNodes(-4), 1)
	cut, ok := solveImprint(im, unitRes())
	if !ok {
		t.Fatal("want crossing, got tangential")
	}
	arc, ok := cut.arc.(geom.Arc3d)
	if !ok {
		t.Fatalf("arc type = %T, want geom.Arc3d", cut.arc)
	}
	if stdmath.Abs(arc.SweepAngle) <= stdmath.Pi {
		t.Fatalf("sweep = %v, want |sweep| > π (the major arc)", arc.SweepAngle)
	}
	for i := 0; i <= 10; i++ {
		p := arc.PointAt(float64(i) / 10)
		if p.Y < -4-1e-9 {
			t.Fatalf("arc point %v dips below the band (y=-4)", p)
		}
	}
}

// TestSolveImprint_DeepDipSelectsMinorOutboardArc is Finding 1's regression: a DEEP dip, where
// the footprint circle's CENTER sits on the FILLET side of the band (here y=-6, below y=-4, the
// side bandNodes/side=1 marks as fillet — see runoutImprintCircle). Most of the circle is now on
// the fillet side, so the outboard (host, y>-4) cap is the MINOR arc — the old "pick the major
// arc" heuristic would have wrongly returned the huge fillet-side arc here. r=8, center-to-band
// distance=2 ⇒ half-angle=arccos(2/8)≈75.52°, so the host-side cap's sweep is ≈151.04° < π.
func TestSolveImprint_DeepDipSelectsMinorOutboardArc(t *testing.T) {
	im := runoutImprintCircle(math.P2(0, -6), 8, bandNodes(-4), 1)
	cut, ok := solveImprint(im, unitRes())
	if !ok {
		t.Fatal("want crossing, got tangential")
	}
	arc, ok := cut.arc.(geom.Arc3d)
	if !ok {
		t.Fatalf("arc type = %T, want geom.Arc3d", cut.arc)
	}
	if stdmath.Abs(arc.SweepAngle) >= stdmath.Pi {
		t.Fatalf("sweep = %v, want |sweep| < π (the minor, host-side arc)", arc.SweepAngle)
	}
	for i := 0; i <= 10; i++ {
		p := arc.PointAt(float64(i) / 10)
		if p.Y < -4-1e-9 {
			t.Fatalf("arc point %v dips below the band (y=-4): not the outboard/host-side arc", p)
		}
	}
}

// TestSolveImprintArc3d is the Arc3d-footprint regression (Task 7): imported STEP feature
// footprints arrive as geom.Arc3d, never geom.Circle, so solveImprint (via footprintConic) must
// accept them too — before this task it honest-rejected (ok=false) on every real fixture. Drives
// the REAL S1 substrate (runoutFixtureCrossingBoss + detectRunouts, not a hand-built fixture) so
// the footprint geometry is genuinely Arc3d, not a synthetic geom.Circle standing in for it.
func TestSolveImprintArc3d(t *testing.T) {
	ef, res := runoutFixtureCrossingBoss(t)
	imprints := detectRunouts(ef, res)
	if len(imprints) != 2 {
		t.Fatalf("want 2 imprints (S1's two independent bosses), got %d: %+v", len(imprints), imprints)
	}
	featureBCut, sawFeatureB := imprintCut{}, false
	for i, im := range imprints {
		arc, ok := im.footprintEdge.Geometry().(geom.Arc3d)
		if !ok {
			t.Fatalf("imprint %d footprint geometry = %T, want geom.Arc3d (fixture assumption changed)",
				i, im.footprintEdge.Geometry())
		}
		cut, ok := solveImprint(im, res)
		if !ok {
			t.Fatalf("imprint %d (radius %v): solveImprint ok=false, want true — Arc3d footprints must be accepted", i, arc.Radius)
		}
		if cut.pMinus.DistanceTo(cut.pPlus) <= res.Weld() {
			t.Fatalf("imprint %d: pMinus/pPlus degenerate (%v, %v)", i, cut.pMinus, cut.pPlus)
		}
		if stdmath.Abs(arc.Radius-8) < res.Weld() { // the r8 top boss (S1's feature-B, on ef.a/hostIsA==true)
			featureBCut, sawFeatureB = cut, true
		}
	}
	if !sawFeatureB {
		t.Fatal("no imprint carried the r8 top-boss (feature-B) footprint")
	}
	wantAbsX := stdmath.Sqrt(48) // top boss (r=8) crosses its receded band (offset 4) at half-width sqrt(8²−4²)
	if gotAbsX := stdmath.Abs(featureBCut.pPlus.X); stdmath.Abs(gotAbsX-wantAbsX) > res.Weld() {
		t.Fatalf("feature-B pPlus.X = %v, want |x| ≈ %v within %v", gotAbsX, wantAbsX, res.Weld())
	}
}

func TestLineCircleRoots_ExactChord(t *testing.T) {
	b := boundaryLine2{origin: math.P2(-100, -4), dir: math.V2(1, 0)}
	t0, t1, ok := lineCircleRoots(b, math.P2(0, 0), 8, 100)
	if !ok {
		t.Fatal("want two roots, got tangential")
	}
	x0, x1 := -100+t0, -100+t1
	wantX := stdmath.Sqrt(48)
	got := []float64{stdmath.Abs(x0), stdmath.Abs(x1)}
	for _, x := range got {
		if stdmath.Abs(x-wantX) > 1e-9 {
			t.Fatalf("root x=%v, want ±%v", x, wantX)
		}
	}
}

func TestLineCircleRoots_TangentRejected(t *testing.T) {
	b := boundaryLine2{origin: math.P2(-100, -8), dir: math.V2(1, 0)}
	if _, _, ok := lineCircleRoots(b, math.P2(0, 0), 8, 100); ok {
		t.Fatal("tangent chord must be rejected")
	}
}
