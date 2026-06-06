//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/model/param"
)

// One parameter row and the popups it can open. Editable cells commit on focus loss
// (IsItemDeactivatedAfterEdit) so a name is not renamed on every keystroke; per-cell
// buffers are dropped after a commit so the model's reformatted value re-seeds them.

// drawParameterRow draws one table row: the eight cells plus the right-click menu.
func drawParameterRow(s *app.Session, row app.ParameterRow) {
	native.PushIDInt(int(row.ID))
	defer native.PopID()
	native.TableNextRow()

	native.TableNextColumn()
	drawNameCell(s, row)
	drawRowContextMenu(s, row)
	native.TableNextColumn()
	native.Text(row.UnitName)
	native.TableNextColumn()
	drawEquationCell(s, row)
	native.TableNextColumn()
	drawValueCell(row)
	native.TableNextColumn()
	native.Text(row.Tolerance)
	native.TableNextColumn()
	drawFlagCell(s, row, "##key", row.IsKey, s.SetParameterKey)
	native.TableNextColumn()
	drawFlagCell(s, row, "##export", row.Export, s.SetParameterExport)
	native.TableNextColumn()
	drawCommentCell(s, row)
}

// drawNameCell edits (or shows) the parameter name.
func drawNameCell(s *app.Session, row app.ParameterRow) {
	if !row.Editable {
		native.Text(row.Name)
		return
	}
	editCell(s, "name", row.ID, row.Name, func(v string) error { return s.SetParameterName(row.ID, v) })
}

// drawEquationCell edits the equation: a dropdown for multi-value, a checkbox for boolean,
// a text field for everything else editable, or plain text for read-only parameters.
func drawEquationCell(s *app.Session, row app.ParameterRow) {
	switch {
	case row.MultiValue:
		drawEquationCombo(s, row)
	case row.ValueType == "boolean":
		v := row.Equation == "true"
		native.SetNextItemWidth(-1)
		if native.Checkbox("##eq-bool", &v) {
			_ = s.SetParameterBool(row.ID, v)
		}
	case row.Editable:
		editCell(s, "eq", row.ID, row.Equation, func(v string) error { return s.SetParameterEquation(row.ID, v) })
	default:
		native.Text(row.Equation)
	}
}

// drawEquationCombo offers the multi-value choices as a dropdown.
func drawEquationCombo(s *app.Session, row app.ParameterRow) {
	native.SetNextItemWidth(-1)
	if !native.BeginCombo("##eq-list", row.Equation) {
		return
	}
	for _, opt := range row.Options {
		if native.Selectable(opt, opt == row.Equation) {
			_ = s.SetParameterEquation(row.ID, opt)
		}
	}
	native.EndCombo()
}

// drawValueCell shows the evaluated value, flagging an unhealthy parameter.
func drawValueCell(row app.ParameterRow) {
	if !row.Healthy {
		native.Text("⚠ " + row.Value)
		if native.IsItemHovered() && row.Health != "" {
			native.SetItemTooltip(row.Health)
		}
		return
	}
	native.Text(row.Value)
}

// drawCommentCell edits (or shows) the comment.
func drawCommentCell(s *app.Session, row app.ParameterRow) {
	if !row.Editable {
		native.Text(row.Comment)
		return
	}
	editCell(s, "comment", row.ID, row.Comment, func(v string) error { return s.SetParameterComment(row.ID, v) })
}

// drawFlagCell draws a Key/Export checkbox (disabled for read-only parameters).
func drawFlagCell(s *app.Session, row app.ParameterRow, id string, value bool, set func(param.ID, bool) error) {
	v := value
	native.BeginDisabled(!row.Editable)
	if native.Checkbox(id, &v) {
		_ = set(row.ID, v)
	}
	native.EndDisabled()
}

// editCell draws a text field seeded once from value; on commit (focus loss) it calls set
// and drops the buffer so the reformatted model value re-seeds it next frame.
func editCell(s *app.Session, field string, id param.ID, value string, set func(string) error) {
	key := field + ":" + idKey(id)
	buf := cellBuf(key, value)
	native.SetNextItemWidth(-1)
	native.InputText("##"+field, buf)
	if native.IsItemDeactivatedAfterEdit() {
		_ = set(bufString(buf))
		delete(parametersUI.cells, key)
	}
}

// cellBuf returns the persistent edit buffer for a cell, seeding it from seed on creation.
func cellBuf(key, seed string) []byte {
	if b, ok := parametersUI.cells[key]; ok {
		return b
	}
	b := make([]byte, paramBufLen)
	copy(b, seed)
	parametersUI.cells[key] = b
	return b
}

// drawRowContextMenu is the right-click menu for the row's name cell.
func drawRowContextMenu(s *app.Session, row app.ParameterRow) {
	if !native.BeginPopupContextItem("##row-menu") {
		return
	}
	if native.MenuItem("Copy to User Parameter") {
		_ = s.CopyParameterToUser(row.ID)
	}
	if row.Editable && native.MenuItem("Delete Parameter") {
		_ = s.DeleteParameter(row.ID)
	}
	if native.MenuItem("Add to Group…") {
		openGroupDialog(row)
	}
	if row.Group != "" && native.MenuItem("Remove from Group") {
		_ = s.RemoveParameterFromGroup(row.ID)
	}
	if row.Editable && row.ValueType != "boolean" {
		drawMultiValueMenuItems(s, row)
	}
	if row.Editable && row.ValueType == "numeric" && native.MenuItem("Edit Tolerance…") {
		openToleranceDialog(row)
	}
	native.EndPopup()
}

// drawMultiValueMenuItems adds the make/edit/clear multi-value entries.
func drawMultiValueMenuItems(s *app.Session, row app.ParameterRow) {
	if row.MultiValue {
		if native.MenuItem("Edit Multi-Value List…") {
			openValueListDialog(row)
		}
		if native.MenuItem("Make Single Value") {
			_ = s.ClearParameterValueList(row.ID)
		}
		return
	}
	if native.MenuItem("Make Multi-Value…") {
		openValueListDialog(row)
	}
}

// idKey renders a parameter id as a stable buffer-key string.
func idKey(id param.ID) string { return strconv.FormatUint(uint64(id), 10) }
