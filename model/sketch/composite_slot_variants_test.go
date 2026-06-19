// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

// assertClosedSlot fails unless ents is a 4-entity slot forming exactly one closed profile.
func assertClosedSlot(t *testing.T, s *Sketch, ents []Entity, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if len(ents) != 4 {
		t.Fatalf("%s = %d entities, want 4", what, len(ents))
	}
	if got := s.Profiles().Count(); got != 1 {
		t.Fatalf("%s profiles = %d, want 1 closed region", what, got)
	}
}

// TestStraightSlotByOverall (#149): the end tips are the overall extents, so the center-to-
// center length is overall − width and the slot still closes.
func TestStraightSlotByOverall(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents, err := s.AddStraightSlotByOverall(gmath.P2(0, 0), gmath.P2(8, 0), 2)
	assertClosedSlot(t, s, ents, err, "AddStraightSlotByOverall")
	// An overall length no greater than the width cannot form a slot.
	if _, err := s.AddStraightSlotByOverall(gmath.P2(0, 0), gmath.P2(1, 0), 2); err == nil {
		t.Error("overall length ≤ width should fail")
	}
}

// TestStraightSlotBySlotCenter (#149): the slot is symmetric about the given center.
func TestStraightSlotBySlotCenter(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents, err := s.AddStraightSlotBySlotCenter(gmath.P2(0, 0), gmath.P2(3, 0), 2)
	assertClosedSlot(t, s, ents, err, "AddStraightSlotBySlotCenter")
}

// TestArcSlotByCenterPoint (#149): center + start + sweep angle closes into a profile.
func TestArcSlotByCenterPoint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents, err := s.AddArcSlotByCenterPoint(gmath.P2(0, 0), gmath.P2(5, 0), math.Pi/2, 1)
	assertClosedSlot(t, s, ents, err, "AddArcSlotByCenterPoint")
}

// TestArcSlotByThreePoints (#149): three centerline points close into a profile; collinear
// points are rejected.
func TestArcSlotByThreePoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents, err := s.AddArcSlotByThreePoints(gmath.P2(5, 0), gmath.P2(0, 5), gmath.P2(-5, 0), 1)
	assertClosedSlot(t, s, ents, err, "AddArcSlotByThreePoints")

	s2 := NewSketches().Add(XYPlane())
	if _, err := s2.AddArcSlotByThreePoints(gmath.P2(0, 0), gmath.P2(2, 0), gmath.P2(4, 0), 1); err == nil {
		t.Error("collinear three points should fail")
	}
}
