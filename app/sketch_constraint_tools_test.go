// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

func TestConstraintToolPicksThenApplies(t *testing.T) {
	s, sk := sketchSession(t)
	l1 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	l2 := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 4))
	// Tool-first: activate the Perpendicular tool, then feed two line picks.
	tool := constraintToolDefs[3].new() // Perpendicular
	s.StartTool(tool)
	if tool.Name() != "Perpendicular" || tool.Prompt(s) == "" {
		t.Error("constraint tool should expose a name and prompt")
	}
	s.feedPick(SketchEntityHandle{Entity: l1})
	if s.ActiveTool() == nil {
		t.Fatal("tool should stay active after one pick")
	}
	s.feedPick(SketchEntityHandle{Entity: l2}) // second pick → auto-applies + deactivates
	if s.ActiveTool() != nil {
		t.Error("tool should deactivate after applying the constraint")
	}
	d1 := l1.A.Position().VectorTo(l1.B.Position())
	d2 := l2.A.Position().VectorTo(l2.B.Position())
	if c := stdmath.Abs(d1.Dot(d2)) / (d1.Length() * d2.Length()); c > 1e-3 {
		t.Errorf("lines not perpendicular after tool applied: |cos| = %v", c)
	}
}

func TestConstraintToolRejectsWrongType(t *testing.T) {
	s, sk := sketchSession(t)
	p := sk.Points().Add(math.P2(1, 1))
	tool := constraintToolDefs[2].new() // Parallel: lines only
	s.StartTool(tool)
	if tool.Accepts(p) {
		t.Error("the parallel tool should not accept a point")
	}
	s.feedPick(SketchEntityHandle{Entity: p}) // rejected
	if len(tool.Picked()) != 0 {
		t.Errorf("rejected pick was recorded: %d picks", len(tool.Picked()))
	}
	if s.ActiveTool() == nil {
		t.Error("tool should stay active after a rejected pick")
	}
}

func TestConstraintToolHighlightsPickedAndCandidate(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(-2, 0), math.P2(2, 0)) // through origin
	tool := constraintToolDefs[3].new()                           // Perpendicular
	s.StartTool(tool)
	// Hovering the line (centre pixel maps to it) reports it as a valid candidate.
	if ent, ok := s.HoverCandidate(100, 100); !ok || ent != sketch.Entity(l) {
		t.Errorf("HoverCandidate = %v/%v, want the line", ent, ok)
	}
	// Once picked, it highlights as selected.
	s.feedPick(SketchEntityHandle{Entity: l})
	if !s.IsSelectedEntity(l) {
		t.Error("a picked entity should highlight as selected")
	}
}

func TestEveryConstraintToolAppliesViaPicks(t *testing.T) {
	p := func(sk *sketch.Sketch, x, y float64) sketch.Entity { return sk.Points().Add(math.P2(x, y)) }
	line := func(sk *sketch.Sketch, x0, y0, x1, y1 float64) sketch.Entity {
		return sk.Lines().AddByTwoPoints(math.P2(x0, y0), math.P2(x1, y1))
	}
	circle := func(sk *sketch.Sketch, cx, cy, r float64) sketch.Entity {
		return sk.Circles().AddByCenterRadius(math.P2(cx, cy), r)
	}
	cases := []struct {
		idx  int
		make func(*sketch.Sketch) []sketch.Entity
	}{
		{0, func(sk *sketch.Sketch) []sketch.Entity { return entList(p(sk, 0, 0), p(sk, 2, 1)) }},                   // Coincident
		{1, func(sk *sketch.Sketch) []sketch.Entity { return entList(line(sk, 0, 0, 4, 0), line(sk, 0, 1, 4, 1)) }}, // Collinear
		{2, func(sk *sketch.Sketch) []sketch.Entity { return entList(line(sk, 0, 0, 4, 0), line(sk, 0, 2, 4, 3)) }}, // Parallel
		{3, func(sk *sketch.Sketch) []sketch.Entity { return entList(line(sk, 0, 0, 4, 0), line(sk, 0, 0, 4, 4)) }}, // Perpendicular
		{4, func(sk *sketch.Sketch) []sketch.Entity { return entList(line(sk, 0, 0, 4, 2)) }},                       // Horizontal
		{5, func(sk *sketch.Sketch) []sketch.Entity { return entList(line(sk, 0, 0, 2, 4)) }},                       // Vertical
		{6, func(sk *sketch.Sketch) []sketch.Entity { return entList(line(sk, -4, 3, 4, 3), circle(sk, 0, 0, 2)) }}, // Tangent
		{7, func(sk *sketch.Sketch) []sketch.Entity { return entList(circle(sk, 0, 0, 2), circle(sk, 5, 3, 4)) }},   // Concentric
		{8, func(sk *sketch.Sketch) []sketch.Entity { return entList(line(sk, 0, 0, 3, 0), line(sk, 0, 5, 7, 5)) }}, // Equal
		{9, func(sk *sketch.Sketch) []sketch.Entity { return entList(p(sk, 1, 1)) }},                                // Fix
		{10, func(sk *sketch.Sketch) []sketch.Entity { // Symmetric: two points + the axis line
			return entList(p(sk, 3, 2), p(sk, 3, -2), line(sk, 0, 0, 1, 0))
		}},
		{11, func(sk *sketch.Sketch) []sketch.Entity { // Smooth: a line + an adjacent spline
			sp := sk.Splines().AddByControlPoints([]math.Point2{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 0}}, false)
			return entList(line(sk, -2, 1, 0, 1), sp)
		}},
	}
	for _, c := range cases {
		s, sk := sketchSession(t)
		tool := constraintToolDefs[c.idx].new()
		s.StartTool(tool)
		for _, e := range c.make(sk) {
			s.feedPick(SketchEntityHandle{Entity: e})
		}
		if s.ActiveTool() != nil {
			t.Errorf("%s tool did not apply and deactivate", constraintToolDefs[c.idx].name)
		}
	}
}

func TestDimensionToolApplies(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(newDimensionTool())
	s.feedPick(SketchEntityHandle{Entity: sk.Points().Add(math.P2(0, 0))})
	s.feedPick(SketchEntityHandle{Entity: sk.Points().Add(math.P2(3, 0))})
	if s.ActiveTool() != nil || sk.DimensionConstraints().Count() != 1 {
		t.Errorf("dimension tool: active=%v dims=%d, want inactive / 1",
			s.ActiveTool() != nil, sk.DimensionConstraints().Count())
	}
}

func TestHoverCandidateRejectsWrongType(t *testing.T) {
	s, sk := sketchSession(t)
	sk.Lines().AddByTwoPoints(math.P2(-2, -2), math.P2(2, 2)) // a line through origin
	s.StartTool(constraintToolDefs[7].new())                  // Concentric: circles only
	if _, ok := s.HoverCandidate(100, 100); ok {
		t.Error("a circles-only tool should not offer a line as a candidate")
	}
}

func TestCoincidentPointToPoint(t *testing.T) {
	s, sk := sketchSession(t)
	p1 := sk.Points().Add(math.P2(0, 0))
	p2 := sk.Points().Add(math.P2(2, 1))
	tool := constraintToolDefs[0].new() // Coincident
	s.StartTool(tool)
	tool.PickSnap(p1, SnapResult{Kind: SnapPoint, Point: p1.Position()})
	s.autoCommitAfterPick()
	if s.ActiveTool() == nil {
		t.Fatal("coincident should wait after one point")
	}
	tool.PickSnap(p2, SnapResult{Kind: SnapPoint, Point: p2.Position()})
	s.autoCommitAfterPick()
	if s.ActiveTool() != nil || !hasCoincident(sk) {
		t.Error("two points should apply a coincident constraint and deactivate")
	}
}

func TestCoincidentPointOnLine(t *testing.T) {
	s, sk := sketchSession(t)
	p := sk.Points().Add(math.P2(1, 3))
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	tool := constraintToolDefs[0].new()
	s.StartTool(tool)
	tool.PickSnap(p, SnapResult{Kind: SnapPoint, Point: p.Position()})
	s.autoCommitAfterPick()
	tool.PickSnap(l, SnapResult{Kind: SnapOnCurve, Point: math.P2(1, 0)}) // on the edge
	s.autoCommitAfterPick()
	if s.ActiveTool() != nil {
		t.Error("point + line should apply and deactivate")
	}
	if !hasConstraint(sk, func(c sketch.Constraint) bool { _, ok := c.(*sketch.PointOnLineConstraint); return ok }) {
		t.Error("expected a point-on-line constraint")
	}
}

func TestCoincidentToMidpoint(t *testing.T) {
	s, sk := sketchSession(t)
	p := sk.Points().Add(math.P2(0, 5))
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	tool := constraintToolDefs[0].new()
	s.StartTool(tool)
	tool.PickSnap(p, SnapResult{Kind: SnapPoint, Point: p.Position()})
	s.autoCommitAfterPick()
	tool.PickSnap(l, SnapResult{Kind: SnapMidpoint, Point: math.P2(2, 0)}) // midpoint snap
	s.autoCommitAfterPick()
	if !hasConstraint(sk, func(c sketch.Constraint) bool { _, ok := c.(*sketch.MidpointConstraint); return ok }) {
		t.Error("a midpoint snap should apply a midpoint constraint")
	}
}

func hasCoincident(sk *sketch.Sketch) bool {
	return hasConstraint(sk, func(c sketch.Constraint) bool { _, ok := c.(*sketch.CoincidentConstraint); return ok })
}

func hasConstraint(sk *sketch.Sketch, pred func(sketch.Constraint) bool) bool {
	for _, c := range sk.GeometricConstraints().All() {
		if pred(c) {
			return true
		}
	}
	return false
}
