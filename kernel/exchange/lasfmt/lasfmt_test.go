// SPDX-License-Identifier: GPL-2.0-only

package lasfmt

import (
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

// --- synthetic LAS builder (a named fake for the on-disk format) ---

// lasBuilder assembles a minimal valid uncompressed LAS file: a public header followed by
// fixed-length point records whose leading int32 X/Y/Z are filled (other fields left zero).
type lasBuilder struct {
	points       [][3]int32
	scale        [3]float64
	offset       [3]float64
	v14          bool   // LAS 1.4 header (375 bytes, 64-bit count) vs 1.2 (227 bytes, 32-bit)
	recordLength uint16 // 0 → format-0 default of 20
}

func (b lasBuilder) bytes() []byte {
	headerSize := 227
	if b.v14 {
		headerSize = 375
	}
	recLen := b.recordLength
	if recLen == 0 {
		recLen = 20 // point data record format 0
	}
	hdr := make([]byte, headerSize)
	copy(hdr, signature)
	hdr[24], hdr[25] = 1, 2
	if b.v14 {
		hdr[25] = 4
	}
	binary.LittleEndian.PutUint16(hdr[94:], uint16(headerSize))
	binary.LittleEndian.PutUint32(hdr[96:], uint32(headerSize)) // point data starts right after header
	hdr[104] = 0                                                // point format 0
	binary.LittleEndian.PutUint16(hdr[105:], recLen)
	b.writeCount(hdr)
	putVec3(hdr, 131, b.scale)
	putVec3(hdr, 155, b.offset)

	out := hdr
	for _, p := range b.points {
		rec := make([]byte, recLen)
		for c, off := 0, 0; c < 3 && off+4 <= int(recLen); c, off = c+1, off+4 {
			binary.LittleEndian.PutUint32(rec[off:], uint32(p[c])) // only write coords the stride holds
		}
		out = append(out, rec...)
	}
	return out
}

// writeCount populates the legacy 32-bit count, and for a 1.4 header the authoritative 64-bit one
// (with the legacy field left zero, the path a big/extended LAS uses).
func (b lasBuilder) writeCount(hdr []byte) {
	if b.v14 {
		binary.LittleEndian.PutUint64(hdr[las14PointCountOffset:], uint64(len(b.points)))
		return
	}
	binary.LittleEndian.PutUint32(hdr[legacyPointCountOffset:], uint32(len(b.points)))
}

func putVec3(b []byte, off int, v [3]float64) {
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint64(b[off+i*8:], math.Float64bits(v[i]))
	}
}

// --- tests ---

func TestParseAndDecodeLAS12(t *testing.T) {
	scale := [3]float64{0.001, 0.001, 0.01}
	offset := [3]float64{100, -50, 0}
	pts := [][3]int32{{1000, 2000, 300}, {-5000, 0, 12345}, {2147483647, -2147483648, 1}}
	doc, err := Parse(lasBuilder{points: pts, scale: scale, offset: offset}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := doc.Vertices()
	if err != nil {
		t.Fatalf("Vertices: %v", err)
	}
	if len(got) != len(pts) {
		t.Fatalf("got %d points, want %d", len(got), len(pts))
	}
	for i, p := range pts {
		wantX := float64(p[0])*scale[0] + offset[0]
		wantY := float64(p[1])*scale[1] + offset[1]
		wantZ := float64(p[2])*scale[2] + offset[2]
		if float64(got[i].X) != wantX || float64(got[i].Y) != wantY || float64(got[i].Z) != wantZ {
			t.Errorf("point %d = (%v,%v,%v), want (%v,%v,%v)", i, got[i].X, got[i].Y, got[i].Z, wantX, wantY, wantZ)
		}
	}
}

func TestDecodeLAS14Count64(t *testing.T) {
	scale := [3]float64{1, 1, 1}
	pts := [][3]int32{{1, 2, 3}, {4, 5, 6}}
	doc, err := Parse(lasBuilder{points: pts, scale: scale, v14: true}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if doc.header.pointCount != 2 {
		t.Fatalf("1.4 count = %d, want 2 (from the 64-bit field)", doc.header.pointCount)
	}
	got, err := doc.Vertices()
	if err != nil || len(got) != 2 || float64(got[1].X) != 4 {
		t.Fatalf("decoded = %v, err=%v", got, err)
	}
}

func TestParseHeaderErrors(t *testing.T) {
	if _, err := Parse([]byte("LASF tiny")); err == nil {
		t.Error("want error for short input")
	}
	bad := make([]byte, minHeaderSize)
	copy(bad, "NOPE")
	if _, err := Parse(bad); err == nil || !strings.Contains(err.Error(), "not a LAS") {
		t.Errorf("want signature error, got %v", err)
	}
	badFmt := lasBuilder{scale: [3]float64{1, 1, 1}}.bytes()
	badFmt[104] = 11 // out-of-range point data record format
	if _, err := Parse(badFmt); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Errorf("want format-range error, got %v", err)
	}
}

func TestVerticesTruncated(t *testing.T) {
	raw := lasBuilder{points: [][3]int32{{1, 1, 1}}, scale: [3]float64{1, 1, 1}}.bytes()
	binary.LittleEndian.PutUint32(raw[legacyPointCountOffset:], 1000) // claim far more points than bytes
	doc, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := doc.Vertices(); err == nil || !strings.Contains(err.Error(), "exceed") {
		t.Errorf("want truncation error, got %v", err)
	}
}

func TestVerticesRecordTooSmall(t *testing.T) {
	doc, err := Parse(lasBuilder{points: [][3]int32{{1, 1, 1}}, scale: [3]float64{1, 1, 1}, recordLength: 8}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := doc.Vertices(); err == nil || !strings.Contains(err.Error(), "too small") {
		t.Errorf("want record-length error, got %v", err)
	}
}
