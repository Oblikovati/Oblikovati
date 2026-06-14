// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app/keymap"
)

// bindingStubTool is a no-op Tool used to put a session into the "tool active" state so
// the undo/redo dispatch guards can be exercised.
type bindingStubTool struct{}

func (bindingStubTool) Name() string              { return "Stub" }
func (bindingStubTool) Start(*Session)            {}
func (bindingStubTool) Pick(*Session, Selectable) {}
func (bindingStubTool) CanCommit() bool           { return false }
func (bindingStubTool) Commit(*Session) error     { return nil }
func (bindingStubTool) Cancel(*Session)           {}

// registerAlias adds a command with the given id/alias for binding tests.
func registerAlias(t *testing.T, s *Session, id, alias string) {
	t.Helper()
	cmd := NewCommand(id, id, "Test", func(*Session) error { return nil })
	if alias != "" {
		cmd = cmd.WithAlias(alias)
	}
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("register %q: %v", id, err)
	}
}

func TestDefaultChordDerivesFromAlias(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	b := s.Bindings()

	c, ok := b.EffectiveChord("Test.Extrude")
	if !ok || c.String() != "E" {
		t.Fatalf("EffectiveChord = (%q, %v), want E", c.String(), ok)
	}
	if id, ok := b.ResolveChord(types.KeyChord{Key: "E"}); !ok || id != "Test.Extrude" {
		t.Errorf("ResolveChord(E) = (%q, %v), want Test.Extrude", id, ok)
	}
}

func TestResolveChordBuiltinDefaults(t *testing.T) {
	b := NewSession().Bindings()
	cases := map[string]string{
		"Ctrl+Z":       ActionUndo,
		"Ctrl+Y":       ActionRedo,
		"Ctrl+Shift+Z": ActionRedo, // extra default chord still resolves
		"Escape":       ActionCancel,
		"Enter":        ActionCommit,
		"V":            ActionToggleVisibility,
	}
	for chordStr, want := range cases {
		c, _ := types.ParseChord(chordStr)
		if id, ok := b.ResolveChord(c); !ok || id != want {
			t.Errorf("ResolveChord(%q) = (%q, %v), want %q", chordStr, id, ok, want)
		}
	}
}

func TestSetChordOverrideRebinds(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	b := s.Bindings()

	if err := b.SetChord("Test.Extrude", types.KeyChord{Key: "E", Ctrl: true, Shift: true}); err != nil {
		t.Fatalf("SetChord: %v", err)
	}
	if c, _ := b.EffectiveChord("Test.Extrude"); c.String() != "Ctrl+Shift+E" {
		t.Errorf("effective chord = %q, want Ctrl+Shift+E", c.String())
	}
	if _, ok := b.ResolveChord(types.KeyChord{Key: "E"}); ok {
		t.Error("old chord E should no longer resolve after rebind")
	}
	if id, ok := b.ResolveChord(types.KeyChord{Key: "E", Ctrl: true, Shift: true}); !ok || id != "Test.Extrude" {
		t.Errorf("new chord should resolve to Test.Extrude, got (%q, %v)", id, ok)
	}
}

func TestSetChordConflictIsRejected(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	err := s.Bindings().SetChord("Test.Extrude", types.KeyChord{Key: "Z", Ctrl: true})
	if err == nil {
		t.Fatal("rebinding to Ctrl+Z (undo) should conflict")
	}
	if !strings.Contains(err.Error(), "Ctrl+Z") || !strings.Contains(err.Error(), ActionUndo) {
		t.Errorf("error %q should name the offending chord and the colliding action", err)
	}
}

func TestSetChordClearThenReset(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	b := s.Bindings()

	if err := b.SetChord("Test.Extrude", types.KeyChord{}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, ok := b.EffectiveChord("Test.Extrude"); ok {
		t.Error("cleared chord should be unbound, not fall back to default")
	}
	if err := b.Reset("Test.Extrude"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if c, ok := b.EffectiveChord("Test.Extrude"); !ok || c.String() != "E" {
		t.Errorf("after reset, chord = (%q, %v), want default E", c.String(), ok)
	}
}

func TestAliasSetResolveConflictCaseInsensitive(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	registerAlias(t, s, "Test.Hole", "H")
	b := s.Bindings()

	if err := b.SetAlias("Test.Extrude", "EXT"); err != nil {
		t.Fatalf("SetAlias: %v", err)
	}
	if id, ok := b.ResolveAlias("ext"); !ok || id != "Test.Extrude" {
		t.Errorf("ResolveAlias is case-insensitive: got (%q, %v)", id, ok)
	}
	if err := b.SetAlias("Test.Hole", "EXT"); err == nil {
		t.Error("a duplicate alias should be rejected")
	}
}

func TestAliasWithoutDefaultResetRemovesIt(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	b := s.Bindings()

	if err := b.SetAlias("Test.Extrude", "X"); err != nil {
		t.Fatalf("SetAlias: %v", err)
	}
	if b.EffectiveAlias("Test.Extrude") != "X" {
		t.Errorf("alias = %q, want X", b.EffectiveAlias("Test.Extrude"))
	}
	if err := b.Reset("Test.Extrude"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if b.EffectiveAlias("Test.Extrude") != "" {
		t.Error("alias had no default; reset should remove it entirely")
	}
}

func TestResetAllClearsEveryCustomization(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	b := s.Bindings()
	_ = b.SetChord("Test.Extrude", types.KeyChord{Key: "E", Ctrl: true})
	_ = b.SetAlias("Test.Extrude", "EXT")

	if err := b.ResetAll(); err != nil {
		t.Fatalf("ResetAll: %v", err)
	}
	for _, bd := range b.Catalog() {
		if bd.Customized {
			t.Errorf("action %q still customized after ResetAll", bd.ActionID)
		}
	}
}

func TestCheckDefaultsReservedIDCollision(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, ActionUndo, "") // a command id colliding with a built-in id
	if err := s.Bindings().CheckDefaults(); err == nil {
		t.Fatal("a command reusing a reserved built-in id should fail CheckDefaults")
	}
}

func TestCheckDefaultsDuplicateChord(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Visibility", "V") // collides with built-in V (toggleVisibility)
	err := s.Bindings().CheckDefaults()
	if err == nil || !strings.Contains(err.Error(), "V") {
		t.Fatalf("duplicate default chord V should fail CheckDefaults, got %v", err)
	}
}

func TestCheckDefaultsCleanRegistry(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "E")
	if err := s.Bindings().CheckDefaults(); err != nil {
		t.Errorf("a clean registry should pass CheckDefaults, got %v", err)
	}
}

func TestKeymapStorePersistsAndReloads(t *testing.T) {
	store := keymap.NewMemStore()
	s1 := NewSession()
	registerAlias(t, s1, "Test.Extrude", "E")
	if err := s1.UseKeymapStore(store); err != nil {
		t.Fatalf("UseKeymapStore: %v", err)
	}
	if err := s1.Bindings().SetChord("Test.Extrude", types.KeyChord{Key: "E", Ctrl: true}); err != nil {
		t.Fatalf("SetChord: %v", err)
	}
	if store.Saved == 0 {
		t.Error("SetChord should have persisted through the store")
	}

	s2 := NewSession()
	if err := s2.UseKeymapStore(store); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if c, ok := s2.Bindings().EffectiveChord("Test.Extrude"); !ok || c.String() != "Ctrl+E" {
		t.Errorf("reloaded chord = (%q, %v), want Ctrl+E", c.String(), ok)
	}
}

func TestDispatchRunsCommand(t *testing.T) {
	s := NewSession()
	ran := false
	cmd := NewCommand("Test.Run", "Run", "Test", func(*Session) error { ran = true; return nil })
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Bindings().Dispatch("Test.Run", s); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if !ran {
		t.Error("Dispatch of a command id should run the command")
	}
}

func TestDispatchCancelClearsToolThenSelection(t *testing.T) {
	s := NewSession()
	s.StartTool(bindingStubTool{})
	if err := s.Bindings().Dispatch(ActionCancel, s); err != nil {
		t.Fatalf("Dispatch cancel: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Error("cancel should clear the active tool")
	}
}

func TestDispatchUndoGuardedWhileToolActive(t *testing.T) {
	s, profile := newPartWithSquare(t, 2)
	trackFromHere(s)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(120, 90)
	ext.SetDistance(5)
	if err := s.OK(); err != nil {
		t.Fatalf("extrude OK: %v", err)
	}
	if !s.CanUndo() {
		t.Fatal("precondition: the extrude should be undoable")
	}

	s.StartTool(bindingStubTool{}) // a tool is now active
	if err := s.Bindings().Dispatch(ActionUndo, s); err != nil {
		t.Fatalf("Dispatch undo: %v", err)
	}
	if !s.CanUndo() {
		t.Error("undo must be a no-op while a tool is active (guard preserved)")
	}

	s.CancelTool()
	if err := s.Bindings().Dispatch(ActionUndo, s); err != nil {
		t.Fatalf("Dispatch undo after cancel: %v", err)
	}
	if s.CanUndo() {
		t.Error("with no tool, undo should consume the extrude transaction")
	}
}
