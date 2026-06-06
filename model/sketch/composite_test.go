// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

func TestRectangleByCornersIsClosedProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	lines := s.AddRectangleByCorners(gmath.P2(0, 0), gmath.P2(4, 3))
	if len(lines) != 4 {
		t.Fatalf("rectangle = %d lines, want 4", len(lines))
	}
	if got := s.Profiles().Count(); got != 1 {
		t.Fatalf("rectangle profiles = %d, want 1 closed region", got)
	}
}

func TestRectangleByThreePointsRejectsZeroEdge(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if _, err := s.AddRectangleByThreePoints(gmath.P2(1, 1), gmath.P2(1, 1), gmath.P2(2, 2)); err == nil {
		t.Error("zero-length first edge should error")
	}
}

func TestPolygonInscribedVerticesOnRadius(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	lines, _, err := s.AddPolygon(gmath.P2(0, 0), gmath.P2(2, 0), 6, true)
	if err != nil {
		t.Fatalf("AddPolygon: %v", err)
	}
	if len(lines) != 6 {
		t.Fatalf("hexagon = %d lines, want 6", len(lines))
	}
	// Every vertex of an inscribed polygon lies on the circumradius (2 here). Check
	// the line endpoints (the polygon also owns a centre point at the origin now).
	for _, l := range lines {
		ln := l.(*Line)
		for _, p := range []*Point{ln.StartPoint(), ln.EndPoint()} {
			d := math.Hypot(float64(p.X), float64(p.Y))
			if math.Abs(d-2) > 1e-9 {
				t.Fatalf("vertex %v at radius %v, want 2", p.Position(), d)
			}
		}
	}
}

func TestPolygonNeedsThreeSides(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if _, _, err := s.AddPolygon(gmath.P2(0, 0), gmath.P2(1, 0), 2, true); err == nil {
		t.Error("2-sided polygon should error")
	}
}

func TestStraightSlotIsClosedProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	ents, err := s.AddStraightSlot(gmath.P2(0, 0), gmath.P2(6, 0), 2)
	if err != nil {
		t.Fatalf("AddStraightSlot: %v", err)
	}
	if len(ents) != 4 {
		t.Fatalf("slot = %d entities, want 4 (2 lines + 2 arcs)", len(ents))
	}
	if got := s.Profiles().Count(); got != 1 {
		t.Fatalf("slot profiles = %d, want 1 closed region", got)
	}
}
