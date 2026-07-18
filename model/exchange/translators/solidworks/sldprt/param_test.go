// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import "testing"

// TestParameters decodes the three global variables of a generated part exactly: width = 20mm,
// height = 12mm, count = 5 (validated against the SolidWorks 2026 EquationMgr oracle).
func TestParameters(t *testing.T) {
	d, err := Open(readTestdata(t, "params_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := map[string]string{}
	for _, p := range d.Parameters() {
		got[p.Name] = p.Expression
	}
	want := map[string]string{"width": "20mm", "height": "12mm", "count": "5"}
	if len(got) != len(want) {
		t.Fatalf("got %d parameters %v, want %v", len(got), got, want)
	}
	for name, expr := range want {
		if got[name] != expr {
			t.Errorf("%s = %q, want %q", name, got[name], expr)
		}
	}
}

// TestNoParameters verifies a part with no equations yields no parameters (no false positives from
// other quoted strings in Config-0).
func TestNoParameters(t *testing.T) {
	d, err := Open(readTestdata(t, "box10_fmtb.sldprt"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if p := d.Parameters(); len(p) != 0 {
		t.Errorf("got %d parameters, want 0: %+v", len(p), p)
	}
}

func TestParseGlobalVar(t *testing.T) {
	cases := []struct {
		in         string
		name, expr string
		ok         bool
	}{
		{`"width" = 20mm`, "width", "20mm", true},
		{`"count" = 5`, "count", "5", true},
		{`"total" = "width" * 2`, "total", `"width" * 2`, true},
		{`"D1@Sketch1" = "width"`, "", "", false}, // dimension equation, not a parameter
		{`"width"`, "", "", false},                // bare name, no '='
		{`ANSI31 (Iron BrickStone)`, "", "", false},
		{`"" = 5`, "", "", false},
	}
	for _, c := range cases {
		name, expr, ok := parseGlobalVar(c.in)
		if ok != c.ok || name != c.name || expr != c.expr {
			t.Errorf("parseGlobalVar(%q) = (%q,%q,%v), want (%q,%q,%v)", c.in, name, expr, ok, c.name, c.expr, c.ok)
		}
	}
}

func TestParameterNumber(t *testing.T) {
	cases := []struct {
		expr  string
		value float64
		unit  string
		ok    bool
	}{
		{"20mm", 20, "mm", true},
		{"12.5mm", 12.5, "mm", true},
		{"5", 5, "", true},
		{"-3deg", -3, "deg", true},
		{`"width" * 2`, 0, "", false},
		{"1in + 2mm", 0, "", false},
	}
	for _, c := range cases {
		v, u, ok := Parameter{Expression: c.expr}.Number()
		if ok != c.ok || v != c.value || u != c.unit {
			t.Errorf("Number(%q) = (%g,%q,%v), want (%g,%q,%v)", c.expr, v, u, ok, c.value, c.unit, c.ok)
		}
	}
}
