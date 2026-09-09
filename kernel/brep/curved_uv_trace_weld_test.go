// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The welder and the boundary-edge filter have to answer "is this one point?" with ONE tolerance. They
// did not: the welder merged within seamWeldGrid and the filter dropped only within arrTol, a hundred
// times tighter, so a step between the two produced an edge running from a vertex to ITSELF. Such an
// edge bounds nothing; left in, it either cancels against a twin and merges two regions, or chains as a
// one-edge loop and emits a phantom face (ADR-0061, the axis-parallel figure-eight).
func TestKeptBoundaryDropsAnEdgeFromAVertexToItself(t *testing.T) {
	t.Parallel()
	// A unit square whose bottom-left corner carries a step a tenth of the weld grid: the two ends of
	// that step are one vertex.
	step := seamWeldGrid / 10
	cell := Face2D{Outer: []math.Point2{
		math.P2(0, 0), math.P2(step, 0), math.P2(1, 0), math.P2(1, 1), math.P2(0, 1),
	}}
	for _, e := range keptBoundaryEdges([]Face2D{cell}, false, false) {
		if e.from == e.to {
			t.Errorf("edge from vertex %d to itself survived: (%v)->(%v)", e.from, e.a, e.b)
		}
	}
}

// The counterpart: a FULL-WRAP edge — an uncut rim or section circle whose two ends are the azimuth
// seam — also welds to one vertex, and it is the one edge of its own closed loop. Dropping it would
// delete a whole boundary, so the filter must separate the two by the edge's (u,v) span, not by the
// welded indices alone.
func TestKeptBoundaryKeepsAFullWrapEdge(t *testing.T) {
	t.Parallel()
	twoPi := 2 * stdmath.Pi
	// A band between two full-turn rims: every edge runs the whole azimuth, both ends on the seam.
	cell := Face2D{Outer: []math.Point2{
		math.P2(0, 0), math.P2(twoPi, 0), math.P2(twoPi, 1), math.P2(0, 1),
	}}
	full := 0
	for _, e := range keptBoundaryEdges([]Face2D{cell}, true, false) {
		if e.from == e.to {
			full++
		}
	}
	if full != 2 {
		t.Errorf("full-wrap edges kept = %d, want 2 (the two rims)", full)
	}
}
