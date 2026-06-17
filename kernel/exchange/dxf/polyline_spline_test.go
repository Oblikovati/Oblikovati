// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"testing"

	"oblikovati.org/kernel/exchange/drawing"
)

// TestLwPolylineRoundTrip checks a closed polyline with an elevation and a single bulged
// segment survives Encode→Decode, and that the bulge stays attached to its vertex.
func TestLwPolylineRoundTrip(t *testing.T) {
	in := &drawing.LwPolyline{
		Closed: true, Elevation: 2,
		Points: [][2]float64{{0, 0}, {2, 0}, {2, 2}},
		Bulges: []float64{0, 0.5, 0},
	}
	dr := reEncode(t, &drawing.Drawing{Entities: []drawing.Entity{in}})
	p := dr.Entities[0].(*drawing.LwPolyline)
	if !p.Closed || len(p.Points) != 3 || !near(p.Elevation, 2) {
		t.Fatalf("polyline header: closed=%v pts=%d elev=%g", p.Closed, len(p.Points), p.Elevation)
	}
	if p.Points[2] != [2]float64{2, 2} {
		t.Errorf("vertex 2 = %v", p.Points[2])
	}
	if !near(bulge(p.Bulges, 1), 0.5) || bulge(p.Bulges, 0) != 0 || bulge(p.Bulges, 2) != 0 {
		t.Errorf("bulges = %v, want only vertex 1 = 0.5", p.Bulges)
	}
}

// TestSplineControlPointsRoundTrip checks a control-point spline round-trips and that the
// encoder synthesises a clamped knot vector when the model carried none.
func TestSplineControlPointsRoundTrip(t *testing.T) {
	want := [][3]float64{{0, 0, 0}, {1, 2, 0}, {3, 2, 0}, {4, 0, 0}}
	in := &drawing.Spline{Degree: 3, ControlPoints: want}
	dr := reEncode(t, &drawing.Drawing{Entities: []drawing.Entity{in}})
	s := dr.Entities[0].(*drawing.Spline)
	if s.Degree != 3 || len(s.ControlPoints) != 4 {
		t.Fatalf("degree=%d ctrl=%d", s.Degree, len(s.ControlPoints))
	}
	for i := range want {
		if s.ControlPoints[i] != want[i] {
			t.Errorf("ctrl[%d] = %v, want %v", i, s.ControlPoints[i], want[i])
		}
	}
	if len(s.Knots) != 4+3+1 {
		t.Errorf("knots = %d, want %d (n+degree+1)", len(s.Knots), 4+3+1)
	}
}

// TestSplineRationalWeights checks a rational spline's per-control-point weights round-trip.
func TestSplineRationalWeights(t *testing.T) {
	in := &drawing.Spline{
		Degree: 2, Rational: true,
		ControlPoints: [][3]float64{{0, 0, 0}, {1, 1, 0}, {2, 0, 0}},
		Weights:       []float64{1, 2, 1},
	}
	dr := reEncode(t, &drawing.Drawing{Entities: []drawing.Entity{in}})
	s := dr.Entities[0].(*drawing.Spline)
	if !s.Rational || len(s.Weights) != 3 || !near(s.Weights[1], 2) {
		t.Errorf("rational=%v weights=%v", s.Rational, s.Weights)
	}
}

// TestSplineFitPoints checks a fit-point spline round-trips its fit points.
func TestSplineFitPoints(t *testing.T) {
	in := &drawing.Spline{Degree: 3, FitPoints: [][3]float64{{0, 0, 0}, {1, 1, 0}, {2, 0, 0}}}
	dr := reEncode(t, &drawing.Drawing{Entities: []drawing.Entity{in}})
	s := dr.Entities[0].(*drawing.Spline)
	if len(s.FitPoints) != 3 || s.FitPoints[1] != [3]float64{1, 1, 0} {
		t.Errorf("fit points = %v", s.FitPoints)
	}
	if len(s.ControlPoints) != 0 {
		t.Errorf("fit-only spline gained %d control points", len(s.ControlPoints))
	}
}

// TestClampedKnots checks the synthesised knot vector is clamped (repeated ends) and the
// right length.
func TestClampedKnots(t *testing.T) {
	k := clampedKnots(5, 3) // n=5, degree=3 → 9 knots, one interior at 0.5
	want := []float64{0, 0, 0, 0, 0.5, 1, 1, 1, 1}
	if len(k) != len(want) {
		t.Fatalf("len = %d, want %d", len(k), len(want))
	}
	for i := range want {
		if !near(k[i], want[i]) {
			t.Errorf("knot[%d] = %g, want %g", i, k[i], want[i])
		}
	}
}

// bulge returns the bulge at index i, or 0.
func bulge(bulges []float64, i int) float64 {
	if i < len(bulges) {
		return bulges[i]
	}
	return 0
}
