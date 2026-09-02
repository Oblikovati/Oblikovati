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
// −45005. A modelling attribute cannot be decided downstream of tessellation, so the global bit is
// certified here against the geometry instead: the divergence theorem over the faces' own analytic
// surfaces, the same rule reorientFaces applies on the polygonal path and orientFaceSigns applies for the
// flux classifier.
//
// What this settles is the GLOBAL bit, and that is all. A single face out of step with its neighbours
// survives it — the shell comes back the right way out with its vector area still not closing — because
// catching that needs a per-face winding rule, which curvedShellPointsInward explains is blocked.
func curvedReorient(faces []curvedFace) []curvedFace {
	pw := newWelder3(geom.ResolutionForBox(curvedFaceBox(faces)).Stitch())
	out := applyCurvedFlips(faces, curvedOrientationFlips(faces, pw))
	if inward, certain := curvedShellPointsInward(out); certain && inward {
		return applyCurvedFlips(out, everyFace(len(out)))
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

// everyFace marks the whole shell, for the global flip that turns an inward-facing colouring outward.
func everyFace(n int) []bool {
	all := make([]bool, n)
	for i := range all {
		all[i] = true
	}
	return all
}

// curvedShellPointsInward reports whether the shell's STORED senses bound a negative volume — its
// material-outward normals pointing into the region rather than out of it. It is the divergence theorem
// over the faces' own analytic surfaces, the same rule reorientFaces already applies on the polygonal
// path, on the coarse grid orientFaceSigns uses (only the sign is wanted).
//
// It settles the GLOBAL bit only. Deriving each face's sense from its own loop winding — which is what
// would also catch a single face out of step with its neighbours — needs loopHandedness, and that reads
// the signed area of the outer ring in (u, v), which a SEAM-WRAPPING band does not have: its rims are
// open polylines in the covering space, so their shoelace is meaningless. Tried, and it inverted a
// plate's bore wall, whose volume then read 564.36 against a true 354.92 — the bore added rather than
// subtracted. Closing a wrapping ring through the seam first is what that needs (Oblikovati/Oblikovati#3504).
//
// certain=false when a face carries no usable (u, v) domain to integrate over, or when the sum lands on
// zero: the caller then leaves the shell exactly as it found it rather than flip on a partial sum.
func curvedShellPointsInward(faces []curvedFace) (inward, certain bool) {
	volume := 0.0
	for _, f := range faces {
		region := faceTrimRegion(f)
		u0, u1, v0, v1, ok := fluxDomain(f, region)
		if !ok {
			return false, false
		}
		ff := fluxFace{cf: f, region: region, u0: u0, u1: u1, v0: v0, v1: v1, sign: 1}
		volume += storedFaceSense(f) * faceVolumeTerm(&ff)
	}
	return volume < 0, volume != 0
}

// storedFaceSense is +1 when a face's material-outward normal is S_u×S_v, −1 when it is the opposite.
func storedFaceSense(f curvedFace) float64 {
	if f.reversed {
		return -1
	}
	return 1
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
