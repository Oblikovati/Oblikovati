//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/head/viewport"
	"oblikovati.org/renderer"
)

// placedMeshSource is the ≤6-method view of the session cachedPlacedMesh needs — a mesh-set
// signature and the draw items. Depending on this narrow interface instead of the whole
// *app.Session keeps head/ui's session coupling from rising (archguard I5, arrowSession pattern).
type placedMeshSource interface {
	MeshDisplaySignature() (string, bool)
	MeshDrawItems() []renderer.DrawItem
}

// Placed-mesh render lane (#1773): a Mesh ▸ Place Mesh reference has no body, so it is drawn as a
// retained atlas region rather than through the per-frame overlay. Its flattened geometry is cached
// on the mesh-SET signature (not a content hash), so a dense scan is flattened ONCE and never
// re-flattened while orbiting or on an unrelated edit — the same retained-source discipline the body
// lane uses (sourceMeshCache), which is what keeps a 1.88M-triangle scan at full frame rate.

// placedMeshCache retains the last flattened placed-mesh geometry and the set signature it was built
// for. The render loop is single-threaded, so one package-level entry is safe (like frameAtlasCache
// and sourceMeshCache).
var placedMeshCache struct {
	sig  string
	mesh viewport.Mesh
	has  bool
}

// cachedPlacedMesh returns the flattened geometry of the active part's placed mesh references plus a
// cheap key identifying the mesh set. The flatten (O(triangles)) runs only when the set signature
// changes — a mesh placed, removed or suppressed — so orbiting a dense scan re-flattens nothing.
// has is false when there are no placed meshes to draw.
func cachedPlacedMesh(s placedMeshSource) (mesh viewport.Mesh, key string, has bool) {
	sig, ok := s.MeshDisplaySignature()
	if !ok {
		placedMeshCache.sig, placedMeshCache.has = "", false
		return viewport.Mesh{}, "", false
	}
	if placedMeshCache.sig == sig && placedMeshCache.has {
		return placedMeshCache.mesh, sig, true
	}
	m := viewport.Flatten(renderer.DrawList{Items: s.MeshDrawItems()})
	placedMeshCache.sig, placedMeshCache.mesh, placedMeshCache.has = sig, m, true
	return m, sig, true
}
