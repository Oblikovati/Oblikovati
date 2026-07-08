// SPDX-License-Identifier: GPL-2.0-only

package brep

import "oblikovati.org/kernel/geom"

// Curved-shell orientation repair (#1818). The near-pinch cut/join assembles a shell from faces built by
// DIFFERENT machinery: the (u,v) arrangement (self-consistent by construction) and the raw whole-loop
// constructors (keyhole holed wall, two-rim stubs, lens caps), each wound independently. A mixed shell can
// therefore be manifold and watertight yet have some shared edges traversed the SAME way by both faces —
// the orientation inconsistency Validate flags. This pass two-colours the face-adjacency graph over shared
// edges (BFS per shell) and flips the marked faces so every shared edge is traversed oppositely, mirroring
// the planar boolean's reorientFaces but on curvedFaces. The outward sense is fixed downstream by the mesh
// pass (orientFacesOutward); this only makes the B-rep edge-uses consistent.

// curvedReorient returns the faces re-wound to a globally consistent orientation (each shared edge traversed
// oppositely by its two faces). Already-consistent input is returned unchanged (the flood fill flips nothing).
func curvedReorient(faces []curvedFace) []curvedFace {
	pw := newWelder3(geom.ResolutionForBox(curvedFaceBox(faces)).Stitch())
	flip := curvedOrientationFlips(faces, pw)
	out := make([]curvedFace, len(faces))
	for i, f := range faces {
		if flip[i] {
			f.reversed = !f.reversed
			loops := make([]curvedLoop, len(f.loops))
			for j, lp := range f.loops {
				loops[j] = reverseCurvedLoop(lp)
			}
			f.loops = loops
		}
		out[i] = f
	}
	return out
}

// curvedOrientationFlips two-colours the face-adjacency graph (BFS over each connected shell) so that, after
// flipping the marked faces, every shared edge is traversed in opposite directions by its two faces.
func curvedOrientationFlips(faces []curvedFace, pw *welder3) []bool {
	adj := curvedFaceAdjacency(faces, pw)
	flip := make([]bool, len(faces))
	seen := make([]bool, len(faces))
	for s := range faces {
		if seen[s] {
			continue
		}
		seen[s] = true
		for q := []int{s}; len(q) > 0; {
			f := q[0]
			q = q[1:]
			for _, e := range adj[f] {
				if seen[e.face] {
					continue
				}
				seen[e.face] = true
				flip[e.face] = flip[f] != e.sameDir // flip[nbr] = flip[f] XOR sameDir
				q = append(q, e.face)
			}
		}
	}
	return flip
}

// curvedFaceAdjacency lists, per face, the faces it shares a manifold edge with and whether that edge is
// traversed the SAME direction by both (so one of them must be flipped for a consistent orientation). Edges
// used other than exactly twice are skipped (they are resolved by the weld/stitch, not orientation).
func curvedFaceAdjacency(faces []curvedFace, pw *welder3) [][]orientFlipNeighbour {
	type useRec struct {
		face int
		dir  bool
	}
	uses := map[[3]int][]useRec{}
	for fi := range faces {
		for _, lp := range faces[fi].loops {
			for _, le := range lp.edges {
				key, dir := reorientEdgeKey(pw, le)
				uses[key] = append(uses[key], useRec{fi, dir})
			}
		}
	}
	adj := make([][]orientFlipNeighbour, len(faces))
	for _, us := range uses {
		if len(us) != 2 {
			continue
		}
		same := us[0].dir == us[1].dir
		adj[us[0].face] = append(adj[us[0].face], orientFlipNeighbour{us[1].face, same})
		adj[us[1].face] = append(adj[us[1].face], orientFlipNeighbour{us[0].face, same})
	}
	return adj
}

// reorientEdgeKey returns a loop edge's weld-canonical key (welded endpoints + midpoint, matching curveWelder)
// and its traversal direction: for a CLOSED edge (a whole seam circle/loop) the sweep sign (t1 < t0); for an
// OPEN edge whether it runs from the lower-keyed endpoint to the higher. Two faces sharing a key with equal
// direction traverse it the same way, so one must be flipped.
func reorientEdgeKey(pw *welder3, le loopEdge) ([3]int, bool) {
	ka, kb := pw.add(le.start()), pw.add(le.end())
	kmid := pw.add(le.curve.PointAt((le.t0 + le.t1) / 2))
	if ka == kb { // closed: oriented by sweep sign, as curveWelder does
		return [3]int{ka, kb, kmid}, le.t1 < le.t0
	}
	if ka > kb {
		return [3]int{kb, ka, kmid}, false // runs high→low
	}
	return [3]int{ka, kb, kmid}, true // runs low→high
}
