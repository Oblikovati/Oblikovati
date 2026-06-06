// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestSplitLineMakesTwoAtPoint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	parts, err := s.SplitLine(l, gmath.P2(2, 0))
	if err != nil {
		t.Fatalf("SplitLine: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("split made %d parts, want 2", len(parts))
	}
	// First part ends at (2,0); the two share that point.
	if p := l.B.Position(); math.Abs(float64(p.X)-2) > 1e-9 {
		t.Errorf("first part ends at X=%v, want 2", p.X)
	}
}

func TestTrimLineRemovesMiddleSegment(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(6, 0))
	// Two crossing verticals at x=2 and x=4.
	s.Lines().AddByTwoPoints(gmath.P2(2, -1), gmath.P2(2, 1))
	s.Lines().AddByTwoPoints(gmath.P2(4, -1), gmath.P2(4, 1))
	parts, err := s.TrimLine(l, gmath.P2(3, 0)) // pick the middle
	if err != nil {
		t.Fatalf("TrimLine: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("trim left %d parts, want 2 (the [0,2] and [4,6] stubs)", len(parts))
	}
	// The reshaped original keeps [0,2].
	if p := l.B.Position(); math.Abs(float64(p.X)-2) > 1e-9 {
		t.Errorf("kept stub ends at X=%v, want 2", p.X)
	}
}

func TestTrimLineFrontStub(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(6, 0))
	s.Lines().AddByTwoPoints(gmath.P2(4, -1), gmath.P2(4, 1))
	parts, err := s.TrimLine(l, gmath.P2(1, 0)) // pick before the only crossing
	if err != nil {
		t.Fatalf("TrimLine: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("front trim left %d parts, want 1", len(parts))
	}
	if p := l.A.Position(); math.Abs(float64(p.X)-4) > 1e-9 {
		t.Errorf("front-trimmed line starts at X=%v, want 4", p.X)
	}
}

func TestExtendLineReachesCrossing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	s.Lines().AddByTwoPoints(gmath.P2(3, -1), gmath.P2(3, 1)) // vertical at x=3
	if _, err := s.ExtendLine(l, true); err != nil {
		t.Fatalf("ExtendLine: %v", err)
	}
	if p := l.B.Position(); math.Abs(float64(p.X)-3) > 1e-9 {
		t.Fatalf("extended end at X=%v, want 3", p.X)
	}
}
