//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console/lualex"
	"oblikovati.org/script/console/textbuf"
)

// Editor palette. The editor draws its own background so its contrast is stable regardless of
// the app theme; routing these through the theme token system (ADR-0021) is a follow-up.
var (
	colEditorBg    = [4]float32{0.12, 0.12, 0.14, 1}
	colEditorText  = [4]float32{0.86, 0.86, 0.88, 1}
	colGutterBg    = [4]float32{0.16, 0.16, 0.19, 1}
	colGutterText  = [4]float32{0.45, 0.46, 0.52, 1}
	colCurrentLine = [4]float32{1, 1, 1, 0.045}
	colSelection   = [4]float32{0.26, 0.45, 0.78, 0.45}
	colCaret       = [4]float32{0.92, 0.92, 0.97, 1}
	colBracket     = [4]float32{0.45, 0.55, 0.40, 0.55}
)

// syntaxPalette maps each token class to its colour — a conventional dark code-editor scheme
// (keywords magenta, strings amber, numbers green, comments grey, builtins cyan).
var syntaxPalette = map[lualex.Kind][4]float32{
	lualex.KindKeyword:  {0.80, 0.47, 0.85, 1},
	lualex.KindBuiltin:  {0.36, 0.72, 0.79, 1},
	lualex.KindString:   {0.83, 0.66, 0.40, 1},
	lualex.KindNumber:   {0.60, 0.80, 0.50, 1},
	lualex.KindComment:  {0.42, 0.46, 0.42, 1},
	lualex.KindOperator: {0.74, 0.74, 0.80, 1},
}

// tokenColor returns the colour for a token kind, defaulting to the plain text colour for
// identifiers (and any unmapped kind).
func tokenColor(k lualex.Kind) [4]float32 {
	if c, ok := syntaxPalette[k]; ok {
		return c
	}
	return colEditorText
}

// render draws the editor: background, current-line highlight, selection, text, gutter, caret —
// all clipped to the editor rectangle so scrolled content never bleeds past the viewport.
func (e *codeEditor) render(ox, oy, width, height float32, m editorMetrics) {
	native.PushClipRect(ox, oy, ox+width, oy+height)
	native.DrawRectFilled(ox, oy, ox+width, oy+height, colEditorBg)
	first, last := e.visibleRange(height, m)
	e.drawCurrentLine(ox, oy, width, m)
	e.drawSelection(ox, oy, m, first, last)
	e.drawBracketMatch(ox, oy, m)
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

// drawText renders each visible line's source with per-token syntax highlighting. The lualex
// State threads across lines (so a long string/comment opened above colours correctly), so the
// scan starts from the top of the buffer up to the first visible line, then draws forward.
func (e *codeEditor) drawText(ox, oy float32, m editorMetrics, first, last int) {
	st := e.startState(first)
	for i := first; i <= last; i++ {
		line := e.model.Line(i)
		toks, next := lualex.TokenizeLine(line, st)
		e.drawTokens(ox, e.lineY(oy, i, m), m, line, toks)
		st = next
	}
}

// startState returns the tokenizer State at the top of line `first` by scanning every earlier
// line. This is O(first) per frame — negligible for an interactive console script.
func (e *codeEditor) startState(first int) lualex.State {
	var st lualex.State
	for i := 0; i < first; i++ {
		_, st = lualex.TokenizeLine(e.model.Line(i), st)
	}
	return st
}

// drawTokens draws one line's coloured token spans (whitespace between tokens is left as the
// editor background).
func (e *codeEditor) drawTokens(ox, y float32, m editorMetrics, line string, toks []lualex.Token) {
	r := []rune(line)
	for _, t := range toks {
		native.DrawTextMono(e.colX(ox, t.Start, m), y, string(r[t.Start:t.End]), tokenColor(t.Kind))
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

// drawBracketMatch outlines the bracket under/next to the caret and its partner with a subtle
// box (drawn under the text so the glyphs stay legible).
func (e *codeEditor) drawBracketMatch(ox, oy float32, m editorMetrics) {
	a, b, ok := e.model.MatchingBracket()
	if !ok {
		return
	}
	for _, p := range [2]textbuf.Position{a, b} {
		x, y := e.colX(ox, p.Col, m), e.lineY(oy, p.Line, m)
		native.DrawRectFilled(x, y, x+m.charW, y+m.lineH, colBracket)
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
