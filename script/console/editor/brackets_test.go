// SPDX-License-Identifier: GPL-2.0-only

package editor

import "testing"

func TestMatchingBracketForwardWithNesting(t *testing.T) {
	m := New("f((a), b)")
	m.SetCaret(pos(0, 1), false) // on the outer '('
	a, b, ok := m.MatchingBracket()
	if !ok || a != pos(0, 1) || b != pos(0, 8) {
		t.Fatalf("match = (%+v,%+v,%v), want outer ( at {0,1} -> ) at {0,8}", a, b, ok)
	}
}

func TestMatchingBracketBackwardFromCloser(t *testing.T) {
	m := New("foo(bar)")
	m.SetCaret(pos(0, 8), false) // caret just after the ')', matches the bracket to its left
	a, b, ok := m.MatchingBracket()
	if !ok || a != pos(0, 7) || b != pos(0, 3) {
		t.Fatalf("match = (%+v,%+v,%v), want ) at {0,7} -> ( at {0,3}", a, b, ok)
	}
}

func TestMatchingBracketAcrossLines(t *testing.T) {
	m := New("function f()\n  g({\n    x = 1\n  })\nend")
	m.SetCaret(pos(1, 4), false) // on the '{' of g({
	a, b, ok := m.MatchingBracket()
	if !ok || a != pos(1, 4) || b != pos(3, 2) {
		t.Fatalf("multiline match = (%+v,%+v,%v), want { {1,4} -> } {3,2}", a, b, ok)
	}
}

func TestNoBracketAtCaret(t *testing.T) {
	m := New("x = 1")
	m.SetCaret(pos(0, 2), false)
	if _, _, ok := m.MatchingBracket(); ok {
		t.Error("expected no match away from any bracket")
	}
}

func TestUnbalancedBracketNoMatch(t *testing.T) {
	m := New("f(a")
	m.SetCaret(pos(0, 1), false) // '(' with no closer
	if _, _, ok := m.MatchingBracket(); ok {
		t.Error("unbalanced '(' should not match")
	}
}
