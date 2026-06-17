// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/app/cmdline"
)

// lastEcho returns the text of the most recent echoed (user-input) scrollback line.
func lastEcho(cl *CommandLine) string {
	lines := cl.Scrollback().Lines()
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].Severity == cmdline.Echo {
			return lines[i].Text
		}
	}
	return ""
}

// TestCtrlChordEchoesCanonicalWord verifies M26 F05: a Ctrl chord runs through the command
// line and echoes the action's AutoCAD word, so Ctrl+Z reads as "UNDO" on the command line.
func TestCtrlChordEchoesCanonicalWord(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	_ = s.PressKey(KeyEvent{Key: "z", Mods: CtrlMod}) // Ctrl+Z → UNDO (nothing to undo: no-op)
	if got := lastEcho(s.CommandLine()); got != "UNDO" {
		t.Errorf("Ctrl+Z echoed %q, want UNDO", got)
	}
}

// TestCtrlSSavesAndEchoes verifies Ctrl+S autofills "SAVE" and runs the save action. The
// fixture document has no file path, so the save reaches SaveActiveDocument and returns its
// "use Save As" error — which proves the chord routed to the save action (and was echoed).
func TestCtrlSSavesAndEchoes(t *testing.T) {
	s, _ := newPartWithSquare(t, 2)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	err := s.PressKey(KeyEvent{Key: "s", Mods: CtrlMod})
	if err == nil || !strings.Contains(err.Error(), "Save As") {
		t.Errorf("Ctrl+S error = %v, want the SaveActiveDocument no-path error", err)
	}
	if got := lastEcho(s.CommandLine()); got != "SAVE" {
		t.Errorf("Ctrl+S echoed %q, want SAVE", got)
	}
}

// TestSingleLetterChordNeedsModifier confirms the M26 rule on the keyboard path: a bare
// single letter does NOT dispatch a command (no plain default exists), while the same letter
// chorded with Control does. Single-letter shortcuts are personalised as Shift/Control chords.
func TestSingleLetterChordNeedsModifier(t *testing.T) {
	s := NewSession()
	ran := false
	if err := s.Commands().Add(NewCommand("Test.Greet", "Greet", "Test",
		func(*Session) error { ran = true; return nil }).WithDefaultChord("Ctrl+G")); err != nil {
		t.Fatalf("add command: %v", err)
	}
	if err := s.PressKey(KeyEvent{Key: "G"}); err != nil { // plain G: no command
		t.Fatalf("PressKey(G): %v", err)
	}
	if ran {
		t.Error("a bare single letter must not dispatch a command")
	}
	if err := s.PressKey(KeyEvent{Key: "G", Mods: CtrlMod}); err != nil {
		t.Fatalf("PressKey(Ctrl+G): %v", err)
	}
	if !ran {
		t.Error("Ctrl+G should dispatch the command")
	}
}

// TestTypedSingleLetterDoesNotResolve pins the M26 rule that single-letter commands are NOT
// part of the static command-window vocabulary: they belong to the keybinding editor, where
// the user personalises them as Shift/Control chords. So a bare "V" typed and Entered is an
// unknown command, not a shortcut — the user presses their configured chord instead.
func TestTypedSingleLetterDoesNotResolve(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if err := s.CommandLine().Submit(s, "V"); err == nil {
		t.Fatal("Submit(V) should report an unknown command (single letters are editor-only)")
	}
	found := false
	for _, l := range s.CommandLine().Scrollback().Lines() {
		if l.Severity == cmdline.Error && strings.Contains(l.Text, "Unknown") {
			found = true
		}
	}
	if !found {
		t.Error("expected an 'Unknown command' line for a typed single letter")
	}
}
