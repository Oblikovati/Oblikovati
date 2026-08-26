// SPDX-License-Identifier: GPL-2.0-only

package dwg

import (
	"math"
	"testing"
)

// bitWriter is an MSB-first bit packer used only by tests to build the packed
// on-disk patterns that BitReader must decode. It mirrors the reader's bit order
// (bit 0 of a byte is 0x80). TestBitWriterMatchesHandComputed pins it against
// hand-computed bytes so a bug here cannot silently mask a reader bug.
type bitWriter struct {
	out []byte
	cur byte
	n   uint8
}

func (w *bitWriter) writeBit(b uint) {
	w.cur = w.cur<<1 | byte(b&1)
	w.n++
	if w.n == 8 {
		w.out = append(w.out, w.cur)
		w.cur, w.n = 0, 0
	}
}

func (w *bitWriter) writeBits(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.writeBit(uint(v>>uint(i)) & 1)
	}
}

func (w *bitWriter) bytes() []byte {
	out := append([]byte(nil), w.out...)
	if w.n > 0 {
		out = append(out, w.cur<<(8-w.n)) // pad trailing partial byte with zeros
	}
	return out
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBitWriterMatchesHandComputed(t *testing.T) {
	var w bitWriter
	w.writeBits(0xA, 4)  // 1010
	w.writeBits(0xBC, 8) // 1011 1100
	// 1010 1011 1100 -> 0xAB, 1100_0000 -> 0xC0
	if got := w.bytes(); !bytesEqual(got, []byte{0xAB, 0xC0}) {
		t.Fatalf("bitWriter cross-check = % X, want AB C0", got)
	}
}

func TestReadBit(t *testing.T) {
	r := NewBitReader([]byte{0xB0}) // 1011 0000
	want := []uint{1, 0, 1, 1, 0, 0, 0, 0}
	for i, exp := range want {
		if got := r.ReadBit(); got != exp {
			t.Fatalf("bit %d = %d, want %d", i, got, exp)
		}
	}
	if r.Err() != nil {
		t.Fatalf("unexpected err: %v", r.Err())
	}
}

func TestReadRCMidByte(t *testing.T) {
	r := NewBitReader([]byte{0xAB, 0xCD}) // read 4 bits then a byte
	if got := r.ReadBits(4); got != 0xA {
		t.Fatalf("nibble = %X, want A", got)
	}
	if got := r.ReadRC(); got != 0xBC {
		t.Fatalf("mid-byte RC = %X, want BC", got)
	}
}

func TestReadRawIntegers(t *testing.T) {
	r := NewBitReader([]byte{0x34, 0x12, 0x78, 0x56, 0x34, 0x12})
	if got := r.ReadRS(); got != 0x1234 {
		t.Fatalf("RS = %X, want 1234", got)
	}
	if got := r.ReadRL(); got != 0x12345678 {
		t.Fatalf("RL = %X, want 12345678", got)
	}
}

func TestReadRD(t *testing.T) {
	bits := math.Float64bits(-3.75)
	buf := make([]byte, 8)
	for i := range 8 {
		buf[i] = byte(bits >> (8 * uint(i))) // little-endian
	}
	if got := NewBitReader(buf).ReadRD(); got != -3.75 {
		t.Fatalf("RD = %v, want -3.75", got)
	}
}

func TestReadBS(t *testing.T) {
	cases := []struct {
		name string
		bits func(*bitWriter)
		want int
	}{
		{"zero", func(w *bitWriter) { w.writeBits(0b10, 2) }, 0},
		{"twofiftysix", func(w *bitWriter) { w.writeBits(0b11, 2) }, 256},
		{"byte", func(w *bitWriter) { w.writeBits(0b01, 2); w.writeBits(0xFE, 8) }, 254},
		{"full", func(w *bitWriter) { w.writeBits(0b00, 2); w.writeBits(0x34, 8); w.writeBits(0x12, 8) }, 0x1234},
	}
	for _, c := range cases {
		var w bitWriter
		c.bits(&w)
		r := NewBitReader(w.bytes())
		if got := r.ReadBS(); got != c.want {
			t.Errorf("%s: ReadBS = %d, want %d", c.name, got, c.want)
		}
		if r.Err() != nil {
			t.Errorf("%s: err %v", c.name, r.Err())
		}
	}
}

func TestReadBL(t *testing.T) {
	cases := []struct {
		name string
		bits func(*bitWriter)
		want int
	}{
		{"zero", func(w *bitWriter) { w.writeBits(0b10, 2) }, 0},
		{"byte", func(w *bitWriter) { w.writeBits(0b01, 2); w.writeBits(0x7F, 8) }, 127},
		{"full", func(w *bitWriter) {
			w.writeBits(0b00, 2)
			for _, b := range []uint64{0x78, 0x56, 0x34, 0x12} {
				w.writeBits(b, 8)
			}
		}, 0x12345678},
	}
	for _, c := range cases {
		var w bitWriter
		c.bits(&w)
		if got := NewBitReader(w.bytes()).ReadBL(); got != c.want {
			t.Errorf("%s: ReadBL = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestReadBD(t *testing.T) {
	var one bitWriter
	one.writeBits(0b01, 2)
	if got := NewBitReader(one.bytes()).ReadBD(); got != 1.0 {
		t.Errorf("BD 01 = %v, want 1.0", got)
	}
	var zero bitWriter
	zero.writeBits(0b10, 2)
	if got := NewBitReader(zero.bytes()).ReadBD(); got != 0.0 {
		t.Errorf("BD 10 = %v, want 0.0", got)
	}
	var full bitWriter
	full.writeBits(0b00, 2)
	bits := math.Float64bits(123.5)
	for i := range 8 {
		full.writeBits(uint64(byte(bits>>(8*uint(i)))), 8)
	}
	if got := NewBitReader(full.bytes()).ReadBD(); got != 123.5 {
		t.Errorf("BD 00 = %v, want 123.5", got)
	}
}

func TestRead3BD(t *testing.T) {
	var w bitWriter
	w.writeBits(0b10, 2) // 0.0
	w.writeBits(0b01, 2) // 1.0
	w.writeBits(0b10, 2) // 0.0
	got := NewBitReader(w.bytes()).Read3BD()
	if got != [3]float64{0, 1, 0} {
		t.Fatalf("3BD = %v, want [0 1 0]", got)
	}
}

func TestReadMC(t *testing.T) {
	cases := []struct {
		name  string
		bytes []byte
		want  int
	}{
		{"five", []byte{0x05}, 5},
		{"neg-five", []byte{0x45}, -5},
		{"sixtyfour", []byte{0xC0, 0x00}, 64},
		{"neg-sixtyfour", []byte{0xC0, 0x40}, -64},
		// #1549: an object-map location delta from a 70 MB file whose 4th 7-bit
		// group sets 0x40, forcing a 5th group to carry the (positive) sign. The
		// old 4-byte cap rejected this as "exceeded 4 bytes".
		{"five-bytes", []byte{0x86, 0x89, 0xb0, 0xde, 0x00}, 197919878},
	}
	for _, c := range cases {
		var w bitWriter
		for _, b := range c.bytes {
			w.writeBits(uint64(b), 8)
		}
		r := NewBitReader(w.bytes())
		if got := r.ReadMC(); got != c.want {
			t.Errorf("%s: ReadMC = %d, want %d", c.name, got, c.want)
		}
		if r.Err() != nil {
			t.Errorf("%s: err %v", c.name, r.Err())
		}
	}
}

func TestReadMS(t *testing.T) {
	cases := []struct {
		name  string
		bytes []byte
		want  int
	}{
		{"single", []byte{0x34, 0x12}, 0x1234},              // word 0x1234, no continuation
		{"double", []byte{0x00, 0x80, 0x02, 0x00}, 0x10000}, // 0x8000 cont, then 2 -> 2<<15
	}
	for _, c := range cases {
		var w bitWriter
		for _, b := range c.bytes {
			w.writeBits(uint64(b), 8)
		}
		if got := NewBitReader(w.bytes()).ReadMS(); got != c.want {
			t.Errorf("%s: ReadMS = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestReadHandle(t *testing.T) {
	var w bitWriter
	w.writeBits(0x42, 8) // code 4, size 2
	w.writeBits(0x05, 8)
	w.writeBits(0x39, 8)
	h := NewBitReader(w.bytes()).ReadHandle()
	if h.Code != 4 || h.Size != 2 || h.Value != 0x0539 {
		t.Fatalf("Handle = %+v, want {Code:4 Size:2 Value:0x539}", h)
	}
}

func TestStickyErrorOnOverrun(t *testing.T) {
	r := NewBitReader([]byte{0xFF})
	r.ReadRC()     // consumes the only byte
	_ = r.ReadRC() // overrun
	if r.Err() == nil {
		t.Fatal("expected sticky error after overrun")
	}
	// Subsequent reads must be zero and must not panic.
	if r.ReadBD() != 0 || r.ReadBS() != 0 {
		t.Fatal("reads after error must return zero")
	}
}

func TestPositionAndAlign(t *testing.T) {
	r := NewBitReader([]byte{0xFF, 0xFF})
	r.ReadBits(3)
	if r.Position() != 3 {
		t.Fatalf("position = %d, want 3", r.Position())
	}
	r.AlignToByte()
	if r.Position() != 8 {
		t.Fatalf("position after align = %d, want 8", r.Position())
	}
	r.AlignToByte() // already aligned: no-op
	if r.Position() != 8 {
		t.Fatalf("position after second align = %d, want 8", r.Position())
	}
}

func TestDetectVersion(t *testing.T) {
	cases := map[string]Version{
		"AC1015": R2000,
		"AC1018": R2004,
		"AC1032": R2018,
	}
	for magic, want := range cases {
		got, err := DetectVersion([]byte(magic + "padding"))
		if err != nil || got != want {
			t.Errorf("DetectVersion(%s) = %v, %v; want %v", magic, got, err, want)
		}
	}
	if _, err := DetectVersion([]byte("AC9999")); err == nil {
		t.Error("expected error for unknown signature")
	}
	if _, err := DetectVersion([]byte("AC1")); err == nil {
		t.Error("expected error for short input")
	}
}

func TestVersionPaged(t *testing.T) {
	if R2000.Paged() {
		t.Error("R2000 must not be paged")
	}
	if !R2018.Paged() || !R2004.Paged() {
		t.Error("R2004+ must be paged")
	}
}
