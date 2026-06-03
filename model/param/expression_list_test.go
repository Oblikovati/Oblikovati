// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

func TestExpressionListRoundTrip(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 mm")
	if p.IsMultiValue() {
		t.Fatal("a fresh parameter should be single-valued")
	}
	if err := p.SetExpressionList([]string{"10 mm", "20 mm", "30 mm"}, false); err != nil {
		t.Fatalf("SetExpressionList: %v", err)
	}
	if !p.IsMultiValue() || len(p.ExpressionList()) != 3 {
		t.Errorf("list = %v, want 3 entries multi-value", p.ExpressionList())
	}
}

func TestSelectValueFromList(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 mm")
	_ = p.SetExpressionList([]string{"10 mm", "20 mm"}, false)

	if err := p.SelectValue("20 mm"); err != nil {
		t.Fatalf("SelectValue: %v", err)
	}
	if !approxScalar(p.Value().Value, 2.0) { // 20 mm = 2 cm
		t.Errorf("value = %v, want 2", p.Value().Value)
	}
	if err := p.SelectValue("99 mm"); err == nil {
		t.Error("selecting a value outside the list (custom not allowed) should fail")
	}
}

func TestCustomValueOverwrites(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 mm")
	_ = p.SetExpressionList([]string{"10 mm", "20 mm"}, true)

	if err := p.SelectValue("35 mm"); err != nil { // a custom value is allowed
		t.Fatalf("custom SelectValue: %v", err)
	}
	if !approxScalar(p.Value().Value, 3.5) {
		t.Errorf("value = %v, want 3.5", p.Value().Value)
	}
	// Picking a different custom value replaces it — there is only ever one current value.
	if err := p.SelectValue("40 mm"); err != nil {
		t.Fatalf("second custom SelectValue: %v", err)
	}
	if !approxScalar(p.Value().Value, 4.0) {
		t.Errorf("value = %v, want 4", p.Value().Value)
	}
	if l := p.ExpressionList(); len(l) != 2 { // the fixed list is unchanged by custom values
		t.Errorf("list = %v, want the original 2 entries", l)
	}
}

func TestClearExpressionList(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d", "10 mm")
	_ = p.SetExpressionList([]string{"10 mm", "20 mm"}, false)
	p.ClearExpressionList()
	if p.IsMultiValue() {
		t.Error("ClearExpressionList should make the parameter single-valued")
	}
}
