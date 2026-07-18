// SPDX-License-Identifier: GPL-2.0-only

package sldprt

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Dimension is a decoded sketch dimension: its name (SolidWorks' "D1", "D2", …) and evaluated value
// in metres (a length) — the driving/driven value the geometry is dimensioned to. The dimension
// KIND (distance vs diameter vs radius vs angle) and the entities it measures are not yet decoded;
// the value lives in a moLengthParameter object keyed by the name.
type Dimension struct {
	Name  string
	Value float64
}

// wideMarker precedes an MFC wide string: ff fe ff then a 1-byte length. A dimension's name follows
// as a "D"-prefixed numeric ("D1", "D12"); the preceding name-object tag varies by container (04 80
// in the SolidWorks-2026 store, 42 80 in older CFBF), so it is not part of the anchor. (A raw-byte
// scan is used rather than regexp, whose \xff matches codepoint U+00FF, not the byte 0xFF.)
var wideMarker = []byte{0xff, 0xfe, 0xff}

// dimensionsIn decodes the sketch dimensions in a region. Each dimension's value follows its name in
// the moLengthParameter object; a fixed 2.0 factor precedes the value, so the first metric float64
// after the name that is not a plain 1.0/2.0 factor is taken as the value (in metres).
func dimensionsIn(region []byte) []Dimension {
	var out []Dimension
	seen := map[string]bool{}
	for i := 0; i < len(region); {
		m := bytes.Index(region[i:], wideMarker)
		if m < 0 {
			break
		}
		at := i + m
		i = at + 1
		name, end, ok := dimNameAt(region, at)
		if !ok || seen[name] {
			continue
		}
		if v, ok := dimValueAfter(region, end); ok {
			seen[name] = true
			out = append(out, Dimension{Name: name, Value: v})
		}
	}
	return out
}

// dimNameAt reads a "D<digits>" dimension name at a wideMarker and returns it with the offset just
// past it. ok is false when the wide string there is not a dimension name.
func dimNameAt(region []byte, at int) (name string, end int, ok bool) {
	if at+4 > len(region) {
		return "", 0, false
	}
	n := int(region[at+3])
	start, stop := at+4, at+4+2*n
	if n < 2 || n > 16 || stop > len(region) {
		return "", 0, false
	}
	name = decodeWide(region[start:stop])
	if len(name) < 2 || name[0] != 'D' {
		return "", 0, false
	}
	for _, c := range name[1:] {
		if c < '0' || c > '9' {
			return "", 0, false
		}
	}
	return name, stop, true
}

// dimValueAfter returns a dimension's value float64 (metres) from the moLengthParameter that follows
// its name. The value is preceded by a fixed 2.0 factor (the diameter/display factor); anchoring on
// that factor skips an annotation-position coordinate that can otherwise sit between the name and
// the value. It scans from the factor for the first finite metric magnitude (0.01 mm … 10 m).
func dimValueAfter(region []byte, from int) (float64, bool) {
	factor := []byte{0, 0, 0, 0, 0, 0, 0, 0x40} // 2.0 little-endian
	f := indexWithin(region, factor, from, 40)
	if f < 0 {
		return 0, false
	}
	end := f + 8 + 48
	if end > len(region)-8 {
		end = len(region) - 8
	}
	for o := f + 8; o <= end; o++ {
		v := math.Float64frombits(binary.LittleEndian.Uint64(region[o:]))
		if !math.IsNaN(v) && math.Abs(v) > 1e-5 && math.Abs(v) < 10 && v != 1.0 && v != 2.0 {
			return v, true
		}
	}
	return 0, false
}

// indexWithin finds sep in region[from:from+window], returning its absolute offset or -1.
func indexWithin(region, sep []byte, from, window int) int {
	end := from + window
	if end > len(region) {
		end = len(region)
	}
	if from >= end {
		return -1
	}
	i := bytes.Index(region[from:end], sep)
	if i < 0 {
		return -1
	}
	return from + i
}

// decodeWide decodes a UTF-16LE byte run (ASCII subset) to a string.
func decodeWide(b []byte) string {
	if len(b)%2 != 0 {
		return ""
	}
	s := make([]byte, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		if b[i+1] != 0 || b[i] < 0x20 || b[i] >= 0x7f {
			return ""
		}
		s = append(s, b[i])
	}
	return string(s)
}
