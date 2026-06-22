// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

// TestDeleteEntitiesRemovesEntityAndOrphanPoints is the regression guard for issue #1232:
// deleting a lone line drops the entity and the two endpoints no other entity references.
func TestDeleteEntitiesRemovesEntityAndOrphanPoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))

	if n := s.DeleteEntities([]Entity{l}); n != 1 {
		t.Fatalf("DeleteEntities returned %d, want 1", n)
	}
	if s.EntityCount() != 0 {
		t.Errorf("entities after delete = %d, want 0", s.EntityCount())
	}
	if got := len(s.AllPoints()); got != 0 {
		t.Errorf("points after delete = %d, want 0 (both endpoints pruned)", got)
	}
}

// TestDeleteEntitiesKeepsSharedPoints: a point shared with a surviving entity is not pruned.
func TestDeleteEntitiesKeepsSharedPoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.NewPoint(gmath.P2(0, 0))
	b := s.NewPoint(gmath.P2(4, 0))
	c := s.NewPoint(gmath.P2(4, 3))
	first := s.Lines().Add(a, b)
	second := s.Lines().Add(b, c) // shares b with first

	if n := s.DeleteEntities([]Entity{first}); n != 1 {
		t.Fatalf("DeleteEntities returned %d, want 1", n)
	}
	if s.EntityCount() != 1 {
		t.Errorf("entities after delete = %d, want 1 (second line survives)", s.EntityCount())
	}
	// b and c remain (second references them); a is pruned.
	if got := len(s.AllPoints()); got != 2 {
		t.Errorf("points after delete = %d, want 2 (shared b + c kept, a pruned)", got)
	}
	_ = second
}

// TestDeleteEntitiesDropsBoundConstraints: a dimension/constraint on a deleted point's
// variables is removed, so no dangling relation survives the delete (issue #1232).
func TestDeleteEntitiesDropsBoundConstraints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	if _, err := s.DimensionConstraints().AddDistance(l.A, l.B, "5 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	s.GeometricConstraints().AddHorizontal(l.A, l.B)
	if s.DimensionConstraints().Count() == 0 || s.GeometricConstraints().Count() == 0 {
		t.Fatal("setup: expected one dimension and one geometric constraint")
	}

	s.DeleteEntities([]Entity{l})

	if got := s.DimensionConstraints().Count(); got != 0 {
		t.Errorf("dimensions after delete = %d, want 0 (bound to the deleted line)", got)
	}
	if got := s.GeometricConstraints().Count(); got != 0 {
		t.Errorf("geometric constraints after delete = %d, want 0 (bound to the deleted line)", got)
	}
}

// TestDeleteEntitiesSkipsNilAndAbsent: nil and not-in-sketch entities are ignored, and the
// returned count reflects only what was actually removed.
func TestDeleteEntitiesSkipsNilAndAbsent(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	stranger := s.Lines().AddByTwoPoints(gmath.P2(2, 0), gmath.P2(3, 0))
	_ = s.DeleteEntities([]Entity{stranger}) // remove it so it is now absent

	if n := s.DeleteEntities([]Entity{nil, stranger, l}); n != 1 {
		t.Fatalf("DeleteEntities returned %d, want 1 (only l present)", n)
	}
}
