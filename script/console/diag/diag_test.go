// SPDX-License-Identifier: GPL-2.0-only

package diag

import (
	"testing"
	"time"
)

// fakeChecker is a named test double recording how many times Check ran and returning a
// canned diagnostic set, so the Analyzer's debounce behaviour is asserted without a real parser.
type fakeChecker struct {
	calls int
	out   []Diagnostic
}

func (f *fakeChecker) Check(string) []Diagnostic {
	f.calls++
	return f.out
}

func TestAnalyzerChecksAfterDebounce(t *testing.T) {
	fc := &fakeChecker{out: []Diagnostic{{Line: 2, Col: 1, Message: "boom"}}}
	a := NewAnalyzer(fc, 500*time.Millisecond)
	t0 := time.Unix(0, 0)

	a.Observe("x =", t0) // change: starts the debounce timer
	a.Observe("x =", t0.Add(200*time.Millisecond))
	if fc.calls != 0 {
		t.Fatalf("checked before debounce elapsed: calls=%d", fc.calls)
	}
	a.Observe("x =", t0.Add(600*time.Millisecond)) // stable past the interval -> check
	if fc.calls != 1 {
		t.Fatalf("calls=%d, want exactly 1 after debounce", fc.calls)
	}
	if got := a.Diagnostics(); len(got) != 1 || got[0].Message != "boom" {
		t.Fatalf("diagnostics = %+v, want the checker's output", got)
	}
}

func TestAnalyzerReChecksOnlyOnceWhileStable(t *testing.T) {
	fc := &fakeChecker{}
	a := NewAnalyzer(fc, 100*time.Millisecond)
	t0 := time.Unix(0, 0)
	a.Observe("a", t0)
	a.Observe("a", t0.Add(150*time.Millisecond)) // first settle -> check
	a.Observe("a", t0.Add(300*time.Millisecond)) // still stable -> no re-check
	if fc.calls != 1 {
		t.Errorf("calls=%d, want 1 (no redundant re-check while text is unchanged)", fc.calls)
	}
}

func TestAnalyzerResetsTimerOnEdit(t *testing.T) {
	fc := &fakeChecker{}
	a := NewAnalyzer(fc, 100*time.Millisecond)
	t0 := time.Unix(0, 0)
	a.Observe("a", t0)
	a.Observe("ab", t0.Add(80*time.Millisecond))  // edit before settling: restart timer
	a.Observe("ab", t0.Add(150*time.Millisecond)) // only 70ms since the edit -> no check yet
	if fc.calls != 0 {
		t.Fatalf("checked too early after a fresh edit: calls=%d", fc.calls)
	}
	a.Observe("ab", t0.Add(200*time.Millisecond)) // 120ms since edit -> check
	if fc.calls != 1 {
		t.Errorf("calls=%d, want 1 once the new text settles", fc.calls)
	}
}
