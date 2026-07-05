//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/param"
)

// The Parameters dialog's three small popups: the Value List Editor (multi-value), the
// Tolerance editor, and Add to Group. Each is a modeless window gated on a target id in
// parametersUI, opened from the row context menu and committed back through a session verb.

// toleranceTypes pairs the model-value selector with its label, in enum order.
var toleranceTypes = []struct {
	kind  param.ModelValueType
	label string
}{
	{param.Nominal, "Nominal"}, {param.Upper, "Upper"}, {param.Lower, "Lower"}, {param.Median, "Median"},
}

// openValueListDialog seeds and opens the Value List Editor for a parameter, pre-filling
// its current list (one entry per line).
func openValueListDialog(row app.ParameterRow) {
	parametersUI.listFor = row.ID
	clearBuf(parametersUI.listText[:])
	copy(parametersUI.listText[:], strings.Join(row.Options, "\n"))
}

// drawValueListEditor renders the Value List Editor while a target is set.
func drawValueListEditor(s *app.Session) {
	if parametersUI.listFor == 0 {
		return
	}
	dialogSize(320, 220)
	if native.Begin("Value List Editor") {
		native.Text("One value per line (or comma-separated):")
		native.InputText("##value-list", parametersUI.listText[:])
		native.Checkbox("Allow custom values", &parametersUI.listCustom)
		if native.Button("OK") {
			applyValueList(s)
		}
		native.SameLine()
		if native.Button("Cancel") {
			parametersUI.listFor = 0
		}
	}
	native.End()
}

// applyValueList parses the edited text into a list and sets it on the parameter.
func applyValueList(s *app.Session) {
	list := splitValues(bufString(parametersUI.listText[:]))
	if err := s.SetParameterValueList(parametersUI.listFor, list, parametersUI.listCustom); err == nil {
		parametersUI.listFor = 0
	}
}

// splitValues splits the editor text on newlines and commas, trimming blanks.
func splitValues(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == ',' })
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if v := strings.TrimSpace(f); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// openToleranceDialog seeds and opens the Tolerance editor.
func openToleranceDialog(row app.ParameterRow) {
	parametersUI.tolFor = row.ID
	parametersUI.tolUpper, parametersUI.tolLower, parametersUI.tolType = 0, 0, 0
}

// drawToleranceEditor renders the Tolerance editor while a target is set. The deviations
// are entered in the document's database units (the same convention as the model band).
func drawToleranceEditor(s *app.Session) {
	if parametersUI.tolFor == 0 {
		return
	}
	dialogSize(300, 200)
	if native.Begin("Tolerance") {
		native.InputFloat("Upper", &parametersUI.tolUpper)
		native.InputFloat("Lower", &parametersUI.tolLower)
		drawToleranceTypeCombo()
		if native.Button("OK") {
			applyTolerance(s)
		}
		native.SameLine()
		if native.Button("Cancel") {
			parametersUI.tolFor = 0
		}
	}
	native.End()
}

// drawToleranceTypeCombo chooses which value of the band the model consumes.
func drawToleranceTypeCombo() {
	if !native.BeginCombo("Model value", toleranceTypes[parametersUI.tolType].label) {
		return
	}
	for i, t := range toleranceTypes {
		if native.Selectable(t.label, i == parametersUI.tolType) {
			parametersUI.tolType = i
		}
	}
	native.EndCombo()
}

// applyTolerance commits the edited band to the parameter.
func applyTolerance(s *app.Session) {
	kind := toleranceTypes[parametersUI.tolType].kind
	err := s.SetParameterTolerance(parametersUI.tolFor, float64(parametersUI.tolUpper), float64(parametersUI.tolLower), kind)
	if err == nil {
		parametersUI.tolFor = 0
	}
}

// openGroupDialog seeds and opens the Add to Group dialog.
func openGroupDialog(row app.ParameterRow) {
	parametersUI.groupFor = row.ID
	clearBuf(parametersUI.groupName[:])
}

// drawAddToGroupDialog renders the Add to Group dialog while a target is set.
func drawAddToGroupDialog(s *app.Session) {
	if parametersUI.groupFor == 0 {
		return
	}
	dialogSize(300, 140)
	if native.Begin("Add to Group") {
		native.Text("Group name:")
		native.InputText("##group-name", parametersUI.groupName[:])
		if native.Button("OK") {
			applyGroup(s)
		}
		native.SameLine()
		if native.Button("Cancel") {
			parametersUI.groupFor = 0
		}
	}
	native.End()
}

// applyGroup assigns the parameter to the named group.
func applyGroup(s *app.Session) {
	name := strings.TrimSpace(bufString(parametersUI.groupName[:]))
	if name == "" {
		return
	}
	if err := s.AddParameterToGroup(parametersUI.groupFor, name); err == nil {
		parametersUI.groupFor = 0
	}
}
