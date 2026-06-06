// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

// Scaling a circle about the origin scales both its center position and its radius.
func TestScaleEntitiesScalesPointsAndRadius(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(2, 0), 1)
	s.ScaleEntities([]Entity{c}, gmath.P2(0, 0), 2)
	if !c.Center.Position().IsEqualTo(gmath.P2(4, 0), 1e-9) {
		t.Errorf("center = %v, want (4,0)", c.Center.Position())
	}
	if math.Abs(float64(c.Radius)-2) > 1e-9 {
		t.Errorf("radius = %v, want 2", c.Radius)
	}
}

// Scaling a line scales its endpoint positions about the center.
func TestScaleEntitiesScalesLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(1, 0), gmath.P2(2, 0))
	s.ScaleEntities([]Entity{l}, gmath.P2(0, 0), 3)
	if !l.A.Position().IsEqualTo(gmath.P2(3, 0), 1e-9) || !l.B.Position().IsEqualTo(gmath.P2(6, 0), 1e-9) {
		t.Errorf("line = %v→%v, want (3,0)→(6,0)", l.A.Position(), l.B.Position())
	}
}

// A non-positive factor is rejected as a no-op.
func TestScaleEntitiesNonPositiveIsNoop(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(2, 0), 1)
	s.ScaleEntities([]Entity{c}, gmath.P2(0, 0), 0)
	if !c.Center.Position().IsEqualTo(gmath.P2(2, 0), 1e-9) || math.Abs(float64(c.Radius)-1) > 1e-9 {
		t.Error("zero factor should leave geometry unchanged")
	}
}

// MovePoints (the Stretch primitive) moves only the listed vertices, deforming geometry
// that shares them: stretching one line endpoint leaves the other fixed.
func TestMovePointsMovesOnlySelected(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	s.MovePoints([]*Point{l.B}, gmath.V2(0, 3))
	if !l.A.Position().IsEqualTo(gmath.P2(0, 0), 1e-9) {
		t.Errorf("A moved to %v, want it fixed at (0,0)", l.A.Position())
	}
	if !l.B.Position().IsEqualTo(gmath.P2(4, 3), 1e-9) {
		t.Errorf("B = %v, want (4,3)", l.B.Position())
	}
}
