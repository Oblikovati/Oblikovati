// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

// cornerLines builds two lines meeting at the origin: (0,0)→(4,0) and (0,0)→(0,4).
func cornerLines(s *Sketch) (*Line, *Line) {
	corner := s.newPoint(gmath.P2(0, 0))
	l1 := s.Lines().Add(corner, s.newPoint(gmath.P2(4, 0)))
	l2 := s.Lines().Add(corner, s.newPoint(gmath.P2(0, 4)))
	return l1, l2
}

func TestAddFilletInsertsTangentArc(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1, l2 := cornerLines(s)
	arc, err := s.AddFillet(l1, l2, 1)
	if err != nil {
		t.Fatalf("AddFillet: %v", err)
	}
	// The tangent arc of a right-angle corner with r=1 is centered at (1,1), radius 1.
	if c := arc.Center.Position(); math.Abs(float64(c.X)-1) > 1e-9 || math.Abs(float64(c.Y)-1) > 1e-9 {
		t.Errorf("arc center = %v, want (1,1)", c)
	}
	if r := float64(arc.Radius()); math.Abs(r-1) > 1e-9 {
		t.Errorf("arc radius = %v, want 1", r)
	}
	// l1 was trimmed back to its tangent point (1,0).
	if p := l1.A.Position(); math.Abs(float64(p.X)-1) > 1e-9 || math.Abs(float64(p.Y)) > 1e-9 {
		t.Errorf("l1 trimmed end = %v, want (1,0)", p)
	}
}

func TestAddChamferInsertsBevelLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1, l2 := cornerLines(s)
	bevel, err := s.AddChamfer(l1, l2, 1, 1)
	if err != nil {
		t.Fatalf("AddChamfer: %v", err)
	}
	// The bevel connects (1,0) and (0,1) ⇒ length √2.
	if got := float64(bevel.Length()); math.Abs(got-math.Sqrt2) > 1e-9 {
		t.Errorf("bevel length = %v, want √2", got)
	}
}

func TestAddFilletRejectsParallelLines(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	shared := s.newPoint(gmath.P2(0, 0))
	l1 := s.Lines().Add(shared, s.newPoint(gmath.P2(4, 0)))
	l2 := s.Lines().Add(shared, s.newPoint(gmath.P2(-4, 0))) // anti-parallel through the corner
	if _, err := s.AddFillet(l1, l2, 1); err == nil {
		t.Error("fillet of (anti)parallel lines should error")
	}
}

func TestAddFilletRejectsDisjointLines(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l1 := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	l2 := s.Lines().AddByTwoPoints(gmath.P2(5, 5), gmath.P2(6, 6))
	if _, err := s.AddFillet(l1, l2, 1); err == nil {
		t.Error("fillet of lines with no shared corner should error")
	}
}
