//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/viewport"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// Instanced viewport assembly (ADR-0038). Each unique component mesh is flattened ONCE in
// component-local space; its placements become a per-instance model matrix, and the native pass
// draws the mesh once per occurrence instead of uploading a transformed copy each. A part is the
// degenerate one-group, one-identity-instance case. Overlays (work planes, sketches, gizmos, the
// ground plane) are a final identity-instance group, so the whole frame goes through one path.

// instStream pairs a viewport.Mesh stream with its native stream id and the order the native pass
// concatenates the streams into the vertex/index buffers (occ, tri, line, hid, topTri, topLine).
type instStream struct {
	id     int32
	verts  func(*viewport.Mesh) []float32
	vcount func(*viewport.Mesh) int
	idx    func(*viewport.Mesh) []uint32
}

// instStreams is the six streams in the native's concatenation order (see obk_viewport_render's
// occBase/triBase/… and the kStream* ids).
var instStreams = []instStream{
	{0, func(m *viewport.Mesh) []float32 { return m.OccVerts }, func(m *viewport.Mesh) int { return m.OccVCount }, func(m *viewport.Mesh) []uint32 { return m.OccIndices }},
	{1, func(m *viewport.Mesh) []float32 { return m.TriVerts }, func(m *viewport.Mesh) int { return m.TriVCount }, func(m *viewport.Mesh) []uint32 { return m.TriIndices }},
	{2, func(m *viewport.Mesh) []float32 { return m.LineVerts }, func(m *viewport.Mesh) int { return m.LineVCount }, func(m *viewport.Mesh) []uint32 { return m.LineIndices }},
	{3, func(m *viewport.Mesh) []float32 { return m.HidVerts }, func(m *viewport.Mesh) int { return m.HidVCount }, func(m *viewport.Mesh) []uint32 { return m.HidIndices }},
	{4, func(m *viewport.Mesh) []float32 { return m.TopTriVerts }, func(m *viewport.Mesh) int { return m.TopTriVCount }, func(m *viewport.Mesh) []uint32 { return m.TopTriIndices }},
	{5, func(m *viewport.Mesh) []float32 { return m.TopLineVerts }, func(m *viewport.Mesh) int { return m.TopLineVCount }, func(m *viewport.Mesh) []uint32 { return m.TopLineIndices }},
}

// instanceBuilder accumulates the per-stream vertices/indices of every group plus the instance
// matrices and the draw records, then resolves the records' offsets to absolute positions in the
// concatenated buffers.
type instanceBuilder struct {
	streamVerts [6][]float32
	streamIdx   [6][]uint32
	mats        []float32
	recs        [][7]int32 // stream, firstIndex(local), indexCount, vertexOffset(local), firstInstance, instanceCount, biased
}

// addGroup flattens one group's source mesh into the accumulator and emits its draw records (one
// per non-empty stream; the tri stream is split into opaque + depth-biased halves at TriBiasFirst).
// transforms are the occurrence world matrices (one instance each); a part passes a single identity.
func (b *instanceBuilder) addGroup(mesh viewport.Mesh, transforms []math.Matrix4) {
	if len(transforms) == 0 {
		return
	}
	firstInstance := int32(len(b.mats) / 16)
	for _, t := range transforms {
		b.mats = append(b.mats, matrixFloats(t)...)
	}
	for _, st := range instStreams {
		b.appendStream(st, &mesh, firstInstance, int32(len(transforms)))
	}
}

// appendStream concatenates one stream of a source mesh into the accumulator and emits its draw
// record(s) — the tri stream splits into opaque + depth-biased halves at TriBiasFirst; the rest are
// one record. The record offsets are LOCAL to the stream here; finish() makes them absolute.
func (b *instanceBuilder) appendStream(st instStream, mesh *viewport.Mesh, firstInstance, instCount int32) {
	idx := st.idx(mesh)
	if len(idx) == 0 {
		return
	}
	vbase := int32(len(b.streamVerts[st.id]) / 16)
	ibase := int32(len(b.streamIdx[st.id]))
	b.streamVerts[st.id] = append(b.streamVerts[st.id], st.verts(mesh)...)
	b.streamIdx[st.id] = append(b.streamIdx[st.id], idx...)
	if st.id != 1 {
		b.recs = append(b.recs, [7]int32{st.id, ibase, int32(len(idx)), vbase, firstInstance, instCount, 0})
		return
	}
	bias := mesh.TriBiasFirst // tri: opaque [0:bias) then depth-biased [bias:len) (overlays only)
	if bias < 0 || bias > len(idx) {
		bias = len(idx)
	}
	if bias > 0 {
		b.recs = append(b.recs, [7]int32{1, ibase, int32(bias), vbase, firstInstance, instCount, 0})
	}
	if len(idx)-bias > 0 {
		b.recs = append(b.recs, [7]int32{1, ibase + int32(bias), int32(len(idx) - bias), vbase, firstInstance, instCount, 1})
	}
}

// finish concatenates the per-stream buffers in the native's order, absolutizes every record's
// firstIndex/vertexOffset to that concatenation, and returns the merged mesh + instance matrices +
// flattened int32 records. Records are sorted by stream so the native binds each pipeline once.
func (b *instanceBuilder) finish() (viewport.Mesh, []float32, []int32) {
	var vbase, ibase [6]int32 // absolute base of each stream in the concatenated vert/index buffers
	var vAcc, iAcc int32
	for _, st := range instStreams {
		vbase[st.id], ibase[st.id] = vAcc, iAcc
		vAcc += int32(len(b.streamVerts[st.id]) / 16)
		iAcc += int32(len(b.streamIdx[st.id]))
	}
	recsByStream := sortRecsByStream(b.recs)
	recs := make([]int32, 0, len(recsByStream)*7)
	for _, r := range recsByStream {
		s := r[0]
		r[1] += ibase[s] // firstIndex → absolute
		r[3] += vbase[s] // vertexOffset → absolute
		recs = append(recs, r[:]...)
	}
	return b.mergedMesh(), b.mats, recs
}

// mergedMesh assembles the six accumulated streams into one viewport.Mesh (the per-stream order
// matches the native concatenation the records' absolute offsets were computed against).
func (b *instanceBuilder) mergedMesh() viewport.Mesh {
	var m viewport.Mesh
	m.OccVerts, m.OccIndices, m.OccVCount = b.streamVerts[0], b.streamIdx[0], len(b.streamVerts[0])/16
	m.TriVerts, m.TriIndices, m.TriVCount = b.streamVerts[1], b.streamIdx[1], len(b.streamVerts[1])/16
	m.LineVerts, m.LineIndices, m.LineVCount = b.streamVerts[2], b.streamIdx[2], len(b.streamVerts[2])/16
	m.HidVerts, m.HidIndices, m.HidVCount = b.streamVerts[3], b.streamIdx[3], len(b.streamVerts[3])/16
	m.TopTriVerts, m.TopTriIndices, m.TopTriVCount = b.streamVerts[4], b.streamIdx[4], len(b.streamVerts[4])/16
	m.TopLineVerts, m.TopLineIndices, m.TopLineVCount = b.streamVerts[5], b.streamIdx[5], len(b.streamVerts[5])/16
	return m
}

// sortRecsByStream returns the records grouped by stream id (stable), so the native binds each
// stream's pipeline once. A simple bucket sort over the six fixed streams.
func sortRecsByStream(recs [][7]int32) [][7]int32 {
	out := make([][7]int32, 0, len(recs))
	for s := int32(0); s <= 5; s++ {
		for _, r := range recs {
			if r[0] == s {
				out = append(out, r)
			}
		}
	}
	return out
}

// buildInstancedFrame flattens each instance group's source mesh once (component-local) and the
// overlay items as one identity instance, returning the merged mesh + instance matrices + draw
// records for native.RenderViewport. ok is false when there is no keyable geometry, so the caller
// falls back to the legacy single-mesh path.
func buildInstancedFrame(groups []app.InstanceGroup, overlay renderer.DrawList, cam scene.Camera,
	lookup renderer.SurfaceLookup, style renderer.VisualStyle,
	decorate func(renderer.DrawList) renderer.DrawList,
) (viewport.Mesh, []float32, []int32, bool) {
	if len(groups) == 0 && len(overlay.Items) == 0 {
		return viewport.Mesh{}, nil, nil, false
	}
	var b instanceBuilder
	for _, g := range groups {
		local := renderer.BuildDrawListStyled([]*topo.Body{g.Source}, cam, ops.DefaultQuality(), lookup, style)
		if decorate != nil { // e.g. selection highlight, recolouring the source's items in place
			local = decorate(local)
		}
		b.addGroup(viewport.Flatten(local), g.Transforms)
	}
	if len(overlay.Items) > 0 {
		b.addGroup(viewport.Flatten(overlay), []math.Matrix4{math.Identity4()})
	}
	m, mats, recs := b.finish()
	return m, mats, recs, len(recs) > 0
}

// matrixFloats returns t as 16 column-major float32 — the per-instance model matrix layout the
// mesh.vert binding-1 mat4 expects.
func matrixFloats(t math.Matrix4) []float32 {
	var out [16]float32
	for col := 0; col < 4; col++ {
		for row := 0; row < 4; row++ {
			out[col*4+row] = float32(t.At(row, col))
		}
	}
	return out[:]
}
