// SPDX-License-Identifier: GPL-2.0-only

package cmdline

import "testing"

func TestScrollbackRingDropsOldest(t *testing.T) {
	sb := NewScrollback(3)
	for _, s := range []string{"a", "b", "c", "d"} {
		sb.Append(s, Info)
	}
	got := sb.Lines()
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (ring bounded)", len(got))
	}
	if got[0].Text != "b" || got[2].Text != "d" {
		t.Errorf("ring = %q, want [b c d]", []string{got[0].Text, got[1].Text, got[2].Text})
	}
}

func TestScrollbackDefaultMaxOnNonPositive(t *testing.T) {
	if NewScrollback(0).max != defaultMaxLines {
		t.Errorf("max = %d, want default %d", NewScrollback(0).max, defaultMaxLines)
	}
}

func TestScrollbackClearKeepsHistory(t *testing.T) {
	sb := NewScrollback(10)
	sb.Append("line", Info)
	sb.RecordCommand("LINE")
	sb.Clear()
	if sb.Len() != 0 {
		t.Errorf("Len = %d, want 0 after Clear", sb.Len())
	}
	if got, ok := sb.LastCommand(); !ok || got != "LINE" {
		t.Errorf("LastCommand = %q,%v, want LINE,true (history survives Clear)", got, ok)
	}
}

func TestScrollbackHistoryCollapsesDuplicates(t *testing.T) {
	sb := NewScrollback(10)
	sb.RecordCommand("LINE")
	sb.RecordCommand("LINE") // immediate repeat collapsed
	sb.RecordCommand("CIRCLE")
	sb.RecordCommand("") // empty ignored
	if got := sb.History(); len(got) != 2 || got[0] != "LINE" || got[1] != "CIRCLE" {
		t.Errorf("History = %v, want [LINE CIRCLE]", got)
	}
}

func TestScrollbackLastCommandEmpty(t *testing.T) {
	if _, ok := NewScrollback(10).LastCommand(); ok {
		t.Error("LastCommand on empty history should be false")
	}
}
