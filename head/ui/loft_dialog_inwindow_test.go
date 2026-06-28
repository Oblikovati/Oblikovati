//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/head/internal/native"
)

// TestInWindowLoftDialogRenders drives the reworked Loft panel (#1521) through real frames in the
// live window so its draw path — the Curves tab's ordered Sections list, the per-row Remove/reorder
// affordances, the guide group, and the Conditions tab — is exercised (and credited with coverage by
// the xvfb+lavapipe CI head job). It asserts the section API the dialog draws on stays consistent as
// the rendered panel mutates it. Skips cleanly when no display/Vulkan is available.
func TestInWindowLoftDialogRenders(t *testing.T) {
	win, err := native.CreateWindow(1100, 800, "obk-loft-dialog-inwindow")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil

	s := loftThreeSectionSession(t)
	l := s.ActiveLoft()
	if l == nil {
		t.Fatal("loft tool is not active")
	}

	// A few full-chrome frames render the panel as the user sees it: drawLoftDialog → the Curves tab
	// (sections list rows, guides, options). refreshLoftUI seeds loftUI here, so the direct tab-body
	// draws below start from a valid editor state.
	for i := 0; i < 3; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if l.SectionCount() != 3 {
		t.Fatalf("seeded section count = %d, want 3", l.SectionCount())
	}

	// ImGui renders only the selected tab, so DrawChrome alone hits just Curves. Drive both tab
	// bodies directly — with a row selected (the Remove-section affordance) and then closed (the
	// no-end-sections note) — so the whole dialog draw path is covered.
	loftUI.open = true
	loftUI.selectedSection = 1
	loftCoverFrame(win, func() {
		drawLoftCurvesTab(l)
		drawLoftConditionsTab(s, l)
	})
	l.SetClosed(true)
	loftCoverFrame(win, func() { drawLoftConditionsTab(s, l) })
	l.SetClosed(false)

	// Removing the selected row through the API the dialog button calls keeps the list consistent.
	l.RemoveSection(1)
	if l.SectionCount() != 2 {
		t.Errorf("after RemoveSection, count = %d, want 2", l.SectionCount())
	}

	// The empty-list state renders its prompt instead of rows.
	l.ClearSections()
	loftCoverFrame(win, func() { drawLoftSectionsList(l) })
	if l.SectionCount() != 0 {
		t.Errorf("after ClearSections, count = %d, want 0", l.SectionCount())
	}
}

// loftCoverFrame renders one window frame whose body is drawn inside a plain panel, so a dialog
// sub-section can be exercised on its own (ImGui shows only the active tab during DrawChrome).
func loftCoverFrame(win *native.Window, body func()) {
	win.BeginFrame()
	if native.Begin("##loft-cover") {
		body()
	}
	native.End()
	win.EndFrame(0.1, 0.1, 0.12)
}
