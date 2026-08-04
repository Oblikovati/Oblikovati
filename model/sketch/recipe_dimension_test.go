// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// applyLocked applies a recipe with the given per-field expressions and returns the sketch.
func applyLocked(t *testing.T, r Recipe, locked []string, behavior types.OverConstrainedDimensionBehavior) *Sketch {
	t.Helper()
	s := NewSketches().Add(XYPlane())
	if _, _, err := s.ApplyWithFields(r, locked, behavior); err != nil {
		t.Fatalf("ApplyWithFields: %v", err)
	}
	return s
}

// A locked field becomes a driving dimension; the untouched one creates nothing. This is the
// contract the reference behaviour shows: a typed value states something, a tracked one does not.
func TestLockedFieldsBecomeDimensions(t *testing.T) {
	s := applyLocked(t, RectangleRecipe(math.P2(0, 0), math.P2(10, 8)),
		[]string{"10 mm", ""}, types.OverConstrainedApplyDriven)
	dims := s.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("dimensions = %d, want 1", len(dims))
	}
	if dims[0].Driven() {
		t.Error("a locked field must create a driving dimension")
	}
	if dims[0].Kind() != DistanceDim {
		t.Errorf("dimension kind = %v, want a distance", dims[0].Kind())
	}
}

// Every dimension kind a recipe can name is reachable.
func TestRecipeDimensionKinds(t *testing.T) {
	cases := []struct {
		name   string
		r      Recipe
		locked []string
		kind   DimKind
	}{
		{"distance", RectangleRecipe(math.P2(0, 0), math.P2(10, 8)), []string{"10 mm", ""}, DistanceDim},
		{"diameter", CircleRecipe(math.P2(0, 0), 5), []string{"20 mm"}, DiameterDim},
		{"radius", ArcRecipe(math.P2(0, 0), math.P2(5, 0), math.P2(0, 5), true), []string{"8 mm", ""}, RadiusDim},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := applyLocked(t, c.r, c.locked, types.OverConstrainedApplyDriven)
			dims := s.DimensionConstraints().All()
			if len(dims) != 1 {
				t.Fatalf("dimensions = %d, want 1", len(dims))
			}
			if dims[0].Kind() != c.kind {
				t.Errorf("kind = %v, want %v", dims[0].Kind(), c.kind)
			}
		})
	}
}

// An angle field that names no reference edge steers the shape but creates no dimension: a
// free-standing shape has nothing to measure its rotation against.
func TestSteeringAngleFieldCreatesNoDimension(t *testing.T) {
	r := LineRecipe(math.P2(0, 0), math.P2(10, 0))
	s := applyLocked(t, r, []string{"", "45 deg"}, types.OverConstrainedApplyDriven)
	if n := len(s.DimensionConstraints().All()); n != 0 {
		t.Errorf("dimensions = %d, want 0 — an angle with no reference edge states nothing", n)
	}
}

// A dimension that would make the sketch redundant is demoted to driven by default, and kept
// driving when the document asks for it. Redundancy is what silently produces degenerate solves,
// so the choice must be honoured rather than assumed.
func TestOverConstrainedBehaviourIsHonoured(t *testing.T) {
	// Two distance dimensions on the same rectangle edge pair: the second is redundant with
	// the first once the horizontal/vertical constraints are in place.
	redundant := RectangleRecipe(math.P2(0, 0), math.P2(10, 8))
	redundant.Fields = append(redundant.Fields, RecipeField{
		Label: "Width again", Unit: FieldLength, Value: 10,
		Dim: RecipeDim{
			Kind: DistanceDim, Points: [2]int{3, 2},
			Entity: NoRecipeEntity, Entity2: NoRecipeEntity, Orientation: HorizontalDistance,
		},
	})
	locked := []string{"10 mm", "", "10 mm"}

	driven := applyLocked(t, redundant, locked, types.OverConstrainedApplyDriven)
	dims := driven.DimensionConstraints().All()
	if len(dims) != 2 {
		t.Fatalf("dimensions = %d, want 2", len(dims))
	}
	if !dims[1].Driven() {
		t.Error("the redundant dimension must be demoted to driven by default")
	}

	driving := applyLocked(t, redundant, locked, types.OverConstrainedApplyDriving)
	if d := driving.DimensionConstraints().All(); d[1].Driven() {
		t.Error("with applyDriving the redundant dimension must stay driving")
	}
}

// An angle field that DOES name two edges creates a real angle dimension. No shipped recipe
// does yet — every angle today steers only — but the path must work for one that does.
func TestAngleFieldWithTwoEdgesCreatesDimension(t *testing.T) {
	r := Recipe{
		Points: []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 8)},
		Entities: []RecipeEntity{
			{Kind: RecipeLine, Points: []int{0, 1}},
			{Kind: RecipeLine, Points: []int{1, 2}},
		},
		Fields: []RecipeField{{
			Label: "Corner", Unit: FieldAngle, Value: 1.5708,
			Dim: RecipeDim{Kind: AngleDim, Points: [2]int{0, 0}, Entity: 0, Entity2: 1},
		}},
	}
	s := applyLocked(t, r, []string{"90 deg"}, types.OverConstrainedApplyDriven)
	dims := s.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("dimensions = %d, want 1", len(dims))
	}
	if dims[0].Kind() != AngleDim {
		t.Errorf("kind = %v, want an angle", dims[0].Kind())
	}
}

// A concentric constraint is dispatchable even though no shipped recipe uses one today, so a
// future recipe naming it does not fail at runtime.
func TestConcentricRecipeConstraintApplies(t *testing.T) {
	r := Recipe{
		Points: []math.Point2{math.P2(0, 0)},
		Entities: []RecipeEntity{
			{Kind: RecipeCircle, Points: []int{0}, Radius: 5},
			{Kind: RecipeCircle, Points: []int{0}, Radius: 3},
		},
		Constraints: []RecipeConstraint{{Kind: ConcentricKind, Entities: []int{0, 1}}},
	}
	s := NewSketches().Add(XYPlane())
	if _, _, err := s.Apply(r, types.OverConstrainedApplyDriven); err != nil {
		t.Fatalf("Apply: %v", err)
	}
}

// An unsupported constraint kind is reported, not silently skipped — a recipe that asks for a
// relation the dispatcher does not know is a programming error, and rolls the shape back.
func TestUnsupportedConstraintKindIsRejected(t *testing.T) {
	r := Recipe{
		Points:      []math.Point2{math.P2(0, 0), math.P2(10, 0)},
		Entities:    []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}}},
		Constraints: []RecipeConstraint{{Kind: SmoothKind, Entities: []int{0}}},
	}
	s := NewSketches().Add(XYPlane())
	if _, _, err := s.Apply(r, types.OverConstrainedApplyDriven); err == nil {
		t.Fatal("an unsupported constraint kind must be reported")
	}
	if n := len(s.Entities()); n != 0 {
		t.Errorf("rollback left %d entities, want 0", n)
	}
}
