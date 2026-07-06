// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"bytes"
	"encoding/binary"
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// tetraTris are the four triangles of the unit tetrahedron used by tetraSTL (the ASCII
// fixture in mesh_mold_test.go), so the binary and ASCII decode paths can be compared on
// the same solid. Each triangle is three (x,y,z) corners.
var tetraTris = [4][3][3]float32{
	{{0, 0, 0}, {0, 1, 0}, {1, 0, 0}},
	{{0, 0, 0}, {1, 0, 0}, {0, 0, 1}},
	{{0, 0, 0}, {0, 0, 1}, {0, 1, 0}},
	{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}},
}

// buildBinarySTL encodes triangles as a little-endian binary STL: 80-byte header, uint32
// triangle count, then per triangle a zeroed normal + three vertices + a uint16 attribute.
// It mirrors the on-disk layout meshio.decodeBinarySTL reads, so tests can exercise the
// binary path without committing a large fixture file.
func buildBinarySTL(tris [][3][3]float32) []byte {
	var b bytes.Buffer
	b.Write(make([]byte, 80)) // header (ignored on read)
	_ = binary.Write(&b, binary.LittleEndian, uint32(len(tris)))
	for _, t := range tris {
		_ = binary.Write(&b, binary.LittleEndian, [3]float32{}) // normal, ignored on read
		for _, v := range t {
			_ = binary.Write(&b, binary.LittleEndian, v)
		}
		_ = binary.Write(&b, binary.LittleEndian, uint16(0)) // attribute byte count
	}
	return b.Bytes()
}

// TestParseSTLDecodesBinary is the #1764 regression: a binary STL now imports as welded
// reference-mesh geometry (it used to be unsupported — "not yet wired"). The tetra's 12
// vertex references weld to 4 distinct corners, exactly like the ASCII path.
func TestParseSTLDecodesBinary(t *testing.T) {
	data := buildBinarySTL(tetraTris[:])
	g, err := ParseSTL(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseSTL(binary): %v", err)
	}
	if len(g.Facets) != 4 {
		t.Errorf("binary STL parsed %d facets, want 4", len(g.Facets))
	}
	if len(g.Vertices) != 4 {
		t.Errorf("binary STL welded to %d vertices, want 4", len(g.Vertices))
	}
}

// TestParseSTLBinaryAsciiParity proves the two encodings of the same solid decode to the
// same welded geometry — same vertex/facet counts and coincident vertex positions — so the
// representation a MeshFeature sees does not depend on the file encoding.
func TestParseSTLBinaryAsciiParity(t *testing.T) {
	ascii, err := ParseSTL(bytes.NewReader([]byte(tetraSTL)))
	if err != nil {
		t.Fatalf("ParseSTL(ascii): %v", err)
	}
	binaryG, err := ParseSTL(bytes.NewReader(buildBinarySTL(tetraTris[:])))
	if err != nil {
		t.Fatalf("ParseSTL(binary): %v", err)
	}
	if len(binaryG.Vertices) != len(ascii.Vertices) || len(binaryG.Facets) != len(ascii.Facets) {
		t.Fatalf("parity: binary=%dv/%df ascii=%dv/%df",
			len(binaryG.Vertices), len(binaryG.Facets), len(ascii.Vertices), len(ascii.Facets))
	}
	// Both weld to the same 4 corners; compare as sets so vertex ordering is not asserted.
	if !sameVertexSet(binaryG.Vertices, ascii.Vertices) {
		t.Errorf("parity: welded vertex sets differ\nbinary=%v\nascii=%v", binaryG.Vertices, ascii.Vertices)
	}
}

// TestDecodeSTLMeshRejectsEmpty keeps the "no facets" guard the ASCII path had: an STL with
// no triangles is an error, not an empty mesh, whichever encoding it claims.
func TestDecodeSTLMeshRejectsEmpty(t *testing.T) {
	if _, err := DecodeSTLMesh([]byte("solid empty\nendsolid empty\n")); err == nil {
		t.Error("an STL with no facets should error")
	}
	if _, err := DecodeSTLMesh(buildBinarySTL(nil)); err == nil {
		t.Error("a binary STL with zero triangles should error")
	}
}

// sameVertexSet reports whether two vertex lists hold the same points (order-independent),
// matched on the same 1e-6 grid the welder uses.
func sameVertexSet(a, b []math.Point3) bool {
	if len(a) != len(b) {
		return false
	}
	key := func(p math.Point3) [3]int64 {
		const tol = 1e-6
		return [3]int64{int64(stdmath.Round(p.X / tol)), int64(stdmath.Round(p.Y / tol)), int64(stdmath.Round(p.Z / tol))}
	}
	set := map[[3]int64]int{}
	for _, p := range a {
		set[key(p)]++
	}
	for _, p := range b {
		set[key(p)]--
	}
	for _, n := range set {
		if n != 0 {
			return false
		}
	}
	return true
}
