// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// abuttingProfileGroups partitions profiles into connected components by SHARED edges: two profiles
// abut when one carries a directed boundary edge whose reverse the other carries — the signature of
// two arrangement cells meeting along that edge. Disjoint profiles are their own singleton group.
// Order-preserving: a group lists its members in ascending profile index, and groups come out in
// first-member order, so prism lineage is stable across recomputes.
//
// Grouping (not a single global merge) is what lets a mixed selection — a slot with corner-relief
// discs abutting it, plus unrelated disjoint bores — dissolve ONLY the touching cluster while every
// disjoint bore keeps its own analytic-cylinder prism (#38, and #33's per-region rule for the rest).
func abuttingProfileGroups(profiles []*sketch.Profile) [][]int {
	edges := map[[2]ptKey]int{} // welded directed edge → profile that owns it
	uf := newUnionFind(len(profiles))
	for pi, p := range profiles {
		poly := ccwPolygon(p.OuterLoop().Polygon())
		for i, n := 0, len(poly); i < n; i++ {
			a, b := weldKey(poly[i]), weldKey(poly[(i+1)%n])
			edges[[2]ptKey{a, b}] = pi
			if other, ok := edges[[2]ptKey{b, a}]; ok {
				uf.union(pi, other)
			}
		}
	}
	return uf.groups()
}

// dissolveGroup fuses a group's profiles into their union outline(s), returning the merged polygons
// and true only when the group has ≥2 profiles, every one is hole-free (the dissolve is on outer
// loops; a hole-carrying profile keeps the per-prism hole-honouring path), and the merge CONSERVES
// area (a non-conserving result means the cells overlapped rather than abutted — not our case, so use
// the safe path). A singleton returns false, so a lone bore keeps its analytic-cylinder prism.
func dissolveGroup(profiles []*sketch.Profile, group []int) ([][]math.Point2, bool) {
	if len(group) < 2 {
		return nil, false
	}
	polys := make([][]math.Point2, 0, len(group))
	var want float64
	for _, pi := range group {
		p := profiles[pi]
		if len(p.InnerLoops()) > 0 {
			return nil, false
		}
		poly := ccwPolygon(p.OuterLoop().Polygon())
		if len(poly) < 3 {
			return nil, false
		}
		polys = append(polys, poly)
		want += polygonArea2D(poly)
	}
	merged := ops.MergeAbuttingLoops(polys)
	if len(merged) == 0 || len(merged) >= len(polys) {
		return nil, false
	}
	var got float64
	for _, m := range merged {
		if len(m) < 3 {
			return nil, false
		}
		got += polygonArea2D(m)
	}
	if stdmath.Abs(got-want) > 1e-6*want {
		return nil, false
	}
	return merged, true
}

// ccwPolygon returns poly wound counter-clockwise, the shared winding both the abutment test and
// MergeAbuttingLoops need so a shared edge appears in both directions and cancels.
func ccwPolygon(poly []math.Point2) []math.Point2 {
	if outwardSign(poly) < 0 {
		return reversePoly(poly)
	}
	return poly
}

// polygonArea2D is the polygon's unsigned area (shoelace).
func polygonArea2D(poly []math.Point2) float64 {
	var a float64
	for i, n := 0, len(poly); i < n; i++ {
		j := (i + 1) % n
		a += poly[i].X*poly[j].Y - poly[j].X*poly[i].Y
	}
	return stdmath.Abs(a) / 2
}

// ptKey is a welded sketch-plane point: two arrangement cells' identically-sampled shared edge welds
// to the same key so the abutment test sees a directed edge and its reverse.
type ptKey [2]int64

// weldKey quantizes a point to a grid far below any real feature size yet well above float noise on
// equal samples.
func weldKey(p math.Point2) ptKey {
	const grid = 1e-7
	return ptKey{int64(stdmath.Round(float64(p.X) / grid)), int64(stdmath.Round(float64(p.Y) / grid))}
}

// unionFind is a tiny disjoint-set over profile indices for [abuttingProfileGroups].
type unionFind struct{ parent []int }

func newUnionFind(n int) *unionFind {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &unionFind{parent: p}
}

func (u *unionFind) find(i int) int {
	for u.parent[i] != i {
		u.parent[i] = u.parent[u.parent[i]]
		i = u.parent[i]
	}
	return i
}

func (u *unionFind) union(a, b int) { u.parent[u.find(a)] = u.find(b) }

// groups returns the members of each set, each ascending, ordered by first member.
func (u *unionFind) groups() [][]int {
	byRoot := map[int][]int{}
	var order []int
	for i := range u.parent {
		r := u.find(i)
		if _, ok := byRoot[r]; !ok {
			order = append(order, r)
		}
		byRoot[r] = append(byRoot[r], i)
	}
	out := make([][]int, 0, len(order))
	for _, r := range order {
		out = append(out, byRoot[r])
	}
	return out
}
