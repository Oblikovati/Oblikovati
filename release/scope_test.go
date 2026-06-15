// SPDX-License-Identifier: GPL-2.0-only

package release

import "testing"

func TestClassifyOne(t *testing.T) {
	cases := []struct {
		msg  string
		want Scope
	}{
		{"feat(head): version in window title", ScopeFeature},
		{"fix(kernel): tessellation winding", ScopePatch},
		{"perf: faster boolean", ScopePatch},
		{"docs: tidy", ScopeNone},
		{"chore: deps", ScopeNone},
		{"ci: pin api", ScopeNone},
		{"refactor: split", ScopeNone},
		{"feat!: drop legacy flag", ScopeBreaking},
		{"feat(app)!: rename", ScopeBreaking},
		{"fix: x\n\nBREAKING CHANGE: removed Y", ScopeBreaking},
		{"Add a thing without a type", ScopePatch},
	}
	for _, c := range cases {
		if got := classifyOne(c.msg); got != c.want {
			t.Errorf("classifyOne(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

func TestClassifyStrongest(t *testing.T) {
	if got := Classify([]string{"docs: a", "fix: b", "feat: c"}); got != ScopeFeature {
		t.Errorf("Classify = %v, want ScopeFeature", got)
	}
	if got := Classify([]string{"docs: a", "ci: b"}); got != ScopeNone {
		t.Errorf("Classify(silent) = %v, want ScopeNone", got)
	}
	if got := Classify(nil); got != ScopeNone {
		t.Errorf("Classify(nil) = %v, want ScopeNone", got)
	}
}

func TestNextMinorPatch(t *testing.T) {
	cases := []struct {
		minor, patch     int
		scope            Scope
		wantMin, wantPat int
	}{
		{1, 4, ScopeNone, 1, 4},
		{1, 4, ScopePatch, 1, 5},
		{1, 4, ScopeFeature, 2, 0},
		{1, 4, ScopeBreaking, 2, 0}, // manual major => breaking is a minor bump
		{0, 0, ScopeFeature, 1, 0},
		{0, 0, ScopePatch, 0, 1},
	}
	for _, c := range cases {
		mi, pa := NextMinorPatch(c.minor, c.patch, c.scope)
		if mi != c.wantMin || pa != c.wantPat {
			t.Errorf("NextMinorPatch(%d,%d,%v) = %d,%d; want %d,%d",
				c.minor, c.patch, c.scope, mi, pa, c.wantMin, c.wantPat)
		}
	}
}
