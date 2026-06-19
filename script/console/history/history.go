// SPDX-License-Identifier: GPL-2.0-only

// Package history is the undo/redo stack for the Script Console editor. It records every
// buffer mutation as a reversible Change (the text removed and the text inserted at a point)
// and replays the inverse on Undo. Consecutive single-rune typing or deletion is coalesced
// into one Change so a word is one undo step, not one-per-keystroke — the behaviour a code
// editor user expects. It is pure Go over textbuf, so the whole stack is unit-tested headlessly.
package history

import "oblikovati.org/script/console/textbuf"

// Change is one reversible edit: at At, the span holding Removed was replaced by Inserted.
// A pure insertion has Removed == "" and a pure deletion has Inserted == "".
type Change struct {
	At       textbuf.Position
	Removed  string
	Inserted string
}

// inverse is the Change that undoes c: swap what was removed and inserted at the same anchor.
func (c Change) inverse() Change {
	return Change{At: c.At, Removed: c.Inserted, Inserted: c.Removed}
}

// apply replays c against buf and returns the caret position past the inserted text. It is the
// single primitive Undo/Redo use, so forward and reverse replay share one code path.
func (c Change) apply(buf *textbuf.Buffer) textbuf.Position {
	end := textbuf.Advance(c.At, c.Removed)
	return buf.ReplaceRange(c.At, end, c.Inserted)
}

// History is the per-editor undo/redo stack: done holds applied changes (newest last) and
// undone holds changes rolled back by Undo (available to Redo). Any fresh Record clears undone.
type History struct {
	done   []Change
	undone []Change
}

// New returns an empty history.
func New() *History { return &History{} }

// CanUndo and CanRedo report whether the respective action has anything to do.
func (h *History) CanUndo() bool { return len(h.done) > 0 }
func (h *History) CanRedo() bool { return len(h.undone) > 0 }

// Record pushes c onto the undo stack, first trying to coalesce it into the previous change so
// a typing/deletion run collapses to one undo step. Recording a new edit invalidates any
// redo history.
func (h *History) Record(c Change) {
	h.undone = nil
	if n := len(h.done); n > 0 && coalesce(&h.done[n-1], c) {
		return
	}
	h.done = append(h.done, c)
}

// Undo reverses the most recent change against buf and returns the resulting caret; ok is
// false when there is nothing to undo.
func (h *History) Undo(buf *textbuf.Buffer) (caret textbuf.Position, ok bool) {
	n := len(h.done)
	if n == 0 {
		return textbuf.Position{}, false
	}
	c := h.done[n-1]
	h.done = h.done[:n-1]
	h.undone = append(h.undone, c)
	return c.inverse().apply(buf), true
}

// Redo re-applies the most recently undone change against buf and returns the resulting caret;
// ok is false when there is nothing to redo.
func (h *History) Redo(buf *textbuf.Buffer) (caret textbuf.Position, ok bool) {
	n := len(h.undone)
	if n == 0 {
		return textbuf.Position{}, false
	}
	c := h.undone[n-1]
	h.undone = h.undone[:n-1]
	h.done = append(h.done, c)
	return c.apply(buf), true
}
