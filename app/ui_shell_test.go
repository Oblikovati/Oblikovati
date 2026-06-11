// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

func TestMarkingMenuDefaultsAndCustomization(t *testing.T) {
	s := NewSession()
	base := s.MarkingMenu(BaseEnvironment)
	if len(base.Quadrants) == 0 {
		t.Fatal("the base environment should have a default radial menu")
	}

	custom := wire.MarkingMenuView{
		Environment: SketchEnvironment,
		Quadrants:   []wire.MarkingMenuItem{{Quadrant: types.QuadrantNorth, CommandID: "X.Top"}},
		Overflow:    []string{"X.More"},
	}
	if err := s.SetMarkingMenu(custom); err != nil {
		t.Fatalf("SetMarkingMenu: %v", err)
	}
	got := s.MarkingMenu(SketchEnvironment)
	if len(got.Quadrants) != 1 || got.Quadrants[0].CommandID != "X.Top" || got.Overflow[0] != "X.More" {
		t.Fatalf("customized menu = %+v, want the replacement", got)
	}

	if err := s.SetMarkingMenu(wire.MarkingMenuView{Quadrants: []wire.MarkingMenuItem{
		{Quadrant: 9, CommandID: "X"},
	}}); err == nil {
		t.Error("an out-of-range quadrant should fail")
	}
	if err := s.SetMarkingMenu(wire.MarkingMenuView{Quadrants: []wire.MarkingMenuItem{
		{Quadrant: types.QuadrantNorth, CommandID: "A"},
		{Quadrant: types.QuadrantNorth, CommandID: "B"},
	}}); err == nil {
		t.Error("a duplicate quadrant should fail")
	}
}

func TestContextMenuInjectionMergesIntoBrowserMenu(t *testing.T) {
	s := NewSession()
	if err := s.Commands().Add(NewCommand("Sim.Analyze", "Analyze stress", "Sim",
		func(*Session) error { return nil })); err != nil {
		t.Fatalf("add command: %v", err)
	}
	if err := s.SetContextMenuItems("com.x.sim", "feature", []wire.ContextMenuItemSpec{
		{Label: "Analyze stress", CommandID: "Sim.Analyze"},
	}); err != nil {
		t.Fatalf("SetContextMenuItems: %v", err)
	}

	items := BrowserMenuFor(s, BrowserNode{Kind: "feature"})
	found := false
	for _, it := range items {
		if it.Label == "Analyze stress" {
			found = true
			if err := it.Invoke(s); err != nil {
				t.Fatalf("injected item Invoke: %v", err)
			}
		}
	}
	if !found {
		t.Fatalf("injected item missing from feature menu: %+v", items)
	}
	if len(BrowserMenuFor(s, BrowserNode{Kind: "body"})) != 0 {
		t.Error("a feature-kind injection must not leak into body menus")
	}

	// Clearing removes the injection.
	if err := s.SetContextMenuItems("com.x.sim", "feature", nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	for _, it := range BrowserMenuFor(s, BrowserNode{Kind: "feature"}) {
		if it.Label == "Analyze stress" {
			t.Error("cleared injection still present")
		}
	}
}

func TestSearchCommandsMatchesIdNameAlias(t *testing.T) {
	s := NewSession()
	noop := func(*Session) error { return nil }
	_ = s.Commands().Add(NewCommand("Model.Extrude", "Extrude", "Create", noop).WithAlias("E"))
	_ = s.Commands().Add(NewCommand("Sketch.Line", "Line", "Draw", noop))

	if hits := s.SearchCommands("extr"); len(hits) != 1 || hits[0].ID() != "Model.Extrude" {
		t.Errorf("SearchCommands(extr) = %v, want the Extrude command", hits)
	}
	if hits := s.SearchCommands("e"); len(hits) < 2 {
		t.Errorf("SearchCommands(e) = %d hits, want both (id/name/alias substrings)", len(hits))
	}
	if hits := s.SearchCommands("  "); hits != nil {
		t.Error("a blank query should return nothing")
	}
}

func TestObjectVisibilityGatesPickablePlanes(t *testing.T) {
	s := sessionWithPart(t)
	if !s.ObjectVisibility().WorkPlanes {
		t.Fatal("planes should default visible")
	}
	before := len(s.PickableWorkPlanes())
	if before == 0 {
		t.Fatal("expected origin planes to be pickable")
	}
	vis := s.ObjectVisibility()
	vis.WorkPlanes = false
	s.SetObjectVisibility(vis)
	if len(s.PickableWorkPlanes()) != 0 {
		t.Error("hidden work planes must not be pickable")
	}
}

func TestEnterExitSketchEmitsEnvironmentChanged(t *testing.T) {
	s := sessionWithPart(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	var got []EnvironmentChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e EnvironmentChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})

	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	s.ExitSketch()
	if len(got) != 2 || got[0].Environment != SketchEnvironment || got[1].Environment != BaseEnvironment {
		t.Fatalf("events = %+v, want sketch then base", got)
	}
}
