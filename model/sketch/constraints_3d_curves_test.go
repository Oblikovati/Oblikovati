// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

func TestSplineFitPoints3DBindsNearestAndSolves(t *testing.T) {
	s := NewSketches3D().Add()
	sp := s.AddSpline3D([]gmath.Point3{
		{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0}, {X: 2, Y: 0, Z: 1},
	}, false, true)
	p := s.AddPoint3D(gmath.P3(1.1, 0.9, 0.1)) // nearest fit point: index 1
	c, err := NewSplineFitPoints3D(sp, p)
	if err != nil {
		t.Fatalf("NewSplineFitPoints3D: %v", err)
	}
	if c.FitIndex != 1 {
		t.Fatalf("FitIndex = %d, want 1 (nearest)", c.FitIndex)
	}
	s.GeometricConstraints3D().add(c)
	solved3D(t, s)
	if d := float64(sp.Points[1].Position().DistanceTo(p.Position())); d > 1e-6 {
		t.Errorf("point %g from fit point after solve, want attached", d)
	}
}

func TestSplineFitPoints3DRejectsControlSpline(t *testing.T) {
	s := NewSketches3D().Add()
	control := s.AddSpline3D([]gmath.Point3{
		{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0}, {X: 2, Y: 0, Z: 0},
	}, false, false)
	p := s.AddPoint3D(gmath.P3(0, 0, 0))
	if _, err := NewSplineFitPoints3D(control, p); err == nil {
		t.Error("expected an error for a control (non-interpolating) spline")
	}
}

func TestHelical3DTiesHelixToCircleAndSolves(t *testing.T) {
	s := NewSketches3D().Add()
	axis, err := gmath.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("axis: %v", err)
	}
	circle := s.AddCircle3D(gmath.P3(0, 0, 0), axis, 5)
	helix := s.AddHelix3D(gmath.P3(0.5, -0.2, 0.3), axis, 4.2, 1, 0, 8, false)
	c, err := NewHelical3D(helix, circle)
	if err != nil {
		t.Fatalf("NewHelical3D: %v", err)
	}
	s.GeometricConstraints3D().add(c)
	solved3D(t, s)
	if d := float64(helix.Origin.Position().DistanceTo(circle.Center.Position())); d > 1e-6 {
		t.Errorf("helix origin %g from circle center after solve, want coincident", d)
	}
	if dr := float64(helix.StartRadius - circle.Radius); dr > 1e-6 || dr < -1e-6 {
		t.Errorf("start radius differs from circle radius by %g after solve", dr)
	}
}

func TestHelical3DRejectsSkewAxes(t *testing.T) {
	s := NewSketches3D().Add()
	zAxis, _ := gmath.NewUnitVector3(0, 0, 1)
	xAxis, _ := gmath.NewUnitVector3(1, 0, 0)
	circle := s.AddCircle3D(gmath.P3(0, 0, 0), zAxis, 5)
	helix := s.AddHelix3D(gmath.P3(0, 0, 0), xAxis, 5, 1, 0, 3, false)
	if _, err := NewHelical3D(helix, circle); err == nil {
		t.Error("expected an error for a helix axis not parallel to the circle axis")
	}
}

// TestCurveConstraints3DRoundTrip checks splineFitPoints (with its fit index) and
// helical survive marshal→apply.
func TestCurveConstraints3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	axis, _ := gmath.NewUnitVector3(0, 0, 1)
	sp := s.AddSpline3D([]gmath.Point3{
		{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0}, {X: 2, Y: 0, Z: 1},
	}, false, true)
	p := s.AddPoint3D(gmath.P3(2, 0, 1))
	fit, err := NewSplineFitPoints3D(sp, p)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	circle := s.AddCircle3D(gmath.P3(0, 0, 4), axis, 3)
	helix := s.AddHelix3D(gmath.P3(0, 0, 4), axis, 3, 1, 0, 5, false)
	hel, err := NewHelical3D(helix, circle)
	if err != nil {
		t.Fatalf("helical: %v", err)
	}
	s.GeometricConstraints3D().add(fit)
	s.GeometricConstraints3D().add(hel)

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.Item(0).GeometricConstraints3D()
	if got.Count() != 2 {
		t.Fatalf("restored constraints = %d, want 2", got.Count())
	}
	rfit, ok := got.Item(0).(*SplineFitPoints3D)
	if !ok || rfit.FitIndex != 2 {
		t.Fatalf("restored[0] = %T (index %v), want *SplineFitPoints3D at fit index 2", got.Item(0), ok)
	}
	if _, ok := got.Item(1).(*Helical3D); !ok {
		t.Fatalf("restored[1] = %T, want *Helical3D", got.Item(1))
	}
}
