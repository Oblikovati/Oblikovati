// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestAllConstraintKindsSurviveRoundTrip exercises restoreConstraint across every geometric
// constraint kind (and the typed ref resolvers it leans on): a sketch carrying one of each is
// serialized and restored, and the restored sketch must hold the same number of constraints.
// This both guards the round trip and covers the constraint-restore dispatcher.
func TestAllConstraintKindsSurviveRoundTrip(t *testing.T) {
	sc := NewSketches()
	s := sc.Add(XYPlane())
	g := s.GeometricConstraints()

	// Points, lines, circles, an arc and an ellipse to constrain.
	p := func(x, y float64) *Point { return s.NewPoint(math.P2(x, y)) }
	p0, p1, p2, p3 := p(0, 0), p(4, 0), p(4, 4), p(0, 4)
	pc, pm := p(2, 2), p(2, 6)
	l1 := s.Lines().Add(p0, p1)
	l2 := s.Lines().Add(p3, p2)
	l3 := s.Lines().Add(p0, p3)
	c1 := s.Circles().Add(pc, 1.5)
	c2 := s.Circles().Add(p1, 1.0)
	arc := s.Arcs().Add(p2, p1, p3, true)
	_ = arc

	g.AddCoincident(p0, s.NewPoint(math.P2(0, 0)))
	g.AddHorizontal(p0, p1)
	g.AddVertical(p0, p3)
	g.AddParallel(l1, l2)
	g.AddPerpendicular(l1, l3)
	g.AddCollinear(l1, l2)
	g.AddEqualLength(l1, l2)
	g.AddConcentric(c1, c2)
	g.AddEqualRadius(c1, c2)
	g.AddCircularTangent(c1, c2)
	g.AddPointOnLine(pm, l1)
	g.AddMidpoint(pc, l1)
	g.AddPointOnCircle(pm, c1)
	g.AddTangent(l1, c1)
	g.AddSymmetry(p1, p3, l3)
	g.AddFix(p0)

	want := g.Count()
	if want < 16 {
		t.Fatalf("set up %d constraints, expected at least 16 kinds", want)
	}

	out := roundTrip(t, sc)
	if got := out.GeometricConstraints().Count(); got != want {
		t.Errorf("constraint count after round trip = %d, want %d", got, want)
	}
}
