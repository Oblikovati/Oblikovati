// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"math"
	"testing"
)

// TestSplineDecode checks a fit-point spline decodes to its four interpolation points in order and is
// recognised as open (the generated spline runs through four distinct points).
func TestSplineDecode(t *testing.T) {
	d, err := Open(readTestdata(t, "spline_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sp := d.Sketches()[0].Splines
	if len(sp) != 1 {
		t.Fatalf("got %d splines, want 1", len(sp))
	}
	want := []Point{{0, 0}, {0.02, 0.03}, {0.04, 0.01}, {0.06, 0.04}}
	if sp[0].Closed {
		t.Error("spline reported closed, want open")
	}
	if len(sp[0].FitPoints) != len(want) {
		t.Fatalf("got %d fit points %v, want %d", len(sp[0].FitPoints), sp[0].FitPoints, len(want))
	}
	for i, w := range want {
		if math.Abs(sp[0].FitPoints[i].X-w.X) > 1e-9 || math.Abs(sp[0].FitPoints[i].Y-w.Y) > 1e-9 {
			t.Errorf("fit point %d = %+v, want %+v", i, sp[0].FitPoints[i], w)
		}
	}
}

// TestSplineFromPointsClosed checks that a repeated first/last point marks the spline closed and the
// duplicate is dropped.
func TestSplineFromPointsClosed(t *testing.T) {
	ordered := []Point{{0, 0}, {1, 0}, {1, 1}, {0, 0}}
	s, ok := splineFromPoints(ordered)
	if !ok || !s.Closed || len(s.FitPoints) != 3 {
		t.Errorf("splineFromPoints = (%+v,%v), want closed 3-point spline", s, ok)
	}
}
