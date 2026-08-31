// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"sort"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Planar-boolean edge NAMING from a radial-edge sew plan (ADR-0047/0043, #1726).
//
// The sew plan (boolean_radial_edge.go / curved_stitch_plan.go) is naming-free: which half-edge
// uses form which manifold edge, and how each welded vertex splits into radial disks. This file
// resolves each edge-group's reference-key lineage (ADR-0043 #1153/#1155) — every intersection edge
// named by its generating parent face-pair, disambiguated by rank along the parents' intersection
// line, the rest by deterministic ordinals. Since ADR-0058 the CONSTRUCTION lives in the one unified
// stitch (curved_stitch.go); the planar boolean injects these names through its stitchNaming hooks
// (planarStitchNaming), keeping the volatile radial topology decoupled from the stable provenance keys.

// namedEdge is the provenance name resolved for one edge-group: its parent-pair lineages (when it
// is an intersection edge), whether it is parented at all, and its rank among edges sharing that
// pair along their intersection line.
type namedEdge struct {
	lo, hi   topo.Lineage
	parented bool
	rank     int
}

// nameEdgeGroups resolves each edge-group's provenance name (parent pair via edgeParents) and then
// ranks the groups that share a parent pair along their intersection line, so several edges born of
// one face crossing get transform-invariant, stable keys (#1155).
func nameEdgeGroups(groups []edgeGroup, verts []math.Point3, prov []imprintSeg) []namedEdge {
	named := make([]namedEdge, len(groups))
	for gi := range groups {
		p, q := verts[groups[gi].pair[0]], verts[groups[gi].pair[1]]
		lo, hi, ok := edgeParents(p, q, prov)
		named[gi] = namedEdge{lo: lo, hi: hi, parented: ok}
	}
	rankNamedEdges(named, groups, verts, prov)
	return named
}

// rankNamedEdges assigns each parented edge its rank among edges sharing the same parent pair,
// ordered by the transform-invariant characteristic along the pair's intersection line. A lone edge
// of a pair keeps rank 0 (no disambiguator); the common case is therefore untouched.
func rankNamedEdges(named []namedEdge, groups []edgeGroup, verts []math.Point3, prov []imprintSeg) {
	byPair := map[string][]int{}
	for i := range named {
		if named[i].parented {
			key := string(named[i].lo.Key()) + "\x00" + string(named[i].hi.Key())
			byPair[key] = append(byPair[key], i)
		}
	}
	for _, idxs := range byPair {
		if len(idxs) < 2 {
			continue
		}
		d, ok := pairLineDir(named[idxs[0]].lo, named[idxs[0]].hi, prov)
		sort.SliceStable(idxs, func(a, b int) bool {
			return ok && lineCharacteristic(groupMid(groups[idxs[a]], verts), d) < lineCharacteristic(groupMid(groups[idxs[b]], verts), d)
		})
		for r, i := range idxs {
			named[i].rank = r
		}
	}
}

// groupMid is an edge-group's midpoint — the witness point the rank disambiguator projects onto the
// parent pair's intersection line.
func groupMid(g edgeGroup, verts []math.Point3) math.Point3 {
	p, q := verts[g.pair[0]], verts[g.pair[1]]
	return p.TranslateBy(p.VectorTo(q).Scale(0.5))
}

// edgeGroupLineage is an edge's lineage: its parent-pair name (with the disambiguating rank) when it
// is an intersection edge, else the ordinal fallback (incrementing idx).
func edgeGroupLineage(n *namedEdge, idx *int) topo.Lineage {
	if !n.parented {
		lin := topo.NewLineage(topo.Tok("brep", "edge", *idx))
		*idx++
		return lin
	}
	return intersectionLineage(n.lo, n.hi, n.rank)
}
