// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/model/compdef"
)

// partWithParams returns a part definition carrying the given user parameters (name→expr).
func partWithParams(t *testing.T, params map[string]string) *compdef.PartComponentDefinition {
	t.Helper()
	def := compdef.NewPartComponentDefinition()
	for name, expr := range params {
		if _, err := def.Parameters().AddUserParameter(name, expr); err != nil {
			t.Fatalf("add parameter %s=%q: %v", name, expr, err)
		}
	}
	return def
}

// TestCountClosureEvaluatesExpression: a pattern count expression is evaluated through the part's
// parameter engine (so the occurrence count tracks the parameter), a blank expression yields the
// numeric fallback, and a non-positive result clamps to 1 (Oblikovati.API#189).
func TestCountClosureEvaluatesExpression(t *testing.T) {
	t.Parallel()
	part := partWithParams(t, map[string]string{"slots": "6", "poles": "10"})

	cases := []struct {
		expr     string
		fallback int
		want     int
	}{
		{"slots", 4, 6},
		{"poles / 2", 4, 5},
		{"slots + 1", 4, 7},
		{"", 8, 8},          // blank ⇒ fallback
		{"slots - 9", 4, 1}, // non-positive ⇒ clamps to 1
	}
	for _, c := range cases {
		fn, err := countClosure(part, c.expr, "test: count", c.fallback)
		if err != nil {
			t.Errorf("countClosure(%q): %v", c.expr, err)
			continue
		}
		if got := fn(); got != c.want {
			t.Errorf("countClosure(%q)() = %d, want %d", c.expr, got, c.want)
		}
	}
}

// TestCountClosureTracksParameterChange: the closure re-reads the parameter on each call, so a
// later parameter edit changes the occurrence count without rebuilding the pattern (#189).
func TestCountClosureTracksParameterChange(t *testing.T) {
	t.Parallel()
	part := partWithParams(t, map[string]string{"slots": "6"})
	fn, err := countClosure(part, "slots", "test: count", 4)
	if err != nil {
		t.Fatalf("countClosure: %v", err)
	}
	if got := fn(); got != 6 {
		t.Fatalf("initial count = %d, want 6", got)
	}
	p, ok := part.Parameters().ByName("slots")
	if !ok {
		t.Fatal("parameter slots not found")
	}
	if err := part.Parameters().SetExpression(p.ID(), "9"); err != nil {
		t.Fatalf("set slots=9: %v", err)
	}
	if got := fn(); got != 9 {
		t.Errorf("after slots=9 count = %d, want 9 (closure must re-read the parameter)", got)
	}
}
