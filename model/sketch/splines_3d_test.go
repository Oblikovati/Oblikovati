// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

// TestSpline3DInterpolatesPoints checks an open fit spline passes through its defining
// points and contributes them as solver DOFs.
func TestSpline3DInterpolatesPoints(t *testing.T) {
	s := NewSketches3D().Add()
	pts := []gmath.Point3{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 2, Z: 0}, {X: 3, Y: 1, Z: 1}, {X: 4, Y: 0, Z: 0}}
	sp := s.AddSpline3D(pts, false, true)

	if !sp.IsFitType() || sp.PointCount() != 4 {
		t.Fatalf("spline fit=%v count=%d, want fit/4", sp.IsFitType(), sp.PointCount())
	}
	if s.DegreesOfFreedom() != 12 {
		t.Errorf("a free 4-point spline has 12 DOF, got %d", s.DegreesOfFreedom())
	}
	sample := sp.Sample()
	// The sampled polyline starts and ends exactly on the first/last defining point.
	if sample[0].DistanceTo(pts[0]) > 1e-9 || sample[len(sample)-1].DistanceTo(pts[3]) > 1e-9 {
		t.Errorf("open spline endpoints %v..%v, want %v..%v", sample[0], sample[len(sample)-1], pts[0], pts[3])
	}
}

// TestSpline3DClosedWraps checks a closed spline produces a wrapped sample (no dangling
// endpoint) and more samples than its control count.
func TestSpline3DClosedWraps(t *testing.T) {
	s := NewSketches3D().Add()
	pts := []gmath.Point3{{X: 0, Y: 0, Z: 0}, {X: 4, Y: 0, Z: 0}, {X: 4, Y: 3, Z: 1}, {X: 0, Y: 3, Z: 1}}
	sp := s.AddSpline3D(pts, true, true)
	if len(sp.Sample()) <= len(pts) {
		t.Errorf("closed spline should sample more densely than its %d control points", len(pts))
	}
}

// TestSpline3DTwoPointsIsChord checks a 2-point spline degrades to its control polygon.
func TestSpline3DTwoPointsIsChord(t *testing.T) {
	s := NewSketches3D().Add()
	sp := s.AddSpline3D([]gmath.Point3{{X: 0}, {X: 5, Y: 5, Z: 5}}, false, true)
	if got := sp.Sample(); len(got) != 2 {
		t.Errorf("a 2-point spline samples to its 2 endpoints, got %d", len(got))
	}
}

// TestFixedSpline3D checks the immutable spline samples through its stored coordinates and
// adds no solver DOFs.
func TestFixedSpline3D(t *testing.T) {
	s := NewSketches3D().Add()
	coords := []gmath.Point3{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 1}, {X: 2, Y: 0, Z: 2}}
	s.AddFixedSpline3D(coords, false)
	if s.DegreesOfFreedom() != 0 {
		t.Errorf("a fixed spline adds no DOF, got %d", s.DegreesOfFreedom())
	}
	fs := s.Entities()[0].(*FixedSpline3D)
	if got := fs.Sample(); got[0].DistanceTo(coords[0]) > 1e-9 {
		t.Errorf("fixed spline start = %v, want %v", got[0], coords[0])
	}
}

// TestEquationCurve3D checks a parametric helix-like curve evaluates correctly and errors
// on a bad expression.
func TestEquationCurve3D(t *testing.T) {
	s := NewSketches3D().Add()
	e, err := s.AddEquationCurve3D("cos(t)", "sin(t)", "t", 0, math.Pi)
	if err != nil {
		t.Fatalf("AddEquationCurve3D: %v", err)
	}
	if got := e.At(0); got.DistanceTo(gmath.P3(1, 0, 0)) > 1e-9 {
		t.Errorf("At(0) = %v, want (1,0,0)", got)
	}
	if got := e.At(math.Pi / 2); got.DistanceTo(gmath.P3(0, 1, math.Pi/2)) > 1e-9 {
		t.Errorf("At(π/2) = %v, want (0,1,π/2)", got)
	}
	if len(e.Sample(8)) != 9 {
		t.Errorf("Sample(8) = %d points, want 9", len(e.Sample(8)))
	}
	if _, err := s.AddEquationCurve3D("cos(t)", "%%bad%%", "t", 0, 1); err == nil {
		t.Error("a malformed expression should error")
	}
}

// TestSplines3DRoundTrip checks the spline family survives marshal→apply.
func TestSplines3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	s.AddSpline3D([]gmath.Point3{{X: 0}, {X: 1, Y: 1}, {X: 2}}, false, true)
	cp := s.AddSpline3D([]gmath.Point3{{X: 0}, {X: 1, Z: 1}, {X: 2}}, false, false)
	cp.SetConstruction(true)
	s.AddFixedSpline3D([]gmath.Point3{{X: 0}, {X: 1, Y: 2, Z: 3}}, false)
	if _, err := s.AddEquationCurve3D("t", "t*t", "0", 0, 2); err != nil {
		t.Fatalf("equation curve: %v", err)
	}

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	ents := dst.Item(0).Entities()
	if len(ents) != 4 {
		t.Fatalf("restored %d entities, want 4", len(ents))
	}
	if sp, ok := ents[0].(*Spline3D); !ok || !sp.IsFitType() {
		t.Errorf("entity 0 should be a fit spline, got %T", ents[0])
	}
	if cp2, ok := ents[1].(*Spline3D); !ok || cp2.IsFitType() || !cp2.IsConstruction() {
		t.Errorf("entity 1 should be a construction control spline, got %T", ents[1])
	}
	if eq, ok := ents[3].(*EquationCurve3D); !ok || eq.YExpr != "t*t" {
		t.Errorf("entity 3 should be the equation curve, got %T", ents[3])
	}
}
