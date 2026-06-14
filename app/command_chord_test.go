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

// TestPlainKeyChordStillDispatches confirms the no-text-field path is unchanged: a plain
// shortcut key dispatched via PressKey still runs its command directly (the head keeps the
// command line focused so, live, a plain letter fills it instead — but PressKey itself, the
// fallback path, must keep working for existing shortcuts).
func TestPlainKeyChordStillDispatches(t *testing.T) {
	s := NewSession()
	ran := false
	if err := s.Commands().Add(NewCommand("Test.Greet", "Greet", "Test",
		func(*Session) error { ran = true; return nil }).WithAlias("G")); err != nil {
		t.Fatalf("add command: %v", err)
	}
	if err := s.PressKey(KeyEvent{Key: "G"}); err != nil {
		t.Fatalf("PressKey(G): %v", err)
	}
	if !ran {
		t.Error("plain shortcut key should still dispatch its command directly")
	}
}

// TestTypedSingleLetterResolvesOnEnter verifies the command line resolves a single-letter
// shortcut typed and committed with Enter (the "fill, await Enter" path): "V" → toggle
// visibility, which is a built-in shortcut not present in the AutoCAD vocabulary.
func TestTypedSingleLetterResolvesOnEnter(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	if err := s.CommandLine().Submit(s, "V"); err != nil {
		t.Fatalf("Submit(V): %v", err)
	}
	// It resolved (no "Unknown command" error line).
	for _, l := range s.CommandLine().Scrollback().Lines() {
		if l.Severity == cmdline.Error && strings.Contains(l.Text, "Unknown") {
			t.Errorf("typed 'V' was not resolved: %q", l.Text)
		}
	}
}
