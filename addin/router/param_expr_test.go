// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// partWith returns a part definition carrying the given user parameters (name→expr).
func partWith(t *testing.T, params map[string]string) *compdef.PartComponentDefinition {
	t.Helper()
	def := compdef.NewPartComponentDefinition()
	for name, expr := range params {
		if _, err := def.Parameters().AddUserParameter(name, expr); err != nil {
			t.Fatalf("add parameter %s=%q: %v", name, expr, err)
		}
	}
	return def
}

// TestResolveQuantityParameterExpression is the headline behavioural check: a
// unit-string field that is a parameter name or a formula over parameters resolves
// against the part's table, while plain literals (and bare numbers) still parse.
func TestResolveQuantityParameterExpression(t *testing.T) {
	t.Parallel()
	part := partWith(t, map[string]string{"bore_r": "10 mm", "slot_depth": "5 mm"})
	cases := []struct {
		src     string
		wantCm  float64
		comment string
	}{
		{"bore_r", 1.0, "bare parameter name"},
		{"bore_r + slot_depth", 1.5, "formula over parameters"},
		{"bore_r * 2", 2.0, "parameter arithmetic"},
		{"23.5 mm", 2.35, "literal unit value still works"},
		{"5", 0.5, "bare number falls back to the default length unit (mm)"},
	}
	for _, c := range cases {
		q, err := resolveQuantity(part, c.src, param.Length)
		if err != nil {
			t.Errorf("resolveQuantity(%q) [%s]: %v", c.src, c.comment, err)
			continue
		}
		if math.Abs(q.Value-c.wantCm) > 1e-9 {
			t.Errorf("resolveQuantity(%q) [%s] = %g cm, want %g", c.src, c.comment, q.Value, c.wantCm)
		}
	}
}

// TestResolveQuantityWrongDimensionFallsBack: an expression evaluating to a
// different dimension than requested falls back to the literal parser (which then
// reports the real error) rather than silently returning the wrong-unit value.
func TestResolveQuantityWrongDimensionFallsBack(t *testing.T) {
	t.Parallel()
	part := partWith(t, map[string]string{"sweep": "30 deg"})
	if _, err := resolveQuantity(part, "sweep", param.Length); err == nil {
		t.Error("an angle parameter used as a length should error, not resolve")
	}
}

// TestRouterUnitStringsGoThroughResolver enforces the solution-wide fix: NO router
// handler may parse a unit string with part.Units().Parse directly — every such field
// must go through resolveQuantity so parameter expressions work everywhere
// (Oblikovati.API#187). The single sanctioned call is the resolver's own literal
// fallback in param_expr.go.
func TestRouterUnitStringsGoThroughResolver(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read router dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "param_expr.go" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(b), "Units().Parse(") {
			t.Errorf("%s calls Units().Parse directly — route unit-string fields through resolveQuantity so parameter expressions work (Oblikovati.API#187)", name)
		}
	}
}
