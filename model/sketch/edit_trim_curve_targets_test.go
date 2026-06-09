// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

// nearPt asserts a point within tolerance.
func nearPt(t *testing.T, got gmath.Point2, wantX, wantY float64) {
	t.Helper()
	if math.Abs(float64(got.X)-wantX) > 1e-9 || math.Abs(float64(got.Y)-wantY) > 1e-9 {
		t.Errorf("point = %v, want (%v,%v)", got, wantX, wantY)
	}
}

// A circle crossed by a diameter line, trimmed on one side, becomes the opposite arc.
func TestTrimCircleBecomesComplementArc(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(0, 0), 2)
	s.Lines().AddByTwoPoints(gmath.P2(0, -3), gmath.P2(0, 3)) // crosses at (0,2),(0,-2)

	parts, err := s.TrimCircle(c, gmath.P2(2, 0)) // pick the right half to remove
	if err != nil {
		t.Fatalf("TrimCircle: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("trim circle made %d parts, want 1 arc", len(parts))
	}
	if s.Circles().Count() != 0 || s.Arcs().Count() != 1 {
		t.Fatalf("circles=%d arcs=%d, want 0 and 1", s.Circles().Count(), s.Arcs().Count())
	}
	arc := s.Arcs().Item(0)
	nearPt(t, arc.Center.Position(), 0, 0)
	if math.Abs(float64(arc.Radius())-2) > 1e-9 {
		t.Errorf("arc radius = %v, want 2", arc.Radius())
	}
	// Kept arc spans the left half: it passes through (-2,0).
	if !arcPassesThrough(arc, gmath.P2(-2, 0)) {
		t.Error("kept arc should pass through (-2,0) (the left half)")
	}
	if arcPassesThrough(arc, gmath.P2(2, 0)) {
		t.Error("removed right half should not be on the kept arc")
	}
}

// A circle with fewer than two crossings cannot be opened.
func TestTrimCircleNeedsTwoCrossings(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(0, 0), 2)
	if _, err := s.TrimCircle(c, gmath.P2(2, 0)); err == nil {
		t.Error("trimming a circle with no crossings should error")
	}
}

// An upper-semicircle arc trimmed past its single crossing keeps the front stub.
func TestTrimArcKeepsFrontStub(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc := s.Arcs().AddByCenterStartEnd(gmath.P2(0, 0), gmath.P2(2, 0), gmath.P2(-2, 0), true) // 0→π
	s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(0, 3))                                   // crosses at (0,2), t=0.5

	parts, err := s.TrimArc(arc, gmath.P2(-1.414213562, 1.414213562)) // pick at t≈0.75 (upper left)
	if err != nil {
		t.Fatalf("TrimArc: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("trim arc made %d parts, want 1", len(parts))
	}
	nearPt(t, arc.End.Position(), 0, 2) // tail removed; kept [0,0.5] ends at (0,2)
}

// An arc with two interior crossings, picked in the middle, splits into two arcs.
func TestTrimArcInteriorBiteMakesTwo(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc := s.Arcs().AddByCenterStartEnd(gmath.P2(0, 0), gmath.P2(2, 0), gmath.P2(-2, 0), true) // 0→π
	s.Lines().AddByTwoPoints(gmath.P2(1, -3), gmath.P2(1, 3))                                  // crosses at (1,√3), t=1/3
	s.Lines().AddByTwoPoints(gmath.P2(-1, -3), gmath.P2(-1, 3))                                // crosses at (-1,√3), t=2/3

	parts, err := s.TrimArc(arc, gmath.P2(0, 2)) // pick the top (t=0.5), between the crossings
	if err != nil {
		t.Fatalf("TrimArc: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("interior bite made %d parts, want 2", len(parts))
	}
	if s.Arcs().Count() != 2 {
		t.Fatalf("arcs after bite = %d, want 2", s.Arcs().Count())
	}
}

// arcPassesThrough reports whether q lies on the arc (on its circle and within its sweep).
func arcPassesThrough(a *Arc, q gmath.Point2) bool {
	ka := entityArc2d(a)
	if math.Abs(a.Center.Position().DistanceTo(q)-float64(a.Radius())) > 1e-6 {
		return false
	}
	_, ok := arcParamOf(ka, q)
	return ok
}
