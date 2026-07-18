// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"strconv"
	"strings"
)

// Parameter is a SolidWorks global variable — a user parameter that drives sketch dimensions. It is
// stored as an equation string `"name" = expr` in Contents/Config-0 (the moRelMgr_c relation graph);
// Expression is the right-hand side as written ("20mm", "5", or a formula referencing other
// variables). Number parses Expression when it is a plain numeric literal with an optional unit.
type Parameter struct {
	Name       string
	Expression string
}

// Parameters decodes the part's global variables from Contents/Config-0 (present in both container
// formats). Dimension-driving equations ("D1@Sketch1" = …) are excluded — their left side names a
// dimension, not a parameter (it contains '@').
func (d *Document) Parameters() []Parameter {
	cfg, err := d.Stream("Contents/Config-0")
	if err != nil {
		return nil
	}
	var out []Parameter
	seen := map[string]bool{}
	for _, s := range utf16Strings(cfg) {
		name, expr, ok := parseGlobalVar(s)
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, Parameter{Name: name, Expression: expr})
	}
	return out
}

// parseGlobalVar splits a global-variable equation `"name" = expr` into its name and right-hand
// side. It returns ok=false for any other string, including dimension equations (name holds '@') and
// bare quoted names with no '='.
func parseGlobalVar(s string) (name, expr string, ok bool) {
	if !strings.HasPrefix(s, `"`) {
		return "", "", false
	}
	end := strings.IndexByte(s[1:], '"')
	if end < 0 {
		return "", "", false
	}
	name = s[1 : 1+end]
	rest := strings.TrimSpace(s[2+end:])
	if name == "" || strings.ContainsAny(name, "@") || !strings.HasPrefix(rest, "=") {
		return "", "", false
	}
	expr = strings.TrimSpace(rest[1:])
	if expr == "" {
		return "", "", false
	}
	return name, expr, true
}

// Number parses a parameter whose expression is a numeric literal with an optional unit suffix
// (e.g. "20mm" -> 20, "mm"; "5" -> 5, ""). It returns ok=false when the expression is a formula
// (references another variable or an operator), which the caller keeps as an expression.
func (p Parameter) Number() (value float64, unit string, ok bool) {
	e := p.Expression
	i := 0
	for i < len(e) && (e[i] == '-' || e[i] == '+' || e[i] == '.' || (e[i] >= '0' && e[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0, "", false
	}
	v, err := strconv.ParseFloat(e[:i], 64)
	if err != nil {
		return 0, "", false
	}
	unit = strings.TrimSpace(e[i:])
	if strings.ContainsAny(unit, `"*/+-()`) { // a formula tail, not a unit
		return 0, "", false
	}
	return v, unit, true
}

// utf16Strings extracts the printable-ASCII UTF-16LE strings (>= 2 chars) from b. SolidWorks stores
// names and equation text as MFC CString wide strings, so decoded text is UTF-16LE runs. A run may
// begin at any byte offset (not only even ones), so the scan advances one byte at a time to find a
// run start, then consumes the run two bytes per character.
func utf16Strings(b []byte) []string {
	isChar := func(i int) bool { return i+1 < len(b) && b[i] >= 0x20 && b[i] < 0x7f && b[i+1] == 0 }
	var out []string
	for i := 0; i+1 < len(b); {
		if !isChar(i) {
			i++
			continue
		}
		start := i
		for isChar(i) {
			i += 2
		}
		if (i-start)/2 >= 2 {
			s := make([]byte, 0, (i-start)/2)
			for j := start; j < i; j += 2 {
				s = append(s, b[j])
			}
			out = append(out, string(s))
		}
	}
	return out
}
