// SPDX-License-Identifier: GPL-2.0-only

// Package viewport adapts the renderer's geometry-as-data (renderer.DrawList) into the
// interleaved vertex/index arrays the Vulkan viewport pipeline consumes. It is pure Go
// with no cgo, so the flattening is unit-tested headlessly; the native package uploads
// the result to the GPU.
package viewport

import "github.com/Oblikovati/oblikovati/renderer"

// VertexFloats is the per-vertex layout the mesh pipeline expects: position (xyz),
// normal (xyz), color (rgba).
const VertexFloats = 10

// Mesh is the flattened, GPU-ready geometry split by primitive. Triangle and line
// indices are 0-based within their own vertex arrays (the pipeline applies the vertex
// offset for lines).
type Mesh struct {
	TriVerts    []float32
	TriVCount   int
	TriIndices  []uint32
	LineVerts   []float32
	LineVCount  int
	LineIndices []uint32
}

// Flatten splits a draw list into triangle and line vertex/index arrays, interleaving
// each vertex as [pos.xyz, normal.xyz, color.rgba] and rebasing indices per item.
func Flatten(list renderer.DrawList) Mesh {
	var m Mesh
	for _, item := range list.Items {
		if item.Primitive == renderer.Triangles {
			m.TriVCount = appendItem(&m.TriVerts, &m.TriIndices, m.TriVCount, item)
		} else {
			m.LineVCount = appendItem(&m.LineVerts, &m.LineIndices, m.LineVCount, item)
		}
	}
	return m
}

// appendItem appends one item's interleaved vertices and rebased indices, returning
// the new running vertex count for that primitive stream.
func appendItem(verts *[]float32, idx *[]uint32, base int, item renderer.DrawItem) int {
	for i, p := range item.Positions {
		var n [3]float32
		if i < len(item.Normals) {
			n = [3]float32{
				float32(item.Normals[i].X), float32(item.Normals[i].Y), float32(item.Normals[i].Z),
			}
		}
		*verts = append(*verts,
			float32(p.X), float32(p.Y), float32(p.Z),
			n[0], n[1], n[2],
			item.Color[0], item.Color[1], item.Color[2], item.Color[3])
	}
	for _, i := range item.Indices {
		*idx = append(*idx, uint32(base+i))
	}
	return base + len(item.Positions)
}
