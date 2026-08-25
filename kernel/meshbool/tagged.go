// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Provenance-tagged boolean (ADR-0054). Co-refinement only ever SPLITS an input
// facet against the other operand's crossing segments — it never fuses two input
// facets into one output triangle, and it never creates a triangle off an original
// surface. So every output triangle descends from exactly one input facet, and an
// integer tag identifying that facet's originating operand surface propagates
// exactly from input to output. The ADR-0054 reconstruction groups the result by tag
// to rebuild each face on its exact original analytic surface, instead of fitting a
// plane per coplanar facet region (which faceted-izes every curved boolean, #2153).

// TaggedSoup is a triangle soup with a provenance tag per triangle. Tags[i] is the
// caller-defined surface id of Tris[i] — the ops adapter assigns operand A's faces
// ids 0..na-1 and operand B's na..na+nb-1. The invariant len(Tags) == len(Tris)
// holds at every stage; [BooleanTagged] preserves it by construction.
type TaggedSoup struct {
	Tris [][3]Point
	Tags []int
}

// add appends one tagged triangle, preserving the len(Tags) == len(Tris) invariant.
func (s *TaggedSoup) add(t [3]Point, tag int) {
	s.Tris = append(s.Tris, t)
	s.Tags = append(s.Tags, tag)
}

// untagged wraps a plain soup as a TaggedSoup whose triangles all carry tag 0, so the
// untagged [Boolean]/[CoRefine] entry points can share the tagged implementation
// without duplicating the classification logic.
func untagged(tris [][3]Point) TaggedSoup {
	return TaggedSoup{Tris: tris, Tags: make([]int, len(tris))}
}

// BooleanTagged is [Boolean] carrying provenance: the result's Tags[i] is the
// originating surface id of Tris[i], copied from whichever operand facet the kept
// triangle descends from. It mirrors Boolean's keep/coplanar/orient decisions exactly
// (same face order), so BooleanTagged(untagged(a), untagged(b), op).Tris is identical
// to Boolean(a, b, op). Difference reverses b's kept faces (orientFromB) but keeps
// their tag — the cavity wall still belongs to b's surface.
func BooleanTagged(a, b TaggedSoup, op Op) TaggedSoup {
	ga, gb := newFaceGrid(a.Tris), newFaceGrid(b.Tris)
	aOut, bOut := coRefineTagged(a, b, ga, gb)
	var res TaggedSoup
	keepTaggedFromA(&res, op, aOut, b.Tris, gb)
	keepTaggedFromB(&res, op, bOut, a.Tris, ga)
	return res
}

// keepTaggedFromA appends the faces of a's refined operand the operation keeps, each
// with its inherited tag (mirrors Boolean's first loop).
func keepTaggedFromA(res *TaggedSoup, op Op, aOut TaggedSoup, b [][3]Point, gb *faceGrid) {
	for i, f := range aOut.Tris {
		if sameDir, coincident := coplanarPartner(f, b, gb); coincident {
			if keepCoplanar(op, sameDir) {
				res.add(f, aOut.Tags[i])
			}
			continue
		}
		if keepFromA(op, insideExact(centroid(f), b, gb)) {
			res.add(f, aOut.Tags[i])
		}
	}
}

// keepTaggedFromB appends the faces of b's refined operand the operation keeps (a's
// coincident copy already represents every coplanar face, so b's is dropped there),
// oriented for the operation and each with its inherited tag (mirrors Boolean's
// second loop).
func keepTaggedFromB(res *TaggedSoup, op Op, bOut TaggedSoup, a [][3]Point, ga *faceGrid) {
	for i, f := range bOut.Tris {
		if _, coincident := coplanarPartner(f, a, ga); coincident {
			continue
		}
		if keepFromB(op, insideExact(centroid(f), a, ga)) {
			res.add(orientFromB(op, f), bOut.Tags[i])
		}
	}
}

// coRefineTagged co-refines two tagged operands, propagating each input facet's tag
// onto its refined children.
func coRefineTagged(a, b TaggedSoup, ga, gb *faceGrid) (aOut, bOut TaggedSoup) {
	return refineAgainstTagged(a, b.Tris, gb), refineAgainstTagged(b, a.Tris, ga)
}

// refineAgainstTagged re-triangulates every face of faces to conform to others,
// giving each child the parent face's tag (RefineFace only subdivides a face, so a
// child's originating surface is its parent's).
func refineAgainstTagged(faces TaggedSoup, others [][3]Point, grid *faceGrid) TaggedSoup {
	var out TaggedSoup
	for i, f := range faces.Tris {
		for _, c := range RefineFace(f, faceConstraints(f, others, grid)) {
			out.add(c, faces.Tags[i])
		}
	}
	return out
}
