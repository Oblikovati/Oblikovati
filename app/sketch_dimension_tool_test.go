// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2022: the dimension tool used to commit on its FIRST line pick, because a line supplies the
// two points a distance dimension needs. That made the two-line angle dimension unreachable
// from the UI even though addDimensionFor built one correctly when called directly — so these
// tests drive the real click path (s.Click), never addDimensionFor. A test that calls the apply
// function passes while the tool is broken.

// clickSketch clicks at the pixel showing sketch point p.
func clickSketch(t *testing.T, s *Session, p math.Point2) {
	t.Helper()
	x, y, ok := sketchToScreen(s, p)
	if !ok {
		t.Fatalf("sketch point %v does not project to the viewport", p)
	}
	s.Click(x, y)
}

// dimKinds lists the kinds of every dimension in the sketch, for readable failures.
func dimKinds(sk *sketch.Sketch) []sketch.DimKind {
	var out []sketch.DimKind
	for _, d := range sk.DimensionConstraints().All() {
		out = append(out, d.Kind())
	}
	return out
}

// TestAngleDimensionBetweenTwoLines is the headline regression: two lines must produce ONE
// angle dimension. Before #2022 the first click committed a length dimension and deactivated
// the tool, so this ended with a DistanceDim and the second line was never seen.
func TestAngleDimensionBetweenTwoLines(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 3))

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, midOf(l1))
	if s.ActiveTool() == nil {
		t.Fatal("the tool deactivated on the first line — the second line can never be picked")
	}
	clickSketch(t, s, midOf(l2))
	clickSketch(t, s, math.P2(2, 2)) // place the label

	if got := dimKinds(sk); len(got) != 1 || got[0] != sketch.AngleDim {
		t.Fatalf("two lines gave %v, want exactly [AngleDim]", got)
	}
}

// TestSingleLineLengthNeedsAPlacementClick pins the other half of the fix: one line is still a
// length dimension, but only once the user places it. If it committed on the pick, the angle
// case above could never work.
func TestSingleLineLengthNeedsAPlacementClick(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, midOf(l))
	if sk.DimensionConstraints().Count() != 0 {
		t.Fatalf("picking a line committed %d dimensions before placement", sk.DimensionConstraints().Count())
	}
	clickSketch(t, s, math.P2(2, 3)) // empty space: place it

	if got := dimKinds(sk); len(got) != 1 || got[0] != sketch.DistanceDim {
		t.Fatalf("one line + placement gave %v, want [DistanceDim]", got)
	}
	if s.ActiveTool() != nil {
		t.Error("the tool should deactivate once the dimension is placed")
	}
}

// TestDimensionLandsWhereItWasPlaced: the placement click is the text position, so a dimension
// appears where the user put it rather than at a derived default offset.
func TestDimensionLandsWhereItWasPlaced(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	want := math.P2(2, 3)

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, midOf(l))
	clickSketch(t, s, want)

	d := sk.DimensionConstraints().Item(0)
	got, placed := d.TextPoint()
	if !placed {
		t.Fatal("the placement click did not set a text point")
	}
	if got.DistanceTo(want) > 1e-6 {
		t.Errorf("dimension text at %v, want %v (where it was clicked)", got, want)
	}
}

// TestPointToPointDimension: two points give a distance, and neither point alone commits.
func TestPointToPointDimension(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 0))

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, a.Position())
	if sk.DimensionConstraints().Count() != 0 {
		t.Fatal("a single point committed a dimension")
	}
	clickSketch(t, s, b.Position())
	clickSketch(t, s, math.P2(2, 2))

	if got := dimKinds(sk); len(got) != 1 || got[0] != sketch.DistanceDim {
		t.Fatalf("two points gave %v, want [DistanceDim]", got)
	}
}

// TestStrayClickWithNothingPickedCommitsNothing: a click in empty space with an incomplete
// pick set must not create anything, or a mis-click would litter the sketch.
func TestStrayClickWithNothingPickedCommitsNothing(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	sk.Points().Add(math.P2(0, 0))

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, math.P2(5, 5)) // empty space, nothing picked
	clickSketch(t, s, math.P2(6, 6))

	if sk.DimensionConstraints().Count() != 0 {
		t.Errorf("stray clicks created %d dimensions, want 0", sk.DimensionConstraints().Count())
	}
	if s.ActiveTool() == nil {
		t.Error("the tool should stay armed after clicks that picked nothing")
	}
}

// TestCircleRadiusDimension keeps the radius path working through the new placement flow.
func TestCircleRadiusDimension(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(math.P2(0, 0), 2)

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, math.P2(2, 0)) // on the rim
	clickSketch(t, s, math.P2(4, 4)) // place

	if got := dimKinds(sk); len(got) != 1 || got[0] != sketch.RadiusDim {
		t.Fatalf("a circle gave %v, want [RadiusDim] (circle r=%v)", got, c.Radius)
	}
}

// TestArcRadiusDimension is the #2160-adjacent regression: the general Dimension tool must give an
// ARC a radius dimension, exactly as it does a circle. The auto-dimension branch matched only
// *Circle, so picking an arc fell through to the point/line path and produced no radius dimension —
// "radius dim not respected on arcs". An arc is a CircularCurve like a circle, so it dimensions the
// same way.
func TestArcRadiusDimension(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	// CCW arc of radius 2 centred at the origin, from (2,0) to (0,2) — its midpoint is (√2,√2).
	sk.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(2, 0), math.P2(0, 2), true)

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, math.P2(1.41421356, 1.41421356)) // on the arc
	clickSketch(t, s, math.P2(4, 4))                   // place

	if got := dimKinds(sk); len(got) != 1 || got[0] != sketch.RadiusDim {
		t.Fatalf("an arc gave %v, want [RadiusDim] — the arc must dimension its radius like a circle", got)
	}
}

// TestPlacementClickNearGeometryDoesNotAbsorbIt: once the pick set is complete and cannot be
// extended, clicking near other geometry places the label instead of silently adding that
// entity — otherwise placing a label over a busy sketch would build the wrong dimension.
func TestPlacementClickNearGeometryDoesNotAbsorbIt(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 0))
	bystander := sk.Points().Add(math.P2(2, 3))

	s.StartTool(NewDimensionTool())
	clickSketch(t, s, a.Position())
	clickSketch(t, s, b.Position())
	clickSketch(t, s, bystander.Position()) // place the label right on a third point

	got := dimKinds(sk)
	if len(got) != 1 || got[0] != sketch.DistanceDim {
		t.Fatalf("got %v, want [DistanceDim] — the bystander point must not join the pick set", got)
	}
	if refs := sk.DimensionConstraints().Item(0).Refs(); len(refs) != 2 {
		t.Errorf("dimension references %d entities, want 2 (the bystander was absorbed)", len(refs))
	}
}

// midOf is a line's midpoint — a pixel guaranteed to hit the line and not its endpoints.
func midOf(l *sketch.Line) math.Point2 {
	return l.A.Position().Midpoint(l.B.Position())
}
