// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestChainLoopsSeparatesBowtieAtDegree4Vertex pins the angular-next-edge tracer (#1403, Steinmetz
// fold-in). Two triangles meeting at one shared vertex (a bow-tie) have a degree-4 boundary vertex; the
// kept-region boundary must trace as TWO separate triangles, not one merged figure-eight. The edges are
// fed in the order that makes a naive "first unused outgoing" walk MERGE them (the shared vertex's lower
// lobe edge precedes its upper lobe edge), so only an angular turn rule — pick the outgoing half-edge
// first in clockwise order from the reversed arrival, keeping material on the left — separates them.
func TestChainLoopsSeparatesBowtieAtDegree4Vertex(t *testing.T) {
	// Vertically opposite triangles sharing the origin (index 0). Top interior y>0, bottom interior y<0,
	// each wound CCW so material is on the left of every directed edge.
	o := math.P2(0, 0)
	t1, t2 := math.P2(-1, 2), math.P2(1, 2)
	b1, b2 := math.P2(1, -2), math.P2(-1, -2)
	// Indices: O=0, T1=1, T2=2, B1=3, B2=4. Bottom lobe's outgoing (0→B2) is listed BEFORE the top lobe's
	// outgoing (0→T2) so a naive index-order walk would cross from the top lobe into the bottom one.
	edges := []dedge{
		{from: 2, to: 1, a: t2, b: t1}, // T2→T1 (top)
		{from: 1, to: 0, a: t1, b: o},  // T1→O  (top, arrives at the shared vertex)
		{from: 0, to: 4, a: o, b: b2},  // O→B2  (bottom outgoing — naive would grab this first)
		{from: 4, to: 3, a: b2, b: b1}, // B2→B1 (bottom)
		{from: 3, to: 0, a: b1, b: o},  // B1→O  (bottom, arrives at the shared vertex)
		{from: 0, to: 2, a: o, b: t2},  // O→T2  (top outgoing)
	}
	loops := chainLoops(edges)
	if len(loops) != 2 {
		t.Fatalf("chainLoops merged the bow-tie into %d loop(s); want 2 separate triangles", len(loops))
	}
	for i, lp := range loops {
		if len(lp) != 3 {
			t.Errorf("loop %d has %d edges; want 3 (a triangle, not a crossed-through merge)", i, len(lp))
		}
	}
}
