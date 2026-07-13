// SPDX-License-Identifier: GPL-2.0-only

package ipt

import (
	"encoding/binary"
	"math"
)

// Mesh is a decoded triangle mesh: vertex positions in cm and triangles as vertex indices.
type Mesh struct {
	Verts [][3]float64
	Tris  [][3]int
}

// gfxPatchTypes are the PmGraphicsSegment nodes that carry a triangulated face patch. In this
// Inventor generation the display tessellation is stored per face as one of these (the clean
// MeshFacets node InventorLoader documents for other versions is absent here); each is a header
// then List2 arrays: points, triangle indices, vertex normals, UVs. D79AD3F3 begins with the
// A79EACD2 layout (points+indices) and appends extra lists, so the same reader handles it.
var gfxPatchTypes = map[uint32]bool{0xA79EACD2: true, 0xD79AD3F3: true}

// GraphicsMesh extracts Inventor's own stored display tessellation from PmGraphicsSegment and
// concatenates every face patch into one triangle mesh — Inventor's trimmed tessellation, with
// curved faces and holes already meshed, usable as a static-body fallback for parts that don't
// rebuild parametrically (e.g. shafts, bearings). Empty when the segment has no patches.
// Duplicate boundary vertices between adjacent patches are expected; the mesh consumer welds
// them. (A part with visible sketches/work planes also renders those as small extra patches;
// they inflate the body slightly — only used when the parametric path produced nothing.)
func GraphicsMesh(d *Document) Mesh {
	var mesh Mesh
	d.walkSegment("PmGraphicsSegment", func(typ uint32, pay []byte) bool {
		if gfxPatchTypes[typ] {
			addPatch(&mesh, pay)
		}
		return true
	})
	return mesh
}

// addPatch parses one face-patch block and appends its triangles, re-basing the block's
// vertex indices onto the running vertex count. The block's first List2 is the float32 point
// array, its second the uint32 triangle-index array.
func addPatch(mesh *Mesh, b []byte) {
	offs := list2Offsets(b)
	if len(offs) < 2 {
		return
	}
	verts, ok := readF32Triples(b, offs[0])
	if !ok {
		return
	}
	idx, ok := readU32List(b, offs[1])
	if !ok || len(idx) == 0 || len(idx)%3 != 0 {
		return
	}
	base := len(mesh.Verts)
	mesh.Verts = append(mesh.Verts, verts...)
	for i := 0; i+2 < len(idx); i += 3 {
		a, bb, c := int(idx[i]), int(idx[i+1]), int(idx[i+2])
		if a < len(verts) && bb < len(verts) && c < len(verts) {
			mesh.Tris = append(mesh.Tris, [3]int{base + a, base + bb, base + c})
		}
	}
}

// list2Offsets returns the byte offsets of every List2 array marker in a block, in order. A
// List2 node begins with code 0x0002 then 0x3000 (LE) — the u32 0x30000002 — then a u32 count.
func list2Offsets(b []byte) []int {
	var offs []int
	for i := 0; i+4 <= len(b); i++ {
		if b[i] == 0x02 && b[i+1] == 0 && b[i+2] == 0 && b[i+3] == 0x30 {
			offs = append(offs, i)
		}
	}
	return offs
}

// list2Data returns (dataStart, count) for the List2 at marker offset o. A non-empty list has
// an 8-byte header (two u32s) between the count and the elements.
func list2Data(b []byte, o int) (int, int) {
	if o+8 > len(b) {
		return 0, 0
	}
	cnt := int(binary.LittleEndian.Uint32(b[o+4:]))
	ds := o + 8
	if cnt > 0 {
		ds += 8
	}
	return ds, cnt
}

// readF32Triples reads a List2 of float32 (x,y,z) triples at marker offset o (finite only).
func readF32Triples(b []byte, o int) ([][3]float64, bool) {
	ds, cnt := list2Data(b, o)
	if cnt <= 0 || ds+cnt*12 > len(b) {
		return nil, false
	}
	out := make([][3]float64, cnt)
	for i := 0; i < cnt; i++ {
		x, y, z := f32(b, ds+i*12), f32(b, ds+i*12+4), f32(b, ds+i*12+8)
		if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) {
			return nil, false
		}
		out[i] = [3]float64{x, y, z}
	}
	return out, true
}

// readU32List reads a List2 of u32 at marker offset o.
func readU32List(b []byte, o int) ([]uint32, bool) {
	ds, cnt := list2Data(b, o)
	if cnt <= 0 || ds+cnt*4 > len(b) {
		return nil, false
	}
	out := make([]uint32, cnt)
	for i := 0; i < cnt; i++ {
		out[i] = binary.LittleEndian.Uint32(b[ds+i*4:])
	}
	return out, true
}

func f32(b []byte, o int) float64 {
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(b[o:])))
}
