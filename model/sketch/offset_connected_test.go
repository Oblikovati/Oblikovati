// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// closedPath wraps ordered, already-connected entities as a closed traversal path (each entity walked
// in its natural direction), so a test drives OffsetConnectedLoop without going through loop detection.
func closedPath(ents ...Entity) *Path {
	return &Path{entities: profileEntities(ents), closed: true}
}

// openPath wraps ordered entities as an open traversal path.
func openPath(ents ...Entity) *Path {
	return &Path{entities: profileEntities(ents), closed: false}
}

func profileEntities(ents []Entity) []ProfileEntity {
	pes := make([]ProfileEntity, len(ents))
	for i, e := range ents {
		pes[i] = ProfileEntity{Entity: e}
	}
	return pes
}

// wantPoint fails unless p is within 1e-6 of (x, y).
func wantPoint(t *testing.T, label string, p gmath.Point2, x, y float64) {
	t.Helper()
	if stdmath.Abs(float64(p.X)-x) > 1e-6 || stdmath.Abs(float64(p.Y)-y) > 1e-6 {
		t.Errorf("%s = (%.6g, %.6g), want (%.6g, %.6g)", label, float64(p.X), float64(p.Y), x, y)
	}
}

func wantLine(t *testing.T, e Entity) *Line {
	t.Helper()
	l, ok := e.(*Line)
	if !ok {
		t.Fatalf("entity is %T, want *Line", e)
	}
	return l
}

func wantArc(t *testing.T, e Entity) *Arc {
	t.Helper()
	a, ok := e.(*Arc)
	if !ok {
		t.Fatalf("entity is %T, want *Arc", e)
	}
	return a
}

// TestOffsetConnectedRectangleInward: the unit square traversed counter-clockwise, offset by +0.25,
// shrinks to the inner square — confirming a POSITIVE distance offsets to the left of travel (inward
// for a CCW loop) and that the four corners meet at the inner-square vertices.
func TestOffsetConnectedRectangleInward(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l0 := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	l1 := s.Lines().AddByTwoPoints(gmath.P2(1, 0), gmath.P2(1, 1))
	l2 := s.Lines().AddByTwoPoints(gmath.P2(1, 1), gmath.P2(0, 1))
	l3 := s.Lines().AddByTwoPoints(gmath.P2(0, 1), gmath.P2(0, 0))

	ents, err := s.OffsetConnectedLoop(closedPath(l0, l1, l2, l3), 0.25)
	if err != nil {
		t.Fatalf("OffsetConnectedLoop(square, +0.25): %v", err)
	}
	if len(ents) != 4 {
		t.Fatalf("got %d offset entities, want 4", len(ents))
	}
	corners := [5][2]float64{{0.25, 0.25}, {0.75, 0.25}, {0.75, 0.75}, {0.25, 0.75}, {0.25, 0.25}}
	for i := range ents {
		l := wantLine(t, ents[i])
		wantPoint(t, "line start", l.A.Position(), corners[i][0], corners[i][1])
		wantPoint(t, "line end", l.B.Position(), corners[i+1][0], corners[i+1][1])
	}
}

// TestOffsetConnectedStadium: a closed stadium (two horizontal lines joined by two radius-1 semicircle
// arcs) offset inward by 0.25 — the lines move to y=0.25 and y=1.75, and each arc stays concentric
// with its original centre at radius 0.75.
func TestOffsetConnectedStadium(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	bottom := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	rightArc := s.Arcs().AddByCenterStartEnd(gmath.P2(4, 1), gmath.P2(4, 0), gmath.P2(4, 2), true)
	top := s.Lines().AddByTwoPoints(gmath.P2(4, 2), gmath.P2(0, 2))
	leftArc := s.Arcs().AddByCenterStartEnd(gmath.P2(0, 1), gmath.P2(0, 2), gmath.P2(0, 0), true)

	ents, err := s.OffsetConnectedLoop(closedPath(bottom, rightArc, top, leftArc), 0.25)
	if err != nil {
		t.Fatalf("OffsetConnectedLoop(stadium, +0.25): %v", err)
	}
	if len(ents) != 4 {
		t.Fatalf("got %d offset entities, want 4", len(ents))
	}
	assertHorizontalLine(t, wantLine(t, ents[0]), 0.25)
	assertHorizontalLine(t, wantLine(t, ents[2]), 1.75)
	assertConcentricArc(t, wantArc(t, ents[1]), 4, 1, 0.75)
	assertConcentricArc(t, wantArc(t, ents[3]), 0, 1, 0.75)
}

// assertHorizontalLine checks both endpoints lie on y within 1e-6.
func assertHorizontalLine(t *testing.T, l *Line, y float64) {
	t.Helper()
	if stdmath.Abs(float64(l.A.Y)-y) > 1e-6 || stdmath.Abs(float64(l.B.Y)-y) > 1e-6 {
		t.Errorf("offset line at y=(%.6g, %.6g), want both at y=%.6g", float64(l.A.Y), float64(l.B.Y), y)
	}
}

// assertConcentricArc checks the arc's centre and radius within 1e-6.
func assertConcentricArc(t *testing.T, a *Arc, cx, cy, r float64) {
	t.Helper()
	wantPoint(t, "arc centre", a.Center.Position(), cx, cy)
	if stdmath.Abs(float64(a.Radius())-r) > 1e-6 {
		t.Errorf("arc radius = %.6g, want %.6g", float64(a.Radius()), r)
	}
}

// TestOffsetConnectedOpenCorner: an open two-line elbow offset by 0.5 — the interior corner is
// mitered to the offset lines' intersection, while the two OUTER ends keep their own untrimmed offset
// endpoints (an open path does not wrap the last corner to the first).
func TestOffsetConnectedOpenCorner(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l0 := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	l1 := s.Lines().AddByTwoPoints(gmath.P2(4, 0), gmath.P2(4, 4))

	ents, err := s.OffsetConnectedLoop(openPath(l0, l1), 0.5)
	if err != nil {
		t.Fatalf("OffsetConnectedLoop(open elbow, +0.5): %v", err)
	}
	if len(ents) != 2 {
		t.Fatalf("got %d offset entities, want 2", len(ents))
	}
	first, second := wantLine(t, ents[0]), wantLine(t, ents[1])
	wantPoint(t, "outer start (untrimmed)", first.A.Position(), 0, 0.5)
	wantPoint(t, "mitered corner (first end)", first.B.Position(), 3.5, 0.5)
	wantPoint(t, "mitered corner (second start)", second.A.Position(), 3.5, 0.5)
	wantPoint(t, "outer end (untrimmed)", second.B.Position(), 3.5, 4)
}
