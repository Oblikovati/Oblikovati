// SPDX-License-Identifier: GPL-2.0-only

package pointcloud

import (
	"bytes"
	"encoding/binary"
	"math"
	"oblikovati.org/kernel/exchange"
	"testing"
)

// TestPLYASCII: an ASCII PLY's vertex positions decode, ignoring extra per-vertex columns and the
// face element.
func TestPLYASCII(t *testing.T) {
	src := "ply\nformat ascii 1.0\n" +
		"element vertex 2\nproperty float x\nproperty float y\nproperty float z\nproperty uchar red\n" +
		"element face 1\nproperty list uchar int vertex_indices\n" +
		"end_header\n" +
		"1 2 3 255\n4 5 6 128\n3 0 1 2\n"
	pts, err := ReadScan("m.ply", []byte(src), exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("Read ascii PLY: %v", err)
	}
	if len(pts) != 2 {
		t.Fatalf("decoded %d points, want 2 (the faces are skipped)", len(pts))
	}
	if pts[0].X != 1 || pts[0].Y != 2 || pts[0].Z != 3 || pts[1].X != 4 {
		t.Errorf("points = %+v, want (1,2,3),(4,5,6)", pts)
	}
}

// TestPLYBinaryLittleEndian: a binary little-endian PLY decodes, striding past a non-position
// property (a uchar) between the floats.
func TestPLYBinaryLittleEndian(t *testing.T) {
	var body bytes.Buffer
	// vertex record layout: float x, float y, float z, uchar flag.
	for _, v := range [][3]float32{{1, 2, 3}, {-4, 5, -6}} {
		for _, c := range v {
			_ = binary.Write(&body, binary.LittleEndian, c)
		}
		body.WriteByte(7) // the trailing uchar property
	}
	src := append([]byte("ply\nformat binary_little_endian 1.0\n"+
		"element vertex 2\nproperty float x\nproperty float y\nproperty float z\nproperty uchar flag\n"+
		"end_header\n"), body.Bytes()...)

	pts, err := ReadScan("m.ply", src, exchange.TranslationOptions{})
	if err != nil {
		t.Fatalf("Read binary PLY: %v", err)
	}
	if len(pts) != 2 || pts[0].X != 1 || pts[0].Z != 3 || pts[1].X != -4 || pts[1].Z != -6 {
		t.Fatalf("decoded %+v, want (1,2,3),(-4,5,-6)", pts)
	}
}

// TestPLYBinaryDoublePrecision: double-precision positions decode.
func TestPLYBinaryDoublePrecision(t *testing.T) {
	var body bytes.Buffer
	for _, c := range []float64{1.5, 2.5, 3.5} {
		_ = binary.Write(&body, binary.LittleEndian, math.Float64bits(c))
	}
	src := append([]byte("ply\nformat binary_little_endian 1.0\n"+
		"element vertex 1\nproperty double x\nproperty double y\nproperty double z\n"+
		"end_header\n"), body.Bytes()...)
	pts, err := ReadScan("m.ply", src, exchange.TranslationOptions{})
	if err != nil || len(pts) != 1 || pts[0].Y != 2.5 {
		t.Fatalf("double PLY = %+v, err %v; want (1.5,2.5,3.5)", pts, err)
	}
}

// TestPLYErrors: a non-PLY blob, a missing vertex element, and missing x/y/z are rejected.
func TestPLYErrors(t *testing.T) {
	if _, err := (plyReader{}).Read([]byte("not a ply")); err == nil {
		t.Error("a non-PLY blob should fail")
	}
	noVtx := "ply\nformat ascii 1.0\nelement face 0\nend_header\n"
	if _, err := (plyReader{}).Read([]byte(noVtx)); err == nil {
		t.Error("a PLY with no vertex element should fail")
	}
	noXYZ := "ply\nformat ascii 1.0\nelement vertex 1\nproperty float u\nproperty float v\nend_header\n0 0\n"
	if _, err := (plyReader{}).Read([]byte(noXYZ)); err == nil {
		t.Error("a vertex without x/y/z should fail")
	}
}
