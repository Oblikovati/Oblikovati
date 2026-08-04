//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Format panel's three selection lists (#2015): line type, colour and thickness. Each shows
// the selection's current value with a preview and applies a value when a row is chosen, rather
// than running a command — which is why it is a control of its own and not the existing
// split-button variant flyout, whose entries are commands and which can show neither a current
// value nor a preview. A line type is picked by seeing its dash pattern.

// formatListHost is the session surface the lists read and drive (audit I5, the arrowSession
// pattern), so the control is testable with a small fake.
type formatListHost interface {
	FormatListSelection(kind app.FormatListKind) int
	ChooseFormatListEntry(kind app.FormatListKind, i int) int
}

var _ formatListHost = (*app.Session)(nil)

// selection-list geometry.
const (
	selectionListWidth   = 132 // total control width (px, before font scaling)
	selectionListPreview = 34  // width of the preview strip inside a row
	selectionListPadX    = 4
)

var selectionListPreviewColor = [4]float32{0.85, 0.87, 0.92, 1}

// drawSelectionList renders one Format-panel list: a combo showing the current row's preview and
// label, whose dropdown lists every row with its own preview. It returns true when the selection
// changed this frame.
func drawSelectionList(s formatListHost, kind app.FormatListKind, id string) bool {
	entries := app.FormatListEntries(kind)
	current := s.FormatListSelection(kind)
	if current < 0 || current >= len(entries) {
		current = 0
	}
	native.SetNextItemWidth(scaledSelectionListWidth())
	if !native.BeginCombo(id, entries[current].Label) {
		return false
	}
	chosen := selectionListRows(entries, kind, current)
	native.EndCombo()
	if chosen < 0 {
		return false
	}
	s.ChooseFormatListEntry(kind, chosen)
	return true
}

// selectionListRows draws every row and returns the index chosen this frame, or -1.
func selectionListRows(entries []app.FormatListEntry, kind app.FormatListKind, current int) int {
	chosen := -1
	for i, e := range entries {
		x, y := native.GetCursorScreenPos()
		if native.Selectable(e.Label+"##"+e.Label, i == current) {
			chosen = i
		}
		drawSelectionListPreview(e, kind, x, y)
	}
	return chosen
}

// drawSelectionListPreview paints a row's sample to the right of its label: a dash pattern for a
// line type, a filled swatch for a colour, a stroke of the right width for a thickness. The
// Default row previews nothing — there is no value to show.
func drawSelectionListPreview(e app.FormatListEntry, kind app.FormatListKind, x, y float32) {
	h := native.TextLineHeight()
	x0 := x + scaledSelectionListWidth() - selectionListPreview - selectionListPadX
	mid := y + h/2
	switch kind {
	case app.LineTypeList:
		drawPatternSample(app.FormatListPattern(e), x0, mid)
	case app.ColorList:
		drawColorSwatch(e, x0, y, h)
	default:
		drawWeightSample(e.LineWeight, x0, mid)
	}
}

// drawPatternSample strokes a short line in the row's dash pattern; a nil pattern draws nothing,
// which is how the Default row reads.
func drawPatternSample(pattern []float64, x0, mid float32) {
	if pattern == nil {
		return
	}
	drawDashedSample(pattern, x0, mid, selectionListPreview)
}

// drawDashedSample walks the dash pattern across the sample width, drawing the on-runs. A
// pattern's entries alternate on and off lengths, in millimetres, so the sample is scaled to fit
// rather than measured — it communicates the rhythm, not the true scale.
func drawDashedSample(pattern []float64, x0, mid, width float32) {
	total := 0.0
	for _, d := range pattern {
		total += absFloat(d)
	}
	if total == 0 {
		native.DrawLine(x0, mid, x0+width, mid, selectionListPreviewColor, 1.2)
		return
	}
	scale := float64(width) / total
	pos := x0
	for i, d := range pattern {
		run := float32(absFloat(d) * scale)
		if i%2 == 0 {
			native.DrawLine(pos, mid, pos+run, mid, selectionListPreviewColor, 1.2)
		}
		pos += run
	}
}

// drawColorSwatch fills the preview strip with the row's colour; the Default row draws nothing.
func drawColorSwatch(e app.FormatListEntry, x0, y, h float32) {
	if !e.Color.IsOverride() {
		return
	}
	rgba := e.Color.Rgba()
	native.DrawRectFilled(x0, y+2, x0+selectionListPreview, y+h-2,
		[4]float32{rgba.R, rgba.G, rgba.B, 1})
}

// drawWeightSample strokes a line of the row's width, so thicknesses compare by eye.
func drawWeightSample(weight float64, x0, mid float32) {
	if weight == 0 {
		return
	}
	native.DrawLine(x0, mid, x0+selectionListPreview, mid, selectionListPreviewColor, weightSampleThickness(weight))
}

// weightSampleThickness maps a plotted millimetre width onto a drawable pixel thickness, floored
// so the thinnest weight still shows and capped so the heaviest fits the row.
func weightSampleThickness(weight float64) float32 {
	px := float32(weight * 6)
	if px < 1 {
		return 1
	}
	if px > 6 {
		return 6
	}
	return px
}

// scaledSelectionListWidth is the control width at the current font scale, so the lists line up
// with the panel's other controls when the UI is scaled.
func scaledSelectionListWidth() float32 {
	return float32(scaledIconPx(selectionListWidth))
}

// absFloat is the unsigned magnitude of a dash length.
func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
