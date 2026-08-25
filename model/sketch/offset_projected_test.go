// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	gmath "oblikovati.org/math"
)

// TestOffsetProjectedClosedLoop is the #2158 follow-up: a projected closed perimeter (a face
// outline, stored as a sampled polyline) must offset as a closed loop of lines — the inner offset a
// user makes after projecting a face. Before it, OffsetEntity rejected the projected curve
// ("unsupported entity *sketch.ProjectedCurve").
func TestOffsetProjectedClosedLoop(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	sq := []gmath.Point2{gmath.P2(0, 0), gmath.P2(4, 0), gmath.P2(4, 4), gmath.P2(0, 4), gmath.P2(0, 0)}
	pc := s.RestoreProjectedCurve(nextID(), sq, "edge", "E1")

	before := s.Lines().Count()
	got, err := s.OffsetEntity(pc, -0.5) // inner offset (d<0 shrinks, like offsetCircle)
	if err != nil {
		t.Fatalf("OffsetEntity(closed projected curve): %v", err)
	}
	if got == nil {
		t.Fatal("offset returned a nil entity")
	}
	if s.Lines().Count() <= before {
		t.Errorf("no offset line geometry created (%d→%d lines)", before, s.Lines().Count())
	}
}

// TestOffsetProjectedOpenCurve: a projected open edge offsets as an open chain of lines.
func TestOffsetProjectedOpenCurve(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	open := []gmath.Point2{gmath.P2(0, 0), gmath.P2(4, 0), gmath.P2(4, 4)}
	pc := s.RestoreProjectedCurve(nextID(), open, "edge", "E2")

	before := s.Lines().Count()
	if _, err := s.OffsetEntity(pc, 0.5); err != nil {
		t.Fatalf("OffsetEntity(open projected curve): %v", err)
	}
	if s.Lines().Count() <= before {
		t.Error("no offset line geometry created for the open projected curve")
	}
}
