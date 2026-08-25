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
	primeViewportSize(t, w, h) // ImGui's viewport size lags a window behind; prime it to w×h first
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
	return settleRibbon(t, win, s)
}

// primeViewportSize works around a harness limitation: ImGui's main-viewport size reflects the
// PREVIOUS window created in the process, not the one being drawn (the new offscreen window's
// framebuffer extent never reaches io.DisplaySize within a test). A throwaway window of the target
// size, rendered one frame then destroyed, becomes that "previous" window — so the real window below
// measures at the intended w×h instead of whatever earlier test's size leaked in. Without this the
// ribbon-fit tests measured against a stale width and were order-dependent.
func primeViewportSize(t *testing.T, w, h int) {
	prime, err := native.CreateWindow(w, h, "ribbon-scroll-prime")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	prime.InitViewport()
	prime.BeginFrame()
	prime.EndFrame(0.1, 0.1, 0.1)
	prime.Destroy()
}

// settleRibbon renders chrome frames until the ribbon's fit decision reaches STEADY STATE — the
// collapsed-panel count and scrollbar flag unchanged for settleStableFrames consecutive frames —
// then returns that settled result. A FIXED frame count is the wrong precondition: the fit reads the
// dock layout's content-region width, which needs an unknown, machine-dependent number of frames to
// lay out (and the band-height↔scrollbar feedback adds a frame of hysteresis), so a fixed five frames
// sometimes sampled BEFORE the layout settled and the result flipped run-to-run (the #1471 tests'
// order-dependent flakiness). Waiting for a stable state makes it deterministic. maxSettleFrames caps
// the wait so a genuine non-convergence fails loudly rather than hanging.
func settleRibbon(t *testing.T, win *native.Window, s *app.Session) (collapsed int, scrollbar bool) {
	t.Helper()
	const settleStableFrames, maxSettleFrames = 4, 240
	stable, prevCollapsed, prevScroll := 0, -1, false
	for i := 0; i < maxSettleFrames && stable < settleStableFrames; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
		if ribbonCollapsedPanels == prevCollapsed && ribbonScrollbarShown == prevScroll {
			stable++
			continue
		}
		stable, prevCollapsed, prevScroll = 0, ribbonCollapsedPanels, ribbonScrollbarShown
	}
	if stable < settleStableFrames {
		t.Fatalf("ribbon fit never settled in %d frames (collapsed=%d scrollbar=%v); it is oscillating",
			maxSettleFrames, ribbonCollapsedPanels, ribbonScrollbarShown)
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
