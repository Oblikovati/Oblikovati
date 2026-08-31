// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	m "oblikovati.org/math"
)

var containLineage = topo.NewLineage(topo.Tok("t", "contain", 0))

// circleEdgeUse adds a full-circle edge (its two endpoints weld to one seam vertex) on the Z=0 plane.
func circleEdgeUse(bld *topo.Builder, cx, cy, r float64) topo.Use {
	c, err := geom.NewCircle(m.P3(m.Scalar(cx), m.Scalar(cy), 0), m.V3(0, 0, 1), r)
	if err != nil {
		panic(err)
	}
	seam := bld.AddVertex(c.PointAt(0), containLineage)
	return topo.Fwd(bld.AddEdge(c, seam, seam, containLineage))
}

// polygonLoopUses adds a closed run of straight edges through the given planar points.
func polygonLoopUses(bld *topo.Builder, pts []m.Point3) []topo.Use {
	vs := make([]*topo.Vertex, len(pts))
	for i, p := range pts {
		vs[i] = bld.AddVertex(p, containLineage)
	}
	uses := make([]topo.Use, len(pts))
	for i := range pts {
		j := (i + 1) % len(pts)
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), vs[i], vs[j], containLineage))
	}
	return uses
}

// planeFaceContained builds a single planar (Z=0) face with the given outer and inner loops and returns
// the HolesContained verdict from the containment check in isolation.
func planeFaceContained(build func(bld *topo.Builder) (outer, inner []topo.Use)) bool {
	bld := topo.NewBuilder(false, containLineage)
	outer, inner := build(bld)
	pl, _ := geom.NewPlane(m.P3(0, 0, 0), m.V3(0, 0, 1))
	bld.AddFace(pl, containLineage, topo.OuterLoop(outer...), topo.InnerLoop(inner...))
	body := bld.Build()
	rep := &ValidationReport{HolesContained: true}
	rep.checkHoleContainment(body)
	return rep.HolesContained
}

// TestHoleContainmentCircleInsideCircle: a small circular hole well inside a circular outer is accepted.
func TestHoleContainmentCircleInsideCircle(t *testing.T) {
	ok := planeFaceContained(func(bld *topo.Builder) ([]topo.Use, []topo.Use) {
		return []topo.Use{circleEdgeUse(bld, 0, 0, 10)}, []topo.Use{circleEdgeUse(bld, 0, 0, 2)}
	})
	if !ok {
		t.Fatal("a circular hole of radius 2 inside a circular outer of radius 10 must be contained")
	}
}

// TestHoleContainmentCircleCrossesOuter: an off-centre hole whose circle straddles the outer boundary is
// a genuine protrusion — the exact circle-vs-circle crossing (not a chord) rejects it (#3478).
func TestHoleContainmentCircleCrossesOuter(t *testing.T) {
	ok := planeFaceContained(func(bld *topo.Builder) ([]topo.Use, []topo.Use) {
		return []topo.Use{circleEdgeUse(bld, 0, 0, 10)}, []topo.Use{circleEdgeUse(bld, 9, 0, 2)}
	})
	if ok {
		t.Fatal("a hole circle centred at (9,0) r=2 reaches radius 11 > 10 and must be rejected")
	}
}

// TestHoleContainmentConcentricLargerHole: a concentric hole larger than the outer never crosses it, so
// the point-in-region parity (not a crossing) is what rejects it.
func TestHoleContainmentConcentricLargerHole(t *testing.T) {
	ok := planeFaceContained(func(bld *topo.Builder) ([]topo.Use, []topo.Use) {
		return []topo.Use{circleEdgeUse(bld, 0, 0, 10)}, []topo.Use{circleEdgeUse(bld, 0, 0, 20)}
	})
	if ok {
		t.Fatal("a concentric hole of radius 20 lies entirely outside the radius-10 outer and must be rejected")
	}
}

// TestHoleContainmentCurvedChordDiscrimination is the arc-discrimination regression the retired
// 64-sample count protected (#3478): a circular hole of radius 9.99 is strictly inside a radius-10 outer
// (margin 0.010), yet sits OUTSIDE the outer's inscribed 64-gon — whose chord sagitta is
// 10·(1−cos(π/64)) ≈ 0.0121, so the inscribed radius dips to ≈9.988. A coarse chord test could read the
// hole as poking out; the exact conic test accepts it, because 9.99 < 10 exactly.
func TestHoleContainmentCurvedChordDiscrimination(t *testing.T) {
	const outerR, holeR = 10.0, 9.99
	sagitta := outerR * (1 - math.Cos(math.Pi/64))
	if holeR <= outerR-sagitta || holeR >= outerR {
		t.Fatalf("premise broken: hole r=%.4f must sit between the 64-gon inscribed r=%.4f and the true r=%.1f",
			holeR, outerR-sagitta, outerR)
	}
	ok := planeFaceContained(func(bld *topo.Builder) ([]topo.Use, []topo.Use) {
		return []topo.Use{circleEdgeUse(bld, 0, 0, outerR)}, []topo.Use{circleEdgeUse(bld, 0, 0, holeR)}
	})
	if !ok {
		t.Fatalf("a hole circle r=%.2f is strictly inside the outer r=%.1f and must NOT be false-rejected", holeR, outerR)
	}
}

// TestHoleContainmentArcHoleInside: a D-shaped hole (a semicircular arc closed by its diameter) inside a
// circular outer exercises the exact arc-vs-curve path and must be contained.
func TestHoleContainmentArcHoleInside(t *testing.T) {
	ok := planeFaceContained(func(bld *topo.Builder) ([]topo.Use, []topo.Use) {
		arc, err := geom.NewArc3d(m.P3(0, 0, 0), m.V3(0, 0, 1), m.V3(1, 0, 0), 2, 0, math.Pi)
		if err != nil {
			t.Fatal(err)
		}
		start := bld.AddVertex(m.P3(2, 0, 0), containLineage)
		end := bld.AddVertex(m.P3(-2, 0, 0), containLineage)
		arcUse := topo.Fwd(bld.AddEdge(arc, start, end, containLineage))
		diameter := topo.Fwd(bld.AddEdge(geom.NewLineSegment(m.P3(-2, 0, 0), m.P3(2, 0, 0)), end, start, containLineage))
		return []topo.Use{circleEdgeUse(bld, 0, 0, 10)}, []topo.Use{arcUse, diameter}
	})
	if !ok {
		t.Fatal("a D-shaped hole of radius 2 inside a circular outer of radius 10 must be contained")
	}
}

// TestHoleContainmentSquareHoleInSquare: the polygon-only path stays correct — a small square hole is
// contained, and one straddling the outer wall is rejected.
func TestHoleContainmentSquareHoleInSquare(t *testing.T) {
	outer := func(bld *topo.Builder) []topo.Use {
		return polygonLoopUses(bld, []m.Point3{m.P3(-10, -10, 0), m.P3(10, -10, 0), m.P3(10, 10, 0), m.P3(-10, 10, 0)})
	}
	inside := planeFaceContained(func(bld *topo.Builder) ([]topo.Use, []topo.Use) {
		return outer(bld), polygonLoopUses(bld, []m.Point3{m.P3(-2, -2, 0), m.P3(2, -2, 0), m.P3(2, 2, 0), m.P3(-2, 2, 0)})
	})
	if !inside {
		t.Fatal("a 4×4 square hole centred in a 20×20 outer must be contained")
	}
	straddle := planeFaceContained(func(bld *topo.Builder) ([]topo.Use, []topo.Use) {
		return outer(bld), polygonLoopUses(bld, []m.Point3{m.P3(8, -2, 0), m.P3(18, -2, 0), m.P3(18, 2, 0), m.P3(8, 2, 0)})
	})
	if straddle {
		t.Fatal("a hole square spanning x∈[8,18] crosses the outer wall at x=10 and must be rejected")
	}
}

// The exact 2D curve-crossing dispatch behind hole containment (#3478). Every line/arc/circle pair has a
// closed form; only an ellipse pair (no closed form in the codebase) samples one operand.

// hitCount is the number of crossings curveCurve2dHits finds between two 2D curves.
func hitCount(a, b geom.Curve2) int { return len(curveCurve2dHits(a, b, 1e-9)) }

// TestCurve2dHitsSegmentAgainstCircleBothOrders: a segment through a circle's centre cuts it twice,
// whichever operand carries the segment.
func TestCurve2dHitsSegmentAgainstCircleBothOrders(t *testing.T) {
	seg := geom.NewLineSegment2d(m.P2(-5, 0), m.P2(5, 0))
	circ := geom.NewCircle2d(m.P2(0, 0), 2)
	if got := hitCount(seg, circ); got != 2 {
		t.Errorf("segment∩circle = %d hits, want 2", got)
	}
	if got := hitCount(circ, seg); got != 2 {
		t.Errorf("circle∩segment = %d hits, want 2 (operand order must not matter)", got)
	}
}

// TestCurve2dHitsInfiniteLineAgainstCircle: an unbounded line reaches the circle even when a segment of
// the same support would not.
func TestCurve2dHitsInfiniteLineAgainstCircle(t *testing.T) {
	ln, err := geom.NewLine2d(m.P2(-50, 1), m.V2(1, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	circ := geom.NewCircle2d(m.P2(0, 0), 2)
	if got := hitCount(ln, circ); got != 2 {
		t.Errorf("line∩circle = %d hits, want 2", got)
	}
	if got := hitCount(circ, ln); got != 2 {
		t.Errorf("circle∩line = %d hits, want 2 (operand order must not matter)", got)
	}
}

// TestCurve2dHitsArcKeepsOnlyItsSweep: an arc is intersected through its SUPPORT circle, so the hit on
// the half it does not cover is dropped — the discrimination a chorded arc would lose.
func TestCurve2dHitsArcKeepsOnlyItsSweep(t *testing.T) {
	upper := geom.NewArc2d(m.P2(0, 0), 2, 0, math.Pi) // the y ≥ 0 half of the r=2 circle
	seg := geom.NewLineSegment2d(m.P2(0, -5), m.P2(0, 5))
	hits := curveCurve2dHits(upper, seg, 1e-9)
	if len(hits) != 1 {
		t.Fatalf("arc∩segment = %d hits, want 1 (the (0,2) crossing only)", len(hits))
	}
	if !near(float64(hits[0].Y), 2, 1e-9) {
		t.Errorf("arc∩segment hit at y=%g, want the arc's own half at y=2", hits[0].Y)
	}
	if got := len(curveCurve2dHits(seg, upper, 1e-9)); got != 1 {
		t.Errorf("segment∩arc = %d hits, want 1 (operand order must not matter)", got)
	}
}

// TestCurve2dHitsCircleAgainstArc: the circle branch is exact for a curved second operand too.
func TestCurve2dHitsCircleAgainstArc(t *testing.T) {
	arc := geom.NewArc2d(m.P2(3, 0), 2, 0, math.Pi)
	circ := geom.NewCircle2d(m.P2(0, 0), 2)
	if got := hitCount(circ, arc); got != 1 {
		t.Errorf("circle∩arc = %d hits, want 1 (the arc covers only the upper crossing)", got)
	}
	if got := hitCount(arc, circ); got != 1 {
		t.Errorf("arc∩circle = %d hits, want 1 (operand order must not matter)", got)
	}
}

// TestCurve2dHitsEllipsePairFallback: two ellipses have no closed form here, so one is sampled. A
// planar line/arc/circle boundary never reaches this path; the crossing count still has to be right.
func TestCurve2dHitsEllipsePairFallback(t *testing.T) {
	wide, err := geom.NewEllipseFull2d(m.P2(0, 0), m.V2(1, 0), 4, 1)
	if err != nil {
		t.Fatalf("NewEllipseFull2d: %v", err)
	}
	tall, err := geom.NewEllipseFull2d(m.P2(0, 0), m.V2(0, 1), 4, 1)
	if err != nil {
		t.Fatalf("NewEllipseFull2d: %v", err)
	}
	if got := hitCount(wide, tall); got != 4 {
		t.Errorf("ellipse∩ellipse = %d hits, want 4 (two congruent ellipses at 90°)", got)
	}
}

// TestNearCurveEnd2dOnUnboundedLine: an infinite line has no endpoint to be near, so the endpoint guard
// reports false rather than sampling an infinite parameter.
func TestNearCurveEnd2dOnUnboundedLine(t *testing.T) {
	ln, err := geom.NewLine2d(m.P2(0, 0), m.V2(1, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	if nearCurveEnd2d(m.P2(0, 0), ln, 1e-6) {
		t.Error("nearCurveEnd2d reported an endpoint on an unbounded line")
	}
	seg := geom.NewLineSegment2d(m.P2(0, 0), m.P2(4, 0))
	if !nearCurveEnd2d(m.P2(4, 0), seg, 1e-9) {
		t.Error("nearCurveEnd2d missed a segment's own endpoint")
	}
}

// TestCurveMidParamOfUnboundedDomain: an unbounded domain has no finite midpoint, so the probe falls
// back to 0.5 instead of an infinity.
func TestCurveMidParamOfUnboundedDomain(t *testing.T) {
	ln, err := geom.NewLine2d(m.P2(0, 0), m.V2(1, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	if got := curveMidParam(ln); got != 0.5 {
		t.Errorf("curveMidParam(unbounded line) = %g, want 0.5", got)
	}
	if got := curveMidParam(geom.NewLineSegment2d(m.P2(0, 0), m.P2(4, 0))); got != 0.5 {
		t.Errorf("curveMidParam(segment) = %g, want 0.5", got)
	}
}

// TestRegionResolution2dHandlesUnboundedEdge: the tolerance scale sizes itself from an unbounded edge's
// [0,1] substitute domain, never from an infinite point.
func TestRegionResolution2dHandlesUnboundedEdge(t *testing.T) {
	ln, err := geom.NewLine2d(m.P2(0, 0), m.V2(1, 0))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	res := regionResolution2d([]geom.Curve2{ln, geom.NewLineSegment2d(m.P2(0, 0), m.P2(4, 0))})
	if !(res.Weld() > 0) || math.IsInf(res.Weld(), 0) {
		t.Errorf("regionResolution2d weld tolerance = %g, want a finite positive scale", res.Weld())
	}
}
