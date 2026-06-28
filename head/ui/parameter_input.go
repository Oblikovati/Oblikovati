//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/head/internal/native"
)

// ParameterInput is the property panels' canonical numeric parameter field (#1519). A tool that
// takes a dimensioned value MUST use it rather than a bare InputFloat with the unit painted beside
// the field: the document unit is rendered INSIDE the input (e.g. "10.000 mm"), so a value and its
// dimension read and edit as one token. A plain text input is reserved for labels and descriptions.
//
// The buffer convention is unchanged from the rows it replaces — the caller owns the displayed value
// (already converted to the document unit) and writes it back when the row returns true.

// parameterFloatRow draws a label, then the value with its document unit shown in-field to `precision`
// decimals. An optional hint (a NON-unit annotation — a sign convention, a ratio range like "(0–1)")
// trails as a dimmed label, since it is genuinely descriptive text, not a dimension. Returns true on
// the frame the value changed.
func parameterFloatRow(label, id, unit string, precision int, hint string, v *float32) bool {
	propertyRow(label)
	native.SetNextItemWidth(propertyFieldWidth)
	changed := native.InputFloatFormat("##"+id, v, parameterDisplayFormat(unit, precision))
	if hint != "" {
		native.SameLine()
		native.Text(hint)
	}
	return changed
}

// parameterDisplayFormat builds the ImGui display/parse format for a parameter value: the number to
// `precision` decimals with the unit appended in-field ("%.3f mm"); a unitless parameter (revolutions,
// a ratio) is just the number ("%.2f"). ImGui trims the trailing unit when parsing, so the user may
// type a bare number or include the unit. precision is clamped to [0,9] (3 if out of range).
func parameterDisplayFormat(unit string, precision int) string {
	if precision < 0 || precision > 9 {
		precision = 3
	}
	p := strconv.Itoa(precision)
	if unit == "" {
		return "%." + p + "f"
	}
	return "%." + p + "f " + unit
}
