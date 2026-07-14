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
func runoutImprintCircle(center math.Point2, r float64, nodes [2]crossing) runoutImprint {
	pl := hostPlaneZ0()
	flat, back := planeFrame(pl)
	circle, _ := geom.NewCircle(back(center), pl.Normal(), r)
	return runoutImprint{
		host:          squareHostFace(pl, 4*r),
		hostIsA:       true,
		plane:         pl,
		footprintEdge: footprintCircleEdge(circle),
		nodes:         nodes,
		flat:          flat,
		back:          back,
	}
}

// unitRes is a Resolution large enough not to interfere with the fixtures' scale (their model
// size is ~8-16 units); the grazing guard here is driven by the host bounding diagonal, not res.
func unitRes() Resolution { return ResolutionForSize(1) }

func TestSolveImprint_CircleCrossing(t *testing.T) {
	im := runoutImprintCircle(math.P2(0, 0), 8, bandNodes(-4))
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
	im := runoutImprintCircle(math.P2(0, 0), 8, bandNodes(-8))
	if _, ok := solveImprint(im, unitRes()); ok {
		t.Fatal("tangential line must not imprint")
	}
}

// TestSolveImprint_ArcIsOutboardMajorArc pins the sub-arc selection: for the circle∩y=-4 case
// the band clips a small cap off the BOTTOM of the circle (center above the line), so the
// outboard (away-from-band) arc is the major arc and must stay entirely at y > -4 (except at
// its endpoints, which sit exactly on the band).
func TestSolveImprint_ArcIsOutboardMajorArc(t *testing.T) {
	im := runoutImprintCircle(math.P2(0, 0), 8, bandNodes(-4))
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
