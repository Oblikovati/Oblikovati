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

// registerAlias adds a command for binding tests with an optional predefined default chord
// (a full chord like "Ctrl+E"; "" ⇒ no default). Single-letter shortcuts are no longer
// auto-derived from an alias (M26), so chord tests pin a real Shift/Control default here.
func registerAlias(t *testing.T, s *Session, id, chord string) {
	t.Helper()
	cmd := NewCommand(id, id, "Test", func(*Session) error { return nil })
	if chord != "" {
		cmd = cmd.WithDefaultChord(chord)
	}
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("register %q: %v", id, err)
	}
}

func TestDefaultChordFromPredefinedChord(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "Ctrl+E")
	b := s.Bindings()

	c, ok := b.EffectiveChord("Test.Extrude")
	if !ok || c.String() != "Ctrl+E" {
		t.Fatalf("EffectiveChord = (%q, %v), want Ctrl+E", c.String(), ok)
	}
	if id, ok := b.ResolveChord(types.KeyChord{Key: "E", Ctrl: true}); !ok || id != "Test.Extrude" {
		t.Errorf("ResolveChord(Ctrl+E) = (%q, %v), want Test.Extrude", id, ok)
	}
}

// TestSingleLetterAliasHasNoDefaultChord pins the M26 rule: a bare single-letter alias does
// NOT become a default keyboard chord — single letters are personalised in the keybinding
// editor as Shift/Control chords.
func TestSingleLetterAliasHasNoDefaultChord(t *testing.T) {
	s := NewSession()
	if err := s.Commands().Add(NewCommand("Test.Q", "Q", "Test", func(*Session) error { return nil }).WithAlias("Q")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if c, ok := s.Bindings().EffectiveChord("Test.Q"); ok {
		t.Errorf("single-letter alias should yield no default chord, got %q", c.String())
	}
	if _, ok := s.Bindings().ResolveChord(types.KeyChord{Key: "Q"}); ok {
		t.Error("a bare single letter must not resolve to a command")
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
		// Toggle Visibility ships unbound (single-letter shortcuts are editor-personalised, M26).
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
	registerAlias(t, s, "Test.Extrude", "Ctrl+E")
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
	registerAlias(t, s, "Test.Extrude", "Ctrl+E")
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
	registerAlias(t, s, "Test.Extrude", "Ctrl+E")
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
	if c, ok := b.EffectiveChord("Test.Extrude"); !ok || c.String() != "Ctrl+E" {
		t.Errorf("after reset, chord = (%q, %v), want default Ctrl+E", c.String(), ok)
	}
}

func TestAliasSetResolveConflictCaseInsensitive(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "")
	registerAlias(t, s, "Test.Hole", "")
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
	registerAlias(t, s, "Test.Extrude", "")
	b := s.Bindings()

	if err := b.SetAlias("Test.Extrude", "XYZ"); err != nil {
		t.Fatalf("SetAlias: %v", err)
	}
	if b.EffectiveAlias("Test.Extrude") != "XYZ" {
		t.Errorf("alias = %q, want XYZ", b.EffectiveAlias("Test.Extrude"))
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
	registerAlias(t, s, "Test.Extrude", "")
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
	registerAlias(t, s, "Test.One", "Ctrl+J")
	registerAlias(t, s, "Test.Two", "Ctrl+J") // two commands claim the same default chord
	err := s.Bindings().CheckDefaults()
	if err == nil || !strings.Contains(err.Error(), "Ctrl+J") {
		t.Fatalf("duplicate default chord Ctrl+J should fail CheckDefaults, got %v", err)
	}
}

// TestStandardCommandsPassCheckDefaults guards the shipped registry: no standard command
// alias may collide with a built-in shortcut or another command's default chord. This is
// the same check the head runs at startup (loadKeymap), here gated headlessly.
func TestStandardCommandsPassCheckDefaults(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if err := s.Bindings().CheckDefaults(); err != nil {
		t.Fatalf("the shipped command registry has a binding conflict: %v", err)
	}
}

func TestCheckDefaultsCleanRegistry(t *testing.T) {
	s := NewSession()
	registerAlias(t, s, "Test.Extrude", "Ctrl+E")
	if err := s.Bindings().CheckDefaults(); err != nil {
		t.Errorf("a clean registry should pass CheckDefaults, got %v", err)
	}
}

func TestKeymapStorePersistsAndReloads(t *testing.T) {
	store := keymap.NewMemStore()
	s1 := NewSession()
	registerAlias(t, s1, "Test.Extrude", "")
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

// TestDispatchCancelDismissesSteeringWheel: Esc dismisses the SteeringWheels menu, the pinned wheel's
// escape hatch so a user can back out without clicking a wedge (#1754).
func TestDispatchCancelDismissesSteeringWheel(t *testing.T) {
	s := NewSession()
	s.ToggleSteeringWheel()
	if !s.SteeringWheelActive() {
		t.Fatal("precondition: the SteeringWheels menu should be active")
	}
	if err := dispatchCancel(s); err != nil {
		t.Fatalf("dispatchCancel: %v", err)
	}
	if s.SteeringWheelActive() {
		t.Error("Esc should dismiss the SteeringWheels menu")
	}
}

// TestReservedBareChordPolicy pins the #1751 keybinding policy at the predicate: bare alphanumeric
// keys (a–z, 0–9) are reserved — pressing one types into the command window, so it may only be a
// shortcut with Ctrl/Alt/Shift — while F-keys, Tab, Delete, Insert and other named/special keys, and
// any modified chord, are NOT reserved and may be bound bare.
func TestReservedBareChordPolicy(t *testing.T) {
	parse := func(s string) types.KeyChord {
		c, err := types.ParseChord(s)
		if err != nil {
			t.Fatalf("ParseChord(%q): %v", s, err)
		}
		return c
	}
	for _, s := range []string{"A", "L", "Z", "0", "5", "9"} {
		if !isReservedBareChord(parse(s)) {
			t.Errorf("%q should be reserved (bare alphanumeric)", s)
		}
	}
	for _, s := range []string{"F1", "F12", "Tab", "Delete", "Insert", "Escape", "Enter", "Ctrl+L", "Shift+A", "Alt+5"} {
		if isReservedBareChord(parse(s)) {
			t.Errorf("%q should NOT be reserved (special key or modified chord)", s)
		}
	}
}

// TestSetChordEnforcesReservedPolicy pins the editor-facing half: SetChord refuses a bare letter or
// digit, but accepts a bare special key (F8) and a modified alphanumeric (Alt+L) — #1751.
func TestSetChordEnforcesReservedPolicy(t *testing.T) {
	s := NewSession()
	if err := s.Commands().Add(NewCommand("Test.Thing", "Thing", "Test", func(*Session) error { return nil })); err != nil {
		t.Fatalf("add command: %v", err)
	}
	parse := func(str string) types.KeyChord { c, _ := types.ParseChord(str); return c }
	if err := s.Bindings().SetChord("Test.Thing", parse("L")); err == nil {
		t.Error("SetChord must reject a bare letter L")
	}
	if err := s.Bindings().SetChord("Test.Thing", parse("5")); err == nil {
		t.Error("SetChord must reject a bare digit 5")
	}
	if err := s.Bindings().SetChord("Test.Thing", parse("F8")); err != nil {
		t.Errorf("SetChord must accept a bare special key F8: %v", err)
	}
	if err := s.Bindings().SetChord("Test.Thing", parse("Alt+L")); err != nil {
		t.Errorf("SetChord must accept a modified alphanumeric Alt+L: %v", err)
	}
}

func TestPressKeyRunsCommandViaDefaultChord(t *testing.T) {
	s := NewSession()
	ran := false
	cmd := NewCommand("Test.Line", "Line", "Test", func(*Session) error { ran = true; return nil }).WithDefaultChord("Ctrl+L")
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.PressKey(KeyEvent{Key: "l", Mods: CtrlMod}); err != nil { // lowercase canonicalizes to L
		t.Fatalf("PressKey: %v", err)
	}
	if !ran {
		t.Error("pressing a command's default chord should run it through the engine")
	}
}

func TestPressKeyRebindTakesEffect(t *testing.T) {
	s := NewSession()
	ran := 0
	cmd := NewCommand("Test.Line", "Line", "Test", func(*Session) error { ran++; return nil }).WithDefaultChord("Ctrl+L")
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Bindings().SetChord("Test.Line", types.KeyChord{Key: "K", Ctrl: true}); err != nil {
		t.Fatalf("SetChord: %v", err)
	}
	_ = s.PressKey(KeyEvent{Key: "L", Mods: CtrlMod}) // the old default chord is no longer bound
	if ran != 0 {
		t.Error("the old chord should not trigger the command after a rebind")
	}
	_ = s.PressKey(KeyEvent{Key: "K", Mods: CtrlMod}) // the new chord
	if ran != 1 {
		t.Errorf("the rebound chord should trigger the command, ran=%d", ran)
	}
}

// TestDispatchUndoFiresWhileToolArmed pins the corrected #1750 semantics at the Dispatch layer:
// undo is NOT blocked merely because an interactive tool is armed — only a genuinely open
// bounded transaction blocks it (see TestKeyboardUndoBlockedDuringOpenTransaction). Before #1750
// the guard was `s.tool != nil`, so an armed tool silently killed undo and this test asserted that
// no-op; it now guards against regressing back to the over-broad guard.
func TestDispatchUndoFiresWhileToolArmed(t *testing.T) {
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

	s.StartTool(bindingStubTool{}) // a tool is now armed, but no bounded transaction is open
	if s.InTransaction() {
		t.Fatal("precondition: a merely-armed tool opens no bounded transaction")
	}
	if err := s.Bindings().Dispatch(ActionUndo, s); err != nil {
		t.Fatalf("Dispatch undo: %v", err)
	}
	if s.CanUndo() {
		t.Error("undo must fire while a tool is armed (no open transaction) — #1750")
	}
	if !s.CanRedo() {
		t.Error("the undone extrude should be redoable")
	}
}
