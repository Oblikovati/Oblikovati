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
// Transversal crossings, single-point touches, and coplanar overlaps are all
// handled: a coplanar overlap imprints the boundary of the shared convex region on
// both faces so they conform there too.
func CoRefine(a, b [][3]Point) (aOut, bOut [][3]Point) {
	return refineAgainst(a, b), refineAgainst(b, a)
}

// refineAgainst re-triangulates every face of faces to conform to others. Two
// faces of the SAME operand sharing an edge stay watertight without extra work: a
// point where the other operand crosses that edge is EdgePlaneCross(P,Q), which is
// bit-identical to EdgePlaneCross(Q,P), so both incident faces split the shared
// edge at the exact same point. (This holds only for a watertight input; the ops
// adapter welds the tessellation to guarantee it.)
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
		case Coplanar:
			segs = append(segs, coplanarConstraints(f, o)...)
		}
	}
	return segs
}

// coplanarConstraints returns the boundary edges of the convex overlap of coplanar
// faces f and o, so both faces are imprinted with the shared region's outline.
func coplanarConstraints(f, o [3]Point) [][2]Point {
	overlap := trianglePolygonOverlap(f, o, planeAxis(f))
	if len(overlap) < 2 {
		return nil // a point touch or no overlap contributes no edge
	}
	if len(overlap) == 2 {
		return [][2]Point{{overlap[0], overlap[1]}} // a shared edge
	}
	segs := make([][2]Point, len(overlap))
	for i := range overlap {
		segs[i] = [2]Point{overlap[i], overlap[(i+1)%len(overlap)]}
	}
	return segs
}
