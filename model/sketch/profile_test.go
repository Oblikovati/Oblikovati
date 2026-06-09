// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// addRectangle adds a closed 4-line rectangle [x0,y0]–[x1,y1] sharing corner points.
func addRectangle(s *Sketch, x0, y0, x1, y1 float64) {
	c00 := s.Points().Add(math.P2(x0, y0))
	c10 := s.Points().Add(math.P2(x1, y0))
	c11 := s.Points().Add(math.P2(x1, y1))
	c01 := s.Points().Add(math.P2(x0, y1))
	s.Lines().Add(c00, c10)
	s.Lines().Add(c10, c11)
	s.Lines().Add(c11, c01)
	s.Lines().Add(c01, c00)
}

func TestRectangleYieldsOneClosedProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 4, 3)
	ps := s.Profiles()
	if ps.Count() != 1 {
		t.Fatalf("Profiles count = %d, want 1", ps.Count())
	}
	p := ps.Item(0)
	if !p.IsClosed() {
		t.Error("rectangle profile is not closed")
	}
	if len(p.OuterLoop().Entities()) != 4 || len(p.InnerLoops()) != 0 {
		t.Errorf("outer=%d inner=%d, want 4 / 0", len(p.OuterLoop().Entities()), len(p.InnerLoops()))
	}
}

func TestProfileContainsPointInsideOutsideAndInHole(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 10, 10)                   // outer
	s.Circles().AddByCenterRadius(math.P2(5, 5), 2) // hole at the center
	p := s.Profiles().Item(0)
	if !p.Contains(math.P2(1, 1)) {
		t.Error("a point inside the rectangle (clear of the hole) should be contained")
	}
	if p.Contains(math.P2(5, 5)) {
		t.Error("the hole center is in an inner loop and must NOT be contained")
	}
	if p.Contains(math.P2(20, 20)) {
		t.Error("a point outside the outer loop must not be contained")
	}
}

func TestOpenProfileContainsNothing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(4, 0))
	c := s.Points().Add(math.P2(4, 3))
	s.Lines().Add(a, b)
	s.Lines().Add(b, c) // an open chain — no enclosed region
	for i := 0; i < s.Profiles().Count(); i++ {
		if s.Profiles().Item(i).Contains(math.P2(2, 1)) {
			t.Error("an open profile should contain no point")
		}
	}
}

func TestNestedLoopsClassifyInnerAndOuter(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 10, 10)                   // outer
	s.Circles().AddByCenterRadius(math.P2(5, 5), 2) // hole inside it
	ps := s.Profiles()
	if ps.Count() != 1 {
		t.Fatalf("Profiles count = %d, want 1 region with a hole", ps.Count())
	}
	p := ps.Item(0)
	if len(p.OuterLoop().Entities()) != 4 {
		t.Errorf("outer loop entities = %d, want 4 (the rectangle)", len(p.OuterLoop().Entities()))
	}
	if len(p.InnerLoops()) != 1 {
		t.Fatalf("inner loops = %d, want 1 (the circle)", len(p.InnerLoops()))
	}
	if !p.InnerLoops()[0].IsClosed() {
		t.Error("inner loop should be closed")
	}
}

func TestMultiRegionProfiles(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 2, 2) // region A
	addRectangle(s, 5, 3, 7, 5) // region B (disjoint)
	if got := s.Profiles().Count(); got != 2 {
		t.Errorf("disjoint rectangles → %d profiles, want 2", got)
	}
}

func TestOpenChainIsOpenProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	c := s.Points().Add(math.P2(2, 1))
	s.Lines().Add(a, b)
	s.Lines().Add(b, c) // open chain, not closed
	ps := s.Profiles()
	if ps.Count() != 1 || ps.Item(0).IsClosed() {
		t.Fatalf("open chain → count=%d closed=%v, want 1 open profile", ps.Count(), ps.Item(0).IsClosed())
	}
}

func TestConstructionGeometryExcludedFromProfiles(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 4, 3)
	// A construction line across the rectangle must not affect region detection.
	d := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 3))
	d.SetConstruction(true)
	if s.Profiles().Count() != 1 {
		t.Errorf("construction geometry changed the profile count: %d", s.Profiles().Count())
	}
}

func TestAllProfilesAccessor(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	addRectangle(s, 0, 0, 1, 1)
	if len(s.Profiles().All()) != 1 {
		t.Error("All() did not return the detected profile")
	}
}
