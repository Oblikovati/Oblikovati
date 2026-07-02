//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
)

// The feature property panels (the Extrusion panel first) share this widget kit: a
// breadcrumb header, collapsible sections, label/control rows on a fixed column grid,
// selection chips that show the pick state (filled / required-but-empty), and rows of
// icon toggle buttons. The layout mirrors the reference panel: section headers span the
// panel, labels sit in a left column, controls align in a right column.

// propertyLabelWidth is the left label column width; propertyFieldWidth sizes numeric
// inputs in the control column so unit suffixes fit beside them; propertyComboWidth
// sizes dropdowns whose entries run longer than a number (thread designations).
const (
	propertyLabelWidth = 95
	propertyFieldWidth = 110
	propertyComboWidth = 170
)

// drawFeatureBreadcrumb renders a property panel's header trail: the feature name and,
// once known, the sketch it consumes (the reference panel's "Feature > Sketch" line).
func drawFeatureBreadcrumb(featureName, sketchName string) {
	native.Text(featureName)
	if sketchName != "" {
		native.SameLine()
		native.Text("> " + sketchName)
	}
	native.Separator()
}

// pickChipText is the chip caption for a single-pick selector: the filled text once
// picked, the required prompt until then.
func pickChipText(picked bool, filled, prompt string) string {
	if picked {
		return filled
	}
	return prompt
}

// countChipText is the chip caption for a multi-pick selector: the pick count with its
// noun ("3 Edges"), or the prompt while empty.
func countChipText(n int, noun, prompt string) string {
	switch n {
	case 0:
		return prompt
	case 1:
		return "1 " + noun
	default:
		return strconv.Itoa(n) + " " + noun + "s"
	}
}

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

// A numeric parameter row lives in parameter_input.go (parameterFloatRow): the document unit is
// rendered INSIDE the field, never as a side label (#1519). There is deliberately no bare
// "InputFloat + unit label" row here — guard_parameter_input_test enforces that.

// propertyComboRow draws one one-of-N property row — label + dropdown — returning the
// newly chosen index, or -1 when the selection did not change this frame.
func propertyComboRow(label, id string, options []string, selected int) int {
	propertyRow(label)
	native.SetNextItemWidth(propertyComboWidth)
	chosen := -1
	if len(options) == 0 || !native.BeginCombo("##"+id, options[clampIdx(selected, len(options))]) {
		return chosen
	}
	for i, opt := range options {
		if native.Selectable(opt, i == selected) {
			chosen = i
		}
	}
	native.EndCombo()
	return chosen
}

// drawPickChipRow draws one Input Geometry pick row — label, selection chip, hover tip —
// invoking onClear when the chip's clear (×) is clicked. The pick itself always happens
// in the viewport; the chip only shows the state.
func drawPickChipRow(label, id, text string, filled bool, tip string, onClear func()) {
	propertyRow(label)
	if propertySelectorChip(id, text, filled, true) {
		onClear()
	}
	native.SetItemTooltip(tip)
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
	native.PushStyleColor("Button", chromeTheme.accentColor)
	native.PushStyleColor("ButtonHovered", chromeTheme.accentColor)
	native.PushStyleColor("ButtonActive", chromeTheme.accentColor)
	native.Button(text + "##" + id)
	native.PopStyleColor(3)
	native.SameLine()
	return native.Button("×##" + id + "-clear") // × — clear the selection
}

// drawEmptySelectorChip renders the empty chip: a prompt-colored button that only
// signals state (picking happens in the viewport, so clicking it has no action).
func drawEmptySelectorChip(id, text string, required bool) {
	if required {
		native.PushStyleColor("Text", chromeTheme.dangerColor)
		defer native.PopStyleColor(1)
	}
	native.Button(text + "##" + id)
}

// propertyArmableSlotChip draws the generic editors' reference-slot chip: clicking the
// chip arms the slot for viewport picking (the reference panel's Active selector state —
// it reads "Selecting…" while armed), and the clear (×) empties it when clearable.
// Returns (armClicked, clearClicked).
func propertyArmableSlotChip(id, text string, filled, armed, clearable bool) (bool, bool) {
	return armableSlotChip(id, text, filled, armed, clearable, true)
}

// propertyOptionalArmableSlotChip is propertyArmableSlotChip for an OPTIONAL slot (e.g. a sweep's
// guide rail): identical arm/clear behaviour, but the empty prompt renders in the normal color
// rather than the required-red, so it doesn't read as a missing required input.
func propertyOptionalArmableSlotChip(id, text string, filled, armed, clearable bool) (bool, bool) {
	return armableSlotChip(id, text, filled, armed, clearable, false)
}

// armableSlotChip is the shared arm/clear chip: clicking the chip arms viewport picking (it reads
// "Selecting…" while armed), and the × empties it when clearable. required tints the empty prompt.
func armableSlotChip(id, text string, filled, armed, clearable, required bool) (bool, bool) {
	if armed {
		text = "Selecting…"
	}
	armClicked := drawArmableChipButton(id, text, filled, armed, required)
	if !clearable {
		return armClicked, false
	}
	native.SameLine()
	return armClicked, native.Button("×##" + id + "-clear")
}

// drawArmableChipButton renders the chip body: accent while armed or filled, otherwise the prompt
// — in the danger color for a required slot, the normal color for an optional one.
func drawArmableChipButton(id, text string, filled, armed, required bool) bool {
	if armed || filled {
		native.PushStyleColor("Button", chromeTheme.accentColor)
		native.PushStyleColor("ButtonHovered", chromeTheme.accentColor)
		native.PushStyleColor("ButtonActive", chromeTheme.accentColor)
		defer native.PopStyleColor(3)
	} else if required {
		native.PushStyleColor("Text", chromeTheme.dangerColor)
		defer native.PopStyleColor(1)
	}
	return native.Button(text + "##" + id)
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
		native.PushStyleColor("Button", chromeTheme.accentColor)
		native.PushStyleColor("ButtonHovered", chromeTheme.accentColor)
		native.PushStyleColor("ButtonActive", chromeTheme.accentColor)
		defer native.PopStyleColor(3)
	}
	clicked := drawPropertyToggleControl(group, key, tip)
	native.SetItemTooltip(tip)
	return clicked
}

// propertyToggleSet bundles an icon-toggle row's glyph keys with their tooltips.
type propertyToggleSet struct{ keys, tips []string }

// booleanToggles / booleanToggleOps are the Output section's Boolean row, shared by the
// volumetric feature panels (Extrude, Revolve, Sweep) in the reference panel's order.
var booleanToggles = propertyToggleSet{
	keys: []string{"bool-join", "bool-cut", "bool-intersect", "bool-new-solid"},
	tips: []string{
		"Join — merge the feature into the body",
		"Cut — remove the feature from the body",
		"Intersect — keep only the overlapping material",
		"New Solid — create a separate body",
	},
}

var booleanToggleOps = []ops.PartFeatureOperation{ops.Join, ops.Cut, ops.Intersect, ops.NewBody}

// booleanToggleIndex maps a feature operation onto the Boolean toggle row.
func booleanToggleIndex(op ops.PartFeatureOperation) int {
	for i, o := range booleanToggleOps {
		if o == op {
			return i
		}
	}
	return 3 // New Solid, the tools' default
}

// drawBooleanPropertyRow renders the Output section's Boolean toggle row, writing a
// clicked operation through set.
func drawBooleanPropertyRow(id string, current ops.PartFeatureOperation, set func(ops.PartFeatureOperation)) {
	propertyRow("Boolean")
	if i := propertyIconToggles(id, booleanToggles.keys, booleanToggles.tips, booleanToggleIndex(current)); i >= 0 {
		set(booleanToggleOps[i])
	}
}

// drawPropertyToggleControl draws the toggle's clickable control: the icon at the small
// glyph size, or a text button when the asset is missing/unuploadable.
func drawPropertyToggleControl(group, key, tip string) bool {
	ipx := scaledIconPx(smallIconPx)
	if tex, ok := icons.texture(key, "", ipx); ok {
		px := float32(ipx)
		return native.ImageButton(group+"-"+key, tex, px, px, identityTint)
	}
	return native.Button(tip + "##" + group + "-" + key)
}
