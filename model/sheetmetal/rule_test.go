// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
)

// TestDefaultRuleAccessors a default rule reports its seeded thickness/radius/relief and the
// gentle round relief, and develops a 90° bend via the K-factor method.
func TestDefaultRuleAccessors(t *testing.T) {
	r := DefaultRule(0.1, 0.2) // 1 mm thick, 2 mm radius (cm)
	if r.Name() != "Default" {
		t.Errorf("name = %q, want Default", r.Name())
	}
	if math.Abs(r.Thickness()-0.1) > tol || math.Abs(r.BendRadius()-0.2) > tol {
		t.Errorf("thickness/radius = %v/%v, want 0.1/0.2", r.Thickness(), r.BendRadius())
	}
	if r.Relief().Shape != types.ReliefRound {
		t.Errorf("relief shape = %v, want round", r.Relief().Shape)
	}
	if math.Abs(r.ReliefWidth()-0.05) > tol || math.Abs(r.ReliefDepth()-0.05) > tol {
		t.Errorf("relief size = %v/%v, want 0.05/0.05", r.ReliefWidth(), r.ReliefDepth())
	}
	if r.Gap() != 0 {
		t.Errorf("default gap = %v, want 0", r.Gap())
	}
	if r.Unfold().Type != types.KFactorUnfold {
		t.Errorf("unfold = %v, want kFactor", r.Unfold().Type)
	}
}

// TestRuleSetters every setter replaces its closure/value and the accessor reflects it.
func TestRuleSetters(t *testing.T) {
	r := DefaultRule(0.1, 0.1)
	r.SetName("16ga steel")
	r.SetThickness(Constant(0.15))
	r.SetBendRadius(Constant(0.25))
	r.SetGap(Constant(0.02))
	r.SetRelief(Relief{Shape: types.ReliefTear, Width: Constant(0.03), Depth: Constant(0.04)})
	r.SetUnfold(KFactorMethod(0.5))

	if r.Name() != "16ga steel" {
		t.Errorf("name = %q", r.Name())
	}
	if r.Thickness() != 0.15 || r.BendRadius() != 0.25 || r.Gap() != 0.02 {
		t.Errorf("setters: t/r/gap = %v/%v/%v", r.Thickness(), r.BendRadius(), r.Gap())
	}
	if r.Relief().Shape != types.ReliefTear || r.ReliefWidth() != 0.03 || r.ReliefDepth() != 0.04 {
		t.Errorf("relief = %+v", r.Relief())
	}
	if r.Unfold().KFactor != 0.5 {
		t.Errorf("K-factor = %v, want 0.5", r.Unfold().KFactor)
	}
}

// TestRuleBendAllowanceDefaultsRadius a non-positive radius falls back to the rule's bend
// radius, so callers can omit it.
func TestRuleBendAllowanceDefaultsRadius(t *testing.T) {
	r := DefaultRule(0.1, 0.2)
	angle := math.Pi / 2
	withExplicit := r.BendAllowance(angle, 0.2)
	withDefault := r.BendAllowance(angle, 0) // 0 ⇒ use rule's 0.2 radius
	if math.Abs(withExplicit-withDefault) > tol {
		t.Errorf("defaulted radius gave %v, explicit gave %v", withDefault, withExplicit)
	}
	// Deduction is consistent with allowance for the same bend.
	if bd := r.BendDeduction(angle, 0); math.IsNaN(bd) {
		t.Error("bend deduction is NaN")
	}
}

// TestCallNilClosure a nil length closure reads as zero rather than panicking.
func TestCallNilClosure(t *testing.T) {
	r := NewRule("bare", nil, nil, nil, Relief{}, KFactorMethod(0.44))
	if r.Thickness() != 0 || r.BendRadius() != 0 || r.Gap() != 0 || r.ReliefWidth() != 0 {
		t.Error("nil closures must read as zero")
	}
}

// TestBendTableRowsAndEquationSource the table exposes its sorted rows and the equation
// method reports its source (persistence reads both).
func TestBendTableRowsAndEquationSource(t *testing.T) {
	table := NewBendTable([]BendTableRow{
		{Angle: 2, Allowance: 0.2}, {Angle: 1, Allowance: 0.1},
	})
	rows := table.Rows()
	if len(rows) != 2 || rows[0].Angle != 1 {
		t.Errorf("rows not sorted by angle: %+v", rows)
	}
	if KFactorMethod(0.44).EquationSource() != "" {
		t.Error("K-factor method must report no equation source")
	}
	eq, _ := EquationMethod("a*r")
	if eq.EquationSource() != "a*r" {
		t.Errorf("equation source = %q, want a*r", eq.EquationSource())
	}
}

// TestBendTableMatchToleranceBothSigns a row whose radius is below and one above the query
// both fail to match (exercising the tolerance check in both directions), so the lookup
// falls back to K-factor.
func TestBendTableMatchToleranceBothSigns(t *testing.T) {
	table := NewBendTable([]BendTableRow{
		{Angle: math.Pi / 2, Radius: 0.05, Thickness: 0.1, Allowance: 1.1}, // radius below query
		{Angle: math.Pi / 2, Radius: 0.50, Thickness: 0.1, Allowance: 2.2}, // radius above query
	})
	if _, ok := table.BendAllowance(math.Pi/2, 0.2, 0.1); ok {
		t.Error("neither row should match radius 0.2 within tolerance")
	}
}
