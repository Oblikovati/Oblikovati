//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// TestInWindowMaterialsWindowDrawsAllTabs opens the real window with the Materials window
// visible and runs frames, so a mismatched ImGui Begin/End (tab bar, combo, disabled) in the
// material/appearance/physical panes would trip Dear ImGui's assertions. It just needs to
// run without aborting.
func TestInWindowMaterialsWindowDrawsAllTabs(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	s := framedSession()

	showMaterials = true
	defer func() { showMaterials = false }()

	// A couple of frames so the window appears and its default tab renders, plus selecting
	// the first built-in material populates the (disabled) property editor.
	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}

	// The library the window reads must be non-empty (the combos would otherwise be blank).
	if len(s.Materials().Materials()) == 0 {
		t.Fatal("materials library empty during window draw")
	}
}
