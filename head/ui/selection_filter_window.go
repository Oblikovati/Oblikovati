//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// selectionFilterPayload tags the drag-and-drop payload carrying a priority-row index, so a
// dropped row only reorders within this window (#1222).
const selectionFilterPayload = "OBK_SELFILTER_ROW"

// drawSelectionFilterWindow renders the Selection Filter & Priority window while it is open
// (#1222): one checkbox per pickable entity type (enable/disable picking it) over a drag-to-
// reorder priority list where the top row wins an ambiguous pick, plus Select All / Deselect All.
// The X in the title bar closes it. Every edit writes straight back to the session's
// SelectionFilterState, so the next click honours it immediately.
func drawSelectionFilterWindow(s *app.Session) {
	if !s.SelectionFilterWindowOpen() {
		return
	}
	native.SetNextWindowSizeOnce(280, 380)
	visible, open := native.BeginClosable("Selection Filter###selection-filter")
	if visible {
		drawSelectionFilterBody(s)
	}
	native.End()
	if !open {
		s.CloseSelectionFilterWindow()
	}
}

// drawSelectionFilterBody draws the explanatory header, the reorderable kind rows, and the
// Select All / Deselect All buttons.
func drawSelectionFilterBody(s *app.Session) {
	st := s.SelectionFilterState()
	native.Text("Tick to allow picking. Drag a row to set priority")
	native.Text("(the topmost type wins an ambiguous pick).")
	native.Separator()
	drawSelectionFilterRows(st)
	native.Separator()
	if native.Button("Select All") {
		st.EnableAll()
	}
	native.SameLine()
	if native.Button("Deselect All") {
		st.DisableAll()
	}
}

// drawSelectionFilterRows draws one checkbox+label row per kind in priority order; each row is a
// drag source and drop target so dragging one onto another reorders the priority list.
func drawSelectionFilterRows(st *app.SelectionFilterState) {
	for i, k := range st.Order() {
		native.PushIDInt(i)
		on := st.Enabled(k)
		if native.Checkbox("##enabled", &on) {
			st.SetEnabled(k, on)
		}
		native.SameLine()
		native.Selectable(app.SelectionKindLabel(k), false) // the draggable row handle
		reorderSelectionFilterRow(st, i, k)
		native.PopID()
	}
}

// reorderSelectionFilterRow wires the last-drawn row as a drag source (carrying its index) and a
// drop target that moves the dragged row to this position.
func reorderSelectionFilterRow(st *app.SelectionFilterState, i int, k app.SelectionKind) {
	if native.BeginDragDropSource() {
		native.SetDragDropPayloadInt(selectionFilterPayload, i)
		native.Text(app.SelectionKindLabel(k))
		native.EndDragDropSource()
	}
	if native.BeginDragDropTarget() {
		if from, ok := native.AcceptDragDropPayloadInt(selectionFilterPayload); ok {
			st.Move(from, i)
		}
		native.EndDragDropTarget()
	}
}
