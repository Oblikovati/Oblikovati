// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

func TestProfileAreaOfRectangle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	ps := s.Profiles()
	if ps.Count() != 1 {
		t.Fatalf("profiles = %d, want 1", ps.Count())
	}
	if got := ps.Item(0).Area(); math.Abs(got-12) > 1e-9 {
		t.Fatalf("4×3 rectangle area = %v, want 12", got)
	}
}

func TestProfileAreaSubtractsHole(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(10, 10)) // outer 100
	// A concentric inner square (hole) 2..8 ⇒ 36; net 64.
	s.AddRectangleByCorners(gmath.P2(2, 2), gmath.P2(8, 8))
	ps := s.Profiles()
	// The annulus profile (outer with the inner hole) should report ~64.
	var withHole *Profile
	for _, p := range ps.All() {
		if len(p.InnerLoops()) == 1 {
			withHole = p
		}
	}
	if withHole == nil {
		t.Fatalf("no profile with a hole found among %d profiles", ps.Count())
	}
	if got := withHole.Area(); math.Abs(got-64) > 1e-9 {
		t.Fatalf("annulus area = %v, want 64 (100 − 36)", got)
	}
}
