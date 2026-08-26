//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/compdef"
)

// TestDockablePanelsRenderThroughSharedPath drives the shared dockable-panel renderer with EVERY
// registered panel open for real frames, then drives the View menu. It is the #1473 live guard: the
// unified BeginClosable path must render every panel body (Model, Parameters, Materials, Lighting,
// the BOM/history/named-views/… panels, the Command REPL) without panicking, and an open panel must
// stay open across frames (the close-'X' is not clicked here, so nothing should close itself).
func TestDockablePanelsRenderThroughSharedPath(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false // rebuild the default layout for this fresh window/context
	icons = nil
	// Snapshot the head-local visibility globals so the test leaves no residue for others.
	defer func(b, m, p bool) { showBrowser, showMaterials, showPreferences = b, m, p }(showBrowser, showMaterials, showPreferences)

	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "panels-test.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	_ = s.Workspace().SetActiveDocument(pd)

	// Open every registered panel through its own setOpen so all bodies render this frame.
	for i := range dockablePanels {
		dockablePanels[i].setOpen(s, true)
	}
	for range 3 {
		win.BeginFrame()
		drawDockablePanels(s)
		if native.BeginMainMenuBar() {
			drawViewMenu(s)
			native.EndMainMenuBar()
		}
		win.EndFrame(0.1, 0.1, 0.1)
	}
	for i := range dockablePanels {
		p := &dockablePanels[i]
		if !p.isOpen(s) {
			t.Errorf("panel %q closed itself without the X being clicked", p.title)
		}
	}
}
