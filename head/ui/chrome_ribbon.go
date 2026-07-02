//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// prevEnv tracks the ribbon environment across frames so the ribbon can auto-switch to the
// contextual tab on entry and back to the modelling tab on exit — but only on the transition
// frame, so the user can still pick tabs by hand afterwards.
var prevEnv app.Environment

// drawRibbon renders the ribbon as a fixed band pinned across the top of the window,
// directly under the menu bar — classic CAD ribbons are window chrome, not dockable
// palettes, so the band claims its slice of the work area each frame and the dockspace
// lays out beneath it. Inside the band: a tab strip, then each tab's panels of command
// buttons. A disabled command renders a disabled button (its predicate is re-evaluated
// every frame). Returns the id of a clicked command, or "".
func drawRibbon(s *app.Session) string {
	force := contextualTab(s)
	var activated string
	bandOpen := native.BeginRibbonBand("##ribbon", ribbonBandHeight())
	if bandOpen && native.BeginTabBar("##ribbon-tabs") {
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
	if bandOpen {
		// Remember (for next frame's height) whether the buttons overflow a too-narrow window, so
		// the band grows to seat the horizontal scrollbar without covering the panel names (#1471).
		ribbonScrollbarShown = native.ScrollMaxX() > 0
	}
	native.End()
	return activated
}

// ribbonScrollbarShown is last frame's overflow state: true when the ribbon's content was wider than
// the window, so a horizontal scrollbar is on screen and ribbonBandHeight reserves room for it.
var ribbonScrollbarShown bool

// ribbonMaxRows is how many small buttons stack in one panel column before the next
// column starts (the classic ribbon stacks its small buttons three deep).
const ribbonMaxRows = 3

// ribbonGridHeight is the height of the band's button-grid area: the tallest column
// shape, ribbonMaxRows rows of small icon buttons.
func ribbonGridHeight(m native.StyleMetrics) float32 {
	row := float32(scaledIconPx(smallIconPx)) + 2*m.FramePadY
	return ribbonMaxRows*row + (ribbonMaxRows-1)*m.ItemSpacingY
}

// ribbonBandHeight is the fixed height of the ribbon band: the tab strip, the button
// grid, and the panel-name strip, plus the paddings between them. Computed from the
// live style so a font or padding change can never clip the band's content.
func ribbonBandHeight() float32 {
	m := native.Metrics()
	content := ribbonGridHeight(m) + m.ItemSpacingY + native.TextLineHeight()
	h := 2*m.WindowPadY + native.FrameHeight() + m.ItemSpacingY + content
	if ribbonScrollbarShown {
		h += native.ScrollbarSize() // seat the horizontal scrollbar below the panel names (#1471)
	}
	return h
}

// drawTabPanels lays the tab's panels out horizontally — each panel is a layout group
// (button columns + name strip) separated by a full-height vertical divider. Every
// panel pins its name at the same band-bottom Y (labelY), which both matches the
// reference ribbon's footer strip and makes the dividers span the full band.
func drawTabPanels(panels []app.RibbonPanel) string {
	m := native.Metrics()
	_, gridTop := native.GetCursorScreenPos()
	labelY := gridTop + ribbonGridHeight(m) + m.ItemSpacingY
	var activated string
	for i, panel := range panels {
		if i > 0 {
			native.SameLine()
			native.SeparatorVertical()
			native.SameLine()
		}
		if id := drawPanel(panel, labelY); id != "" {
			activated = id
		}
	}
	return activated
}

// contextualTab returns the ribbon tab to force-select on the frame the ribbon environment
// changes: the Sketch / 3D Sketch tab on entering that environment, the Create & Modify tab on
// returning to the base environment. "" on a non-transition frame, so manual tab picks stick.
func contextualTab(s *app.Session) string {
	cur := app.CurrentEnvironment(s)
	if cur == prevEnv {
		return ""
	}
	prevEnv = cur
	switch cur {
	case app.SketchEnvironment:
		return "Sketch"
	case app.Sketch3DEnvironment:
		return "3D Sketch"
	default:
		return "Create & Modify"
	}
}

// packPanelColumns lays a panel's buttons into the ribbon's column-major flow: a large
// button stands alone as its own full-height column, while small/compact/text buttons
// stack up to ribbonMaxRows deep before the next column starts — so a registration
// order of Move, Copy, Rotate, Trim, … reads down each column like the reference ribbon.
func packPanelColumns(buttons []app.RibbonButton) [][]app.RibbonButton {
	var cols [][]app.RibbonButton
	var stack []app.RibbonButton
	flush := func() {
		if len(stack) > 0 {
			cols = append(cols, stack)
			stack = nil
		}
	}
	for _, b := range buttons {
		if b.Command.ButtonStyle() == app.LargeIconButton {
			flush()
			cols = append(cols, []app.RibbonButton{b})
			continue
		}
		stack = append(stack, b)
		if len(stack) == ribbonMaxRows {
			flush()
		}
	}
	flush()
	return cols
}

// drawPanel renders one ribbon panel as a self-contained layout group — its button
// columns with the panel name centered beneath in the band's footer strip — so the
// whole panel is one narrow, horizontally-placeable unit. Returns the id of a clicked
// command, or "".
func drawPanel(panel app.RibbonPanel, labelY float32) string {
	if panel.Selector != nil {
		return drawSelectorPanel(panel, labelY)
	}
	native.BeginGroup()
	activated := drawPanelColumns(packPanelColumns(panel.Buttons))
	drawPanelName(panel.Name, labelY)
	native.EndGroup()
	return activated
}

// drawPanelColumns draws the packed columns side by side, each column a vertical group
// of its buttons, and leaves the whole block as the last item so the caller can measure
// it (ItemRectMin/Max) to center the panel name.
func drawPanelColumns(cols [][]app.RibbonButton) string {
	var activated string
	native.BeginGroup()
	for i, col := range cols {
		if i > 0 {
			native.SameLine()
		}
		native.BeginGroup()
		for _, btn := range col {
			if id := drawRibbonButton(btn); id != "" {
				activated = id
			}
		}
		native.EndGroup()
	}
	native.EndGroup()
	return activated
}

// drawPanelName centers the panel name under the button block just drawn, pinned at
// the shared band-bottom labelY — the reference ribbon's panel-name footer strip.
func drawPanelName(name string, labelY float32) {
	x0, _ := native.ItemRectMin()
	x1, _ := native.ItemRectMax()
	off := ((x1 - x0) - native.CalcTextWidth(name)) / 2
	if off < 0 {
		off = 0
	}
	native.SetCursorScreenPos(x0+off, labelY)
	native.Text(name)
}

// selectorWidth is the pixel width of a ribbon selection-box (the Visual Style drop-down) —
// wide enough for the longest label ("Wireframe with Visible Edges Only").
const selectorWidth = 230

// drawSelectorPanel renders a panel as a labelled selection box (a ribbon combo
// control): a drop-down previewing the current choice with the panel name beneath,
// matching the button panels' layout. Choosing an option returns its command id so the
// caller runs it (which sets the new selection); the box reflects the session state on
// the next frame.
func drawSelectorPanel(panel app.RibbonPanel, labelY float32) string {
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
	drawPanelName(panel.Name, labelY)
	native.EndGroup()
	return activated
}

// drawRibbonButton renders one command in its configured style (text, small, large, or
// compact icon), greyed when its predicate is false, with the command tooltip on hover.
// A command with variants renders as a split button: the head control plus a dropdown
// arrow listing the variant tools (the variant flyout). It returns the id of the
// command (head or chosen variant) clicked this frame, else "".
func drawRibbonButton(btn app.RibbonButton) string {
	if btn.Command.Kind() == app.PopupControl {
		return drawPopupMenuButton(btn)
	}
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

// drawPopupMenuButton renders a PopupControl (M05-F03): the button itself runs nothing —
// it opens a menu of the control's resolved items, and choosing one returns that item's
// command id. Unlike a split button there is no head action, so the whole face is the
// menu trigger.
func drawPopupMenuButton(btn app.RibbonButton) string {
	native.BeginDisabled(!btn.Enabled)
	if drawButtonControl(btn) {
		native.OpenPopup(btn.Command.ID() + "##popup")
	}
	native.EndDisabled()
	var chosen string
	if native.BeginPopup(btn.Command.ID() + "##popup") {
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
		native.EndPopup()
	}
	return chosen
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
		native.PushStyleColor("Button", chromeTheme.accentColor)
		native.PushStyleColor("ButtonHovered", chromeTheme.accentColor)
		native.PushStyleColor("ButtonActive", chromeTheme.accentColor)
		defer native.PopStyleColor(3)
	}
	if px, ok := iconSizeFor(btn.Command.ButtonStyle()); ok {
		if tex, ok := icons.texture(btn.Command.Icon(), btn.Command.InlineIconSVG(), px); ok {
			return drawIconButton(btn, tex, float32(px))
		}
	}
	clicked := native.Button(btn.Command.DisplayName())
	setCommandTooltip(btn.Command)
	return clicked
}

// iconSizeFor returns the rasterization size for an icon button style, or false for a
// text-only command.
func iconSizeFor(s app.ButtonStyle) (int, bool) {
	switch s {
	case app.SmallIconButton, app.CompactIconButton:
		return scaledIconPx(smallIconPx), true
	case app.LargeIconButton:
		return scaledIconPx(largeIconPx), true
	default:
		return 0, false
	}
}

// identityTint draws an icon texture exactly as composed — the glyph's colors are
// baked in by iconCache.compose from the icon.* tokens, so no per-draw tinting.
var identityTint = [4]float32{1, 1, 1, 1}

// drawIconButton renders an icon button in its style: large is a captioned panel
// heading, small is an icon with its name beside it, and compact is the bare icon for
// dense grids like the constraint palette. The icon is the click target in every style.
func drawIconButton(btn app.RibbonButton, tex uint64, px float32) bool {
	switch btn.Command.ButtonStyle() {
	case app.LargeIconButton:
		return drawLargeIconButton(btn, tex, px)
	case app.SmallIconButton:
		return drawLabeledIconButton(btn, tex, px)
	default: // CompactIconButton
		clicked := native.ImageButton(btn.Command.ID(), tex, px, px, identityTint)
		setCommandTooltip(btn.Command)
		return clicked
	}
}

// drawLabeledIconButton renders a small button as icon + name side by side, the label
// vertically centered on the icon row — the stacked rows of a ribbon panel column.
func drawLabeledIconButton(btn app.RibbonButton, tex uint64, px float32) bool {
	m := native.Metrics()
	native.BeginGroup()
	clicked := native.ImageButton(btn.Command.ID(), tex, px, px, identityTint)
	setCommandTooltip(btn.Command)
	native.SameLine()
	x, y := native.GetCursorScreenPos()
	native.SetCursorScreenPos(x, y+(px+2*m.FramePadY-native.TextLineHeight())/2)
	native.Text(btn.Command.DisplayName())
	native.EndGroup()
	return clicked
}

// drawLargeIconButton renders a large button as a panel heading: the icon centered over
// its caption, the cell as wide as the wider of the two — so "Start 2D Sketch" doesn't
// hang off a 40px icon.
func drawLargeIconButton(btn app.RibbonButton, tex uint64, px float32) bool {
	m := native.Metrics()
	frameW := px + 2*m.FramePadX // ImageButton draws its frame padding around the glyph
	textW := native.CalcTextWidth(btn.Command.DisplayName())
	cellW := frameW
	if textW > cellW {
		cellW = textW
	}
	native.BeginGroup()
	x, y := native.GetCursorScreenPos()
	native.SetCursorScreenPos(x+(cellW-frameW)/2, y)
	clicked := native.ImageButton(btn.Command.ID(), tex, px, px, identityTint)
	setCommandTooltip(btn.Command)
	_, captionY := native.GetCursorScreenPos()
	native.SetCursorScreenPos(x+(cellW-textW)/2, captionY)
	native.Text(btn.Command.DisplayName())
	native.EndGroup()
	return clicked
}

// progressiveHoverDelay is how long a hover lasts (seconds) before the expanded
// text joins the tooltip — the ProgressiveToolTip behavior (M05-F09).
const progressiveHoverDelay = 1.2

// setCommandTooltip renders the command's tooltip for the last item: the optional
// title heads it, and the expanded text appears after a longer hover.
func setCommandTooltip(c *app.CommandDefinition) {
	text := c.Tooltip()
	if title := c.TooltipTitle(); title != "" {
		if text == "" {
			text = title
		} else {
			text = title + "\n" + text
		}
	}
	if expanded := c.TooltipExpanded(); expanded != "" && native.HoverSeconds() > progressiveHoverDelay {
		if text != "" {
			text += "\n\n"
		}
		text += expanded
	}
	native.SetItemTooltip(text)
}
