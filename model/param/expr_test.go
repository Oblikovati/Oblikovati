// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	stdmath "math"
	"testing"
)

// mapScope is a fake [Scope] backed by a map, for evaluator tests.
type mapScope map[ID]Quantity

func (m mapScope) ValueOf(id ID) (Quantity, bool) {
	q, ok := m[id]
	return q, ok
}

// evalConst parses and evaluates a reference-free expression.
func evalConst(t *testing.T, src string) Quantity {
	t.Helper()
	e, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	q, err := e.Eval(nil)
	if err != nil {
		t.Fatalf("Eval(%q): %v", src, err)
	}
	return q
}

func TestEvalArithmeticAndPrecedence(t *testing.T) {
	if q := evalConst(t, "2 + 3 * 4"); q.Value != 14 || q.Unit != Unitless {
		t.Errorf("2+3*4 = %v, want 14 unitless", q)
	}
	if q := evalConst(t, "(2 + 3) * 4"); q.Value != 20 {
		t.Errorf("(2+3)*4 = %v, want 20", q)
	}
	if q := evalConst(t, "-3 + 1"); q.Value != -2 {
		t.Errorf("-3+1 = %v, want -2", q)
	}
}

func TestEvalUnitLiteralsAndArithmetic(t *testing.T) {
	// 5 mm = 0.5 cm; + 1 cm = 1.5 cm in database units.
	if q := evalConst(t, "5 mm + 1 cm"); !approxScalar(q.Value, 1.5) || q.Unit != Length {
		t.Errorf("5mm+1cm = %v, want 1.5 length", q)
	}
	// 2 * width-style scaling with a unit literal.
	if q := evalConst(t, "2 * 3 mm"); !approxScalar(q.Value, 0.6) || q.Unit != Length {
		t.Errorf("2*3mm = %v, want 0.6 length", q)
	}
}

func TestEvalDimensionalMismatchErrors(t *testing.T) {
	e, err := Parse("1 mm + 1 deg")
	if err != nil {
		t.Fatalf("Parse should succeed (error is at eval): %v", err)
	}
	if _, err := e.Eval(nil); err == nil {
		t.Fatal("1 mm + 1 deg should error at evaluation")
	}
}

func TestEvalFunctions(t *testing.T) {
	if q := evalConst(t, "sin(30 deg)"); !approxScalar(q.Value, 0.5) || q.Unit != Unitless {
		t.Errorf("sin(30 deg) = %v, want 0.5 unitless", q)
	}
	if q := evalConst(t, "sqrt(9)"); !approxScalar(q.Value, 3) || q.Unit != Unitless {
		t.Errorf("sqrt(9) = %v, want 3 unitless", q)
	}
	if q := evalConst(t, "max(3 mm, 5 mm, 1 mm)"); !approxScalar(q.Value, 0.5) || q.Unit != Length {
		t.Errorf("max(...) = %v, want 0.5 length (5 mm)", q)
	}
	if q := evalConst(t, "atan(1)"); !approxScalar(q.Value, stdmath.Pi/4) || q.Unit != Angle {
		t.Errorf("atan(1) = %v, want π/4 angle", q)
	}
}

func TestEvalFunctionUnitErrors(t *testing.T) {
	e, _ := Parse("exp(5 mm)") // exp needs a unitless argument
	if _, err := e.Eval(nil); err == nil {
		t.Error("exp(5 mm) should error")
	}
	e2, _ := Parse("max(3 mm, 5 deg)") // mixed units
	if _, err := e2.Eval(nil); err == nil {
		t.Error("max with mixed units should error")
	}
}

func TestEvalReferences(t *testing.T) {
	e, err := Parse("2 * width + 5 mm")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if refs := e.References(); len(refs) != 1 || refs[0] != "width" {
		t.Fatalf("References = %v, want [width]", refs)
	}
	// Bind "width" to id 1, then evaluate against a scope.
	if unresolved := e.Bind(func(name string) (ID, bool) {
		if name == "width" {
			return 1, true
		}
		return 0, false
	}); len(unresolved) != 0 {
		t.Fatalf("unresolved refs: %v", unresolved)
	}
	q, err := e.Eval(mapScope{1: Q(10, Length)}) // width = 10 cm
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if !approxScalar(q.Value, 20.5) || q.Unit != Length { // 2*10 + 0.5
		t.Errorf("2*width+5mm = %v, want 20.5 length", q)
	}
}

func TestUnboundAndUndefinedReferences(t *testing.T) {
	e, _ := Parse("x + 1")
	if _, err := e.Eval(mapScope{}); err == nil {
		t.Error("evaluating an unbound reference should error")
	}
	e.Bind(func(string) (ID, bool) { return 0, false }) // x unresolved
	if _, err := e.Eval(mapScope{}); err == nil {
		t.Error("unresolved reference should error at eval")
	}
}

func TestParseErrorsReportPosition(t *testing.T) {
	for _, src := range []string{"2 +", "2 @ 3", "(2 + 3", "sin(", "* 5"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q) should fail", src)
		}
	}
}

func TestConstantFolding(t *testing.T) {
	// A reference-free expression folds to a single numberNode at parse time.
	e, err := Parse("2 * (3 + 4)")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := e.root.(numberNode); !ok {
		t.Errorf("expected folded numberNode, got %T", e.root)
	}
}
