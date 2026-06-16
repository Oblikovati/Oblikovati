// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/sheetmetal"
)

// TestEnableSheetMetalSeedsRule a fresh part enters the environment with a default rule and
// the backing Thickness/BendRadius parameters.
func TestEnableSheetMetalSeedsRule(t *testing.T) {
	d := NewPartComponentDefinition()
	if d.IsSheetMetal() {
		t.Fatal("a fresh part must not be sheet metal")
	}
	rule, err := d.EnableSheetMetal()
	if err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	if !d.IsSheetMetal() || d.SheetMetal() != rule {
		t.Fatal("EnableSheetMetal did not attach the rule")
	}
	if _, ok := d.Parameters().ByName("Thickness"); !ok {
		t.Error("Thickness parameter was not created")
	}
	if _, ok := d.Parameters().ByName("BendRadius"); !ok {
		t.Error("BendRadius parameter was not created")
	}
	if math.Abs(rule.Thickness()-defaultSheetThickness) > 1e-9 {
		t.Errorf("default thickness = %v cm, want %v", rule.Thickness(), defaultSheetThickness)
	}
}

// TestEnableSheetMetalIdempotent re-enabling keeps the same rule (and does not duplicate
// the parameters).
func TestEnableSheetMetalIdempotent(t *testing.T) {
	d := NewPartComponentDefinition()
	first, _ := d.EnableSheetMetal()
	second, err := d.EnableSheetMetal()
	if err != nil {
		t.Fatalf("re-EnableSheetMetal: %v", err)
	}
	if first != second {
		t.Error("re-enabling produced a different rule")
	}
}

// TestRuleIsParameterBacked editing the Thickness parameter changes the rule's thickness —
// the core invariant that makes a thickness/K-factor edit repropagate to every wall.
func TestRuleIsParameterBacked(t *testing.T) {
	d := NewPartComponentDefinition()
	rule, _ := d.EnableSheetMetal()
	if err := d.SetSheetMetalLengthParam("Thickness", "3 mm"); err != nil {
		t.Fatalf("set Thickness: %v", err)
	}
	if got := rule.Thickness(); math.Abs(got-0.3) > 1e-9 { // 3 mm = 0.3 cm
		t.Errorf("rule.Thickness() after edit = %v cm, want 0.3", got)
	}
}

// TestSheetMetalRecipeRoundTrips a sheet-metal part with an edited rule (thickness, relief,
// equation unfold) survives a marshal/restore: the part is still sheet metal and the rule
// matches.
func TestSheetMetalRecipeRoundTrips(t *testing.T) {
	src := NewPartComponentDefinition()
	rule, _ := src.EnableSheetMetal()
	_ = src.SetSheetMetalLengthParam("Thickness", "2 mm")
	rule.SetRelief(sheetmetal.Relief{Shape: types.ReliefSquare, Width: sheetmetal.Constant(0.05), Depth: sheetmetal.Constant(0.07)})
	eq, err := sheetmetal.EquationMethod("a * (r + 0.4 * t)")
	if err != nil {
		t.Fatalf("equation: %v", err)
	}
	rule.SetUnfold(eq)

	blob, err := src.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	dst := NewPartComponentDefinition()
	if err := dst.ApplyRecipe(blob); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.SheetMetal()
	if got == nil {
		t.Fatal("restored part is not sheet metal")
	}
	if math.Abs(got.Thickness()-0.2) > 1e-9 {
		t.Errorf("restored thickness = %v cm, want 0.2", got.Thickness())
	}
	if got.Relief().Shape != types.ReliefSquare {
		t.Errorf("restored relief shape = %v, want square", got.Relief().Shape)
	}
	if got.Unfold().Type != types.EquationUnfold {
		t.Errorf("restored unfold type = %v, want equation", got.Unfold().Type)
	}
	// The restored equation must compute the same allowance as the original.
	want := rule.BendAllowance(math.Pi/2, 0.2)
	if g := got.BendAllowance(math.Pi/2, 0.2); math.Abs(g-want) > 1e-9 {
		t.Errorf("restored bend allowance = %v, want %v", g, want)
	}
}
