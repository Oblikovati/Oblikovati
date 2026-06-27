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

// driveRibbon opens a window of the given size, runs a few real chrome frames, and reports whether
// the ribbon ended up overflowing (its horizontal scrollbar showing). Skips when no GPU/display.
func driveRibbon(t *testing.T, w, h int) bool {
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
	defer func() { ribbonScrollbarShown = false }()
	s := ribbonSession(t)
	for i := 0; i < 5; i++ { // settle the dock layout + let the overflow flag stabilise
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
	return ribbonScrollbarShown
}

// TestRibbonShowsScrollbarWhenNarrow is the #1471 guard: a window too narrow to fit the active tab's
// buttons must overflow, so the band reports its horizontal scrollbar is showing.
func TestRibbonShowsScrollbarWhenNarrow(t *testing.T) {
	if !driveRibbon(t, 360, 600) {
		t.Error("a 360px-wide window should overflow the ribbon and show a horizontal scrollbar (#1471)")
	}
}

// TestRibbonNoScrollbarWhenWide is the converse: a wide window fits the active tab, so no scrollbar
// is shown and the band keeps its compact height.
func TestRibbonNoScrollbarWhenWide(t *testing.T) {
	if driveRibbon(t, 3200, 600) {
		t.Error("a 3200px-wide window should fit the default tab without a ribbon scrollbar (#1471)")
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
