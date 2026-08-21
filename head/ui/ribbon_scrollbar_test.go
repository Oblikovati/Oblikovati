//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/compdef"
)

// ribbonSession is a session with one part open, so the ribbon builds its full tab/panel set.
func ribbonSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil { // populate the ribbon's tabs/panels
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	pd, err := compdef.AddPart(s.Workspace(), "ribbon.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	return s
}

// driveRibbon opens a window of the given size, runs a few real chrome frames, and reports how
// the ribbon settled: how many of the active tab's panels collapsed to flyout tiles, and whether
// the band still overflowed after that (its horizontal scrollbar showing). Skips when no
// GPU/display.
func driveRibbon(t *testing.T, w, h int) (collapsed int, scrollbar bool) {
	t.Helper()
	win, err := native.CreateWindow(w, h, "ribbon-scroll-test")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil
	ribbonScrollbarShown = false
	ribbonCollapsedPanels = 0
	clear(ribbonPanelWidth) // measured widths are per-style, not per-window, but start each case clean
	defer func() {
		ribbonScrollbarShown = false
		ribbonCollapsedPanels = 0
		clear(ribbonPanelWidth)
	}()
	s := ribbonSession(t)
	for i := 0; i < 5; i++ { // settle the dock layout + let the fit decision stabilise
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	return ribbonCollapsedPanels, ribbonScrollbarShown
}

// TestRibbonCollapsesPanelsWhenNarrow is the overflow guard: a window too narrow to fit the
// active tab's panels must collapse panels to flyout tiles, so every command stays reachable
// rather than running off the edge of the band.
func TestRibbonCollapsesPanelsWhenNarrow(t *testing.T) {
	if collapsed, _ := driveRibbon(t, 360, 600); collapsed == 0 {
		t.Error("a 360px-wide window should collapse ribbon panels to flyout tiles, none collapsed")
	}
}

// TestRibbonCollapseAvoidsTheScrollbar is the point of the collapse system: at a width that
// cannot seat the tab's panels expanded, collapsing them must bring the band back within the
// window — so the #1471 horizontal scrollbar stays the last-resort tier rather than the routine
// overflow story it had become (it showed even on a maximized 3840px 4K window).
func TestRibbonCollapseAvoidsTheScrollbar(t *testing.T) {
	collapsed, scrollbar := driveRibbon(t, 1400, 600)
	if collapsed == 0 {
		t.Fatal("a 1400px-wide window should have collapsed at least one panel")
	}
	if scrollbar {
		t.Error("collapsing panels should have fitted the band; the scrollbar is still showing")
	}
}

// TestRibbonNoScrollbarWhenWide is the converse: a wide window fits the active tab, so nothing
// collapses, no scrollbar is shown, and the band keeps its compact height.
func TestRibbonNoScrollbarWhenWide(t *testing.T) {
	collapsed, scrollbar := driveRibbon(t, 3200, 600)
	if scrollbar {
		t.Error("a 3200px-wide window should fit the default tab without a ribbon scrollbar (#1471)")
	}
	if collapsed != 0 {
		t.Errorf("a 3200px-wide window collapsed %d panels, want 0 (everything fits)", collapsed)
	}
}

// TestRibbonBandHeightReservesScrollbar checks the height arithmetic directly: when the scrollbar is
// showing the band grows by exactly the scrollbar thickness, so the panel-name strip is never hidden
// behind it.
func TestRibbonBandHeightReservesScrollbar(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	defer func() { ribbonScrollbarShown = false }()

	win.BeginFrame()
	ribbonScrollbarShown = false
	without := ribbonBandHeight()
	ribbonScrollbarShown = true
	with := ribbonBandHeight()
	bar := native.ScrollbarSize()
	win.EndFrame(0.1, 0.1, 0.1)

	if got := with - without; got != bar {
		t.Errorf("band height grew by %.2f reserving the scrollbar, want exactly ScrollbarSize=%.2f", got, bar)
	}
}
