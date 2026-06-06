// SPDX-License-Identifier: GPL-2.0-only

// Package brep implements the planar-faced B-rep boolean (M20·F01, the Option-A kernel):
// imprint face–face intersections, split faces along them via a 2D planar arrangement,
// classify the sub-faces against the other solid, and stitch the kept faces into a clean,
// low-face-count, chainable B-rep. This file is the keystone 2D arrangement: it subdivides
// a set of undirected segments into the bounded faces they enclose (with holes), which the
// 3D boolean uses to split each planar face by its imprint segments.
package brep

import (
	stdmath "math"
	"sort"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// arrTol is the planar-arrangement coincidence/intersection tolerance (database units).
const arrTol = 1e-9

// parallelDenomTol is the magnitude below which a line/ray·edge or line/ray·plane denominator
// is treated as zero — the two are parallel, so there is no single crossing. Below arrTol
// because it bounds a cross/dot product of (roughly unit) directions, not a length.
const parallelDenomTol = 1e-12

// Face2D is one region of a planar arrangement: a counter-clockwise outer loop and any
// clockwise hole loops nested directly inside it.
type Face2D struct {
	Outer []math.Point2
	Holes [][]math.Point2
}

// Arrange computes the planar subdivision induced by the undirected segments and returns
// the bounded faces (the regions they enclose), each with its holes. The unbounded outer
// region is excluded. Segments are split at every interior crossing and coincident
// endpoints are welded, so the result is a valid cell complex.
func Arrange(segments [][2]math.Point2) []Face2D {
	pts, edges := planarize(segments)
	if len(edges) == 0 {
		return nil
	}
	cycles := traceCycles(pts, edges)
	return nestFaces(cycles)
}

// planarize splits every segment at its intersections with the others and welds the
// resulting points, returning the welded points and the elementary (crossing-free)
// undirected edges as index pairs.
func planarize(segments [][2]math.Point2) ([]math.Point2, [][2]int) {
	weld := newWelder()
	edges := map[[2]int]bool{}
	for i, seg := range segments {
		for _, e := range splitOne(seg, segments, i, weld) {
			if e[0] != e[1] {
				edges[canonEdge(e[0], e[1])] = true
			}
		}
	}
	out := make([][2]int, 0, len(edges))
	for e := range edges {
		out = append(out, e)
	}
	// Sort for determinism (map iteration order is random and was producing
	// run-to-run-different arrangements on tolerance-fragile inputs).
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return weld.points, out
}

// splitOne returns segment i's elementary edges: the chain of welded vertex indices along
// it, split at every interior intersection with another segment.
func splitOne(seg [2]math.Point2, all [][2]math.Point2, i int, weld *welder) [][2]int {
	si := geom.NewLineSegment2d(seg[0], seg[1])
	type cut struct {
		t float64
		p math.Point2
	}
	cuts := []cut{{0, seg[0]}, {1, seg[1]}}
	for j, other := range all {
		if j == i {
			continue
		}
		if p, s, _, ok := geom.Segment2dIntersection(si, geom.NewLineSegment2d(other[0], other[1]), arrTol); ok && s > arrTol && s < 1-arrTol {
			cuts = append(cuts, cut{s, p})
		}
	}
	sort.Slice(cuts, func(a, b int) bool { return cuts[a].t < cuts[b].t })
	var chain [][2]int
	prev := weld.add(cuts[0].p)
	for k := 1; k < len(cuts); k++ {
		cur := weld.add(cuts[k].p)
		if cur != prev {
			chain = append(chain, [2]int{prev, cur})
			prev = cur
		}
	}
	return chain
}

// welder merges coincident points onto a shared index list (a coincidence grid).
type welder struct {
	index  map[[2]int64]int
	points []math.Point2
}

func newWelder() *welder { return &welder{index: map[[2]int64]int{}} }

func (w *welder) add(p math.Point2) int {
	const grid = 1e-7
	k := [2]int64{int64(stdmath.Round(p.X / grid)), int64(stdmath.Round(p.Y / grid))}
	if i, ok := w.index[k]; ok {
		return i
	}
	w.index[k] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}

func canonEdge(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// signedArea2D returns the shoelace signed area of a loop (positive = counter-clockwise).
func signedArea2D(loop []math.Point2) float64 {
	a := 0.0
	for i, n := 0, len(loop); i < n; i++ {
		j := (i + 1) % n
		a += loop[i].X*loop[j].Y - loop[j].X*loop[i].Y
	}
	return a / 2
}

// pointInPolygon2D reports whether p is strictly inside the polygon (even–odd ray cast).
func pointInPolygon2D(p math.Point2, poly []math.Point2) bool {
	in := false
	for i, n := 0, len(poly); i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := a.X + (p.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X)
			if p.X < x {
				in = !in
			}
		}
	}
	return in
}
