// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"bytes"
	"sort"
)

// Provenance naming (ADR-0043). A derived entity — an edge a boolean cuts, a fillet's tangent
// edge, a delete-face stitch edge — should be named by the INPUT entities that GENERATED it, not
// by a construction-order counter. A parent-derived name survives an unrelated upstream edit that
// reorders the build/stitch (the property an ordinal index lacks), because the parents' lineages
// are themselves stable. The boolean established this for face-pair intersection edges
// (kernel/brep); this is the shared, op-agnostic mechanism every generator names through.

// NameByParents composes a provenance lineage from the lineages of the entities that generated a
// derived entity. The parents are CANONICALLY ORDERED (by serialized key) so the name is
// independent of the order the generator happened to discover them in (operand order, stitch
// order); their token runs are concatenated with sep between each parent; and rank, when > 0, is
// appended as a disambiguator token carrying rankSeed's feature/role — distinguishing several
// derived entities that share the same parents (e.g. one parent pair crossed twice), ordered by a
// transform-invariant geometric characteristic the caller computes, never by a build counter.
//
// sep and rankSeed must carry no '/' ':' '#' in their feature/role (they are ids by convention),
// so the composite key parses unambiguously back into parent token runs.
//
// Example (a boolean intersection edge of two faces): NameByParents([]Lineage{faceA, faceB},
// Tok("brep","x",0), Tok("brep","seg",0), 0) → faceA / brep:x#0 / faceB.
func NameByParents(parents []Lineage, sep, rankSeed LineageToken, rank int) Lineage {
	ordered := append([]Lineage(nil), parents...)
	sort.Slice(ordered, func(i, j int) bool {
		return bytes.Compare(ordered[i].Key(), ordered[j].Key()) < 0
	})
	var toks []LineageToken
	for i, p := range ordered {
		if i > 0 {
			toks = append(toks, sep)
		}
		toks = append(toks, p.tokens...)
	}
	if rank > 0 {
		toks = append(toks, Tok(rankSeed.Feature, rankSeed.Role, rank))
	}
	return Lineage{tokens: toks}
}

// RelineageByFaceProvenance renames a freshly-built body's EDGES and VERTICES from face provenance
// (ADR-0043): faceProv maps each result face to the lineage of the input entity that generated it
// (the faces themselves are already named by their provenance at build time). An edge is renamed by
// NameByParents over its bordering faces' provenance; a vertex by the faces meeting at it. An edge
// or vertex any of whose faces is absent from faceProv keeps its existing (ordinal) name — so a
// PARTIALLY-provenanced body (e.g. a constant fillet's cylinder + caps provenanced, a variable
// fillet's ruling strips not yet) stays consistent, the un-provenanced region simply keeping its
// build-order name. It mutates identity, so call it only during construction, before the body is
// observed elsewhere. sep/rankSeed parameterize the composed names (see NameByParents).
func (b *Body) RelineageByFaceProvenance(faceProv map[*Face]Lineage, sep, rankSeed LineageToken) {
	for _, e := range b.Edges() {
		if name, ok := provNameFromFaces(e.Faces(), faceProv, sep, rankSeed); ok {
			e.lineage = name
		}
	}
	for _, v := range b.Vertices() {
		if name, ok := provNameFromFaces(vertexFaces(v), faceProv, sep, rankSeed); ok {
			v.lineage = name
		}
	}
}

// provNameFromFaces composes a provenance name from faces' provenance lineages, or ok=false when
// any face has no provenance (so the entity keeps its existing name).
func provNameFromFaces(faces []*Face, faceProv map[*Face]Lineage, sep, rankSeed LineageToken) (Lineage, bool) {
	if len(faces) == 0 {
		return Lineage{}, false
	}
	parents := make([]Lineage, 0, len(faces))
	for _, f := range faces {
		p, ok := faceProv[f]
		if !ok {
			return Lineage{}, false
		}
		parents = append(parents, p)
	}
	return NameByParents(parents, sep, rankSeed, 0), true
}

// vertexFaces returns the distinct faces meeting at v, via its incident edges.
func vertexFaces(v *Vertex) []*Face {
	seen := map[*Face]bool{}
	var out []*Face
	for _, e := range v.edges {
		for _, f := range e.Faces() {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	return out
}
