// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// TestAddSplineLengthMeasuresPolyline checks the dimension reads the spline's
// representative-polyline arc length — for collinear fit points that length is exact.
func TestAddSplineLengthMeasuresPolyline(t *testing.T) {
	s := NewSketches3D().Add()
	sp := s.AddSpline3D([]gmath.Point3{
		{X: 0, Y: 0, Z: 0}, {X: 1, Y: 0, Z: 0}, {X: 2, Y: 0, Z: 0},
	}, false, true)
	d, err := s.DimensionConstraints3D().AddSplineLength(sp, "2 cm")
	if err != nil {
		t.Fatalf("AddSplineLength: %v", err)
	}
	if got := d.Measured(); stdmath.Abs(got-2) > 1e-9 {
		t.Errorf("measured = %v, want 2 (straight spline)", got)
	}
	if d.KindName() != "splineLength" {
		t.Errorf("kind = %q, want splineLength", d.KindName())
	}
}

// TestSplineLengthDrivesGeometry checks the driving dimension stretches the spline to
// the target length on solve.
func TestSplineLengthDrivesGeometry(t *testing.T) {
	s := NewSketches3D().Add()
	sp := s.AddSpline3D([]gmath.Point3{
		{X: 0, Y: 0, Z: 0}, {X: 1, Y: 0, Z: 0}, {X: 2, Y: 0, Z: 0},
	}, false, true)
	d, err := s.DimensionConstraints3D().AddSplineLength(sp, "3 cm")
	if err != nil {
		t.Fatalf("AddSplineLength: %v", err)
	}
	// Anchor one end so the solver stretches rather than translates.
	s.GeometricConstraints3D().add(NewGround3D(sp.Points[0]))
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve did not converge: %+v", res)
	}
	if got := d.Measured(); stdmath.Abs(got-3) > 1e-6 {
		t.Errorf("length after solve = %v, want 3", got)
	}
}

// TestAddSplineLengthRejectsDegenerateSpline guards the input validation.
func TestAddSplineLengthRejectsDegenerateSpline(t *testing.T) {
	s := NewSketches3D().Add()
	sp := &Spline3D{entityBase: newEntity(), Points: nil, fit: true}
	if _, err := s.DimensionConstraints3D().AddSplineLength(sp, "1 cm"); err == nil {
		t.Error("expected an error for a spline with no points")
	}
}

// TestSplineLengthRoundTrip checks the dimension survives marshal→apply.
func TestSplineLengthRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	sp := s.AddSpline3D([]gmath.Point3{
		{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0}, {X: 2, Y: 0, Z: 1},
	}, false, true)
	if _, err := s.DimensionConstraints3D().AddSplineLength(sp, "5 cm"); err != nil {
		t.Fatalf("AddSplineLength: %v", err)
	}

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	dims := dst.Item(0).DimensionConstraints3D()
	if dims.Count() != 1 || dims.Item(0).KindName() != "splineLength" {
		t.Fatalf("restored dimensions = %d (%q), want one splineLength", dims.Count(), dims.Item(0).KindName())
	}
}
