//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestInWindowMarkingMenuEditorDraws opens the Customize Marking Menu window and runs
// frames, exercising the tab bar, slot table, search section, and footer buttons
// without tripping Dear ImGui's Begin/End assertions (mirrors TestInWindowKeymapEditorDraws).
func TestInWindowMarkingMenuEditorDraws(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}

	s.OpenMarkingMenuEditor()
	defer s.CloseMarkingMenuEditor()

	// Unfiltered: the slot table renders for the default Base environment.
	drawKeymapFrames(win, s, 3)
	if !s.MarkingMenuEditorOpen() {
		t.Error("Customize Marking Menu should stay open across frames")
	}

	// Filtered: seed the filter buffer so the search path renders too.
	copyText(mmEditorUI.filter, "ext")
	drawKeymapFrames(win, s, 3)
	if !s.MarkingMenuEditorOpen() {
		t.Error("Customize Marking Menu should stay open while filtering")
	}
}
