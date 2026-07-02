// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"sort"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Pinched-vertex resolution for the planar-boolean stitch (Oblikovati#1693).
//
// Two configurations weld a contact down to a vertex the per-edge checks cannot see:
//
//   - a contact PATCH thinner than the weld grid (a fan blade tip designed exactly on its rim's
//     faceted inner wall meets it in a ~14 µm lens) collapses onto one vertex shared by two
//     disjoint face fans;
//   - a tangent contact EDGE (a fine-pitch coil whose turns touch — pitch equal to the wire
//     diameter — welds turn-to-turn contact chains) is correctly split by resolveEdgeUses into
//     two coincident manifold edges, but the endpoints of every resolved pair remain ONE welded
//     vertex, pinching at each chain vertex (319 of them on the NopSCADlib threaded rod). The
//     nudge retry cannot help there: a SELF-contact of one operand moves with itself, and the
//     opposing contact normals cancel in awayFromContacts.
//
// Either way every edge still carries exactly two faces; only the Euler characteristic drops,
// one per pinch, shipping an inadmissible "valid-looking" solid. The repair mirrors the tangent-
// EDGE resolution one dimension down: group each vertex's incident RESOLVED edges into fans
// (two edges connect iff some face uses both at that vertex) and give every fan beyond the first
// its own coincident duplicate vertex — the shells then touch at a point, the resolution-faithful
// result for a sub-grid contact (the CSG cage applies the same repair in ops).

// pinchedEndpoints returns, for every planned edge build, the topo vertex to use at each of its
// two endpoints: the shared welded vertex for a manifold vertex, or the build's fan-specific
// coincident duplicate at a pinched one. Keys are (build index, welded vertex index).
func pinchedEndpoints(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, builds []edgeBuild) map[[2]int]*topo.Vertex {
	incident := map[int][]int{} // welded vertex → indices into builds
	for bi := range builds {
		incident[builds[bi].k[0]] = append(incident[builds[bi].k[0]], bi)
		incident[builds[bi].k[1]] = append(incident[builds[bi].k[1]], bi)
	}
	order := make([]int, 0, len(incident))
	for v := range incident {
		order = append(order, v)
	}
	sort.Ints(order) // deterministic duplicate lineage regardless of map iteration
	endpoint := make(map[[2]int]*topo.Vertex, 2*len(builds))
	dup := 0
	for _, v := range order {
		inc := incident[v]
		for fanIdx, fan := range buildFans(builds, inc) {
			use := tv[v]
			if fanIdx > 0 {
				use = bld.AddVertex(verts[v], topo.NewLineage(topo.Tok("brep", "pinch", dup)))
				dup++
			}
			for _, bi := range fan {
				endpoint[[2]int{bi, v}] = use
			}
		}
	}
	return endpoint
}

// buildFans partitions a vertex's incident edge builds into fans: two builds are one fan iff
// some face loop uses both (their use groups share a face). A manifold vertex yields one fan.
func buildFans(builds []edgeBuild, inc []int) [][]int {
	parent := make(map[int]int, len(inc))
	var find func(x int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	for _, bi := range inc {
		parent[bi] = bi
	}
	byFace := map[int]int{} // face → first incident build seen using it
	for _, bi := range inc {
		for _, u := range builds[bi].group {
			if first, ok := byFace[u.face]; ok {
				parent[find(bi)] = find(first)
			} else {
				byFace[u.face] = bi
			}
		}
	}
	return groupFans(inc, find)
}

// groupFans collects union-find components in first-seen order.
func groupFans(inc []int, find func(int) int) [][]int {
	groups := map[int][]int{}
	order := []int{}
	for _, i := range inc {
		r := find(i)
		if _, seen := groups[r]; !seen {
			order = append(order, r)
		}
		groups[r] = append(groups[r], i)
	}
	fans := make([][]int, 0, len(order))
	for _, r := range order {
		fans = append(fans, groups[r])
	}
	return fans
}
