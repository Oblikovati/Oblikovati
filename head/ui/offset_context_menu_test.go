//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
)

// TestInWindowOffsetContextMenuOptions renders the right-click menu with the Offset tool active and
// checks its two in-command toggles (Loop Select / Constrain Offset) draw through the real cgo path.
// It backs the live visual confirmation for Inventor's Offset right-click options and guards the
// session wiring the menu reads.
func TestInWindowOffsetContextMenuOptions(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	s.StartTool(app.NewSketchOffsetTool(0.5))
	if got := len(s.ActiveToolMenuOptions()); got != 2 {
		t.Fatalf("offset tool exposed %d context-menu options, want 2 (Loop Select, Constrain Offset)", got)
	}

	render := func() {
		for range 8 {
			win.BeginFrame()
			DrawChrome(win, s)
			win.EndFrame(0.1, 0.1, 0.1)
		}
	}

	// Both styles must draw the options without crashing (classic linear and default radial).
	s.SetClassicContextMenu(true)
	openMarkingMenuOnFirstFrame = true
	render()
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "mm-offset-options-classic.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}

	s.SetClassicContextMenu(false)
	openMarkingMenuOnFirstFrame = true
	render()
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "mm-offset-options-radial.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
