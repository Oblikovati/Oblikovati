// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"
)

// addAliased registers a flag-setting command and gives it the custom alias.
func addAliased(t *testing.T, s *Session, id, alias string, ran *bool) {
	t.Helper()
	cmd := NewCommand(id, id, "Test", func(*Session) error { *ran = true; return nil })
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add %q: %v", id, err)
	}
	if alias != "" {
		if err := s.Bindings().SetAlias(id, alias); err != nil {
			t.Fatalf("SetAlias %q: %v", id, err)
		}
	}
}

func TestCommandInputCommitsCustomAlias(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	addAliased(t, s, "Test.Extrude", "EXT", &ran)

	s.BeginCommandInput()
	for _, r := range "ext" { // case-insensitive
		s.AppendCommandInputRune(r)
	}
	if err := s.CommitCommandInput(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if !ran {
		t.Error("commit should dispatch the aliased command")
	}
	if s.CommandInputActive() {
		t.Error("commit should close the box")
	}
}

func TestCommandInputRejectsSingleLetter(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	cmd := NewCommand("Test.Line", "Line", "Test", func(*Session) error { ran = true; return nil }).WithDefaultChord("Ctrl+L")
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.BeginCommandInput()
	s.SetCommandInputText("L")
	// A bare single letter is not a command word (M26) — it does not resolve, and nothing runs.
	if err := s.CommitCommandInput(); err == nil {
		t.Error("a single letter should not resolve to a command")
	}
	if ran {
		t.Error("a single letter must not run a command")
	}
}

func TestCommandInputMatchesPrefix(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	addAliased(t, s, "Test.Extrude", "EXT", &ran)
	s.BeginCommandInput()
	s.SetCommandInputText("ex")

	m := s.CommandInputMatches()
	if len(m) != 1 || !strings.Contains(m[0], "EXT") {
		t.Errorf("matches = %v, want one EXT hint", m)
	}
}

func TestCommandInputCommitMissStaysOpen(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.BeginCommandInput()
	s.SetCommandInputText("nope")
	if err := s.CommitCommandInput(); err == nil {
		t.Fatal("an unknown alias should error")
	}
	if !s.CommandInputActive() {
		t.Error("a miss should keep the box open")
	}
	if !strings.Contains(s.Notice(), "nope") {
		t.Errorf("notice %q should name the bad alias", s.Notice())
	}
}

func TestCommandInputBufferEditingViaPressKey(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	addAliased(t, s, "Test.Extrude", "EXT", &ran)

	s.BeginCommandInput()
	for _, k := range []string{"e", "x", "q"} {
		_ = s.PressKey(KeyEvent{Key: k})
	}
	_ = s.PressKey(KeyEvent{Key: "Backspace"}) // delete the stray q
	_ = s.PressKey(KeyEvent{Key: "t"})
	if s.CommandInputText() != "ext" {
		t.Fatalf("buffer = %q, want ext", s.CommandInputText())
	}
	if err := s.PressKey(KeyEvent{Key: "Enter"}); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	if !ran || s.CommandInputActive() {
		t.Errorf("Enter should commit and close: ran=%v active=%v", ran, s.CommandInputActive())
	}
}

func TestCommandInputEscapeCancels(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.BeginCommandInput()
	s.SetCommandInputText("ex")
	if err := s.PressKey(KeyEvent{Key: "Escape"}); err != nil {
		t.Fatalf("Escape: %v", err)
	}
	if s.CommandInputActive() {
		t.Error("Escape should cancel the box")
	}
}

// While the box is open, a chord must NOT trigger its global shortcut — the keystroke is
// captured by the buffer instead (the head additionally gates this on text focus).
func TestCommandInputCapturesKeysFromShortcuts(t *testing.T) {
	t.Parallel()
	s := NewSession()
	ran := false
	cmd := NewCommand("Test.Line", "Line", "Test", func(*Session) error { ran = true; return nil }).WithAlias("L")
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add: %v", err)
	}
	s.BeginCommandInput()
	_ = s.PressKey(KeyEvent{Key: "L"}) // would run the command if it were a shortcut
	if ran {
		t.Error("a key typed into the open box must not fire its shortcut")
	}
	if s.CommandInputText() != "L" {
		t.Errorf("the key should land in the buffer, got %q", s.CommandInputText())
	}
}
