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

// Reset re-points the reader at buf positioned at bit offset bitPos and clears the
// sticky error, so a single reader can be reused across the many per-object decodes
// in a drawing without allocating a fresh [BitReader] each time (the object pass runs
// once per object — 100k+ on large files — and a stack/collector-held reader avoids
// that many heap allocations).
func (r *BitReader) Reset(buf []byte, bitPos int) {
	r.buf = buf
	r.bytePos = bitPos / 8
	r.bitPos = uint8(bitPos % 8)
	r.err = nil
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
//
// It consumes the stream in chunks of up to 8 bits (each spanning at most two
// bytes) rather than one bit per call: ReadBit-per-bit made the bit reader ~53%
// of total DWG decode CPU (every RC/RS/RL/RD on a mid-byte cursor fell to an
// 8-iteration loop). Behaviour is identical; only the throughput changes.
func (r *BitReader) ReadBits(n int) uint64 {
	if n < 0 || n > 64 {
		r.fail("ReadBits invalid count %d (want 0..64)", n)
		return 0
	}
	var v uint64
	for n > 0 {
		take := min(n, 8)
		v = v<<uint(take) | uint64(r.readUpTo8(take))
		n -= take
	}
	return v
}

// readUpTo8 reads take bits (1..8) MSB-first from the current cursor, which may
// straddle one byte boundary, and advances by take. On overrun it records the
// sticky error and returns 0. Factored out of [BitReader.ReadBits] so the common
// 1–8 bit reads avoid a per-bit loop.
func (r *BitReader) readUpTo8(take int) uint8 {
	if r.err != nil {
		return 0
	}
	avail := 8 - int(r.bitPos) // bits left in the current byte
	if take <= avail {
		if r.bytePos >= len(r.buf) {
			r.fail("ReadBits past end (len=%d bytes)", len(r.buf))
			return 0
		}
		v := r.buf[r.bytePos] >> uint(avail-take) & (1<<uint(take) - 1)
		r.advance(take)
		return v
	}
	if r.bytePos+1 >= len(r.buf) {
		r.fail("ReadBits past end (len=%d bytes)", len(r.buf))
		return 0
	}
	need := take - avail // bits taken from the next byte
	hi := r.buf[r.bytePos] & (1<<uint(avail) - 1)
	lo := r.buf[r.bytePos+1] >> uint(8-need)
	r.advance(take)
	return hi<<uint(need) | lo
}

// ReadRC reads a raw 8-bit char (RC). Valid even when the cursor is mid-byte — a
// mid-byte read is a single two-byte combine, not eight ReadBit calls.
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
	if r.bytePos+1 >= len(r.buf) {
		r.fail("ReadRC past end (len=%d bytes)", len(r.buf))
		return 0
	}
	// The byte spans the low (8-bitPos) bits of the current byte and the high
	// bitPos bits of the next; advancing 8 bits leaves bitPos unchanged (bytePos++).
	k := r.bitPos
	b := r.buf[r.bytePos]<<k | r.buf[r.bytePos+1]>>(8-k)
	r.bytePos++
	return b
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
// with the terminating group's 0x40 bit as the sign. Spans at most 5 bytes: a
// large positive value whose 4th 7-bit group already uses 0x40 needs a 5th group
// to carry the sign (e.g. object-map location deltas in big files; see #1549).
// This matches the reference decoder (libredwg bit_read_MC reads byte[5]).
func (r *BitReader) ReadMC() int {
	var result uint
	var shift uint
	for range 5 {
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
	r.fail("ReadMC exceeded 5 bytes")
	return 0
}

// ReadUMC reads an unsigned Modular Char: little-endian 7-bit groups, high bit =
// continue, with no sign bit. The object map stores handle offsets this way —
// they are always positive (the 0x40 bit of the final byte is a value bit, not a
// sign, unlike [BitReader.ReadMC]).
func (r *BitReader) ReadUMC() uint64 {
	var result uint64
	var shift uint
	for range 9 {
		b := uint64(r.ReadRC())
		result |= (b & 0x7f) << shift
		if b&0x80 == 0 {
			return result
		}
		shift += 7
	}
	r.fail("ReadUMC exceeded 9 bytes")
	return 0
}

// ReadMS reads a Modular Short: little-endian 15-bit groups (one RS each), high
// bit = continue. Spans at most 2 words.
func (r *BitReader) ReadMS() int {
	var result uint
	var shift uint
	for range 2 {
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

// CheckCount validates a decoded element count, failing the reader (sticky) when
// it is negative or exceeds limit. Callers pass the object's byte size as limit:
// every element occupies at least one bit, so a count above the byte size is a
// desync, and guarding before allocating/looping prevents a giant alloc or hang.
// Returns the count when valid, 0 when it fails.
func (r *BitReader) CheckCount(n, limit int) int {
	if r.err != nil {
		return 0
	}
	if n < 0 || n > limit {
		r.fail("element count %d out of range [0,%d] (desync)", n, limit)
		return 0
	}
	return n
}

// ReadBOT reads a Bit Object Type (R2010+): a 2-bit selector then the type.
//
//	00 → one byte (RC)             01 → one byte + 0x1F0
//	10/11 → a 16-bit short (RS)
//
// Earlier versions store the object type as a plain BitShort; see the version
// branch in the object decoder.
func (r *BitReader) ReadBOT() int {
	switch r.ReadBits(2) {
	case 0:
		return int(r.ReadRC())
	case 1:
		return int(r.ReadRC()) + 0x1F0
	default:
		return int(r.ReadRS())
	}
}

// ReadDD reads a BitDouble-with-default (DD): a 2-bit selector that either keeps
// the default, patches some of its little-endian bytes, or replaces it wholesale.
//
//	00 → default unchanged          01 → low 4 bytes replaced
//	10 → bytes 4,5 then 0..3        11 → full 64-bit RD
func (r *BitReader) ReadDD(def float64) float64 {
	switch r.ReadBits(2) {
	case 0:
		return def
	case 1:
		bits := math.Float64bits(def)&^uint64(0xFFFFFFFF) | uint64(r.ReadRL())
		return math.Float64frombits(bits)
	case 2:
		b4, b5 := uint64(r.ReadRC()), uint64(r.ReadRC())
		b0, b1 := uint64(r.ReadRC()), uint64(r.ReadRC())
		b2, b3 := uint64(r.ReadRC()), uint64(r.ReadRC())
		keep := math.Float64bits(def) & 0xFFFF000000000000 // bytes 6,7
		return math.Float64frombits(keep | b5<<40 | b4<<32 | b3<<24 | b2<<16 | b1<<8 | b0)
	default:
		return r.ReadRD()
	}
}

// ReadBT reads a Bit Thickness (R2000+): a flag bit, then 0.0 if set or a BD if
// clear. (Pre-R2000 thickness is always a BD; this package targets R2000+.)
func (r *BitReader) ReadBT() float64 {
	if r.ReadBit() == 1 {
		return 0
	}
	return r.ReadBD()
}

// ReadBE reads a Bit Extrusion (R2000+): a flag bit selecting the default Z axis
// (0,0,1), or three BDs otherwise.
func (r *BitReader) ReadBE() [3]float64 {
	if r.ReadBit() == 1 {
		return [3]float64{0, 0, 1}
	}
	return r.Read3BD()
}

// ReadBLL reads a Bit Long Long: a 3-bit byte count then that many little-endian
// bytes. Used for R2010+ sizes such as the preview length.
func (r *BitReader) ReadBLL() uint64 {
	n := int(r.ReadBits(3))
	var v uint64
	for i := range n {
		v |= uint64(r.ReadRC()) << (8 * uint(i))
	}
	return v
}

// ReadHandle reads an object handle reference (H): a code nibble, a size nibble,
// then Size big-endian value bytes.
func (r *BitReader) ReadHandle() Handle {
	first := r.ReadRC()
	code := first >> 4
	n := int(first & 0x0f)
	var val uint64
	for range n {
		val = val<<8 | uint64(r.ReadRC())
	}
	return Handle{Code: code, Size: uint8(n), Value: val}
}
