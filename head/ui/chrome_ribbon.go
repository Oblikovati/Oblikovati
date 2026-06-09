//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
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
	if panel.Selector != nil {
		return drawSelectorPanel(panel)
	}
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

// selectorWidth is the pixel width of a ribbon selection-box (the Visual Style drop-down) —
// wide enough for the longest label ("Wireframe with Visible Edges Only").
const selectorWidth = 230

// drawSelectorPanel renders a panel as a labelled selection box (Inventor's combo control):
// a drop-down previewing the current choice with the panel title beneath, matching the button
// panels' layout. Choosing an option returns its command id so the caller runs it (which sets
// the new selection); the box reflects the session state on the next frame.
func drawSelectorPanel(panel app.RibbonPanel) string {
	sel := panel.Selector
	preview := ""
	if sel.SelectedIndex >= 0 && sel.SelectedIndex < len(sel.Options) {
		preview = sel.Options[sel.SelectedIndex].Label
	}
	var activated string
	native.BeginGroup()
	native.SetNextItemWidth(selectorWidth)
	if native.BeginCombo("##"+panel.Name, preview) {
		for i, opt := range sel.Options {
			if native.Selectable(opt.Label, i == sel.SelectedIndex) {
				activated = opt.CommandID
			}
			if opt.Tooltip != "" {
				native.SetItemTooltip(opt.Tooltip)
			}
		}
		native.EndCombo()
	}
	native.Text(panel.Name) // panel title under the selection box
	native.EndGroup()
	return activated
}

// drawRibbonButton renders one command in its configured style (text, small icon, or
// large icon), greyed when its predicate is false, with the command tooltip on hover.
// A command with variants renders as a split button: the head control plus a dropdown
// arrow listing the variant tools (Inventor's variant flyout). It returns the id of the
// command (head or chosen variant) clicked this frame, else "".
func drawRibbonButton(btn app.RibbonButton) string {
	native.BeginDisabled(!btn.Enabled)
	clicked := drawButtonControl(btn)
	native.EndDisabled()
	if clicked {
		return btn.Command.ID()
	}
	if len(btn.Variants) > 0 {
		return drawVariantDropdown(btn)
	}
	return ""
}

// variantArrowWidth is the pixel width of a split button's dropdown arrow box — just wide
// enough for the combo's caret, since its preview text is empty (the head shows the label).
const variantArrowWidth = 18

// drawVariantDropdown renders a split button's dropdown next to its head: a narrow combo
// whose entries are the variant tools. Choosing one returns its command id so the caller
// runs it; a disabled variant is shown greyed and is not selectable. Returns "" if nothing
// was chosen this frame.
func drawVariantDropdown(btn app.RibbonButton) string {
	native.SameLine()
	native.SetNextItemWidth(variantArrowWidth)
	var chosen string
	if native.BeginCombo("##"+btn.Command.ID()+"-variants", "") {
		for _, v := range btn.Variants {
			native.BeginDisabled(!v.Enabled)
			if native.Selectable(v.Label, false) {
				chosen = v.CommandID
			}
			native.EndDisabled()
			if v.Tooltip != "" {
				native.SetItemTooltip(v.Tooltip)
			}
		}
		native.EndCombo()
	}
	return chosen
}

// drawButtonControl draws the command's clickable control and its tooltip, returning
// whether it was clicked. An icon-style command falls back to a labeled text button
// when its glyph texture is unavailable (missing asset or upload failure), so a missing
// icon never hides the command.
func drawButtonControl(btn app.RibbonButton) bool {
	if btn.Active { // a toggled-on stateful control renders in the accent color
		native.PushStyleColor("Button", accentColor)
		native.PushStyleColor("ButtonHovered", accentColor)
		native.PushStyleColor("ButtonActive", accentColor)
		defer native.PopStyleColor(3)
	}
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
