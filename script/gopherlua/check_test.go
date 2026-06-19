// SPDX-License-Identifier: GPL-2.0-only

package gopherlua

import "testing"

func TestSyntaxCheckerCleanSource(t *testing.T) {
	if d := (SyntaxChecker{}).Check("local x = 1\nprint(x)\n"); d != nil {
		t.Errorf("valid source produced diagnostics: %+v", d)
	}
}

func TestSyntaxCheckerReportsErrorWithLine(t *testing.T) {
	// `then` without a condition's `end` — a syntax error on line 2 (0-based line 1).
	src := "local x = 1\nif x then\nprint(x)\n"
	d := (SyntaxChecker{}).Check(src)
	if len(d) != 1 {
		t.Fatalf("diagnostics = %+v, want exactly one", d)
	}
	if d[0].Line < 0 || d[0].Line > 3 { // an EOF error clamps onto the last line index
		t.Errorf("diagnostic line = %d, want it clamped within the source", d[0].Line)
	}
	if d[0].Message == "" {
		t.Error("diagnostic message is empty")
	}
}

func TestSyntaxCheckerBadTokenOnLine(t *testing.T) {
	// An illegal `=` where a statement is expected, on the second line.
	d := (SyntaxChecker{}).Check("ok = 1\n= bad\n")
	if len(d) != 1 {
		t.Fatalf("diagnostics = %+v, want one", d)
	}
	if d[0].Line != 1 {
		t.Errorf("line = %d, want 1 (the offending second line)", d[0].Line)
	}
}
