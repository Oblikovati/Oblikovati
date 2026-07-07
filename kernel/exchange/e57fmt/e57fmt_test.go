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
// multiple packets and a page-boundary crossing. Optional colours (Integer 0..255, or ScaledInteger
// scale 0.5 when scaledColor is set) and intensity (ScaledInteger 0..1000) extend the prototype so
// tests can exercise the colour/intensity decode (#645); nil channels reproduce the XYZ-only file.
type e57Builder struct {
	points      [][3]float64
	colors      [][3]int // optional colorRed/Green/Blue in 0..255; nil = no colour channel
	intensities []int    // optional intensity in 0..1000; nil = no intensity channel
	scaledColor bool     // encode colour as ScaledInteger (scale 0.5) rather than Integer
	perPacket   int
}

// bfield is one prototype field the builder emits: its XML descriptor and its bit-packed encoding
// share the same min/max/scale so a decoded value round-trips to the value the test supplied.
type bfield struct {
	name    string
	typ     string // "ScaledInteger" or "Integer"
	min     int64
	max     int64
	scale   float64
	valueAt func(i int) float64 // the real value stored for record i
}

// fieldSpecs is the ordered prototype (X, Y, Z, then colour, then intensity) that drives both the
// XML descriptor and the packet encoding, so they can never disagree.
func (b e57Builder) fieldSpecs() []bfield {
	xyz := func(c int) func(int) float64 { return func(i int) float64 { return b.points[i][c] } }
	specs := []bfield{
		{"cartesianX", "ScaledInteger", testMin, testMax, 1, xyz(0)},
		{"cartesianY", "ScaledInteger", testMin, testMax, 1, xyz(1)},
		{"cartesianZ", "ScaledInteger", testMin, testMax, 1, xyz(2)},
	}
	if b.colors != nil {
		typ, cmax, scale := "Integer", int64(255), 1.0
		if b.scaledColor {
			typ, cmax, scale = "ScaledInteger", 510, 0.5
		}
		col := func(c int) func(int) float64 { return func(i int) float64 { return float64(b.colors[i][c]) } }
		specs = append(specs,
			bfield{"colorRed", typ, 0, cmax, scale, col(0)},
			bfield{"colorGreen", typ, 0, cmax, scale, col(1)},
			bfield{"colorBlue", typ, 0, cmax, scale, col(2)},
		)
	}
	if b.intensities != nil {
		specs = append(specs, bfield{"intensity", "ScaledInteger", 0, 1000, 1, func(i int) float64 { return float64(b.intensities[i]) }})
	}
	return specs
}

// bitWriter packs values LSB-first to mirror bitReader, flushing whole bytes as they fill.
type bitWriter struct {
	buf []byte
	acc uint64
	n   uint
}

func (w *bitWriter) write(v uint64, bits uint) {
	w.acc |= (v & ((uint64(1) << bits) - 1)) << w.n
	w.n += bits
	for w.n >= 8 {
		w.buf = append(w.buf, byte(w.acc))
		w.acc >>= 8
		w.n -= 8
	}
}

// flush byte-aligns the stream (the E57 per-bytestream byte padding the reader tolerates).
func (w *bitWriter) flush() []byte {
	if w.n > 0 {
		w.buf = append(w.buf, byte(w.acc))
		w.acc, w.n = 0, 0
	}
	return w.buf
}

// encodeField bit-packs one field's records for the given point range, applying the inverse of the
// decoder's value = (packed+min)*scale rule so the round-trip is exact.
func (f bfield) encodeField(lo, hi int) []byte {
	bits := bitsToEncode(uint64(f.max - f.min))
	var w bitWriter
	for i := lo; i < hi; i++ {
		packed := int64(math.Round(f.valueAt(i)/f.scale)) - f.min
		w.write(uint64(packed), bits)
	}
	return w.flush()
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
	specs := b.fieldSpecs()
	var out []byte
	for i := 0; i < len(b.points); i += b.perPacket {
		end := i + b.perPacket
		if end > len(b.points) {
			end = len(b.points)
		}
		out = append(out, onePacket(specs, i, end)...)
	}
	return out
}

// onePacket encodes records [lo,hi) of every field as a data packet: a per-bytestream length table
// followed by the byte-aligned bytestreams, in field order.
func onePacket(specs []bfield, lo, hi int) []byte {
	bufs := make([][]byte, len(specs))
	for c, f := range specs {
		bufs[c] = f.encodeField(lo, hi)
	}
	body := make([]byte, 0)
	for _, buf := range bufs {
		body = binary.LittleEndian.AppendUint16(body, uint16(len(buf)))
	}
	for _, buf := range bufs {
		body = append(body, buf...)
	}
	packetLen := 6 + len(body)
	head := []byte{dataPacketType, 0}
	head = binary.LittleEndian.AppendUint16(head, uint16(packetLen-1))
	head = binary.LittleEndian.AppendUint16(head, uint16(len(specs)))
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
	var proto strings.Builder
	for _, f := range b.fieldSpecs() {
		fmt.Fprintf(&proto, `          <%s type="%s" minimum="%d" maximum="%d" scale="%g"/>`+"\n", f.name, f.typ, f.min, f.max, f.scale)
	}
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<e57Root type="Structure" xmlns="http://www.astm.org/COMMIT/E57/2010-e57-v1.0">
  <data3D type="Vector">
    <vectorChild type="Structure">
      <points type="CompressedVector" fileOffset="%d" recordCount="%d">
        <prototype type="Structure">
%s        </prototype>
      </points>
    </vectorChild>
  </data3D>
</e57Root>`, headerSize, len(b.points), proto.String()))
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

// want255 is the 0..1 colour the reader should produce for an 8-bit value, computed the same way as
// normColor (float64 division then narrow) so the float32 comparison is exact, not tolerance-based.
func want255(c [3]int) [3]float32 {
	return [3]float32{
		float32(float64(c[0]) / 255.0),
		float32(float64(c[1]) / 255.0),
		float32(float64(c[2]) / 255.0),
	}
}

func sampleColors(n int) [][3]int {
	cols := make([][3]int, n)
	for i := range cols {
		cols[i] = [3]int{i % 256, (i * 5) % 256, (i * 11) % 256}
	}
	return cols
}

func sampleIntensities(n int) []int {
	is := make([]int, n)
	for i := range is {
		is[i] = (i * 3) % 1001
	}
	return is
}

// TestScanXYZOnly proves a colour/intensity-free scan still decodes and reports no channels.
func TestScanXYZOnly(t *testing.T) {
	pts := samplePoints(10)
	doc, err := Parse(e57Builder{points: pts, perPacket: 16}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	scan, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(scan.Points) != len(pts) {
		t.Fatalf("got %d points, want %d", len(scan.Points), len(pts))
	}
	if scan.HasRGB() || scan.HasIntensity() {
		t.Errorf("XYZ-only scan reports HasRGB=%v HasIntensity=%v, want false/false", scan.HasRGB(), scan.HasIntensity())
	}
}

// TestScanRGBInteger decodes an Integer 0..255 colour channel and checks it normalises to 0..1.
func TestScanRGBInteger(t *testing.T) {
	pts, cols := samplePoints(300), sampleColors(300)
	doc, err := Parse(e57Builder{points: pts, colors: cols, perPacket: 64}.bytes()) // multi-packet
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	scan, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !scan.HasRGB() || scan.HasIntensity() {
		t.Fatalf("HasRGB=%v HasIntensity=%v, want true/false", scan.HasRGB(), scan.HasIntensity())
	}
	if len(scan.RGB) != len(pts) {
		t.Fatalf("got %d colours, want %d", len(scan.RGB), len(pts))
	}
	for i := range scan.RGB {
		want := want255(cols[i])
		if scan.RGB[i] != want {
			t.Fatalf("colour %d = %v, want %v (raw %v)", i, scan.RGB[i], want, cols[i])
		}
	}
}

// TestScanRGBScaledInteger decodes a ScaledInteger (scale 0.5) colour channel, the other common
// E57 colour encoding, and checks it still normalises to 0..1 from its declared bounds.
func TestScanRGBScaledInteger(t *testing.T) {
	pts, cols := samplePoints(50), sampleColors(50)
	doc, err := Parse(e57Builder{points: pts, colors: cols, scaledColor: true, perPacket: 16}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	scan, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !scan.HasRGB() {
		t.Fatal("want HasRGB for a ScaledInteger colour scan")
	}
	for i := range scan.RGB {
		want := want255(cols[i])
		if scan.RGB[i] != want {
			t.Fatalf("scaled colour %d = %v, want %v", i, scan.RGB[i], want)
		}
	}
}

// TestScanIntensityOnly decodes an intensity-only scan and checks intensity passes through raw.
func TestScanIntensityOnly(t *testing.T) {
	pts, is := samplePoints(120), sampleIntensities(120)
	doc, err := Parse(e57Builder{points: pts, intensities: is, perPacket: 32}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	scan, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if scan.HasRGB() || !scan.HasIntensity() {
		t.Fatalf("HasRGB=%v HasIntensity=%v, want false/true", scan.HasRGB(), scan.HasIntensity())
	}
	for i := range scan.Intensity {
		if scan.Intensity[i] != float64(is[i]) {
			t.Fatalf("intensity %d = %v, want raw %d", i, scan.Intensity[i], is[i])
		}
	}
}

// TestScanRGBAndIntensity decodes a scan carrying both channels across page boundaries and checks
// positions, colours, and intensities all stay row-aligned.
func TestScanRGBAndIntensity(t *testing.T) {
	pts, cols, is := samplePoints(900), sampleColors(900), sampleIntensities(900)
	doc, err := Parse(e57Builder{points: pts, colors: cols, intensities: is, perPacket: 64}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	scan, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !scan.HasRGB() || !scan.HasIntensity() {
		t.Fatalf("HasRGB=%v HasIntensity=%v, want true/true", scan.HasRGB(), scan.HasIntensity())
	}
	if len(scan.Points) != 900 || len(scan.RGB) != 900 || len(scan.Intensity) != 900 {
		t.Fatalf("column lengths points=%d rgb=%d intensity=%d, want 900 each", len(scan.Points), len(scan.RGB), len(scan.Intensity))
	}
	for i := range scan.Points {
		if float64(scan.Points[i].X) != pts[i][0] || float64(scan.Points[i].Y) != pts[i][1] || float64(scan.Points[i].Z) != pts[i][2] {
			t.Fatalf("point %d = (%v,%v,%v), want %v", i, scan.Points[i].X, scan.Points[i].Y, scan.Points[i].Z, pts[i])
		}
		want := want255(cols[i])
		if scan.RGB[i] != want {
			t.Fatalf("colour %d = %v, want %v", i, scan.RGB[i], want)
		}
		if scan.Intensity[i] != float64(is[i]) {
			t.Fatalf("intensity %d = %v, want %d", i, scan.Intensity[i], is[i])
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

// TestCartesianIntegerResolution covers the #1789 signal: cartesian XYZ stored at integer
// resolution (unit-scale ScaledInteger, or plain Integer) is flagged, while a sub-unit scale, a
// Float, or a missing channel is not — those keep the ASTM E2807 metre unit.
func TestCartesianIntegerResolution(t *testing.T) {
	xyz := func(kind fieldKind, scale float64) []protoField {
		return []protoField{
			{name: "cartesianX", kind: kind, scale: scale},
			{name: "cartesianY", kind: kind, scale: scale},
			{name: "cartesianZ", kind: kind, scale: scale},
		}
	}
	cases := []struct {
		name   string
		fields []protoField
		want   bool
	}{
		{"scaled-integer unit scale", xyz(kindScaledInteger, 1), true},
		{"plain integer", xyz(kindInteger, 1), true},
		{"scaled-integer sub-unit scale", xyz(kindScaledInteger, 0.001), false},
		{"float", xyz(kindFloat, 1), false},
		{"missing cartesianZ", []protoField{
			{name: "cartesianX", kind: kindScaledInteger, scale: 1},
			{name: "cartesianY", kind: kindScaledInteger, scale: 1},
		}, false},
		{"one float among integers", []protoField{
			{name: "cartesianX", kind: kindScaledInteger, scale: 1},
			{name: "cartesianY", kind: kindFloat},
			{name: "cartesianZ", kind: kindScaledInteger, scale: 1},
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := &Document{points: pointsSection{fields: c.fields}}
			if got := doc.CartesianIntegerResolution(); got != c.want {
				t.Errorf("CartesianIntegerResolution() = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCartesianIntegerResolutionFromBytes confirms the flag survives a real parse: the builder's
// default cartesian is a unit-scale ScaledInteger, the exact non-conformant pattern of #1789.
func TestCartesianIntegerResolutionFromBytes(t *testing.T) {
	doc, err := Parse(e57Builder{points: samplePoints(4), perPacket: 16}.bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !doc.CartesianIntegerResolution() {
		t.Error("a unit-scale ScaledInteger cartesian scan should report integer resolution")
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
