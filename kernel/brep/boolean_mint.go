// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"sort"

	"oblikovati.org/kernel/geom"
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
// ordered by the transform-invariant characteristic along the pair's intersection line — or, for a
// pair with no such line (two rims a wall and a cap share), by the total order on the edges'
// midpoints, the order the curved relineage ranks by. A lone edge of a pair keeps rank 0 (no
// disambiguator); the common case is therefore untouched.
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
			ma, mb := groupMid(groups[idxs[a]], verts), groupMid(groups[idxs[b]], verts)
			if ok {
				return lineCharacteristic(ma, d) < lineCharacteristic(mb, d)
			}
			return topo.LessPoint(ma, mb)
		})
		for r, i := range idxs {
			named[i].rank = r
		}
	}
}

// curvedRimLineages parents every unparented CURVED edge group by the faces that border it. Such an
// edge is a rim a curved trim emitted — a bore's circle where a tool wall meets a cap — which no planar
// imprint segment generated, so nameEdgeGroups had nothing to read and the stitch minted a build-order
// ordinal for it: the one name in a drilled plate that renumbered under an unrelated upstream edit
// (ADR-0061 stage 4). The bordering faces carry the provenance the curved relineage would have read;
// it is applied here, to these groups only, so every planar name the goldens pin stays as minted. A
// straight unparented edge is left alone: it is the planar path's split-original fragment, whose
// ordinal is that path's own convention.
func curvedRimLineages(named []namedEdge, groups []edgeGroup, verts []math.Point3, all []curvedFace) {
	for gi := range groups {
		if named[gi].parented || !groupIsCurved(groups[gi], all) {
			continue
		}
		lo, hi, ok := borderingFaceParents(groups[gi], all)
		if !ok {
			continue
		}
		named[gi] = namedEdge{lo: lo, hi: hi, parented: true}
	}
	rankNamedEdges(named, groups, verts, nil)
}

// groupIsCurved reports whether the group's edge runs on a curve that is not straight.
func groupIsCurved(g edgeGroup, all []curvedFace) bool {
	if len(g.uses) == 0 {
		return false
	}
	u := g.uses[0]
	return !geom.IsStraightCurve(all[u.face].loops[u.ring].edges[u.pos].curve)
}

// borderingFaceParents returns the canonical (lo, hi) lineages of the two faces using the group, or
// ok=false when it is not bordered by exactly two keyed faces.
func borderingFaceParents(g edgeGroup, all []curvedFace) (lo, hi topo.Lineage, ok bool) {
	if len(g.uses) != 2 {
		return topo.Lineage{}, topo.Lineage{}, false
	}
	a, b := all[g.uses[0].face].lineage, all[g.uses[1].face].lineage
	if len(a.Key()) == 0 || len(b.Key()) == 0 {
		return topo.Lineage{}, topo.Lineage{}, false
	}
	if string(a.Key()) > string(b.Key()) {
		a, b = b, a
	}
	return a, b, true
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
