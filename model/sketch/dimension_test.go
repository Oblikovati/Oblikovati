// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

func TestDistanceDimensionResidualTracksParameter(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(3, 0)) // current distance 3 cm
	d, err := s.DimensionConstraints().AddDistance(a, b, "5 cm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	// Target is 5 cm (5 in db units); measured is 3 → residual -2.
	if r := d.Residuals(); len(r) != 1 || !approx(r[0], -2) {
		t.Errorf("residual = %v, want [-2]", r)
	}
	// Move b so geometry matches the parameter → residual zero (the solver would
	// do this; here we verify the residual definition).
	b.SetPosition(math.P2(5, 0))
	if r := d.Residuals(); !approx(r[0], 0) {
		t.Errorf("residual after leveling = %v, want 0", r[0])
	}
	if !approx(d.Measured(), 5) {
		t.Errorf("Measured = %v, want 5", d.Measured())
	}
}

func TestEditingDimensionParameterChangesTarget(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(5, 0))
	d, _ := s.DimensionConstraints().AddDistance(a, b, "5 cm")
	if !approx(d.Residuals()[0], 0) {
		t.Fatal("expected satisfied at 5 cm")
	}
	// Edit the parameter expression: the target (and thus residual) changes — this
	// is what drives geometry through the solver.
	if err := d.Parameter().SetExpression("8 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	if !approx(d.Residuals()[0], 5-8) {
		t.Errorf("residual after edit = %v, want -3", d.Residuals()[0])
	}
}

func TestRadiusDiameterAngleArcLengthDimensions(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	dc := s.DimensionConstraints()
	c := s.Circles().AddByCenterRadius(math.P2(0, 0), 2)
	rd, _ := dc.AddRadius(c, "2 cm")
	if !approx(rd.Residuals()[0], 0) {
		t.Error("radius dimension not satisfied")
	}
	dd, _ := dc.AddDiameter(c, "4 cm")
	if !approx(dd.Residuals()[0], 0) {
		t.Errorf("diameter residual = %v, want 0 (2*r == 4)", dd.Residuals()[0])
	}
	// Right angle between axis-aligned lines.
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 1))
	ad, _ := dc.AddAngle(l1, l2, "90 deg")
	if !approx(ad.Measured(), stdmath.Pi/2) || !approx(ad.Residuals()[0], 0) {
		t.Errorf("angle dim: measured=%v residual=%v", ad.Measured(), ad.Residuals()[0])
	}
	// Quarter arc of radius 2 → length = 2 * (π/2) = π.
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(2, 0), math.P2(0, 2), true)
	al, _ := dc.AddArcLength(arc, "0 cm")
	if !approx(al.Measured(), stdmath.Pi) {
		t.Errorf("arc length measured = %v, want pi", al.Measured())
	}
}

func TestDrivenDimensionReportsButDoesNotConstrain(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(7, 0))
	d, _ := s.DimensionConstraints().AddDistance(a, b, "5 cm")
	d.SetDriven(true)
	if !d.Driven() {
		t.Fatal("SetDriven ignored")
	}
	if d.Residuals() != nil || d.Variables() != nil {
		t.Error("driven dimension still contributes residuals/variables")
	}
	if !approx(d.Measured(), 7) {
		t.Errorf("driven dimension reports %v, want measured 7", d.Measured())
	}
	// Driven dimensions are excluded from the solver's constraint set.
	for _, c := range s.Constraints() {
		if c.EntityID() == d.EntityID() {
			t.Error("driven dimension leaked into Constraints()")
		}
	}
}

func TestConstraintLimitsClampDrive(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	d, _ := s.DimensionConstraints().AddDistance(a, b, "1 cm")
	d.SetLimits(2, 5)
	if l := d.Limits(); !l.Enabled || l.Min != 2 || l.Max != 5 {
		t.Fatalf("limits = %+v", l)
	}
	// Driving above max clamps to max.
	if err := d.Drive(10); err != nil {
		t.Fatalf("Drive: %v", err)
	}
	if !approx(d.Parameter().ModelValue(), 5) {
		t.Errorf("driven-to value = %v, want clamped 5", d.Parameter().ModelValue())
	}
	// Driving below min clamps to min.
	_ = d.Drive(0)
	if !approx(d.Parameter().ModelValue(), 2) {
		t.Errorf("driven-to value = %v, want clamped 2", d.Parameter().ModelValue())
	}
}

func TestConstraintsAggregatesGeometricAndDrivingDims(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	s.GeometricConstraints().AddHorizontal(a, b)
	_, _ = s.DimensionConstraints().AddDistance(a, b, "5 cm")
	if got := len(s.Constraints()); got != 2 {
		t.Errorf("Constraints = %d, want 2 (1 geometric + 1 driving dim)", got)
	}
}

func TestDimensionCollectionAndParameterAccessors(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if s.Parameters() == nil {
		t.Fatal("sketch has no default parameter store")
	}
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	d, _ := s.DimensionConstraints().AddDistance(a, b, "5 cm")
	if d.Kind() != DistanceDim || len(d.Variables()) != 4 {
		t.Errorf("dimension kind/vars wrong: %v / %d", d.Kind(), len(d.Variables()))
	}
	dc := s.DimensionConstraints()
	if dc.Count() != 1 || dc.Item(0) != d || len(dc.All()) != 1 {
		t.Error("dimension collection tracking wrong")
	}
	// Swapping in a shared parameter store re-points the dimension collection.
	shared := param.NewParameters()
	s.SetParameters(shared)
	if s.Parameters() != shared {
		t.Error("SetParameters did not take")
	}
	circ := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	if _, err := s.DimensionConstraints().AddRadius(circ, "1 cm"); err != nil {
		t.Fatalf("AddRadius after SetParameters: %v", err)
	}
	if shared.Count() == 0 {
		t.Error("new dimension did not use the shared parameter store")
	}
}
