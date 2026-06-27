//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/bom"
)

// The Bill of Materials panel (Assemble ▸ Bill of Materials, #768): a window listing the active
// assembly's components in the chosen view (Structured / Parts Only) with an Export-to-CSV button.
// The BOM itself — counting, grouping, the CSV — is model/bom, surfaced through Session.AssemblyBOM
// / ExportBOMCSV; this only renders it. Part numbers/descriptions are blank until a component's
// iProperties feed the BOM (#718); the structure and quantities are correct regardless.

// drawBOMWindow renders the BOM panel when it is open. It rebuilds the view each frame from the
// live assembly, so placing or deleting a component updates the list immediately.
func drawBOMBody(s *app.Session) {
	drawBOMControls(s)
	native.Separator()
	if view, err := s.AssemblyBOM(); err != nil {
		native.Text(err.Error())
	} else {
		drawBOMTable(view)
	}
	native.Separator()
	if native.Button("Done") {
		s.CloseBOM()
	}
}

// drawBOMControls renders the view chooser (Structured / Parts Only) and the Export CSV button,
// which arms the file dialog to write the current view.
func drawBOMControls(s *app.Session) {
	cur := s.BOMViewKind()
	native.SetNextItemWidth(160)
	if native.BeginCombo("View##bom-view", cur.String()) {
		for _, k := range bom.ViewKinds() {
			if native.Selectable(k.String(), k == cur) {
				s.SetBOMViewKind(k)
			}
		}
		native.EndCombo()
	}
	native.SameLine()
	if native.Button("Export CSV") {
		fileModal.openFor(dialogExportBOM)
	}
}

// drawBOMTable renders the rows as a table whose headers mirror the CSV export's standard columns
// (so the on-screen and exported BOM never disagree). A structured view nests sub-assembly
// children, indented under their parent. The cells are drawn from the row fields directly rather
// than through the columns' value funcs, because the panel adds tree indentation the CSV does not.
func drawBOMTable(view *bom.View) {
	if len(view.Rows) == 0 {
		native.Text("  (no components)")
		return
	}
	columns := bom.StandardColumns()
	if !native.BeginTable("##bom-table", len(columns), 0, 0) {
		return
	}
	for _, c := range columns {
		native.TableSetupColumn(c.Header)
	}
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
	for _, r := range view.Rows {
		drawBOMRow(r, 0)
	}
	native.EndTable()
}

// drawBOMRow draws one row and recurses into its children, indenting the part-number cell by depth
// so the structured hierarchy reads as a tree.
func drawBOMRow(r *bom.Row, depth int) {
	native.TableNextRow()
	native.TableNextColumn()
	native.Text(strconv.Itoa(r.ItemNumber))
	native.TableNextColumn()
	native.Text(strings.Repeat("  ", depth) + r.PartNumber)
	native.TableNextColumn()
	native.Text(r.Description)
	native.TableNextColumn()
	native.Text(strconv.Itoa(r.Quantity))
	native.TableNextColumn()
	native.Text(r.Structure.String())
	for _, child := range r.Children {
		drawBOMRow(child, depth+1)
	}
}
