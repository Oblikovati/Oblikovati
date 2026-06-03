// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

func TestAddTextUserParameter(t *testing.T) {
	ps := NewParameters()
	p, err := ps.AddTextUserParameter("material", "steel")
	if err != nil {
		t.Fatalf("AddTextUserParameter: %v", err)
	}
	if !p.IsText() || p.IsNumeric() {
		t.Errorf("value flavor = (text %v numeric %v), want text", p.IsText(), p.IsNumeric())
	}
	if p.Text() != "steel" {
		t.Errorf("Text = %q, want steel", p.Text())
	}
	if p.Expression() != `"steel"` { // surfaced as a quoted literal
		t.Errorf("Expression = %q, want \"steel\" quoted", p.Expression())
	}
	if !p.Health().OK() {
		t.Errorf("health = %+v, want healthy", p.Health())
	}
}

func TestAddBooleanUserParameter(t *testing.T) {
	ps := NewParameters()
	p, err := ps.AddBooleanUserParameter("vented", true)
	if err != nil {
		t.Fatalf("AddBooleanUserParameter: %v", err)
	}
	if !p.IsBoolean() || !p.Bool() {
		t.Errorf("got (boolean %v value %v), want boolean true", p.IsBoolean(), p.Bool())
	}
	if p.Expression() != "true" {
		t.Errorf("Expression = %q, want true", p.Expression())
	}
	if err := p.SetBool(false); err != nil {
		t.Fatalf("SetBool: %v", err)
	}
	if p.Bool() || p.Expression() != "false" {
		t.Errorf("after SetBool(false): value %v expr %q, want false/false", p.Bool(), p.Expression())
	}
}

func TestEditTextParameter(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddTextUserParameter("label", "a")
	if err := p.SetText("b c"); err != nil {
		t.Fatalf("SetText: %v", err)
	}
	if p.Text() != "b c" || p.Expression() != `"b c"` {
		t.Errorf("Text=%q Expression=%q, want \"b c\"", p.Text(), p.Expression())
	}
}

// TestNumericOpsRejectedOnNonNumeric proves text/bool parameters refuse expressions and
// numeric SetValue, and numeric parameters refuse SetText/SetBool.
func TestNumericOpsRejectedOnNonNumeric(t *testing.T) {
	ps := NewParameters()
	text, _ := ps.AddTextUserParameter("t", "x")
	if err := text.SetExpression("1 mm"); err == nil {
		t.Error("SetExpression on a text parameter should fail")
	}
	if err := text.SetValue(Q(1, Length)); err == nil {
		t.Error("SetValue on a text parameter should fail")
	}

	num, _ := ps.AddUserParameter("n", "1 mm")
	if err := num.SetText("hi"); err == nil {
		t.Error("SetText on a numeric parameter should fail")
	}
	if err := num.SetBool(true); err == nil {
		t.Error("SetBool on a numeric parameter should fail")
	}
}
