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

func le16(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }
func le32(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }
func le64(v uint64) []byte { b := make([]byte, 8); binary.LittleEndian.PutUint64(b, v); return b }
