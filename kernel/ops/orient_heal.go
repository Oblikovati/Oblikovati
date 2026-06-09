// SPDX-License-Identifier: GPL-2.0-only

package ops

import "oblikovati.org/math"

// orientFacesOutward flips per-face meshes so every face of the body winds consistently outward (a
// positive enclosed volume) — the M25 import-orientation heal for faces whose B-rep sense came in
// inverted (the Normal-Debug red faces on imported solids). It is purely topological: weld the shared
// vertices, 2-colour the faces by edge adjacency (two faces traversing a shared edge in OPPOSITE
// directions agree, the SAME direction disagree), and pick the global sign by signed volume. This is
// robust where a per-face normal test is not (a non-convex duct has many concave-but-correct faces).
// A FRUSTRATED face (its links disagree — a non-orientable tessellation defect) is left exactly as
// tessellated rather than flipped on a guess, so a correct face can never be mis-oriented. Operates on
// the tessellation, correcting the render and the mesh-divergence volume without touching the B-rep.
func orientFacesOutward(fm []*Mesh) {
	adj := faceAdjacency(fm)
	parity := twoColorFaces(adj)
	frustrated := frustratedFaces(adj, parity)
	globalFlip := orientationVolume(fm, parity) < 0
	for fi := range fm {
		if !frustrated[fi] && (parity[fi] == 1) != globalFlip {
			reverseMesh(fm[fi])
		}
	}
}

// orientLink is a face adjacency across one shared manifold edge; flip is true when the two faces
// traverse the canonical edge the same way (their windings disagree).
type orientLink struct {
	other int
	flip  bool
}

// faceAdjacency builds, per face, its adjacency links over the welded shared edges of all face meshes.
func faceAdjacency(fm []*Mesh) [][]orientLink {
	type use struct {
		face int
		dir  bool
	}
	uses := map[segKey][]use{}
	for fi, m := range fm {
		for t := 0; t+2 < len(m.Indices); t += 3 {
			for k := 0; k < 3; k++ {
				a, b := m.Positions[m.Indices[t+k]], m.Positions[m.Indices[t+(k+1)%3]]
				key := weldSeg(a, b)
				if len(uses[key]) < 2 {
					uses[key] = append(uses[key], use{fi, quantLess(a, b)})
				} else {
					uses[key] = append(uses[key], use{-1, false}) // mark non-manifold (>2)
				}
			}
		}
	}
	adj := make([][]orientLink, len(fm))
	for _, us := range uses {
		if len(us) != 2 || us[0].face < 0 || us[1].face < 0 || us[0].face == us[1].face {
			continue // boundary / non-manifold / internal edge
		}
		flip := us[0].dir == us[1].dir
		adj[us[0].face] = append(adj[us[0].face], orientLink{us[1].face, flip})
		adj[us[1].face] = append(adj[us[1].face], orientLink{us[0].face, flip})
	}
	return adj
}

// twoColorFaces assigns each face a parity (0/1) propagated through the adjacency links, per connected
// component, in deterministic face-index order.
func twoColorFaces(adj [][]orientLink) []int {
	parity := make([]int, len(adj))
	for i := range parity {
		parity[i] = -1
	}
	for s := range adj {
		if parity[s] != -1 {
			continue
		}
		parity[s] = 0
		for queue := []int{s}; len(queue) > 0; {
			u := queue[0]
			queue = queue[1:]
			for _, l := range adj[u] {
				if parity[l.other] != -1 {
					continue
				}
				parity[l.other] = parity[u] ^ flipBit(l.flip)
				queue = append(queue, l.other)
			}
		}
	}
	return parity
}

// frustratedFaces marks faces incident to a link the 2-colouring could not satisfy (an odd cycle).
func frustratedFaces(adj [][]orientLink, parity []int) []bool {
	bad := make([]bool, len(adj))
	for u := range adj {
		for _, l := range adj[u] {
			if parity[l.other] != parity[u]^flipBit(l.flip) {
				bad[u], bad[l.other] = true, true
			}
		}
	}
	return bad
}

// orientationVolume returns the body's signed volume with every face oriented to parity 0.
func orientationVolume(fm []*Mesh, parity []int) float64 {
	vol := 0.0
	for fi, m := range fm {
		vol += meshSignedVolume(m, parity[fi] == 1)
	}
	return vol
}

func flipBit(b bool) int {
	if b {
		return 1
	}
	return 0
}

// meshSignedVolume returns the divergence-theorem volume contribution of m, optionally with every
// triangle's winding flipped.
func meshSignedVolume(m *Mesh, flip bool) float64 {
	vol := 0.0
	for t := 0; t+2 < len(m.Indices); t += 3 {
		a := math.Point3{}.VectorTo(m.Positions[m.Indices[t]])
		b := math.Point3{}.VectorTo(m.Positions[m.Indices[t+1]])
		c := math.Point3{}.VectorTo(m.Positions[m.Indices[t+2]])
		s := float64(a.Dot(b.Cross(c)))
		if flip {
			s = -s
		}
		vol += s / 6.0
	}
	return vol
}

// quantLess reports whether a sorts before b in quantized coordinates (the canonical edge direction).
func quantLess(a, b math.Point3) bool {
	ka, kb := quantCoord(a), quantCoord(b)
	for i := 0; i < 3; i++ {
		if ka[i] != kb[i] {
			return ka[i] < kb[i]
		}
	}
	return false
}
