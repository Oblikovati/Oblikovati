// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestRelaxSolveDragsFullyDimensionedLine checks that RelaxSolve moves a fully-dimensioned
// line (which a normal DragSolve would hold rigid) to follow the cursor, and relaxes the
// driving distance dimension to the new measured length — Inventor's Relax Mode (#791).
func TestRelaxSolveDragsFullyDimensionedLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(3, 0)) // length 3 cm
	d, err := s.DimensionConstraints().AddDistance(a, b, "3 cm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	// A normal drag is blocked by the dimension: the pin fights the 3 cm constraint, so b
	// barely moves. Relax drops the dimension, so b follows the cursor and the dimension
	// updates to the new length.
	s.RelaxSolve([]PinTarget{{P: b, Target: math.P2(7, 0)}})

	if !b.Position().IsEqualTo(math.P2(7, 0), 1e-6) {
		t.Errorf("relaxed drag landed b at %v, want it pulled to (7,0)", b.Position())
	}
	if got := d.Measured(); got < 7-1e-6 || got > 7+1e-6 {
		t.Errorf("measured length after relax = %v, want 7", got)
	}
	// The dimension relaxed: its target now matches the dragged geometry (zero residual).
	for _, r := range d.Residuals() {
		if r > 1e-6 || r < -1e-6 {
			t.Errorf("dimension not relaxed: residual %v should be ~0", r)
		}
	}
}

// TestRelaxSolveKeepsGeometricConstraints checks that relaxing dimensions still honours the
// geometric constraints: a horizontal dimensioned line stays horizontal through a relax drag.
func TestRelaxSolveKeepsGeometricConstraints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(4, 0))
	s.GeometricConstraints().AddHorizontal(a, b)
	if _, err := s.DimensionConstraints().AddDistance(a, b, "4 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	s.RelaxSolve([]PinTarget{{P: b, Target: math.P2(9, 5)}})

	// Horizontal is a geometric constraint, so it is kept: a tracks b's Y.
	if d := b.Position().Y - a.Position().Y; d > 1e-6 || d < -1e-6 {
		t.Errorf("horizontal broken by relax: a.Y=%v b.Y=%v", a.Position().Y, b.Position().Y)
	}
	if b.Position().X <= 4 {
		t.Errorf("relaxed endpoint did not follow the cursor in +x: %v", b.Position())
	}
}

// TestRelaxSolveLeavesUnrelatedDimensionsAlone checks that a dimension whose geometry the
// drag does not disturb keeps its original expression (it is not flattened to a literal),
// so relax only rewrites the dimensions it actually moves.
func TestRelaxSolveLeavesUnrelatedDimensionsAlone(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(3, 0))
	dragged, _ := s.DimensionConstraints().AddDistance(a, b, "3 cm")

	// A second, disconnected dimensioned pair the drag cannot reach.
	c := s.Points().Add(math.P2(10, 10))
	e := s.Points().Add(math.P2(15, 10))
	far, _ := s.DimensionConstraints().AddDistance(c, e, "5 cm")
	farExpr := far.Parameter().Expression()

	s.RelaxSolve([]PinTarget{{P: b, Target: math.P2(8, 0)}})

	if far.Parameter().Expression() != farExpr {
		t.Errorf("untouched dimension expression changed to %q, want %q", far.Parameter().Expression(), farExpr)
	}
	if got := dragged.Measured(); got < 8-1e-6 || got > 8+1e-6 {
		t.Errorf("dragged dimension measured %v, want 8", got)
	}
}
