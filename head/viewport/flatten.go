// SPDX-License-Identifier: GPL-2.0-only

// Package viewport adapts the renderer's geometry-as-data (renderer.DrawList) into the
// interleaved vertex/index arrays the Vulkan viewport pipeline consumes. It is pure Go
// with no cgo, so the flattening is unit-tested headlessly; the native package uploads
// the result to the GPU.
package viewport

import "github.com/Oblikovati/oblikovati/renderer"

// VertexFloats is the per-vertex layout the mesh pipeline expects: position (xyz),
// normal (xyz), color/albedo (rgba), metallic, roughness, emissive (rgb), and the shading
// mode (float-encoded renderer.Shading). The per-material PBR fields + mode travel per vertex
// so model bodies shade with their appearance (PBR/NPR) while UI overlays stay flat-lit, all
// in one draw call (ADR-0023 §2).
const VertexFloats = 16

// Mesh is the flattened, GPU-ready geometry split into the four viewport streams: shaded
// triangles, depth-only occluder triangles (hidden-line modes), solid edge lines, and dashed
// hidden-edge lines (reversed depth test). Indices are 0-based within each stream's own vertex
// array (the pipeline applies the vertex offset).
type Mesh struct {
	TriVerts    []float32
	TriVCount   int
	TriIndices  []uint32
	OccVerts    []float32
	OccVCount   int
	OccIndices  []uint32
	LineVerts   []float32
	LineVCount  int
	LineIndices []uint32
	HidVerts    []float32
	HidVCount   int
	HidIndices  []uint32
}

// Flatten splits a draw list into the four viewport streams, interleaving each vertex as
// [pos.xyz, normal.xyz, color.rgba, metallic, roughness, emissive.rgb, mode] and rebasing
// indices per item. Triangles route to the occluder stream when DrawItem.Occluder is set;
// lines route to the hidden stream when DrawItem.Hidden is set.
func Flatten(list renderer.DrawList) Mesh {
	var m Mesh
	for _, item := range list.Items {
		switch {
		case item.Primitive == renderer.Triangles && item.Occluder:
			m.OccVCount = appendItem(&m.OccVerts, &m.OccIndices, m.OccVCount, item)
		case item.Primitive == renderer.Triangles:
			m.TriVCount = appendItem(&m.TriVerts, &m.TriIndices, m.TriVCount, item)
		case item.Hidden:
			m.HidVCount = appendItem(&m.HidVerts, &m.HidIndices, m.HidVCount, item)
		default:
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
			item.Color[0], item.Color[1], item.Color[2], item.Color[3],
			item.Metallic, item.Roughness,
			item.Emissive[0], item.Emissive[1], item.Emissive[2],
			float32(item.Shading))
	}
	for _, i := range item.Indices {
		*idx = append(*idx, uint32(base+i))
	}
	return base + len(item.Positions)
}
