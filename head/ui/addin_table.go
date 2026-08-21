// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/head/internal/native"
)

// addInTableRows caps the member table's height so it shares the docked panel with the tree above
// and the Place button below; the table scrolls internally past this many rows.
const addInTableRows = 8

// tableScrolledValue records the selection value each PanelTable last scrolled into view, keyed
// windowID+"/"+controlID (mirroring treeSeeded). It lets drawTableRow scroll a programmatically
// selected row into view ONCE when the selection changes, instead of every frame — an unconditional
// scroll would fight the user's own scrolling (#1933).
var tableScrolledValue = map[string]string{}

// tableSelectionChanged reports (and records) whether a PanelTable's selection changed to value
// since it last scrolled there. True exactly once per programmatic selection change, so the
// below-the-fold pre-selected row scrolls into view on that frame and manual scrolling is left
// alone while the selection is unchanged.
func tableSelectionChanged(windowID, controlID, value string) bool {
	key := windowID + "/" + controlID
	if tableScrolledValue[key] == value {
		return false
	}
	tableScrolledValue[key] = value
	return true
}

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

// drawPanelTable renders a PanelTable: a vertically-scrolling data grid with a pinned header whose
// columns stretch to equal widths filling the dock. A row click pushes the row Key to the add-in
// (which arms Place). No per-frame allocation: the spec's strings are drawn as-is.
func drawPanelTable(s panelEditSession, windowID string, control wire.PanelControlSpec) {
	cols := len(control.TableColumns)
	if cols == 0 {
		return
	}
	h := rowHeight() * addInTableRows
	if !native.BeginTableStretch("##"+control.ID, cols, -1, h) {
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
	if row.Key == control.Value {
		// Paint the whole selected row in the chrome accent (opaque) so the selection is unmistakable
		// and PERSISTS after the click. A Selectable's own selected fill is the theme's dark resting
		// Header color, near-invisible against the row stripes — hence the explicit accent RowBg here.
		a := chromeTheme.accentColor
		native.TableSetRowBg(a[0], a[1], a[2], 1)
	}
	for c := 0; c < len(control.TableColumns); c++ {
		if !native.TableNextColumn() {
			continue
		}
		if c == 0 {
			// SpanAllColumns so hovering and clicking anywhere on the row (not just column 0) selects
			// it; the persistent selected fill is drawn above via TableSetRowBg, so this Selectable
			// is never itself "selected" (that fill would be the near-invisible dark Header).
			if native.SelectableSpanAllColumns(cellAt(row, 0), false) {
				s.PanelValueChanged(windowID, control.ID, row.Key)
			}
			// Scroll a below-the-fold pre-selected row into view once, when an add-in changes the
			// selection programmatically — otherwise the table stays at the top with the highlighted
			// row off-screen (#1933). Guarded so it fires only on a selection change, not every frame.
			if row.Key == control.Value && tableSelectionChanged(windowID, control.ID, control.Value) {
				native.SetScrollHereY()
			}
			continue
		}
		native.Text(cellAt(row, c))
	}
}
