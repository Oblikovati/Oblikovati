// SPDX-License-Identifier: GPL-2.0-only

package editor

import (
	"oblikovati.org/script/console/history"
	"oblikovati.org/script/console/textbuf"
)

// Insert replaces the current selection (if any) with text and leaves the caret past it. A
// pasted multi-line string flows through the same path as a typed rune, so one undo step covers
// the whole paste (coalescing only applies to single typed runes).
func (m *Model) Insert(text string) {
	if text == "" {
		return
	}
	start, end := m.sel.Ordered()
	m.collapseTo(m.record(start, end, text))
}

// Newline inserts a line break that carries the current line's leading whitespace, so the new
// line keeps the block's indentation — the baseline auto-indent every editor provides.
func (m *Model) Newline() {
	start, _ := m.sel.Ordered()
	m.Insert("\n" + leadingWhitespace(m.buf.Line(start.Line)))
}

// Backspace deletes the selection, or the rune left of the caret when there is none (joining
// lines at a line start). It is a no-op at the document start.
func (m *Model) Backspace() {
	if m.HasSelection() {
		m.deleteSelection()
		return
	}
	from := m.buf.Left(m.sel.Caret)
	if from == m.sel.Caret {
		return
	}
	m.collapseTo(m.record(from, m.sel.Caret, ""))
}

// Delete removes the selection, or the rune right of the caret when there is none (pulling the
// next line up at a line end). It is a no-op at the document end.
func (m *Model) Delete() {
	if m.HasSelection() {
		m.deleteSelection()
		return
	}
	to := m.buf.Right(m.sel.Caret)
	if to == m.sel.Caret {
		return
	}
	m.collapseTo(m.record(m.sel.Caret, to, ""))
}

// Undo and Redo step the history, moving the caret to where the (re)applied change ends.
func (m *Model) Undo() {
	if caret, ok := m.hist.Undo(m.buf); ok {
		m.collapseTo(caret)
	}
}

func (m *Model) Redo() {
	if caret, ok := m.hist.Redo(m.buf); ok {
		m.collapseTo(caret)
	}
}

// deleteSelection removes the ordered selection span and collapses the caret to its start.
func (m *Model) deleteSelection() {
	start, end := m.sel.Ordered()
	m.collapseTo(m.record(start, end, ""))
}

// record applies a replacement of [a, c) with text, journals it for undo, and returns the
// resulting caret. It is the single mutation point so every edit is reversible.
func (m *Model) record(a, c textbuf.Position, text string) textbuf.Position {
	removed := m.buf.TextInRange(a, c)
	if removed == "" && text == "" {
		return a
	}
	caret := m.buf.ReplaceRange(a, c, text)
	m.hist.Record(history.Change{At: a, Removed: removed, Inserted: text})
	return caret
}

// collapseTo sets a bare caret at p and resets the vertical goal column to it.
func (m *Model) collapseTo(p textbuf.Position) {
	m.sel = textbuf.Selection{Anchor: p, Caret: p}
	m.goalCol = p.Col
}

// leadingWhitespace returns the run of spaces and tabs at the start of line.
func leadingWhitespace(line string) string {
	for i, r := range line {
		if r != ' ' && r != '\t' {
			return line[:i]
		}
	}
	return line
}
