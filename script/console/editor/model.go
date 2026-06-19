// SPDX-License-Identifier: GPL-2.0-only

// Package editor is the headless command core of the Script Console code editor. It owns the
// text buffer, the caret/selection, and the undo history, and exposes the editor's behaviour
// as plain method calls (Insert, Backspace, MoveLeft, Newline, …) plus read accessors. The cgo
// widget (head/ui/code_editor.go) translates key/char/mouse events into these calls and reads
// back the state to draw — so every editing behaviour is unit-tested without the UI (ADR-0028).
package editor

import (
	"oblikovati.org/script/console/history"
	"oblikovati.org/script/console/textbuf"
)

// Model is one editor instance: the document, the current selection, and undo history. goalCol
// remembers the column a vertical-movement run aims for so Up/Down across short lines is sticky
// the way every code editor behaves.
type Model struct {
	buf     *textbuf.Buffer
	sel     textbuf.Selection
	hist    *history.History
	goalCol int
}

// New returns a Model over the given initial text with the caret at the document start.
func New(text string) *Model {
	return &Model{buf: textbuf.New(text), hist: history.New()}
}

// Text returns the whole document as a single "\n"-joined string (what Run sends to the engine).
func (m *Model) Text() string { return m.buf.String() }

// SetText replaces the whole document, collapsing the caret to the start and clearing undo —
// used when the console loads a different script.
func (m *Model) SetText(text string) {
	m.buf = textbuf.New(text)
	m.sel = textbuf.Selection{}
	m.hist = history.New()
	m.goalCol = 0
}

// LineCount and Line expose the document for the renderer and for tokenizing visible lines.
func (m *Model) LineCount() int    { return m.buf.LineCount() }
func (m *Model) Line(i int) string { return m.buf.Line(i) }

// Caret returns the current caret position; Selection returns the full anchored selection.
func (m *Model) Caret() textbuf.Position      { return m.sel.Caret }
func (m *Model) Selection() textbuf.Selection { return m.sel }
func (m *Model) HasSelection() bool           { return !m.sel.Empty() }

// CanUndo / CanRedo expose the history state for enabling toolbar actions.
func (m *Model) CanUndo() bool { return m.hist.CanUndo() }
func (m *Model) CanRedo() bool { return m.hist.CanRedo() }

// SelectedText returns the text currently selected, or "" when the selection is empty (the
// payload for Copy/Cut).
func (m *Model) SelectedText() string {
	if m.sel.Empty() {
		return ""
	}
	start, end := m.sel.Ordered()
	return m.buf.TextInRange(start, end)
}
