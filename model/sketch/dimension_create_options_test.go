// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestLinearDiameterDoublesOffsetMeasure: an offset dimension created with linearDiameter reports
// twice the perpendicular distance, so it presents a diameter value (#1875).
func TestLinearDiameterDoublesOffsetMeasure(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	p := s.Points().Add(math.P2(2, 3)) // perpendicular distance 3

	d, err := s.DimensionConstraints().AddOffsetDim(p, l, true, "1 cm")
	if err != nil {
		t.Fatalf("AddOffsetDim: %v", err)
	}
	if !d.LinearDiameter() {
		t.Error("LinearDiameter() = false, want true")
	}
	if got := d.Measured(); stdmath.Abs(got-6) > 1e-9 {
		t.Errorf("linear-diameter offset measure = %v, want 6 (2×3)", got)
	}
}

// TestLinearDiameterDrivesToHalfDistance: driving a linear-diameter offset dimension to "6 cm"
// solves the perpendicular distance to 3 — the diameter target is met at half the linear distance,
// proving report and drive stay consistent (#1875).
func TestLinearDiameterDrivesToHalfDistance(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	p := s.Points().Add(math.P2(2, 5)) // starts at distance 5
	g := s.GeometricConstraints()
	g.AddFix(l.A)
	g.AddFix(l.B)

	if _, err := s.DimensionConstraints().AddOffsetDim(p, l, true, "6 cm"); err != nil {
		t.Fatalf("AddOffsetDim: %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual=%v", r.Residual)
	}
	if got := stdmath.Abs(float64(p.Y)); stdmath.Abs(got-3) > 1e-6 {
		t.Errorf("solved perpendicular distance = %v, want 3 (half the 6 cm diameter)", got)
	}
}

// TestTextPointSetAndGet round-trips a text placement through the accessors (#1875).
func TestTextPointSetAndGet(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	p := s.Points().Add(math.P2(2, 3))
	d, _ := s.DimensionConstraints().AddOffsetDim(p, l, false, "3 cm")

	if _, ok := d.TextPoint(); ok {
		t.Error("a fresh dimension should have no text point")
	}
	d.SetTextPoint(math.P2(1.5, 2))
	if tp, ok := d.TextPoint(); !ok || tp != math.P2(1.5, 2) {
		t.Errorf("TextPoint() = (%v, %v), want (1.5,2), true", tp, ok)
	}
}

// TestDimensionCreateOptionsSurviveRoundTrip serializes an offset dimension carrying both
// linearDiameter and a text point, then restores it and checks both survived (#1875).
func TestDimensionCreateOptionsSurviveRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	p := s.Points().Add(math.P2(2, 3))
	d, _ := s.DimensionConstraints().AddOffsetDim(p, l, true, "6 cm")
	d.SetTextPoint(math.P2(1.5, 2))

	out := roundTrip(t, sc)
	dims := out.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("restored dimension count = %d, want 1", len(dims))
	}
	got := dims[0]
	if !got.LinearDiameter() {
		t.Error("restored dim lost its linearDiameter flag")
	}
	if tp, ok := got.TextPoint(); !ok || tp != math.P2(1.5, 2) {
		t.Errorf("restored textPoint = (%v, %v), want (1.5,2), true", tp, ok)
	}
	if v := got.Measured(); stdmath.Abs(v-6) > 1e-9 {
		t.Errorf("restored linear-diameter measure = %v, want 6", v)
	}
}

// TestCopyCarriesTextPointAndLinearDiameter copies an offset dimension carrying both attributes to
// another sketch and checks they travel with it (#1875).
func TestCopyCarriesTextPointAndLinearDiameter(t *testing.T) {
	src := NewSketches().Add(XYPlane())
	l := src.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	p := src.Points().Add(math.P2(2, 3))
	d, _ := src.DimensionConstraints().AddOffsetDim(p, l, true, "6 cm")
	d.SetTextPoint(math.P2(1.5, 2))

	target := NewSketches().Add(XYPlane())
	if _, warns := target.CopyEntitiesWithConstraints(src, src.Entities(), math.V2(10, 0)); len(warns) != 0 {
		t.Fatalf("unexpected copy warnings: %v", warns)
	}
	dims := target.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("copied dimension count = %d, want 1", len(dims))
	}
	got := dims[0]
	if !got.LinearDiameter() {
		t.Error("copy dropped the linearDiameter flag")
	}
	if tp, ok := got.TextPoint(); !ok || tp != math.P2(1.5, 2) {
		t.Errorf("copied textPoint = (%v, %v), want (1.5,2), true", tp, ok)
	}
}
