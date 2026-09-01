// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// aliasCommand registers a no-op command with the given alias and reports whether it ran, so a
// test can assert a keystroke did NOT dispatch it (the bare-letter typing policy, #1751 S2).
func aliasCommand(t *testing.T, s *Session, id, alias string, ran *bool) {
	t.Helper()
	cmd := NewCommand(id, "Thing", "Test", func(*Session) error { *ran = true; return nil }).WithAlias(alias)
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("add command %q: %v", id, err)
	}
}

// TestBareLetterBeginsCommandTyping pins #1751 S2: a bare letter with the viewport focused hands
// focus to the Command Window seeded with that (lower-cased) letter, rather than firing a shortcut.
func TestBareLetterBeginsCommandTyping(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	aliasCommand(t, s, "Test.Line", "L", &ran)

	if err := s.PressKey(KeyEvent{Key: "L"}); err != nil {
		t.Fatalf("PressKey(L): %v", err)
	}
	if ran {
		t.Error("a bare letter must not dispatch a command")
	}
	if !s.TakeCommandInputFocus() {
		t.Error("a bare letter must request Command Window focus")
	}
	if seed, ok := s.TakeCommandTypeSeed(); !ok || seed != "l" {
		t.Errorf("seed = %q,%v; want \"l\",true", seed, ok)
	}
}

// TestBareDigitBeginsCommandTyping confirms digits 0–9 are reserved for typing too.
func TestBareDigitBeginsCommandTyping(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.PressKey(KeyEvent{Key: "5"}); err != nil {
		t.Fatalf("PressKey(5): %v", err)
	}
	if !s.TakeCommandInputFocus() {
		t.Error("a bare digit must request Command Window focus")
	}
	if seed, ok := s.TakeCommandTypeSeed(); !ok || seed != "5" {
		t.Errorf("seed = %q,%v; want \"5\",true", seed, ok)
	}
}

// TestSpecialKeyDoesNotBeginTyping confirms a bare special key (F8) is NOT reserved: it dispatches
// its bound command directly and begins no command-window typing (#1751 S1 policy, exercised via S2).
func TestSpecialKeyDoesNotBeginTyping(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	cmd := NewCommand("Test.Refresh", "Refresh", "Test",
		func(*Session) error { ran = true; return nil }).WithDefaultChord("F8")
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("add command: %v", err)
	}
	if err := s.PressKey(KeyEvent{Key: "F8"}); err != nil {
		t.Fatalf("PressKey(F8): %v", err)
	}
	if !ran {
		t.Error("a bare special key F8 must dispatch its bound command")
	}
	if s.TakeCommandInputFocus() {
		t.Error("a bare special key must not request command typing focus")
	}
	if _, ok := s.TakeCommandTypeSeed(); ok {
		t.Error("a bare special key must not seed the command input")
	}
}

// TestModifiedLetterDoesNotBeginTyping confirms a modified alphanumeric (Ctrl+L) resolves as a
// shortcut and does not begin typing — modifiers are exactly what turn a letter into a shortcut.
func TestModifiedLetterDoesNotBeginTyping(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	cmd := NewCommand("Test.Line", "Line", "Test",
		func(*Session) error { ran = true; return nil }).WithDefaultChord("Ctrl+L")
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("add command: %v", err)
	}
	if err := s.PressKey(KeyEvent{Key: "L", Mods: CtrlMod}); err != nil {
		t.Fatalf("PressKey(Ctrl+L): %v", err)
	}
	if !ran {
		t.Error("Ctrl+L should dispatch its bound command")
	}
	if _, ok := s.TakeCommandTypeSeed(); ok {
		t.Error("a modified letter must not seed the command input")
	}
}
