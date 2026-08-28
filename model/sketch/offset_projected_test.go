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

// TestOffsetProjectedSplineIsUnsupported: a non-analytic projection is a grounded reference Spline
// (ADR-0055 phase 3), and OffsetEntity offsets only analytic line/circle/arc geometry — so the old
// faceted-polyline offset (whose broken-up segments the architecture change deliberately removed) is
// no longer produced. Analytic projections (circle/arc) still offset cleanly
// (TestProjectedCircleOffsetsAnalytic / TestProjectedArcOffsetsAnalytic).
func TestOffsetProjectedSplineIsUnsupported(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pc := s.addReferencePolyline(ellipsePts(4, 2, 0, 2*stdmath.Pi, 24))
	if _, ok := pc.(*Spline); !ok {
		t.Fatalf("a non-analytic projection should be a reference *Spline, got %T", pc)
	}
	if _, err := s.OffsetEntity(pc, -0.5); err == nil {
		t.Fatal("offsetting a reference spline should be unsupported (analytic projections offset, splines do not)")
	}
}
