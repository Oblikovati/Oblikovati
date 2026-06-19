// SPDX-License-Identifier: GPL-2.0-only

package editor

import "oblikovati.org/script/console/textbuf"

// The Move* methods move the caret. extend=true keeps the selection anchor (shift-movement,
// growing/shrinking the range); extend=false drops a bare caret. Horizontal and word/line moves
// reset the vertical goal column; Up/Down preserve it so a run of vertical moves tracks the
// original column across short lines.

// MoveLeft / MoveRight step one rune, wrapping across line boundaries. With a non-extending
// move and an active selection they collapse to the near edge rather than stepping (the
// familiar arrow-collapses-selection behaviour).
func (m *Model) MoveLeft(extend bool) {
	if !extend && m.HasSelection() {
		start, _ := m.sel.Ordered()
		m.collapseTo(start)
		return
	}
	m.moveHoriz(m.buf.Left(m.sel.Caret), extend)
}

func (m *Model) MoveRight(extend bool) {
	if !extend && m.HasSelection() {
		_, end := m.sel.Ordered()
		m.collapseTo(end)
		return
	}
	m.moveHoriz(m.buf.Right(m.sel.Caret), extend)
}

// MoveWordLeft / MoveWordRight jump by word.
func (m *Model) MoveWordLeft(extend bool)  { m.moveHoriz(m.buf.WordLeft(m.sel.Caret), extend) }
func (m *Model) MoveWordRight(extend bool) { m.moveHoriz(m.buf.WordRight(m.sel.Caret), extend) }

// MoveHome toggles between the first non-blank column and column 0; MoveEnd goes to line end.
func (m *Model) MoveHome(extend bool) { m.moveHoriz(m.buf.LineHome(m.sel.Caret), extend) }
func (m *Model) MoveEnd(extend bool)  { m.moveHoriz(m.buf.LineEnd(m.sel.Caret), extend) }

// MoveDocStart / MoveDocEnd jump to the document bounds.
func (m *Model) MoveDocStart(extend bool) { m.moveHoriz(m.buf.DocStart(), extend) }
func (m *Model) MoveDocEnd(extend bool)   { m.moveHoriz(m.buf.DocEnd(), extend) }

// MoveUp / MoveDown move a line, keeping the vertical goal column.
func (m *Model) MoveUp(extend bool)   { m.moveVert(m.buf.Up(m.sel.Caret, m.goalCol), extend) }
func (m *Model) MoveDown(extend bool) { m.moveVert(m.buf.Down(m.sel.Caret, m.goalCol), extend) }

// SetCaret places the caret at an arbitrary (clamped) position — a mouse click, optionally
// shift-clicking to extend the selection.
func (m *Model) SetCaret(p textbuf.Position, extend bool) { m.moveHoriz(m.buf.Clamp(p), extend) }

// SelectAll selects the whole document with the caret at the end.
func (m *Model) SelectAll() {
	m.sel = textbuf.Selection{Anchor: m.buf.DocStart(), Caret: m.buf.DocEnd()}
	m.goalCol = m.sel.Caret.Col
}

// SelectWord selects the word under p (a double-click), or leaves a bare caret when p is not on
// a word rune.
func (m *Model) SelectWord(p textbuf.Position) {
	p = m.buf.Clamp(p)
	start := m.buf.WordStartAt(p)
	end := m.buf.WordEndAt(p)
	if start == end {
		m.collapseTo(p)
		return
	}
	m.sel = textbuf.Selection{Anchor: start, Caret: end}
	m.goalCol = end.Col
}

// moveHoriz applies a horizontal/absolute caret target and resets the goal column.
func (m *Model) moveHoriz(p textbuf.Position, extend bool) {
	m.applyCaret(p, extend)
	m.goalCol = p.Col
}

// moveVert applies a vertical caret target, leaving the goal column intact.
func (m *Model) moveVert(p textbuf.Position, extend bool) { m.applyCaret(p, extend) }

// applyCaret sets the caret to p, dropping the anchor unless the move extends the selection.
func (m *Model) applyCaret(p textbuf.Position, extend bool) {
	m.sel.Caret = p
	if !extend {
		m.sel.Anchor = p
	}
}
