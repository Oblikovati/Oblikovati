// SPDX-License-Identifier: GPL-2.0-only

// Package disjoint is the union-find over a parent slice that several operation families
// share: sewing clusters coincident vertices with it, the identical-bodies signature
// groups faces, the analytic face integral groups loops.
//
// It lived in sew.go because that is where the first caller was, which made a
// disjoint-set lookup a reason to depend on the whole operation layer.
//
//	parent := make([]int, n)
//	for i := range parent { parent[i] = i }
//	disjoint.Union(parent, i, j)
package disjoint

// union joins i's and j's clusters (plain union-find with path squash on find).
func Union(parent []int, i, j int) {
	ri, rj := Find(parent, i), Find(parent, j)
	if ri != rj {
		parent[rj] = ri
	}
}

func Find(parent []int, i int) int {
	for parent[i] != i {
		parent[i] = parent[parent[i]]
		i = parent[i]
	}
	return i
}
