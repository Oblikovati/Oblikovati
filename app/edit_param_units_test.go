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

// TestEvalFieldDisplayResolvesFormulas covers the head ParameterInput's evaluator (#1519): a field
// accepts a bare number, a unit-bearing literal, OR a formula over the part's parameters
// ("D0 * 10.5 mm"), and returns the value in the document's preferred unit; an incomplete formula
// does not resolve, so the field keeps its last good value.
func TestEvalFieldDisplayResolvesFormulas(t *testing.T) {
	s := NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	part, err := activePart(s)
	if err != nil {
		t.Fatalf("activePart: %v", err)
	}
	if _, err := part.Parameters().AddUserParameter("len", "20 mm"); err != nil { // 2 cm
		t.Fatalf("add len: %v", err)
	}
	if _, err := part.Parameters().AddUserParameter("n", "3"); err != nil { // unitless
		t.Fatalf("add n: %v", err)
	}
	lenCases := []struct {
		src  string
		want float64 // in mm (the document length unit)
	}{
		{"10.5 mm", 10.5},
		{"len", 20},
		{"len * 0.5", 10},
		{"n * 10.5 mm", 31.5}, // the user's "D0 * 10.5 mm" shape: a unitless parameter scaling a length
		{"len + 5 mm", 25},
		{"7", 7}, // a bare number uses the document length unit
	}
	for _, c := range lenCases {
		got, ok := s.EvalLengthDisplay(c.src)
		if !ok {
			t.Errorf("EvalLengthDisplay(%q): did not resolve", c.src)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("EvalLengthDisplay(%q) = %g mm, want %g", c.src, got, c.want)
		}
	}
	if _, ok := s.EvalLengthDisplay("len *"); ok {
		t.Error("an incomplete formula should not resolve")
	}
	if got, ok := s.EvalAngleDisplay("45 deg"); !ok || math.Abs(got-45) > 1e-9 {
		t.Errorf("EvalAngleDisplay(\"45 deg\") = %g, ok=%v, want 45", got, ok)
	}
	if got, ok := s.EvalUnitless("n * 2"); !ok || math.Abs(got-6) > 1e-9 {
		t.Errorf("EvalUnitless(\"n * 2\") = %g, ok=%v, want 6", got, ok)
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
