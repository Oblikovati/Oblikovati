// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"testing"

	"oblikovati.org/api/types"
)

func TestConvertValue(t *testing.T) {
	got, err := ConvertValue(1, "in", "cm")
	if err != nil || !approxScalar(got, 2.54) {
		t.Errorf("1 in→cm = (%g, %v), want 2.54", got, err)
	}
	if _, err := ConvertValue(1, "mm", "deg"); err == nil {
		t.Error("mm→deg must error (different categories)")
	}
	if _, err := ConvertValue(1, "furlong", "cm"); err == nil {
		t.Error("an unknown unit must error")
	}
}

func TestParseUnitCategoryAndUnitCategoryOf(t *testing.T) {
	if cat, ok := ParseUnitCategory("length"); !ok || cat != Length {
		t.Errorf("ParseUnitCategory(length) = (%v, %v), want Length", cat, ok)
	}
	if _, ok := ParseUnitCategory("nope"); ok {
		t.Error("ParseUnitCategory must reject unknown categories")
	}
	if cat, ok := UnitCategoryOf("deg"); !ok || cat != Angle {
		t.Errorf("UnitCategoryOf(deg) = (%v, %v), want Angle", cat, ok)
	}
}

func TestEvaluateExpressionAndReferences(t *testing.T) {
	ps := NewParameters()
	if _, err := ps.AddUserParameter("width", "4 cm"); err != nil {
		t.Fatalf("add width: %v", err)
	}
	q, err := ps.EvaluateExpression("width * 2")
	if err != nil || q.Unit != Length || !approxScalar(q.Value, 8) {
		t.Errorf("width*2 = (%g %s, %v), want 8 cm", q.Value, q.Unit, err)
	}
	if _, err := ps.EvaluateExpression("missing + 1"); err == nil {
		t.Error("an undefined reference must error")
	}
	refs, err := ExpressionReferences("a + b * a")
	if err != nil || len(refs) != 2 || refs[0] != "a" || refs[1] != "b" {
		t.Errorf("references = (%v, %v), want [a b] in first-seen order", refs, err)
	}
}

func TestUnitsOfMeasurePrecisionFields(t *testing.T) {
	u := DefaultUnitsOfMeasure()
	if u.LengthPrecision() != 3 || u.AnglePrecision() != 2 || u.LengthFormat() != types.DisplayFormatDecimal {
		t.Fatalf("defaults = %d/%d/%v, want 3/2/decimal", u.LengthPrecision(), u.AnglePrecision(), u.LengthFormat())
	}
	// Clone is independent: editing the clone's unit map leaves the original alone.
	c := u.Clone()
	if err := c.SetPreferred(Length, "in"); err != nil {
		t.Fatalf("set clone length: %v", err)
	}
	if u.PreferredName(Length) != "mm" {
		t.Errorf("original length changed to %q after editing the clone", u.PreferredName(Length))
	}
	if err := c.SetLengthPrecision(5); err != nil {
		t.Fatalf("set precision: %v", err)
	}
	if c.LengthPrecision() != 5 {
		t.Errorf("clone precision = %d, want 5", c.LengthPrecision())
	}
	if err := c.SetLengthPrecision(-1); err == nil {
		t.Error("a negative precision must error")
	}
	if err := c.SetAnglePrecision(1); err != nil || c.AnglePrecision() != 1 {
		t.Errorf("SetAnglePrecision = (%v); angle precision %d, want 1", err, c.AnglePrecision())
	}
	if err := c.SetAnglePrecision(-2); err == nil {
		t.Error("a negative angle precision must error")
	}
	c.SetLengthFormat(types.DisplayFormatFractional)
	if c.LengthFormat() != types.DisplayFormatFractional {
		t.Errorf("length format = %v, want fractional", c.LengthFormat())
	}
}
