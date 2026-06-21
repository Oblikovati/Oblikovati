//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	stdmath "math"
	"strconv"

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

// instanceBuilder accumulates the per-stream vertices/indices of every SOURCE mesh plus, per source
// region, the draw-record TEMPLATES (stream + local offsets + biased flag, with NO instance range).
// It builds the camera- and cull-independent atlas that frameAtlasCache retains; the per-frame
// instance matrices and final records are produced separately by frameAtlas.assemble, so orbiting
// no longer re-concatenates the vertex/index streams every frame — the F1 appendStream cost that
// was ~53% of orbit allocation (M34-F1b).
type instanceBuilder struct {
	streamVerts [6][]float32
	streamIdx   [6][]uint32
	recs        [][5]int32 // template: stream, firstIndex(local), indexCount, vertexOffset(local), biased
	regions     []atlasRegion
}

// atlasRegion maps one source body (nil marks the overlay) to the contiguous span recs[start:end]
// that draws it, so assemble can emit those records with a frame's instance range when the source
// is visible.
type atlasRegion struct {
	source     *topo.Body
	start, end int
}

// addSource concatenates a source mesh's streams into the atlas and records its draw-record
// templates as one region; transforms are NOT taken here — they are per-frame (assemble).
func (b *instanceBuilder) addSource(src *topo.Body, mesh viewport.Mesh) {
	start := len(b.recs)
	for _, st := range instStreams {
		b.appendStream(st, &mesh)
	}
	b.regions = append(b.regions, atlasRegion{source: src, start: start, end: len(b.recs)})
}

// appendStream concatenates one stream of a source mesh into the atlas and emits its record
// template(s) — the tri stream splits into opaque + depth-biased halves at TriBiasFirst; the rest
// are one record. Offsets are LOCAL to the stream here; finishAtlas makes them absolute.
func (b *instanceBuilder) appendStream(st instStream, mesh *viewport.Mesh) {
	idx := st.idx(mesh)
	if len(idx) == 0 {
		return
	}
	vbase := int32(len(b.streamVerts[st.id]) / 16)
	ibase := int32(len(b.streamIdx[st.id]))
	b.streamVerts[st.id] = append(b.streamVerts[st.id], st.verts(mesh)...)
	b.streamIdx[st.id] = append(b.streamIdx[st.id], idx...)
	if st.id != 1 {
		b.recs = append(b.recs, [5]int32{st.id, ibase, int32(len(idx)), vbase, 0})
		return
	}
	bias := mesh.TriBiasFirst // tri: opaque [0:bias) then depth-biased [bias:len) (overlays only)
	if bias < 0 || bias > len(idx) {
		bias = len(idx)
	}
	if bias > 0 {
		b.recs = append(b.recs, [5]int32{1, ibase, int32(bias), vbase, 0})
	}
	if len(idx)-bias > 0 {
		b.recs = append(b.recs, [5]int32{1, ibase + int32(bias), int32(len(idx) - bias), vbase, 1})
	}
}

// finishAtlas concatenates the per-stream buffers in the native's order, absolutizes every record
// template's firstIndex/vertexOffset to that concatenation, and returns the retained atlas (merged
// mesh + absolute templates + regions). key identifies the geometry+overlay state it was built for.
func (b *instanceBuilder) finishAtlas(key string) frameAtlas {
	var vbase, ibase [6]int32 // absolute base of each stream in the concatenated vert/index buffers
	var vAcc, iAcc int32
	for _, st := range instStreams {
		vbase[st.id], ibase[st.id] = vAcc, iAcc
		vAcc += int32(len(b.streamVerts[st.id]) / 16)
		iAcc += int32(len(b.streamIdx[st.id]))
	}
	for i := range b.recs {
		s := b.recs[i][0]
		b.recs[i][1] += ibase[s] // firstIndex → absolute
		b.recs[i][3] += vbase[s] // vertexOffset → absolute
	}
	return frameAtlas{key: key, mesh: b.mergedMesh(), recs: b.recs, regions: b.regions}
}

// frameAtlas is the retained, camera/cull-independent merged geometry of every source mesh plus the
// overlay: the concatenated streams, the absolute draw-record templates, and the per-source regions.
// Held by frameAtlasCache and rebuilt only when the geometry/style/selection (sourceKey) or the
// overlay changes — not when the camera orbits (M34-F1b).
type frameAtlas struct {
	key     string
	mesh    viewport.Mesh
	recs    [][5]int32 // absolute templates: stream, firstIndex, indexCount, vertexOffset, biased
	regions []atlasRegion
}

// assemble builds the per-frame instance matrices and draw records from the retained atlas and the
// frame's visible (frustum-culled) instances: each region whose source is visible contributes its
// transforms to mats and its templates as records carrying that instance range. The overlay region
// (nil source) always draws as one identity instance. Records are stream-sorted so native binds
// each pipeline once. This is all the per-frame work now — the heavy streams come from the cache.
func (a *frameAtlas) assemble(visible map[*topo.Body][]math.Matrix4) ([]float32, []int32) {
	var mats []float32
	var recs [][7]int32
	for _, region := range a.regions {
		tfs := regionTransforms(region, visible)
		if len(tfs) == 0 {
			continue
		}
		first := int32(len(mats) / 16)
		for _, t := range tfs {
			mats = append(mats, matrixFloats(t)...)
		}
		for _, tmpl := range a.recs[region.start:region.end] {
			recs = append(recs, [7]int32{tmpl[0], tmpl[1], tmpl[2], tmpl[3], first, int32(len(tfs)), tmpl[4]})
		}
	}
	return mats, flattenRecs(sortRecsByStream(recs))
}

// regionTransforms returns the world matrices to draw a region this frame: the overlay (nil source)
// is a single identity instance; a source region uses its visible (culled) transforms, or none.
func regionTransforms(region atlasRegion, visible map[*topo.Body][]math.Matrix4) []math.Matrix4 {
	if region.source == nil {
		return identityInstance
	}
	return visible[region.source]
}

// identityInstance is the overlay's single, fixed instance (work planes/sketches/gizmos live in
// world space and are placed by the view-projection, not a model matrix).
var identityInstance = []math.Matrix4{math.Identity4()}

// flattenRecs packs the stream-sorted [7]int32 records into the flat []int32 native expects.
func flattenRecs(recs [][7]int32) []int32 {
	out := make([]int32, 0, len(recs)*7)
	for _, r := range recs {
		out = append(out, r[:]...)
	}
	return out
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

// buildInstancedFrame returns the merged mesh + per-frame instance matrices + draw records for
// native.RenderViewport. The merged mesh comes from a retained atlas built over ALL source meshes
// (allGroups) plus the overlay — rebuilt only when the geometry/overlay changes, not on orbit
// (M34-F1b) — while the matrices/records are assembled each frame from the frustum-culled instances
// (culledGroups, M34-F1). ok is false when there is nothing to draw, so the caller falls back to
// the legacy single-mesh path.
func buildInstancedFrame(allGroups, culledGroups []app.InstanceGroup, overlay renderer.DrawList, cam scene.Camera,
	lookup renderer.SurfaceLookup, style renderer.VisualStyle,
	decorate func(renderer.DrawList) renderer.DrawList, sourceKey string,
) (viewport.Mesh, []float32, []int32, bool) {
	if len(allGroups) == 0 && len(overlay.Items) == 0 {
		return viewport.Mesh{}, nil, nil, false
	}
	atlas := cachedFrameAtlas(allGroups, overlay, cam, lookup, style, decorate, sourceKey)
	visible := make(map[*topo.Body][]math.Matrix4, len(culledGroups))
	for _, g := range culledGroups {
		visible[g.Source] = g.Transforms
	}
	mats, recs := atlas.assemble(visible)
	return atlas.mesh, mats, recs, len(recs) > 0
}

// frameAtlasCache retains the last built atlas. The render loop is single-threaded, so a single
// package-level entry is safe; alternating between two viewports with different content simply
// rebuilds (degrading to the pre-F1b cost, never incorrectly).
var frameAtlasCache frameAtlas

// cachedFrameAtlas returns the atlas for the given sources + overlay, rebuilding it only when the
// source signature (sourceKey, which bumps on any geometry/style/selection/placement change) or the
// overlay content changes. The overlay is flattened every frame (it is small) so its hash can key
// the cache; on a hit the expensive per-source concatenation is skipped entirely.
func cachedFrameAtlas(allGroups []app.InstanceGroup, overlay renderer.DrawList, cam scene.Camera,
	lookup renderer.SurfaceLookup, style renderer.VisualStyle,
	decorate func(renderer.DrawList) renderer.DrawList, sourceKey string,
) frameAtlas {
	overlayMesh := viewport.Flatten(overlay)
	key := sourceKey + "|ov:" + strconv.FormatUint(overlayHash(overlayMesh), 16)
	if frameAtlasCache.key == key && key != "" {
		return frameAtlasCache
	}
	var b instanceBuilder
	for _, g := range allGroups {
		// The per-source tessellate+flatten is cached by sourceMeshCache; addSource only concatenates
		// the cached streams into the atlas, which is itself retained across frames by this cache.
		b.addSource(g.Source, cachedSourceMesh(g.Source, cam, lookup, style, decorate, sourceKey))
	}
	if len(overlay.Items) > 0 {
		b.addSource(nil, overlayMesh) // the overlay region: one identity instance in assemble
	}
	frameAtlasCache = b.finishAtlas(key)
	return frameAtlasCache
}

// overlayHash is an order-sensitive FNV-1a digest of the overlay mesh's streams, so the atlas cache
// rebuilds when an overlay (work plane, sketch, gizmo, ground) appears, moves or disappears, but
// holds while it is unchanged during an orbit.
func overlayHash(m viewport.Mesh) uint64 {
	h := fnv.New64a()
	for _, s := range [][]float32{m.TriVerts, m.OccVerts, m.LineVerts, m.HidVerts, m.TopTriVerts, m.TopLineVerts} {
		hashFloat32s(h, s)
	}
	for _, s := range [][]uint32{m.TriIndices, m.OccIndices, m.LineIndices, m.HidIndices, m.TopTriIndices, m.TopLineIndices} {
		hashUint32s(h, s)
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(m.TriBiasFirst))
	_, _ = h.Write(b[:])
	return h.Sum64()
}

func hashFloat32s(h hash.Hash64, xs []float32) {
	var b [4]byte
	for _, x := range xs {
		binary.LittleEndian.PutUint32(b[:], stdmath.Float32bits(x))
		_, _ = h.Write(b[:])
	}
}

func hashUint32s(h hash.Hash64, xs []uint32) {
	var b [4]byte
	for _, x := range xs {
		binary.LittleEndian.PutUint32(b[:], x)
		_, _ = h.Write(b[:])
	}
}

// sourceMeshCache memoises the flattened, styled mesh of each unique component body, keyed on the
// geometry/style/selection signature (sourceKey). Flattening a source tessellates it, extracts its
// edges and packs the vertex streams — tens of ms for a dense part — and is camera-independent, so
// without this an assembly re-did it for the one shared mesh every frame. Cleared lazily: an entry
// whose key no longer matches is rebuilt (a geometry edit, placement, style or selection change
// bumps the key). The render loop is single-threaded, so a package map is safe (like bodyGeometryCache).
var sourceMeshCache = map[*topo.Body]struct {
	key  string
	mesh viewport.Mesh
}{}

// instancedSourceKey signs everything that changes a cached source mesh: the active model's
// geometry version (bumped on any edit/recompute/placement), the visual style, and the selection
// (the highlight recolours a source's items). It is camera-independent, so it is stable while
// orbiting — the cache then holds and the dense source is not re-tessellated. Empty (no keyable
// model) disables the source cache for that frame.
func instancedSourceKey(s *app.Session) string {
	ver, ok := activeModelGeometryVersion(s)
	if !ok {
		return ""
	}
	ec := displayEdgeColor(s) // M16-F07: edge-color override is baked into the source mesh
	return ver + "|" + strconv.Itoa(int(s.VisualStyle())) + "|" + fmt.Sprintf("%v", s.Selection().First()) +
		"|" + fmt.Sprintf("%.3f", ec[0]+ec[1]*2+ec[2]*3)
}

// cachedSourceMesh returns g's flattened source mesh, rebuilding only when sourceKey changed.
func cachedSourceMesh(src *topo.Body, cam scene.Camera, lookup renderer.SurfaceLookup,
	style renderer.VisualStyle, decorate func(renderer.DrawList) renderer.DrawList, sourceKey string,
) viewport.Mesh {
	if c, ok := sourceMeshCache[src]; ok && c.key == sourceKey {
		return c.mesh
	}
	local := renderer.BuildDrawListStyled([]*topo.Body{src}, cam, ops.DefaultQuality(), lookup, style)
	if decorate != nil { // selection highlight recolours the source's items (in the key)
		local = decorate(local)
	}
	m := viewport.Flatten(local)
	sourceMeshCache[src] = struct {
		key  string
		mesh viewport.Mesh
	}{sourceKey, m}
	return m
}

// instancedBounds is the world-space bounding box of every instance, computed from each source's
// range box transformed by its occurrence matrices — O(instances) and NO tessellation, so a 10k-copy
// assembly frames its shadow/ground without building a world-body mesh. ok is false when empty.
func instancedBounds(groups []app.InstanceGroup) (min, max [3]float32, ok bool) {
	min = [3]float32{stdmath.MaxFloat32, stdmath.MaxFloat32, stdmath.MaxFloat32}
	max = [3]float32{-stdmath.MaxFloat32, -stdmath.MaxFloat32, -stdmath.MaxFloat32}
	for _, g := range groups {
		bx := g.Source.RangeBox()
		corners := [8]math.Point3{
			math.P3(bx.Min.X, bx.Min.Y, bx.Min.Z), math.P3(bx.Max.X, bx.Min.Y, bx.Min.Z),
			math.P3(bx.Min.X, bx.Max.Y, bx.Min.Z), math.P3(bx.Max.X, bx.Max.Y, bx.Min.Z),
			math.P3(bx.Min.X, bx.Min.Y, bx.Max.Z), math.P3(bx.Max.X, bx.Min.Y, bx.Max.Z),
			math.P3(bx.Min.X, bx.Max.Y, bx.Max.Z), math.P3(bx.Max.X, bx.Max.Y, bx.Max.Z),
		}
		for _, t := range g.Transforms {
			for _, c := range corners {
				p := t.TransformPoint(c)
				widenBounds(&min, &max, float32(p.X), float32(p.Y), float32(p.Z))
				ok = true
			}
		}
	}
	return min, max, ok
}

// widenBounds expands the running min/max to include (x,y,z).
func widenBounds(min, max *[3]float32, x, y, z float32) {
	for i, c := range [3]float32{x, y, z} {
		if c < min[i] {
			min[i] = c
		}
		if c > max[i] {
			max[i] = c
		}
	}
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
