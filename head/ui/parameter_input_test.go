//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestFormatParamValue locks the canonical text ParameterInput shows when not being edited: the
// value to the document precision with the unit appended IN the field, a bare number when unitless,
// and a 3-decimal fallback for an out-of-range precision (#1519).
func TestFormatParamValue(t *testing.T) {
	cases := []struct {
		value float64
		unit  string
		prec  int
		want  string
	}{
		{10, "mm", 3, "10.000 mm"},
		{2.5, "deg", 2, "2.50 deg"},
		{7, "in", 0, "7 in"},
		{0.5, "", 4, "0.5000"},
		{10, "mm", -1, "10.000 mm"}, // out of range → default 3
		{10, "mm", 99, "10.000 mm"}, // out of range → default 3
	}
	for _, c := range cases {
		if got := formatParamValue(c.value, c.unit, c.prec); got != c.want {
			t.Errorf("formatParamValue(%g, %q, %d) = %q, want %q", c.value, c.unit, c.prec, got, c.want)
		}
	}
}

// TestParamSeedText locks the edit-dialog seed: the value with its unit appended ("10 mm") so the
// dimension is part of the editable expression, and a bare number when there is no unit.
func TestParamSeedText(t *testing.T) {
	if got := paramSeedText(10, "mm"); got != "10 mm" {
		t.Errorf("paramSeedText(10,\"mm\") = %q, want \"10 mm\"", got)
	}
	if got := paramSeedText(3, ""); got != "3" {
		t.Errorf("paramSeedText(3,\"\") = %q, want \"3\"", got)
	}
}

// bareInputFloatAllowlist lists the only head/ui files that may call native.InputFloat directly,
// because their fields are NOT document-unit feature/tool parameters (#1519). Each entry says why.
var bareInputFloatAllowlist = map[string]string{
	"appearance_editor.go":         "PBR scalars (Metallic/Roughness/Opacity) are dimensionless [0,1] ratios, not a document dimension",
	"openpbr_appearance_editor.go": "OpenPBR Surface scalars (weight/roughness/IOR/anisotropy/color channels/thickness/luminance) are physical-material properties in fixed units or [0,1] ratios, not document-unit feature dimensions — mirrors appearance_editor.go's legacy PBR-scalar exemption",
	"parameters_dialogs.go":        "the parameter table's tolerance editor enters deviations in database units by design",
	"preferences_window.go":        "an application preference (grid spacing) with a fixed unit, not a per-document parameter",
}

// unitTextAllowlist lists files that may render a unit name as a standalone Text — a table column,
// not a label painted beside an input field.
var unitTextAllowlist = map[string]bool{
	"parameters_row.go": true, // the Parameters table's dedicated Unit column
}

// TestParameterInputIsEnforced is the #1519 guard: a tool that takes a dimensioned value MUST use the
// ParameterInput component (parameterFloatRow / parameterField), which renders the unit INSIDE the
// field and accepts formulas — never a bare native.InputFloat with the unit painted beside it as a
// label, and never a unit name dropped as a sibling Text. It scans the head/ui sources so a new
// dialog cannot silently reintroduce the antipattern; a genuinely-exempt surface must be added to an
// allowlist above with a justification, which makes the exception a reviewed decision.
func TestParameterInputIsEnforced(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read head/ui dir: %v", err)
	}
	bareInput := regexp.MustCompile(`native\.InputFloat\(`)
	unitLabel := regexp.MustCompile(`native\.Text\([^)]*UnitName`)
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		scanned++
		for i, line := range strings.Split(string(src), "\n") {
			if bareInput.MatchString(line) {
				if _, ok := bareInputFloatAllowlist[name]; !ok {
					t.Errorf("%s:%d uses bare native.InputFloat — a parameter field must use ParameterInput "+
						"(parameterFloatRow) so the document unit shows INSIDE the field, not as a label (#1519): %s",
						name, i+1, strings.TrimSpace(line))
				}
			}
			if unitLabel.MatchString(line) && !unitTextAllowlist[name] {
				t.Errorf("%s:%d paints a unit name as a label beside a field — embed it in the ParameterInput "+
					"format (parameterDisplayFormat) instead (#1519): %s", name, i+1, strings.TrimSpace(line))
			}
		}
	}
	if scanned < 20 {
		t.Fatalf("guard scanned only %d head/ui sources; the working directory is wrong", scanned)
	}
}
