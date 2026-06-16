// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// decompressR2004 expands one R2004+ compressed region into exactly decompSize
// bytes. The format is the ODA LZ77 variant (Open Design Specification §4.7): a
// leading literal run, then a sequence of opcodes that each copy a back-reference
// from the already-decompressed output and then some literal bytes, until the
// 0x11 terminator.
//
// A notable property of this LZ77: the copy length may exceed the back-offset, so
// a back-reference can read bytes it is itself producing (run-length style); the
// byte-at-a-time copy below is therefore required, not a single memmove.
//
// Example:
//
//	page, err := decompressR2004(comp, 0x67e0) // page-map page
func decompressR2004(src []byte, decompSize int) ([]byte, error) {
	d := &lz77{src: src, dst: make([]byte, 0, decompSize)}
	opcode, isOp := d.leadingLiterals()
	if !isOp {
		opcode = d.next()
	}
	for d.err == nil {
		if opcode == 0x11 { // terminator
			break
		}
		compBytes, compOffset, litCount := d.decodeOpcode(opcode)
		d.copyBackref(compBytes, compOffset)
		next, more := d.copyTrailingLiterals(litCount)
		if more { // a literal-length byte with a set high nibble IS the next opcode
			opcode = next
		} else {
			opcode = d.next()
		}
	}
	if d.err != nil {
		return nil, d.err
	}
	if len(d.dst) != decompSize {
		return nil, fmt.Errorf("dwg lz77: produced %d bytes, expected %d", len(d.dst), decompSize)
	}
	return d.dst, nil
}

// lz77 holds the decompression cursor: input src/pos and the growing output dst.
type lz77 struct {
	src []byte
	pos int
	dst []byte
	err error
}

func (d *lz77) fail(format string, args ...any) {
	if d.err == nil {
		d.err = fmt.Errorf("dwg lz77 at in=%d out=%d: %s", d.pos, len(d.dst), fmt.Sprintf(format, args...))
	}
}

func (d *lz77) next() byte {
	if d.err != nil {
		return 0
	}
	if d.pos >= len(d.src) {
		d.fail("read past end of %d compressed bytes", len(d.src))
		return 0
	}
	b := d.src[d.pos]
	d.pos++
	return b
}

// leadingLiterals copies the section's opening literal run. It returns (opcode,
// true) when the literal length encoded a set high nibble — meaning zero leading
// literals and that byte is actually the first opcode.
func (d *lz77) leadingLiterals() (byte, bool) {
	n, opcode, isOp := d.literalLength()
	if isOp {
		return opcode, true
	}
	d.appendLiterals(n)
	return 0, false
}

// copyTrailingLiterals copies litCount literal bytes after a back-reference. When
// litCount is 0 the count is re-read as a Literal Length, which may instead signal
// the next opcode (returned as (opcode, true)).
func (d *lz77) copyTrailingLiterals(litCount int) (byte, bool) {
	if litCount == 0 {
		n, opcode, isOp := d.literalLength()
		if isOp {
			return opcode, true
		}
		litCount = n
	}
	d.appendLiterals(litCount)
	return 0, false
}

// appendLiterals copies n bytes verbatim from the input to the output.
func (d *lz77) appendLiterals(n int) {
	if d.err != nil {
		return
	}
	if d.pos+n > len(d.src) {
		d.fail("literal run of %d overruns input", n)
		return
	}
	d.dst = append(d.dst, d.src[d.pos:d.pos+n]...)
	d.pos += n
}

// copyBackref copies compBytes from compOffset bytes behind the write head. Offset
// is zero-based distance-minus-one (offset 0 = the immediately preceding byte), so
// the source index is len-compOffset-1; copying one byte at a time supports the
// length>offset run-length case.
func (d *lz77) copyBackref(compBytes, compOffset int) {
	if d.err != nil {
		return
	}
	start := len(d.dst) - compOffset - 1
	if start < 0 {
		d.fail("back-reference offset %d before start of output", compOffset)
		return
	}
	for i := 0; i < compBytes; i++ {
		d.dst = append(d.dst, d.dst[start+i])
	}
}

// decodeOpcode reads the operands for opcode1, returning the back-reference length
// and offset and the trailing literal count (0 meaning "read a Literal Length").
// Ranges and constants are from ODA §4.7.
func (d *lz77) decodeOpcode(opcode1 byte) (compBytes, compOffset, litCount int) {
	switch {
	case opcode1 == 0x10:
		compBytes = d.longLength() + 9
		compOffset, litCount = d.twoByteOffset()
		compOffset += 0x3FFF
	case opcode1 >= 0x12 && opcode1 <= 0x1F:
		compBytes = int(opcode1&0x0F) + 2
		compOffset, litCount = d.twoByteOffset()
		compOffset += 0x3FFF
	case opcode1 == 0x20:
		compBytes = d.longLength() + 0x21
		compOffset, litCount = d.twoByteOffset()
	case opcode1 >= 0x21 && opcode1 <= 0x3F:
		compBytes = int(opcode1) - 0x1E
		compOffset, litCount = d.twoByteOffset()
	case opcode1 >= 0x40:
		compBytes = int(opcode1&0xF0)>>4 - 1
		opcode2 := int(d.next())
		compOffset = opcode2<<2 | int(opcode1&0x0C)>>2
		litCount = int(opcode1 & 0x03)
	default: // 0x00–0x0F never appear as an opcode; 0x11 handled by the caller
		d.fail("invalid opcode %#02x", opcode1)
	}
	return compBytes, compOffset, litCount
}

// twoByteOffset decodes the two-byte (offset, litCount) operand (ODA §4.7).
func (d *lz77) twoByteOffset() (offset, litCount int) {
	first := d.next()
	offset = int(first>>2) | int(d.next())<<6
	litCount = int(first & 0x03)
	return offset, litCount
}

// literalLength decodes a Literal Length (ODA §4.7). A byte with any high-nibble
// bit set means length 0 and that the byte is the next opcode (returned via the
// third result); a 0x00 byte starts a 0xFF-accumulating run; 0x01..0x0F map to
// value+3.
func (d *lz77) literalLength() (length int, opcode byte, isOpcode bool) {
	b := d.next()
	if b&0xF0 != 0 {
		return 0, b, true
	}
	if b == 0x00 {
		return d.accumulate(0x0F) + 3, 0, false
	}
	return int(b) + 3, 0, false
}

// longLength decodes a Long Compression Offset (ODA §4.7): a non-zero byte is the
// value; a 0x00 byte starts a 0xFF-accumulating run seeded at 0xFF.
func (d *lz77) longLength() int {
	b := d.next()
	if b != 0x00 {
		return int(b)
	}
	return d.accumulate(0xFF)
}

// accumulate implements the shared "0x00 adds 0xFF, non-zero adds and stops"
// run used by both length encodings, starting from the given running total.
func (d *lz77) accumulate(total int) int {
	for d.err == nil {
		n := d.next()
		if n == 0x00 {
			total += 0xFF
			continue
		}
		return total + int(n)
	}
	return total
}
