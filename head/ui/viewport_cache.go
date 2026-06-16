// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// bodyGeometryCache memoises the tessellated, styled body draw list. Without it the viewport reran
// ops.TessellateBody (plus smooth-shading + visible-edge extraction) for every body EVERY frame —
// even while merely orbiting — which pegged a CPU core on a model with many curved faces. The
// tessellation is camera-independent, so the cache is keyed on the model geometry version, the
// visual style, and the visible body set; all of those bump the part's version via MarkChanged on
// any geometry / appearance / visibility change, so the list rebuilds exactly when it must.
//
// The render loop is single-threaded, so a package-level cache is safe. (A body that was entirely
// behind the camera when the cache was built keeps the build-time frustum-cull decision until the
// next geometry change — acceptable for clustered CAD models; the camera-dependent dash/cull are
// otherwise negligible next to the tessellation this avoids.)
var bodyGeometryCache struct {
	key  string
	list renderer.DrawList
}

// cachedBodyDrawList returns the styled body geometry, rebuilding only when the key changes.
// Callers receive a shallow COPY of the item slice so selection-highlight (which recolours items in
// place) and the overlays (which append) never corrupt the cached list.
func cachedBodyDrawList(s *app.Session, cam scene.Camera) renderer.DrawList {
	build := func() renderer.DrawList {
		renderer.SetEdgeColor(displayEdgeColor(s)) // M16-F07: the document's display-settings edge color
		if on, perTri := s.MeshColors(); on {      // each face/triangle a distinct color (viewport.setMeshColors)
			return renderer.BuildDrawListMeshColors(activeBodies(s), cam, ops.DefaultQuality(), perTri)
		}
		return renderer.BuildDrawListStyled(activeBodies(s), cam, ops.DefaultQuality(), s.SurfaceLookup(), s.VisualStyle())
	}
	key := bodyGeometryKey(s)
	if key == "" {
		return build() // no active part to key on; don't cache
	}
	if key != bodyGeometryCache.key {
		bodyGeometryCache.key = key
		bodyGeometryCache.list = build()
	}
	return renderer.DrawList{Items: append([]renderer.DrawItem(nil), bodyGeometryCache.list.Items...)}
}

// bodyGeometryKey identifies the cached geometry: the active model's geometry version + the visual
// style + the visible body ids. A part AND an assembly both report a geometry version (bumped on any
// geometry/appearance edit, recompute, or — for an assembly — occurrence change), so the cache holds
// for assemblies too; a part-only version here returned "" for an assembly, defeating the cache and
// re-tessellating every placed body every frame (the unstable-viewport bug when a component is placed).
func bodyGeometryKey(s *app.Session) string {
	version, ok := activeModelGeometryVersion(s)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(version)
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(int(s.VisualStyle())))
	ec := displayEdgeColor(s) // M16-F07: edge-color override is baked into the list, so it keys the cache
	b.WriteByte('|')
	b.WriteString(strconv.FormatFloat(float64(ec[0]+ec[1]*2+ec[2]*3), 'f', 3, 64))
	if on, perTri := s.MeshColors(); on { // mesh-debug-colors uses a different builder; key it apart
		if perTri {
			b.WriteString("|tricolors")
		} else {
			b.WriteString("|facecolors")
		}
	}
	for _, body := range s.VisibleBodies() {
		b.WriteByte('|')
		b.WriteString(strconv.FormatUint(body.ID(), 10))
	}
	return b.String()
}

// displayEdgeColor is the active document's display-settings edge color as an rgba float array,
// falling back to the renderer's default when there is no active document (M16-F07 #643).
func displayEdgeColor(s *app.Session) [4]float32 {
	if s.ActiveDocument() == nil {
		return renderer.DefaultEdgeColor()
	}
	return s.DocumentDisplaySettings(0).EdgeColor().Rgba().Array()
}

// modelGeometryVersioned is the active document content that reports a geometry version — a part or
// an assembly (compdef.PartComponentDefinition / AssemblyComponentDefinition). Matched structurally
// so the cache keys on either without importing or switching on the concrete type.
type modelGeometryVersioned interface{ ModelGeometryVersion() string }

// activeModelGeometryVersion returns the active document's geometry version, or false when no
// renderable model (part or assembly) is active — in which case there is nothing to cache.
func activeModelGeometryVersion(s *app.Session) (string, bool) {
	d := s.ActiveDocument()
	if d == nil {
		return "", false
	}
	m, ok := d.Content().(modelGeometryVersioned)
	if !ok {
		return "", false
	}
	return m.ModelGeometryVersion(), true
}
