// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "fmt"

// decompressR2004 expands one R2004+ compressed region into exactly decompSize
// bytes. The format is the ODA LZ77 variant (Open Design Specification §4.7): a
// leading literal run, then opcodes that each copy a back-reference from the
// already-produced output followed by some literal bytes, bounded by decompSize
// (or an explicit 0x11 terminator).
//
// The opcode operand encoding has three families — the high opcodes 0x40-0xFF (and
// the unused-but-tolerated <0x10), the 0x10-0x1F group, and the 0x20-0x3F group —
// each with its own length mask and offset bias. A copy length may exceed its
// back-offset (run-length extension), so copies proceed one byte at a time.
//
// Example:
//
//	page, err := decompressR2004(comp, hdr.decompSize)
func decompressR2004(src []byte, decompSize int) ([]byte, error) {
	if decompSize < 0 {
		return nil, fmt.Errorf("dwg lz77: negative decompressed size %d", decompSize)
	}
	d := &lz77{src: src, dst: make([]byte, decompSize)}
	opcode := d.next()
	if opcode&0xF0 == 0 { // leading literal run
		d.appendLiterals(d.literalLength(opcode))
		opcode = d.next()
	}
	for d.err == nil && d.out < decompSize && opcode != 0x11 {
		compBytes, compOffset, post := d.decodeOpcode(opcode)
		d.copyBackref(compBytes, compOffset)
		opcode = d.trailingLiterals(post)
	}
	if d.err != nil {
		return nil, d.err
	}
	if d.out != decompSize {
		return nil, fmt.Errorf("dwg lz77: produced %d bytes, expected %d", d.out, decompSize)
	}
	return d.dst, nil
}

// lz77 holds the decompression cursors: the input (src/pos) and the fixed-size
// output buffer (dst/out). The output is pre-sized to the declared decompressed
// length so back-references and the tolerated exact-fill final copy stay in bounds.
type lz77 struct {
	src []byte
	pos int
	dst []byte
	out int
	err error
}

func (d *lz77) fail(format string, args ...any) {
	if d.err == nil {
		d.err = fmt.Errorf("dwg lz77 at in=%d out=%d: %s", d.pos, d.out, fmt.Sprintf(format, args...))
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

// appendLiterals copies n bytes verbatim from input to output.
func (d *lz77) appendLiterals(n int) {
	if d.err != nil {
		return
	}
	if d.out+n > len(d.dst) {
		d.fail("literal run of %d overflows output", n)
		return
	}
	if d.pos+n > len(d.src) {
		d.fail("literal run of %d overruns input", n)
		return
	}
	copy(d.dst[d.out:], d.src[d.pos:d.pos+n])
	d.out += n
	d.pos += n
}

// copyBackref copies compBytes from compOffset bytes behind the write head. The
// offset is one-based (offset 1 = the immediately preceding byte) because each
// opcode family folds its +1/+plus bias into compOffset; copying one byte at a
// time supports the length>offset run-length case.
func (d *lz77) copyBackref(compBytes, compOffset int) {
	if d.err != nil {
		return
	}
	start := d.out - compOffset
	if start < 0 {
		d.fail("back-reference offset %d before start of output", compOffset)
		return
	}
	if d.out+compBytes > len(d.dst) {
		d.fail("back-reference of %d overflows output", compBytes)
		return
	}
	for i := 0; i < compBytes; i++ {
		d.dst[d.out] = d.dst[start+i]
		d.out++
	}
}

// trailingLiterals copies the literal bytes that follow a back-reference and
// returns the next opcode. The count comes from the low two bits of post (the
// opcode left after operand decoding); when those are 0 the next byte is read and,
// if its high nibble is clear, decoded as a Literal Length — otherwise that byte
// is itself the next opcode (no literals).
func (d *lz77) trailingLiterals(post byte) byte {
	litCount := int(post & 3)
	if litCount == 0 {
		opcode := d.next()
		if opcode&0xF0 != 0 {
			return opcode // high nibble set: zero literals, this is the next opcode
		}
		d.appendLiterals(d.literalLength(opcode))
		return d.next()
	}
	d.appendLiterals(litCount)
	return d.next()
}

// decodeOpcode reads the back-reference operands for opcode and returns the copy
// length, the (one-based) copy offset, and the opcode value that carries the
// trailing literal count in its low two bits. Masks and biases per ODA §4.7 as
// realised in the reference decoder.
func (d *lz77) decodeOpcode(opcode byte) (compBytes, compOffset int, post byte) {
	switch {
	case opcode < 0x10 || opcode >= 0x40:
		compBytes = int(opcode>>4) - 1
		op2 := int(d.next())
		compOffset = (int(opcode>>2)&3 | op2<<2) + 1
		post = opcode // literal count from the original opcode
	case opcode < 0x20: // 0x10-0x1F
		compBytes = d.compressedBytes(opcode, 7)
		compOffset = int(opcode&8) << 11
		post = d.twoByteOffset(0x4000, &compOffset)
	default: // 0x20-0x3F
		compBytes = d.compressedBytes(opcode, 0x1F)
		post = d.twoByteOffset(1, &compOffset)
	}
	return compBytes, compOffset, post
}

// twoByteOffset folds the next two bytes into offset (OR-ing the existing partial
// value) and adds the family bias plus; it returns the first byte, whose low two
// bits give the trailing literal count and which becomes the next opcode.
func (d *lz77) twoByteOffset(plus int, offset *int) byte {
	first := d.next()
	*offset |= int(first >> 2)
	*offset |= int(d.next()) << 6
	*offset += plus
	return first
}

// compressedBytes decodes a copy length: the low (opcode&mask) bits, or — when
// those are zero — a run where each 0x00 byte adds 0xFF and the first non-zero
// byte adds itself plus mask. Always +2 (the minimum copy is 2 bytes).
func (d *lz77) compressedBytes(opcode, mask byte) int {
	n := int(opcode & mask)
	if n != 0 {
		return n + 2
	}
	var last byte
	for {
		last = d.next()
		if last != 0 || d.err != nil {
			break
		}
		n += 0xFF
	}
	return n + int(last) + int(mask) + 2
}

// literalLength decodes a Literal Length whose low nibble is opcode&0x0F, or —
// when that nibble is zero — a 0x00-run (each 0x00 adds 0xFF, first non-zero adds
// itself plus 0x0F). Always +3.
func (d *lz77) literalLength(opcode byte) int {
	n := int(opcode & 0x0F)
	if n != 0 {
		return n + 3
	}
	var last byte
	for {
		last = d.next()
		if last != 0 || d.err != nil {
			break
		}
		n += 0xFF
	}
	return n + 0x0F + int(last) + 3
}
