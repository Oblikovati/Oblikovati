// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
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
		return ordered[i].KeyString() < ordered[j].KeyString()
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
	return newLineage(toks)
}

// RelineageByFaceProvenance renames a freshly-built body's EDGES and VERTICES from face provenance
// (ADR-0043): faceProv maps each result face to the lineage of the input entity that generated it
// (the faces themselves are already named by their provenance at build time). An edge is renamed by
// NameByParents over its bordering faces' provenance; a vertex by the faces meeting at it. An edge
// or vertex any of whose faces is absent from faceProv keeps its existing (ordinal) name — so a
// PARTIALLY-provenanced body (e.g. a constant fillet's cylinder + caps provenanced, a variable
// fillet's ruling strips not yet) stays consistent, the un-provenanced region simply keeping its
// build-order name.
//
// Two derived entities can share the SAME parent set — two surface-intersection curves between the
// same face pair (a cone crossing a cylinder cuts two branches), or two seam vertices where the same
// faces meet. Naming both by the bare parent name would collide (one reference key, two entities), so
// a parent set borne by more than one entity is disambiguated by a transform-invariant geometric
// order (the entity's midpoint/position) into ranks 1..n — the rank disambiguator NameByParents
// reserves. Edges and vertices are ranked in SEPARATE groups so a vertex's name never depends on the
// edges around it. It mutates identity, so call it only during construction, before the body is
// observed elsewhere. sep/rankSeed parameterize the composed names (see NameByParents).
func (b *Body) RelineageByFaceProvenance(faceProv map[*Face]Lineage, sep, rankSeed LineageToken) {
	edges := make([]provTarget, 0, len(b.Edges()))
	for _, e := range b.Edges() {
		edges = append(edges, provTarget{faces: e.Faces(), at: edgeRankPoint(e), set: func(l Lineage) { e.lineage = l }})
	}
	assignProvNames(edges, faceProv, sep, rankSeed)

	verts := make([]provTarget, 0, len(b.Vertices()))
	for _, v := range b.Vertices() {
		verts = append(verts, provTarget{faces: vertexFaces(v), at: v.point, set: func(l Lineage) { v.lineage = l }})
	}
	assignProvNames(verts, faceProv, sep, rankSeed)
}

// provTarget is one entity (edge or vertex) to be provenance-named: its bordering faces, a
// transform-invariant geometric point that orders siblings sharing a parent set, and a setter that
// stamps the resolved lineage onto the entity.
type provTarget struct {
	faces []*Face
	at    math.Point3
	set   func(Lineage)
}

// provGroup collects the targets that resolve to one parent set, so a set borne by several entities
// can be disambiguated by rank.
type provGroup struct {
	parents []Lineage
	members []provTarget
}

// assignProvNames names each target by its faces' provenance, disambiguating targets that share the
// same parent set with a geometric rank (see RelineageByFaceProvenance). A target any of whose faces
// lacks provenance keeps its existing name.
func assignProvNames(targets []provTarget, faceProv map[*Face]Lineage, sep, rankSeed LineageToken) {
	groups := map[string]*provGroup{}
	var order []string
	for _, t := range targets {
		parents, ok := parentLineages(t.faces, faceProv)
		if !ok {
			continue
		}
		key := string(NameByParents(parents, sep, rankSeed, 0).Key())
		if groups[key] == nil {
			groups[key] = &provGroup{parents: parents}
			order = append(order, key)
		}
		groups[key].members = append(groups[key].members, t)
	}
	for _, key := range order {
		nameGroup(groups[key], sep, rankSeed)
	}
}

// nameGroup stamps a group's members: the bare parent name when one entity bears the set, else the
// parent name plus a geometric rank (1..n, ordered by the members' points) so the keys stay distinct.
func nameGroup(g *provGroup, sep, rankSeed LineageToken) {
	if len(g.members) == 1 {
		g.members[0].set(NameByParents(g.parents, sep, rankSeed, 0))
		return
	}
	sort.Slice(g.members, func(i, j int) bool { return lessPoint(g.members[i].at, g.members[j].at) })
	for i, m := range g.members {
		m.set(NameByParents(g.parents, sep, rankSeed, i+1))
	}
}

// parentLineages collects the provenance lineages of an entity's faces, or ok=false when any face has
// no provenance (so the entity keeps its existing name).
func parentLineages(faces []*Face, faceProv map[*Face]Lineage) ([]Lineage, bool) {
	if len(faces) == 0 {
		return nil, false
	}
	parents := make([]Lineage, 0, len(faces))
	for _, f := range faces {
		p, ok := faceProv[f]
		if !ok {
			return nil, false
		}
		parents = append(parents, p)
	}
	return parents, true
}

// edgeRankPoint is a transform-invariant point that orders edges sharing a parent set. It samples the
// edge's CURVE at its mid-parameter, not the endpoint midpoint, so the two arcs of a bigon — two
// intersection branches between the same face pair that share BOTH endpoints (e.g. a Steinmetz
// front/back pair, or a closed seam circle whose endpoints coincide) — get DISTINCT points and stable
// ranks. A non-finite domain (an unbounded line) falls back to the endpoint midpoint.
func edgeRankPoint(e *Edge) math.Point3 {
	lo, hi := e.curve.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) {
		return edgeMidpoint(e)
	}
	return e.curve.PointAt((lo + hi) / 2)
}

// lessPoint is a total order on points (x, then y, then z) for stable sibling ranking.
func lessPoint(a, b math.Point3) bool {
	if a.X != b.X {
		return a.X < b.X
	}
	if a.Y != b.Y {
		return a.Y < b.Y
	}
	return a.Z < b.Z
}

// InheritOriginalEdges restores original edge identity to a derived body (ADR-0043): a result edge
// whose two endpoints coincide with an input edge's endpoints — an untouched boundary the operation
// passed through WHOLE — takes that input edge's lineage, instead of the build-order fallback a
// boolean assigns to surviving (non-intersection) edges. Only an input edge matched by EXACTLY ONE
// result edge (an unsplit survivor) is inherited, so two result edges never claim the same lineage;
// a split edge's fragments and the intersection edges (which have new endpoints) are left untouched.
// Coincidence is tested at a tolerance scaled to the body's size. Call during construction.
func (b *Body) InheritOriginalEdges(originals []*Edge) {
	tol := float64(b.RangeBox().Diagonal().Length())*1e-9 + 1e-9
	res := b.Edges()
	for _, o := range originals {
		var match *Edge
		count := 0
		for _, r := range res {
			if edgeEndpointsCoincide(r, o, tol) {
				match, count = r, count+1
			}
		}
		if count == 1 {
			match.lineage = o.lineage
		}
	}
}

// edgeEndpointsCoincide reports whether r and o have the same endpoint PAIR (in either direction),
// within tol — the test that r is the same boundary as the original edge o.
func edgeEndpointsCoincide(r, o *Edge, tol float64) bool {
	rs, re, os, oe := r.start.point, r.end.point, o.start.point, o.end.point
	if float64(rs.DistanceTo(os)) <= tol && float64(re.DistanceTo(oe)) <= tol {
		return true
	}
	return float64(rs.DistanceTo(oe)) <= tol && float64(re.DistanceTo(os)) <= tol
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
