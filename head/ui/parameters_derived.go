//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Parameters dialog's Linked-parameters section (M39-F04, #1560): the list of
// derived-parameter tables this document links from others, each with a Delete, plus a
// "Link…" picker that derives parameters from another open document — a part or an assembly.
// All edits go through the session derived-table verbs (AddDerivedParameterTable, …); this
// file is pure layout, reading presentation rows each frame.

// derivedUI is the link picker's cross-frame state. source is the chosen source document's
// full name ("" until one is picked); checked maps a source-parameter name to whether it is
// selected for linking.
var derivedUI = struct {
	pickerOpen bool
	source     string
	checked    map[string]bool
}{}

// drawDerivedTablesSection lists the active document's linked parameter tables and offers the
// Link… picker. Rendered for any parameter holder (part or assembly), below the User table.
func drawDerivedTablesSection(s *app.Session) {
	native.SeparatorText("Linked Parameters")
	rows := s.DerivedTableRows()
	if len(rows) == 0 {
		native.Text("  (none linked)")
	} else {
		drawDerivedTableList(s, rows)
	}
	if native.Button("Link…") {
		openLinkPicker()
	}
}

// drawDerivedTableList renders one row per linked table: its source, the linked names, the
// health, and a Delete button.
func drawDerivedTableList(s *app.Session, rows []app.DerivedTableRow) {
	if !native.BeginTable("##derived-tables", 4, 0, paramTableHeight(len(rows))) {
		return
	}
	for _, c := range []string{"Source", "Linked", "Health", ""} {
		native.TableSetupColumn(c)
	}
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
	for _, row := range rows {
		drawDerivedTableRow(s, row)
	}
	native.EndTable()
}

// drawDerivedTableRow draws one linked-table row; Delete removes the table and its derived
// parameters.
func drawDerivedTableRow(s *app.Session, row app.DerivedTableRow) {
	native.TableNextRow()
	native.TableNextColumn()
	native.Text(row.SourceDisplay)
	native.TableNextColumn()
	native.Text(strings.Join(row.Linked, ", "))
	native.TableNextColumn()
	native.Text(derivedHealthLabel(row.Health))
	native.TableNextColumn()
	if native.Button("Delete##dt-" + strconv.Itoa(row.ID)) {
		_ = s.DeleteDerivedParameterTable(row.ID)
	}
}

// derivedHealthLabel shows "OK" for a healthy table and the reason otherwise.
func derivedHealthLabel(health string) string {
	if health == "" {
		return "OK"
	}
	return health
}

// openLinkPicker resets and opens the Link Parameters picker.
func openLinkPicker() {
	derivedUI.pickerOpen = true
	derivedUI.source = ""
	derivedUI.checked = map[string]bool{}
}

// drawLinkParametersDialog renders the Link Parameters picker while it is open. The
// dockable-panel registry runs this via the panel's `after` hook (a separate ImGui window).
func drawLinkParametersDialog(s *app.Session) {
	if !derivedUI.pickerOpen {
		return
	}
	candidates := s.LinkableSourceDocuments()
	dialogSize(360, 320)
	if native.Begin("Link Parameters") {
		if len(candidates) == 0 {
			native.Text("No other open document offers parameters to link.")
		} else {
			drawSourcePicker(candidates)
			drawSourceParameterChecklist(candidates)
		}
		drawLinkPickerButtons(s)
	}
	native.End()
}

// drawSourcePicker renders the source-document combo; changing the selection resets the
// per-parameter checkboxes.
func drawSourcePicker(candidates []app.SourceDocumentRow) {
	native.Text("Derive parameters from:")
	if !native.BeginCombo("##link-source", sourcePreview(candidates)) {
		return
	}
	for _, c := range candidates {
		if native.Selectable(c.Display, c.FullName == derivedUI.source) && c.FullName != derivedUI.source {
			derivedUI.source = c.FullName
			derivedUI.checked = map[string]bool{}
		}
	}
	native.EndCombo()
}

// sourcePreview is the combo's current label: the chosen source's display name, or a prompt.
func sourcePreview(candidates []app.SourceDocumentRow) string {
	for _, c := range candidates {
		if c.FullName == derivedUI.source {
			return c.Display
		}
	}
	return "(choose a document)"
}

// drawSourceParameterChecklist lists the chosen source's parameters as checkboxes.
func drawSourceParameterChecklist(candidates []app.SourceDocumentRow) {
	row, ok := sourceRow(candidates, derivedUI.source)
	if !ok {
		return
	}
	native.SeparatorText("Parameters")
	for _, name := range row.Parameters {
		checked := derivedUI.checked[name]
		if native.Checkbox(name+"##link-"+name, &checked) {
			derivedUI.checked[name] = checked
		}
	}
}

// sourceRow finds the candidate row for a full document name.
func sourceRow(candidates []app.SourceDocumentRow, fullName string) (app.SourceDocumentRow, bool) {
	for _, c := range candidates {
		if c.FullName == fullName {
			return c, true
		}
	}
	return app.SourceDocumentRow{}, false
}

// drawLinkPickerButtons renders Link (commits the derived table) and Cancel.
func drawLinkPickerButtons(s *app.Session) {
	native.Separator()
	if native.Button("Link") {
		commitLink(s)
	}
	native.SameLine()
	if native.Button("Cancel") {
		derivedUI.pickerOpen = false
	}
}

// commitLink derives the checked parameters from the chosen source, closing the picker on
// success (a failure — no source, self-link — leaves it open so the user can correct it).
func commitLink(s *app.Session) {
	if derivedUI.source == "" {
		return
	}
	var linked []string
	for name, on := range derivedUI.checked {
		if on {
			linked = append(linked, name)
		}
	}
	if _, err := s.AddDerivedParameterTable(derivedUI.source, linked); err == nil {
		derivedUI.pickerOpen = false
	}
}
