// SPDX-License-Identifier: GPL-2.0-only

package textbuf

import "testing"

func TestLeftRightWrapAcrossLines(t *testing.T) {
	b := New("ab\ncd")
	if got := b.Right(Position{0, 2}); got != (Position{1, 0}) {
		t.Errorf("Right at EOL = %+v, want {1,0}", got)
	}
	if got := b.Left(Position{1, 0}); got != (Position{0, 2}) {
		t.Errorf("Left at BOL = %+v, want {0,2}", got)
	}
	if got := b.Left(Position{0, 0}); got != (Position{0, 0}) {
		t.Errorf("Left at doc start moved: %+v", got)
	}
	if got := b.Right(Position{1, 2}); got != (Position{1, 2}) {
		t.Errorf("Right at doc end moved: %+v", got)
	}
}

func TestUpDownKeepGoalColumn(t *testing.T) {
	b := New("longline\nx\nanother")
	// From col 6 on line 0, Up/Down keep the goal column, clamping on the short middle line.
	down := b.Down(Position{0, 6}, 6)
	if down != (Position{1, 1}) {
		t.Fatalf("Down onto short line = %+v, want {1,1}", down)
	}
	back := b.Down(down, 6) // goal column survives the short line
	if back != (Position{2, 6}) {
		t.Errorf("Down restoring goal column = %+v, want {2,6}", back)
	}
}

func TestLineHomeTogglesIndent(t *testing.T) {
	b := New("    code")
	if got := b.LineHome(Position{0, 8}); got != (Position{0, 4}) {
		t.Fatalf("Home from EOL = %+v, want first-non-blank {0,4}", got)
	}
	if got := b.LineHome(Position{0, 4}); got != (Position{0, 0}) {
		t.Errorf("Home at indent = %+v, want col 0", got)
	}
}

func TestWordRightStopsAtBoundaries(t *testing.T) {
	b := New("foo.bar baz")
	steps := []Position{{0, 3}, {0, 4}, {0, 8}, {0, 11}}
	p := Position{0, 0}
	for i, want := range steps {
		p = b.WordRight(p)
		if p != want {
			t.Fatalf("WordRight step %d = %+v, want %+v", i, p, want)
		}
	}
}

func TestWordLeftStopsAtWordStart(t *testing.T) {
	b := New("foo bar")
	if got := b.WordLeft(Position{0, 7}); got != (Position{0, 4}) {
		t.Fatalf("WordLeft from EOL = %+v, want {0,4}", got)
	}
	if got := b.WordLeft(Position{0, 4}); got != (Position{0, 0}) {
		t.Errorf("WordLeft from word start = %+v, want {0,0}", got)
	}
}

func TestSelectionOrderedAndEmpty(t *testing.T) {
	s := Selection{Anchor: Position{2, 1}, Caret: Position{0, 3}}
	start, end := s.Ordered()
	if start != (Position{0, 3}) || end != (Position{2, 1}) {
		t.Errorf("Ordered = (%+v,%+v), want caret-first", start, end)
	}
	if (Selection{Anchor: Position{1, 1}, Caret: Position{1, 1}}).Empty() != true {
		t.Error("equal anchor/caret should be Empty")
	}
}
