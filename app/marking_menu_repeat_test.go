// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// countingCommand registers a test command that tallies its runs, returning the session and a
// pointer to the run counter.
func countingCommand(t *testing.T, s *Session, id string) *int {
	t.Helper()
	runs := 0
	if err := s.Commands().Add(NewCommand(id, "My "+id, "Test", func(*Session) error {
		runs++
		return nil
	})); err != nil {
		t.Fatalf("register %q: %v", id, err)
	}
	return &runs
}

// TestLastCommandTrackedAndRepeated: executing a command records it as the last, and
// RepeatLastCommand re-runs it (#915 C5).
func TestLastCommandTrackedAndRepeated(t *testing.T) {
	t.Parallel()
	s := NewSession()
	runs := countingCommand(t, s, "Test.Ping")

	if _, ok := s.LastCommandID(); ok {
		t.Error("a fresh session should have no last command")
	}
	if err := s.Execute("Test.Ping"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if id, ok := s.LastCommandID(); !ok || id != "Test.Ping" {
		t.Errorf("LastCommandID = (%q, %v), want (Test.Ping, true)", id, ok)
	}

	if err := s.RepeatLastCommand(); err != nil {
		t.Fatalf("repeat: %v", err)
	}
	if *runs != 2 {
		t.Errorf("command ran %d times, want 2 (once executed, once repeated)", *runs)
	}
}

// TestRepeatLastNoPriorCommandIsNoOp: with no prior command, repeat does nothing.
func TestRepeatLastNoPriorCommandIsNoOp(t *testing.T) {
	t.Parallel()
	if err := NewSession().RepeatLastCommand(); err != nil {
		t.Errorf("repeat with no prior command should be a no-op, got %v", err)
	}
}

// TestRepeatMenuEntryIdleAndActive: the Repeat entry shows the last command's name when idle, and
// is withheld while a tool is active or before any command has run (#915 C5).
func TestRepeatMenuEntryIdleAndActive(t *testing.T) {
	t.Parallel()
	s := NewSession()
	countingCommand(t, s, "Test.Extrude")

	if _, _, ok := s.RepeatMenuEntry(); ok {
		t.Error("no Repeat entry before any command has run")
	}

	if err := s.Execute("Test.Extrude"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	label, id, ok := s.RepeatMenuEntry()
	if !ok || id != "Test.Extrude" || label != "Repeat My Test.Extrude" {
		t.Errorf("RepeatMenuEntry = (%q, %q, %v), want (Repeat My Test.Extrude, Test.Extrude, true)", label, id, ok)
	}

	// While a tool is active the menu offers in-command actions, not Repeat.
	s.StartTool(NewLineTool())
	if _, _, ok := s.RepeatMenuEntry(); ok {
		t.Error("the Repeat entry must be withheld while a tool is active")
	}
}

// TestContextMenuStyleToggle: the right-click style defaults to the radial marking menu and flips
// to the classic linear menu (#915 C8).
func TestContextMenuStyleToggle(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.ClassicContextMenu() {
		t.Error("the right-click menu should default to the radial marking menu")
	}
	s.ToggleContextMenuStyle()
	if !s.ClassicContextMenu() {
		t.Error("after toggling, the classic linear menu should be selected")
	}
	s.SetClassicContextMenu(false)
	if s.ClassicContextMenu() {
		t.Error("SetClassicContextMenu(false) should restore the radial marking menu")
	}
}
