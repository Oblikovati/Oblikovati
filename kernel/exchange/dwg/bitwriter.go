// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "math"

// BitWriter is the inverse of BitReader: it appends DWG bit-codes MSB-first into a byte
// buffer, used by the R2000 writer (write.go). Every method mirrors a BitReader reader so
// a value written here decodes to the same value there.
type BitWriter struct {
	buf    []byte
	bitPos uint8 // 0..7, the next bit to fill in the last byte
}

// NewBitWriter returns an empty writer positioned at bit 0.
func NewBitWriter() *BitWriter { return &BitWriter{} }

// Position returns the current bit offset (the number of bits written).
func (w *BitWriter) Position() int { return len(w.buf)*8 - int((8-w.bitPos)%8) }

// Bytes returns the written bytes, the final partial byte zero-padded.
func (w *BitWriter) Bytes() []byte { return w.buf }

// WriteBit appends one bit.
func (w *BitWriter) WriteBit(b uint) {
	if w.bitPos == 0 {
		w.buf = append(w.buf, 0)
	}
	if b&1 != 0 {
		w.buf[len(w.buf)-1] |= 1 << (7 - w.bitPos)
	}
	w.bitPos = (w.bitPos + 1) & 7
}

// WriteBits appends the low n bits of v, MSB-first.
func (w *BitWriter) WriteBits(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.WriteBit(uint(v>>uint(i)) & 1)
	}
}

// WriteRC appends a raw 8-bit byte (byte-aligned fast path, else bit-packed).
func (w *BitWriter) WriteRC(b byte) {
	if w.bitPos == 0 {
		w.buf = append(w.buf, b)
		return
	}
	w.WriteBits(uint64(b), 8)
}

// WriteRS appends a little-endian unsigned 16-bit short.
func (w *BitWriter) WriteRS(v uint16) {
	w.WriteRC(byte(v))
	w.WriteRC(byte(v >> 8))
}

// WriteRL appends a little-endian unsigned 32-bit long.
func (w *BitWriter) WriteRL(v uint32) {
	w.WriteRS(uint16(v))
	w.WriteRS(uint16(v >> 16))
}

// WriteRD appends a little-endian IEEE-754 double.
func (w *BitWriter) WriteRD(f float64) {
	bits := math.Float64bits(f)
	w.WriteRL(uint32(bits))
	w.WriteRL(uint32(bits >> 32))
}

// WriteBS writes a BitShort using the explicit-value forms: 0 and 256 have dedicated
// selectors, a byte-sized value uses the one-byte form, otherwise a full RS.
func (w *BitWriter) WriteBS(v int) {
	switch {
	case v == 0:
		w.WriteBits(2, 2) // selector 10 → 0
	case v == 256:
		w.WriteBits(3, 2) // selector 11 → 256
	case v > 0 && v < 256:
		w.WriteBits(1, 2) // selector 01 → one byte
		w.WriteRC(byte(v))
	default:
		w.WriteBits(0, 2) // selector 00 → full RS
		w.WriteRS(uint16(v))
	}
}

// WriteBL writes a BitLong: 0 and byte-sized values use the short selectors, else full RL.
func (w *BitWriter) WriteBL(v int) {
	switch {
	case v == 0:
		w.WriteBits(2, 2) // selector 10 → 0
	case v > 0 && v < 256:
		w.WriteBits(1, 2) // selector 01 → one byte
		w.WriteRC(byte(v))
	default:
		w.WriteBits(0, 2) // selector 00 → full RL
		w.WriteRL(uint32(v))
	}
}

// WriteBD writes a BitDouble: 0.0 and 1.0 use the short selectors, else a full RD.
func (w *BitWriter) WriteBD(f float64) {
	switch f {
	case 0.0:
		w.WriteBits(2, 2) // selector 10 → 0.0
	case 1.0:
		w.WriteBits(1, 2) // selector 01 → 1.0
	default:
		w.WriteBits(0, 2) // selector 00 → full RD
		w.WriteRD(f)
	}
}

// Write2RD writes two raw doubles.
func (w *BitWriter) Write2RD(p [2]float64) { w.WriteRD(p[0]); w.WriteRD(p[1]) }

// Write3BD writes three BitDoubles (a 3D point).
func (w *BitWriter) Write3BD(p [3]float64) { w.WriteBD(p[0]); w.WriteBD(p[1]); w.WriteBD(p[2]) }

// WriteDD writes a BitDouble-with-default. The full-RD selector (11) is always used, so
// the value is exact regardless of the default — the reader's default-patch forms are an
// encoder size optimisation we do not need.
func (w *BitWriter) WriteDD(f float64) {
	w.WriteBits(3, 2)
	w.WriteRD(f)
}

// WriteBT writes a Bit Thickness: a set flag bit for 0.0, else a clear bit then the BD.
func (w *BitWriter) WriteBT(f float64) {
	if f == 0 {
		w.WriteBit(1)
		return
	}
	w.WriteBit(0)
	w.WriteBD(f)
}

// WriteBE writes a Bit Extrusion: a set flag bit for the default Z axis (0,0,1), else a
// clear bit then three BDs.
func (w *BitWriter) WriteBE(v [3]float64) {
	if v == [3]float64{0, 0, 1} {
		w.WriteBit(1)
		return
	}
	w.WriteBit(0)
	w.Write3BD(v)
}

// WriteMS writes a Modular Short: 15-bit little-endian groups, high bit = continue.
func (w *BitWriter) WriteMS(v int) {
	u := uint(v)
	for {
		word := uint16(u & 0x7fff)
		u >>= 15
		if u != 0 {
			word |= 0x8000
		}
		w.WriteRS(word)
		if u == 0 {
			return
		}
	}
}

// WriteMC writes a Modular Char: 7-bit little-endian groups, high bit = continue, with the
// sign carried in the 0x40 bit of the terminating group.
func (w *BitWriter) WriteMC(v int) {
	neg := v < 0
	u := uint(v)
	if neg {
		u = uint(-v)
	}
	var groups []byte
	for {
		groups = append(groups, byte(u&0x7f))
		u >>= 7
		if u == 0 {
			break
		}
	}
	// The sign sits in 0x40 of the last group; if the last group already uses 0x40, an
	// extra empty group carries the sign (matches ODA / the reader's 0x40 handling).
	if groups[len(groups)-1]&0x40 != 0 {
		groups = append(groups, 0)
	}
	if neg {
		groups[len(groups)-1] |= 0x40
	}
	for i := 0; i < len(groups)-1; i++ {
		w.WriteRC(groups[i] | 0x80)
	}
	w.WriteRC(groups[len(groups)-1])
}

// WriteUMC writes an unsigned Modular Char: 7-bit little-endian groups, high bit = continue.
func (w *BitWriter) WriteUMC(v uint64) {
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		w.WriteRC(b)
		if v == 0 {
			return
		}
	}
}

// WriteHandle writes a handle reference with the given code and absolute value: the
// (code<<4 | nbytes) lead byte then the value's big-endian bytes (shortest form).
func (w *BitWriter) WriteHandle(code uint8, value uint64) {
	var bytes []byte
	for v := value; v != 0; v >>= 8 {
		bytes = append([]byte{byte(v)}, bytes...)
	}
	w.WriteRC(code<<4 | byte(len(bytes)))
	for _, b := range bytes {
		w.WriteRC(b)
	}
}

// Append writes the first o.Position() bits of o into w, preserving bit alignment. Used
// to assemble an object once its size-dependent prefix (bitsize) is known from a
// pre-encoded body.
func (w *BitWriter) Append(o *BitWriter) {
	n := o.Position()
	src := o.buf
	for i := 0; i < n; i++ {
		bit := uint(src[i/8]>>(7-uint(i%8))) & 1
		w.WriteBit(bit)
	}
}

// AlignToByte pads with zero bits to the next byte boundary.
func (w *BitWriter) AlignToByte() {
	for w.bitPos != 0 {
		w.WriteBit(0)
	}
}
