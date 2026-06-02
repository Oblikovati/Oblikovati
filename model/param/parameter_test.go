// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

func TestParameterTriadFromExpression(t *testing.T) {
	ps := NewParameters()
	p, err := ps.AddUserParameter("width", "25 mm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	if !p.Health().OK() {
		t.Fatalf("health = %+v, want healthy", p.Health())
	}
	if !approxScalar(p.Value().Value, 2.5) || p.Unit() != Length { // 25 mm = 2.5 cm
		t.Errorf("Value = %v, want 2.5 length", p.Value())
	}
	if !approxScalar(p.ModelValue(), 2.5) { // no tolerance → model == nominal
		t.Errorf("ModelValue = %v, want 2.5", p.ModelValue())
	}
}

func TestSetExpressionUpdatesValue(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("a", "1 cm")
	if err := p.SetExpression("3 cm + 5 mm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	if !approxScalar(p.Value().Value, 3.5) { // 3 cm + 0.5 cm
		t.Errorf("Value after SetExpression = %v, want 3.5", p.Value().Value)
	}
}

func TestSetValueEquivalentToConstantExpression(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("a", "1 cm")
	if err := p.SetValue(Q(7, Length)); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if p.Value() != (Quantity{7, Length}) {
		t.Errorf("Value = %v, want {7 length}", p.Value())
	}
	// The expression now re-evaluates to the same constant.
	if q, err := p.expr.Eval(nil); err != nil || q != (Quantity{7, Length}) {
		t.Errorf("constant expr eval = %v, %v; want {7 length}", q, err)
	}
}

func TestDimensionalExpressionMarksSick(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("a", "1 cm")
	if err := p.SetExpression("1 mm + 1 deg"); err != nil {
		t.Fatalf("SetExpression (parse) should succeed: %v", err)
	}
	if p.Health().Status != Failed {
		t.Errorf("health = %+v, want Failed", p.Health())
	}
}

func TestReadOnlyParametersRejectEdits(t *testing.T) {
	ps := NewParameters()
	ref, err := ps.AddReferenceParameter("measured", Q(5, Length))
	if err != nil {
		t.Fatalf("AddReferenceParameter: %v", err)
	}
	if err := ref.SetExpression("10 mm"); err == nil {
		t.Error("reference parameter should reject SetExpression")
	}
	if err := ref.SetValue(Q(1, Length)); err == nil {
		t.Error("reference parameter should reject SetValue")
	}
	if ref.Kind() != ReferenceParam || !ref.Health().OK() {
		t.Errorf("reference param kind/health = %s/%+v", ref.Kind(), ref.Health())
	}
}

func TestMalformedExpressionRejected(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("a", "1 cm")
	if err := p.SetExpression("3 +"); err == nil {
		t.Error("malformed expression should error from SetExpression")
	}
}
