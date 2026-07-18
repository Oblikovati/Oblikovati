// SPDX-License-Identifier: GPL-2.0-only

package ipt

import "testing"

// ip builds a point with incidence references (the curve ids through it).
func ip(x, y float64, refs ...uint32) incPoint { return incPoint{p: Point2D{x, y}, inc: refs} }

// TestLineProfilesDecodesRevolveProfile is the byte-level integration: the corpus revolve's
// PmDCSegment must reconstruct a closed rectangular profile (1..3 × 0.5..1.5) plus its separate
// axis line, straight from the geometry points' incidence lists — no rank-alignment.
func TestLineProfilesDecodesRevolveProfile(t *testing.T) {
	d := openDoc(t, "16_revolve.ipt")
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		t.Fatal("no PmDCSegment")
	}
	profiles := LineProfiles(seg)
	var loop *Sketch
	for i := range profiles {
		if len(profiles[i].Lines) == 4 && isClosedRing(profiles[i].Lines) {
			loop = &profiles[i]
		}
	}
	if loop == nil {
		t.Fatalf("no closed 4-line profile; got %+v", profiles)
	}
	minX, maxX, minY, maxY := bbox(loop.Lines)
	if absf(minX-1) > 1e-4 || absf(maxX-3) > 1e-4 || absf(minY-0.5) > 1e-4 || absf(maxY-1.5) > 1e-4 {
		t.Errorf("profile bbox = [%.3f,%.3f]x[%.3f,%.3f], want [1,3]x[0.5,1.5]", minX, maxX, minY, maxY)
	}
}

// TestCleanEdgesPairsSharedRefs: a unit square whose four corners each list their two incident
// edge ids resolves to exactly four edges, one per reference shared by two corners.
func TestCleanEdgesPairsSharedRefs(t *testing.T) {
	pts := []incPoint{ip(0, 0, 10, 13), ip(1, 0, 10, 11), ip(1, 1, 11, 12), ip(0, 1, 12, 13)}
	if e := cleanEdges(pts); len(e) != 4 {
		t.Errorf("cleanEdges = %d edges, want 4", len(e))
	}
	sk := componentSketches(pts, cleanEdges(pts))
	if len(sk) != 1 || len(sk[0].Lines) != 4 || !isClosedRing(sk[0].Lines) {
		t.Errorf("componentSketches did not rebuild one closed square: %+v", sk)
	}
}

// TestCollinearEdgesRecoversHiddenSegment locks degree-completion: three collinear points sharing
// a vertical-constraint reference (a stepped shaft's x-run) hide the edge between the two that are
// otherwise short of degree two; collinearEdges recovers exactly that segment and no other.
func TestCollinearEdgesRecoversHiddenSegment(t *testing.T) {
	pts := []incPoint{
		ip(0, 0, 20, 30),     // A: constraint 20 + clean edge 30
		ip(0, 1, 20, 31),     // B: constraint 20 + clean edge 31
		ip(0, 2, 20, 32, 33), // C: constraint 20 + two clean edges (already degree 2)
		ip(1, 0, 30), ip(1, 1, 31), ip(1, 2, 32), ip(-1, 2, 33),
	}
	clean := cleanEdges(pts)
	add := collinearEdges(pts, clean)
	if len(add) != 1 {
		t.Fatalf("collinearEdges = %d edges, want 1 (the hidden A-B step): %+v", len(add), add)
	}
	got := add[0]
	if !((got.a == 0 && got.b == 1) || (got.a == 1 && got.b == 0)) {
		t.Errorf("recovered edge = %+v, want A(0)-B(1)", got)
	}
}

// TestCollinearEdgesIgnoresNonAxisGroup: a shared reference among points that are NOT axis-aligned
// (an arc's construction points) must not be completed into a straight chord.
func TestCollinearEdgesIgnoresNonAxisGroup(t *testing.T) {
	pts := []incPoint{ip(0, 0, 20), ip(1, 1, 20), ip(2, 0, 20)} // ref 20 shared, not collinear
	if add := collinearEdges(pts, nil); len(add) != 0 {
		t.Errorf("collinearEdges invented %d edges on a non-axis group, want 0", len(add))
	}
}

// TestPruneLeavesDropsDanglingEdge locks the fix for a diameter-dimension leaf: a square with a
// dangling edge to an off-ring witness point (and a SECOND, coincident node at one corner, as the
// real files carry) must prune to the clean four-edge ring — the degree that decides a leaf is
// counted per coincident vertex, not per raw point index.
func TestPruneLeavesDropsDanglingEdge(t *testing.T) {
	pts := []incPoint{
		ip(0, 0), ip(1, 0), ip(1, 1), ip(0, 1), // square corners 0..3
		ip(1, 1), // 4: a second node coincident with corner 2 (the dimension's anchor)
		ip(2, 1), // 5: the witness point (off-ring leaf)
	}
	edges := []edge{{0, 1}, {1, 2}, {2, 3}, {3, 0}, {4, 5}} // square + leaf hanging off the coincident corner
	sk := componentSketches(pts, edges)
	if len(sk) != 1 {
		t.Fatalf("want 1 component, got %d", len(sk))
	}
	if len(sk[0].Lines) != 4 || !isClosedRing(sk[0].Lines) {
		t.Errorf("leaf not pruned: got %d lines %+v", len(sk[0].Lines), sk[0].Lines)
	}
}

// TestComponentSketchesSeparatesDisjointLoops: two squares that share no reference and no vertex
// decode as two independent sketches (the connected-component = sketch rule that replaces the
// 800-byte clustering for line profiles).
func TestComponentSketchesSeparatesDisjointLoops(t *testing.T) {
	pts := []incPoint{
		ip(0, 0, 10, 13), ip(1, 0, 10, 11), ip(1, 1, 11, 12), ip(0, 1, 12, 13),
		ip(5, 5, 20, 23), ip(6, 5, 20, 21), ip(6, 6, 21, 22), ip(5, 6, 22, 23),
	}
	sk := componentSketches(pts, cleanEdges(pts))
	if len(sk) != 2 {
		t.Fatalf("want 2 disjoint sketches, got %d", len(sk))
	}
	for _, s := range sk {
		if len(s.Lines) != 4 || !isClosedRing(s.Lines) {
			t.Errorf("component is not a closed square: %+v", s.Lines)
		}
	}
}

// bbox returns the axis-aligned bounds of a set of lines.
func bbox(lines []Line) (minX, maxX, minY, maxY float64) {
	minX, minY = 1e18, 1e18
	maxX, maxY = -1e18, -1e18
	for _, l := range lines {
		for _, p := range [2]Point2D{l.A, l.B} {
			minX, maxX = minf(minX, p.X), maxf(maxX, p.X)
			minY, maxY = minf(minY, p.Y), maxf(maxY, p.Y)
		}
	}
	return
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
