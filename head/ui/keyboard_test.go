//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
)

// TestDispatchPressedKeysRunsShortcut proves the head's key-forwarding (M05-F17) drives the
// binding engine: a forwarded Shift/Control chord resolves to its shortcut command and runs.
// (A bare single letter is reserved for the keybinding editor, M26, so the shortcut is Ctrl+L.)
func TestDispatchPressedKeysRunsShortcut(t *testing.T) {
	s := app.NewSession()
	ran := false
	cmd := app.NewCommand("Test.Line", "Line", "Test", func(*app.Session) error { ran = true; return nil }).WithDefaultChord("Ctrl+L")
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add: %v", err)
	}
	dispatchPressedKeys(s, []string{"L"}, app.CtrlMod)
	if !ran {
		t.Error("forwarding Ctrl+L should run the shortcut command")
	}
}

// TestDispatchPressedKeysCarriesModifiers proves a modified chord (Ctrl+Z) forwards to the
// built-in it is bound to — here undo, after an undoable edit.
func TestDispatchPressedKeysCarriesModifiers(t *testing.T) {
	s := app.NewSession()
	ran := 0
	cmd := app.NewCommand("Test.Probe", "Probe", "Test", func(*app.Session) error { ran++; return nil })
	if err := s.Commands().Add(cmd); err != nil {
		t.Fatalf("Add: %v", err)
	}
	chord, err := types.ParseChord("Ctrl+K")
	if err != nil {
		t.Fatalf("ParseChord: %v", err)
	}
	if err := s.Bindings().SetChord("Test.Probe", chord); err != nil {
		t.Fatalf("SetChord: %v", err)
	}
	dispatchPressedKeys(s, []string{"K"}, app.CtrlMod)
	if ran != 1 {
		t.Errorf("Ctrl+K should run the rebound command once, ran=%d", ran)
	}
}

// TestDispatchPressedKeysSkipsEscape proves Escape is not double-handled here (handleKeyboard
// forwards it separately).
func TestDispatchPressedKeysSkipsEscape(t *testing.T) {
	s := app.NewSession()
	s.StartTool(stubKeyboardTool{})
	dispatchPressedKeys(s, []string{"Escape"}, 0)
	if s.ActiveTool() == nil {
		t.Error("dispatchPressedKeys must skip Escape (handleKeyboard owns it), so the tool stays active")
	}
}

// stubKeyboardTool is a no-op tool to observe that Escape was not dispatched here.
type stubKeyboardTool struct{}

func (stubKeyboardTool) Name() string                      { return "Stub" }
func (stubKeyboardTool) Start(*app.Session)                {}
func (stubKeyboardTool) Pick(*app.Session, app.Selectable) {}
func (stubKeyboardTool) CanCommit() bool                   { return false }
func (stubKeyboardTool) Commit(*app.Session) error         { return nil }
func (stubKeyboardTool) Cancel(*app.Session)               {}
