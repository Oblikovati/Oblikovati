// SPDX-License-Identifier: GPL-2.0-only

// This file is the keystone 2D arrangement of the planar boolean (the package
// comment lives in doc.go — #1669, M40 audit D12): it subdivides a set of
// undirected segments into the bounded faces they enclose (with holes), which the
// 3D boolean uses to split each planar face by its imprint segments.

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// arrTol is the planar-arrangement coincidence/intersection tolerance (database units).
//
// tol:calibrated — the 2D arrangement welds points computed by EXACT planar segment intersection
// (no accumulated curved-surface error), so this matched set (arrTol / tjTol / the welder grid) is
// scale-robust as an absolute across the validated µm→km range (ops TestScaleSweepInvariance).
// Relativising the welder grid to size·ε coarsens it on a large part and risks merging distinct
// arrangement vertices — a net regression — so under #1399 it stays absolute and validated.
const arrTol = 1e-9 // tol:calibrated — exact-planar-intersection weld; see the note above

// parallelDenomTol is the magnitude below which a line/ray·edge or line/ray·plane denominator
// is treated as zero — the two are parallel, so there is no single crossing. Below arrTol
// because it bounds a cross/dot product of (roughly unit) directions, not a length.
const parallelDenomTol = 1e-12 // tol:numeric — cross/dot denominator of unit directions

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
	cells, _ := ArrangeChecked(segments)
	return cells
}

// ArrangeChecked is [Arrange] with the T-junction pass's convergence reported. ok=false means the
// subdivision hit [tjSplitBudget] and the cell complex CANNOT be trusted — the caller must decline,
// never use the cells. Arrange returns them anyway for the callers that predate this and cannot act
// on the answer; every caller that can refuse should use this one.
func ArrangeChecked(segments [][2]math.Point2) ([]Face2D, bool) {
	pts, edges, converged := planarize(segments)
	if !converged {
		return nil, false
	}
	if len(edges) == 0 {
		return nil, true
	}
	cycles := traceCycles(pts, edges)
	return nestFaces(cycles), true
}

// planarize splits every segment at its intersections with the others and welds the
// resulting points, returning the welded points and the elementary (crossing-free)
// undirected edges as index pairs. Pair candidacy comes from a uniform grid hash over the
// segments' padded AABBs (#1607), retiring the O(S²) all-pairs scan; the narrow phase and
// its ordering are unchanged, so the arrangement is identical.
func planarize(segments [][2]math.Point2) ([]math.Point2, [][2]int, bool) {
	weld := newWelder()
	edges := map[[2]int]bool{}
	cull := newSegmentCullGrid(segments)
	for i, seg := range segments {
		for _, e := range splitOne(seg, segments, cull.candidates(i), weld) {
			if e[0] != e[1] {
				edges[canonEdge(e[0], e[1])] = true
			}
		}
	}
	converged := splitTJunctions(weld.points, edges)
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
	return weld.points, out, converged
}

// tjTol bounds the perpendicular distance at which a welded vertex counts as lying ON an
// edge — a T-junction. It matches the welder's coincidence grid (1e-7): any point that would
// weld onto the line is at most that far off it, while genuine features sit orders of
// magnitude further (≥1e-2), so this never splits a near-miss.
const tjTol = 1e-7 // tol:calibrated — matches the welder grid; see arrTol

// splitTJunctions subdivides every edge at any welded vertex lying strictly on its interior,
// repeating until stable. splitOne only cuts at proper interior crossings of two segments;
// when one segment merely ENDS on another's interior (a T-junction — e.g. a coplanar imprint
// chain clipped to land exactly on a hole-loop edge, #860), the touch point welds as a vertex
// but the host edge is left whole, so the chain dangles and the face never partitions. This
// pass welds such chains shut, the crux of robust planar arrangement under faceted-curve cuts.
func splitTJunctions(pts []math.Point2, edges map[[2]int]bool) bool {
	// The welded point set is fixed here (only edges split), so one grid hash over it culls
	// every vertex-on-edge scan below (#1607).
	verts := newVertexCullGrid(pts)
	budget := tjSplitBudget(len(pts))
	for changed := true; changed; {
		changed = false
		for e := range edges {
			c := vertexOnEdgeInterior(pts, e[0], e[1], verts)
			if c < 0 {
				continue
			}
			if budget--; budget < 0 {
				return false // churning, not converging: see tjSplitBudget
			}
			delete(edges, e)
			edges[canonEdge(e[0], c)] = true
			edges[canonEdge(c, e[1])] = true
			changed = true
		}
	}
	return true
}

// tjSplitBudget is the PROVABLE bound on how many T-junction splits a converging run can make, not a
// tuned number. Each split replaces one edge with two whose endpoints are existing welded vertices,
// so it strictly grows a SET keyed by canonical index pairs; a set of undirected pairs over n
// vertices holds at most n(n−1)/2 members, so at most that many splits can ever add anything. A run
// that exceeds it is not subdividing towards a fixed point, it is revisiting pairs it already has.
//
// It has to exist because the loop's termination argument silently depends on scale. tjTol is an
// ABSOLUTE 1e-7, and it is used twice over: as a perpendicular DISTANCE to the edge and as a
// dimensionless bound on the parameter t along it. On geometry whose own features are near 1e-7 —
// measured: the RING corpus body cut by an axial drill of radius 1.585e-7 — those two readings stop
// agreeing, the pass keeps finding "interior" vertices on edges it has just made, and the boolean
// never returns. A hang is neither a refusal nor a wrong body, and the ground rules admit only those
// two (ADR-0061 stage 6, review round 2).
func tjSplitBudget(n int) int { return n * (n - 1) / 2 }

// vertexOnEdgeInterior returns a vertex index lying strictly inside segment a→b (within
// [tjTol] of it, parameter away from both ends), or −1 if none. The lowest such index is
// returned for determinism. Candidates come from the vertex grid hash over the edge's padded
// box (#1607) — every qualifying vertex lies within tjTol of the edge, so none can escape it —
// with the qualification arithmetic unchanged from the retired full scan.
func vertexOnEdgeInterior(pts []math.Point2, a, b int, verts *vertexCullGrid) int {
	pa, pb := pts[a], pts[b]
	ab := pa.VectorTo(pb)
	lenSq := ab.LengthSquared()
	if lenSq < tjTol*tjTol {
		return -1
	}
	best := -1
	x0 := min(float64(pa.X), float64(pb.X)) - tjCullPad
	y0 := min(float64(pa.Y), float64(pb.Y)) - tjCullPad
	x1 := max(float64(pa.X), float64(pb.X)) + tjCullPad
	y1 := max(float64(pa.Y), float64(pb.Y)) + tjCullPad
	verts.eachInBox(x0, y0, x1, y1, func(c int) {
		if c == a || c == b || (best >= 0 && c >= best) {
			return
		}
		t := pa.VectorTo(pts[c]).Dot(ab) / lenSq
		if t <= tjTol || t >= 1-tjTol {
			return
		}
		if pa.TranslateBy(ab.Scale(t)).DistanceTo(pts[c]) > tjTol {
			return
		}
		best = c
	})
	return best
}

// splitOne returns a segment's elementary edges: the chain of welded vertex indices along
// it, split at every interior intersection with another segment. `cand` is the segment's
// grid-culled candidate list (ascending, self excluded, #1607): a superset of every j the
// narrow phase can accept, in the retired brute scan's order — the cut list feeds an
// unstable sort, so insertion order must not drift.
func splitOne(seg [2]math.Point2, all [][2]math.Point2, cand []int, weld *welder) [][2]int {
	si := geom.NewLineSegment2d(seg[0], seg[1])
	type cut struct {
		t float64
		p math.Point2
	}
	cuts := []cut{{0, seg[0]}, {1, seg[1]}}
	for _, j := range cand {
		other := all[j]
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
	const grid = 1e-7 // tol:calibrated — the arrangement welder coincidence grid; see arrTol
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
