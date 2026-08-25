// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// TestProjectedCurveEntityPolyline is the #2158 follow-up: a projected reference curve must expose
// its sampled polyline through EntityPolyline so it can be hit-tested (selected) and offset, and a
// closed projected perimeter (a face's outline) must report closed. Before the fix a ProjectedCurve
// fell through to an endpoint chord that was empty, so it could not be picked at all.
func TestProjectedCurveEntityPolyline(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// A non-circular closed projection (an ellipse) stays a polyline; it must still expose that
	// polyline through EntityPolyline and report closed, so it is pickable and offsettable.
	loop := ellipsePts(4, 2, 0, 2*stdmath.Pi, 24)
	pc := s.RestoreProjectedCurve(nextID(), loop, "edge", "E1")
	got, closed := EntityPolyline(pc)
	if len(got) < 3 {
		t.Fatalf("EntityPolyline(closed ProjectedCurve) = %d points, want the sampled loop", len(got))
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
