// SPDX-License-Identifier: GPL-2.0-only

package dwg

import "testing"

// TestBitWriterRoundTrip writes each bit-code with BitWriter and reads it back with
// BitReader, asserting the value survives — the property the R2000 writer relies on for a
// byte-faithful round trip through the decoder.
func TestBitWriterRoundTrip(t *testing.T) {
	w := NewBitWriter()
	w.WriteBit(1)
	w.WriteBits(0b101, 3)
	w.WriteRC(0xAB)
	w.WriteRS(0x1234)
	w.WriteRL(0x89ABCDEF)
	w.WriteRD(3.14159)
	for _, v := range []int{0, 1, 255, 256, 1000, 65535} {
		w.WriteBS(v)
	}
	for _, v := range []int{0, 5, 255, 70000} {
		w.WriteBL(v)
	}
	for _, f := range []float64{0, 1, -2.5, 1e12} {
		w.WriteBD(f)
	}
	w.Write3BD([3]float64{1, -2, 3})
	w.WriteDD(42.5)
	w.WriteBT(0)
	w.WriteBT(7.5)
	w.WriteBE([3]float64{0, 0, 1})
	w.WriteBE([3]float64{1, 0, 0})
	for _, v := range []int{0, 100, 40000} {
		w.WriteMS(v)
	}
	for _, v := range []int{0, 63, 64, -64, 12345, -12345} {
		w.WriteMC(v)
	}
	for _, v := range []uint64{0, 127, 128, 1 << 20} {
		w.WriteUMC(v)
	}
	w.WriteHandle(0, 0x4F2)

	r := NewBitReader(w.Bytes())
	eq := func(name string, got, want any) {
		if got != want {
			t.Errorf("%s round-trip = %v, want %v", name, got, want)
		}
	}
	eq("bit", r.ReadBit(), uint(1))
	eq("bits", r.ReadBits(3), uint64(0b101))
	eq("rc", r.ReadRC(), byte(0xAB))
	eq("rs", r.ReadRS(), uint16(0x1234))
	eq("rl", r.ReadRL(), uint32(0x89ABCDEF))
	eq("rd", r.ReadRD(), 3.14159)
	for _, v := range []int{0, 1, 255, 256, 1000, 65535} {
		eq("bs", r.ReadBS(), v)
	}
	for _, v := range []int{0, 5, 255, 70000} {
		eq("bl", r.ReadBL(), v)
	}
	for _, f := range []float64{0, 1, -2.5, 1e12} {
		eq("bd", r.ReadBD(), f)
	}
	eq("3bd", r.Read3BD(), [3]float64{1, -2, 3})
	eq("dd", r.ReadDD(0), 42.5)
	eq("bt0", r.ReadBT(), 0.0)
	eq("bt", r.ReadBT(), 7.5)
	eq("be-def", r.ReadBE(), [3]float64{0, 0, 1})
	eq("be", r.ReadBE(), [3]float64{1, 0, 0})
	for _, v := range []int{0, 100, 40000} {
		eq("ms", r.ReadMS(), v)
	}
	for _, v := range []int{0, 63, 64, -64, 12345, -12345} {
		eq("mc", r.ReadMC(), v)
	}
	for _, v := range []uint64{0, 127, 128, 1 << 20} {
		eq("umc", r.ReadUMC(), v)
	}
	if h := r.ReadHandle(); h.Value != 0x4F2 {
		t.Errorf("handle round-trip = %#x, want 0x4F2", h.Value)
	}
	if err := r.Err(); err != nil {
		t.Fatalf("reader error after round-trip: %v", err)
	}
}
