//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console/complete"
)

// Completion popup palette and layout.
var (
	colComplBg   = [4]float32{0.18, 0.18, 0.21, 0.98}
	colComplSel  = [4]float32{0.26, 0.45, 0.78, 0.85}
	colComplText = [4]float32{0.88, 0.88, 0.90, 1}
	colComplKind = [4]float32{0.55, 0.58, 0.66, 1}
)

// maxComplRows caps the popup height; the selection still scrolls through all candidates.
const maxComplRows = 8

// drawCompletion renders the autocomplete popup just below the caret: a list of candidates with
// the highlighted one filled, each prefixed by a one-letter kind tag. It draws nothing when the
// popup is not visible.
func (e *codeEditor) drawCompletion(ox, oy float32, m editorMetrics) {
	if !e.completionVisible() {
		return
	}
	c := &e.completion
	first, shown := complWindow(c.sel, len(c.items))
	x := e.colX(ox, c.ctx.ReplaceStart, m)
	y := e.lineY(oy, e.model.Caret().Line+1, m)
	w := complWidth(c.items) * m.charW
	native.DrawRectFilled(x, y, x+w, y+float32(shown)*m.lineH, colComplBg)
	for row := 0; row < shown; row++ {
		e.drawComplRow(x, y, w, m, first+row, row)
	}
}

// drawComplRow draws candidate idx at popup row, highlighting it when selected.
func (e *codeEditor) drawComplRow(x, y, w float32, m editorMetrics, idx, row int) {
	c := &e.completion
	ry := y + float32(row)*m.lineH
	if idx == c.sel {
		native.DrawRectFilled(x, ry, x+w, ry+m.lineH, colComplSel)
	}
	it := c.items[idx]
	native.DrawTextMono(x+m.charW*0.3, ry, kindTag(it.Kind), colComplKind)
	native.DrawTextMono(x+m.charW*2.2, ry, it.Text, colComplText)
}

// complWindow returns the first visible row and the visible count, scrolling the window so the
// selected candidate stays in view.
func complWindow(sel, total int) (first, shown int) {
	shown = total
	if shown > maxComplRows {
		shown = maxComplRows
	}
	first = 0
	if sel >= maxComplRows {
		first = sel - maxComplRows + 1
	}
	return first, shown
}

// complWidth returns the popup width in character cells: the longest candidate plus room for the
// kind tag and padding.
func complWidth(items []complete.Candidate) float32 {
	longest := 0
	for _, it := range items {
		if n := len([]rune(it.Text)); n > longest {
			longest = n
		}
	}
	return float32(longest + 4)
}

// kindTag is the one-letter glyph shown before a candidate: k(eyword), b(uiltin), m(odule),
// f(unction/method).
func kindTag(k complete.Kind) string {
	switch k {
	case complete.KindKeyword:
		return "k"
	case complete.KindBuiltin:
		return "b"
	case complete.KindModule:
		return "m"
	case complete.KindMethod:
		return "f"
	default:
		return " "
	}
}
