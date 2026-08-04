// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
)

// The dual rule the issue describes: with geometry selected the button converts the selection;
// with nothing selected it toggles a creation mode for what is drawn next.
func TestFormatToggleConvertsSelection(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	s.Select(SketchEntityHandle{Entity: l})

	if n := s.ToggleConstruction(); n != 1 {
		t.Fatalf("converted = %d, want 1", n)
	}
	if !l.IsConstruction() {
		t.Error("the selected line must become construction")
	}
	if s.ConstructionMode() {
		t.Error("converting a selection must NOT also arm the creation mode")
	}
}

func TestFormatToggleArmsModeWithNoSelection(t *testing.T) {
	s, _ := sketchSession(t)
	if n := s.ToggleConstruction(); n != 0 {
		t.Fatalf("converted = %d, want 0 — nothing was selected", n)
	}
	if !s.ConstructionMode() {
		t.Error("with no selection the button must arm the creation mode")
	}
	s.ToggleConstruction()
	if s.ConstructionMode() {
		t.Error("a second press must disarm it")
	}
}

// The rule is the same for all four toggles, so they cannot drift apart.
func TestEveryFormatToggleFollowsTheDualRule(t *testing.T) {
	cases := []struct {
		name   string
		toggle func(*Session) int
		armed  func(*Session) bool
	}{
		{"construction", (*Session).ToggleConstruction, (*Session).ConstructionMode},
		{"centerline", (*Session).ToggleCenterline, (*Session).CenterlineMode},
		{"center point", (*Session).ToggleCenterPoint, (*Session).CenterPointMode},
		{"driven dimension", (*Session).ToggleDrivenDimension, (*Session).DrivenDimensionMode},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := sketchSession(t)
			if n := c.toggle(s); n != 0 {
				t.Fatalf("converted = %d with an empty selection, want 0", n)
			}
			if !c.armed(s) {
				t.Error("an empty selection must arm the creation mode")
			}
			c.toggle(s)
			if c.armed(s) {
				t.Error("pressing again must disarm it")
			}
		})
	}
}

// Armed construction mode marks what is drawn next, through the recipe commit path.
func TestConstructionModeMarksNewGeometry(t *testing.T) {
	s, sk := sketchSession(t)
	s.ToggleConstruction() // arm the mode
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0), math.P2(10, 8)}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	for i := 0; i < sk.Lines().Count(); i++ {
		if !sk.Lines().Item(i).IsConstruction() {
			t.Fatalf("line %d is not construction — the mode did not apply", i)
		}
	}
}

// With no mode armed, new geometry is ordinary — the mode must not leak.
func TestNoModeLeavesNewGeometryOrdinary(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0), math.P2(10, 8)}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if sk.Lines().Item(0).IsConstruction() {
		t.Error("an unarmed session must create ordinary geometry")
	}
}

// Centerline mode marks new lines; a centreline is always construction too.
func TestCenterlineModeMarksNewLines(t *testing.T) {
	s, sk := sketchSession(t)
	s.ToggleCenterline()
	tool := NewLineTool()
	tool.points = []math.Point2{math.P2(0, 0), math.P2(10, 0)}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	l := sk.Lines().Item(0)
	if !l.IsCenterline() || !l.IsConstruction() {
		t.Errorf("centerline=%v construction=%v, want both true", l.IsCenterline(), l.IsConstruction())
	}
}

// Centre-point mode marks points the Point tool places.
func TestCenterPointModeMarksNewPoints(t *testing.T) {
	s, sk := sketchSession(t)
	s.ToggleCenterPoint()
	tool := NewPointTool()
	tool.pts = []math.Point2{math.P2(3, 4)}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if sk.Points().Count() == 0 {
		t.Fatal("the point was not created")
	}
	if !sk.Points().Item(0).IsCenterPoint() {
		t.Error("centre-point mode must mark the new point")
	}
}

// Converting a selected dimension flips it between driving and driven.
func TestDrivenToggleConvertsSelectedDimension(t *testing.T) {
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(10, 0))
	d, err := sk.DimensionConstraints().AddDistance(a, b, "10 mm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	s.Select(SketchDimensionHandle{Dim: d})

	if n := s.ToggleDrivenDimension(); n != 1 {
		t.Fatalf("converted = %d, want 1", n)
	}
	if !d.Driven() {
		t.Error("the selected dimension must become driven")
	}
	if s.DrivenDimensionMode() {
		t.Error("converting must not also arm the mode")
	}
}

// Driven mode combined with #2014's redundancy demotion must yield one driven dimension, not an
// error — both paths set the same flag.
func TestDrivenModeWithLockedField(t *testing.T) {
	s, sk := sketchSession(t)
	s.ToggleDrivenDimension()
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	for _, r := range "10" {
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	tool.corners = append(tool.corners, math.P2(1, 0.8))
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	dims := sk.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("dimensions = %d, want 1", len(dims))
	}
	if !dims[0].Driven() {
		t.Error("driven mode must make the new dimension driven")
	}
}

// Driven mode touches only the dimensions this commit created, not ones already in the sketch.
func TestDrivenModeLeavesExistingDimensionsAlone(t *testing.T) {
	s, sk := sketchSession(t)
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(10, 0))
	existing, err := sk.DimensionConstraints().AddDistance(a, b, "10 mm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	s.ToggleDrivenDimension()

	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0), math.P2(10, 8)}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if existing.Driven() {
		t.Error("a dimension that predates the commit must be left alone")
	}
}
