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
