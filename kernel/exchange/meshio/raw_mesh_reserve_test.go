// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bytes"
	"encoding/binary"
	"testing"

	"oblikovati.org/math"
)

// TestReserveFillsWithoutRealloc is the #1765 guard: after reserving for N triangles, adding
// exactly N triangles must not reallocate — capacity stays at the reserved size, proving the
// append-doubling churn is gone. It also confirms Reserve preserves the soup's contents.
func TestReserveFillsWithoutRealloc(t *testing.T) {
	const n = 1000
	var m RawMesh
	m.Reserve(n)
	vertsCap, trisCap := cap(m.Verts), cap(m.Tris)
	if vertsCap != n*3 || trisCap != n {
		t.Fatalf("Reserve(%d) gave cap verts=%d tris=%d, want %d/%d", n, vertsCap, trisCap, n*3, n)
	}
	for i := range n {
		f := float64(i)
		m.AddTriangle(math.P3(f, 0, 0), math.P3(0, f, 0), math.P3(0, 0, f))
	}
	if cap(m.Verts) != vertsCap || cap(m.Tris) != trisCap {
		t.Errorf("filling the reserved soup reallocated: verts cap %d->%d, tris cap %d->%d",
			vertsCap, cap(m.Verts), trisCap, cap(m.Tris))
	}
	if m.TriangleCount() != n || len(m.Verts) != n*3 {
		t.Errorf("filled soup has %d tris / %d verts, want %d/%d", m.TriangleCount(), len(m.Verts), n, n*3)
	}
}

// TestReserveGuards covers the edge cases: a non-positive count is a no-op, and Reserve grows
// (never shrinks) capacity relative to what is already stored.
func TestReserveGuards(t *testing.T) {
	var empty RawMesh
	empty.Reserve(0)
	empty.Reserve(-5)
	if empty.Verts != nil || empty.Tris != nil {
		t.Errorf("Reserve with a non-positive count should be a no-op, got %d verts / %d tris", len(empty.Verts), len(empty.Tris))
	}
	m := RawMesh{Verts: []math.Point3{{}, {}, {}}, Tris: [][3]int{{0, 1, 2}}}
	m.Reserve(10) // must keep the one existing triangle and reserve room for 10 more
	if m.TriangleCount() != 1 || len(m.Verts) != 3 {
		t.Errorf("Reserve dropped contents: %d tris / %d verts, want 1/3", m.TriangleCount(), len(m.Verts))
	}
	if cap(m.Verts) < 3+10*3 || cap(m.Tris) < 1+10 {
		t.Errorf("Reserve did not grow capacity: verts cap=%d tris cap=%d", cap(m.Verts), cap(m.Tris))
	}
}

// TestDecodeBinarySTLPreallocatesExactly guards that the binary STL decoder sizes the soup
// from the header count, so decoding a dense mesh does not realloc (#1765). Capacity landing
// exactly at 3×count / count means every AddTriangle hit reserved space.
func TestDecodeBinarySTLPreallocatesExactly(t *testing.T) {
	const n = 512
	tris := make([][3][3]float32, n)
	for i := range tris {
		f := float32(i)
		tris[i] = [3][3]float32{{f, 0, 0}, {0, f, 0}, {0, 0, f}}
	}
	raw, err := DecodeSTL(binarySTLBytes(tris))
	if err != nil {
		t.Fatalf("DecodeSTL(binary): %v", err)
	}
	if raw.TriangleCount() != n {
		t.Fatalf("decoded %d triangles, want %d", raw.TriangleCount(), n)
	}
	if cap(raw.Verts) != n*3 || cap(raw.Tris) != n {
		t.Errorf("binary decode reallocated: verts cap=%d (want %d), tris cap=%d (want %d)",
			cap(raw.Verts), n*3, cap(raw.Tris), n)
	}
}

// binarySTLBytes encodes triangles as little-endian binary STL (80-byte header, uint32 count,
// per triangle: zeroed normal + 3 vertices + uint16 attribute) so the decoder sees a real file.
func binarySTLBytes(tris [][3][3]float32) []byte {
	var b bytes.Buffer
	b.Write(make([]byte, 80))
	_ = binary.Write(&b, binary.LittleEndian, uint32(len(tris)))
	for _, t := range tris {
		_ = binary.Write(&b, binary.LittleEndian, [3]float32{})
		for _, v := range t {
			_ = binary.Write(&b, binary.LittleEndian, v)
		}
		_ = binary.Write(&b, binary.LittleEndian, uint16(0))
	}
	return b.Bytes()
}
