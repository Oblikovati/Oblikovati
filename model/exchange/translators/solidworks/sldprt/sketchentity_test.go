// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"math"
	"testing"
)

// hasLine reports whether the segment (x1,y1)-(x2,y2) is present in either direction.
func hasLine(lines []Line, x1, y1, x2, y2, tol float64) bool {
	near := func(p Point, x, y float64) bool { return math.Abs(p.X-x) <= tol && math.Abs(p.Y-y) <= tol }
	for _, l := range lines {
		if (near(l.A, x1, y1) && near(l.B, x2, y2)) || (near(l.A, x2, y2) && near(l.B, x1, y1)) {
			return true
		}
	}
	return false
}

// TestEntityGraphRoundedRectangle is the headline case for the point-index entity graph: a rounded
// rectangle (4 lines + 4 fillet arcs) previously decoded through the geometry-first chainEntities
// guess (Exact=false). The graph names each entity's endpoints outright, so all 8 entities and every
// arc centre come out exact.
func TestEntityGraphRoundedRectangle(t *testing.T) {
	d, err := Open(readTestdata(t, "rrect_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sk := d.Sketches()
	if len(sk) != 1 {
		t.Fatalf("got %d sketches, want 1", len(sk))
	}
	s := sk[0]
	if !s.Exact {
		t.Error("rounded rectangle should decode exactly via the entity graph, not the chain guess")
	}
	if len(s.Lines) != 4 || len(s.Arcs) != 4 {
		t.Fatalf("got %d lines / %d arcs, want 4/4", len(s.Lines), len(s.Arcs))
	}
	// The four straight edges of a 40x20 mm rounded rectangle with 5 mm corner radii (metres).
	edges := [][4]float64{
		{-0.015, 0.010, 0.015, 0.010},   // top
		{0.020, 0.005, 0.020, -0.005},   // right
		{0.015, -0.010, -0.015, -0.010}, // bottom
		{-0.020, -0.005, -0.020, 0.005}, // left
	}
	for _, e := range edges {
		if !hasLine(s.Lines, e[0], e[1], e[2], e[3], 1e-9) {
			t.Errorf("missing edge %v; got %v", e, s.Lines)
		}
	}
	// Every fillet arc must have the 5 mm radius, with its centre equidistant from both endpoints.
	for i, a := range s.Arcs {
		if math.Abs(a.Radius-0.005) > 1e-9 {
			t.Errorf("arc %d radius = %v, want 0.005 m", i, a.Radius)
		}
		if math.Abs(dist(a.Center, a.Start)-dist(a.Center, a.End)) > 1e-9 {
			t.Errorf("arc %d centre %v is not equidistant from its endpoints", i, a.Center)
		}
	}
}

// TestEntityGraphOpenProfile checks the graph preserves true open topology: two disjoint segments,
// not a closed loop. The duplicate origin record occupies a point index, so this also covers the
// raw (un-deduplicated) index space.
func TestEntityGraphOpenProfile(t *testing.T) {
	d, err := Open(readTestdata(t, "twoline_open_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s := d.Sketches()[0]
	if len(s.Lines) != 2 {
		t.Fatalf("got %d lines, want 2 disjoint segments", len(s.Lines))
	}
	if !hasLine(s.Lines, 0, 0, 0.050, 0, 1e-9) || !hasLine(s.Lines, 0, 0.020, 0.030, 0.020, 1e-9) {
		t.Errorf("open segments wrong: %v", s.Lines)
	}
}

// TestEntityGraphCircles covers the closed-curve case (a == b): a circle's centre is cached
// immediately before its rim point.
func TestEntityGraphCircles(t *testing.T) {
	d, err := Open(readTestdata(t, "twocirc_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s := d.Sketches()[0]
	if len(s.Circles) != 2 {
		t.Fatalf("got %d circles, want 2", len(s.Circles))
	}
	want := map[float64][2]float64{0.004: {0, 0}, 0.006: {0.030, 0}}
	for _, c := range s.Circles {
		ctr, ok := want[math.Round(c.Radius*1e6)/1e6]
		if !ok {
			t.Errorf("unexpected circle radius %v", c.Radius)
			continue
		}
		if math.Abs(c.Center.X-ctr[0]) > 1e-9 || math.Abs(c.Center.Y-ctr[1]) > 1e-9 {
			t.Errorf("circle r=%v centre = %v, want %v", c.Radius, c.Center, ctr)
		}
	}
}

// TestEntityGraphReusedClass is the reason the graph exists: perno's sketches re-use their entity
// kinds, so only the first carries a class string. The graph types the later ones from the point
// indices alone — sketch 2 is two lines plus two circles.
func TestEntityGraphReusedClass(t *testing.T) {
	d, err := Open(readTestdata(t, "perno_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sk := d.Sketches()
	if len(sk) < 3 {
		t.Fatalf("got %d sketches, want at least 3", len(sk))
	}
	s := sk[2]
	if len(s.Lines) != 2 || len(s.Circles) != 2 {
		t.Fatalf("re-used-class sketch decoded %d lines / %d circles, want 2/2", len(s.Lines), len(s.Circles))
	}
	if !s.Exact {
		t.Error("entity-graph decode must be exact")
	}
}

// TestKindsFromGaps checks the record-size chain: each flag-to-flag gap names both entities' kinds,
// the chain must agree at every overlap, and an unknown gap (e.g. an ellipse record) is rejected.
func TestKindsFromGaps(t *testing.T) {
	rec := func(offs ...int) []entityIndexRecord {
		out := make([]entityIndexRecord, len(offs))
		for i, o := range offs {
			out[i] = entityIndexRecord{off: o}
		}
		return out
	}
	// rrect: four lines then four arcs -> 92,92,92,104,112,112,112.
	got, ok := kindsFromGaps(rec(0, 92, 184, 276, 380, 492, 604, 716))
	if !ok {
		t.Fatal("rrect gap chain rejected")
	}
	want := []bool{false, false, false, false, true, true, true, true}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entity %d isArc = %v, want %v", i, got[i], want[i])
		}
	}
	if _, ok := kindsFromGaps(rec(0, 77)); ok {
		t.Error("an unknown record size must be rejected, not guessed")
	}
}
