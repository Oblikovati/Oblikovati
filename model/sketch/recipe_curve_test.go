// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// The six shapes that were already well-formed before #2014 keep their degrees of freedom:
// their DOF is their intrinsic parameter count, so adding constraints here would be a
// regression, not an improvement.
func TestCurveRecipeDOFUnchanged(t *testing.T) {
	cases := []struct {
		name string
		r    Recipe
		dof  int
	}{
		{"line", LineRecipe(math.P2(0, 0), math.P2(10, 0)), 4},
		{"circle", CircleRecipe(math.P2(0, 0), 5), 3},
		{"arc", ArcRecipe(math.P2(0, 0), math.P2(5, 0), math.P2(0, 5), true), 5},
		{"ellipse", EllipseRecipe(math.P2(0, 0), math.V2(1, 0), 5, 3), 5},
		{"point", PointRecipe(math.P2(1, 2)), 2},
		{"spline", SplineRecipe([]math.Point2{math.P2(0, 0), math.P2(3, 4), math.P2(7, 1)}, true), 6},
	}
	for _, c := range cases {
		assertDOF(t, c.name, c.r, c.dof)
	}
}

// The polygon is the clearest case of #2014's root cause: AddPolygon already built a rigid
// hexagon, while the interactive tool rebuilt the same shape with no constraints at all.
func TestPolygonRecipeIsRigid(t *testing.T) {
	s := assertDOF(t, "hexagon", PolygonRecipe(math.P2(0, 0), math.P2(5, 0), 6, true), 4)
	if got := countConstruction(s); got != 1 {
		t.Errorf("construction entities = %d, want 1 circumcircle", got)
	}
}

func TestPolygonRecipeSideCounts(t *testing.T) {
	for _, sides := range []int{3, 5, 8, 12} {
		r := PolygonRecipe(math.P2(0, 0), math.P2(4, 0), sides, true)
		// sides lines + one construction circumcircle.
		if len(r.Entities) != sides+1 {
			t.Errorf("sides=%d: entities = %d, want %d", sides, len(r.Entities), sides+1)
		}
		assertDOF(t, "polygon", r, 4)
	}
}

// A polygon must stay regular under a drag — every vertex on one circle, every edge equal.
func TestPolygonStaysRegularUnderDrag(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	_, pts, err := s.Apply(PolygonRecipe(math.P2(0, 0), math.P2(5, 0), 6, true), types.OverConstrainedApplyDriven)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	center := pts[len(pts)-1]
	s.DragSolve([]PinTarget{{P: pts[0], Target: math.P2(8, 3)}})
	first := center.Position().DistanceTo(pts[0].Position())
	for i := 1; i < 6; i++ {
		if d := center.Position().DistanceTo(pts[i].Position()); absDiff(d, first) > 1e-6 {
			t.Errorf("vertex %d radius = %v, want %v — the polygon is not regular", i, d, first)
		}
	}
}

// absDiff is the unsigned difference of two measurements.
func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestCircleRecipeFieldIsDiameter(t *testing.T) {
	r := CircleRecipe(math.P2(0, 0), 5)
	if len(r.Fields) != 1 || r.Fields[0].Label != "Diameter" {
		t.Fatalf("fields = %+v, want a single Diameter", r.Fields)
	}
	if r.Fields[0].Value != 10 {
		t.Errorf("diameter = %v, want 10", r.Fields[0].Value)
	}
	if r.Fields[0].Dim.Kind != DiameterDim || r.Fields[0].Dim.Entity != 0 {
		t.Errorf("dim = %+v, want a DiameterDim on entity 0", r.Fields[0].Dim)
	}
}

func TestCircleRecipeRejectsZeroRadius(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if _, _, err := s.Apply(CircleRecipe(math.P2(1, 1), 0), types.OverConstrainedApplyDriven); err == nil {
		t.Fatal("a zero-radius circle must be rejected")
	}
}
