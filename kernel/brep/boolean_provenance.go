// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"bytes"
	"sort"

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
// built along the segment. ownerN/otherN are those faces' outward normals — the geometry the F05
// disambiguator (#1155) needs to order several edges that share one parent pair by a
// transform-invariant characteristic.
type imprintSeg struct {
	a, b           math.Point3
	owner, other   topo.Lineage
	ownerN, otherN math.Vector3
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
			prov = appendTagged(prov, onA, fa[i], fb[j])
			prov = appendTagged(prov, onB, fb[j], fa[i])
		}
	}
	return prov
}

// appendTagged appends each segment tagged with the owner/other faces' lineages and normals.
func appendTagged(prov []imprintSeg, segs [][2]math.Point3, owner, other planarFace) []imprintSeg {
	for _, s := range segs {
		prov = append(prov, imprintSeg{a: s[0], b: s[1],
			owner: owner.lineage, other: other.lineage, ownerN: owner.normal, otherN: other.normal})
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

// pairLineDir returns the intersection line's direction for the parent pair (lo, hi), derived
// equivariantly from the two faces' normals (n_lo × n_hi) so it rotates and translates rigidly
// with the geometry — the canonical axis along which the F05 disambiguator orders several edges
// that share this parent pair. ok is false when the pair has no segment in prov or the normals
// are parallel (no unique line).
func pairLineDir(lo, hi topo.Lineage, prov []imprintSeg) (math.Vector3, bool) {
	nLo, nHi, ok := pairNormals(lo, hi, prov)
	if !ok {
		return math.Vector3{}, false
	}
	d := nLo.Cross(nHi)
	if d.LengthSquared() < 1e-18 {
		return math.Vector3{}, false
	}
	return d.AsUnit().AsVector(), true
}

// pairNormals finds, in prov, the outward normals of the two faces named by the canonical pair
// (lo, hi), mapping each seg's owner/other normal to the matching member of the pair.
func pairNormals(lo, hi topo.Lineage, prov []imprintSeg) (nLo, nHi math.Vector3, ok bool) {
	loK, hiK := lo.Key(), hi.Key()
	for _, s := range prov {
		switch {
		case bytes.Equal(s.owner.Key(), loK) && bytes.Equal(s.other.Key(), hiK):
			return s.ownerN, s.otherN, true
		case bytes.Equal(s.owner.Key(), hiK) && bytes.Equal(s.other.Key(), loK):
			return s.otherN, s.ownerN, true
		}
	}
	return math.Vector3{}, math.Vector3{}, false
}

// lineCharacteristic projects p onto direction d — the transform-invariant scalar (a dot product
// is preserved by rotation; differences of it are preserved by translation) that orders edges
// sharing one parent pair along their common intersection line.
func lineCharacteristic(p math.Point3, d math.Vector3) float64 {
	return float64(d.X)*float64(p.X) + float64(d.Y)*float64(p.Y) + float64(d.Z)*float64(p.Z)
}

// fragmentMark prefixes a split-face fragment's cutting set; cuttingSep precedes each bordering
// cutting face so the variable-length set parses unambiguously (the parent tokens, then
// brep:cut#0, then one brep:by#0 + that face's tokens per cutting face).
var (
	fragmentMark = topo.Tok("brep", "cut", 0)
	cuttingSep   = topo.Tok("brep", "by", 0)
)

// fragmentCuttingFaces returns the sorted, unique set of OTHER-operand face lineages whose imprint
// borders the fragment sf of `parent` — the faces that cut this piece out (M31-F04, #1154). Only
// segments recorded ON the parent (owner == parent) count, so each "other" is a genuine cutting
// face. The set, not an ordinal index, identifies the piece, so it survives an upstream edit that
// merely reorders the pieces.
func fragmentCuttingFaces(parent topo.Lineage, sf subFace, prov []imprintSeg) []topo.Lineage {
	found := map[string]topo.Lineage{}
	collect := func(ring []math.Point3) {
		for i := range ring {
			mid := ring[i].TranslateBy(ring[i].VectorTo(ring[(i+1)%len(ring)]).Scale(0.5))
			for _, s := range prov {
				if bytes.Equal(s.owner.Key(), parent.Key()) && pointOnSegment3(mid, s.a, s.b, edgeParentTol) {
					found[string(s.other.Key())] = s.other
				}
			}
		}
	}
	collect(sf.outer)
	for _, h := range sf.holes {
		collect(h)
	}
	return sortedLineages(found)
}

// sortedLineages returns a lineage map's values ordered by key, so a cutting-face SET yields one
// canonical sequence regardless of discovery order.
func sortedLineages(m map[string]topo.Lineage) []topo.Lineage {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]topo.Lineage, len(keys))
	for i, k := range keys {
		out[i] = m[k]
	}
	return out
}

// fragmentLineage names a split-face fragment by its parent plus the canonical set of cutting
// faces bordering it: parent / brep:cut#0 / (brep:by#0 / cuttingFace)*. `dup` disambiguates two
// fragments of one face that share the same cutting set (a single straight cut halving a face) —
// 0 in the common distinct-border case, an interim index F05 (#1155) replaces geometrically.
func fragmentLineage(parent topo.Lineage, cutting []topo.Lineage, dup int) topo.Lineage {
	toks := append([]topo.LineageToken{}, parent.Tokens()...)
	toks = append(toks, fragmentMark)
	for _, c := range cutting {
		toks = append(toks, cuttingSep)
		toks = append(toks, c.Tokens()...)
	}
	if dup > 0 {
		toks = append(toks, topo.Tok("brep", "frag", dup))
	}
	return topo.NewLineage(toks...)
}

// vertexMark prefixes an intersection vertex's meeting-face set; meetSep precedes each face so the
// variable-length set parses unambiguously (brep:meet#0, then one brep:at#0 + that face's tokens).
var (
	vertexMark = topo.Tok("brep", "meet", 0)
	meetSep    = topo.Tok("brep", "at", 0)
)

// vertexLineages names each welded vertex (M31-F05, #1155). An INTERSECTION vertex — one lying on
// an imprint segment, i.e. created by the boolean — is named by the order-independent SET of faces
// meeting at it, each now carrying a parent-derived name, so the vertex name is stable across
// edits and rigid placements. A vertex on no imprint (an original corner) keeps its ordinal index
// (its welder position v), so the change is confined to the topology the boolean creates.
func vertexLineages(verts []math.Point3, faces []builtFace, prov []imprintSeg) []topo.Lineage {
	sets := vertexFaceSets(verts, faces)
	out := make([]topo.Lineage, len(verts))
	dups := map[string]int{}
	for v := range verts {
		if len(sets[v]) == 0 || !pointOnAnyImprint(verts[v], prov) {
			out[v] = topo.NewLineage(topo.Tok("brep", "vertex", v))
			continue
		}
		setKey := string(vertexLineage(sets[v], 0).Key())
		out[v] = vertexLineage(sets[v], dups[setKey])
		dups[setKey]++
	}
	return out
}

// vertexFaceSets returns, per welded vertex, the sorted unique lineages of the faces whose loops
// use it — the set that identifies an intersection vertex.
func vertexFaceSets(verts []math.Point3, faces []builtFace) [][]topo.Lineage {
	seen := make([]map[string]topo.Lineage, len(verts))
	for v := range verts {
		seen[v] = map[string]topo.Lineage{}
	}
	for _, f := range faces {
		for _, ring := range f.rings {
			for _, v := range ring {
				seen[v][string(f.lineage.Key())] = f.lineage
			}
		}
	}
	out := make([][]topo.Lineage, len(verts))
	for v := range verts {
		out[v] = sortedLineages(seen[v])
	}
	return out
}

// pointOnAnyImprint reports whether p lies on some imprint segment — the test for "this vertex was
// created where the operands cross" (endpoints included, since a vertex is a segment endpoint).
func pointOnAnyImprint(p math.Point3, prov []imprintSeg) bool {
	for _, s := range prov {
		if pointOnSegment3(p, s.a, s.b, edgeParentTol) {
			return true
		}
	}
	return false
}

// vertexLineage names an intersection vertex by the canonical set of faces meeting at it:
// brep:meet#0 / (brep:at#0 / face)*. `dup` disambiguates two vertices with the same meeting set.
func vertexLineage(faces []topo.Lineage, dup int) topo.Lineage {
	toks := []topo.LineageToken{vertexMark}
	for _, f := range faces {
		toks = append(toks, meetSep)
		toks = append(toks, f.Tokens()...)
	}
	if dup > 0 {
		toks = append(toks, topo.Tok("brep", "vtx", dup))
	}
	return topo.NewLineage(toks...)
}
