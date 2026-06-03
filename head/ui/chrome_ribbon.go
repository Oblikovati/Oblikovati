//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// prevInSketch tracks the sketch-environment state across frames so the ribbon can
// auto-switch to the contextual Sketch tab on entry and back to 3D Model on exit — but
// only on the transition frame, so the user can still pick tabs by hand afterwards.
var prevInSketch bool

// drawRibbon renders the ribbon as Inventor's two-level layout: a tab bar of command
// tabs, each tab holding its panels of command buttons. A disabled command renders a
// disabled button (its predicate is re-evaluated every frame). The contextual Sketch
// tab is auto-selected when entering the sketch environment. Returns the id of a
// clicked command, or "".
func drawRibbon(s *app.Session) string {
	force := contextualTab(s)
	var activated string
	if native.Begin("Ribbon") && native.BeginTabBar("##ribbon-tabs") {
		for _, tab := range app.BuildRibbon(s).Tabs {
			if native.BeginTabItemSelected(tab.Name, tab.Name == force) {
				if id := drawTabPanels(tab.Panels); id != "" {
					activated = id
				}
				native.EndTabItem()
			}
		}
		native.EndTabBar()
	}
	native.End()
	return activated
}

// drawTabPanels lays the tab's panels out horizontally — each panel is a layout group
// (button row + title) and panels sit SameLine with a vertical divider between them, so
// no panel is pushed off-screen the way a vertical stack hid the Sketch tab's Exit panel.
func drawTabPanels(panels []app.RibbonPanel) string {
	var activated string
	for i, panel := range panels {
		if i > 0 {
			native.SameLine()
			native.SeparatorVertical()
			native.SameLine()
		}
		if id := drawPanel(panel); id != "" {
			activated = id
		}
	}
	return activated
}

// contextualTab returns the ribbon tab to force-select this frame when the sketch
// environment was just entered ("Sketch") or left ("3D Model"), else "".
func contextualTab(s *app.Session) string {
	cur := s.InSketch()
	if cur == prevInSketch {
		return ""
	}
	prevInSketch = cur
	if cur {
		return "Sketch"
	}
	return "3D Model"
}

// ribbonMaxRows caps how many button rows a panel uses (Inventor stacks small buttons a
// few rows deep); the column count grows to fit, keeping each panel narrow so panels sit
// side-by-side without running off the ribbon.
const ribbonMaxRows = 3

// panelCols returns how many columns to wrap a panel's n buttons into, bounded so the
// panel is at most ribbonMaxRows tall.
func panelCols(n int) int {
	if n <= ribbonMaxRows {
		return 1
	}
	return (n + ribbonMaxRows - 1) / ribbonMaxRows
}

// drawPanel renders one ribbon panel as a self-contained layout group: a compact grid of
// command buttons with the panel title beneath them (Inventor's panel layout), so the
// whole panel is one narrow, horizontally-placeable unit. The title uses plain Text (not
// SeparatorText, which would stretch the group to the full window width and hide every
// panel to its right). Returns the id of a clicked command, or "".
func drawPanel(panel app.RibbonPanel) string {
	var activated string
	cols := panelCols(len(panel.Buttons))
	native.BeginGroup()
	for i, btn := range panel.Buttons {
		if i > 0 && i%cols != 0 {
			native.SameLine() // continue the current row; a new row starts at each multiple of cols
		}
		if id := drawRibbonButton(btn); id != "" {
			activated = id
		}
	}
	native.Text(panel.Name) // panel title under its buttons
	native.EndGroup()
	return activated
}

// drawRibbonButton renders one command in its configured style (text, small icon, or
// large icon), greyed when its predicate is false, with the command tooltip on hover.
// It returns the command id when clicked this frame, else "".
func drawRibbonButton(btn app.RibbonButton) string {
	native.BeginDisabled(!btn.Enabled)
	clicked := drawButtonControl(btn)
	native.EndDisabled()
	if clicked {
		return btn.Command.ID()
	}
	return ""
}

// drawButtonControl draws the command's clickable control and its tooltip, returning
// whether it was clicked. An icon-style command falls back to a labeled text button
// when its glyph texture is unavailable (missing asset or upload failure), so a missing
// icon never hides the command.
func drawButtonControl(btn app.RibbonButton) bool {
	if px, ok := iconSizeFor(btn.Command.ButtonStyle()); ok {
		if tex, ok := icons.texture(btn.Command.Icon(), px); ok {
			return drawIconButton(btn, tex, float32(px))
		}
	}
	clicked := native.Button(btn.Command.DisplayName())
	native.SetItemTooltip(btn.Command.Tooltip())
	return clicked
}

// iconSizeFor returns the rasterization size for an icon button style, or false for a
// text-only command.
func iconSizeFor(s app.ButtonStyle) (int, bool) {
	switch s {
	case app.SmallIconButton:
		return smallIconPx, true
	case app.LargeIconButton:
		return largeIconPx, true
	default:
		return 0, false
	}
}

// drawIconButton renders an icon button: small ones are icon-only (dense tool grids),
// large ones place the name as a caption beneath the icon (Inventor's large button).
// The icon is the click target either way.
func drawIconButton(btn app.RibbonButton, tex uint64, px float32) bool {
	if btn.Command.ButtonStyle() != app.LargeIconButton {
		clicked := native.ImageButton(btn.Command.ID(), tex, px, px, iconTint)
		native.SetItemTooltip(btn.Command.Tooltip())
		return clicked
	}
	native.BeginGroup()
	clicked := native.ImageButton(btn.Command.ID(), tex, px, px, iconTint)
	native.SetItemTooltip(btn.Command.Tooltip())
	native.Text(btn.Command.DisplayName())
	native.EndGroup()
	return clicked
}
