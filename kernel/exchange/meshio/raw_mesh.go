// SPDX-License-Identifier: GPL-2.0-only

// Package meshio is the kernel-side mesh-exchange engine (M17-F04): it decodes/encodes
// faceted-mesh files (STL, OBJ, 3MF) and turns a triangle soup into a real B-rep body
// via the sub-D cage→B-rep weld (kernel/subd), so an imported mesh is a faceted SOLID
// downstream features can operate on — not an inert display mesh. It owns thin
// per-format reader/writer types behind exchange.BodyImporter/BodyExporter (the kernel
// seam), mirroring kernel/exchange/step. Pure Go, headless (ADR-0008).
package meshio

import "oblikovati.org/math"

// RawMesh is a decoded triangle soup straight from a file, BEFORE welding: a flat list
// of triangles each naming three vertex indices into Verts. STL/3MF emit per-triangle
// vertices (no sharing); OBJ shares them. Welding (Weld) collapses coincident vertices
// so faces share topology — the precondition for subd.ToBody's closed-cage detection.
type RawMesh struct {
	Verts []math.Point3
	Tris  [][3]int
}

// TriangleCount returns the number of triangles in the soup.
func (m RawMesh) TriangleCount() int { return len(m.Tris) }

// Reserve grows capacity for an expected additional triangle count so a decoder that knows
// its size up front (binary STL's header count, a 3MF mesh's triangle list) fills the soup
// without the repeated slice reallocation that dominates allocation churn on a dense import
// (#1765). STL/3MF store three unshared vertices per triangle, so this reserves 3× the
// triangles in Verts. It only ever grows capacity; a non-positive count is a no-op.
func (m *RawMesh) Reserve(triangles int) {
	if triangles <= 0 {
		return
	}
	if need := len(m.Verts) + triangles*3; cap(m.Verts) < need {
		m.Verts = append(make([]math.Point3, 0, need), m.Verts...)
	}
	if need := len(m.Tris) + triangles; cap(m.Tris) < need {
		m.Tris = append(make([][3]int, 0, need), m.Tris...)
	}
}

// AddTriangle appends a triangle from three explicit positions, growing Verts. Used by
// decoders (STL/3MF) that carry positions per triangle rather than an index table.
func (m *RawMesh) AddTriangle(a, b, c math.Point3) {
	i := len(m.Verts)
	m.Verts = append(m.Verts, a, b, c)
	m.Tris = append(m.Tris, [3]int{i, i + 1, i + 2})
}
