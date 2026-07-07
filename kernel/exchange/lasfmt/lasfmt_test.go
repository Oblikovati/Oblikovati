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
	v14          bool     // LAS 1.4 header (375 bytes, 64-bit count) vs 1.2 (227 bytes, 32-bit)
	recordLength uint16   // 0 → format-0 default of 20
	vlrs         [][]byte // full VLRs (54-byte header + payload), placed between the header and points
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
	vlrBytes := 0
	for _, v := range b.vlrs {
		vlrBytes += len(v)
	}
	binary.LittleEndian.PutUint16(hdr[94:], uint16(headerSize))
	binary.LittleEndian.PutUint32(hdr[96:], uint32(headerSize+vlrBytes)) // point data starts after header + VLRs
	binary.LittleEndian.PutUint32(hdr[100:], uint32(len(b.vlrs)))
	hdr[104] = 0 // point format 0
	binary.LittleEndian.PutUint16(hdr[105:], recLen)
	b.writeCount(hdr)
	putVec3(hdr, 131, b.scale)
	putVec3(hdr, 155, b.offset)

	out := hdr
	for _, v := range b.vlrs {
		out = append(out, v...)
	}
	for _, p := range b.points {
		rec := make([]byte, recLen)
		for c, off := 0, 0; c < 3 && off+4 <= int(recLen); c, off = c+1, off+4 {
			binary.LittleEndian.PutUint32(rec[off:], uint32(p[c])) // only write coords the stride holds
		}
		out = append(out, rec...)
	}
	return out
}

// crsVLR assembles one "LASF_Projection" VLR (54-byte header + payload) for the CRS record id.
func crsVLR(recordID uint16, payload []byte) []byte {
	v := make([]byte, vlrHeaderSize+len(payload))
	copy(v[2:18], crsUserID)
	binary.LittleEndian.PutUint16(v[18:], recordID)
	binary.LittleEndian.PutUint16(v[20:], uint16(len(payload)))
	copy(v[vlrHeaderSize:], payload)
	return v
}

// geoKeyDir packs a GeoKey directory holding a single (keyID, tagLoc, count, valueOffset) entry.
func geoKeyDir(keyID, tagLoc, valueOffset uint16) []byte {
	dir := make([]byte, 16)                   // 4-uint16 header + one 4-uint16 key
	binary.LittleEndian.PutUint16(dir[0:], 1) // KeyDirectoryVersion
	binary.LittleEndian.PutUint16(dir[6:], 1) // NumberOfKeys
	binary.LittleEndian.PutUint16(dir[8:], keyID)
	binary.LittleEndian.PutUint16(dir[10:], tagLoc)
	binary.LittleEndian.PutUint16(dir[12:], 1) // count
	binary.LittleEndian.PutUint16(dir[14:], valueOffset)
	return dir
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

// format3LAS builds a one-record LAS 1.2 file in point data record format 3 (XYZ + intensity + RGB),
// exercising the channel decode Scan adds (#1788).
func format3LAS(xyz [3]int32, intensity uint16, rgb [3]uint16) []byte {
	data := make([]byte, 227+34)
	copy(data, signature)
	data[24], data[25] = 1, 2
	binary.LittleEndian.PutUint16(data[94:], 227)
	binary.LittleEndian.PutUint32(data[96:], 227)
	data[104] = 3 // point data record format 3
	binary.LittleEndian.PutUint16(data[105:], 34)
	binary.LittleEndian.PutUint32(data[legacyPointCountOffset:], 1)
	putVec3(data, 131, [3]float64{1, 1, 1})
	rec := data[227:]
	binary.LittleEndian.PutUint32(rec[0:], uint32(xyz[0]))
	binary.LittleEndian.PutUint32(rec[4:], uint32(xyz[1]))
	binary.LittleEndian.PutUint32(rec[8:], uint32(xyz[2]))
	binary.LittleEndian.PutUint16(rec[12:], intensity)
	binary.LittleEndian.PutUint16(rec[28:], rgb[0])
	binary.LittleEndian.PutUint16(rec[30:], rgb[1])
	binary.LittleEndian.PutUint16(rec[32:], rgb[2])
	return data
}

// TestScanDecodesIntensityAndRGB: a format-3 record's intensity (offset 12) and RGB (offset 28)
// decode into 1:1 columns alongside the positions.
func TestScanDecodesIntensityAndRGB(t *testing.T) {
	doc, err := Parse(format3LAS([3]int32{10, 20, 30}, 77, [3]uint16{9, 8, 7}))
	if err != nil {
		t.Fatal(err)
	}
	s, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(s.Points) != 1 || !s.HasIntensity() || !s.HasRGB() {
		t.Fatalf("scan = %+v, want 1 point with intensity+rgb", s)
	}
	// LAS colour is 16-bit, normalised to 0..1 by /65535 at decode (#1787); intensity stays raw.
	wantRGB := [3]float32{float32(9) / 65535, float32(8) / 65535, float32(7) / 65535}
	if s.Intensity[0] != 77 || s.RGB[0] != wantRGB || float64(s.Points[0].X) != 10 {
		t.Errorf("scan row = pt %+v int %v rgb %v, want x=10 int=77 rgb=9,8,7 /65535", s.Points[0], s.Intensity[0], s.RGB[0])
	}
}

// TestScanFormat0Channels: a format-0 record carries no colour (nil RGB column) but its 20-byte
// stride reaches the standard intensity field, so an intensity column is present.
func TestScanFormat0Channels(t *testing.T) {
	doc, err := Parse(lasBuilder{points: [][3]int32{{1, 2, 3}}, scale: [3]float64{1, 1, 1}}.bytes())
	if err != nil {
		t.Fatal(err)
	}
	s, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if s.HasRGB() {
		t.Error("format 0 has no RGB, want nil colour column")
	}
	if !s.HasIntensity() {
		t.Error("format 0's 20-byte record holds the intensity field, want an intensity column")
	}
}

// TestScanRecordTooShortForIntensity: a 12-byte stride holds only the XYZ triple, so neither an
// intensity nor a colour column is produced (but the positions decode).
func TestScanRecordTooShortForIntensity(t *testing.T) {
	doc, err := Parse(lasBuilder{points: [][3]int32{{1, 2, 3}}, scale: [3]float64{1, 1, 1}, recordLength: 12}.bytes())
	if err != nil {
		t.Fatal(err)
	}
	s, err := doc.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if s.HasIntensity() || s.HasRGB() {
		t.Errorf("a 12-byte record has no channels, got int=%v rgb=%v", s.Intensity, s.RGB)
	}
}

// TestRGBRecordOffset maps each colour point format to its red-channel offset and rejects the
// colourless formats.
func TestRGBRecordOffset(t *testing.T) {
	cases := map[uint8]struct {
		off int
		ok  bool
	}{
		0: {0, false}, 1: {0, false},
		2: {20, true}, 3: {28, true}, 5: {28, true},
		7: {30, true}, 8: {30, true}, 10: {30, true},
	}
	for format, want := range cases {
		if off, ok := rgbRecordOffset(format); off != want.off || ok != want.ok {
			t.Errorf("rgbRecordOffset(%d) = %d, %v; want %d, %v", format, off, ok, want.off, want.ok)
		}
	}
}

// --- CRS / unit VLR tests (#1789) ---

func lasWithVLR(vlr []byte) *Document {
	doc, err := Parse(lasBuilder{points: [][3]int32{{1, 1, 1}}, scale: [3]float64{1, 1, 1}, vlrs: [][]byte{vlr}}.bytes())
	if err != nil {
		panic(err)
	}
	return doc
}

// TestCoordinateUnitMetresWKT: a projected WKT declares a linear unit whose metre factor is read
// straight from the string (WKT2 LENGTHUNIT or WKT1 UNIT), regardless of the unit's name.
func TestCoordinateUnitMetresWKT(t *testing.T) {
	cases := []struct {
		name string
		wkt  string
		want float64
		ok   bool
	}{
		{"wkt2 us survey foot",
			`PROJCRS["NAD83 / test",BASEGEOGCRS["NAD83",DATUM["x",ELLIPSOID["GRS 1980",6378137,298.257,LENGTHUNIT["metre",1]]],ANGLEUNIT["degree",0.0174532925199433]],CS[Cartesian,2],AXIS["e",east],AXIS["n",north],LENGTHUNIT["US survey foot",0.30480060960121924]]`,
			0.30480060960121924, true},
		{"wkt1 metre last unit",
			`PROJCS["x",GEOGCS["y",DATUM["d",SPHEROID["s",6378137,298.257]],PRIMEM["Greenwich",0],UNIT["degree",0.0174532925199433]],PROJECTION["Transverse_Mercator"],PARAMETER["scale",1],UNIT["metre",1],AXIS["E",EAST],AXIS["N",NORTH]]`,
			1.0, true},
		{"geographic only (degrees) is not linear",
			`GEOGCS["WGS 84",DATUM["d",SPHEROID["WGS 84",6378137,298.257]],PRIMEM["Greenwich",0],UNIT["degree",0.0174532925199433]]`,
			0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := lasWithVLR(crsVLR(wktRecordID, append([]byte(c.wkt), 0))).CoordinateUnitMetres()
			if ok != c.ok || (ok && got != c.want) {
				t.Errorf("CoordinateUnitMetres() = %v, %v; want %v, %v", got, ok, c.want, c.ok)
			}
		})
	}
}

// TestCoordinateUnitMetresGeoKey: a GeoTIFF ProjLinearUnitsGeoKey with an inline EPSG code maps to
// the unit's size in metres; an unknown code declines so the caller falls back to the heuristic.
func TestCoordinateUnitMetresGeoKey(t *testing.T) {
	cases := []struct {
		name string
		code uint16
		want float64
		ok   bool
	}{
		{"metre", 9001, 1.0, true},
		{"international foot", 9002, 0.3048, true},
		{"us survey foot", 9003, 0.30480060960121924, true},
		{"unknown code declines", 9999, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := geoKeyDir(projLinearUnitsGeoKey, 0, c.code) // tagLoc 0 → value inline
			got, ok := lasWithVLR(crsVLR(geoKeyDirRecordID, dir)).CoordinateUnitMetres()
			if ok != c.ok || (ok && got != c.want) {
				t.Errorf("CoordinateUnitMetres() = %v, %v; want %v, %v", got, ok, c.want, c.ok)
			}
		})
	}
}

// TestCoordinateUnitMetresNoCRS: a file with no projection VLR declares no unit, so the reader falls
// back to its own policy.
func TestCoordinateUnitMetresNoCRS(t *testing.T) {
	doc, err := Parse(lasBuilder{points: [][3]int32{{1, 1, 1}}, scale: [3]float64{0.01, 0.01, 0.01}}.bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := doc.CoordinateUnitMetres(); ok {
		t.Error("a LAS with no CRS VLR must not declare a coordinate unit")
	}
}

// TestCoordinateUnitMetresGeoKeyUserDefined: a user-defined linear unit (32767) reads its metre size
// from the ProjLinearUnitSizeGeoKey entry in the GeoDoubleParams VLR.
func TestCoordinateUnitMetresGeoKeyUserDefined(t *testing.T) {
	dir := make([]byte, 24)                   // 4-uint16 header + two keys
	binary.LittleEndian.PutUint16(dir[0:], 1) // KeyDirectoryVersion
	binary.LittleEndian.PutUint16(dir[6:], 2) // NumberOfKeys
	binary.LittleEndian.PutUint16(dir[8:], projLinearUnitsGeoKey)
	binary.LittleEndian.PutUint16(dir[14:], geoKeyUserDefined) // tagLoc 0, value inline = user-defined
	binary.LittleEndian.PutUint16(dir[16:], projLinearSizeGeoKey)
	binary.LittleEndian.PutUint16(dir[18:], geoDoubleRecordID) // size lives in GeoDoubleParams
	binary.LittleEndian.PutUint16(dir[22:], 0)                 // at index 0
	doubles := make([]byte, 8)
	binary.LittleEndian.PutUint64(doubles, math.Float64bits(0.5)) // 0.5 m per unit

	doc, err := Parse(lasBuilder{points: [][3]int32{{1, 1, 1}}, scale: [3]float64{1, 1, 1},
		vlrs: [][]byte{crsVLR(geoKeyDirRecordID, dir), crsVLR(geoDoubleRecordID, doubles)}}.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := doc.CoordinateUnitMetres(); !ok || got != 0.5 {
		t.Errorf("user-defined unit = %v, %v; want 0.5, true", got, ok)
	}
}

// TestCoordinateUnitMetresDeclines covers the malformed / non-linear inputs that must fall back to
// the caller's policy rather than guess a unit.
func TestCoordinateUnitMetresDeclines(t *testing.T) {
	// A GeoKey whose linear-units key is stored out-of-line (tagLoc != 0) is malformed.
	badLoc := geoKeyDir(projLinearUnitsGeoKey, geoDoubleRecordID, 0)
	if _, ok := lasWithVLR(crsVLR(geoKeyDirRecordID, badLoc)).CoordinateUnitMetres(); ok {
		t.Error("an out-of-line linear-units key must decline")
	}
	// A GeoKey directory that carries no linear-units key at all declines.
	other := geoKeyDir(4099 /* VerticalUnitsGeoKey */, 0, 9001)
	if _, ok := lasWithVLR(crsVLR(geoKeyDirRecordID, other)).CoordinateUnitMetres(); ok {
		t.Error("a directory without ProjLinearUnitsGeoKey must decline")
	}
	// A VLR whose declared length runs past the file: the walk stops, no unit.
	v := crsVLR(wktRecordID, []byte("PROJCS"))
	binary.LittleEndian.PutUint16(v[20:], 9999) // lie about the payload length
	if _, ok := lasWithVLR(v).CoordinateUnitMetres(); ok {
		t.Error("a truncated VLR must decline")
	}
}

// TestUnitFactorAfterName exercises the WKT unit-value parser's success and every reject branch.
func TestUnitFactorAfterName(t *testing.T) {
	if f, ok := unitFactorAfterName(`"metre",0.5]`); !ok || f != 0.5 {
		t.Errorf("valid = %v, %v; want 0.5, true", f, ok)
	}
	if f, ok := unitFactorAfterName(`"metre",2`); !ok || f != 2 { // no trailing bracket (stop < 0)
		t.Errorf("unterminated factor = %v, %v; want 2, true", f, ok)
	}
	for _, bad := range []string{`no quote`, `"unterminated`, `"m"]`, `"m",notanumber]`, `"m",-1]`} {
		if _, ok := unitFactorAfterName(bad); ok {
			t.Errorf("%q must decline", bad)
		}
	}
}

// TestGeoDoubleAtBounds: an out-of-range index or a non-positive size declines.
func TestGeoDoubleAtBounds(t *testing.T) {
	if _, ok := geoDoubleAt(nil, 0); ok {
		t.Error("empty doubles must decline")
	}
	neg := make([]byte, 8)
	binary.LittleEndian.PutUint64(neg, math.Float64bits(-1))
	if _, ok := geoDoubleAt(neg, 0); ok {
		t.Error("non-positive size must decline")
	}
}

// TestCoordinateUnitMetresWalkEdges covers the VLR-walk edges: a leading non-CRS record is skipped,
// a VLR count larger than the file stops the walk after the real records, and a GeoKey directory
// claiming more keys than it holds stops at the truncation.
func TestCoordinateUnitMetresWalkEdges(t *testing.T) {
	metreWKT := append([]byte(`PROJCS["x",UNIT["metre",1]]`), 0)

	other := make([]byte, vlrHeaderSize+4)
	copy(other[2:18], "OTHER_PRODUCER")
	binary.LittleEndian.PutUint16(other[20:], 4)
	doc, err := Parse(lasBuilder{points: [][3]int32{{1, 1, 1}}, scale: [3]float64{1, 1, 1},
		vlrs: [][]byte{other, crsVLR(wktRecordID, metreWKT)}}.bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := doc.CoordinateUnitMetres(); !ok || got != 1.0 {
		t.Errorf("leading non-CRS VLR: got %v, %v; want 1, true", got, ok)
	}

	raw := lasBuilder{points: [][3]int32{{1, 1, 1}}, scale: [3]float64{1, 1, 1},
		vlrs: [][]byte{crsVLR(wktRecordID, metreWKT)}}.bytes()
	binary.LittleEndian.PutUint32(raw[100:], 5) // claim 5 VLRs; only 1 present
	if doc, err = Parse(raw); err != nil {
		t.Fatal(err)
	}
	if got, ok := doc.CoordinateUnitMetres(); !ok || got != 1.0 {
		t.Errorf("VLR count past end: got %v, %v; want 1, true (first read, rest skipped)", got, ok)
	}

	short := geoKeyDir(projLinearUnitsGeoKey, 0, 9001)
	binary.LittleEndian.PutUint16(short[6:], 5) // claim 5 keys; only 1 present
	if got, ok := lasWithVLR(crsVLR(geoKeyDirRecordID, short)).CoordinateUnitMetres(); !ok || got != 1.0 {
		t.Errorf("truncated geokey dir: got %v, %v; want 1, true", got, ok)
	}
}
