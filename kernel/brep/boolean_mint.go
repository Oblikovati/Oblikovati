// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Minting topo entities from a radial-edge sew plan (ADR-0047/0043, #1726).
//
// radialSew (boolean_radial_edge.go) produces a naming-free sewPlan: which half-edge uses form
// which manifold edge, and how each welded vertex splits into radial disks. This file is the
// downstream CONSUMER that turns that plan into named topo entities, holding two concerns the pure
// sew deliberately does not: (1) reference-key naming (ADR-0043 #1153/#1155) — every intersection
// edge named by its generating parent face-pair, disambiguated by rank along the parents'
// intersection line; every duplicate pinch vertex named by a deterministic ordinal; and (2)
// construction — the topo.Builder AddVertex/AddEdge calls. Splitting the naming out of the sew is
// what lets the volatile radial topology change without disturbing the stable provenance keys.

// namedEdge is the provenance name resolved for one edge-group: its parent-pair lineages (when it
// is an intersection edge), whether it is parented at all, and its rank among edges sharing that
// pair along their intersection line.
type namedEdge struct {
	lo, hi   topo.Lineage
	parented bool
	rank     int
}

// mintEntities builds the shared topo edges for a sew plan and maps each half-edge use to its edge.
// Pinched vertices are cut into per-disk coincident duplicates first (mintDiskVertices), then each
// edge-group is minted between its disks' vertices with its provenance name. It replaces the old
// buildResolvedEdges, which tangled the radial resolution into the naming and construction.
func mintEntities(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, plan sewPlan, prov []imprintSeg) map[[3]int]*topo.Edge {
	endpoint := mintDiskVertices(bld, verts, tv, plan.disks)
	named := nameEdgeGroups(plan.groups, verts, prov)
	useEdge := make(map[[3]int]*topo.Edge)
	idx := 0
	for gi := range plan.groups {
		g := &plan.groups[gi]
		e := bld.AddEdge(geom.NewLineSegment(verts[g.pair[0]], verts[g.pair[1]]),
			endpoint[[2]int{gi, g.pair[0]}], endpoint[[2]int{gi, g.pair[1]}],
			edgeGroupLineage(&named[gi], &idx))
		for _, h := range g.uses {
			useEdge[[3]int{h.face, h.ring, h.pos}] = e
		}
	}
	return useEdge
}

// mintDiskVertices returns, for every (group, welded vertex) endpoint, the topo vertex to use: the
// shared welded vertex on a manifold disk, or a fresh coincident duplicate on every disk beyond the
// first at a pinch. Vertices are walked in sorted order so the duplicate lineage (brep:pinch#dup) is
// deterministic regardless of map iteration (#1693).
func mintDiskVertices(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, disks map[int][]vertexDisk) map[[2]int]*topo.Vertex {
	endpoint := make(map[[2]int]*topo.Vertex)
	dup := 0
	for _, v := range sortedDiskVertices(disks) {
		for _, disk := range disks[v] {
			use := tv[v]
			if disk.copy > 0 {
				use = bld.AddVertex(verts[v], topo.NewLineage(topo.Tok("brep", "pinch", dup)))
				dup++
			}
			for _, gi := range disk.groups {
				endpoint[[2]int{gi, v}] = use
			}
		}
	}
	return endpoint
}

// sortedDiskVertices returns the welded-vertex keys of the disk partition in ascending order, so
// the pinch-duplicate lineage counter advances deterministically.
func sortedDiskVertices(disks map[int][]vertexDisk) []int {
	order := make([]int, 0, len(disks))
	for v := range disks {
		order = append(order, v)
	}
	sort.Ints(order)
	return order
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
