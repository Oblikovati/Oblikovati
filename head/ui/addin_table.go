// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/head/internal/native"
)

// addInTableRows caps the member table's height so it shares the docked panel with the tree above
// and the Place button below; the table scrolls internally past this many rows.
const addInTableRows = 8

// rowHeight is one table row's pixel height: the font's line height plus a small padding margin.
// No table-sizing helper existed outside the Parameters dialog's own paramTableHeight (which bakes
// in a fixed row px and an 8-row cap tuned for that grid), so PanelTable gets its own — this stays
// proportional to the active font instead of a hardcoded pixel count.
func rowHeight() float32 {
	return native.TextLineHeight() + 6
}

// cellAt returns row's cell for column col, or "" when the row is shorter than the header (a
// ragged catalog row must not panic the render loop).
func cellAt(row wire.TableRow, col int) string {
	if col < 0 || col >= len(row.Cells) {
		return ""
	}
	return row.Cells[col]
}

// drawPanelTable renders a PanelTable: a scrolling, horizontally-scrolling data grid with a pinned
// header. A row click pushes the row Key to the add-in (which arms Place). No per-frame allocation:
// the spec's strings are drawn as-is.
func drawPanelTable(s panelEditSession, windowID string, control wire.PanelControlSpec) {
	cols := len(control.TableColumns)
	if cols == 0 {
		return
	}
	h := rowHeight() * addInTableRows
	if !native.BeginTableScrollX("##"+control.ID, cols, -1, h) {
		return
	}
	defer native.EndTable()
	for _, name := range control.TableColumns {
		native.TableSetupColumn(name)
	}
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
	for i := range control.TableRows {
		drawTableRow(s, windowID, control, i)
	}
}

// drawTableRow draws one selectable row spanning all columns; the first column carries a
// row-spanning Selectable so a click anywhere on the row selects it.
func drawTableRow(s panelEditSession, windowID string, control wire.PanelControlSpec, i int) {
	row := control.TableRows[i]
	native.PushIDInt(i)
	defer native.PopID()
	native.TableNextRow()
	for c := 0; c < len(control.TableColumns); c++ {
		if !native.TableNextColumn() {
			continue
		}
		if c == 0 {
			if native.Selectable(cellAt(row, 0), row.Key == control.Value) {
				s.PanelValueChanged(windowID, control.ID, row.Key)
			}
			continue
		}
		native.Text(cellAt(row, c))
	}
}
