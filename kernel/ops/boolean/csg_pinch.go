// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Pinched-vertex splitting for the CSG cage (#1693).
//
// A near-tangent boolean contact — a fan blade tip designed exactly on its rim's inner wall
// touches it in a lens thinner than the facet resolution — collapses under the vertex weld into
// ONE vertex whose incident triangles form TWO disjoint fans. That pinch is invisible to
// edge-manifoldness (every edge still has exactly two faces) but is not a valid closed 2-manifold:
// each pinch drops the Euler characteristic by one, which is exactly how the blade JOIN measured
// χ = −1 and a reparametrized threaded rod χ = −319. The standard mesh-repair resolution is to CUT
// the vertex apart: one duplicate vertex (same coordinates) per extra fan, giving coincident but
// topologically distinct shells that touch at a point — the resolution-faithful result when the
// true contact patch is far below the weld grid.

// splitPinchedVertices duplicates every vertex whose incident triangles form more than one
// edge-connected fan, rewriting the extra fans onto fresh coincident vertices. Faces are updated
// in place; the (possibly extended) vertex slice is returned.
func splitPinchedVertices(verts []math.Point3, faces [][3]int) []math.Point3 {
	incident := make([][]int, len(verts))
	for fi, f := range faces {
		for _, v := range f {
			incident[v] = append(incident[v], fi)
		}
	}
	for v := range incident {
		verts = splitVertexFans(verts, faces, v, incident[v])
	}
	return verts
}

// splitVertexFans groups vertex v's incident faces into fans connected by shared v-incident
// edges and moves every fan beyond the first onto a fresh duplicate vertex.
func splitVertexFans(verts []math.Point3, faces [][3]int, v int, inc []int) []math.Point3 {
	fans := vertexFans(faces, v, inc)
	// An ORPHAN vertex (a coordinate in verts referenced by no triangle — CSG clipping can leave one)
	// has no incident fans, and a clean manifold vertex has exactly one; neither needs splitting. Guard
	// the fans[1:] slice so an orphan does not panic the last-resort CSG fallback (#1693 regression).
	if len(fans) < 2 {
		return verts
	}
	for _, fan := range fans[1:] {
		nv := len(verts)
		verts = append(verts, verts[v])
		for _, fi := range fan {
			for k, w := range faces[fi] {
				if w == v {
					faces[fi][k] = nv
				}
			}
		}
	}
	return verts
}

// vertexFans partitions the faces incident to vertex v into edge-connected components: two faces
// belong to the same fan iff they share an edge THROUGH v (the same opposite endpoint). A clean
// manifold vertex yields exactly one fan.
func vertexFans(faces [][3]int, v int, inc []int) [][]int {
	return topo.ComponentGroups(inc, func(join func(a, b int)) {
		byOther := map[int]int{} // opposite endpoint of a v-incident edge → first face seen
		for _, fi := range inc {
			for _, w := range faces[fi] {
				if w == v {
					continue
				}
				if first, ok := byOther[w]; ok {
					join(fi, first)
				} else {
					byOther[w] = fi
				}
			}
		}
	})
}
