//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// TestInWindowKeymapEditorDraws opens the real Customize Keyboard window and runs frames,
// exercising the sorted/filtered table, the filter input, the per-row edit fields, and the
// footer/close buttons without tripping Dear ImGui's Begin/End assertions (issue #1232).
func TestInWindowKeymapEditorDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}

	s.OpenKeymapEditor()
	defer s.CloseKeymapEditor()

	// Unfiltered: the whole catalog renders, sorted.
	drawKeymapFrames(win, s, 3)
	if !s.KeymapEditorOpen() {
		t.Error("Customize Keyboard should stay open across frames")
	}

	// Filtered: seed the filter buffer so the narrowed path renders too.
	copyText(keymapUI.filter, "dim")
	drawKeymapFrames(win, s, 3)
	if !s.KeymapEditorOpen() {
		t.Error("Customize Keyboard should stay open while filtering")
	}
}

// drawKeymapFrames renders n full chrome frames (the window draws as part of chrome).
func drawKeymapFrames(win *native.Window, s *app.Session, n int) {
	for range n {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}
