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

// AddTriangle appends a triangle from three explicit positions, growing Verts. Used by
// decoders (STL/3MF) that carry positions per triangle rather than an index table.
func (m *RawMesh) AddTriangle(a, b, c math.Point3) {
	i := len(m.Verts)
	m.Verts = append(m.Verts, a, b, c)
	m.Tris = append(m.Tris, [3]int{i, i + 1, i + 2})
}
