// SPDX-License-Identifier: GPL-2.0-only

package dxf

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// pair is one DXF group-code/value record: an integer code naming the datum's role
// (10 = X, 40 = a real, 0 = an entity-type marker, …) and its value as raw text. The typed
// accessors (float/integer/handle) parse on demand and tolerate the leading/trailing
// padding real-world writers add (some right-justify integers, AutoCAD pads codes).
type pair struct {
	code  int
	value string
}

// scanPairs splits a DXF byte stream into its group-code/value pairs. Lines come two at a
// time (code then value); CR is stripped so both LF and CRLF files parse. A code line that
// is not an integer is a malformed file and errors with its line number and content.
func scanPairs(data []byte) ([]pair, error) {
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024) // DXF lines are short, but files are large
	var lines []string
	for sc.Scan() {
		lines = append(lines, strings.TrimRight(sc.Text(), "\r"))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("dxf: read: %w", err)
	}
	pairs := make([]pair, 0, len(lines)/2)
	for i := 0; i+1 < len(lines); i += 2 {
		code, err := strconv.Atoi(strings.TrimSpace(lines[i]))
		if err != nil {
			return nil, fmt.Errorf("dxf: line %d: group code %q is not an integer", i+1, lines[i])
		}
		pairs = append(pairs, pair{code: code, value: lines[i+1]})
	}
	return pairs, nil
}

// float parses the value as a real, trimming any padding. The error names the code and the
// offending text.
func (p pair) float() (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(p.value), 64)
	if err != nil {
		return 0, fmt.Errorf("dxf: code %d: %q is not a real", p.code, p.value)
	}
	return v, nil
}

// integer parses the value as an int, trimming any padding (libredwg right-justifies ints).
func (p pair) integer() (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(p.value))
	if err != nil {
		return 0, fmt.Errorf("dxf: code %d: %q is not an integer", p.code, p.value)
	}
	return v, nil
}

// handle parses a DXF hex handle (code 5/330/…); a missing or malformed handle is 0, since
// the handle is identity metadata the geometry does not depend on.
func (p pair) handle() uint64 {
	v, err := strconv.ParseUint(strings.TrimSpace(p.value), 16, 64)
	if err != nil {
		return 0
	}
	return v
}

// text returns the value with surrounding whitespace trimmed — DXF names (layers, block
// names, entity types) carry no significant padding.
func (p pair) text() string { return strings.TrimSpace(p.value) }
