// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// ellipsePts samples an ellipse (semi-axes ra, rb) — an obliquely projected arc/circle, which has no
// analytic geom.Curve2 yet (ADR-0055 phase 2), so a restored-from-points projection of it offsets as
// a polyline.
func ellipsePts(ra, rb, start, sweep float64, n int) []gmath.Point2 {
	pts := make([]gmath.Point2, n+1)
	for i := range pts {
		a := start + sweep*float64(i)/float64(n)
		pts[i] = gmath.P2(gmath.Scalar(ra*stdmath.Cos(a)), gmath.Scalar(rb*stdmath.Sin(a)))
	}
	return pts
}

// TestOffsetProjectedClosedLoop: a non-circular closed projection (an ellipse) offsets as a closed
// loop of lines — the polyline fallback (#2158 follow-up). A circular projection takes the analytic
// path instead (TestProjectedCircleOffsetsAnalytic).
func TestOffsetProjectedClosedLoop(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pc := s.RestoreProjectedCurve(nextID(), ellipsePts(4, 2, 0, 2*stdmath.Pi, 24), "edge", "E1")
	if _, ok := pc.AnalyticCurve(); ok {
		t.Fatal("an ellipse polyline should have no analytic curve (it offsets as a polyline)")
	}
	before := s.Lines().Count()
	got, err := s.OffsetEntity(pc, -0.5)
	if err != nil {
		t.Fatalf("OffsetEntity(closed ellipse projection): %v", err)
	}
	if got == nil {
		t.Fatal("offset returned a nil entity")
	}
	if s.Lines().Count() <= before {
		t.Errorf("no offset line geometry created (%d→%d lines)", before, s.Lines().Count())
	}
}

// TestOffsetProjectedOpenCurve: a non-circular open projection offsets as an open chain of lines.
func TestOffsetProjectedOpenCurve(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pc := s.RestoreProjectedCurve(nextID(), ellipsePts(4, 2, 0, stdmath.Pi, 16), "edge", "E2")
	if _, ok := pc.AnalyticCurve(); ok {
		t.Fatal("a half-ellipse polyline should have no analytic curve (it offsets as a polyline)")
	}
	before := s.Lines().Count()
	if _, err := s.OffsetEntity(pc, 0.5); err != nil {
		t.Fatalf("OffsetEntity(open ellipse projection): %v", err)
	}
	if s.Lines().Count() <= before {
		t.Error("no offset line geometry created for the open projection")
	}
}
