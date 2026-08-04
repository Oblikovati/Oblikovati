// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// assertDOF applies r to a fresh sketch and pins its degrees of freedom and redundancy.
//
// DOF alone is not enough. A duplicated constraint can leave a sketch reporting DOF 0 while the
// solver settles on a degenerate, self-intersecting configuration that extrudes to an empty
// solid — so redundancy is asserted alongside it (#2014).
func assertDOF(t *testing.T, name string, r Recipe, wantDOF int) *Sketch {
	t.Helper()
	s := NewSketches().Add(XYPlane())
	if _, _, err := s.Apply(r, types.OverConstrainedApplyDriven); err != nil {
		t.Fatalf("%s: Apply: %v", name, err)
	}
	a := s.AnalyzeConstraints()
	if a.DOF != wantDOF {
		t.Errorf("%s: DOF = %d, want %d (vars=%d eqs=%d rank=%d)", name, a.DOF, wantDOF, a.Variables, a.Equations, a.Rank)
	}
	if a.Redundant != 0 {
		t.Errorf("%s: Redundant = %d, want 0", name, a.Redundant)
	}
	return s
}

// countConstruction reports how many of the sketch's entities are construction geometry.
func countConstruction(s *Sketch) int {
	n := 0
	for _, e := range s.Entities() {
		if c, ok := e.(interface{ IsConstruction() bool }); ok && c.IsConstruction() {
			n++
		}
	}
	return n
}

func TestRectangleRecipeDOF(t *testing.T) {
	assertDOF(t, "two-point rectangle", RectangleRecipe(math.P2(0, 0), math.P2(10, 8)), 4)
}

func TestThreePointRectangleRecipeDOF(t *testing.T) {
	r := ThreePointRectangleRecipe(math.P2(0, 0), math.P2(10, 0), math.P2(10, 8))
	assertDOF(t, "three-point rectangle", r, 5)
}

func TestCenterRectangleRecipeDOF(t *testing.T) {
	s := assertDOF(t, "centre rectangle", CenterRectangleRecipe(math.P2(0, 0), math.P2(5, 4)), 4)
	if got := countConstruction(s); got != 2 {
		t.Errorf("construction entities = %d, want 2 diagonals", got)
	}
}

// A rectangle must stay a rectangle when a corner is dragged — the defect #2014 reported.
func TestRectangleStaysSquareUnderDrag(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents, pts, err := s.Apply(RectangleRecipe(math.P2(0, 0), math.P2(10, 8)), types.OverConstrainedApplyDriven)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	s.DragSolve([]PinTarget{{P: pts[2], Target: math.P2(14, 11)}})
	// The solver converges to a tolerance, so edges are compared within one: a sheared
	// rectangle is off by the drag distance, not by 1e-9.
	const rigidTol = 1e-9
	bottom := ents[0].(*Line)
	if stdmath.Abs(float64(bottom.A.Y-bottom.B.Y)) > rigidTol {
		t.Errorf("bottom edge sheared: y %v vs %v — the rectangle is not rigid", bottom.A.Y, bottom.B.Y)
	}
	right := ents[1].(*Line)
	if stdmath.Abs(float64(right.A.X-right.B.X)) > rigidTol {
		t.Errorf("right edge sheared: x %v vs %v — the rectangle is not rigid", right.A.X, right.B.X)
	}
}

func TestRectangleRecipeFields(t *testing.T) {
	r := RectangleRecipe(math.P2(0, 0), math.P2(10, 8))
	if len(r.Fields) != 2 {
		t.Fatalf("fields = %d, want 2", len(r.Fields))
	}
	if r.Fields[0].Label != "Width" || r.Fields[1].Label != "Height" {
		t.Errorf("labels = %q/%q, want Width/Height", r.Fields[0].Label, r.Fields[1].Label)
	}
	if r.Fields[0].Value != 10 || r.Fields[1].Value != 8 {
		t.Errorf("values = %v/%v, want 10/8", r.Fields[0].Value, r.Fields[1].Value)
	}
	if r.Fields[0].Dim.Orientation != HorizontalDistance {
		t.Error("width must be a horizontal distance dimension")
	}
	if r.Fields[1].Dim.Orientation != VerticalDistance {
		t.Error("height must be a vertical distance dimension")
	}
}

// A rectangle drawn right-to-left or bottom-to-top must still come out rigid and correctly
// sized — the drag direction only decides which corner is which.
func TestRectangleRecipeHandlesInvertedDrag(t *testing.T) {
	r := RectangleRecipe(math.P2(10, 8), math.P2(0, 0))
	assertDOF(t, "inverted rectangle", r, 4)
	if r.Fields[0].Value != 10 || r.Fields[1].Value != 8 {
		t.Errorf("values = %v/%v, want positive 10/8", r.Fields[0].Value, r.Fields[1].Value)
	}
}
