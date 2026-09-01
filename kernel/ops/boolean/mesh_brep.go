// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// MeshToBRep converts a closed welded mesh — shared vertices and facets given as ordered
// vertex-index loops (each typically a triangle) — into a faceted B-rep solid: one planar face
// per facet, with shared edges and vertices. Facets with more than three vertices are
// fan-triangulated. The cage is re-oriented to positive volume so an inward-wound (but
// consistent) input still yields an outward solid. Returns nil for an empty mesh.
//
// Example: tetra := MeshToBRep(verts, [][]int{{0,1,2},{0,2,3},{0,3,1},{1,3,2}}, "mesh") — a
// validated 4-face solid.
func MeshToBRep(verts []math.Point3, facets [][]int, feat string) *topo.Body {
	tris := facetTriangles(verts, facets)
	if len(tris) == 0 {
		return nil
	}
	body := trianglesToBody(tris, feat)
	if body == nil {
		return nil
	}
	// cageToBody trusts the facet winding for face normals; if the mesh was wound inward
	// (negative volume) flip every facet so the result is a proper outward solid.
	if query.BodyGeometryProperties(body, DefaultQuality()).Volume < 0 {
		body = trianglesToBody(reversedTris(tris), feat)
	}
	return body
}

// facetTriangles fan-triangulates each facet loop into CSG triangles, dropping degenerate ones
// and any facet whose indices are out of range.
func facetTriangles(verts []math.Point3, facets [][]int) []mesh.Tri {
	var tris []mesh.Tri
	for _, f := range facets {
		if len(f) < 3 || !facetInRange(f, len(verts)) {
			continue
		}
		for i := 1; i+1 < len(f); i++ {
			if t, ok := mesh.NewTri(verts[f[0]], verts[f[i]], verts[f[i+1]]); ok {
				tris = append(tris, t)
			}
		}
	}
	return tris
}

func facetInRange(f []int, n int) bool {
	for _, idx := range f {
		if idx < 0 || idx >= n {
			return false
		}
	}
	return true
}

// reversedTris flips each triangle's winding (swapping two corners).
func reversedTris(tris []mesh.Tri) []mesh.Tri {
	out := make([]mesh.Tri, 0, len(tris))
	for _, t := range tris {
		if rt, ok := mesh.NewTri(t.A, t.C, t.B); ok {
			out = append(out, rt)
		}
	}
	return out
}
