// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

// TestPaths3DOpenChain checks three connected segments form a single open path.
func TestPaths3DOpenChain(t *testing.T) {
	s := NewSketches3D().Add()
	s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(1, 1, 0))
	s.AddLine3D(gmath.P3(1, 1, 0), gmath.P3(1, 1, 2))

	paths := s.Paths3D()
	if len(paths) != 1 {
		t.Fatalf("got %d paths, want 1 connected chain", len(paths))
	}
	if paths[0].IsClosed() {
		t.Error("an open polyline should not be closed")
	}
	if paths[0].Count() != 4 {
		t.Errorf("path has %d points, want 4", paths[0].Count())
	}
}

// TestPaths3DClosedLoop checks a square loop is detected as a single closed path.
func TestPaths3DClosedLoop(t *testing.T) {
	s := NewSketches3D().Add()
	s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(4, 0, 0))
	s.AddLine3D(gmath.P3(4, 0, 0), gmath.P3(4, 3, 0))
	s.AddLine3D(gmath.P3(4, 3, 0), gmath.P3(0, 3, 0))
	s.AddLine3D(gmath.P3(0, 3, 0), gmath.P3(0, 0, 0))

	paths := s.Paths3D()
	if len(paths) != 1 || !paths[0].IsClosed() {
		t.Fatalf("got %d paths (closed=%v), want 1 closed loop", len(paths), paths[0].IsClosed())
	}
}

// TestProfiles3DPlanarSquare checks a planar closed loop yields a profile with the right
// area and normal, and that a non-planar closed loop is a path but not a profile.
func TestProfiles3DPlanarSquare(t *testing.T) {
	s := NewSketches3D().Add()
	// A 4×3 rectangle in the z=2 plane (area 12, normal ±Z).
	s.AddLine3D(gmath.P3(0, 0, 2), gmath.P3(4, 0, 2))
	s.AddLine3D(gmath.P3(4, 0, 2), gmath.P3(4, 3, 2))
	s.AddLine3D(gmath.P3(4, 3, 2), gmath.P3(0, 3, 2))
	s.AddLine3D(gmath.P3(0, 3, 2), gmath.P3(0, 0, 2))

	profiles := s.Profiles3D()
	if len(profiles) != 1 {
		t.Fatalf("got %d profiles, want 1", len(profiles))
	}
	if math.Abs(profiles[0].Area()-12) > 1e-9 {
		t.Errorf("profile area = %v, want 12", profiles[0].Area())
	}
	if len(profiles[0].Points()) != 4 {
		t.Errorf("profile has %d vertices, want 4", len(profiles[0].Points()))
	}
	if n := profiles[0].Normal(); math.Abs(math.Abs(float64(n.Z))-1) > 1e-9 {
		t.Errorf("profile normal = %v, want ±Z", n)
	}
	if !profiles[0].IsClosed() {
		t.Error("a profile is always closed")
	}
}

// TestProfiles3DNonPlanarLoopExcluded checks a closed but non-planar loop is a path, not
// a profile.
func TestProfiles3DNonPlanarLoopExcluded(t *testing.T) {
	s := NewSketches3D().Add()
	// A skew quadrilateral whose fourth point is lifted out of the first three's plane.
	s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(4, 0, 0))
	s.AddLine3D(gmath.P3(4, 0, 0), gmath.P3(4, 3, 0))
	s.AddLine3D(gmath.P3(4, 3, 0), gmath.P3(0, 3, 5)) // lifted in Z
	s.AddLine3D(gmath.P3(0, 3, 5), gmath.P3(0, 0, 0))

	if len(s.Paths3D()) != 1 || !s.Paths3D()[0].IsClosed() {
		t.Fatal("the skew loop should still be one closed path")
	}
	if got := len(s.Profiles3D()); got != 0 {
		t.Errorf("a non-planar loop should yield 0 profiles, got %d", got)
	}
}

// TestPaths3DDisjointChainsAndConstruction checks two disjoint chains are separate paths
// and that construction segments are excluded.
func TestPaths3DDisjointChainsAndConstruction(t *testing.T) {
	s := NewSketches3D().Add()
	s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	s.AddLine3D(gmath.P3(5, 5, 5), gmath.P3(6, 5, 5)) // disconnected
	con := s.AddLine3D(gmath.P3(9, 9, 9), gmath.P3(8, 8, 8))
	con.SetConstruction(true)
	// Non-segment entities (a standalone point, a circle) are skipped by chaining.
	s.AddPoint3D(gmath.P3(2, 2, 2))
	z, _ := gmath.NewUnitVector3(0, 0, 1)
	s.AddCircle3D(gmath.P3(3, 3, 3), z, 1)

	paths := s.Paths3D()
	if len(paths) != 2 {
		t.Fatalf("got %d paths, want 2 (construction + non-segments excluded)", len(paths))
	}
}

// TestPaths3DOutOfOrderAndReversed checks the walk extends both directions and matches a
// segment by either endpoint (segments added out of order, some reversed).
func TestPaths3DOutOfOrderAndReversed(t *testing.T) {
	s := NewSketches3D().Add()
	s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(2, 0, 0)) // middle
	s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(0, 0, 0)) // shares the head, reversed
	s.AddLine3D(gmath.P3(3, 0, 0), gmath.P3(2, 0, 0)) // shares the tail, reversed
	paths := s.Paths3D()
	if len(paths) != 1 || paths[0].Count() != 4 {
		t.Fatalf("out-of-order chain = %d paths / %d pts, want 1/4", len(paths), paths[0].Count())
	}
}

// TestProfiles3DDegenerateLoopExcluded checks a collinear closed chain yields no profile.
func TestProfiles3DDegenerateLoopExcluded(t *testing.T) {
	s := NewSketches3D().Add()
	s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(2, 0, 0))
	s.AddLine3D(gmath.P3(2, 0, 0), gmath.P3(0, 0, 0)) // out-and-back ⇒ no area
	if got := len(s.Profiles3D()); got != 0 {
		t.Errorf("a collinear loop should yield 0 profiles, got %d", got)
	}
}

// TestProfiles3DArcInChain checks an arc participates in chaining (a line
// closed by an arc forms a closed loop).
func TestProfiles3DArcInChain(t *testing.T) {
	s := NewSketches3D().Add()
	s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(2, 0, 0))
	// An arc back from (2,0,0) to (0,0,0) through (1,1,0) closes the loop.
	s.AddArc3D(gmath.P3(1, 0, 0), gmath.P3(2, 0, 0), gmath.P3(0, 0, 0), true)
	paths := s.Paths3D()
	if len(paths) != 1 || !paths[0].IsClosed() {
		t.Fatalf("line+arc should form one closed path, got %d (closed=%v)", len(paths), paths[0].IsClosed())
	}
}
