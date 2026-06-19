// SPDX-License-Identifier: GPL-2.0-only

package editor

import "testing"

func TestFindReturnsAllOccurrences(t *testing.T) {
	m := New("foo bar foo\nfoofoo")
	got := m.Find("foo")
	if len(got) != 4 {
		t.Fatalf("matches = %d, want 4 (incl. two adjacent on line 2)", len(got))
	}
	if got[0].Start != pos(0, 0) || got[0].End != pos(0, 3) {
		t.Errorf("first match = %+v, want {0,0}-{0,3}", got[0])
	}
	// Non-overlapping: the two on line 2 start at col 0 and 3.
	if got[2].Start != pos(1, 0) || got[3].Start != pos(1, 3) {
		t.Errorf("line-2 matches = %+v / %+v, want cols 0 and 3", got[2].Start, got[3].Start)
	}
}

func TestFindEmptyQuery(t *testing.T) {
	if got := New("anything").Find(""); got != nil {
		t.Errorf("empty query matched %v, want nil", got)
	}
}

func TestSelectMatchSelectsSpan(t *testing.T) {
	m := New("alpha beta")
	matches := m.Find("beta")
	m.SelectMatch(matches[0])
	if m.SelectedText() != "beta" {
		t.Errorf("selected %q, want %q", m.SelectedText(), "beta")
	}
}

func TestReplaceAllCountsAndUndoesAsOne(t *testing.T) {
	m := New("x = x + x")
	n := m.ReplaceAll("x", "y")
	if n != 3 || m.Text() != "y = y + y" {
		t.Fatalf("ReplaceAll = %d, text %q, want 3 and %q", n, m.Text(), "y = y + y")
	}
	m.Undo()
	if m.Text() != "x = x + x" {
		t.Errorf("after undo = %q, want one-step revert to original", m.Text())
	}
}

func TestReplaceAllNoMatchLeavesHistory(t *testing.T) {
	m := New("abc")
	if n := m.ReplaceAll("z", "q"); n != 0 {
		t.Fatalf("ReplaceAll no-match = %d, want 0", n)
	}
	if m.CanUndo() {
		t.Error("a no-op ReplaceAll must not push an undo step")
	}
}
