// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

// TestProjectedCurveEntityPolyline is the #2158 follow-up: a projected reference curve must expose
// its sampled polyline through EntityPolyline so it can be hit-tested (selected) and offset, and a
// closed projected perimeter (a face's outline) must report closed. Before the fix a ProjectedCurve
// fell through to an endpoint chord that was empty, so it could not be picked at all.
func TestProjectedCurveEntityPolyline(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	square := []gmath.Point2{gmath.P2(0, 0), gmath.P2(4, 0), gmath.P2(4, 4), gmath.P2(0, 4), gmath.P2(0, 0)}
	pc := s.RestoreProjectedCurve(nextID(), square, "edge", "E1")
	got, closed := EntityPolyline(pc)
	if len(got) != len(square) {
		t.Fatalf("EntityPolyline(closed ProjectedCurve) = %d points, want %d", len(got), len(square))
	}
	if !closed {
		t.Error("a projected closed perimeter must report closed (offset needs a loop)")
	}

	open := s.RestoreProjectedCurve(nextID(), []gmath.Point2{gmath.P2(0, 0), gmath.P2(2, 1)}, "edge", "E2")
	pts, oClosed := EntityPolyline(open)
	if len(pts) != 2 {
		t.Fatalf("EntityPolyline(open ProjectedCurve) = %d points, want 2", len(pts))
	}
	if oClosed {
		t.Error("a projected open edge must report open")
	}
}
