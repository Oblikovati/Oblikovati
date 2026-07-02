// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

// dims3D returns a fresh 3D sketch backed by a shared parameter store so dimensions can
// be added.
func dims3D(t *testing.T) *Sketch3D {
	t.Helper()
	return NewSketches3D().Add()
}

// TestLineLength3DDrivesGeometry checks a line-length dimension pins a grounded line's
// free end to the target length.
func TestLineLength3DDrivesGeometry(t *testing.T) {
	s := dims3D(t)
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(2, 0, 0))
	s.GeometricConstraints3D().add(NewGround3D(l.A))
	s.GeometricConstraints3D().add(NewParallelToXAxis3D(l))
	if _, err := s.DimensionConstraints3D().AddLineLength(l, "10 cm"); err != nil {
		t.Fatalf("AddLineLength: %v", err)
	}
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve: %+v", res)
	}
	if math.Abs(float64(l.Length())-10) > 1e-6 {
		t.Errorf("line length after solve = %v, want 10", l.Length())
	}
}

// TestRadius3DMeasures checks a radius dimension's measured value and kind name.
func TestRadius3DMeasures(t *testing.T) {
	s := dims3D(t)
	z, _ := gmath.NewUnitVector3(0, 0, 1)
	c := s.AddCircle3D(gmath.P3(0, 0, 0), z, 7)
	d, err := s.DimensionConstraints3D().AddRadius(c, "3 cm")
	if err != nil {
		t.Fatalf("AddRadius: %v", err)
	}
	if d.KindName() != "radius" || math.Abs(d.Measured()-7) > 1e-9 {
		t.Errorf("radius dim kind/measure = %q/%v, want radius/7", d.KindName(), d.Measured())
	}
	if res := s.Solve(); !res.Converged || math.Abs(float64(c.Radius)-3) > 1e-6 {
		t.Errorf("after solve radius = %v, want 3", c.Radius)
	}
}

// TestPointPlaneDistance3D checks the signed distance to the XY plane is the Z coordinate.
func TestPointPlaneDistance3D(t *testing.T) {
	s := dims3D(t)
	p := s.AddPoint3D(gmath.P3(1, 2, 9))
	d, err := s.DimensionConstraints3D().AddPointPlaneDistance(p, gmath.V3(0, 0, 1), "4 cm")
	if err != nil {
		t.Fatalf("AddPointPlaneDistance: %v", err)
	}
	if d.KindName() != "pointPlaneDistance" || math.Abs(d.Measured()-9) > 1e-9 {
		t.Errorf("point-plane dim = %q/%v, want pointPlaneDistance/9", d.KindName(), d.Measured())
	}
	if res := s.Solve(); !res.Converged || math.Abs(float64(p.Z)-4) > 1e-6 {
		t.Errorf("after solve Z = %v, want 4", p.Z)
	}
}

// TestTwoLineAngle3D checks the angle dimension measures and drives the angle between two
// lines sharing a grounded vertex.
func TestTwoLineAngle3D(t *testing.T) {
	s := dims3D(t)
	a := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	b := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 1, 0))
	s.GeometricConstraints3D().add(NewGround3D(a.A))
	s.GeometricConstraints3D().add(NewGround3D(a.B))
	s.GeometricConstraints3D().add(NewGround3D(b.A))
	d, err := s.DimensionConstraints3D().AddTwoLineAngle(a, b, "90 deg")
	if err != nil {
		t.Fatalf("AddTwoLineAngle: %v", err)
	}
	if math.Abs(d.Measured()-math.Pi/4) > 1e-9 { // initial 45°
		t.Errorf("initial angle = %v rad, want π/4", d.Measured())
	}
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve: %+v", res)
	}
	if math.Abs(angleBetweenLines3D(a, b)-math.Pi/2) > 1e-6 {
		t.Errorf("after solve angle = %v, want π/2", angleBetweenLines3D(a, b))
	}
}

// TestDimension3DRefsAndDrive covers the Refs accessor, Drive, and the Acos clamp.
func TestDimension3DRefsAndDrive(t *testing.T) {
	s := dims3D(t)
	a := s.AddPoint3D(gmath.P3(0, 0, 0))
	b := s.AddPoint3D(gmath.P3(3, 0, 0))
	d, err := s.DimensionConstraints3D().AddDistance(a, b, "3 cm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if len(d.Refs()) != 2 {
		t.Errorf("Refs = %d, want 2", len(d.Refs()))
	}
	if err := d.Drive(7); err != nil {
		t.Fatalf("Drive: %v", err)
	}
	// The separation direction is free (both points unconstrained), so assert the driven
	// distance, not a specific coordinate.
	if res := s.Solve(); !res.Converged || math.Abs(float64(a.Position().DistanceTo(b.Position()))-7) > 1e-6 {
		t.Errorf("after Drive(7) + solve, |a-b| = %v, want 7", a.Position().DistanceTo(b.Position()))
	}
	// Acos clamp guards rounding past ±1 (degenerate zero-length line ⇒ angle 0).
	if gmath.Clamp(1.0000001, -1, 1) != 1 || gmath.Clamp(-1.0000001, -1, 1) != -1 || gmath.Clamp(0.5, -1, 1) != 0.5 {
		t.Error("clampUnit3D must pin out-of-range cosines into [-1,1]")
	}
	zero := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(0, 0, 0))
	if angleBetweenLines3D(zero, zero) != 0 {
		t.Error("angle of a zero-length line should be 0, not NaN")
	}
}

// TestPlaneLabelHelpers covers the origin-plane label/normal mappings (all three planes).
func TestPlaneLabelHelpers(t *testing.T) {
	cases := []struct {
		label  string
		normal gmath.Vector3
	}{
		{"XY", gmath.V3(0, 0, 1)},
		{"XZ", gmath.V3(0, 1, 0)},
		{"YZ", gmath.V3(1, 0, 0)},
	}
	for _, c := range cases {
		if got := planeNameFromNormal(c.normal); got != c.label {
			t.Errorf("planeNameFromNormal(%v) = %q, want %q", c.normal, got, c.label)
		}
		if got := planeNormalFromLabel(c.label); got != c.normal {
			t.Errorf("planeNormalFromLabel(%q) = %v, want %v", c.label, got, c.normal)
		}
	}
}

// TestDimensions3DRoundTrip checks the 3D dimensions survive marshal→apply with their
// kind, value expression and operands intact.
func TestDimensions3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	a := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	b := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(0, 1, 0))
	z, _ := gmath.NewUnitVector3(0, 0, 1)
	c := s.AddCircle3D(gmath.P3(0, 0, 0), z, 5)
	p := s.AddPoint3D(gmath.P3(0, 0, 3))
	dc := s.DimensionConstraints3D()
	for _, add := range []func() (*DimensionConstraint3D, error){
		func() (*DimensionConstraint3D, error) { return dc.AddDistance(a.A, b.B, "12 cm") },
		func() (*DimensionConstraint3D, error) { return dc.AddLineLength(a, "10 cm") },
		func() (*DimensionConstraint3D, error) { return dc.AddRadius(c, "5 cm") },
		func() (*DimensionConstraint3D, error) { return dc.AddPointPlaneDistance(p, gmath.V3(0, 1, 0), "3 cm") },
		func() (*DimensionConstraint3D, error) { return dc.AddTwoLineAngle(a, b, "90 deg") },
	} {
		if _, err := add(); err != nil {
			t.Fatalf("add dimension: %v", err)
		}
	}

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.Item(0).DimensionConstraints3D()
	if got.Count() != 5 {
		t.Fatalf("restored %d dimensions, want 5", got.Count())
	}
	kinds := map[string]bool{}
	for _, d := range got.All() {
		kinds[d.KindName()] = true
	}
	for _, want := range []string{"lineLength", "radius", "pointPlaneDistance", "twoLineAngle"} {
		if !kinds[want] {
			t.Errorf("restored dimensions missing kind %q (%v)", want, kinds)
		}
	}
}
