// SPDX-License-Identifier: GPL-2.0-only

package brep

import "oblikovati.org/kernel/geom"

// Curved-shell orientation repair (#1818). The near-pinch cut/join assembles a shell from faces built by
// DIFFERENT machinery: the (u,v) arrangement (self-consistent by construction) and the raw whole-loop
// constructors (keyhole holed wall, two-rim stubs, lens caps), each wound independently. A mixed shell can
// therefore be manifold and watertight yet have some shared edges traversed the SAME way by both faces —
// the orientation inconsistency Validate flags. This pass two-colours the face-adjacency graph over shared
// edges (BFS per shell) and flips the marked faces so every shared edge is traversed oppositely, mirroring
// the planar boolean's reorientFaces but on curvedFaces. Consistency is only half the answer, though —
// see curvedReorient for the other half.

// curvedReorient returns the faces re-wound to a globally consistent orientation (each shared edge traversed
// oppositely by its two faces) and turned OUTWARD. Already-consistent, already-outward input is returned
// unchanged (the flood fill flips nothing and the certification agrees).
//
// The two-colouring alone cannot say WHICH of the two consistent colourings is the right one. It seeds at a
// face and propagates, so the sense the whole shell ends up with is whichever sense that one face happened
// to arrive with. On a wrapped emboss join the seed was the pad's own cap and the fill dutifully flipped the
// OTHER TEN faces — the host's cone, its cylinder and both its caps — to agree with the one. Nothing
// downstream objected: the shell stayed manifold and Validate stayed happy, because both of those check
// TRAVERSAL. What inverted was every face's material side, so a host that integrated to +45029.49 came
// back as −45005.68 (Oblikovati/Oblikovati#3504).
//
// The sense used to be left to the tessellator's orientFacesOutward, which repairs the per-face MESH and
// never touches the B-rep — which is why the mesh read the emboss body at +45026 while its own faces said
// −45005. A modelling attribute cannot be decided downstream of tessellation, so it is derived here from
// the geometry instead: senseFromLoopWinding gives every face the sense its own loop winding implies,
// with the shell's signed volume settling the one global bit a winding cannot see.
func curvedReorient(faces []curvedFace) []curvedFace {
	pw := newWelder3(geom.ResolutionForBox(curvedFaceBox(faces)).Stitch())
	out := applyCurvedFlips(faces, curvedOrientationFlips(faces, pw))
	return senseFromLoopWinding(out)
}

// senseFromLoopWinding DERIVES every face's stored sense from its own loop winding, which is what ties
// the reversed flag to the geometry instead of letting it travel independently.
//
// It sets the flag ALONE and never touches a loop. That is the whole point: reversing both leaves the
// disagreement exactly where it was, since a face whose flag contradicts its winding still contradicts
// it afterwards. Setting the flag alone also leaves the two-colouring's work intact, because traversal
// consistency is a property of the loops. With every flag agreeing with its own winding AND every
// shared edge traversed oppositely, the faces' material sides necessarily agree — which is the
// invariant the two-colouring alone does not give (Oblikovati/Oblikovati#3504).
//
// The rule is orientFaceSigns, already the authority for the flux point classifier: each face's outer
// ring signed area in (u, v), with the shell's own signed volume settling the one global bit a winding
// cannot see. One mechanism decides orientation for the kernel rather than two that can disagree.
//
// This was tried once before and reverted, because loopHandedness could not read a seam-wrapping band:
// a plate's bore wall reported the handedness of its own opposite and the bore integrated as ADDED,
// 564.36 against a true 354.92. That was #3506, and it is fixed — the band's rims are now read as the
// one circuit they bound.
//
// A face carrying no usable (u, v) domain leaves the shell exactly as the two-colouring left it: a
// shell the integrator cannot read is not one this can certify, and guessing is worse.
func senseFromLoopWinding(faces []curvedFace) []curvedFace {
	ff := make([]fluxFace, 0, len(faces))
	for _, f := range faces {
		region := faceTrimRegion(f)
		u0, u1, v0, v1, ok := fluxDomain(f, region)
		if !ok {
			return faces
		}
		ff = append(ff, fluxFace{cf: f, region: region, u0: u0, u1: u1, v0: v0, v1: v1, sign: 1})
	}
	out := make([]curvedFace, len(faces))
	copy(out, faces)
	for i, sign := range orientFaceSigns(ff) {
		out[i].reversed = sign < 0
	}
	return out
}

// applyCurvedFlips returns the faces with the marked ones reversed — the stored sense and the loop
// traversal together, so a flip never breaks the consistency the two-colouring just established.
func applyCurvedFlips(faces []curvedFace, flip []bool) []curvedFace {
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
	flip := make([]bool, len(faces))
	walkFaceComponents(curvedFaceAdjacency(faces, pw), func(from, to int, sameDir bool) {
		flip[to] = flip[from] != sameDir // flip[nbr] = flip[f] XOR sameDir
	})
	return flip
}

// connectedFaceComponents groups the faces into connected components over the adjacency (one per shell),
// each listed in discovery order from its lowest-index face.
func connectedFaceComponents(adj [][]orientFlipNeighbour) [][]int {
	var comps [][]int
	comp := -1
	seen := make([]bool, len(adj))
	walkFaceComponents(adj, func(from, to int, _ bool) {
		if !seen[from] {
			seen[from] = true
			comps = append(comps, []int{from})
			comp = len(comps) - 1
		}
		seen[to] = true
		comps[comp] = append(comps[comp], to)
	})
	for f := range adj {
		if !seen[f] {
			comps = append(comps, []int{f})
		}
	}
	return comps
}

// walkFaceComponents breadth-first walks every connected component of the adjacency, calling visit on
// each tree edge (from an already-reached face to a newly reached neighbour) in discovery order.
func walkFaceComponents(adj [][]orientFlipNeighbour, visit func(from, to int, sameDir bool)) {
	seen := make([]bool, len(adj))
	for s := range adj {
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
				visit(f, e.face, e.sameDir)
				q = append(q, e.face)
			}
		}
	}
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
