// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestDragSolvePullsCoincidentPoint checks that dragging one of two coincident points re-solves
// the sketch so the other point follows — the constraint stays satisfied, unlike a blind translate.
func TestDragSolvePullsCoincidentPoint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(1, 1))
	b := s.Points().Add(math.P2(1, 1))
	co := s.GeometricConstraints().AddCoincident(a, b)

	s.DragSolve([]PinTarget{{P: a, Target: math.P2(4, 3)}})

	if !a.Position().IsEqualTo(math.P2(4, 3), 1e-6) {
		t.Errorf("dragged point a = %v, want it pulled to (4,3)", a.Position())
	}
	if !b.Position().IsEqualTo(a.Position(), 1e-6) {
		t.Errorf("coincident point b = %v, should follow a to %v", b.Position(), a.Position())
	}
	for _, r := range co.Residuals() {
		if r > 1e-6 || r < -1e-6 {
			t.Errorf("coincident constraint violated after drag (residual %v)", r)
		}
	}
}

// TestDragSolveRespectsHorizontalConstraint checks a horizontal line dragged by one endpoint
// keeps both endpoints at the same Y (the constraint slides the other endpoint), so the line
// follows the cursor within its remaining freedom rather than tilting.
func TestDragSolveRespectsHorizontalConstraint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(4, 0))
	s.GeometricConstraints().AddHorizontal(a, b)

	// Drag endpoint b up and to the right; horizontal forces a's Y to track b's Y.
	s.DragSolve([]PinTarget{{P: b, Target: math.P2(6, 2)}})

	if d := b.Position().Y - a.Position().Y; d > 1e-6 || d < -1e-6 {
		t.Errorf("horizontal violated: a.Y=%v b.Y=%v should match", a.Position().Y, b.Position().Y)
	}
	if b.Position().X <= 4 {
		t.Errorf("dragged endpoint did not follow the cursor in +x: %v", b.Position())
	}
}

// TestDragSolveFreePointMovesToTarget checks the unconstrained case still lands the point exactly
// on the cursor (so the existing free-drag behaviour is preserved by the solve path).
func TestDragSolveFreePointMovesToTarget(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p := s.Points().Add(math.P2(2, 2))
	s.DragSolve([]PinTarget{{P: p, Target: math.P2(-1, 5)}})
	if !p.Position().IsEqualTo(math.P2(-1, 5), 1e-6) {
		t.Errorf("free point drag = %v, want exactly (-1,5)", p.Position())
	}
}

func TestDefiningPointsByEntity(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pt := s.Points().Add(math.P2(0, 0))
	ln := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	ci := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	ar := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), true)
	el := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 2, 1)
	ea := s.EllipticalArcs().Add(math.P2(0, 0), math.V2(1, 0), 2, 1, 0, 1)

	cases := []struct {
		name string
		ent  Entity
		want int
	}{
		{"point", pt, 1},
		{"line", ln, 2},
		{"circle", ci, 1},
		{"arc", ar, 3},
		{"ellipse", el, 1},
		{"ellipticalArc", ea, 1},
		{"unknown", nil, 0},
	}
	for _, c := range cases {
		if got := DefiningPoints(c.ent); len(got) != c.want {
			t.Errorf("DefiningPoints(%s) = %d points, want %d", c.name, len(got), c.want)
		}
	}
	if got := DefiningPoints(pt); got[0] != pt {
		t.Error("DefiningPoints(point) should return the point itself")
	}
}
