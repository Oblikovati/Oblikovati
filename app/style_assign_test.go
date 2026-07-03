// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/style"
)

// bodyKeyOf returns the reference key of the boxBodySession's single body.
func bodyKeyOf(t *testing.T, def *compdef.PartComponentDefinition) string {
	t.Helper()
	bodies := def.SurfaceBodies().All()
	if len(bodies) == 0 {
		t.Fatal("no body in the session")
	}
	return string(bodies[0].ReferenceKey())
}

// TestAssignColorStyleToBody checks a body's color-style assignment round-trips on the active
// document and that an unknown style is rejected (M16-F02 #403/#408, S5 #1640).
func TestAssignColorStyleToBody(t *testing.T) {
	s, def := boxBodySession(t)
	key := bodyKeyOf(t, def)
	if err := s.AssignColorStyleToBody(key, "Brass"); err != nil {
		t.Fatalf("assign Brass: %v", err)
	}
	if name, ok := s.BodyColorStyle(key); !ok || name != "Brass" {
		t.Errorf("BodyColorStyle = (%q, %v), want (Brass, true)", name, ok)
	}
	if err := s.AssignColorStyleToBody(key, "Nope"); err == nil {
		t.Error("assigning an unknown style should error")
	}
	s.ClearBodyColorStyle(key)
	if _, ok := s.BodyColorStyle(key); ok {
		t.Error("assignment should be gone after ClearBodyColorStyle")
	}
}

// TestAssignColorStyleIsPerDocument checks the assignment lives on the document, not the session — a
// second document does not see the first's color (the pre-#1640 session-global map leaked across docs).
func TestAssignColorStyleIsPerDocument(t *testing.T) {
	s, def := boxBodySession(t)
	key := bodyKeyOf(t, def)
	if err := s.AssignColorStyleToBody(key, "Brass"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if _, err := s.NewPart(); err != nil { // a second, now-active document
		t.Fatalf("NewPart: %v", err)
	}
	if _, ok := s.BodyColorStyle(key); ok {
		t.Error("a fresh document must not inherit the first document's color assignment")
	}
}

// TestBodyColorStyleUndoRedo: assign → undo clears it → redo restores it (S5 #1640 undo lifecycle).
func TestBodyColorStyleUndoRedo(t *testing.T) {
	s, def := boxBodySession(t)
	key := bodyKeyOf(t, def)
	if err := s.AssignColorStyleToBody(key, "Brass"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if _, ok := s.BodyColorStyle(key); ok {
		t.Error("undo should clear the color assignment")
	}
	if err := s.Redo(); err != nil {
		t.Fatalf("redo: %v", err)
	}
	if name, ok := s.BodyColorStyle(key); !ok || name != "Brass" {
		t.Errorf("redo should restore the assignment, got (%q,%v)", name, ok)
	}
}

// TestBodyColorStyleEmitsEvent checks exactly one BodyColorStyleChanged fires on assign, carrying the
// body key and style name (the granular appearance event add-ins observe, S5 #1640).
func TestBodyColorStyleEmitsEvent(t *testing.T) {
	s, def := boxBodySession(t)
	key := bodyKeyOf(t, def)
	var got []BodyColorStyleChanged
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e BodyColorStyleChanged) event.Outcome {
		got = append(got, e)
		return event.Continue()
	})
	if err := s.AssignColorStyleToBody(key, "Brass"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if len(got) != 1 || got[0].BodyKey != key || got[0].Style != "Brass" {
		t.Fatalf("events = %+v, want one {key:%s, Brass}", got, key)
	}
	s.ClearBodyColorStyle(key)
	if len(got) != 2 || got[1].Style != "" {
		t.Fatalf("clear should fire a second event with empty style, got %+v", got)
	}
}

// TestColorStyleNoActiveDocument covers the guard branches: with no document open, reads report "no
// style" and a clear is a safe no-op that surfaces errNoActiveDocument rather than panicking.
func TestColorStyleNoActiveDocument(t *testing.T) {
	s := NewSession() // no documents open → ActiveDocument() is nil
	if _, ok := s.BodyColorStyle("k"); ok {
		t.Error("BodyColorStyle with no active document should report no assignment")
	}
	s.ClearBodyColorStyle("k") // exercises setBodyColorStyle's no-active-document early return
}

// TestBodyColorStyleChangedEventID pins the event's stable type id.
func TestBodyColorStyleChangedEventID(t *testing.T) {
	if got := (BodyColorStyleChanged{}).EventID(); got != tidBodyColorStyleChanged {
		t.Errorf("EventID = %v, want tidBodyColorStyleChanged", got)
	}
}

// TestStyleSurfaceUsesDiffuseAlbedo checks the style→surface conversion drives albedo from the
// diffuse color and roughness from shininess.
func TestStyleSurfaceUsesDiffuseAlbedo(t *testing.T) {
	cs := style.ColorStyle{Diffuse: types.NewColor(255, 0, 0), Shininess: 1, Opacity: 1}
	surf := styleSurface(cs)
	if surf.Albedo[0] != 1 || surf.Albedo[1] != 0 {
		t.Errorf("albedo = %v, want red", surf.Albedo)
	}
	if surf.Roughness != 0 {
		t.Errorf("roughness = %v, want 0 (shininess 1)", surf.Roughness)
	}
}
