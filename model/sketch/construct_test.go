// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestCircleByThreePointsIsCircumcircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// Three points of the unit circle: (1,0),(0,1),(-1,0) ⇒ center (0,0), r=1.
	c, err := s.Circles().AddByThreePoints(gmath.P2(1, 0), gmath.P2(0, 1), gmath.P2(-1, 0))
	if err != nil {
		t.Fatalf("AddByThreePoints: %v", err)
	}
	if got := c.Center.Position(); math.Abs(float64(got.X)) > 1e-9 || math.Abs(float64(got.Y)) > 1e-9 {
		t.Errorf("center = %v, want (0,0)", got)
	}
	if got := float64(c.Radius); math.Abs(got-1) > 1e-9 {
		t.Errorf("radius = %v, want 1", got)
	}
}

func TestThreePointConstructorsRejectCollinear(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if _, err := s.Circles().AddByThreePoints(gmath.P2(0, 0), gmath.P2(1, 0), gmath.P2(2, 0)); err == nil {
		t.Error("circle through collinear points should error")
	}
	if _, err := s.Arcs().AddByThreePoints(gmath.P2(0, 0), gmath.P2(1, 0), gmath.P2(2, 0)); err == nil {
		t.Error("arc through collinear points should error")
	}
}

func TestArcByThreePointsWindsThroughMiddle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// Semicircle (2,0)→(0,2)→(-2,0): center (0,0), CCW.
	arc, err := s.Arcs().AddByThreePoints(gmath.P2(2, 0), gmath.P2(0, 2), gmath.P2(-2, 0))
	if err != nil {
		t.Fatalf("AddByThreePoints: %v", err)
	}
	if got := float64(arc.Radius()); math.Abs(got-2) > 1e-9 {
		t.Errorf("arc radius = %v, want 2", got)
	}
	if !arc.CounterClockwise {
		t.Error("arc through (2,0),(0,2),(-2,0) should wind CCW")
	}
}
