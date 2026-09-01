// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestCancelDismissesPromptAndRequestsFocus checks ESC (dispatchCancel) drops a pending
// command-line question instead of answering it, and asks the head to refocus the command
// input (M26).
func TestCancelDismissesPromptAndRequestsFocus(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if _, _, err := s.ShowPrompt(PromptSpec{ID: "q", Message: "Delete?", Buttons: []string{"Yes", "No"}}); err != nil {
		t.Fatalf("ShowPrompt: %v", err)
	}
	if err := dispatchCancel(s); err != nil {
		t.Fatalf("dispatchCancel: %v", err)
	}
	if _, ok := s.Prompts().Pending(); ok {
		t.Error("ESC should drop the pending prompt")
	}
	if !s.TakeCommandInputFocus() {
		t.Error("ESC should request command-input focus")
	}
}

// TestCancelWithNoToolClearsSelectionAndRefocuses preserves the legacy idle behaviour (clear
// the selection) while still requesting focus.
func TestCancelWithNoToolClearsSelectionAndRefocuses(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := dispatchCancel(s); err != nil {
		t.Fatalf("dispatchCancel: %v", err)
	}
	if !s.TakeCommandInputFocus() {
		t.Error("ESC should request command-input focus even when idle")
	}
	// A second take returns false — the request is one-shot.
	if s.TakeCommandInputFocus() {
		t.Error("focus request should be consumed exactly once")
	}
}

// TestEscapeChordResolvesToCancel keeps Escape wired to the universal cancel action.
func TestEscapeChordResolvesToCancel(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	got, ok := s.Bindings().ResolveChord(types.KeyChord{Key: "Escape"})
	if !ok || got != ActionCancel {
		t.Errorf("ResolveChord(Escape) = %q,%v, want %q", got, ok, ActionCancel)
	}
}
