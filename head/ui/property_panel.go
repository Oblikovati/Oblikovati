//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/head/internal/native"

// The feature property panels (the Extrusion panel first) share this widget kit: a
// breadcrumb header, collapsible sections, label/control rows on a fixed column grid,
// selection chips that show the pick state (filled / required-but-empty), and rows of
// icon toggle buttons. The layout mirrors the reference panel: section headers span the
// panel, labels sit in a left column, controls align in a right column.

// propertyLabelWidth is the left label column width; propertyFieldWidth sizes numeric
// inputs in the control column so unit suffixes fit beside them.
const (
	propertyLabelWidth = 95
	propertyFieldWidth = 110
)

// propertySection draws a full-width collapsible section header, open by default the
// first time it is shown. Draw the section's rows only when it returns true.
func propertySection(title string) bool {
	native.SetNextItemOpen(true, true)
	return native.CollapsingHeader(title)
}

// propertyRow starts a "label left, control right" row: it draws the label and moves
// the cursor to the control column, so the caller's next widget aligns with every
// other row's control.
func propertyRow(label string) {
	x, y := native.GetCursorScreenPos()
	native.Text(label)
	native.SameLine()
	native.SetCursorScreenPos(x+propertyLabelWidth, y)
}

// propertyFloatRow draws one numeric property row — label, input field, unit suffix —
// returning true on the frame the value changed.
func propertyFloatRow(label, id, suffix string, v *float32) bool {
	propertyRow(label)
	native.SetNextItemWidth(propertyFieldWidth)
	changed := native.InputFloat("##"+id, v)
	if suffix != "" {
		native.SameLine()
		native.Text(suffix)
	}
	return changed
}

// propertySelectorChip draws a selection chip showing a pick's state: filled renders in
// the accent color with a clear (×) button beside it; empty-but-required renders the
// prompt in the danger color (the reference panel's red "Select …" state); an optional
// empty chip renders plain. Returns true when the clear button was clicked.
func propertySelectorChip(id, text string, filled, required bool) bool {
	if !filled {
		drawEmptySelectorChip(id, text, required)
		return false
	}
	native.PushStyleColor("Button", accentColor)
	native.PushStyleColor("ButtonHovered", accentColor)
	native.PushStyleColor("ButtonActive", accentColor)
	native.Button(text + "##" + id)
	native.PopStyleColor(3)
	native.SameLine()
	return native.Button("×##" + id + "-clear") // × — clear the selection
}

// drawEmptySelectorChip renders the empty chip: a prompt-colored button that only
// signals state (picking happens in the viewport, so clicking it has no action).
func drawEmptySelectorChip(id, text string, required bool) {
	if required {
		native.PushStyleColor("Text", dangerColor)
		defer native.PopStyleColor(1)
	}
	native.Button(text + "##" + id)
}

// propertyIconToggles draws a row of icon toggle buttons (the reference panel's
// Direction and Boolean rows), highlighting the active index in the accent color.
// Each option is an icon key + tooltip; a missing glyph falls back to its tooltip's
// text so the option never disappears. Returns the index clicked this frame, or -1.
func propertyIconToggles(id string, keys, tips []string, active int) int {
	clicked := -1
	for i, key := range keys {
		if i > 0 {
			native.SameLine()
		}
		if drawPropertyToggle(id, key, tips[i], i == active) {
			clicked = i
		}
	}
	return clicked
}

// drawPropertyToggle draws one toggle of the group: an accent-highlighted icon button
// when selected, with the option tooltip on hover.
func drawPropertyToggle(group, key, tip string, selected bool) bool {
	if selected {
		native.PushStyleColor("Button", accentColor)
		native.PushStyleColor("ButtonHovered", accentColor)
		native.PushStyleColor("ButtonActive", accentColor)
		defer native.PopStyleColor(3)
	}
	clicked := drawPropertyToggleControl(group, key, tip)
	native.SetItemTooltip(tip)
	return clicked
}

// drawPropertyToggleControl draws the toggle's clickable control: the icon at the small
// glyph size, or a text button when the asset is missing/unuploadable.
func drawPropertyToggleControl(group, key, tip string) bool {
	if tex, ok := icons.texture(key, smallIconPx); ok {
		px := float32(smallIconPx)
		return native.ImageButton(group+"-"+key, tex, px, px, identityTint)
	}
	return native.Button(tip + "##" + group + "-" + key)
}
