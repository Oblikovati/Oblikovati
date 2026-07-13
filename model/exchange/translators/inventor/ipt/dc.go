// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
)

// PmDCSegment (Document Content) holds the parametric model as a node graph. This file
// decodes it into the pieces a translator maps onto Oblikovati. Reverse-engineered for
// Inventor 2027 (the InventorLoader/older node-UUID scheme is absent in this version).
const (
	dcNodeTag = 0x80000003 // a graph node: tag, then a u32 node id
	dcMarker  = 0x00000C20 // recurring marker preceding a named node's payload
	nullRef   = 0xFFFFFFFF
)

// Parameter is a user parameter: its name and value in database units (centimetres
// for a length; the unit kind is not yet decoded — length is assumed for now).
type Parameter struct {
	Name  string
	Value float64
}

// DecodeParameters extracts the user parameters from a decompressed PmDCSegment.
// A user parameter is a nameful node whose RAW name is a clean single-byte-ASCII
// identifier (auto model parameters "d0" -> d\x00 and UTF-16-named nodes carry null
// bytes and are excluded — they are recreated when features replay). The value is the
// first non-zero float64 after the unit/type id fields.
func DecodeParameters(seg []byte) []Parameter {
	var out []Parameter
	forEachNamefulNode(seg, func(rawName []byte, valueStart int) {
		if !isIdentifier(string(rawName)) {
			return
		}
		if v, ok := firstNonZeroDouble(seg, valueStart, 40); ok {
			out = append(out, Parameter{Name: string(rawName), Value: v})
		}
	})
	return out
}

// forEachNamefulNode calls fn(rawName, valueStart) for each named graph node:
// dcNodeTag, u32 id, nullRef, a schema marker(+12), a length-prefixed name (namelen>0 at
// +16). valueStart is the byte offset just past the name (where the node's payload begins).
// The +12 marker is NOT keyed on — it is 0x0C20 in API-authored corpus parts but varies in
// real interactively-modelled parts (0x0A96, 0x0B5C, …); the namelen range guards against
// false matches (point nodes carry 0 there, enum nodes a kind/value pair > 64).
func forEachNamefulNode(seg []byte, fn func(rawName []byte, valueStart int)) {
	for i := 0; i+20 <= len(seg); i++ {
		if binary.LittleEndian.Uint32(seg[i:]) != dcNodeTag ||
			binary.LittleEndian.Uint32(seg[i+8:]) != nullRef {
			continue
		}
		n := int(binary.LittleEndian.Uint32(seg[i+16:]))
		if n < 1 || n > 64 || i+20+n > len(seg) {
			continue
		}
		fn(seg[i+20:i+20+n], i+20+n)
	}
}

// firstNonZeroDouble scans up to window bytes from offset for the first finite,
// non-zero float64 (the parameter's value follows its unit/type id fields).
func firstNonZeroDouble(seg []byte, offset, window int) (float64, bool) {
	v, _, ok := firstNonZeroDoubleAt(seg, offset, window)
	return v, ok
}

// firstNonZeroDoubleAt is firstNonZeroDouble but also returns the byte offset where the value was
// found — callers that need a fixed-offset field after the value (e.g. the parameter's unit-type
// id, 20 bytes past it) use `at` as the anchor.
func firstNonZeroDoubleAt(seg []byte, offset, window int) (value float64, at int, ok bool) {
	end := offset + window
	if end > len(seg)-8 {
		end = len(seg) - 8
	}
	for o := offset; o <= end; o++ {
		c := newCursor(seg)
		c.i = o
		v := c.f64()
		if v == v && absf(v) > 1e-9 && absf(v) < 1e6 {
			return v, o, true // skip zeros and denormal padding; real values are O(0.001..1000) cm
		}
	}
	return 0, 0, false
}

func isIdentifier(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		letter := c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		digit := c >= '0' && c <= '9'
		if i == 0 && !letter {
			return false
		}
		if !letter && !digit {
			return false
		}
	}
	return true
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
