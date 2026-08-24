// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Whole-mesh co-refinement: given two triangle soups (the two boolean operands),
// return both, each re-triangulated so the two meshes share their mutual
// intersection curve exactly — the merged conforming complex the ADR-0052 cell
// labelling and surface extraction run on. Conformance is the property whose
// absence tears #2084; here it holds across the whole pair by construction,
// because each face is refined against the exact IntersectTriangles segments.

// CoRefine returns a and b re-triangulated to conform to each other along their
// intersection. Faces that meet nothing pass through unchanged. Coordinates stay
// exact; the caller rounds. PRECONDITION: both meshes have non-degenerate faces.
//
// Coplanar face-face overlaps are not yet resolved (IntersectTriangles reports
// them as Coplanar and they are skipped here); that 2D-overlap case is a later
// layer. Transversal crossings and single-point touches are handled.
func CoRefine(a, b [][3]Point) (aOut, bOut [][3]Point) {
	return refineAgainst(a, b), refineAgainst(b, a)
}

// refineAgainst re-triangulates every face of faces to conform to others.
func refineAgainst(faces, others [][3]Point) [][3]Point {
	var out [][3]Point
	for _, f := range faces {
		out = append(out, RefineFace(f, faceConstraints(f, others))...)
	}
	return out
}

// faceConstraints collects the exact intersection segments face f must conform to
// — one per other face it crosses. A Touching contact becomes a degenerate segment
// so its point is inserted as a shared vertex; a Coplanar overlap is skipped (see
// CoRefine).
func faceConstraints(f [3]Point, others [][3]Point) [][2]Point {
	var segs [][2]Point
	for _, o := range others {
		switch r := IntersectTriangles(f, o); r.Kind {
		case Crossing:
			segs = append(segs, [2]Point{r.P, r.Q})
		case Touching:
			segs = append(segs, [2]Point{r.P, r.P})
		}
	}
	return segs
}
