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
				if id := drawTabPanels(s, tab.Panels); id != "" {
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

const ribbonControlRows = 4

// ribbonGridHeight is the height of the band's button-grid area: the tallest column
// shape, ribbonMaxRows rows of small icon buttons.
func ribbonGridHeight(m native.StyleMetrics) float32 {
	buttonRow := float32(scaledIconPx(smallIconPx)) + 2*m.FramePadY
	buttonGrid := ribbonRowsHeight(ribbonMaxRows, buttonRow, m.ItemSpacingY)
	controlRow := maxF32(native.FrameHeight(), intensityChartHeight)
	controlGrid := ribbonRowsHeight(ribbonControlRows, controlRow, m.ItemSpacingY)
	return maxF32(buttonGrid, controlGrid)
}

func ribbonRowsHeight(rows int, row, spacing float32) float32 {
	if rows <= 0 {
		return 0
	}
	return float32(rows)*row + float32(rows-1)*spacing
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
func drawTabPanels(s pointCloudControlHost, panels []app.RibbonPanel) string {
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
		if id := drawPanel(s, panel, labelY); id != "" {
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
func drawPanel(s pointCloudControlHost, panel app.RibbonPanel, labelY float32) string {
	if panel.Name == app.PanelPointCloud {
		return drawPointCloudPanel(s, panel, labelY)
	}
	if panel.Selector != nil {
		return drawSelectorPanel(panel, labelY)
	}
	native.BeginGroup()
	activated := drawPanelColumns(packPanelColumns(panel.Buttons))
	drawPanelName(panel.Name, labelY)
	native.EndGroup()
	return activated
}

const (
	pointCloudSliderWidth = 82
	pointCloudComboWidth  = 124
)

// pointCloudControlHost is the session surface the ribbon's point-cloud controls mutate — the only
// session coupling in the whole panel-drawing path (audit I5, the arrowSession pattern). The
// density/size sliders and the intensity-ramp swatches write straight to session state on drag, so
// the drawing functions take this ≤6-method seam instead of the whole *app.Session and are testable
// against a small fake host.
type pointCloudControlHost interface {
	SetPointCloudRenderDensity(float32)
	SetPointCloudPointSize(float32)
	SetPointCloudIntensityRamp(low, high [4]float32)
}

var _ pointCloudControlHost = (*app.Session)(nil)

// drawPointCloudPanel renders the consolidated Point Cloud panel as one compact 3-column grid:
// Import (a small labeled-icon button, matching Move/Crop) heads the first column over the stacked
// "Size" and "Density" sliders, Move / Work Point sit over the display-mode selector in column two,
// and Crop / Fit Work Plane fill column three. Shrinking Import to a small icon lifts the sliders
// and the intensity ramp clear of the panel-name footer. The intensity ramp spans the full width
// beneath, but only when the target cloud is in Intensity mode (the app leaves IntensityRamp nil
// otherwise). Returns the id of a clicked tool or a chosen display mode, or "".
func drawPointCloudPanel(s pointCloudControlHost, panel app.RibbonPanel, labelY float32) string {
	var activated string
	pick := func(id string) {
		if got := drawPointCloudButton(panel, id); got != "" {
			activated = got
		}
	}
	native.BeginGroup()

	// The columns and ramp sit in one inner group so drawPanelName measures the full grid width
	// (ItemRectMin/Max span the whole block) and centers the panel name under it — without this
	// the last item measured would be column 3 alone and the label would sit off to the right.
	native.BeginGroup()

	native.BeginGroup() // column 1: Import over the stacked point-size ("Size") and Density sliders
	pick("PointCloud.Import")
	drawPointCloudSlider(s, "Size", panel.PointSizeSlider)
	drawPointCloudSlider(s, "Density", panel.Slider)
	native.EndGroup()

	native.SameLine()
	native.BeginGroup() // column 2: Move / Work Point over the display-mode selector
	pick("PointCloud.Move")
	pick("PointCloud.WorkPoint")
	if got := drawPointCloudSelector(panel); got != "" {
		activated = got
	}
	native.EndGroup()

	native.SameLine()
	native.BeginGroup() // column 3: Crop / Fit Work Plane
	pick("PointCloud.CropBox")
	pick("PointCloud.FitPlane")
	native.EndGroup()

	if panel.IntensityRamp != nil {
		drawIntensityRampControls(s, panel.IntensityRamp)
	}
	native.EndGroup() // close the measurable content block

	drawPanelName(panel.Name, labelY)
	native.EndGroup()
	return activated
}

// drawPointCloudButton draws the one panel button with the given command id in place, so the grid
// can seat each tool in its designated cell rather than the generic column-packing order. Returns
// the command id if clicked, else "".
func drawPointCloudButton(panel app.RibbonPanel, id string) string {
	for _, btn := range panel.Buttons {
		if btn.Command.ID() == id {
			return drawRibbonButton(btn)
		}
	}
	return ""
}

// drawPointCloudSlider draws a captioned display slider inline (bar then label) so the whole
// control stays one grid row, the caption trailing the bar to the right. The caption greys with the
// bar when the slider is disabled (no target cloud). A nil slider (session exposes none this frame)
// draws nothing.
func drawPointCloudSlider(s pointCloudControlHost, label string, slider *app.RibbonSlider) {
	if slider == nil {
		return
	}
	native.BeginDisabled(slider.Disabled)
	defer native.EndDisabled()
	drawRibbonSlider(s, slider, pointCloudSliderWidth)
	native.SameLine()
	native.Text(label)
}

// drawPointCloudSelector draws the display-mode combo (RGB / Intensity / …) for the grid,
// returning the chosen mode's command id, or "" when nothing changed or no selector exists.
func drawPointCloudSelector(panel app.RibbonPanel) string {
	sel := panel.Selector
	if sel == nil {
		return ""
	}
	preview := ""
	if sel.SelectedIndex >= 0 && sel.SelectedIndex < len(sel.Options) {
		preview = sel.Options[sel.SelectedIndex].Label
	}
	var activated string
	native.SetNextItemWidth(pointCloudComboWidth)
	if native.BeginCombo("##PointCloudDisplayMode", preview) {
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

func drawRibbonSlider(s pointCloudControlHost, slider *app.RibbonSlider, width float32) {
	native.BeginDisabled(slider.Disabled)
	defer native.EndDisabled()
	value := slider.Value
	native.SetNextItemWidth(width)
	var changed bool
	if slider.Percent {
		changed = native.SliderPercent("##"+slider.ID, &value, slider.Min, slider.Max)
	} else {
		changed = native.SliderFloat("##"+slider.ID, &value, slider.Min, slider.Max)
	}
	if changed {
		applyRibbonSlider(s, slider.ID, value)
	}
	if slider.Tooltip != "" {
		native.SetItemTooltip(slider.Tooltip)
	}
}

func applyRibbonSlider(s pointCloudControlHost, id string, value float32) {
	switch id {
	case "PointCloud.RenderDensity":
		s.SetPointCloudRenderDensity(value)
	case "PointCloud.PointSize":
		s.SetPointCloudPointSize(value)
	}
}

func drawIntensityRampControls(s pointCloudControlHost, ramp *app.RibbonColorRamp) {
	low, high := ramp.Low.Value, ramp.High.Value
	if native.ColorSwatch("##"+ramp.Low.ID, &low) {
		s.SetPointCloudIntensityRamp(low, high)
	}
	if ramp.Low.Tooltip != "" {
		native.SetItemTooltip(ramp.Low.Tooltip)
	}
	native.SameLine()
	drawIntensityHistogramChart(ramp.Histogram, low, high, intensityChartWidth, intensityChartHeight)
	native.SameLine()
	if native.ColorSwatch("##"+ramp.High.ID, &high) {
		s.SetPointCloudIntensityRamp(low, high)
	}
	if ramp.High.Tooltip != "" {
		native.SetItemTooltip(ramp.High.Tooltip)
	}
}

const (
	intensityChartWidth  = 92
	intensityChartHeight = 24
)

func drawIntensityHistogramChart(hist []float32, low, high [4]float32, width, height float32) {
	x, y := native.GetCursorScreenPos()
	bottom := y + height
	native.DrawRectFilled(x, y, x+width, bottom, [4]float32{0.08, 0.08, 0.08, 0.35})
	if len(hist) > 0 {
		drawIntensityHistogramArea(hist, low, high, x, y, width, height)
	}
	native.DrawLine(x, bottom-1, x+width, bottom-1, [4]float32{0.95, 0.95, 0.95, 0.55}, 1)
	native.Dummy(width, height)
}

func drawIntensityHistogramArea(hist []float32, low, high [4]float32, x, y, width, height float32) {
	if len(hist) == 1 {
		top := y + height*(1-clampChart01(hist[0]))
		native.DrawRectFilled(x, top, x+width, y+height, low)
		return
	}
	for i := 0; i < len(hist)-1; i++ {
		x0 := x + width*float32(i)/float32(len(hist)-1)
		x1 := x + width*float32(i+1)/float32(len(hist)-1)
		y0 := y + height*(1-clampChart01(hist[i]))
		y1 := y + height*(1-clampChart01(hist[i+1]))
		c := lerpChartColor(low, high, float32(i)/float32(len(hist)-1))
		native.DrawQuadFilled(x0, y+height, x1, y+height, x1, y1, x0, y0, c)
	}
}

func clampChart01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerpChartColor(a, b [4]float32, t float32) [4]float32 {
	return [4]float32{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
		0.8,
	}
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
