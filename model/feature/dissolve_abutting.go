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
// and true only when the group has ≥2 profiles and the merge of their OUTER loops CONSERVES area (a
// non-conserving result means the cells overlapped rather than abutted — not our case, so use the safe
// path). A singleton returns false, so a lone bore keeps its analytic-cylinder prism. Any inner (hole)
// loops the group's profiles carry are gathered and assigned to the merged region that contains them,
// so a hole-carrying dog-bone (a relieved slot with a bore in it) dissolves too — the blind-pocket
// bottom that cracked BigChunkyPlate at z=1.8 — instead of falling back to the coincident-wall path.
type mergedRegion struct {
	outer  []math.Point2
	inners []sketch.Loop
}

func dissolveGroup(profiles []*sketch.Profile, group []int) ([]mergedRegion, bool) {
	if len(group) < 2 {
		return nil, false
	}
	polys := make([][]math.Point2, 0, len(group))
	var inners []sketch.Loop
	var want float64
	for _, pi := range group {
		p := profiles[pi]
		poly := ccwPolygon(p.OuterLoop().Polygon())
		if len(poly) < 3 {
			return nil, false
		}
		polys = append(polys, poly)
		want += polygonArea2D(poly)
		inners = append(inners, p.InnerLoops()...)
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
	regions := make([]mergedRegion, len(merged))
	for i, m := range merged {
		regions[i] = mergedRegion{outer: m}
	}
	for _, in := range inners {
		idx := regionContaining(regions, in)
		if idx < 0 {
			return nil, false // a hole outside every merged outline: bail to the safe per-profile path
		}
		regions[idx].inners = append(regions[idx].inners, in)
	}
	return regions, true
}

// regionContaining returns the index of the merged region whose outer polygon contains the loop's
// first point, or -1 when none does.
func regionContaining(regions []mergedRegion, loop sketch.Loop) int {
	poly := loop.Polygon()
	if len(poly) == 0 {
		return -1
	}
	for i, r := range regions {
		if pointInPolygon2D(poly[0], r.outer) {
			return i
		}
	}
	return -1
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
