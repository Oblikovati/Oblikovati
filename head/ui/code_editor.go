//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/head/internal/native"
	"oblikovati.org/script/console/editor"
	"oblikovati.org/script/console/textbuf"
)

// codeEditor is the Script Console's text editor widget. The editing brain is the headless
// editor.Model (script/console/editor); this type is the thin cgo shell that draws the model
// onto the window draw list (gutter, caret, selection, text) and translates raw key/char/mouse
// events into Model commands. Keeping the logic in the Model means this file carries no editing
// rules — only layout, input plumbing, and drawing (lua-scripting-plan, ADR-0028).
type codeEditor struct {
	model    *editor.Model
	scrollY  float32 // vertical scroll offset in logical pixels
	focused  bool    // consumes keyboard only when focused (clicked into)
	blink    float32 // caret blink phase accumulator (seconds)
	dragging bool    // a left-drag selection is in progress
}

// newCodeEditor builds an editor over initial source text.
func newCodeEditor(text string) *codeEditor { return &codeEditor{model: editor.New(text)} }

// Text returns the current source (what the console Runs); SetText replaces it.
func (e *codeEditor) Text() string     { return e.model.Text() }
func (e *codeEditor) SetText(s string) { e.model.SetText(s); e.scrollY = 0 }

// editorMetrics is the per-frame cell geometry: a fixed-width glyph advance, the line height,
// and the gutter width sized to the line-number digit count.
type editorMetrics struct {
	charW   float32
	lineH   float32
	gutterW float32
}

// metrics computes the current cell geometry from the mono font and the line count.
func (e *codeEditor) metrics() editorMetrics {
	cw := native.MonoCharWidth()
	digits := len(strconv.Itoa(e.model.LineCount()))
	return editorMetrics{charW: cw, lineH: native.MonoLineHeight(), gutterW: cw * float32(digits+2)}
}

// Draw lays the editor out at the current cursor position sized width×height: it reserves an
// input-capturing region, processes this frame's input, then renders. Call inside the console
// window's Begin/End.
func (e *codeEditor) Draw(width, height float32) {
	if width <= 0 { // 0 ⇒ fill the available content width (the console pane)
		width, _ = native.ContentRegionAvail()
	}
	ox, oy := native.GetCursorScreenPos()
	native.InvisibleButton("##code-editor", width, height)
	hovered := native.IsItemHovered()
	m := e.metrics()
	e.handleInput(ox, oy, width, height, m, hovered)
	e.render(ox, oy, width, height, m)
}

// handleInput dispatches mouse, then (only when focused) scroll, keys and typed text, finally
// keeping the caret on screen and advancing the blink clock.
func (e *codeEditor) handleInput(ox, oy, width, height float32, m editorMetrics, hovered bool) {
	e.handleMouse(ox, oy, m, hovered)
	if !e.focused {
		return
	}
	e.handleScroll(hovered, m)
	e.handleKeys()
	e.handleChars()
	e.ensureCaretVisible(height, m)
	e.clampScroll(height, m)
	e.blink += native.DeltaTime()
}

// handleMouse focuses on click, positions the caret (double-click selects a word), and extends
// the selection while dragging.
func (e *codeEditor) handleMouse(ox, oy float32, m editorMetrics, hovered bool) {
	if hovered && native.IsItemClicked(0) {
		e.focused = true
		p := e.posAt(ox, oy, m)
		if native.IsMouseDoubleClicked(0) {
			e.model.SelectWord(p)
		} else {
			e.model.SetCaret(p, native.KeyShift())
			e.dragging = true
		}
		e.resetBlink()
		return
	}
	if e.dragging && native.MouseDown(0) {
		e.model.SetCaret(e.posAt(ox, oy, m), true)
		return
	}
	e.dragging = false
}

// handleScroll wheels the view when the pointer is over the editor.
func (e *codeEditor) handleScroll(hovered bool, m editorMetrics) {
	if !hovered {
		return
	}
	if w := native.MouseWheel(); w != 0 {
		e.scrollY -= w * m.lineH * 3 // three lines per wheel notch
	}
}

// handleChars inserts this frame's typed text (Tab/Enter arrive as keys, not characters).
func (e *codeEditor) handleChars() {
	if s := native.InputChars(); s != "" {
		e.model.Insert(s)
		e.resetBlink()
	}
}

// posAt converts the current mouse position to a buffer Position (the Model clamps it). The
// half-column bias makes a click land on the nearer glyph gap.
func (e *codeEditor) posAt(ox, oy float32, m editorMetrics) textbuf.Position {
	mx, my := native.MousePos()
	line := int((my - oy + e.scrollY) / m.lineH)
	col := int((mx-ox-m.gutterW)/m.charW + 0.5)
	return textbuf.Position{Line: max0(line), Col: max0(col)}
}

// visibleRange returns the first and last line indices to draw for the current scroll/height.
func (e *codeEditor) visibleRange(height float32, m editorMetrics) (first, last int) {
	first = max0(int(e.scrollY / m.lineH))
	last = first + int(height/m.lineH) + 1
	if last >= e.model.LineCount() {
		last = e.model.LineCount() - 1
	}
	return first, last
}

// ensureCaretVisible scrolls the minimum amount to keep the caret line within the viewport.
func (e *codeEditor) ensureCaretVisible(height float32, m editorMetrics) {
	caretY := float32(e.model.Caret().Line) * m.lineH
	if caretY < e.scrollY {
		e.scrollY = caretY
	} else if caretY+m.lineH > e.scrollY+height {
		e.scrollY = caretY + m.lineH - height
	}
}

// clampScroll keeps the scroll offset within [0, maxScroll] so the view never overshoots the
// document.
func (e *codeEditor) clampScroll(height float32, m editorMetrics) {
	maxScroll := float32(e.model.LineCount())*m.lineH - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.scrollY > maxScroll {
		e.scrollY = maxScroll
	}
	if e.scrollY < 0 {
		e.scrollY = 0
	}
}

// resetBlink makes the caret solid right after an edit/caret move (the caret should be visible
// the instant it moves, then resume blinking).
func (e *codeEditor) resetBlink() { e.blink = 0 }

// max0 clamps n to a minimum of 0.
func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
