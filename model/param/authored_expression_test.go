// SPDX-License-Identifier: GPL-2.0-only

package param

import "testing"

// mmDoc is a millimetre-length document (the default), where the database unit is the centimetre,
// so a bare "7" authored for a Length field must resolve to 7 mm = 0.7 cm, not 7 cm.
func mmDoc() UnitsOfMeasure { return DefaultUnitsOfMeasure() }

func TestQualifyAuthoredBareNumberGetsDisplayUnit(t *testing.T) {
	ps := NewParameters()
	p, err := ps.AddUserParameter("d0", "10 mm")
	if err != nil {
		t.Fatalf("AddUserParameter: %v", err)
	}
	cases := []struct{ src, want string }{
		{"7", "7 mm"},           // the reported bug: a lone bare number
		{" 7 ", "7 mm"},         // surrounding whitespace trimmed
		{"3*2", "(3*2) * 1 mm"}, // compound unitless expression
		{"7 mm", "7 mm"},        // already unit-bearing — untouched
		{"1 cm", "1 cm"},        // a different length unit — untouched (still Length)
		{"d0/2", "d0/2"},        // formula resolving to Length — kept live, untouched
	}
	for _, c := range cases {
		if got := p.QualifyAuthored(c.src, Length, mmDoc()); got != c.want {
			t.Errorf("QualifyAuthored(%q, Length) = %q, want %q", c.src, got, c.want)
		}
	}
}

// TestQualifyAuthoredBareNumberResolvesToDisplayValue is the end-to-end proof: a bare "7" set on a
// Length parameter through the seam evaluates to 0.7 cm (7 mm), where raw SetExpression gives 7 cm.
func TestQualifyAuthoredBareNumberResolvesToDisplayValue(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("d0", "10 mm")

	if err := p.SetExpression(p.QualifyAuthored("7", Length, mmDoc())); err != nil {
		t.Fatalf("SetExpression(qualified): %v", err)
	}
	if got := p.Value(); got.Unit != Length || got.Value < 0.6999 || got.Value > 0.7001 {
		t.Errorf("bare \"7\" resolved to %+v, want {0.7 Length} (7 mm)", got)
	}

	// The compound form round-trips through the parser to the same 6 mm = 0.6 cm.
	if err := p.SetExpression(p.QualifyAuthored("3*2", Length, mmDoc())); err != nil {
		t.Fatalf("SetExpression(compound): %v", err)
	}
	if got := p.Value(); got.Unit != Length || got.Value < 0.5999 || got.Value > 0.6001 {
		t.Errorf("\"3*2\" resolved to %+v, want {0.6 Length} (6 mm)", got)
	}
}

func TestQualifyAuthoredAngleAndUnitlessCategory(t *testing.T) {
	ps := NewParameters()
	p, _ := ps.AddUserParameter("a0", "30 deg")

	if got := p.QualifyAuthored("45", Angle, mmDoc()); got != "45 deg" {
		t.Errorf("QualifyAuthored(\"45\", Angle) = %q, want %q", got, "45 deg")
	}
	// A unitless target category takes the bare number verbatim (no unit to attach).
	if got := p.QualifyAuthored("5", Unitless, mmDoc()); got != "5" {
		t.Errorf("QualifyAuthored(\"5\", Unitless) = %q, want %q", got, "5")
	}
}
