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
		if on, perTri := s.MeshColors(); on { // each face/triangle a distinct color (viewport.setMeshColors)
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

// bodyGeometryKey identifies the cached geometry: the part's geometry version (bumped on any
// geometry/appearance edit and on recompute) + the visual style + the visible body ids.
func bodyGeometryKey(s *app.Session) string {
	part := activePart(s)
	if part == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(part.ModelGeometryVersion())
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(int(s.VisualStyle())))
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
