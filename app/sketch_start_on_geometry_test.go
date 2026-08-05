// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Starting a shape ON existing geometry must JOIN it, not merely land on it. Point inference was
// reachable only through the line path, so a rectangle or an arc begun on an existing point had
// its coordinates snapped and nothing recorded why: drag the other geometry and the two came
// apart. Every creation tool routes through the shared recipe commit now, which infers (#2030).

// existingCornerSession returns a session editing a sketch that already holds one line, plus the
// screen position of that line's far endpoint — the point a new shape will be started on.
func existingCornerSession(t *testing.T) (*Session, *sketch.Sketch, float64, float64) {
	t.Helper()
	s, sk := sketchSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(120, 40)
	_ = s.PressKey(KeyEvent{Key: "Escape"})
	if sk.Lines().Count() != 1 {
		t.Fatalf("setup: got %d lines, want 1", sk.Lines().Count())
	}
	return s, sk, 120, 40
}

// coincidenceCount is how many coincident relations the sketch carries.
func coincidenceCount(sk *sketch.Sketch) int {
	n := 0
	for _, g := range sk.ConstraintGlyphs() {
		if g.Kind == sketch.CoincidentKind {
			n++
		}
	}
	return n
}

// TestRectangleStartedOnAPointJoinsIt is the reported case: the rectangle's first corner is
// clicked onto the line's endpoint.
func TestRectangleStartedOnAPointJoinsIt(t *testing.T) {
	s, sk, px, py := existingCornerSession(t)
	before := coincidenceCount(sk)

	s.StartTool(NewRectangleTool())
	s.Click(px, py)
	s.Click(px+60, py+60)

	if got := coincidenceCount(sk); got <= before {
		t.Errorf("coincidences %d → %d: the rectangle corner was placed on the line's endpoint without joining it", before, got)
	}
}

// TestArcStartedOnAPointJoinsIt is the other reported case. The arc tool did not even use the
// shared recipe commit, so it could never have inferred anything.
func TestArcStartedOnAPointJoinsIt(t *testing.T) {
	s, sk, px, py := existingCornerSession(t)
	before := coincidenceCount(sk)

	s.StartTool(NewArcTool())
	s.Click(px, py)       // start, on the line's endpoint
	s.Click(px+80, py)    // end
	s.Click(px+40, py-40) // a point on the arc

	if sk.Arcs().Count() != 1 {
		t.Fatalf("got %d arcs, want 1", sk.Arcs().Count())
	}
	if got := coincidenceCount(sk); got <= before {
		t.Errorf("coincidences %d → %d: the arc start was placed on the line's endpoint without joining it", before, got)
	}
}

// TestCircleCentredOnAPointJoinsIt: the same gap ran through every tool that bypassed the recipe
// commit, not only the two that were reported.
func TestCircleCentredOnAPointJoinsIt(t *testing.T) {
	s, sk, px, py := existingCornerSession(t)
	before := coincidenceCount(sk)

	s.StartTool(NewCircleTool())
	s.Click(px, py)
	s.Click(px+50, py)

	if sk.Circles().Count() != 1 {
		t.Fatalf("got %d circles, want 1", sk.Circles().Count())
	}
	if got := coincidenceCount(sk); got <= before {
		t.Errorf("coincidences %d → %d: the circle centre did not join the point it was placed on", before, got)
	}
}

// TestShapeAwayFromGeometryGainsNoConstraint: inference must only fire where the user actually
// placed a point on something. A shape drawn in open space picking up a coincidence would be far
// worse than the bug — it would silently tie unrelated geometry together.
func TestShapeAwayFromGeometryGainsNoConstraint(t *testing.T) {
	s, sk, _, _ := existingCornerSession(t)
	before := coincidenceCount(sk)

	s.StartTool(NewRectangleTool())
	s.Click(300, 300)
	s.Click(380, 380)

	if got := coincidenceCount(sk); got != before {
		t.Errorf("coincidences %d → %d: a rectangle drawn in open space invented a relation", before, got)
	}
}

// TestInferenceOffLeavesShapesUnconstrained: the point-inference preference has to reach the new
// path too, or a user who turned it off would still get constraints they did not ask for.
func TestInferenceOffLeavesShapesUnconstrained(t *testing.T) {
	s, sk, px, py := existingCornerSession(t)
	opts := s.SketchInferenceOptions()
	opts.InferEnabled = false
	s.SetSketchInferenceOptions(opts)
	before := coincidenceCount(sk)

	s.StartTool(NewRectangleTool())
	s.Click(px, py)
	s.Click(px+60, py+60)

	if got := coincidenceCount(sk); got != before {
		t.Errorf("coincidences %d → %d with point inference disabled", before, got)
	}
}

// TestCommittedShapeMatchesItsPreview: routing the commits through their recipes closes the drift
// the recipe layer exists to prevent — the arc tool previewed one shape and committed another
// code path's. They must agree.
func TestCommittedShapeMatchesItsPreview(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewArcTool()
	s.StartTool(tool)
	s.Click(40, 40)
	s.Click(140, 40)
	cursor, ok := s.CursorSketchPoint(90, 100)
	if !ok {
		t.Fatal("the cursor is not over the sketch plane")
	}
	r, ok := tool.PendingRecipe(s, cursor, nil)
	if !ok {
		t.Fatal("no arc preview with two points placed")
	}

	s.Click(90, 100)

	if sk.Arcs().Count() != 1 {
		t.Fatalf("got %d arcs, want 1", sk.Arcs().Count())
	}
	// Compare the arcs' DEFINING points, not their sampled outlines: the two are tessellated at
	// different densities, so a chord-error difference would say nothing about the geometry.
	arc := sk.Arcs().Item(0)
	for i, got := range []math.Point2{arc.Center.Position(), arc.Start.Position(), arc.End.Position()} {
		if d := got.DistanceTo(r.Points[i]); d > 1e-9 {
			t.Errorf("committed arc point %d at %v, previewed at %v — preview and commit disagree", i, got, r.Points[i])
		}
	}
}

// TestPointToolStillPlacesEveryPoint guards the tool whose commit creates SEVERAL entities: it
// now applies one recipe per point, and dropping the loop would silently place only one.
func TestPointToolStillPlacesEveryPoint(t *testing.T) {
	s, sk := sketchSession(t)
	before := sk.Points().Count()
	tool := NewPointTool()
	s.StartTool(tool)
	tool.pts = []math.Point2{{X: 1, Y: 1}, {X: 2, Y: 2}, {X: 3, Y: 3}}

	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := sk.Points().Count() - before; got != 3 {
		t.Errorf("placed %d points, want 3", got)
	}
}
