// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/model/param"
)

// TestEvalParamTextRoutesThroughParameterEvaluator is the UI-side counterpart to the
// router's resolveQuantity check: a feature/tool dialog field that is a parameter
// name or a formula over parameters resolves against the active part's table, while
// literals and bare numbers still parse (Oblikovati.API#187, UI side).
func TestEvalParamTextRoutesThroughParameterEvaluator(t *testing.T) {
	s := NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	part, err := activePart(s)
	if err != nil {
		t.Fatalf("activePart: %v", err)
	}
	if _, err := part.Parameters().AddUserParameter("len", "20 mm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	cases := []struct {
		src    string
		wantCm float64
		what   string
	}{
		{"len", 2.0, "bare parameter name"},
		{"len + 5 mm", 2.5, "formula over a parameter"},
		{"len * 2", 4.0, "parameter arithmetic"},
		{"12 mm", 1.2, "literal still works"},
		{"7", 0.7, "bare number uses the default length unit (mm)"},
	}
	for _, c := range cases {
		q, err := s.evalParamText(c.src, param.Length)
		if err != nil {
			t.Errorf("evalParamText(%q) [%s]: %v", c.src, c.what, err)
			continue
		}
		if math.Abs(q.Value-c.wantCm) > 1e-9 {
			t.Errorf("evalParamText(%q) [%s] = %g cm, want %g", c.src, c.what, q.Value, c.wantCm)
		}
	}
}

// TestEvalParamTextWrongDimensionFallsBack: a parameter of a different dimension than
// the field is not silently accepted — it falls back to the literal parser, which
// reports the real error.
func TestEvalParamTextWrongDimensionFallsBack(t *testing.T) {
	s := NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	part, err := activePart(s)
	if err != nil {
		t.Fatalf("activePart: %v", err)
	}
	if _, err := part.Parameters().AddUserParameter("ang", "30 deg"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	if _, err := s.evalParamText("ang", param.Length); err == nil {
		t.Error("an angle parameter used as a length should error, not resolve")
	}
}
