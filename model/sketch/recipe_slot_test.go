// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

func TestStraightSlotRecipeDOF(t *testing.T) {
	s := assertDOF(t, "straight slot", StraightSlotRecipe(math.P2(0, 0), math.P2(10, 0), 2), 5)
	if got := countConstruction(s); got != 1 {
		t.Errorf("construction entities = %d, want 1 centreline", got)
	}
}

func TestArcSlotRecipeDOF(t *testing.T) {
	r := ArcSlotRecipe(math.P2(0, 0), math.P2(10, 0), math.P2(0, 10), 2, true)
	s := assertDOF(t, "arc slot", r, 6)
	// Unlike the straight slot, no construction centreline: an arc entity carries an implicit
	// circularity relation that the rest of the shape already implies, so a centreline arc
	// would be permanently redundant while anchoring nothing (see ArcSlotRecipe).
	if got := countConstruction(s); got != 0 {
		t.Errorf("construction entities = %d, want 0", got)
	}
}

func TestStraightSlotRecipeFields(t *testing.T) {
	r := StraightSlotRecipe(math.P2(0, 0), math.P2(10, 0), 2)
	want := []string{"Length", "Angle", "Width"}
	if len(r.Fields) != len(want) {
		t.Fatalf("fields = %d, want %d", len(r.Fields), len(want))
	}
	for i, label := range want {
		if r.Fields[i].Label != label {
			t.Errorf("field %d = %q, want %q", i, r.Fields[i].Label, label)
		}
	}
	if r.Fields[0].Value != 10 {
		t.Errorf("length = %v, want 10", r.Fields[0].Value)
	}
	if r.Fields[2].Value != 2 {
		t.Errorf("width = %v, want 2", r.Fields[2].Value)
	}
}

// The slot's sides must stay parallel and tangent to its caps under a drag — the rigidity a
// bare 4-entity loop (DOF 10) did not have.
func TestStraightSlotStaysRigidUnderDrag(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents, pts, err := s.Apply(StraightSlotRecipe(math.P2(0, 0), math.P2(10, 0), 2), types.OverConstrainedApplyDriven)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	s.DragSolve([]PinTarget{{P: pts[4], Target: math.P2(14, 3)}})
	capA, capB := ents[1].(*Arc), ents[3].(*Arc)
	rA := capA.Center.Position().DistanceTo(capA.Start.Position())
	rB := capB.Center.Position().DistanceTo(capB.Start.Position())
	if stdmath.Abs(rA-rB) > 1e-9 {
		t.Errorf("cap radii diverged: %v vs %v — equal-radius is not holding", rA, rB)
	}
	// Width is one of the slot's five free parameters, so the drag may change it. What must
	// survive is the shape: each side stays perpendicular to the cap radius at the point they
	// share, which is what tangency means once the touch point is pinned.
	side := ents[0].(*Line)
	u := side.A.Position().VectorTo(side.B.Position())
	radial := capA.Start.Position().VectorTo(capA.Center.Position())
	cosPsi := u.Dot(radial) / (u.Length() * radial.Length())
	if stdmath.Abs(cosPsi) > 1e-9 {
		t.Errorf("side is not perpendicular to the cap radius (cos = %v) — tangency is not holding", cosPsi)
	}
}

// A zero-length slot has no direction to build from and must be rejected, not silently
// produce NaN geometry.
func TestStraightSlotRecipeRejectsCoincidentCentres(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	r := StraightSlotRecipe(math.P2(3, 3), math.P2(3, 3), 2)
	if _, _, err := s.Apply(r, types.OverConstrainedApplyDriven); err == nil {
		t.Fatal("coincident slot centres must be rejected")
	}
}
