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

// TestLineChainRecipeConnectsEverySegment: the continuous line tool previews its whole chain
// through this, so the segments must share endpoints — a chain of separate two-point lines would
// preview as a dashed-looking run of disconnected pieces and commit as unjoined geometry.
func TestLineChainRecipeConnectsEverySegment(t *testing.T) {
	r := LineChainRecipe([]math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 5), math.P2(0, 5)})

	if len(r.Entities) != 3 {
		t.Fatalf("got %d segments for 4 points, want 3", len(r.Entities))
	}
	for i, e := range r.Entities {
		if want := []int{i, i + 1}; e.Points[0] != want[0] || e.Points[1] != want[1] {
			t.Errorf("segment %d joins points %v, want %v — consecutive segments must share an endpoint", i, e.Points, want)
		}
	}
}

// TestLineChainFieldsDescribeTheSegmentBeingDrawn: the dynamic-input Length/Angle steer the
// segment at the cursor. Measuring the chain's FIRST segment instead would show the user a length
// they already committed and could no longer change.
func TestLineChainFieldsDescribeTheSegmentBeingDrawn(t *testing.T) {
	// Two segments: a long one already placed, then a short one at the cursor.
	r := LineChainRecipe([]math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 3)})

	if len(r.Fields) == 0 {
		t.Fatal("a chain must still offer its Length/Angle input")
	}
	if got := r.Fields[0].Value; got != 3 {
		t.Errorf("Length field = %v, want 3 — the field measures the segment at the cursor, not the first one", got)
	}
	if got := r.Fields[0].Dim.Points; got != [2]int{1, 2} {
		t.Errorf("Length dimension spans points %v, want the last segment {1 2}", got)
	}
}

// TestLineChainRecipeNeedsTwoPoints: one click is not a segment, and a recipe built from it would
// name a line with a single endpoint.
func TestLineChainRecipeNeedsTwoPoints(t *testing.T) {
	if r := LineChainRecipe([]math.Point2{math.P2(1, 1)}); len(r.Entities) != 0 {
		t.Errorf("a one-point chain produced %d entities, want none", len(r.Entities))
	}
}
