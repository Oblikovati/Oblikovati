// SPDX-License-Identifier: GPL-2.0-only

package e57fmt

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"testing"
)

// --- synthetic E57 builder (a named fake for the on-disk format, per the test guidelines) ---

const (
	testPageSize = 1024
	testPayload  = testPageSize - 4
	testMin      = -100
	testMax      = 100
)

// e57Builder assembles a minimal valid E57: a checksummed-page container holding a CompressedVector
// of cartesian ScaledInteger points (scale 1, range [-100,100] → 8 bits/value) and an XML
// descriptor. perPacket sets how many records each data packet carries, so a test can force
// multiple packets and a page-boundary crossing.
type e57Builder struct {
	points    [][3]float64
	perPacket int
}

// physOf maps a logical offset to its physical offset under testPageSize paging (4-byte CRC/page).
func physOf(logical int) uint64 {
	return uint64((logical/testPayload)*testPageSize + logical%testPayload)
}

func (b e57Builder) bytes() []byte {
	logical := make([]byte, headerSize) // header occupies logical [0,48)
	logical = append(logical, b.sectionHeader()...)
	logical = append(logical, b.packets()...)
	sectionLen := len(logical) - headerSize
	binary.LittleEndian.PutUint64(logical[headerSize+8:], uint64(sectionLen)) // patch sectionLogicalLength
	xmlStart := len(logical)
	xmlDoc := b.xml()
	logical = append(logical, xmlDoc...)
	b.writeHeader(logical, xmlStart, len(xmlDoc))
	phys := paginate(logical)
	binary.LittleEndian.PutUint64(phys[16:], uint64(len(phys))) // filePhysicalLength
	return phys
}

func (b e57Builder) sectionHeader() []byte {
	sec := make([]byte, cvSectionHeaderSize)
	sec[0] = cvSectionType
	binary.LittleEndian.PutUint64(sec[16:], physOf(headerSize+cvSectionHeaderSize)) // dataPhysicalOffset
	return sec
}

func (b e57Builder) packets() []byte {
	var out []byte
	for i := 0; i < len(b.points); i += b.perPacket {
		end := i + b.perPacket
		if end > len(b.points) {
			end = len(b.points)
		}
		out = append(out, b.onePacket(b.points[i:end])...)
	}
	return out
}

func (b e57Builder) onePacket(chunk [][3]float64) []byte {
	bufs := make([][]byte, 3)
	for _, p := range chunk {
		for c := 0; c < 3; c++ {
			bufs[c] = append(bufs[c], byte(int(p[c])-testMin)) // raw = value - minimum, 8 bits
		}
	}
	body := make([]byte, 0, 6+len(chunk)*3)
	for c := 0; c < 3; c++ {
		body = binary.LittleEndian.AppendUint16(body, uint16(len(bufs[c])))
	}
	for c := 0; c < 3; c++ {
		body = append(body, bufs[c]...)
	}
	packetLen := 6 + len(body)
	head := []byte{dataPacketType, 0}
	head = binary.LittleEndian.AppendUint16(head, uint16(packetLen-1))
	head = binary.LittleEndian.AppendUint16(head, 3)
	return append(head, body...)
}

func (b e57Builder) writeHeader(logical []byte, xmlStart, xmlLen int) {
	copy(logical, signature)
	binary.LittleEndian.PutUint32(logical[8:], 1)
	binary.LittleEndian.PutUint64(logical[24:], physOf(xmlStart))
	binary.LittleEndian.PutUint64(logical[32:], uint64(xmlLen))
	binary.LittleEndian.PutUint64(logical[40:], testPageSize)
}

func (b e57Builder) xml() []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<e57Root type="Structure" xmlns="http://www.astm.org/COMMIT/E57/2010-e57-v1.0">
  <data3D type="Vector">
    <vectorChild type="Structure">
      <points type="CompressedVector" fileOffset="%d" recordCount="%d">
        <prototype type="Structure">
          <cartesianX type="ScaledInteger" minimum="%d" maximum="%d"/>
          <cartesianY type="ScaledInteger" minimum="%d" maximum="%d"/>
          <cartesianZ type="ScaledInteger" minimum="%d" maximum="%d"/>
        </prototype>
      </points>
    </vectorChild>
  </data3D>
</e57Root>`, headerSize, len(b.points), testMin, testMax, testMin, testMax, testMin, testMax))
}

// paginate splits a logical stream into testPageSize pages, padding the last and appending a
// zero CRC after each page's payload (the reader does not verify the CRC).
func paginate(logical []byte) []byte {
	var phys []byte
	for i := 0; i < len(logical); i += testPayload {
		end := i + testPayload
		if end > len(logical) {
			end = len(logical)
		}
		page := logical[i:end]
		phys = append(phys, page...)
		phys = append(phys, make([]byte, testPayload-len(page))...) // pad partial last page
		phys = append(phys, 0, 0, 0, 0)                             // page CRC (unchecked)
	}
	return phys
}

// --- tests ---

func samplePoints(n int) [][3]float64 {
	pts := make([][3]float64, n)
	for i := range pts {
		pts[i] = [3]float64{float64(i%200 - 100), float64((i*3)%200 - 100), float64((i*7)%200 - 100)}
	}
	return pts
}

func TestParseAndDecodeSinglePacket(t *testing.T) {
	want := [][3]float64{{1, 2, 3}, {-50, 0, 99}, {100, -100, 42}}
	doc, err := Parse(e57Builder{points: want, perPacket: 16}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := doc.Vertices()
	if err != nil {
		t.Fatalf("Vertices: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d points, want %d", len(got), len(want))
	}
	for i, p := range got {
		if float64(p.X) != want[i][0] || float64(p.Y) != want[i][1] || float64(p.Z) != want[i][2] {
			t.Errorf("point %d = (%v,%v,%v), want %v", i, p.X, p.Y, p.Z, want[i])
		}
	}
}

// TestDecodeMultiPacketAcrossPages forces several small packets and enough data to span page
// boundaries, exercising the CRC-skipping cursor across both packets and pages.
func TestDecodeMultiPacketAcrossPages(t *testing.T) {
	want := samplePoints(900) // > testPayload bytes of packet data → crosses pages
	doc, err := Parse(e57Builder{points: want, perPacket: 64}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := doc.Vertices()
	if err != nil {
		t.Fatalf("Vertices: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d points, want %d", len(got), len(want))
	}
	for i := range got {
		if float64(got[i].X) != want[i][0] || float64(got[i].Y) != want[i][1] || float64(got[i].Z) != want[i][2] {
			t.Fatalf("point %d = (%v,%v,%v), want %v", i, got[i].X, got[i].Y, got[i].Z, want[i])
		}
	}
}

func TestParseHeaderErrors(t *testing.T) {
	if _, err := Parse([]byte("too short")); err == nil {
		t.Error("want error for short input")
	}
	bad := make([]byte, headerSize)
	copy(bad, "NOT-E57!")
	if _, err := Parse(bad); err == nil || !strings.Contains(err.Error(), "not an E57") {
		t.Errorf("want signature error, got %v", err)
	}
	noPage := make([]byte, headerSize)
	copy(noPage, signature)
	if _, err := Parse(noPage); err == nil || !strings.Contains(err.Error(), "page size") {
		t.Errorf("want page-size error, got %v", err)
	}
}

func TestVerticesErrorsWithoutCartesian(t *testing.T) {
	doc := &Document{points: pointsSection{fields: []protoField{{name: "intensity"}}}}
	if _, _, _, err := doc.cartesianIndices(); err == nil {
		t.Error("want error when cartesian channels are absent")
	}
}

func TestBitReaderLSBFirst(t *testing.T) {
	r := bitReader{buf: []byte{0b1010_0011}} // low bits first: 3-bit reads → 011, 100, 01(0 pad)
	if v := r.read(3); v != 0b011 {
		t.Errorf("first 3 bits = %b, want 011", v)
	}
	if v := r.read(3); v != 0b100 {
		t.Errorf("next 3 bits = %b, want 100", v)
	}
}

func TestBitsToEncode(t *testing.T) {
	cases := map[uint64]uint{0: 0, 1: 1, 2: 2, 3: 2, 255: 8, 256: 9, 200: 8, 4294967294: 32}
	for span, want := range cases {
		if got := bitsToEncode(span); got != want {
			t.Errorf("bitsToEncode(%d) = %d, want %d", span, got, want)
		}
	}
}

func TestDecodeFloatsSingleAndDouble(t *testing.T) {
	single := make([]byte, 8)
	binary.LittleEndian.PutUint32(single, math.Float32bits(1.5))
	binary.LittleEndian.PutUint32(single[4:], math.Float32bits(-2.25))
	if got := decodeFloats(single, false); len(got) != 2 || got[0] != 1.5 || got[1] != -2.25 {
		t.Errorf("single floats = %v", got)
	}
	double := make([]byte, 8)
	binary.LittleEndian.PutUint64(double, math.Float64bits(3.14))
	if got := decodeFloats(double, true); len(got) != 1 || got[0] != 3.14 {
		t.Errorf("double floats = %v", got)
	}
}
