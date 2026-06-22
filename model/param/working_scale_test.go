// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"math"
	"testing"
)

// TestWorkingScaleDefaultIsCentimetre is the back-compatibility guarantee: the default
// working scale is 1 (the centimetre), so every parse/format/convert is byte-identical to
// the pre-Phase-2 behaviour — a stored length Value is still centimetres.
func TestWorkingScaleDefaultIsCentimetre(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	if m.WorkingScale() != 1 {
		t.Fatalf("default WorkingScale = %v, want 1", m.WorkingScale())
	}
	q, err := m.Parse("5", Length) // preferred mm
	if err != nil {
		t.Fatal(err)
	}
	if !approxRelWS(q.Value, 0.5) { // 5 mm = 0.5 cm, stored in cm
		t.Errorf("Parse(5 mm) Value = %v, want 0.5 cm", q.Value)
	}
}

// TestCenteredOnLengthStoresWorkingUnit is the heart of ADR-0042 Phase 2: a length authored
// in the document's unit is stored as an O(1) working coordinate, and round-trips back.
func TestCenteredOnLengthStoresWorkingUnit(t *testing.T) {
	for _, tc := range []struct {
		unit    string
		cmPer   float64
		entered float64 // value entered AND its preferred display unit = tc.unit
	}{
		{"µm", 1e-4, 5}, {"mm", 0.1, 5}, {"km", 1e5, 5}, {"pm", 1e-10, 5},
	} {
		m, err := DefaultUnitsOfMeasure().CenteredOnLength(tc.unit)
		if err != nil {
			t.Fatalf("CenteredOnLength(%q): %v", tc.unit, err)
		}
		if err := m.SetPreferred(Length, tc.unit); err != nil {
			t.Fatal(err)
		}
		q, err := m.Parse("5", Length)
		if err != nil {
			t.Fatal(err)
		}
		// Stored as the working unit ⇒ O(1), equal to the entered number.
		if !approxRelWS(q.Value, tc.entered) {
			t.Errorf("%s: stored Value = %v, want %v (working = display unit)", tc.unit, q.Value, tc.entered)
		}
		// realCm matches the unit's centimetre size × 5.
		if got := q.Value * m.WorkingScale(); !approxRelWS(got, 5*tc.cmPer) {
			t.Errorf("%s: realCm = %v, want %v", tc.unit, got, 5*tc.cmPer)
		}
		// Round-trips through Format.
		if got := m.FormatValue(q); got != "5" {
			t.Errorf("%s: FormatValue = %q, want \"5\"", tc.unit, got)
		}
	}
}

// TestWorkingScaleCrossUnitDisplay checks a working value displays correctly in a DIFFERENT
// unit than the working one: 5 working µm shown in mm is 5e-3 mm.
func TestWorkingScaleCrossUnitDisplay(t *testing.T) {
	m, _ := DefaultUnitsOfMeasure().CenteredOnLength("µm") // working = µm
	if err := m.SetPreferred(Length, "mm"); err != nil {   // but DISPLAY in mm
		t.Fatal(err)
	}
	q := Quantity{Value: 5, Unit: Length} // 5 working µm
	if got := m.ToPreferred(q); !approxRelWS(got, 5e-3) {
		t.Errorf("5 µm in mm = %v, want 5e-3", got)
	}
}

// TestWorkingScaleAreaVolumeExponent verifies the working scale is raised to the unit's
// length exponent: an area uses workingScale², a volume workingScale³.
func TestWorkingScaleAreaVolumeExponent(t *testing.T) {
	m, _ := DefaultUnitsOfMeasure().CenteredOnLength("µm") // ws = 1e-4
	// 1 mm² = (1000 µm)² = 1e6 working µm².
	area, err := m.Parse("1 mm^2", Area)
	if err != nil {
		t.Fatal(err)
	}
	if !approxRelWS(area.Value, 1e6) {
		t.Errorf("1 mm² in working µm² = %v, want 1e6", area.Value)
	}
	// 1 mm³ = (1000 µm)³ = 1e9 working µm³.
	vol, err := m.Parse("1 mm^3", Volume)
	if err != nil {
		t.Fatal(err)
	}
	if !approxRelWS(vol.Value, 1e9) {
		t.Errorf("1 mm³ in working µm³ = %v, want 1e9", vol.Value)
	}
}

// TestToFromPreferredRoundTripScaled round-trips at a non-cm working scale.
func TestToFromPreferredRoundTripScaled(t *testing.T) {
	m, _ := DefaultUnitsOfMeasure().CenteredOnLength("km")
	if err := m.SetPreferred(Length, "m"); err != nil {
		t.Fatal(err)
	}
	q := m.FromPreferred(2500, Length) // 2500 m
	if got := m.ToPreferred(q); !approxRelWS(got, 2500) {
		t.Errorf("round-trip 2500 m = %v", got)
	}
	if !approxRelWS(q.Value*m.WorkingScale(), 250000) { // 2500 m = 250000 cm
		t.Errorf("2500 m realCm = %v, want 250000", q.Value*m.WorkingScale())
	}
}

// TestWorkingScaleGuardsAndRejects covers the validation: a zero-value units object reads as
// the cm default, and invalid scale/unit are rejected naming the offending value.
func TestWorkingScaleGuardsAndRejects(t *testing.T) {
	var zero UnitsOfMeasure // workingScale field == 0
	if zero.WorkingScale() != 1 {
		t.Errorf("zero-value WorkingScale = %v, want 1 (guarded)", zero.WorkingScale())
	}
	if _, err := DefaultUnitsOfMeasure().WithWorkingScale(0); err == nil {
		t.Error("WithWorkingScale(0) should be rejected")
	}
	if _, err := DefaultUnitsOfMeasure().WithWorkingScale(-1); err == nil {
		t.Error("WithWorkingScale(-1) should be rejected")
	}
	if _, err := DefaultUnitsOfMeasure().CenteredOnLength("deg"); err == nil {
		t.Error("CenteredOnLength(deg) should be rejected (not a length unit)")
	}
}

// TestCloneKeepsWorkingScale ensures the working scale survives a Clone.
func TestCloneKeepsWorkingScale(t *testing.T) {
	m, _ := DefaultUnitsOfMeasure().CenteredOnLength("µm")
	if got := m.Clone().WorkingScale(); got != 1e-4 {
		t.Errorf("cloned WorkingScale = %v, want 1e-4", got)
	}
}

func approxRelWS(got, want float64) bool {
	if want == 0 {
		return math.Abs(got) < 1e-12
	}
	return math.Abs(got-want)/math.Abs(want) < 1e-9
}
