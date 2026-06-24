// SPDX-License-Identifier: GPL-2.0-only

package ops

import "oblikovati.org/math"

// consistentOutwardFlips returns, per triangle of m, whether that triangle's winding must be
// reversed for the whole mesh to wind consistently OUTWARD (a positive enclosed volume) — the
// divergence-theorem prerequisite for mass properties (Oblikovati/Oblikovati#1318).
//
// It recovers the orientation TOPOLOGICALLY: weld the shared vertices, 2-colour the triangles
// across their shared edges (neighbours that traverse a shared edge in OPPOSITE directions agree,
// the SAME direction disagree), then fix the one global sign by signed volume. It deliberately does
// NOT consult per-vertex shading normals: those diverge at a saddle or on a coarse-mesh silhouette
// sliver until their sum is nearly perpendicular to the facet, so the old `outwardRef·geomNormal < 0`
// test flipped facets at random there and corrupted the signed sum. A triangle on a non-orientable
// defect (its links disagree — a frustrated component) is left exactly as wound rather than flipped
// on a guess, mirroring orientFacesOutward's frustrated-face rule, so a correct facet is never
// mis-oriented.
//
//	flips := consistentOutwardFlips(mesh) // flips[i]==true ⇒ swap two corners of triangle i
func consistentOutwardFlips(m *Mesh) []bool {
	adj := triangleAdjacency(m)
	parity := twoColorFaces(adj)
	frustrated := frustratedFaces(adj, parity)
	globalFlip := triangleParityVolume(m, parity) < 0
	flips := make([]bool, len(parity))
	for ti := range flips {
		flips[ti] = !frustrated[ti] && (parity[ti] == 1) != globalFlip
	}
	return flips
}

// triangleAdjacency builds, per triangle of m, its orientation links over the welded shared edges —
// the per-triangle analogue of faceAdjacency, so the same 2-colouring drives orientation on a single
// merged mesh (mass properties) and on per-face meshes (orientFacesOutward).
func triangleAdjacency(m *Mesh) [][]orientLink {
	type use struct {
		tri int
		dir bool
	}
	uses := map[segKey][]use{}
	nt := m.TriangleCount()
	for ti := 0; ti < nt; ti++ {
		for k := 0; k < 3; k++ {
			a, b := triEdge(m, ti, k)
			key := weldSeg(a, b)
			if len(uses[key]) < 2 {
				uses[key] = append(uses[key], use{ti, quantLess(a, b)})
			} else {
				uses[key] = append(uses[key], use{-1, false}) // mark non-manifold (>2 uses)
			}
		}
	}
	adj := make([][]orientLink, nt)
	for _, us := range uses {
		if len(us) != 2 || us[0].tri < 0 || us[1].tri < 0 || us[0].tri == us[1].tri {
			continue // boundary / non-manifold / self edge
		}
		flip := us[0].dir == us[1].dir
		adj[us[0].tri] = append(adj[us[0].tri], orientLink{us[1].tri, flip})
		adj[us[1].tri] = append(adj[us[1].tri], orientLink{us[0].tri, flip})
	}
	return adj
}

// triEdge returns the k-th directed edge (corner k → corner k+1) of triangle ti.
func triEdge(m *Mesh, ti, k int) (math.Point3, math.Point3) {
	return m.Positions[m.Indices[3*ti+k]], m.Positions[m.Indices[3*ti+(k+1)%3]]
}

// triVerts returns triangle ti's three corner points.
func triVerts(m *Mesh, ti int) (math.Point3, math.Point3, math.Point3) {
	return m.Positions[m.Indices[3*ti]], m.Positions[m.Indices[3*ti+1]], m.Positions[m.Indices[3*ti+2]]
}

// triangleParityVolume is the divergence-theorem signed volume of m with every parity-1 triangle's
// winding flipped — the sign tells consistentOutwardFlips whether the whole component is inside-out.
func triangleParityVolume(m *Mesh, parity []int) float64 {
	vol := 0.0
	for ti := 0; ti < m.TriangleCount(); ti++ {
		a := math.Point3{}.VectorTo(m.Positions[m.Indices[3*ti]])
		b := math.Point3{}.VectorTo(m.Positions[m.Indices[3*ti+1]])
		c := math.Point3{}.VectorTo(m.Positions[m.Indices[3*ti+2]])
		s := float64(a.Dot(b.Cross(c)))
		if parity[ti] == 1 {
			s = -s
		}
		vol += s / 6.0
	}
	return vol
}
