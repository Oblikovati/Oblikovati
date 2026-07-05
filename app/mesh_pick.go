// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/model/feature"
)

// Placed-mesh picking (#1776): a placed mesh reference is selectable by ray-casting its facets. The
// heavy per-triangle work is behind a cached BVH (ops.MeshRayIndex) on the RayPicker, so hover — which
// hit-tests every frame — stays cheap even for a 1.88M-triangle scan.

// MeshFaceHandle is a picked facet of a placed mesh reference: the owning mesh feature and the facet
// index, so a click can select the mesh and the head can highlight the hit facet.
type MeshFaceHandle struct {
	Mesh  *feature.MeshFeature
	Facet int
}

// SelectionKind implements Selectable.
func (MeshFaceHandle) SelectionKind() SelectionKind { return SelectMeshFace }

// Face returns the picked facet handle (its vertices/centroid), e.g. for the selection highlight.
func (h MeshFaceHandle) Face() feature.MeshFace { return h.Mesh.Faces().Item(h.Facet) }

// PickableMeshes returns the active part's visible placed mesh references — the RayPicker's mesh-facet
// hit-test targets (#1776). Empty when the active document is not a part or has no mesh references.
func (s *Session) PickableMeshes() []*feature.MeshFeature {
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	feats := part.Features()
	var out []*feature.MeshFeature
	for i := 0; i < feats.Count(); i++ {
		pf := feats.Item(i)
		if pf.Suppressed() {
			continue
		}
		if mf, ok := pf.Definition().(*feature.MeshFeature); ok {
			out = append(out, mf)
		}
	}
	return out
}

// meshTriangles fan-triangulates a mesh's facets into index triples for the ray BVH, plus a parallel
// facetOf mapping each triangle back to its facet (an STL facet is one triangle; a polygon facet
// fans into several that all point back to it).
func meshTriangles(g *feature.MeshGeometry) (tris [][3]int32, facetOf []int) {
	for fi, f := range g.Facets {
		for k := 2; k < len(f); k++ {
			tris = append(tris, [3]int32{int32(f[0]), int32(f[k-1]), int32(f[k])})
			facetOf = append(facetOf, fi)
		}
	}
	return tris, facetOf
}
