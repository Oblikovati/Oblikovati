//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Ribbon panel collapse — the ribbon's answer to a tab wider than the window.
//
// Reference behaviour. Inventor's ribbon can be "minimized to tabs / panel titles / panel
// buttons" (C-inventor-ui-reference.md:29, citations [2][29]), and a panel that cannot show
// everything puts the remainder behind "a drop-down arrow for additional commands"
// (C-inventor-ui-reference.md:8 and :27, citation [1][2]; the flyout's own affordances —
// expand, pushpin, tear-off — are C-inventor-ui-reference.md:183, citation [2][29][64]). The
// collapsed form is therefore a documented part of Inventor's ribbon: a panel shrunk to one
// button whose drop-down flies its full contents out.
//
// Deliberate v1 divergence. Inventor's minimize is a GLOBAL, USER-CHOSEN ribbon state (picked
// from the right-click Ribbon Appearance menu) and applies to every panel at once; the
// reference doc does not document a width-driven, per-panel automatic resize, so we do not
// claim one. What Oblikovati does here is apply the same collapsed FORM automatically and per
// panel: when the active tab's panels are wider than the band, the rightmost panels collapse
// to flyout tiles until the rest fit. That keeps every command reachable at any window width —
// the defect this replaces let the right-hand panels run off a maximized 3840px window with
// only a thin accent-coloured scrollbar to hint at it. A user-selectable minimize state and
// Inventor's medium button tier are deferred; see F-visual-gaps-implementation.md.
//
// Measure-then-decide. Dear ImGui is immediate mode, so a panel's laid-out width is only
// knowable after it has been drawn. Frame N measures each expanded panel and caches its width;
// frame N+1 decides from the cache. The ribbon already uses this idiom for its overflow
// scrollbar (chrome_ribbon.go, ribbonScrollbarShown), and it converges in one frame because a
// panel's expanded width does not depend on how many of its neighbours are expanded.

// ribbonPanelWidth caches each panel's measured expanded width in screen pixels, keyed by tab
// and panel name (panels are unique per tab; the tab is in the key because a command may be
// registered onto several tabs, app/ribbon.go ribbonBuilder.add). A panel absent from the map
// has never been drawn: it is assumed to fit so that it gets drawn once and measured.
var ribbonPanelWidth = map[string]float32{}

// ribbonCollapsedPanels is how many of the active tab's panels the last frame drew collapsed —
// 0 when the whole tab fitted the band. Kept at package scope, like ribbonScrollbarShown, so a
// test can drive real chrome frames at a given window width and assert that collapse engaged.
var ribbonCollapsedPanels int

func ribbonPanelWidthKey(tab, panel string) string { return tab + "\x00" + panel }

// cachedPanelWidths returns the measured expanded widths of a tab's panels in order, with 0
// for a panel not yet measured.
func cachedPanelWidths(tab string, panels []app.RibbonPanel) []float32 {
	out := make([]float32, len(panels))
	for i, p := range panels {
		out[i] = ribbonPanelWidth[ribbonPanelWidthKey(tab, p.Name)]
	}
	return out
}

// panelsThatFit reports how many LEADING panels of a tab can stay expanded within avail
// pixels, the rest collapsing to flyout tiles of width collapsed with sep pixels of divider
// between every pair. Panels collapse from the right, so a tab's leading panels — the ones
// Inventor puts first because they carry the tab's primary task — keep their full tiles
// longest. widths[i] is panel i's measured expanded width, or 0 when it has never been drawn.
// Returns len(widths) when everything fits, and 0 when even an all-collapsed row does not (the
// band's horizontal scrollbar is the last-resort tier under that, #1471).
func panelsThatFit(widths []float32, sep, collapsed, avail float32) int {
	for n := len(widths); n > 0; n-- {
		if ribbonRowWidth(widths, n, sep, collapsed) <= avail {
			return n
		}
	}
	return 0
}

// ribbonRowWidth is the laid-out width of a tab whose first n panels are expanded and whose
// remaining panels are collapsed, dividers included.
func ribbonRowWidth(widths []float32, n int, sep, collapsed float32) float32 {
	total := float32(0)
	for i, w := range widths {
		if i < n {
			total += w
			continue
		}
		total += collapsed
	}
	if len(widths) > 1 {
		total += sep * float32(len(widths)-1)
	}
	return total
}

// ribbonSeparatorWidth is the horizontal cost of the divider drawn between two panels: the
// rule itself plus the item spacing on either side of it (drawTabPanels lays it out as
// SameLine / SeparatorVertical / SameLine).
func ribbonSeparatorWidth(m native.StyleMetrics) float32 { return 2*m.ItemSpacingX + 1 }

// collapsedPanelWidth is the width of one collapsed panel's tile: wide enough for the panel
// name that sits beneath it in the band's footer strip, and never narrower than a large icon
// button, so a row of collapsed tiles keeps the band's rhythm.
func collapsedPanelWidth(name string, m native.StyleMetrics) float32 {
	w := native.CalcTextWidth(name) + 2*m.FramePadX
	if min := float32(scaledIconPx(largeIconPx)) + 2*m.FramePadX; w < min {
		w = min
	}
	return w
}

// widestCollapsedPanel is the collapsed-tile width used for the fit arithmetic: the widest
// tile any of the tab's panels would need, so the estimate can never under-count and leave the
// row overflowing after a collapse.
func widestCollapsedPanel(panels []app.RibbonPanel, m native.StyleMetrics) float32 {
	var w float32
	for _, p := range panels {
		if c := collapsedPanelWidth(p.Name, m); c > w {
			w = c
		}
	}
	return w
}

// ribbonCaretPx is the half-width of a ribbon drop-down caret, before the
// icon-scale preference is applied. The caret is drawn with the draw list rather than typed as
// a glyph because the chrome font carries no guaranteed arrow codepoint.
const ribbonCaretPx = 7

// drawCollapsedPanel renders a panel in its collapsed form: one full-height tile carrying a
// drop-down caret, the panel name pinned in the band's footer strip beneath it — so a
// collapsed panel reads in the same grammar as an expanded one, name and all — and a flyout
// holding the panel's real contents. Returns the id of a command activated inside the flyout,
// or "".
func drawCollapsedPanel(s ribbonControlHost, panel app.RibbonPanel, tabName string, labelY, gridH float32) string {
	m := native.Metrics()
	w := collapsedPanelWidth(panel.Name, m)
	id := "##ribbon-panel-flyout-" + panel.Name

	native.BeginGroup()
	x, y := native.GetCursorScreenPos()
	if native.ButtonSized(id, w, gridH) {
		native.OpenPopup(id)
	}
	native.SetItemTooltip(panel.Name + " — show this panel's commands")
	drawRibbonCaret(x+w/2, y+gridH/2)
	// The tile is still the last item here (the caret is a draw-list call and the tooltip
	// registers none), so the name centres under the tile exactly as it does under a grid.
	drawPanelName(panel.Name, labelY)
	native.EndGroup()
	return drawCollapsedPanelFlyout(s, panel, id, tabName, x, labelY, m)
}

// drawCollapsedPanelFlyout opens the collapsed panel's flyout (popup id) — a window of its own,
// opened outside the tile's layout group and pinned just under the band rather than at the pointer,
// because Inventor drops a panel's flyout from the panel. Its left edge is pulled back when the panel
// is too wide to drop straight down: an explicit SetNextWindowPos overrides Dear ImGui's own on-screen
// clamp, so the flyout would otherwise run off the right of the display — the failure this whole
// system exists to end. Returns the id of a command activated inside it, or "".
func drawCollapsedPanelFlyout(s ribbonControlHost, panel app.RibbonPanel, id, tabName string, x, labelY float32, m native.StyleMetrics) string {
	vpW, _ := native.MainViewportSize()
	flyoutX := clampFlyoutX(x, ribbonPanelWidth[ribbonPanelWidthKey(tabName, panel.Name)], vpW)
	native.SetNextWindowPos(flyoutX, labelY+native.TextLineHeight()+m.ItemSpacingY)
	var activated string
	if native.BeginPopup(id) {
		if got := drawPanelFlyoutBody(s, panel); got != "" {
			activated = got
			native.CloseCurrentPopup()
		}
		native.EndPopup()
	}
	return activated
}

// clampFlyoutX places a flyout of the given content width inside a viewport vpW wide: dropping
// straight down from x when it fits there, otherwise pulled left until its right edge is on
// screen, and never past the left edge. A width of 0 (the panel has somehow never been measured
// expanded) drops straight down and leaves the rest to Dear ImGui. The viewport width is a
// parameter rather than a native call so the arithmetic is testable without a window.
func clampFlyoutX(x, width, vpW float32) float32 {
	if width <= 0 {
		return x
	}
	if right := x + width; right > vpW {
		x = vpW - width
	}
	if x < 0 {
		x = 0
	}
	return x
}

// drawRibbonCaret draws a downward drop-down caret centred at (cx, cy) — the collapsed panel's
// tile and a split button's variant arrow — in the theme accent, so it reads as the interactive
// drop-down it is. Drawn rather than typed because the chrome font carries no arrow codepoint.
func drawRibbonCaret(cx, cy float32) {
	half := float32(scaledIconPx(ribbonCaretPx))
	native.DrawTriangleFilled(cx-half, cy-half/2, cx+half, cy-half/2, cx, cy+half/2, chromeTheme.accentColor)
}
