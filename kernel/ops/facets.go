// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Queryable facet/stroke sets (M07-F03 remainder, Oblikovati/Oblikovati#293):
// the reference API lets clients calculate facets at a tolerance and retrieve
// them later without re-faceting. This file computes the snapshots; the
// tolerance-keyed cache lives in model/facetstore.

// BodyFacets is one body's facet set at one tolerance: the merged mesh, the
// triangle-index count each face contributed (in body face order), and the
// faces' own meshes for face-level retrieval.
type BodyFacets struct {
	Mesh              *Mesh
	IndexCountPerFace []int
	Faces             []*topo.Face
	FaceMeshes        []*Mesh
	uvs               []float64
}

// CalculateBodyFacets facets every face at the chordal tolerance through the
// same pipeline the display tessellation uses (conformance repair + outward
// orientation), tracking each face's contribution.
//
// Example: fs := ops.CalculateBodyFacets(body, ops.Quality{ChordTolerance: 0.01})
func CalculateBodyFacets(b *topo.Body, q Quality) *BodyFacets {
	faces, fm := tessellateBodyFaces(b, q)
	out := &BodyFacets{Mesh: &Mesh{}, Faces: faces, FaceMeshes: fm}
	for _, m := range fm {
		out.IndexCountPerFace = append(out.IndexCountPerFace, len(m.Indices))
		mergeMesh(out.Mesh, m)
	}
	return out
}

// TextureCoordinates returns one (u, v) pair per mesh vertex, taken from each
// owning face's surface parameterization — the reference texture-map facet
// variant. Computed on first call and memoized.
func (bf *BodyFacets) TextureCoordinates() []float64 {
	if bf.uvs != nil {
		return bf.uvs
	}
	bf.uvs = make([]float64, 0, 2*len(bf.Mesh.Positions))
	for i, m := range bf.FaceMeshes {
		surf := bf.Faces[i].Geometry()
		for _, p := range m.Positions {
			u, v := surf.ParamAt(p)
			bf.uvs = append(bf.uvs, u, v)
		}
	}
	return bf.uvs
}

// BodyStrokes is one body's wireframe stroke set at one tolerance: each edge
// sampled into a polyline (healed edges return their snapped seam exactly).
type BodyStrokes struct {
	Edges     []*topo.Edge
	Polylines [][]math.Point3
}

// CalculateBodyStrokes samples every edge of the body at the chordal tolerance.
//
// Example: st := ops.CalculateBodyStrokes(body, ops.Quality{ChordTolerance: 0.01})
func CalculateBodyStrokes(b *topo.Body, q Quality) *BodyStrokes {
	return strokeEdges(b.Edges(), q)
}

// CalculateFaceStrokes samples one face's boundary edges at the tolerance.
func CalculateFaceStrokes(f *topo.Face, q Quality) *BodyStrokes {
	return strokeEdges(f.Edges(), q)
}

func strokeEdges(edges []*topo.Edge, q Quality) *BodyStrokes {
	out := &BodyStrokes{Edges: edges, Polylines: make([][]math.Point3, len(edges))}
	for i, e := range edges {
		out.Polylines[i] = discretizeEdge(e, q)
	}
	return out
}
