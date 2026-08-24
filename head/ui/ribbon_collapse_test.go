//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "testing"

// TestPanelsThatFitEverythingFits: a band wide enough for every panel collapses none.
func TestPanelsThatFitEverythingFits(t *testing.T) {
	widths := []float32{100, 200, 300}
	// 600 of panels + 2 dividers of 10 = 620.
	if got := panelsThatFit(widths, 10, 50, 620); got != 3 {
		t.Errorf("panelsThatFit at exactly the needed width = %d, want 3 (all expanded)", got)
	}
	if got := panelsThatFit(widths, 10, 50, 5000); got != 3 {
		t.Errorf("panelsThatFit in a very wide band = %d, want 3 (all expanded)", got)
	}
}

// TestPanelsThatFitCollapsesFromTheRight: one pixel too narrow collapses the LAST panel only,
// and the leading panels keep their full tiles.
func TestPanelsThatFitCollapsesFromTheRight(t *testing.T) {
	widths := []float32{100, 200, 300}
	// 619 cannot seat all three (620); with the last collapsed the row is 100+200+50+20 = 370.
	if got := panelsThatFit(widths, 10, 50, 619); got != 2 {
		t.Errorf("panelsThatFit one pixel short = %d, want 2 (rightmost panel collapsed)", got)
	}
	// 369 cannot seat two expanded either; one expanded is 100+50+50+20 = 220.
	if got := panelsThatFit(widths, 10, 50, 369); got != 1 {
		t.Errorf("panelsThatFit at 369 = %d, want 1 (two rightmost panels collapsed)", got)
	}
}

// TestPanelsThatFitAllCollapsed: when not even an all-collapsed row fits, every panel collapses
// and the band's horizontal scrollbar (#1471) is the last-resort tier beneath it.
func TestPanelsThatFitAllCollapsed(t *testing.T) {
	widths := []float32{100, 200, 300}
	if got := panelsThatFit(widths, 10, 50, 40); got != 0 {
		t.Errorf("panelsThatFit in a hopelessly narrow band = %d, want 0 (all collapsed)", got)
	}
}

// TestPanelsThatFitUnmeasuredPanelIsAssumedToFit: a panel the ribbon has never drawn has no
// cached width (0), so it must be drawn — and hence measured — rather than pre-emptively
// collapsed, otherwise a freshly-opened tab could never learn its own widths.
func TestPanelsThatFitUnmeasuredPanelIsAssumedToFit(t *testing.T) {
	if got := panelsThatFit([]float32{0, 0, 0}, 10, 50, 30); got != 3 {
		t.Errorf("panelsThatFit with no measurements = %d, want 3 (draw once to measure)", got)
	}
}

// TestPanelsThatFitEmptyTab guards the degenerate tab (no panels → nothing to expand).
func TestPanelsThatFitEmptyTab(t *testing.T) {
	if got := panelsThatFit(nil, 10, 50, 500); got != 0 {
		t.Errorf("panelsThatFit on an empty tab = %d, want 0", got)
	}
}

// TestRibbonRowWidthCountsDividersOnce checks the arithmetic the fit decision rests on: n-1
// dividers for n panels, whatever their collapse state.
func TestRibbonRowWidthCountsDividersOnce(t *testing.T) {
	widths := []float32{100, 200}
	if got := ribbonRowWidth(widths, 2, 10, 50); got != 310 {
		t.Errorf("ribbonRowWidth(all expanded) = %v, want 310 (100+200+one 10px divider)", got)
	}
	if got := ribbonRowWidth(widths, 0, 10, 50); got != 110 {
		t.Errorf("ribbonRowWidth(all collapsed) = %v, want 110 (50+50+one 10px divider)", got)
	}
	if got := ribbonRowWidth([]float32{100}, 1, 10, 50); got != 100 {
		t.Errorf("ribbonRowWidth(single panel) = %v, want 100 (no divider)", got)
	}
}

// TestRibbonPanelWidthKeyIsPerTab: a command may be registered onto several tabs
// (app/ribbon.go ribbonBuilder.add), so a panel of the same name can appear on two tabs with
// different contents — and must not share one cached width.
func TestRibbonPanelWidthKeyIsPerTab(t *testing.T) {
	if ribbonPanelWidthKey("Create & Modify", "Sketch") == ribbonPanelWidthKey("Surfaces & Mesh", "Sketch") {
		t.Error("the same panel name on two tabs shares one width-cache key")
	}
}

// TestClampFlyoutXStaysOnScreen: a flyout wider than the room to the right of its tile is
// pulled left rather than allowed off the display — an explicit SetNextWindowPos defeats Dear
// ImGui's own clamping, so this arithmetic is the only thing keeping the flyout reachable.
func TestClampFlyoutXStaysOnScreen(t *testing.T) {
	const vpW = 3840 // this machine's 4K display, where the overflow defect was reproduced
	if got := clampFlyoutX(10, 400, vpW); got != 10 {
		t.Errorf("a flyout that fits where it drops moved to %v, want 10", got)
	}
	if got := clampFlyoutX(2147, 1747, vpW); got != vpW-1747 {
		t.Errorf("the overhanging Modify flyout landed at %v, want %v (right edge on screen)",
			got, float32(vpW-1747))
	}
	if got := clampFlyoutX(2000, 2*vpW, vpW); got != 0 {
		t.Errorf("a flyout wider than the display landed at %v, want 0 (never past the left edge)", got)
	}
	if got := clampFlyoutX(123, 0, vpW); got != 123 {
		t.Errorf("an unmeasured flyout moved to %v, want 123 (drop straight down)", got)
	}
}
