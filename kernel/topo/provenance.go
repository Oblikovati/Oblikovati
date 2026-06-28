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
