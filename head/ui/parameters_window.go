//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/param"
)

// The Parameters dialog (Manage ▸ Parameters): a search box, the Model and User
// parameter tables, an Add row, and Done. The session owns the parameters and the edit
// verbs; this file is pure layout, reading presentation rows each frame and calling the
// verbs on change. Per-cell text buffers are seeded once and dropped after a commit so
// the model's reformatted value shows next frame.

const paramBufLen = 128

// parametersUI is the dialog's cross-frame widget state.
var parametersUI = struct {
	search   [paramBufLen]byte
	cells    map[string][]byte // per-cell InputText buffers, keyed "field:id"
	addKind  int               // 0 numeric, 1 text, 2 boolean
	addName  [paramBufLen]byte
	addValue [paramBufLen]byte
	addBool  bool

	listFor    param.ID // multi-value list editor target (0 ⇒ closed)
	listText   [512]byte
	listCustom bool
	tolFor     param.ID // tolerance editor target (0 ⇒ closed)
	tolUpper   float32
	tolLower   float32
	tolType    int
	groupFor   param.ID // add-to-group target (0 ⇒ closed)
	groupName  [paramBufLen]byte
}{}

var addKindNames = []string{"Numeric", "Text", "True/False"}

// drawParametersWindow renders the Parameters dialog while it is open.
func drawParametersBody(s *app.Session) {
	if parametersUI.cells == nil {
		parametersUI.cells = map[string][]byte{}
	}
	native.SetNextItemWidth(-1)
	native.InputText("##param-search", parametersUI.search[:])
	model, user := s.ParameterRows(bufString(parametersUI.search[:]))
	drawParameterSection(s, "Model Parameters", "##model-params", model)
	drawParameterSection(s, "User Parameters", "##user-params", user)
	drawAddParameterRow(s)
	drawDerivedTablesSection(s) // linked parameters from other documents (M39-F04, #1560)
	native.Separator()
	if native.Button("Done") {
		s.CloseParameters()
	}
}

// drawParametersEditors renders the Parameters panel's trailing child windows — the value-list,
// tolerance and add-to-group editors are separate ImGui windows targeted from the table rows, so
// they cannot live inside the panel body. The dockable-panel registry runs this via its `after`
// hook, after the body's End (#1473).
func drawParametersEditors(s *app.Session) {
	drawValueListEditor(s)
	drawToleranceEditor(s)
	drawAddToGroupDialog(s)
	drawLinkParametersDialog(s) // the Link… source-document picker (M39-F04, #1560)
}

// drawParameterSection draws one labeled table (Model or User), or a placeholder line when
// it is empty.
func drawParameterSection(s *app.Session, title, id string, rows []app.ParameterRow) {
	native.SeparatorText(title)
	if len(rows) == 0 {
		native.Text("  (none)")
		return
	}
	if !native.BeginTable(id, 8, 0, paramTableHeight(len(rows))) {
		return
	}
	for _, c := range []string{"Name", "Unit", "Equation", "Value", "Tol.", "Key", "Export", "Comment"} {
		native.TableSetupColumn(c)
	}
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
	for _, row := range rows {
		drawParameterRow(s, row)
	}
	native.EndTable()
}

// paramTableHeight caps a table at eight visible rows so both tables and the Add row fit.
func paramTableHeight(rows int) float32 {
	const rowPx, headerPx, maxRows = 24, 26, 8
	if rows > maxRows {
		rows = maxRows
	}
	return float32(headerPx + rows*rowPx)
}

// drawAddParameterRow draws the bottom Add control: a kind chooser, a name and value
// field, and the Add button — Inventor's Add Numeric / Text / True-False drop-down.
func drawAddParameterRow(s *app.Session) {
	native.Separator()
	native.Text("Add")
	native.SameLine()
	native.SetNextItemWidth(120)
	if native.BeginCombo("##add-kind", addKindNames[parametersUI.addKind]) {
		for i, name := range addKindNames {
			if native.Selectable(name, i == parametersUI.addKind) {
				parametersUI.addKind = i
			}
		}
		native.EndCombo()
	}
	native.SameLine()
	native.SetNextItemWidth(140)
	native.InputText("##add-name", parametersUI.addName[:])
	native.SameLine()
	drawAddValueField()
	native.SameLine()
	if native.Button("Add") {
		commitAddParameter(s)
	}
}

// drawAddValueField draws the value editor for the chosen add-kind (a checkbox for
// True/False, a text field otherwise).
func drawAddValueField() {
	if parametersUI.addKind == 2 {
		native.Checkbox("value##add-bool", &parametersUI.addBool)
		return
	}
	native.SetNextItemWidth(140)
	native.InputText("##add-value", parametersUI.addValue[:])
}

// commitAddParameter adds a parameter of the chosen kind and, on success, clears the Add
// fields so the row is ready for the next entry.
func commitAddParameter(s *app.Session) {
	name := bufString(parametersUI.addName[:])
	value := bufString(parametersUI.addValue[:])
	var err error
	switch parametersUI.addKind {
	case 1:
		err = s.AddTextUserParameter(name, value)
	case 2:
		err = s.AddBooleanUserParameter(name, parametersUI.addBool)
	default:
		err = s.AddNumericUserParameter(name, value)
	}
	if err == nil {
		clearBuf(parametersUI.addName[:])
		clearBuf(parametersUI.addValue[:])
	}
}
