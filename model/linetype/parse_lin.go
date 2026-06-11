// SPDX-License-Identifier: GPL-2.0-only

package linetype

import (
	"fmt"
	"strconv"
	"strings"
)

// The .lin definition format: a header line `*NAME,description` followed by one
// pattern line `A,len1,len2,...` per definition. Positive lengths are dashes,
// negative are gaps, zero is a dot; `;` starts a comment line. Embedded text and
// shape elements (`["…",STYLE,…]` / `[SHAPE,…]`) are a drawing-annotation feature
// this kernel does not render — definitions using them are rejected by name so the
// failure is visible instead of silently dropping marks.

// ParseLIN parses every definition in a .lin file's contents.
//
//	defs, err := linetype.ParseLIN(src)
func ParseLIN(src string) ([]Definition, error) {
	defs := []Definition{}
	var pending *Definition
	for n, raw := range strings.Split(src, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		d, err := parseLINLine(line, n+1, &pending)
		if err != nil {
			return nil, err
		}
		if d != nil {
			defs = append(defs, *d)
		}
	}
	return defs, nil
}

// parseLINLine consumes one non-comment line: a `*` header opens a pending
// definition, a pattern line completes it (returned non-nil).
func parseLINLine(line string, n int, pending **Definition) (*Definition, error) {
	if strings.HasPrefix(line, "*") {
		name, desc, _ := strings.Cut(strings.TrimPrefix(line, "*"), ",")
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("linetype: line %d: header %q has an empty name (want *NAME,description)", n, line)
		}
		*pending = &Definition{Name: strings.TrimSpace(name), Description: strings.TrimSpace(desc)}
		return nil, nil
	}
	if *pending == nil {
		return nil, fmt.Errorf("linetype: line %d: pattern %q before any *NAME header", n, line)
	}
	d := *pending
	*pending = nil
	pattern, err := parseLINPattern(d.Name, line, n)
	if err != nil {
		return nil, err
	}
	d.Pattern = pattern
	return d, nil
}

// parseLINPattern parses the `A,len1,len2,...` line of one definition.
func parseLINPattern(name, line string, n int) ([]float64, error) {
	if strings.Contains(line, "[") {
		return nil, fmt.Errorf("linetype: line %d: definition %q embeds a text/shape element, which is not supported", n, name)
	}
	fields := strings.Split(line, ",")
	if strings.ToUpper(strings.TrimSpace(fields[0])) != "A" {
		return nil, fmt.Errorf("linetype: line %d: definition %q pattern starts with %q, want alignment code \"A\"", n, name, fields[0])
	}
	pattern := make([]float64, 0, len(fields)-1)
	for _, f := range fields[1:] {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return nil, fmt.Errorf("linetype: line %d: definition %q has a non-numeric element %q", n, name, f)
		}
		pattern = append(pattern, v)
	}
	if len(pattern) == 0 {
		return nil, fmt.Errorf("linetype: line %d: definition %q has an empty pattern", n, name)
	}
	return pattern, nil
}

// Find returns the definition with the given name (case-insensitive, the .lin
// convention of uppercase names notwithstanding).
func Find(defs []Definition, name string) (Definition, bool) {
	for _, d := range defs {
		if strings.EqualFold(d.Name, name) {
			return d, true
		}
	}
	return Definition{}, false
}
