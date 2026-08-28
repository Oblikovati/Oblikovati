// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// TestProjectedCurveEntityPolyline is the #2158 follow-up: a projected reference curve must be
// hit-tested (selected) and offset like native geometry, and a closed projected perimeter (a face's
// outline) must report closed. A non-analytic projection is now a grounded reference Spline
// (ADR-0055 phase 3), so EntityPolyline reads it through the native Spline case.
func TestProjectedCurveEntityPolyline(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// A non-circular closed projection (an ellipse loop) becomes a closed reference spline; it must
	// still expose a polyline through EntityPolyline and report closed, so it is pickable/offsettable.
	loop := ellipsePts(4, 2, 0, 2*stdmath.Pi, 24)
	pc := s.addReferencePolyline(loop)
	got, closed := EntityPolyline(pc)
	if len(got) < 3 {
		t.Fatalf("EntityPolyline(closed reference spline) = %d points, want the sampled loop", len(got))
	}
	if !closed {
		t.Error("a projected closed perimeter must report closed (offset needs a loop)")
	}

	open := s.addReferencePolyline([]gmath.Point2{gmath.P2(0, 0), gmath.P2(2, 1)})
	pts, oClosed := EntityPolyline(open)
	if len(pts) < 2 {
		t.Fatalf("EntityPolyline(open reference spline) = %d points, want at least 2", len(pts))
	}
	if oClosed {
		t.Error("a projected open edge must report open")
	}
}
