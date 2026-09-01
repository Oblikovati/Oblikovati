// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sheetmetal"
	"oblikovati.org/model/sketch"
)

// addSquareFace adds a closed square-profile sketch and a base sheet-metal Face over it to
// the part, then recomputes — the minimal sheet-metal body for persistence tests.
func addSquareFace(d *PartComponentDefinition, side float64) {
	sk := d.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(gmath.P2(0, 0))
	c1 := sk.Points().Add(gmath.P2(side, 0))
	c2 := sk.Points().Add(gmath.P2(side, side))
	c3 := sk.Points().Add(gmath.P2(0, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewSheetMetalFaceFeatures(d.Features()).Add(&feature.SheetMetalFaceDefinition{Sketch: sk, ProfileIndex: 0, Operation: ops.NewBody})
	d.Recompute()
}

// TestEnableSheetMetalSeedsRule a fresh part enters the environment with a default rule and
// the backing Thickness/BendRadius parameters.
func TestEnableSheetMetalSeedsRule(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	src := NewPartComponentDefinition()
	rule, _ := src.EnableSheetMetal()
	_ = src.SetSheetMetalLengthParam("Thickness", "2 mm")
	rule.SetRelief(sheetmetal.Relief{Shape: types.ReliefStraight, Width: sheetmetal.Constant(0.05), Depth: sheetmetal.Constant(0.07)})
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
	if got.Relief().Shape != types.ReliefStraight {
		t.Errorf("restored relief shape = %v, want straight", got.Relief().Shape)
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

// TestSheetMetalFaceSurvivesRoundTrip a sheet-metal part with a base Face wall round-trips
// through the recipe: the restored part is still sheet metal and rebuilds the same single
// solid wall (the Face reads the restored Thickness parameter live).
func TestSheetMetalFaceSurvivesRoundTrip(t *testing.T) {
	t.Parallel()
	src := NewPartComponentDefinition()
	if _, err := src.EnableSheetMetal(); err != nil {
		t.Fatalf("enable: %v", err)
	}
	addSquareFace(src, 4)
	if got := src.bodies.Count(); got != 1 {
		t.Fatalf("source has %d bodies, want 1 wall", got)
	}

	blob, err := src.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewPartComponentDefinition()
	if err := dst.ApplyRecipe(blob); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !dst.IsSheetMetal() {
		t.Fatal("restored part lost its sheet-metal environment")
	}
	if got := dst.bodies.Count(); got != 1 {
		t.Errorf("restored part rebuilt %d bodies, want 1 wall", got)
	}
}

// TestSheetMetalBendTableRecipeRoundTrips a bend-table unfold method (its measured rows)
// survives marshal/restore and develops the same length.
func TestSheetMetalBendTableRecipeRoundTrips(t *testing.T) {
	t.Parallel()
	src := NewPartComponentDefinition()
	rule, _ := src.EnableSheetMetal()
	table := sheetmetal.NewBendTable([]sheetmetal.BendTableRow{
		{Angle: math.Pi / 2, Radius: 0.1, Thickness: 0.1, Allowance: 0.31},
	})
	rule.SetUnfold(sheetmetal.BendTableMethod(table))

	blob, err := src.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewPartComponentDefinition()
	if err := dst.ApplyRecipe(blob); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.SheetMetal()
	if got == nil || got.Unfold().Type != types.BendTableUnfold {
		t.Fatalf("restored unfold = %v, want bendTable", got)
	}
	if ba := got.BendAllowance(math.Pi/2, 0.1); math.Abs(ba-0.31) > 1e-9 {
		t.Errorf("restored table allowance = %v, want 0.31", ba)
	}
}

// TestCornerReliefRecipeRoundTrips (#1960): the corner relief is a separate style property from
// the bend relief, so a document that carries an edited one must reopen with the same corner cut
// — falling back to the default would restyle the part silently.
func TestCornerReliefRecipeRoundTrips(t *testing.T) {
	t.Parallel()
	src := NewPartComponentDefinition()
	rule, _ := src.EnableSheetMetal()
	rule.SetCornerRelief(sheetmetal.CornerRelief{
		Shape:          types.CornerSquare,
		Size:           sheetmetal.Constant(0.31),
		Placement:      types.CornerReliefAtBendIntersection,
		ThreeBendShape: types.CornerFullRound,
		ThreeBendSize:  sheetmetal.Constant(0.12),
	})
	blob, err := src.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewPartComponentDefinition()
	if err := dst.ApplyRecipe(blob); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.SheetMetal().CornerRelief()
	if got.Shape != types.CornerSquare || got.Placement != types.CornerReliefAtBendIntersection {
		t.Errorf("restored corner relief = %v at %v, want square at bendIntersection", got.Shape, got.Placement)
	}
	if got.ThreeBendShape != types.CornerFullRound {
		t.Errorf("restored three-bend shape = %v, want fullRound", got.ThreeBendShape)
	}
	if math.Abs(dst.SheetMetal().CornerReliefSize()-0.31) > 1e-9 {
		t.Errorf("restored corner relief size = %v, want 0.31", dst.SheetMetal().CornerReliefSize())
	}
	if math.Abs(dst.SheetMetal().ThreeBendReliefSize()-0.12) > 1e-9 {
		t.Errorf("restored three-bend size = %v, want 0.12", dst.SheetMetal().ThreeBendReliefSize())
	}
}

// TestLegacySquareReliefStillReads: "square" was this enum's spelling for the rectangular BEND
// relief before it was reconciled with Inventor's (#1960). A document written with it must reopen
// with the same relief rather than failing to parse.
func TestLegacySquareReliefStillReads(t *testing.T) {
	t.Parallel()
	shape, ok := types.ParseReliefShape("square")
	if !ok || shape != types.ReliefStraight {
		t.Fatalf(`ParseReliefShape("square") = (%v, %v), want ReliefStraight`, shape, ok)
	}
}

// TestStandardParameterRoster (#1962): a sheet-metal part exposes Inventor's roster of named
// parameters, each carrying the EXPRESSION Inventor's Default style states rather than a frozen
// number — which is what makes the whole style track the gauge.
func TestStandardParameterRoster(t *testing.T) {
	t.Parallel()
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	for name, want := range map[string]string{
		"BendReliefWidth":  "Thickness",
		"BendReliefDepth":  "Thickness * 0.5",
		"CornerReliefSize": "Thickness * 4",
		"MinimumRemnant":   "Thickness * 2",
		"TransitionRadius": "BendRadius",
		"GapSize":          "Thickness",
	} {
		p, ok := d.Parameters().ByName(name)
		if !ok {
			t.Errorf("no %s parameter on a sheet-metal part", name)
			continue
		}
		if p.Expression() != want {
			t.Errorf("%s = %q, want %q", name, p.Expression(), want)
		}
	}
}

// TestEditingReliefParameterRepropagates (#1962): the relief sizes are parameters, so re-authoring
// one by expression has to move the rule — a rule holding a value captured when the part was
// created would keep cutting the old notch.
func TestEditingReliefParameterRepropagates(t *testing.T) {
	t.Parallel()
	d := NewPartComponentDefinition()
	rule, err := d.EnableSheetMetal()
	if err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	before := rule.ReliefWidth()
	p, _ := d.Parameters().ByName("BendReliefWidth")
	if err := d.Parameters().SetExpression(p.ID(), "Thickness * 3"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	if got := rule.ReliefWidth(); math.Abs(got-3*rule.Thickness()) > 1e-9 {
		t.Errorf("relief width after the edit = %v, want 3x the %v thickness (was %v)", got, rule.Thickness(), before)
	}
}

// TestReliefSizesFollowTheGauge (#1962): every roster size is stated against Thickness, so a gauge
// change moves the reliefs with the walls instead of leaving them at the old part's dimensions.
func TestReliefSizesFollowTheGauge(t *testing.T) {
	t.Parallel()
	d := NewPartComponentDefinition()
	rule, err := d.EnableSheetMetal()
	if err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	if err := d.SetSheetMetalLengthParam("Thickness", "4 mm"); err != nil {
		t.Fatalf("set thickness: %v", err)
	}
	if got := rule.ReliefWidth(); math.Abs(got-0.4) > 1e-9 {
		t.Errorf("relief width at a 4 mm gauge = %v, want 0.4 cm", got)
	}
	if got := rule.CornerReliefSize(); math.Abs(got-1.6) > 1e-9 {
		t.Errorf("corner relief size at a 4 mm gauge = %v, want 1.6 cm (4x)", got)
	}
}
