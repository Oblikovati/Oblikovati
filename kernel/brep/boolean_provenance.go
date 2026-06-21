// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"bytes"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Imprint provenance (M31-F02, Oblikovati/Oblikovati#1152). An intersection edge a boolean
// creates is named today by an ordinal index (boolean_nonmanifold.go buildResolvedEdges), which
// renumbers when an unrelated upstream edit reorders the stitched vertices — the topological
// naming problem. The robust name is the pair of faces whose crossing produced the edge (Kripac
// 1997). That pair is known where the imprint is computed (imprintAll), then discarded. This
// file captures it: provenanceOf tags every intersection/overlap segment with the lineages of
// the two faces that produced it, and edgeParents resolves a built edge back to that pair by
// midpoint containment. F03 (#1153) consumes edgeParents to name the edges.

// imprintSeg is an intersection (or coplanar-overlap) segment tagged with the lineages of the
// two faces whose crossing produced it: owner is the face the segment was recorded on, other is
// the crossing face. The unordered {owner, other} pair is the generating parentage of any edge
// built along the segment.
type imprintSeg struct {
	a, b         math.Point3
	owner, other topo.Lineage
}

// pairImprints returns the intersection/overlap segments of one crossing face pair, as recorded
// on each face: onA lies interior to a, onB interior to b. Shared by imprintAll (which needs the
// geometry to split faces) and provenanceOf (which needs the same segments to tag), so the two
// can never drift — see the boundary-segment caveat on imprintAll.
func pairImprints(a, b planarFace) (onA, onB [][2]math.Point3) {
	if coplanar(a, b) {
		return coplanarOverlapSegments(a, faceEdges3D(b)), coplanarOverlapSegments(b, faceEdges3D(a))
	}
	segs := imprint(a, b)
	return interiorSegments(a, segs), interiorSegments(b, segs)
}

// provenanceOf tags every imprint segment of every crossing face pair with its generating face
// lineages. Each segment is recorded twice (once per owning face, with owner/other swapped), so
// an edge that survives on either side resolves to the same unordered pair; edgeParents
// canonicalizes it. The list is the input F03 names boolean edges from.
func provenanceOf(fa, fb []planarFace) []imprintSeg {
	var prov []imprintSeg
	for i := range fa {
		for j := range fb {
			onA, onB := pairImprints(fa[i], fb[j])
			prov = appendTagged(prov, onA, fa[i].lineage, fb[j].lineage)
			prov = appendTagged(prov, onB, fb[j].lineage, fa[i].lineage)
		}
	}
	return prov
}

// appendTagged appends each segment tagged with the owner/other face lineages that produced it.
func appendTagged(prov []imprintSeg, segs [][2]math.Point3, owner, other topo.Lineage) []imprintSeg {
	for _, s := range segs {
		prov = append(prov, imprintSeg{a: s[0], b: s[1], owner: owner, other: other})
	}
	return prov
}

// edgeParentTol is how far an edge midpoint may sit off an imprint segment and still count as
// lying on it (model units, cm) — matches the imprint weld tolerance used elsewhere.
const edgeParentTol = 1e-7

// edgeParents returns the canonical {owner, other} face-lineage pair of the imprint segment the
// edge p→q lies on, or ok=false when the edge is not an intersection edge (e.g. an original face
// boundary, which lies on no imprint segment). The edge midpoint is used as the witness: a built
// intersection edge is a sub-span of its imprint segment, so its midpoint lies on that segment.
func edgeParents(p, q math.Point3, prov []imprintSeg) (topo.Lineage, topo.Lineage, bool) {
	mid := p.TranslateBy(p.VectorTo(q).Scale(0.5))
	for _, s := range prov {
		if pointOnSegment3(mid, s.a, s.b, edgeParentTol) {
			lo, hi := canonicalPair(s.owner, s.other)
			return lo, hi, true
		}
	}
	return topo.Lineage{}, topo.Lineage{}, false
}

// canonicalPair orders two lineages by their serialized key so the unordered pair {x, y} always
// yields the same (lo, hi) regardless of which face the segment was recorded on — the property
// that makes an intersection edge's name independent of operand order.
func canonicalPair(x, y topo.Lineage) (lo, hi topo.Lineage) {
	if bytes.Compare(x.Key(), y.Key()) <= 0 {
		return x, y
	}
	return y, x
}

// intersectionSep separates the two parent lineages in an intersection edge's name. Its feature
// id "brep" and role "x" (for "crossing") carry no separators, so the composed key parses
// unambiguously (lo tokens / brep:x#0 / hi tokens).
var intersectionSep = topo.Tok("brep", "x", 0)

// intersectionLineage names an edge born where two faces cross by the canonical concatenation of
// its two PARENT faces' lineages: lo / brep:x#0 / hi. Because the parents are the ORIGINAL faces
// (captured before any split), this name is invariant to how those faces are later subdivided and
// to the stitch's vertex ordering — the property the ordinal index lacked (#1153). `dup` is 0 for
// the common one-edge-per-pair case; a second edge sharing the same parent pair (a face crossed
// twice) gets dup>0, an interim disambiguator the geometric one in F05 (#1155) replaces.
func intersectionLineage(lo, hi topo.Lineage, dup int) topo.Lineage {
	toks := append([]topo.LineageToken{}, lo.Tokens()...)
	toks = append(toks, intersectionSep)
	toks = append(toks, hi.Tokens()...)
	if dup > 0 {
		toks = append(toks, topo.Tok("brep", "seg", dup))
	}
	return topo.NewLineage(toks...)
}
