//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// ParameterInput is the property panels' canonical numeric parameter field (#1519). A tool that
// takes a dimensioned value MUST use it rather than a bare InputFloat with the unit painted beside
// the field. It does two things at once:
//
//   - the document unit is rendered INSIDE the field ("10.000 mm"), so a value and its dimension read
//     and edit as one token; and
//   - the field accepts a number, a unit-bearing literal ("10.5 mm"), OR a formula referencing the
//     part's parameters ("D0 * 10.5 mm"), evaluated live by the parameter parser (it is a text field,
//     not a numeric spinner, so an expression is a first-class input).
//
// The value is held by the caller in the document's preferred unit (*v) — the same convention as the
// rows it replaces. While the field is focused the user's text is preserved; once it loses focus it
// reformats to the canonical value, so a committed formula shows its evaluated result.

// paramFieldKind selects how a field formats and evaluates: a length, an angle, or a unitless value.
type paramFieldKind int

const (
	paramUnitless paramFieldKind = iota
	paramLength
	paramAngle
)

// paramFieldBufLen bounds a parameter-expression buffer ("D0 * 10.5 mm" and the like).
const paramFieldBufLen = 96

// paramFieldBufs holds each field's live expression text across frames, keyed by the field id, so an
// in-progress formula survives between frames while it is being typed.
var paramFieldBufs = map[string][]byte{}

// paramFieldBuffer returns the persistent text buffer for field id, and whether it was just created
// (so the caller can seed it with the current value before the first render).
func paramFieldBuffer(id string) (buf []byte, fresh bool) {
	if b, ok := paramFieldBufs[id]; ok {
		return b, false
	}
	b := make([]byte, paramFieldBufLen)
	paramFieldBufs[id] = b
	return b, true
}

// parameterField is the ParameterInput primitive: the expression input alone (no label), so callers
// can compose it (e.g. a greyed field beside a toggle row). It shows *v with its unit, evaluates the
// typed text through the parameter parser on every edit, and writes the resolved value back to *v in
// the document's preferred unit. Returns true on the frame *v changed. Width is set by the caller.
func parameterField(s paramFieldEvaluator, id, unit string, precision int, kind paramFieldKind, v *float32) bool {
	buf, fresh := paramFieldBuffer(id)
	if fresh {
		setBuf(buf, formatParamValue(float64(*v), unit, precision))
	}
	changed := false
	if native.InputText("##"+id, buf) {
		if val, ok := evalParamField(s, bufString(buf), kind); ok && float32(val) != *v {
			*v = float32(val)
			changed = true
		}
	}
	if !native.IsItemActive() { // not being edited → show the canonical value with its unit
		setBuf(buf, formatParamValue(float64(*v), unit, precision))
	}
	return changed
}

// parameterFloatRow draws a labelled ParameterInput row: the label, the expression field (unit and
// precision derived from kind + the document), and an optional NON-unit hint (a sign convention or a
// ratio range like "(0–1)") as a trailing label. Returns true on the frame the value changed.
func parameterFloatRow(s *app.Session, label, id string, kind paramFieldKind, hint string, v *float32) bool {
	unit, precision := paramFieldUnit(s, kind)
	propertyRow(label)
	native.SetNextItemWidth(propertyFieldWidth)
	changed := parameterField(s, id, unit, precision, kind, v)
	if hint != "" {
		native.SameLine()
		native.Text(hint)
	}
	return changed
}

// paramFieldUnit returns the document unit name and display precision for a field kind.
func paramFieldUnit(s *app.Session, kind paramFieldKind) (string, int) {
	switch kind {
	case paramLength:
		return s.LengthUnitName(), s.LengthPrecision()
	case paramAngle:
		return s.AngleUnitName(), s.AnglePrecision()
	default:
		return "", -1
	}
}

// paramFieldEvaluator is the slice of the session a parameter field needs to turn typed text into a
// value: the active part's parameter parser, per field kind. Taking this instead of the whole
// *app.Session is the audit-I5 consumer-interface pattern — a field evaluates an expression and
// does nothing else, and the signature now says so.
type paramFieldEvaluator interface {
	EvalLengthDisplay(text string) (float64, bool)
	EvalAngleDisplay(text string) (float64, bool)
	EvalUnitless(text string) (float64, bool)
}

var _ paramFieldEvaluator = (*app.Session)(nil)

// evalParamField evaluates a field's text to a value in the document's preferred unit for its kind,
// using the active part's parameter parser (so formulas referencing parameters resolve). ok is false
// while the text does not yet resolve, so the caller keeps the last good value.
func evalParamField(s paramFieldEvaluator, text string, kind paramFieldKind) (float64, bool) {
	switch kind {
	case paramLength:
		return s.EvalLengthDisplay(text)
	case paramAngle:
		return s.EvalAngleDisplay(text)
	default:
		return s.EvalUnitless(text)
	}
}

// formatParamValue renders a value to `precision` decimals with its unit appended ("10.000 mm"), or
// just the number for a unitless field. precision is clamped to [0,9] (3 if out of range). This is
// the canonical text the field shows when it is not being edited.
func formatParamValue(value float64, unit string, precision int) string {
	if precision < 0 || precision > 9 {
		precision = 3
	}
	s := strconv.FormatFloat(value, 'f', precision, 64)
	if unit != "" {
		s += " " + unit
	}
	return s
}
