// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
	"unicode/utf16"
)

// cursor is a little-endian read cursor over a byte slice. Out-of-range reads
// yield zero values and set bad, so callers can decode defensively without panics.
type cursor struct {
	b   []byte
	i   int
	bad bool
}

func newCursor(b []byte) *cursor { return &cursor{b: b} }

func (c *cursor) has(n int) bool { return c.i+n <= len(c.b) && !c.bad }

func (c *cursor) u8() uint8 {
	if !c.has(1) {
		c.bad = true
		return 0
	}
	v := c.b[c.i]
	c.i++
	return v
}

func (c *cursor) u16() uint16 {
	if !c.has(2) {
		c.bad = true
		return 0
	}
	v := binary.LittleEndian.Uint16(c.b[c.i:])
	c.i += 2
	return v
}

func (c *cursor) u32() uint32 {
	if !c.has(4) {
		c.bad = true
		return 0
	}
	v := binary.LittleEndian.Uint32(c.b[c.i:])
	c.i += 4
	return v
}

func (c *cursor) i32() int32 { return int32(c.u32()) }

func (c *cursor) f64() float64 {
	if !c.has(8) {
		c.bad = true
		return 0
	}
	v := math.Float64frombits(binary.LittleEndian.Uint64(c.b[c.i:]))
	c.i += 8
	return v
}

func (c *cursor) skip(n int) {
	if !c.has(n) {
		c.bad = true
		return
	}
	c.i += n
}

// text8 reads a u32-length-prefixed 8-bit (latin1) string.
func (c *cursor) text8() string {
	n := int(c.u32())
	if !c.has(n) {
		c.bad = true
		return ""
	}
	s := string(c.b[c.i : c.i+n])
	c.i += n
	return trimNul(s)
}

// text16 reads a u32-length-prefixed UTF-16LE string (length is in code units).
func (c *cursor) text16() string {
	n := int(c.u32())
	if !c.has(2 * n) {
		c.bad = true
		return ""
	}
	u := make([]uint16, n)
	for k := 0; k < n; k++ {
		u[k] = binary.LittleEndian.Uint16(c.b[c.i+2*k:])
	}
	c.i += 2 * n
	return trimNul(string(utf16.Decode(u)))
}

func trimNul(s string) string {
	for len(s) > 0 && (s[len(s)-1] == 0 || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}
