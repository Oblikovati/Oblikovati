// SPDX-License-Identifier: GPL-2.0-only

// Package viewport adapts the renderer's geometry-as-data (renderer.DrawList) into the
// interleaved vertex/index arrays the Vulkan viewport pipeline consumes. It is pure Go
// with no cgo, so the flattening is unit-tested headlessly; the native package uploads
// the result to the GPU.
package viewport

import "oblikovati/renderer"

// VertexFloats is the per-vertex layout the mesh pipeline expects: position (xyz),
// normal (xyz), color/albedo (rgba), metallic, roughness, emissive (rgb), and the shading
// mode (float-encoded renderer.Shading). The per-material PBR fields + mode travel per vertex
// so model bodies shade with their appearance (PBR/NPR) while UI overlays stay flat-lit, all
// in one draw call (ADR-0023 §2).
const VertexFloats = 16

// Mesh is the flattened, GPU-ready geometry split into the viewport streams: shaded
// triangles, depth-only occluder triangles (hidden-line modes), solid edge lines, dashed
// hidden-edge lines (reversed depth test), and the two on-top streams (triangles and lines
// drawn with the depth test disabled — client-graphics overlay/burn-through, PBI-067).
// Indices are 0-based within each stream's own vertex array (the pipeline applies the
// vertex offset).
type Mesh struct {
	TriVerts       []float32
	TriVCount      int
	TriIndices     []uint32
	OccVerts       []float32
	OccVCount      int
	OccIndices     []uint32
	LineVerts      []float32
	LineVCount     int
	LineIndices    []uint32
	HidVerts       []float32
	HidVCount      int
	HidIndices     []uint32
	TopTriVerts    []float32
	TopTriVCount   int
	TopTriIndices  []uint32
	TopLineVerts   []float32
	TopLineVCount  int
	TopLineIndices []uint32
}

// Flatten splits a draw list into the viewport streams, interleaving each vertex as
// [pos.xyz, normal.xyz, color.rgba, metallic, roughness, emissive.rgb, mode] and rebasing
// indices per item. DrawItem.OnTop routes triangles/lines to the depth-disabled on-top
// streams (drawn over the model); otherwise triangles route to the occluder stream when
// Occluder is set, and lines to the hidden stream when Hidden is set.
func Flatten(list renderer.DrawList) Mesh {
	var m Mesh
	for _, item := range list.Items {
		switch {
		case item.OnTop && item.Primitive == renderer.Triangles:
			m.TopTriVCount = appendItem(&m.TopTriVerts, &m.TopTriIndices, m.TopTriVCount, item)
		case item.OnTop:
			m.TopLineVCount = appendItem(&m.TopLineVerts, &m.TopLineIndices, m.TopLineVCount, item)
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
		c := vertexColor(item, i)
		*verts = append(*verts,
			float32(p.X), float32(p.Y), float32(p.Z),
			n[0], n[1], n[2],
			c[0], c[1], c[2], c[3],
			item.Metallic, item.Roughness,
			item.Emissive[0], item.Emissive[1], item.Emissive[2],
			float32(item.Shading))
	}
	for _, i := range item.Indices {
		*idx = append(*idx, uint32(base+i))
	}
	return base + len(item.Positions)
}

// vertexColor returns vertex i's color: the per-vertex DrawItem.Colors entry when present
// (client-graphics heatmaps / per-vertex binding), else the item's single broadcast Color.
func vertexColor(item renderer.DrawItem, i int) [4]float32 {
	if i < len(item.Colors) {
		return item.Colors[i]
	}
	return item.Color
}
