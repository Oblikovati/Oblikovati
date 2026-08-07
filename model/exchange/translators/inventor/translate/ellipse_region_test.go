// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/model/exchange/translators/inventor/ipt"
)

// onEllipse reports how far a point strays from the ellipse's rim (0 = exactly on it), using the
// implicit form in the ellipse's own frame: (x'/a)^2 + (y'/b)^2 = 1.
func onEllipse(e ipt.EllipseArc, p ipt.Point2D) float64 {
	ux, uy := e.MajorAxis.X, e.MajorAxis.Y
	dx, dy := p.X-e.Center.X, p.Y-e.Center.Y
	x := dx*ux + dy*uy
	y := -dx*uy + dy*ux
	return stdmath.Abs((x/e.MajorR)*(x/e.MajorR) + (y/e.MinorR)*(y/e.MinorR) - 1)
}

// TestEllipseArcPolylineWholeOnEllipse checks the whole-ellipse sampling (Start==End): every sample
// lands on the rim, for a rotated ellipse so the frame math is exercised, not just an axis-aligned one.
func TestEllipseArcPolylineWholeOnEllipse(t *testing.T) {
	// major axis at 30°, a=4, b=2, centred at (5,-3).
	c, s := stdmath.Cos(stdmath.Pi/6), stdmath.Sin(stdmath.Pi/6)
	e := ipt.EllipseArc{Center: ipt.Point2D{X: 5, Y: -3}, MajorAxis: ipt.Point2D{X: c, Y: s}, MajorR: 4, MinorR: 2}
	poly := ellipseArcPolyline(e)
	if len(poly) != arcSegments {
		t.Fatalf("whole ellipse sampled %d points, want %d", len(poly), arcSegments)
	}
	for i, q := range poly {
		if d := onEllipse(e, ipt.Point2D{X: float64(q.X), Y: float64(q.Y)}); d > 1e-9 {
			t.Errorf("sample %d strays from rim by %.2e", i, d)
		}
	}
}

// TestEllipseArcPolylineArcEndpoints checks a trimmed span: the polyline starts at Start, ends at
// End, and every intermediate point lies on the rim.
func TestEllipseArcPolylineArcEndpoints(t *testing.T) {
	e := ipt.EllipseArc{Center: ipt.Point2D{}, MajorAxis: ipt.Point2D{X: 1}, MajorR: 3, MinorR: 1}
	// Start at t=0 -> (3,0); End at t=pi/2 -> (0,1). Points ON the ellipse.
	e.Start = ipt.Point2D{X: 3, Y: 0}
	e.End = ipt.Point2D{X: 0, Y: 1}
	poly := ellipseArcPolyline(e)
	if len(poly) < 2 {
		t.Fatalf("arc sampled %d points", len(poly))
	}
	first, lastP := poly[0], poly[len(poly)-1]
	if stdmath.Hypot(float64(first.X)-3, float64(first.Y)-0) > 1e-9 {
		t.Errorf("arc start = (%.3f,%.3f), want (3,0)", first.X, first.Y)
	}
	if stdmath.Hypot(float64(lastP.X)-0, float64(lastP.Y)-1) > 1e-9 {
		t.Errorf("arc end = (%.3f,%.3f), want (0,1)", lastP.X, lastP.Y)
	}
	for i, q := range poly {
		if d := onEllipse(e, ipt.Point2D{X: float64(q.X), Y: float64(q.Y)}); d > 1e-9 {
			t.Errorf("sample %d strays from rim by %.2e", i, d)
		}
	}
	// The quarter-arc bows AWAY from the chord's midpoint through (>0,>0): a midpoint sample must have
	// both coords positive (the short CCW span 0->pi/2), proving direction, not the long way round.
	mid := poly[len(poly)/2]
	if mid.X <= 0 || mid.Y <= 0 {
		t.Errorf("arc midpoint (%.3f,%.3f) not in the +,+ quadrant — walked the wrong way", mid.X, mid.Y)
	}
}
