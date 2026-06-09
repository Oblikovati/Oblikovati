// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

func TestAddPolylineClosedLIsOneProfile(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// An L outline: 6 vertices, one reflex corner at (1.5,1.5).
	lInfo := []gmath.Point2{
		gmath.P2(0, 0), gmath.P2(3, 0), gmath.P2(3, 1.5),
		gmath.P2(1.5, 1.5), gmath.P2(1.5, 3), gmath.P2(0, 3),
	}
	lines, err := s.AddPolyline(lInfo, true)
	if err != nil {
		t.Fatalf("AddPolyline(closed L): %v", err)
	}
	if len(lines) != 6 {
		t.Fatalf("closed L = %d lines, want 6", len(lines))
	}
	if got := s.Profiles().Count(); got != 1 {
		t.Fatalf("closed L profiles = %d, want 1 closed region", got)
	}
}

func TestAddPolylineOpenIsAChain(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	// An open chain of 3 points makes 2 connected segments (no closing edge).
	lines, err := s.AddPolyline([]gmath.Point2{gmath.P2(0, 0), gmath.P2(2, 0), gmath.P2(2, 2)}, false)
	if err != nil {
		t.Fatalf("AddPolyline(open): %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("open 3-point polyline = %d lines, want 2", len(lines))
	}
	// Endpoints are shared: the two segments meet at the middle point (one connected chain).
	mid1 := lines[0].(*Line).EndPoint()
	mid2 := lines[1].(*Line).StartPoint()
	if mid1 != mid2 {
		t.Fatalf("open polyline segments do not share the middle point (chain is disconnected)")
	}
}

func TestAddPolylineRejectsTooFewPoints(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	if _, err := s.AddPolyline([]gmath.Point2{gmath.P2(0, 0)}, false); err == nil {
		t.Error("open polyline with 1 point should error")
	}
	if _, err := s.AddPolyline([]gmath.Point2{gmath.P2(0, 0), gmath.P2(1, 0)}, true); err == nil {
		t.Error("closed polyline with 2 points should error")
	}
}

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
