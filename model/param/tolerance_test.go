// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"testing"

	"oblikovati.org/api/types"
)

func TestModelValueFollowsModelValueType(t *testing.T) {
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
		ps := NewParameters()
		p, _ := ps.AddUserParameter("d", "10 cm")
		if err := p.SetToleranceDeviation(0.2, -0.1); err != nil {
			t.Fatalf("SetToleranceDeviation: %v", err)
		}
		if err := p.SetModelValueType(c.typ); err != nil {
			t.Fatalf("SetModelValueType(%s): %v", c.typ, err)
		}
		if got := p.ModelValue(); !approxScalar(got, c.want) {
			t.Errorf("ModelValue(type=%s) = %v, want %v", c.typ, got, c.want)
		}
	}
}

func TestToleranceAffectsModelValueNotNominal(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 cm")
	if err := p.SetToleranceDeviation(0.2, -0.2); err != nil {
		t.Fatalf("SetToleranceDeviation: %v", err)
	}
	if err := p.SetModelValueType(Upper); err != nil {
		t.Fatalf("SetModelValueType: %v", err)
	}
	if !approxScalar(p.Value().Value, 10) {
		t.Errorf("nominal Value changed to %v, want 10", p.Value().Value)
	}
	if !approxScalar(p.ModelValue(), 10.2) {
		t.Errorf("ModelValue = %v, want 10.2", p.ModelValue())
	}
}

func TestToleranceSetOperations(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 cm")

	if err := p.SetToleranceSymmetric(0.05); err != nil {
		t.Fatalf("SetToleranceSymmetric: %v", err)
	}
	if tol := p.Tolerance(); tol.Type != types.ToleranceSymmetric || tol.Upper != 0.05 || tol.Lower != -0.05 {
		t.Errorf("symmetric tolerance = %+v, want ±0.05", tol)
	}
	// Limits are absolute values, stored as deviations from the 10 cm nominal.
	if err := p.SetToleranceLimits(10.3, 9.8); err != nil {
		t.Fatalf("SetToleranceLimits: %v", err)
	}
	if tol := p.Tolerance(); tol.Type != types.ToleranceLimitsStacked || !approxScalar(tol.Upper, 0.3) || !approxScalar(tol.Lower, -0.2) {
		t.Errorf("limits tolerance = %+v, want +0.3/-0.2", tol)
	}
	if err := p.SetToleranceMinMax(types.ToleranceMax); err != nil {
		t.Fatalf("SetToleranceMinMax: %v", err)
	}
	if tol := p.Tolerance(); tol.Type != types.ToleranceMax || tol.Upper != 0 {
		t.Errorf("max tolerance = %+v, want bandless max", tol)
	}
	if err := p.SetToleranceDefault(); err != nil {
		t.Fatalf("SetToleranceDefault: %v", err)
	}
	if tol := p.Tolerance(); tol != (Tolerance{}) || tol.Kind() != types.ToleranceDefault {
		t.Errorf("default tolerance = %+v (kind %s), want zero/default", tol, tol.Kind())
	}
}

func TestToleranceSetOperationsRejectBadInput(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 cm")
	if err := p.SetToleranceDeviation(-0.1, 0.1); err == nil {
		t.Error("deviation with upper < lower must be rejected")
	}
	if err := p.SetToleranceSymmetric(-0.1); err == nil {
		t.Error("negative symmetric band must be rejected")
	}
	if err := p.SetToleranceMinMax(types.ToleranceSymmetric); err == nil {
		t.Error("SetToleranceMinMax with a non-min/max type must be rejected")
	}
	if err := p.SetModelValueType(ModelValueType(7)); err == nil {
		t.Error("unknown model value type must be rejected")
	}
	txt, _ := ps.AddTextUserParameter("label", "lid")
	if err := txt.SetToleranceSymmetric(0.1); err == nil {
		t.Error("tolerance on a text parameter must be rejected")
	}
	if err := txt.SetModelValueType(Upper); err == nil {
		t.Error("model value type on a text parameter must be rejected")
	}
}

func TestPrecisionIsDisplayOnly(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 cm")
	before := p.Value()
	p.Precision = 2
	p.DisplayFormat = DisplayFormatFractional
	// Changing precision/format must not touch the stored value or model value.
	if p.Value() != before || !approxScalar(p.ModelValue(), 10) {
		t.Errorf("precision changed numeric state: value=%v modelValue=%v", p.Value(), p.ModelValue())
	}
}
