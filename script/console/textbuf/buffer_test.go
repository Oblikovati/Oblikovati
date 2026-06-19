// SPDX-License-Identifier: GPL-2.0-only

package textbuf

import "testing"

func TestNewNormalizesCRLFAndKeepsOneLine(t *testing.T) {
	b := New("a\r\nb\nc")
	if got := b.LineCount(); got != 3 {
		t.Fatalf("LineCount = %d, want 3", got)
	}
	if got := b.Line(0); got != "a" {
		t.Errorf("line 0 = %q, want %q (trailing \\r not stripped)", got, "a")
	}
	if b2 := New(""); b2.LineCount() != 1 || b2.Line(0) != "" {
		t.Errorf("empty source = %d lines %q, want 1 empty line", b2.LineCount(), b2.Line(0))
	}
}

func TestInsertSingleLineReturnsEndCaret(t *testing.T) {
	b := New("héllo") // multi-byte rune to prove columns count runes, not bytes
	end := b.Insert(Position{0, 5}, "!")
	if end != (Position{0, 6}) {
		t.Fatalf("end = %+v, want {0,6}", end)
	}
	if b.String() != "héllo!" {
		t.Errorf("String = %q, want %q", b.String(), "héllo!")
	}
}

func TestInsertMultilineSplitsAndStitchesTail(t *testing.T) {
	b := New("abcd")
	end := b.Insert(Position{0, 2}, "X\nY\nZ")
	if end != (Position{2, 1}) {
		t.Fatalf("end = %+v, want {2,1}", end)
	}
	want := "abX\nY\nZcd"
	if b.String() != want {
		t.Errorf("String = %q, want %q", b.String(), want)
	}
}

func TestDeleteRangeAcrossLinesReturnsRemoved(t *testing.T) {
	b := New("abX\nY\nZcd")
	removed := b.DeleteRange(Position{0, 2}, Position{2, 1})
	if removed != "X\nY\nZ" {
		t.Fatalf("removed = %q, want %q", removed, "X\nY\nZ")
	}
	if b.String() != "abcd" {
		t.Errorf("String = %q, want %q", b.String(), "abcd")
	}
}

func TestReplaceRangeRoundTrips(t *testing.T) {
	b := New("one two three")
	end := b.ReplaceRange(Position{0, 4}, Position{0, 7}, "TWO")
	if end != (Position{0, 7}) {
		t.Fatalf("end = %+v, want {0,7}", end)
	}
	if b.String() != "one TWO three" {
		t.Errorf("String = %q, want %q", b.String(), "one TWO three")
	}
}

func TestClampConstrainsLineAndColumn(t *testing.T) {
	b := New("ab\ncde")
	cases := []struct{ in, want Position }{
		{Position{-1, 5}, Position{0, 0}},
		{Position{0, 9}, Position{0, 2}},
		{Position{9, 0}, Position{1, 3}},
	}
	for _, c := range cases {
		if got := b.Clamp(c.in); got != c.want {
			t.Errorf("Clamp(%+v) = %+v, want %+v", c.in, got, c.want)
		}
	}
}
