//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console/textbuf"
)

// Editor palette. The editor draws its own background so its contrast is stable regardless of
// the app theme; Phase 3 will route these through the theme token system (ADR-0021).
var (
	colEditorBg    = [4]float32{0.12, 0.12, 0.14, 1}
	colEditorText  = [4]float32{0.86, 0.86, 0.88, 1}
	colGutterBg    = [4]float32{0.16, 0.16, 0.19, 1}
	colGutterText  = [4]float32{0.45, 0.46, 0.52, 1}
	colCurrentLine = [4]float32{1, 1, 1, 0.045}
	colSelection   = [4]float32{0.26, 0.45, 0.78, 0.45}
	colCaret       = [4]float32{0.92, 0.92, 0.97, 1}
)

// render draws the editor: background, current-line highlight, selection, text, gutter, caret —
// all clipped to the editor rectangle so scrolled content never bleeds past the viewport.
func (e *codeEditor) render(ox, oy, width, height float32, m editorMetrics) {
	native.PushClipRect(ox, oy, ox+width, oy+height)
	native.DrawRectFilled(ox, oy, ox+width, oy+height, colEditorBg)
	first, last := e.visibleRange(height, m)
	e.drawCurrentLine(ox, oy, width, m)
	e.drawSelection(ox, oy, m, first, last)
	e.drawText(ox, oy, m, first, last)
	e.drawGutter(ox, oy, width, height, m, first, last)
	e.drawCaret(ox, oy, m)
	native.PopClipRect()
}

// lineY returns the top y of line i in screen space for the current scroll.
func (e *codeEditor) lineY(oy float32, i int, m editorMetrics) float32 {
	return oy + float32(i)*m.lineH - e.scrollY
}

// colX returns the screen x of text column col (past the gutter).
func (e *codeEditor) colX(ox float32, col int, m editorMetrics) float32 {
	return ox + m.gutterW + float32(col)*m.charW
}

// drawText renders each visible line's source in the fixed-width face. Phase 3 replaces the
// single colour with per-token highlighting from the lualex tokenizer.
func (e *codeEditor) drawText(ox, oy float32, m editorMetrics, first, last int) {
	for i := first; i <= last; i++ {
		native.DrawTextMono(ox+m.gutterW, e.lineY(oy, i, m), e.model.Line(i), colEditorText)
	}
}

// drawCurrentLine tints the caret's line across the text area (only when nothing is selected,
// so it does not fight the selection highlight).
func (e *codeEditor) drawCurrentLine(ox, oy, width float32, m editorMetrics) {
	if e.model.HasSelection() {
		return
	}
	y := e.lineY(oy, e.model.Caret().Line, m)
	native.DrawRectFilled(ox+m.gutterW, y, ox+width, y+m.lineH, colCurrentLine)
}

// drawSelection fills the selected span across the visible lines: a partial first/last line and
// full-width middle lines, extended half a cell past a line end to signal the newline is included.
func (e *codeEditor) drawSelection(ox, oy float32, m editorMetrics, first, last int) {
	if !e.model.HasSelection() {
		return
	}
	sel := e.model.Selection()
	start, end := sel.Ordered()
	for i := max0(first); i <= last && i <= end.Line; i++ {
		if i < start.Line {
			continue
		}
		x0, x1 := e.selRowExtent(ox, m, i, start, end)
		y := e.lineY(oy, i, m)
		native.DrawRectFilled(x0, y, x1, y+m.lineH, colSelection)
	}
}

// selRowExtent returns the selection rectangle's x bounds on line i given the ordered selection
// endpoints. Lines strictly inside the selection run to half a cell past their end.
func (e *codeEditor) selRowExtent(ox float32, m editorMetrics, i int, start, end textbuf.Position) (x0, x1 float32) {
	startCol := 0
	if i == start.Line {
		startCol = start.Col
	}
	if i == end.Line {
		return e.colX(ox, startCol, m), e.colX(ox, end.Col, m)
	}
	endX := e.colX(ox, len([]rune(e.model.Line(i))), m) + m.charW/2
	return e.colX(ox, startCol, m), endX
}

// drawGutter fills the gutter strip and right-aligns each visible line number within it.
func (e *codeEditor) drawGutter(ox, oy, width, height float32, m editorMetrics, first, last int) {
	_ = width
	native.DrawRectFilled(ox, oy, ox+m.gutterW, oy+height, colGutterBg)
	for i := first; i <= last; i++ {
		num := strconv.Itoa(i + 1)
		x := ox + m.gutterW - m.charW*float32(len(num)+1)
		native.DrawTextMono(x, e.lineY(oy, i, m), num, colGutterText)
	}
}

// drawCaret draws the blinking caret bar at the caret position when the editor is focused.
func (e *codeEditor) drawCaret(ox, oy float32, m editorMetrics) {
	if !e.focused || int(e.blink*2)%2 != 0 {
		return
	}
	c := e.model.Caret()
	x := e.colX(ox, c.Col, m)
	y := e.lineY(oy, c.Line, m)
	native.DrawRectFilled(x, y, x+1.5, y+m.lineH, colCaret)
}
