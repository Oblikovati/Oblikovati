// SPDX-License-Identifier: GPL-2.0-only

// Package viewport adapts the renderer's geometry-as-data (renderer.DrawList) into the
// interleaved vertex/index arrays the Vulkan viewport pipeline consumes. It is pure Go
// with no cgo, so the flattening is unit-tested headlessly; the native package uploads
// the result to the GPU.
package viewport

import (
	"oblikovati.org/math"
	"oblikovati.org/renderer"
)

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
	TriVerts   []float32
	TriVCount  int
	TriIndices []uint32
	// TriBiasFirst is the index into TriIndices at which the depth-biased triangles begin
	// (Biased items — work-plane/ground-plane fills — are appended after the opaque triangles).
	// The native pass draws [0,TriBiasFirst) at zero bias and [TriBiasFirst,len) with a small
	// bias so coplanar reference overlays do not z-fight solid geometry.
	TriBiasFirst  int
	OccVerts      []float32
	OccVCount     int
	OccIndices    []uint32
	LineVerts     []float32
	LineVCount    int
	LineIndices   []uint32
	HidVerts      []float32
	HidVCount     int
	HidIndices    []uint32
	TopTriVerts   []float32
	TopTriVCount  int
	TopTriIndices []uint32
	// TopTriSolidFirst is the index into TopTriIndices at which the OPAQUE SOLID on-top
	// triangles begin (client-graphics glyphs — fixed-support cubes, load arrows — appended
	// after the flat/translucent on-top triangles). The native pass draws [0,TopTriSolidFirst)
	// flat with the depth test disabled (the burn-through overlay), then clears depth and draws
	// [TopTriSolidFirst,len) depth-tested + lit so each solid self-occludes instead of rendering
	// as a scatter of arbitrary faces — issue #1489.
	TopTriSolidFirst int
	TopLineVerts     []float32
	TopLineVCount    int
	TopLineIndices   []uint32
	// The wide-line streams carry stroked lines (DrawItem.Width > 1) as TRIANGLES: each segment
	// becomes a quad the vertex shader expands in screen space, so the stroke stays a constant
	// pixel width at any zoom. They are separate streams because they need triangle topology and
	// the expanding shader, while hairlines keep the cheaper line pipeline. WideLine is
	// depth-tested; TopWideLine ignores depth, mirroring Line/TopLine. #2015.
	WideLineVerts      []float32
	WideLineVCount     int
	WideLineIndices    []uint32
	TopWideLineVerts   []float32
	TopWideLineVCount  int
	TopWideLineIndices []uint32
}

// Flatten splits a draw list into the viewport streams, interleaving each vertex as
// [pos.xyz, normal.xyz, color.rgba, metallic, roughness, emissive.rgb, mode] and rebasing
// indices per item. DrawItem.OnTop routes triangles/lines to the depth-disabled on-top
// streams (drawn over the model); otherwise triangles route to the occluder stream when
// Occluder is set, and lines to the hidden stream when Hidden is set.
func Flatten(list renderer.DrawList) Mesh {
	var m Mesh
	var biased, solidOnTop []renderer.DrawItem
	for _, item := range list.Items {
		// Opaque, normal-bearing on-top triangle meshes (client-graphics glyphs) are held to the
		// tail of the on-top stream so the native pass can draw them depth-tested + lit; the flat
		// on-top path (translucent ghosts/highlights, normal-less flood plots, lines) is untouched.
		if isSolidOnTopTriangle(item) {
			solidOnTop = append(solidOnTop, item)
			continue
		}
		biased = routeItem(&m, item, biased)
	}
	// Biased reference overlays go at the tail of the triangle stream so the native pass can draw
	// them with a depth bias (after the opaque triangles at zero bias).
	m.TriBiasFirst = len(m.TriIndices)
	for _, item := range biased {
		m.TriVCount = appendItem(&m.TriVerts, &m.TriIndices, m.TriVCount, item)
	}
	// Solid on-top glyphs go at the tail of the on-top triangle stream (after the flat overlays),
	// drawn depth-cleared + depth-tested + lit so each reads as a real solid (#1489).
	m.TopTriSolidFirst = len(m.TopTriIndices)
	for _, item := range solidOnTop {
		m.TopTriVCount = appendItem(&m.TopTriVerts, &m.TopTriIndices, m.TopTriVCount, item)
	}
	return m
}

// isSolidOnTopTriangle reports whether item is an OPAQUE, normal-bearing on-top triangle mesh —
// a client-graphics solid glyph (support cube, load arrow) that must self-occlude to read as a
// 3D solid AND sit on top of every other overlay (#1489). Translucent on-top overlays
// (feature-preview ghosts, face highlights) and normal-less flat overlays (heatmap flood plots)
// deliberately stay on the flat burn-through path: a flood plot is a flat data skin that must NOT
// be lit and must stay UNDER the glyphs, and depth-writing a translucent overlay changes its look.
func isSolidOnTopTriangle(item renderer.DrawItem) bool {
	return item.OnTop && item.Primitive == renderer.Triangles &&
		len(item.Normals) > 0 && isOpaqueItem(item)
}

// isOpaqueItem reports whether a draw item is fully opaque: an explicit fractional Opacity
// (0,1) marks translucency, otherwise the item's broadcast color alpha decides.
func isOpaqueItem(item renderer.DrawItem) bool {
	if item.Opacity > 0 && item.Opacity < 1 {
		return false
	}
	return item.Color[3] >= 1
}

// routeItem appends one draw item to the stream its flags select, returning the (possibly grown)
// list of biased items, which are held back and appended to the triangle stream's tail by Flatten.
func routeItem(m *Mesh, item renderer.DrawItem, biased []renderer.DrawItem) []renderer.DrawItem {
	if item.Primitive != renderer.Triangles {
		routeLineItem(m, item)
		return biased
	}
	switch {
	case item.OnTop:
		m.TopTriVCount = appendItem(&m.TopTriVerts, &m.TopTriIndices, m.TopTriVCount, item)
	case item.Occluder:
		m.OccVCount = appendItem(&m.OccVerts, &m.OccIndices, m.OccVCount, item)
	case item.Biased:
		biased = append(biased, item) // appended after the opaque triangles (depth-biased tail)
	default:
		m.TriVCount = appendItem(&m.TriVerts, &m.TriIndices, m.TriVCount, item)
	}
	return biased
}

// routeLineItem picks the lane for a line item. OnTop is matched first, keeping the precedence the
// single switch had before the stroked lanes existed. Hidden then beats width: the hidden lane's
// whole purpose is its reversed depth test and there is no stroked equivalent, so letting width win
// would draw an occluded edge as a plainly visible one — a wrong picture is worse than a thin one.
// Below those, a stroked line must take the expanding lane, since falling through to the hairline
// lane would silently drop its width.
func routeLineItem(m *Mesh, item renderer.DrawItem) {
	switch {
	case item.OnTop && item.IsWideLine():
		m.TopWideLineVCount = appendWideLineItem(&m.TopWideLineVerts, &m.TopWideLineIndices, m.TopWideLineVCount, item)
	case item.OnTop:
		m.TopLineVCount = appendItem(&m.TopLineVerts, &m.TopLineIndices, m.TopLineVCount, item)
	case item.Hidden:
		m.HidVCount = appendItem(&m.HidVerts, &m.HidIndices, m.HidVCount, item)
	case item.IsWideLine():
		m.WideLineVCount = appendWideLineItem(&m.WideLineVerts, &m.WideLineIndices, m.WideLineVCount, item)
	default:
		m.LineVCount = appendItem(&m.LineVerts, &m.LineIndices, m.LineVCount, item)
	}
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

// Wide-line encoding. The stroke must be a constant width in PIXELS, so the quad corners can only
// be placed once the camera is known — and they must not be placed on the CPU: the merged geometry
// is content-keyed and its GPU upload is deliberately skipped while that key holds across an orbit
// (#1422), so camera-dependent vertices would either go stale or force a full re-upload every
// frame. The expansion therefore happens in the vertex shader, and each vertex carries what the
// shader needs to do it.
//
// The vertex layout is the standard 16-float mesh vertex, with three slots repurposed for the
// wide-line streams only — they are dead there, since a line is drawn flat/unlit and takes no
// material. This function is the only writer of that encoding and shaders/wideline.vert the only
// reader; the two must be changed together.
//
//	normal.xyz ← the segment's OTHER endpoint, in the same space as position
//	metallic   ← the side of the stroke this corner sits on (+1 / -1)
//	roughness  ← the stroke width in pixels

// appendWideLineItem expands each of the item's line segments into a quad (4 vertices, 6 indices)
// and appends it, returning the new running vertex count. The item's Indices are read in pairs,
// matching the line list the hairline path would have drawn.
//
// Each corner records its own endpoint, the opposite endpoint and a side sign. The shader derives
// the screen-space direction from the two endpoints, so the OTHER end's corners must flip their
// side to offset the same way round — otherwise the quad comes out as a bow tie.
func appendWideLineItem(verts *[]float32, idx *[]uint32, base int, item renderer.DrawItem) int {
	n := base
	for i := 0; i+1 < len(item.Indices); i += 2 {
		a, b := item.Indices[i], item.Indices[i+1]
		if a >= len(item.Positions) || b >= len(item.Positions) {
			continue // a malformed item must not panic the render loop
		}
		pa, pb := item.Positions[a], item.Positions[b]
		ca, cb := vertexColor(item, a), vertexColor(item, b)
		appendWideLineVertex(verts, pa, pb, ca, +1, item)
		appendWideLineVertex(verts, pa, pb, ca, -1, item)
		appendWideLineVertex(verts, pb, pa, cb, +1, item) // opposite end ⇒ mirrored side
		appendWideLineVertex(verts, pb, pa, cb, -1, item)
		*idx = append(*idx, uint32(n), uint32(n+1), uint32(n+2), uint32(n), uint32(n+2), uint32(n+3))
		n += 4
	}
	return n
}

// appendWideLineVertex writes one expanded corner in the layout documented above.
func appendWideLineVertex(verts *[]float32, at, other math.Point3, c [4]float32, side float32, item renderer.DrawItem) {
	*verts = append(*verts,
		float32(at.X), float32(at.Y), float32(at.Z),
		float32(other.X), float32(other.Y), float32(other.Z), // the other endpoint
		c[0], c[1], c[2], c[3],
		side,       // which side of the stroke
		item.Width, // stroke width in pixels
		0, 0, 0,    // emissive: unused by a flat stroke
		float32(item.Shading))
}

// vertexColor returns vertex i's color: the per-vertex DrawItem.Colors entry when present
// (client-graphics heatmaps / per-vertex binding), else the item's single broadcast Color.
func vertexColor(item renderer.DrawItem, i int) [4]float32 {
	if i < len(item.Colors) {
		return item.Colors[i]
	}
	return item.Color
}
