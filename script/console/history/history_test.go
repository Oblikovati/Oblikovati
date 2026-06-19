// SPDX-License-Identifier: GPL-2.0-only

package history

import (
	"testing"

	"oblikovati.org/script/console/textbuf"
)

// pos is a keyed-literal shorthand so the table stays readable without tripping vet's
// unkeyed-fields check on the cross-package textbuf.Position.
func pos(line, col int) textbuf.Position { return textbuf.Position{Line: line, Col: col} }

// typeRun records single-rune insertions as the editor would while the user types s starting
// at p, applying each to buf — the path coalescing must collapse to one undo step per word.
func typeRun(h *History, buf *textbuf.Buffer, p textbuf.Position, s string) textbuf.Position {
	for _, r := range s {
		ins := string(r)
		buf.Insert(p, ins)
		h.Record(Change{At: p, Inserted: ins})
		p = textbuf.Advance(p, ins)
	}
	return p
}

func TestUndoRedoRestoresText(t *testing.T) {
	buf := textbuf.New("")
	h := New()
	typeRun(h, buf, pos(0, 0), "ab")
	if buf.String() != "ab" {
		t.Fatalf("after typing = %q, want %q", buf.String(), "ab")
	}
	if _, ok := h.Undo(buf); !ok || buf.String() != "" {
		t.Fatalf("undo: ok=%v buf=%q, want true and empty", ok, buf.String())
	}
	if _, ok := h.Redo(buf); !ok || buf.String() != "ab" {
		t.Fatalf("redo: ok=%v buf=%q, want true and %q", ok, buf.String(), "ab")
	}
}

func TestTypingRunCoalescesToOneUndo(t *testing.T) {
	buf := textbuf.New("")
	h := New()
	typeRun(h, buf, pos(0, 0), "foo bar") // space breaks the run
	if len(h.done) != 3 {
		t.Fatalf("undo steps = %d, want 3 (foo / space / bar)", len(h.done))
	}
	caret, _ := h.Undo(buf)
	if buf.String() != "foo " || caret != (pos(0, 4)) {
		t.Fatalf("first undo = %q caret %+v, want %q at {0,4}", buf.String(), caret, "foo ")
	}
}

func TestBackspaceRunCoalesces(t *testing.T) {
	buf := textbuf.New("abc")
	h := New()
	// Backspace from end: remove 'c', then 'b', then 'a' — one growing-leftward deletion.
	for col := 3; col > 0; col-- {
		at := pos(0, col-1)
		removed := buf.Line(0)[col-1 : col]
		buf.DeleteRange(at, pos(0, col))
		h.Record(Change{At: at, Removed: removed})
	}
	if len(h.done) != 1 {
		t.Fatalf("backspace undo steps = %d, want 1", len(h.done))
	}
	if _, ok := h.Undo(buf); !ok || buf.String() != "abc" {
		t.Fatalf("undo backspace run: ok=%v buf=%q, want %q restored", ok, buf.String(), "abc")
	}
}

func TestRecordInvalidatesRedo(t *testing.T) {
	buf := textbuf.New("")
	h := New()
	typeRun(h, buf, pos(0, 0), "x")
	h.Undo(buf)
	if !h.CanRedo() {
		t.Fatal("expected redo available after undo")
	}
	typeRun(h, buf, pos(0, 0), "y")
	if h.CanRedo() {
		t.Error("a new edit must clear the redo stack")
	}
}

func TestReplacementIsNotCoalesced(t *testing.T) {
	h := New()
	h.Record(Change{At: pos(0, 0), Removed: "a", Inserted: "b"})
	h.Record(Change{At: pos(0, 1), Removed: "c", Inserted: "d"})
	if len(h.done) != 2 {
		t.Errorf("replacements coalesced: steps = %d, want 2", len(h.done))
	}
}
