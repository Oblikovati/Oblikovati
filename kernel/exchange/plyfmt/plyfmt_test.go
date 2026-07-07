// SPDX-License-Identifier: GPL-2.0-only

package plyfmt

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// TestParseASCIIVerticesAndFaces: an ASCII PLY's vertices (ignoring extra columns) and faces decode.
func TestParseASCIIVerticesAndFaces(t *testing.T) {
	src := "ply\nformat ascii 1.0\n" +
		"element vertex 3\nproperty float x\nproperty float y\nproperty float z\nproperty uchar r\n" +
		"element face 1\nproperty list uchar int vertex_indices\n" +
		"end_header\n" +
		"0 0 0 255\n1 0 0 128\n0 1 0 64\n3 0 1 2\n"
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	v, err := d.Vertices()
	if err != nil || len(v) != 3 || v[1].X != 1 || v[2].Y != 1 {
		t.Fatalf("vertices = %+v, err %v", v, err)
	}
	f, err := d.Faces()
	if err != nil || len(f) != 1 || len(f[0]) != 3 || f[0][2] != 2 {
		t.Fatalf("faces = %+v, err %v", f, err)
	}
}

// TestParseBinaryVerticesAndFaces: a binary little-endian PLY's vertices and a triangle face decode,
// striding past a non-position vertex property.
func TestParseBinaryVerticesAndFaces(t *testing.T) {
	var b bytes.Buffer
	for _, v := range [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}} {
		for _, c := range v {
			_ = binary.Write(&b, binary.LittleEndian, c)
		}
		b.WriteByte(9) // a uchar property after xyz
	}
	b.WriteByte(3) // face: list count (uchar) = 3
	for _, idx := range []int32{0, 1, 2} {
		_ = binary.Write(&b, binary.LittleEndian, idx)
	}
	src := append([]byte("ply\nformat binary_little_endian 1.0\n"+
		"element vertex 3\nproperty float x\nproperty float y\nproperty float z\nproperty uchar flag\n"+
		"element face 1\nproperty list uchar int vertex_indices\n"+
		"end_header\n"), b.Bytes()...)

	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	v, _ := d.Vertices()
	if len(v) != 3 || v[1].X != 1 {
		t.Fatalf("binary vertices = %+v", v)
	}
	f, err := d.Faces()
	if err != nil || len(f) != 1 || f[0][0] != 0 || f[0][2] != 2 {
		t.Fatalf("binary faces = %+v, err %v", f, err)
	}
}

// TestScalarTypes: every PLY scalar type decodes to the right value and reports the right size.
func TestScalarTypes(t *testing.T) {
	le := binary.ByteOrder(binary.LittleEndian)
	cases := []struct {
		typ  string
		size int
		b    []byte
		want float64
	}{
		{"char", 1, []byte{0xFE}, -2},
		{"uchar", 1, []byte{200}, 200},
		{"short", 2, le16(0xFFFF), -1},
		{"ushort", 2, le16(513), 513},
		{"int", 4, le32(0xFFFFFFFF), -1},
		{"uint", 4, le32(70000), 70000},
		{"float", 4, le32(math.Float32bits(2.5)), 2.5},
		{"double", 8, le64(math.Float64bits(-3.25)), -3.25},
		{"weird", 0, nil, 0},
	}
	for _, c := range cases {
		if got := plyTypeSize(c.typ); got != c.size {
			t.Errorf("plyTypeSize(%s) = %d, want %d", c.typ, got, c.size)
		}
		if c.b != nil {
			if got := scalarValue(c.b, c.typ, c.size, le); got != c.want {
				t.Errorf("scalarValue(%s) = %v, want %v", c.typ, got, c.want)
			}
		}
	}
}

// TestParseErrors: a non-PLY blob, no vertex element, and missing x/y/z are rejected; no face
// element yields nil faces.
func TestParseErrors(t *testing.T) {
	if _, err := Parse([]byte("nope")); err == nil {
		t.Error("non-PLY should fail")
	}
	noVtx, _ := Parse([]byte("ply\nformat ascii 1.0\nelement face 0\nend_header\n"))
	if _, err := noVtx.Vertices(); err == nil {
		t.Error("no vertex element should fail Vertices")
	}
	if f, err := noVtx.Faces(); err != nil || f != nil {
		t.Errorf("no face element should yield nil faces, got %+v err %v", f, err)
	}
	noXYZ, _ := Parse([]byte("ply\nformat ascii 1.0\nelement vertex 1\nproperty float u\nproperty float v\nend_header\n0 0\n"))
	if _, err := noXYZ.Vertices(); err == nil {
		t.Error("vertex without x/y/z should fail")
	}
}

// TestScanASCIIChannels: an ascii vertex carrying intensity and RGB decodes into 1:1 columns with
// the raw property values preserved, and the positions still decode.
func TestScanASCIIChannels(t *testing.T) {
	src := "ply\nformat ascii 1.0\n" +
		"element vertex 2\nproperty float x\nproperty float y\nproperty float z\n" +
		"property ushort intensity\nproperty uchar red\nproperty uchar green\nproperty uchar blue\n" +
		"end_header\n1 2 3 77 9 8 7\n4 5 6 12 1 2 3\n"
	d, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s, err := d.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(s.Points) != 2 || !s.HasIntensity() || !s.HasRGB() {
		t.Fatalf("scan = %+v, want 2 points with intensity+rgb", s)
	}
	// uchar colour is normalised to 0..1 by /255 at decode (#1787); intensity stays raw.
	if s.Points[1].X != 4 || s.Intensity[0] != 77 || s.RGB[0] != uchar255(9, 8, 7) {
		t.Errorf("row 0 = pt %+v int %v rgb %v, want x=1 int=77 rgb=9,8,7 /255", s.Points[0], s.Intensity[0], s.RGB[0])
	}
	if s.Intensity[1] != 12 || s.RGB[1] != uchar255(1, 2, 3) {
		t.Errorf("row 1 = int %v rgb %v, want 12 and 1,2,3 /255", s.Intensity[1], s.RGB[1])
	}
}

// uchar255 is the expected 0..1 normalisation of a uchar colour triple (the same float32 /255 the
// decoder applies, so the comparison is bit-exact).
func uchar255(r, g, b float32) [3]float32 {
	return [3]float32{r / 255, g / 255, b / 255}
}

// TestScanBinaryChannels: a binary vertex with a uint16 intensity and uchar RGB decodes its channels
// at their record offsets, striding correctly past the float positions.
func TestScanBinaryChannels(t *testing.T) {
	var b bytes.Buffer
	for _, c := range []float32{1, 2, 3} {
		_ = binary.Write(&b, binary.LittleEndian, c)
	}
	_ = binary.Write(&b, binary.LittleEndian, uint16(77))
	b.Write([]byte{9, 8, 7})
	src := append([]byte("ply\nformat binary_little_endian 1.0\n"+
		"element vertex 1\nproperty float x\nproperty float y\nproperty float z\n"+
		"property ushort intensity\nproperty uchar red\nproperty uchar green\nproperty uchar blue\n"+
		"end_header\n"), b.Bytes()...)
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s, err := d.Scan()
	if err != nil || len(s.Points) != 1 {
		t.Fatalf("Scan = %+v, err %v", s, err)
	}
	if s.Intensity[0] != 77 || s.RGB[0] != uchar255(9, 8, 7) || s.Points[0].Z != 3 {
		t.Errorf("scan row = pt %+v int %v rgb %v, want z=3 int=77 rgb=9,8,7 /255", s.Points[0], s.Intensity[0], s.RGB[0])
	}
}

// TestColorScale maps each PLY colour scalar type to its 0..1 normaliser: integers by their unsigned
// max, float/unknown passed through (#1787).
func TestColorScale(t *testing.T) {
	cases := map[string]float32{
		"uchar": 255, "uint8": 255, "char": 255, "int8": 255,
		"ushort": 65535, "uint16": 65535, "short": 65535, "int16": 65535,
		"uint": 4294967295, "int32": 4294967295,
		"float": 1, "double": 1, "weird": 1,
	}
	for typ, want := range cases {
		if got := colorScale(typ); got != want {
			t.Errorf("colorScale(%q) = %v, want %v", typ, got, want)
		}
	}
}

// TestScanUint16ColorNormalised: a ushort colour triple normalises by /65535, not /255, so a value
// that would read near-white as 8-bit reads its true dim shade (#1787).
func TestScanUint16ColorNormalised(t *testing.T) {
	var b bytes.Buffer
	for _, c := range []float32{0, 0, 0} {
		_ = binary.Write(&b, binary.LittleEndian, c)
	}
	for _, c := range []uint16{200, 100, 65535} {
		_ = binary.Write(&b, binary.LittleEndian, c)
	}
	src := append([]byte("ply\nformat binary_little_endian 1.0\n"+
		"element vertex 1\nproperty float x\nproperty float y\nproperty float z\n"+
		"property ushort red\nproperty ushort green\nproperty ushort blue\n"+
		"end_header\n"), b.Bytes()...)
	d, _ := Parse(src)
	s, err := d.Scan()
	if err != nil || len(s.RGB) != 1 {
		t.Fatalf("Scan = %+v, err %v", s, err)
	}
	want := [3]float32{float32(200) / 65535, float32(100) / 65535, 1}
	if s.RGB[0] != want {
		t.Errorf("uint16 colour = %v, want %v (/65535, blue saturates to 1)", s.RGB[0], want)
	}
}

// TestScanNoChannels: a positions-only vertex yields nil colour and intensity columns.
func TestScanNoChannels(t *testing.T) {
	src := "ply\nformat ascii 1.0\nelement vertex 1\nproperty float x\nproperty float y\nproperty float z\nend_header\n1 2 3\n"
	d, _ := Parse([]byte(src))
	s, err := d.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if s.HasRGB() || s.HasIntensity() {
		t.Errorf("positions-only scan should have nil channel columns, got rgb=%v int=%v", s.RGB, s.Intensity)
	}
}

func le16(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }
func le32(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }
func le64(v uint64) []byte { b := make([]byte, 8); binary.LittleEndian.PutUint64(b, v); return b }
