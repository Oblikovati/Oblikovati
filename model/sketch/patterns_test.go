// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestRectangularPatternCountAndPositions(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	seed := s.Points().Add(gmath.P2(0, 0))
	copies, err := s.RectangularPattern([]Entity{seed}, gmath.V2(2, 0), 3, gmath.V2(0, 5), 2)
	if err != nil {
		t.Fatalf("RectangularPattern: %v", err)
	}
	if len(copies) != 5 { // 3×2 − 1 seed
		t.Fatalf("copies = %d, want 5", len(copies))
	}
	// Total points: seed + 5 copies = 6.
	if s.Points().Count() != 6 {
		t.Fatalf("points = %d, want 6", s.Points().Count())
	}
	// A copy exists at the far corner (2*2, 5*1) = (4,5).
	if !somesketchPointAt(s, gmath.P2(4, 5)) {
		t.Fatalf("no patterned point at (4,5)")
	}
}

func TestCircularPatternStepsAround(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	seed := s.Points().Add(gmath.P2(2, 0))
	copies, err := s.CircularPattern([]Entity{seed}, gmath.P2(0, 0), 4, 2*math.Pi)
	if err != nil {
		t.Fatalf("CircularPattern: %v", err)
	}
	if len(copies) != 3 { // 4 instances − seed
		t.Fatalf("copies = %d, want 3", len(copies))
	}
	// 360°/4 = 90° steps ⇒ copies at (0,2),(-2,0),(0,-2).
	for _, want := range []gmath.Point2{gmath.P2(0, 2), gmath.P2(-2, 0), gmath.P2(0, -2)} {
		if !somesketchPointAt(s, want) {
			t.Errorf("no patterned point near %v", want)
		}
	}
}

func TestPatternsRejectBadCounts(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p := s.Points().Add(gmath.P2(1, 0))
	if _, err := s.RectangularPattern([]Entity{p}, gmath.V2(1, 0), 0, gmath.V2(0, 1), 1); err == nil {
		t.Error("zero count1 should error")
	}
	if _, err := s.CircularPattern([]Entity{p}, gmath.P2(0, 0), 1, math.Pi); err == nil {
		t.Error("count 1 circular should error")
	}
}

// somesketchPointAt reports whether any constrainable point sits at q (within 1e-9).
func somesketchPointAt(s *Sketch, q gmath.Point2) bool {
	for _, p := range s.AllPoints() {
		if math.Abs(float64(p.X-q.X)) < 1e-9 && math.Abs(float64(p.Y-q.Y)) < 1e-9 {
			return true
		}
	}
	return false
}
