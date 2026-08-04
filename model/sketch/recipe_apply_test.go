// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// applyRecipe is the shared harness: apply r to a fresh sketch and return it with the created
// entities.
func applyRecipe(t *testing.T, r Recipe) (*Sketch, []Entity) {
	t.Helper()
	s := NewSketches().Add(XYPlane())
	ents, _, err := s.Apply(r, types.OverConstrainedApplyDriven)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return s, ents
}

func TestApplyCreatesEntitiesWithSharedPoints(t *testing.T) {
	r := Recipe{
		Points: []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 8)},
		Entities: []RecipeEntity{
			{Kind: RecipeLine, Points: []int{0, 1}},
			{Kind: RecipeLine, Points: []int{1, 2}},
		},
		Constraints: []RecipeConstraint{{Kind: SingleLineHorizontalKind, Entities: []int{0}}},
	}
	s, ents := applyRecipe(t, r)
	if len(ents) != 2 {
		t.Fatalf("entities = %d, want 2", len(ents))
	}
	l0, l1 := ents[0].(*Line), ents[1].(*Line)
	if l0.B != l1.A {
		t.Error("consecutive lines must share the middle point (structural coincidence)")
	}
	if a := s.AnalyzeConstraints(); a.Equations != 1 || a.Redundant != 0 {
		t.Errorf("eqs=%d redundant=%d, want 1 and 0", a.Equations, a.Redundant)
	}
}

func TestApplyMarksConstructionEntities(t *testing.T) {
	r := Recipe{
		Points:   []math.Point2{math.P2(0, 0), math.P2(10, 0)},
		Entities: []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}, Construction: true}},
	}
	_, ents := applyRecipe(t, r)
	c, ok := ents[0].(interface{ IsConstruction() bool })
	if !ok || !c.IsConstruction() {
		t.Error("Construction entity must be flagged construction")
	}
}

func TestApplyRollsBackOnBadConstraint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	r := Recipe{
		Points:      []math.Point2{math.P2(0, 0), math.P2(10, 0)},
		Entities:    []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}}},
		Constraints: []RecipeConstraint{{Kind: PerpendicularKind, Entities: []int{0, 9}}},
	}
	if _, _, err := s.Apply(r, types.OverConstrainedApplyDriven); err == nil {
		t.Fatal("Apply must reject an out-of-range entity index")
	}
	if n := len(s.Entities()); n != 0 {
		t.Errorf("rollback left %d entities, want 0", n)
	}
}

func TestApplyRejectsWrongPointArity(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	r := Recipe{
		Points:   []math.Point2{math.P2(0, 0)},
		Entities: []RecipeEntity{{Kind: RecipeLine, Points: []int{0}}},
	}
	_, _, err := s.Apply(r, types.OverConstrainedApplyDriven)
	if err == nil {
		t.Fatal("a line with one point must be rejected")
	}
	if n := len(s.Entities()); n != 0 {
		t.Errorf("rollback left %d entities, want 0", n)
	}
	// Entity count alone would miss a stray constrainable point: it is not an entity, but it
	// inflates the sketch's degrees of freedom for ever after.
	if a := s.AnalyzeConstraints(); a.Variables != 0 {
		t.Errorf("rollback left %d free variables, want 0 — a minted point leaked", a.Variables)
	}
}

// A recipe's curve endpoints are shared points, not standalone point entities — the same
// choice AddStraightSlot and AddPolygon make. Only a RecipePoint is listed in its own right.
func TestApplyListsOnlyStandalonePoints(t *testing.T) {
	_, ents := applyRecipe(t, Recipe{
		Points:   []math.Point2{math.P2(0, 0), math.P2(10, 0)},
		Entities: []RecipeEntity{{Kind: RecipeLine, Points: []int{0, 1}}},
	})
	if len(ents) != 1 {
		t.Errorf("entities = %d, want 1 line and no standalone endpoint entities", len(ents))
	}

	s, pointEnts := applyRecipe(t, Recipe{
		Points:   []math.Point2{math.P2(3, 4)},
		Entities: []RecipeEntity{{Kind: RecipePoint, Points: []int{0}}},
	})
	if len(pointEnts) != 1 {
		t.Fatalf("entities = %d, want 1", len(pointEnts))
	}
	if s.Points().Count() != 1 {
		t.Errorf("Points().Count() = %d, want 1 — a RecipePoint must be listed", s.Points().Count())
	}
}
