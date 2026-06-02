// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

func TestToleranceModelValue(t *testing.T) {
	nominal := 10.0
	cases := []struct {
		typ  ModelValueType
		want float64
	}{
		{Nominal, 10},
		{Upper, 10.2},
		{Lower, 9.9},
		{Median, 10.05}, // 10 + (0.2 + -0.1)/2
	}
	for _, c := range cases {
		tol := Tolerance{Upper: 0.2, Lower: -0.1, Type: c.typ}
		if got := tol.ModelValue(nominal); !approxScalar(got, c.want) {
			t.Errorf("ModelValue(type=%d) = %v, want %v", c.typ, got, c.want)
		}
	}
}

func TestToleranceAffectsModelValueNotNominal(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 cm")
	p.SetTolerance(Tolerance{Upper: 0.2, Lower: -0.2, Type: Upper})
	if !approxScalar(p.Value().Value, 10) {
		t.Errorf("nominal Value changed to %v, want 10", p.Value().Value)
	}
	if !approxScalar(p.ModelValue(), 10.2) {
		t.Errorf("ModelValue = %v, want 10.2", p.ModelValue())
	}
}

func TestPrecisionIsDisplayOnly(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 cm")
	before := p.Value()
	p.Precision = 2
	p.DisplayFormat = ShowValue
	// Changing precision/format must not touch the stored value or model value.
	if p.Value() != before || !approxScalar(p.ModelValue(), 10) {
		t.Errorf("precision changed numeric state: value=%v modelValue=%v", p.Value(), p.ModelValue())
	}
}
