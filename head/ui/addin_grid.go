//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui/gridlayout"
)

// Container rendering for the CSS-grid-like panel layout (ADR-0019). ImGui has no grid, so a
// grid is laid out by hand: the pure gridlayout package resolves column pixel widths and flows
// children into rows, then each cell is positioned with SetCursorPos and its content is width-
// constrained to the cell. Rows are content-height (ADR-0020): a row's height is the tallest
// cell, measured from the cursor after each cell's group. group and tabs are the simpler
// containers built on the same recursion.

const (
	// groupIndent insets a group's children under its caption, like a QGroupBox frame.
	groupIndent = 8
	// autoColumnPadding pads a measured auto-column width past the bare text so the content
	// isn't flush against the next column.
	autoColumnPadding = 16
)

// drawGrid lays its children into the grid declared by control.Columns/ColumnGap/RowGap. With
// no columns it degrades to a vertical stack.
func drawGrid(s *app.Session, windowID string, control wire.PanelControlSpec) {
	n := len(control.Columns)
	if n == 0 {
		drawControlList(s, windowID, control.Children)
		return
	}
	rows := gridlayout.FlowRows(childCells(control.Children), n)
	availW, _ := native.ContentRegionAvail()
	widths := gridlayout.ResolveColumnWidths(control.Columns, float64(availW), control.ColumnGap,
		measureAutoColumns(control, rows, n))
	offsets := columnOffsets(widths, control.ColumnGap)

	originX, originY := native.GetCursorPos()
	y := layoutGridRows(s, windowID, control, rows, widths, offsets, originX, float64(originY))
	// Leave the cursor below the whole grid, then submit a zero-size Dummy so ImGui grows the
	// parent's content bounds to here — a bare SetCursorPos that extends the boundary without a
	// following item trips ImGui's manual-layout assertion.
	native.SetCursorPos(originX, float32(y))
	native.Dummy(0, 0)
}

// layoutGridRows positions each row's cells absolutely, returning the y below the last row.
// A row's height is its tallest cell (content-height auto-flow, ADR-0020), measured from the
// cursor after each cell's group.
func layoutGridRows(s *app.Session, windowID string, control wire.PanelControlSpec,
	rows [][]gridlayout.Placement, widths, offsets []float64, originX float32, startY float64) float64 {
	y := startY
	for _, row := range rows {
		bottom := y
		for _, p := range row {
			native.SetCursorPos(originX+float32(offsets[p.Col]), float32(y))
			native.BeginGroup()
			cellW := spanWidth(widths, control.ColumnGap, p.Col, p.ColSpan)
			drawGridCell(s, windowID, p.Child, control.Children[p.Child], float32(cellW))
			native.EndGroup()
			if _, by := native.GetCursorPos(); float64(by) > bottom {
				bottom = float64(by)
			}
		}
		y = bottom + control.RowGap
	}
	return y
}

// childCells extracts each child's optional placement for the flow algorithm.
func childCells(children []wire.PanelControlSpec) []*types.GridCell {
	cells := make([]*types.GridCell, len(children))
	for i := range children {
		cells[i] = children[i].Cell
	}
	return cells
}

// columnOffsets is the left x of each column = prefix sum of widths and gaps.
func columnOffsets(widths []float64, gap float64) []float64 {
	offsets := make([]float64, len(widths))
	x := 0.0
	for i, w := range widths {
		offsets[i] = x
		x += w + gap
	}
	return offsets
}

// spanWidth is the pixel width a cell covers: its columns plus the gaps it bridges.
func spanWidth(widths []float64, gap float64, col, span int) float64 {
	if span < 1 {
		span = 1
	}
	w := 0.0
	for i := col; i < col+span && i < len(widths); i++ {
		w += widths[i]
	}
	return w + gap*float64(span-1)
}

// measureAutoColumns measures each auto column's natural width as the widest text of the
// single-span children placed in it (labels dominate auto columns), so a label column sizes to
// its content rather than flexing. Non-auto columns get 0 (ignored by the resolver).
func measureAutoColumns(control wire.PanelControlSpec, rows [][]gridlayout.Placement, n int) []float64 {
	autoW := make([]float64, n)
	for ci := range n {
		if control.Columns[ci].Kind != types.GridTrackAuto {
			continue
		}
		widest := 0.0
		for _, row := range rows {
			for _, p := range row {
				if p.Col != ci || p.ColSpan != 1 {
					continue
				}
				if w := float64(native.CalcTextWidth(control.Children[p.Child].Text)) + autoColumnPadding; w > widest {
					widest = w
				}
			}
		}
		autoW[ci] = widest
	}
	return autoW
}

// drawGridCell renders one cell. Editable controls render bare (no stacked caption — the label
// is a sibling cell) and width-constrained to the cell; everything else (labels, buttons,
// checkboxes, nested containers) delegates to the ordinary control renderer.
func drawGridCell(s *app.Session, windowID string, index int, control wire.PanelControlSpec, cellW float32) {
	switch control.Kind {
	case types.PanelTextBox, types.PanelValueEditor, types.PanelComboBox, types.PanelDropdown, types.PanelSlider:
		native.PushIDInt(index)
		defer native.PopID()
		drawCellEditable(s, windowID, control, cellW)
	default:
		drawAddInPanelControl(s, windowID, index, control)
	}
}

// drawCellEditable renders an editable control sized to cellW with a suppressed label.
func drawCellEditable(s *app.Session, windowID string, control wire.PanelControlSpec, cellW float32) {
	switch control.Kind {
	case types.PanelTextBox, types.PanelValueEditor, types.PanelComboBox:
		buf := panelBuffer(windowID+"/"+control.ID, control.Value)
		native.SetNextItemWidth(cellW)
		if native.InputText("##cell", buf) {
			s.PanelValueChanged(windowID, control.ID, bufString(buf))
		}
	case types.PanelDropdown:
		native.SetNextItemWidth(cellW)
		if native.BeginCombo("##cell", control.Value) {
			for _, opt := range control.Options {
				if native.Selectable(opt, opt == control.Value) {
					s.PanelValueChanged(windowID, control.ID, opt)
				}
			}
			native.EndCombo()
		}
	case types.PanelSlider:
		v, _ := strconv.ParseFloat(control.Value, 64)
		f := float32(v)
		native.SetNextItemWidth(cellW)
		if native.SliderFloat("##cell", &f, float32(control.Min), float32(control.Max)) {
			s.PanelValueChanged(windowID, control.ID, strconv.FormatFloat(float64(f), 'g', -1, 64))
		}
	}
}

// drawGroup renders a titled box: a caption separator over the children, indented like a frame.
func drawGroup(s *app.Session, windowID string, control wire.PanelControlSpec) {
	if control.Title != "" {
		native.SeparatorText(control.Title)
	}
	native.Indent(groupIndent)
	drawControlList(s, windowID, control.Children)
	native.Unindent(groupIndent)
}

// drawTabs renders each child as a tab whose caption is the child's Title.
func drawTabs(s *app.Session, windowID string, control wire.PanelControlSpec) {
	if !native.BeginTabBar("##tabs") {
		return
	}
	for i, pane := range control.Children {
		native.PushIDInt(i)
		if native.BeginTabItem(tabCaption(pane, i)) {
			drawTabPane(s, windowID, pane)
			native.EndTabItem()
		}
		native.PopID()
	}
	native.EndTabBar()
}

// tabCaption is a pane's tab label, falling back to a 1-based index when it has no Title.
func tabCaption(pane wire.PanelControlSpec, i int) string {
	if pane.Title != "" {
		return pane.Title
	}
	return fmt.Sprintf("Tab %d", i+1)
}

// drawTabPane renders a pane's body. A group pane renders its children without repeating its
// title (the tab already shows it); any other pane (e.g. a grid) renders normally.
func drawTabPane(s *app.Session, windowID string, pane wire.PanelControlSpec) {
	if pane.Kind == types.PanelGroup {
		drawControlList(s, windowID, pane.Children)
		return
	}
	drawAddInPanelControl(s, windowID, 0, pane)
}
