// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// typeInto feeds characters into the active in-place input field.
func typeInto(s *Session, text string) {
	for _, r := range text {
		s.PlacementFieldInput(r)
	}
}

// The fields a shape offers come from its recipe, so each tool shows its own quantities rather
// than a fixed generic pair.
func TestPlacementFieldsComeFromTheRecipe(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	views := s.PlacementFields()
	if len(views) != 2 {
		t.Fatalf("fields = %d, want 2", len(views))
	}
	if views[0].Label != "Width" || views[1].Label != "Height" {
		t.Errorf("labels = %q/%q, want Width/Height", views[0].Label, views[1].Label)
	}
	if !views[0].Active || views[1].Active {
		t.Error("the first field starts active")
	}
}

// Typing a width and Tabbing locks it: the drag then changes only the height. This is the
// behaviour the reference image shows, with the locked field carrying a padlock.
func TestLockedFieldFreezesTheDrag(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60) // establish the field list
	typeInto(s, "10")
	s.PlacementFieldTab()

	r, ok := s.ActiveToolRecipe(math.P2(99, 8))
	if !ok {
		t.Fatal("recipe expected")
	}
	if r.Fields[0].Value != 1 { // 10 mm == 1 cm in model units
		t.Errorf("locked width = %v, want 1 cm (10 mm typed)", r.Fields[0].Value)
	}
	if r.Fields[1].Value != 8 {
		t.Errorf("height = %v, want 8 — an unlocked field still tracks the cursor", r.Fields[1].Value)
	}
}

// A locked field becomes a driving dimension; an untyped one creates nothing. That contract is
// the whole point of the in-place input.
func TestLockedFieldCreatesOneDrivingDimension(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	typeInto(s, "10")
	s.PlacementFieldTab()
	tool.corners = append(tool.corners, math.P2(1, 0.8))
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	dims := sk.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("dimensions = %d, want 1 (only the locked field)", len(dims))
	}
	if dims[0].Driven() {
		t.Error("a locked field must create a DRIVING dimension")
	}
}

// Nothing typed means nothing dimensioned.
func TestUntouchedFieldsCreateNoDimensions(t *testing.T) {
	s, sk := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0), math.P2(1, 0.8)}
	s.StartTool(tool)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if n := len(sk.DimensionConstraints().All()); n != 0 {
		t.Errorf("dimensions = %d, want 0 — a field that tracked the cursor states nothing", n)
	}
}

// The parameter engine is unit-strict: a bare "10" would silently mean 10 cm, the kernel's
// length unit, rather than 10 of the document's unit.
func TestLockedFieldExpressionCarriesItsUnit(t *testing.T) {
	s, _ := sketchSession(t)
	got := s.placementFieldExpression("10", sketch.FieldLength)
	if !strings.Contains(got, " ") || got == "10" {
		t.Errorf("expression = %q, want a unit suffix", got)
	}
	if a := s.placementFieldExpression("45", sketch.FieldAngle); a != "45 deg" {
		t.Errorf("angle expression = %q, want \"45 deg\"", a)
	}
}

// Tab cycles the fields, wrapping at the end.
func TestPlacementFieldTabCycles(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	s.PlacementFieldTab()
	if v := s.PlacementFields(); !v[1].Active {
		t.Error("Tab must move focus to the second field")
	}
	s.PlacementFieldTab()
	if v := s.PlacementFields(); !v[0].Active {
		t.Error("Tab must wrap back to the first field")
	}
}

// Backspace edits the active field and releases its lock, so a mistyped value can be corrected.
func TestPlacementFieldBackspaceUnlocks(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	typeInto(s, "10")
	s.PlacementFieldTab()
	s.PlacementFieldTab() // back to the width field, now locked
	s.PlacementFieldBackspace()
	if v := s.PlacementFields(); v[0].Locked {
		t.Error("editing a locked field must release its lock")
	}
}

// Escape clears typed state and returns every box to cursor tracking.
func TestPlacementFieldCancelClearsTyping(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	typeInto(s, "25")
	if !s.PlacementFieldEngaged() {
		t.Fatal("typing must engage the field strip")
	}
	s.PlacementFieldCancel()
	if s.PlacementFieldEngaged() {
		t.Error("cancel must disengage the field strip")
	}
}

// Typing alone must not freeze the drag — only locking does. Until the user presses Tab or
// Enter the value is still being composed, so the shape keeps following the cursor.
func TestTypingWithoutLockingDoesNotFreezeTheDrag(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	typeInto(s, "10")
	r, ok := s.ActiveToolRecipe(math.P2(7, 8))
	if !ok {
		t.Fatal("recipe expected")
	}
	if r.Fields[0].Value != 7 {
		t.Errorf("width = %v, want 7 — an unlocked field still tracks the cursor", r.Fields[0].Value)
	}
}

// An entry that is not a number must not freeze the drag either, even once locked: a lone sign
// is not a measurement.
func TestUnparseableEntryKeepsTrackingTheCursor(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	tool.corners = []math.Point2{math.P2(0, 0)}
	s.StartTool(tool)
	s.CursorSketchPoint(140, 60)
	typeInto(s, "-")
	s.PlacementFieldTab()
	r, ok := s.ActiveToolRecipe(math.P2(7, 8))
	if !ok {
		t.Fatal("recipe expected")
	}
	if r.Fields[0].Value != 7 {
		t.Errorf("width = %v, want 7 — an unparseable entry must keep tracking the cursor", r.Fields[0].Value)
	}
}
