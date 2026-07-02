// SPDX-License-Identifier: GPL-2.0-only

package topo

// ComponentGroups partitions ids into connected components under caller-declared equivalences:
// link is invoked once with a join function the caller uses to declare two ids equivalent, and
// the components come back in first-seen id order (deterministic for deterministic input). The
// boolean stitch and the CSG cage both use it to group a vertex's incident faces/edges into
// fans when cutting pinched vertices apart (Oblikovati#1693).
//
//	fans := topo.ComponentGroups(incident, func(join func(a, b int)) {
//		for _, pair := range sharedEdges { join(pair[0], pair[1]) }
//	})
func ComponentGroups(ids []int, link func(join func(a, b int))) [][]int {
	parent := make(map[int]int, len(ids))
	var find func(x int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	for _, id := range ids {
		parent[id] = id
	}
	link(func(a, b int) {
		if _, ok := parent[a]; !ok {
			return
		}
		if _, ok := parent[b]; !ok {
			return
		}
		parent[find(a)] = find(b)
	})
	return collectComponents(ids, find)
}

// collectComponents gathers ids by union-find root, in first-seen order.
func collectComponents(ids []int, find func(int) int) [][]int {
	groups := map[int][]int{}
	order := []int{}
	for _, id := range ids {
		r := find(id)
		if _, seen := groups[r]; !seen {
			order = append(order, r)
		}
		groups[r] = append(groups[r], id)
	}
	out := make([][]int, 0, len(order))
	for _, r := range order {
		out = append(out, groups[r])
	}
	return out
}
