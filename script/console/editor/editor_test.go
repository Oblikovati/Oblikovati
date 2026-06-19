// SPDX-License-Identifier: GPL-2.0-only

package editor

import (
	"testing"

	"oblikovati.org/script/console/textbuf"
)

func pos(line, col int) textbuf.Position { return textbuf.Position{Line: line, Col: col} }

func TestInsertAtCaretAdvances(t *testing.T) {
	m := New("")
	m.Insert("hi")
	if m.Text() != "hi" || m.Caret() != pos(0, 2) {
		t.Fatalf("text=%q caret=%+v, want %q at {0,2}", m.Text(), m.Caret(), "hi")
	}
}

func TestInsertReplacesSelection(t *testing.T) {
	m := New("hello world")
	m.SelectWord(pos(0, 1)) // select "hello"
	m.Insert("hi")
	if m.Text() != "hi world" {
		t.Fatalf("text=%q, want %q", m.Text(), "hi world")
	}
	if m.HasSelection() {
		t.Error("selection should collapse after typing over it")
	}
}

func TestNavigationAndSetTextCoverage(t *testing.T) {
	m := New("alpha\nbeta gamma")
	m.MoveDocEnd(false)
	if m.Caret() != pos(1, 10) {
		t.Fatalf("DocEnd caret=%+v, want {1,10}", m.Caret())
	}
	m.MoveWordLeft(false)
	if m.Caret() != pos(1, 5) {
		t.Errorf("WordLeft caret=%+v, want start of 'gamma' {1,5}", m.Caret())
	}
	m.MoveHome(false) // first non-blank == col 0 here
	m.MoveUp(true)    // extend up a line
	if !m.HasSelection() {
		t.Error("shift-up should create a selection")
	}
	m.MoveDocStart(false)
	if m.Caret() != pos(0, 0) || m.HasSelection() {
		t.Errorf("DocStart should collapse to {0,0}, got %+v sel=%v", m.Caret(), m.HasSelection())
	}
	m.SetText("fresh")
	if m.Text() != "fresh" || m.CanUndo() {
		t.Errorf("SetText should reset doc and history: text=%q canUndo=%v", m.Text(), m.CanUndo())
	}
}

func TestSelectWordOffWordCollapses(t *testing.T) {
	m := New("a  b")
	m.SelectWord(pos(0, 1)) // on a space
	if m.HasSelection() {
		t.Errorf("double-click off a word should not select: %q", m.SelectedText())
	}
}

func TestBackspaceJoinsLines(t *testing.T) {
	m := New("ab\ncd")
	m.SetCaret(pos(1, 0), false)
	m.Backspace()
	if m.Text() != "abcd" || m.Caret() != pos(0, 2) {
		t.Fatalf("text=%q caret=%+v, want %q at {0,2}", m.Text(), m.Caret(), "abcd")
	}
}

func TestDeleteForwardPullsNextLine(t *testing.T) {
	m := New("ab\ncd")
	m.SetCaret(pos(0, 2), false)
	m.Delete()
	if m.Text() != "abcd" {
		t.Fatalf("text=%q, want %q", m.Text(), "abcd")
	}
}

func TestNewlineAutoIndents(t *testing.T) {
	m := New("    foo")
	m.MoveEnd(false)
	m.Newline()
	if m.Text() != "    foo\n    " {
		t.Fatalf("text=%q, want the new line to inherit the 4-space indent", m.Text())
	}
	if m.Caret() != pos(1, 4) {
		t.Errorf("caret=%+v, want {1,4} (after inherited indent)", m.Caret())
	}
}

func TestUndoRedoThroughModel(t *testing.T) {
	m := New("")
	m.Insert("foo")
	m.Insert(" bar")
	m.Undo() // removes " bar"
	if m.Text() != "foo" {
		t.Fatalf("after undo text=%q, want %q", m.Text(), "foo")
	}
	m.Redo()
	if m.Text() != "foo bar" {
		t.Fatalf("after redo text=%q, want %q", m.Text(), "foo bar")
	}
}

func TestArrowCollapsesSelectionToEdge(t *testing.T) {
	m := New("hello")
	m.SelectAll()
	m.MoveLeft(false) // non-extending: collapse to the start
	if m.HasSelection() || m.Caret() != pos(0, 0) {
		t.Fatalf("caret=%+v hasSel=%v, want collapse to {0,0}", m.Caret(), m.HasSelection())
	}
}

func TestVerticalMoveKeepsGoalColumn(t *testing.T) {
	m := New("longline\nx\nlongline")
	m.SetCaret(pos(0, 6), false)
	m.MoveDown(false) // clamps onto the short middle line
	if m.Caret() != pos(1, 1) {
		t.Fatalf("caret=%+v, want clamp to {1,1}", m.Caret())
	}
	m.MoveDown(false) // goal column 6 should be restored on the long line
	if m.Caret() != pos(2, 6) {
		t.Errorf("caret=%+v, want goal column restored at {2,6}", m.Caret())
	}
}

func TestDeleteAndBackspaceWithSelection(t *testing.T) {
	m := New("foo bar baz")
	m.SelectWord(pos(0, 5)) // "bar"
	m.Delete()              // delete-with-selection path
	if m.Text() != "foo  baz" {
		t.Fatalf("Delete selection text=%q, want %q", m.Text(), "foo  baz")
	}
	m2 := New("foo bar baz")
	m2.SelectWord(pos(0, 5))
	m2.Backspace() // backspace-with-selection path
	if m2.Text() != "foo  baz" {
		t.Fatalf("Backspace selection text=%q, want %q", m2.Text(), "foo  baz")
	}
}

func TestEditNoOpsAtBoundaries(t *testing.T) {
	m := New("ab")
	m.MoveDocStart(false)
	m.Backspace() // at doc start: no-op
	m.MoveDocEnd(false)
	m.Delete() // at doc end: no-op
	m.Insert("")
	if m.Text() != "ab" || m.CanUndo() {
		t.Errorf("boundary no-ops mutated state: text=%q canUndo=%v", m.Text(), m.CanUndo())
	}
}

func TestAccessorsAndRightCollapse(t *testing.T) {
	m := New("ab\ncde")
	if m.LineCount() != 2 || m.Line(1) != "cde" {
		t.Fatalf("LineCount/Line = %d/%q, want 2/%q", m.LineCount(), m.Line(1), "cde")
	}
	m.SelectAll()
	if m.Selection().Empty() {
		t.Error("expected a non-empty selection after SelectAll")
	}
	m.MoveRight(false) // collapse-to-end branch
	if m.HasSelection() || m.Caret() != pos(1, 3) {
		t.Errorf("MoveRight should collapse to end {1,3}, got %+v sel=%v", m.Caret(), m.HasSelection())
	}
	m.MoveDocStart(false)
	m.MoveWordRight(false) // past "ab" to the next token start, wrapping the line break
	if m.Caret() != pos(1, 0) {
		t.Errorf("WordRight caret=%+v, want {1,0}", m.Caret())
	}
}

func TestCanRedoReflectsHistory(t *testing.T) {
	m := New("")
	if m.CanRedo() {
		t.Error("fresh model should have nothing to redo")
	}
	m.Insert("z")
	m.Undo()
	if !m.CanRedo() {
		t.Error("after undo, redo should be available")
	}
}

func TestSelectWordOnDoubleClick(t *testing.T) {
	m := New("foo barbaz qux")
	m.SelectWord(pos(0, 5)) // inside "barbaz"
	if m.SelectedText() != "barbaz" {
		t.Errorf("selected %q, want %q", m.SelectedText(), "barbaz")
	}
}
