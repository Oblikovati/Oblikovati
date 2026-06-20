// SPDX-License-Identifier: GPL-2.0-only

package e57fmt

// bitReader pulls little-endian (LSB-first) bit-packed integers out of an E57 bytestream: the
// first value occupies the low bits of the first byte, and each value's low-order bit comes first.
// It tolerates reads past the buffer end by treating missing bytes as zero, so a trailing partial
// value (bit padding at the packet tail) yields a harmless extra value the caller trims by record
// count. Widths are bounded by maxBitPackedBits so the 64-bit accumulator never overflows.
type bitReader struct {
	buf []byte
	pos int    // next byte to consume
	acc uint64 // staged bits not yet returned
	n   uint   // number of valid bits in acc
}

// read returns the next `bits`-wide value (bits in [1, maxBitPackedBits]).
func (r *bitReader) read(bits uint) uint64 {
	for r.n < bits {
		var b byte
		if r.pos < len(r.buf) {
			b = r.buf[r.pos]
		}
		r.pos++
		r.acc |= uint64(b) << r.n
		r.n += 8
	}
	v := r.acc & ((uint64(1) << bits) - 1)
	r.acc >>= bits
	r.n -= bits
	return v
}
