// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	stdmath "math"
	"testing"
)

func TestParseExplicitUnitsToDatabase(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	cases := []struct {
		in       string
		wantVal  float64
		wantUnit Unit
	}{
		{"25 mm", 2.5, Length}, // 25 mm = 2.5 cm
		{"25mm", 2.5, Length},  // no space
		{"1 m", 100, Length},   // 1 m = 100 cm
		{"180 deg", stdmath.Pi, Angle},
		{"2 cm^2", 2, Area},
		{"-5 mm", -0.5, Length},
		{"1.5e2 mm", 15, Length}, // 150 mm = 15 cm
	}
	for _, c := range cases {
		q, err := m.Parse(c.in, Length)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", c.in, err)
			continue
		}
		if !approxScalar(q.Value, c.wantVal) || q.Unit != c.wantUnit {
			t.Errorf("Parse(%q) = {%g %s}, want {%g %s}", c.in, q.Value, q.Unit, c.wantVal, c.wantUnit)
		}
	}
}

func TestParseBareNumberUsesPreferredUnit(t *testing.T) {
	m := DefaultUnitsOfMeasure() // length pref = mm
	q, err := m.Parse("25", Length)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approxScalar(q.Value, 2.5) { // 25 mm → 2.5 cm
		t.Errorf("bare number = %g cm, want 2.5", q.Value)
	}
}

func TestParseFormatRoundTrip(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	for _, in := range []string{"25 mm", "180 deg", "3.5 m", "12.5 mm"} {
		q, err := m.Parse(in, Length)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		q2, err := m.Parse(m.Format(q), Length)
		if err != nil {
			t.Fatalf("re-parse of %q: %v", m.Format(q), err)
		}
		if !approxScalar(q.Value, q2.Value) || q.Unit != q2.Unit {
			t.Errorf("round-trip %q: {%g %s} != {%g %s}", in, q.Value, q.Unit, q2.Value, q2.Unit)
		}
	}
}

func TestSwitchingDisplayUnitsKeepsStoredValue(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	q := Q(1, Length) // 1 cm in the database
	if got := m.Format(q); got != "10 mm" {
		t.Errorf("default format = %q, want \"10 mm\"", got)
	}
	if err := m.SetPreferred(Length, "cm"); err != nil {
		t.Fatalf("SetPreferred: %v", err)
	}
	if got := m.Format(q); got != "1 cm" {
		t.Errorf("after switch, format = %q, want \"1 cm\"", got)
	}
	// The stored value never changed — only its presentation did.
	if q.Value != 1 {
		t.Errorf("stored value changed to %g, want 1", q.Value)
	}
}

func TestSetPreferredRejectsWrongCategory(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	if err := m.SetPreferred(Length, "deg"); err == nil {
		t.Error("setting an angle unit as the length preference should error")
	}
}

// TestToFromPreferredRoundTrip pins the database↔preferred-unit conversion pair
// the UI fields cross every frame: ToPreferred expresses a stored cm value in the
// preferred unit, and FromPreferred is its exact inverse.
func TestToFromPreferredRoundTrip(t *testing.T) {
	m := DefaultUnitsOfMeasure().Clone()
	if err := m.SetPreferred(Length, "in"); err != nil {
		t.Fatalf("SetPreferred: %v", err)
	}
	// 2.54 cm == 1 in in the preferred unit.
	if got := m.ToPreferred(Q(2.54, Length)); !approxScalar(got, 1) {
		t.Errorf("ToPreferred(2.54 cm) = %g in, want 1", got)
	}
	// FromPreferred(1 in) rebuilds the 2.54 cm database quantity.
	q := m.FromPreferred(1, Length)
	if q.Unit != Length || !approxScalar(q.Value, 2.54) {
		t.Errorf("FromPreferred(1 in) = %+v, want {2.54 Length}", q)
	}
	// Round-trip through both is identity.
	if back := m.ToPreferred(m.FromPreferred(3.7, Length)); !approxScalar(back, 3.7) {
		t.Errorf("round-trip = %g, want 3.7", back)
	}
}

func TestParseErrors(t *testing.T) {
	m := DefaultUnitsOfMeasure()
	if _, err := m.Parse("12 furlongs", Length); err == nil {
		t.Error("unknown unit should error")
	}
	if _, err := m.Parse("abc", Length); err == nil {
		t.Error("non-numeric should error")
	}
}

// approxScalar reports whether two database values are equal within a tight
// tolerance, the shared numeric check for param tests.
func approxScalar(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}
