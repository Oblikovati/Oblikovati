// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"fmt"
	"math"
)

// BitReader reads the unaligned ODA bit-stream primitives that make up a DWG file.
// DWG packs values across byte boundaries (a BitShort can start mid-byte), so the
// cursor tracks both a byte index and a bit offset (0..7), MSB-first within each
// byte — bit 0 of a byte is its 0x80 bit, matching the ODA convention.
//
// Errors are sticky: once a read runs past the end of the buffer (or hits a
// malformed primitive) the first error is recorded and every subsequent read
// returns a zero value. This keeps the many small reads in an entity decoder
// readable; callers check [BitReader.Err] once after a decode block rather than on
// every primitive.
type BitReader struct {
	buf     []byte
	bytePos int
	bitPos  uint8 // 0..7, MSB-first
	err     error
}

// Handle is a raw DWG object handle reference: a 4-bit code plus a big-endian
// value of Size bytes. The code dictates how Value resolves relative to the host
// object's own handle (offset, soft/hard ownership, …); that resolution is the
// caller's concern — ReadHandle only unpacks the on-disk form.
type Handle struct {
	Code  uint8
	Size  uint8
	Value uint64
}

// NewBitReader returns a reader positioned at the first bit of buf.
func NewBitReader(buf []byte) *BitReader { return &BitReader{buf: buf} }

// NewBitReaderAt returns a reader positioned at the given absolute bit offset,
// used when a section map gives an object's start as a bit position.
func NewBitReaderAt(buf []byte, bitPos int) *BitReader {
	return &BitReader{buf: buf, bytePos: bitPos / 8, bitPos: uint8(bitPos % 8)}
}

// Err returns the first error encountered, or nil. A non-nil result invalidates
// every value read since the error occurred.
func (r *BitReader) Err() error { return r.err }

// Position returns the absolute bit offset of the cursor.
func (r *BitReader) Position() int { return r.bytePos*8 + int(r.bitPos) }

// SetPosition moves the cursor to an absolute bit offset.
func (r *BitReader) SetPosition(bitPos int) {
	r.bytePos = bitPos / 8
	r.bitPos = uint8(bitPos % 8)
}

// BitLen returns the total size of the underlying buffer in bits.
func (r *BitReader) BitLen() int { return len(r.buf) * 8 }

// Remaining returns how many bits are left before the end of the buffer.
func (r *BitReader) Remaining() int { return r.BitLen() - r.Position() }

// AlignToByte advances the cursor to the next byte boundary if it is not already
// on one. DWG aligns the start of several structures (e.g. string streams) to a
// byte after a run of single bits.
func (r *BitReader) AlignToByte() {
	if r.bitPos != 0 {
		r.bytePos++
		r.bitPos = 0
	}
}

func (r *BitReader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf("dwg bitreader at bit %d: %s", r.Position(), fmt.Sprintf(format, args...))
	}
}

func (r *BitReader) advance(nbits int) {
	total := int(r.bitPos) + nbits
	r.bytePos += total / 8
	r.bitPos = uint8(total % 8)
}

// ReadBit reads a single bit (B).
func (r *BitReader) ReadBit() uint {
	if r.err != nil {
		return 0
	}
	if r.bytePos >= len(r.buf) {
		r.fail("ReadBit past end (len=%d bytes)", len(r.buf))
		return 0
	}
	bit := uint(r.buf[r.bytePos]>>(7-r.bitPos)) & 1
	r.advance(1)
	return bit
}

// ReadBits reads n bits (0..64) MSB-first into the low bits of the result. Used
// for the 2-bit and 3-bit selector fields and for handle nibbles.
func (r *BitReader) ReadBits(n int) uint64 {
	if n < 0 || n > 64 {
		r.fail("ReadBits invalid count %d (want 0..64)", n)
		return 0
	}
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | uint64(r.ReadBit())
	}
	return v
}

// ReadRC reads a raw 8-bit char (RC). Valid even when the cursor is mid-byte.
func (r *BitReader) ReadRC() byte {
	if r.err != nil {
		return 0
	}
	if r.bitPos == 0 {
		if r.bytePos >= len(r.buf) {
			r.fail("ReadRC past end (len=%d bytes)", len(r.buf))
			return 0
		}
		b := r.buf[r.bytePos]
		r.bytePos++
		return b
	}
	return byte(r.ReadBits(8))
}

// ReadRS reads a raw little-endian unsigned 16-bit short (RS).
func (r *BitReader) ReadRS() uint16 {
	lo := uint16(r.ReadRC())
	hi := uint16(r.ReadRC())
	return lo | hi<<8
}

// ReadRL reads a raw little-endian unsigned 32-bit long (RL).
func (r *BitReader) ReadRL() uint32 {
	lo := uint32(r.ReadRS())
	hi := uint32(r.ReadRS())
	return lo | hi<<16
}

// ReadRD reads a raw little-endian IEEE-754 double (RD).
func (r *BitReader) ReadRD() float64 {
	lo := uint64(r.ReadRL())
	hi := uint64(r.ReadRL())
	return math.Float64frombits(lo | hi<<32)
}

// ReadBS reads a BitShort: a 2-bit selector then the value.
//
//	00 → full 16-bit RS follows   01 → one unsigned byte follows
//	10 → value is 0               11 → value is 256
func (r *BitReader) ReadBS() int {
	switch r.ReadBits(2) {
	case 0:
		return int(r.ReadRS())
	case 1:
		return int(r.ReadRC())
	case 2:
		return 0
	default:
		return 256
	}
}

// ReadBL reads a BitLong: a 2-bit selector then the value.
//
//	00 → full 32-bit RL follows   01 → one unsigned byte follows
//	10 → value is 0               11 → unused (error)
func (r *BitReader) ReadBL() int {
	switch r.ReadBits(2) {
	case 0:
		return int(r.ReadRL())
	case 1:
		return int(r.ReadRC())
	case 2:
		return 0
	default:
		r.fail("ReadBL selector 11 is unused")
		return 0
	}
}

// ReadBD reads a BitDouble: a 2-bit selector then the value.
//
//	00 → full 64-bit RD follows   01 → value is 1.0
//	10 → value is 0.0             11 → unused (error)
func (r *BitReader) ReadBD() float64 {
	switch r.ReadBits(2) {
	case 0:
		return r.ReadRD()
	case 1:
		return 1.0
	case 2:
		return 0.0
	default:
		r.fail("ReadBD selector 11 is unused")
		return 0
	}
}

// Read2RD reads two raw doubles (a 2D point, RD pair).
func (r *BitReader) Read2RD() [2]float64 { return [2]float64{r.ReadRD(), r.ReadRD()} }

// Read2BD reads two BitDoubles (a 2D point).
func (r *BitReader) Read2BD() [2]float64 { return [2]float64{r.ReadBD(), r.ReadBD()} }

// Read3BD reads three BitDoubles (a 3D point).
func (r *BitReader) Read3BD() [3]float64 { return [3]float64{r.ReadBD(), r.ReadBD(), r.ReadBD()} }

// ReadMC reads a Modular Char: little-endian 7-bit groups, high bit = continue,
// with the terminating group's 0x40 bit as the sign. Spans at most 4 bytes.
func (r *BitReader) ReadMC() int {
	var result uint
	var shift uint
	for i := 0; i < 4; i++ {
		b := uint(r.ReadRC())
		if b&0x80 == 0 { // terminating group
			result |= (b & 0x3f) << shift
			if b&0x40 != 0 {
				return -int(result)
			}
			return int(result)
		}
		result |= (b & 0x7f) << shift
		shift += 7
	}
	r.fail("ReadMC exceeded 4 bytes")
	return 0
}

// ReadMS reads a Modular Short: little-endian 15-bit groups (one RS each), high
// bit = continue. Spans at most 2 words.
func (r *BitReader) ReadMS() int {
	var result uint
	var shift uint
	for i := 0; i < 2; i++ {
		w := uint(r.ReadRS())
		if w&0x8000 == 0 {
			result |= w << shift
			return int(result)
		}
		result |= (w & 0x7fff) << shift
		shift += 15
	}
	r.fail("ReadMS exceeded 2 words")
	return 0
}

// ReadHandle reads an object handle reference (H): a code nibble, a size nibble,
// then Size big-endian value bytes.
func (r *BitReader) ReadHandle() Handle {
	first := r.ReadRC()
	code := first >> 4
	n := int(first & 0x0f)
	var val uint64
	for i := 0; i < n; i++ {
		val = val<<8 | uint64(r.ReadRC())
	}
	return Handle{Code: code, Size: uint8(n), Value: val}
}
