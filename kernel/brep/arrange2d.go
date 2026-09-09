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
// (no accumulated curved-surface error), so this matched set (arrTol / the tjTol floor / the welder
// grid) is scale-robust as an absolute across the validated µm→km range (ops TestScaleSweepInvariance).
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

// ArrangeChecked computes the planar subdivision induced by the undirected segments and returns
// the bounded faces (the regions they enclose), each with its holes. The unbounded outer
// region is excluded. Segments are split at every interior crossing and coincident
// endpoints are welded, so the result is a valid cell complex.
//
// ok=false means the T-junction pass hit [tjSplitBudget] and the cell complex CANNOT be trusted: the
// caller must decline or report, never use the cells.
//
// There is deliberately no unchecked sibling. One existed for a single commit, returning the cells and
// discarding the flag "for the callers that cannot act on the answer" — and both of its production
// callers COULD act on it. The planar boolean's splitFace read nil cells as "this face has no material
// sub-faces" and dropped the face with no error and no diagnostic, leaving ops.Validate to report an
// open body whose cause had been erased; the tessellator's overlapping-hole path dropped every cell of
// a face and meshed nothing. Making the flag impossible to discard is what stops that recurring
// (ADR-0061 stage 6, review round 3).
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
	return weld.points, sortedEdgePairs(edges), converged
}

// sortedEdgePairs is the edge set in one total order — by first index, then second. Map iteration
// order is random and was producing run-to-run-different arrangements on tolerance-fragile inputs;
// every walk over the set, the T-junction pass included, reads it through this.
func sortedEdgePairs(edges map[[2]int]bool) [][2]int {
	out := make([][2]int, 0, len(edges))
	for e := range edges {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}

// tjTol is the FLOOR of the T-junction on-edge distance (see [tjOnEdgeTol]), and the value the
// pass's small-scale behaviour is calibrated at. It is 100 × the welder grid — the same ratio
// geom.Resolution keeps between Plane() and Weld() — so a point that welded onto the line is two
// orders inside it, while genuine features sit orders of magnitude further (≥1e-2).
//
// It is a FLOOR rather than the tolerance because arrTol is an ABSOLUTE under #1399: on an
// arrangement smaller than one database unit a purely relative on-edge distance would drop below
// the welder grid that generates the very offsets it has to absorb, and a welded chain endpoint
// would stop counting as lying on its host edge (#3513).
//
// Its downward consequence, named because it is a real cliff: held absolute, the on-edge distance is a
// GROWING FRACTION of a shrinking arrangement — 1% of the extent at E = 1e-5 — and by E ≈ 2e-7 every
// edge is either below the degeneracy check (lenSq < tol²) or has tEndPad ≥ ½, so the T-junction pass
// does nothing at all. That is where we want to be, because a body whose whole arrangement is 2e-7
// across is refused upstream by the boolean's size classification (geom.Resolution.Resolves) long
// before it gets here — but the pass going quiet is a floor effect, not a property of the input.
const tjTol = 100 * arrTol // tol:calibrated — 100 × the absolute welder grid; see arrTol

// tjOnEdgeTol is the perpendicular DISTANCE at which a welded vertex counts as lying ON an edge —
// a T-junction. It is measured in the arrangement's OWN frame — metric for a projected planar face,
// parametric for a (u,v) band — and the comparison it serves has both operands in that same frame,
// which is what the single-class rule asks. So it reads the on-line member of the arrangement's
// [geom.Resolution] (Plane: exactly "how far a point may sit from a segment and still count as on
// it") over that frame's own extent, floored at [tjTol].
//
// Before #3513 this was the bare tjTol, used BOTH as this distance and as a bound on the
// dimensionless parameter t along the edge — one constant read in two classes, which is the
// comparison the ground rules forbid. The parameter reading now converts through the edge's
// |dP/dt| in [vertexOnEdgeInterior]; this is the length reading.
func tjOnEdgeTol(res geom.Resolution) float64 { return max(res.Plane(), tjTol) }

// tjCullPad inflates an edge's query box in the T-junction pass: 10 × the on-edge distance
// tolerance, so every vertex within it of the edge is guaranteed to be visited.
func tjCullPad(tol float64) float64 { return 10 * tol }

// splitTJunctions subdivides every edge at any welded vertex lying strictly on its interior,
// repeating until stable. splitOne only cuts at proper interior crossings of two segments;
// when one segment merely ENDS on another's interior (a T-junction — e.g. a coplanar imprint
// chain clipped to land exactly on a hole-loop edge, #860), the touch point welds as a vertex
// but the host edge is left whole, so the chain dangles and the face never partitions. This
// pass welds such chains shut, the crux of robust planar arrangement under faceted-curve cuts.
//
// Each pass walks a SORTED snapshot of the set, never the live map. The budget below counts the
// pair-adding splits in the order they are made, and whether a given split adds a pair depends on
// which splits came before it — so on a converging input near the budget, walking the map in its
// random order made decline-versus-converge a run-to-run coin toss (final fix wave, finding 7). Halves
// added during a pass are not in its snapshot; the next pass takes them.
func splitTJunctions(pts []math.Point2, edges map[[2]int]bool) bool {
	// The welded point set is fixed here (only edges split), so one grid hash over it culls
	// every vertex-on-edge scan below (#1607). The on-edge distance comes from the arrangement's
	// OWN 2D extent — the frame the points live in, metric for a planar face split and
	// parametric for a (u,v) band — floored at tjTol (#3513).
	verts := newVertexCullGrid(pts)
	tol := tjOnEdgeTol(geom.ResolutionForPoints2D(pts))
	budget := tjSplitBudget(len(pts))
	for {
		changed, ok := splitTJunctionPass(pts, edges, verts, tol, &budget)
		if !ok {
			return false // churning, not converging: see tjSplitBudget
		}
		if !changed {
			return true
		}
	}
}

// splitTJunctionPass is one walk over a sorted snapshot of the edge set, splitting each edge at the
// lowest welded vertex on its interior. It reports whether it changed anything and whether the budget
// survived. Halves added during a pass are not in its snapshot; the next pass takes them.
func splitTJunctionPass(pts []math.Point2, edges map[[2]int]bool, verts *vertexCullGrid, tol float64, budget *int) (changed, ok bool) {
	for _, e := range sortedEdgePairs(edges) {
		c := vertexOnEdgeInterior(pts, e[0], e[1], verts, tol)
		if c < 0 {
			continue
		}
		if !splitEdgeAt(edges, e, c, budget) {
			return changed, false
		}
		changed = true
	}
	return changed, true
}

// splitEdgeAt replaces edge e with its two halves through vertex c, charging the budget when the
// split ADDS a pair. It returns false when the budget is exhausted.
//
// A split that adds a pair is the only kind that can run away, and the budget counts exactly
// those. One that adds neither half strictly shrinks the set (it removes e and re-adds two pairs
// already in it), so it cannot loop on its own account and is free.
func splitEdgeAt(edges map[[2]int]bool, e [2]int, c int, budget *int) bool {
	lo, hi := canonEdge(e[0], c), canonEdge(c, e[1])
	if !edges[lo] || !edges[hi] {
		if *budget--; *budget < 0 {
			return false
		}
	}
	delete(edges, e)
	edges[lo] = true
	edges[hi] = true
	return true
}

// tjSplitBudget is the ENFORCED bound on how many PAIR-ADDING T-junction splits a run may make; the
// termination argument is the bound itself plus one fact about the other kind of split, and the
// number n(n−1)/2 is the size the bound is set to, not a theorem about the pass. The argument:
//
//   - A split replaces one edge with two whose endpoints are existing welded vertices, so every edge
//     the pass can ever hold is one of the n(n−1)/2 unordered index pairs over the n welded points.
//   - A split that adds at least one pair not currently in the set is counted against the budget, so
//     there are at most n(n−1)/2 of them. (Pairs CAN be deleted and re-added, so this is not "each pair
//     is added once"; it is that a run needing more pair-adding splits than there are distinct pairs
//     has re-added a pair it already removed, which is churn, not progress.)
//   - A split that adds neither half only removes an edge, strictly shrinking a finite set, so between
//     two pair-adding splits the pass makes finitely many of them and is not counted.
//
// Hence the pass terminates, and exceeding the budget is the failure below.
//
// Counting DISTINCT pairs ever added instead — which would make n(n−1)/2 a theorem — was considered and
// rejected (final fix wave, finding 7): such a count is bounded by n(n−1)/2 by construction, so it can
// never exceed the budget and the pass would never decline. The runaway this bound exists for IS
// re-adding: a vertex that did not qualify on an edge qualifies on the shorter half that replaces it.
// What makes the decline honest is that the count is taken in ONE order (splitTJunctions walks a
// sorted snapshot), so decline-versus-converge is a function of the input alone.
//
// WHERE THE RUNAWAY CAME FROM, and why the bound stays anyway. It was ONE tolerance read in two
// classes: tjTol was compared both as a perpendicular DISTANCE to the edge and as a bound on the
// dimensionless parameter t along it. A fixed t-pad is a length only on a unit-length edge, so on an
// edge shorter than a database unit it excluded almost nothing near the ends, and a vertex a hair
// inside an end kept qualifying on every shorter half the split produced — measured on the RING corpus
// body cut by an axial drill of radius 1.585e-7, which did not return in any budget the suite could
// give it. #3513 converts the parameter reading through the edge's |dP/dt| (vertexOnEdgeInterior), so
// the end exclusion is a length on every edge and each half is strictly longer than the tolerance.
//
// Measured over the kernel/brep + kernel/ops suites (24694 arrangements, -count=1 -v, clean trees):
// before #3513, 42 runs exhausted this budget and 5100 splits were made, 5082 of them inside those 42
// runs; after, 0 runs exhaust it and 29 splits are made in total. So the churn is gone — but the
// CONVERGING path moves too, 18 → 29 splits, in both directions (kernel/brep 9 → 5, kernel/ops/boolean
// 9 → 24), and the two directions are different things. See ADR-0061's "G9 closed" section for the
// per-incidence measurement: what is lost was never a weld (splits taken 1.6e-8…5.4e-8 from an
// endpoint of an edge 0.014–0.10 long — inside the on-edge tolerance of that endpoint, so it IS that
// endpoint), and what is gained is interior crossings at 80–94% of the new tolerance which the old
// absolute could not see on a 5.8-unit arrangement. The gained splits fix nothing: the bodies they
// touch are valid, closed and diagnostic-free on BOTH sides, half of them are no-ops on the result,
// and the rest move one body by ten vertices and 2.7e-9 of its volume. This half of the change is
// carried by the RULE (one class per comparison), not by an outcome.
//
// The bound stays because it is the pass's TERMINATION argument, not a patch for that one input:
// "until stable" is a fixpoint loop over a set the pass itself grows, and a bound plus a named decline
// is the only thing that keeps a future conditioning failure from becoming a hang — the outcome the
// ground rules do not admit (ADR-0061 stage 6, review round 2). The argument itself never depended on
// the value of tol, and the runaway it bounds (a pair deleted and later re-added) is untouched here;
// only the short-edge re-qualification cascade is removed.
func tjSplitBudget(n int) int { return n * (n - 1) / 2 }

// vertexOnEdgeInterior returns a vertex index lying strictly inside segment a→b — within `tol`
// of it perpendicularly, and further than `tol` ALONG it from either end — or −1 if none. The
// lowest such index is returned for determinism. Candidates come from the vertex grid hash over
// the edge's padded box (#1607) — every qualifying vertex lies within `tol` of the edge, so none
// can escape it — with the qualification arithmetic otherwise unchanged from the retired full scan.
//
// Both readings of `tol` here are extents in the arrangement's frame; neither is a bare parameter.
// The end exclusion used to compare the dimensionless parameter t against the same absolute (#3513):
// a fixed t-pad is a length only on a unit-length edge, so on a short edge it excluded nothing and a
// vertex a hair inside the end kept qualifying on each shorter half the split produced — the churn
// tjSplitBudget exists to stop. tEndPad converts through the segment's |dP/dt|, which for the chord
// a→b is the constant |ab|, so the conversion is exact up to one Sqrt.
//
// What that buys is INVARIANCE: `t ≤ tol/|ab|` is the same predicate as "projected distance from pa
// ≤ tol", which does not mention the edge's length — so a vertex excluded for sitting within tol of
// an endpoint stays excluded on every sub-edge carrying that endpoint, and the re-qualification
// cascade has nowhere to start. (Exact up to the child edge's direction change: the split vertex may
// sit tol off the parent's line, turning the child by ≤ tol/L′ radians and moving the projection foot
// by O(tol²/L′) — second order, and far inside the tolerance it is compared against.)
func vertexOnEdgeInterior(pts []math.Point2, a, b int, verts *vertexCullGrid, tol float64) int {
	pa, pb := pts[a], pts[b]
	ab := pa.VectorTo(pb)
	lenSq := ab.LengthSquared()
	if lenSq < tol*tol {
		return -1 // shorter than the tolerance: it has no interior to split at
	}
	tEndPad := tol / stdmath.Sqrt(lenSq)
	best := -1
	x0, y0, x1, y1 := paddedEdgeBox(pa, pb, tjCullPad(tol))
	verts.eachInBox(x0, y0, x1, y1, func(c int) {
		if c == a || c == b || (best >= 0 && c >= best) {
			return
		}
		if onEdgeInteriorAt(pa, ab, lenSq, pts[c], tEndPad, tol) {
			best = c
		}
	})
	return best
}

// onEdgeInteriorAt is the qualification itself: p projects into the edge's interior (further than
// tEndPad from either end in parameter, which is `tol` in the frame — see [vertexOnEdgeInterior]) and
// sits within `tol` of the line perpendicularly.
//
// It states the offset test POSITIVELY (`<= tol`) where the inlined original rejected on `> tol`. On
// every finite input the two are the same predicate; they differ only on a NaN distance, which the
// reject-form ACCEPTED as a T-junction (`NaN > tol` is false, so nothing rejected it) and this form
// declines. That is a deliberate divergence from the pre-#3513 shape, not an oversight: a vertex whose
// distance to the edge is not a number cannot be shown to lie on it, and admitting it would split an
// edge at a point the pass knows nothing about. Pinned by TestANaNVertexIsNotOnAnEdge.
func onEdgeInteriorAt(pa math.Point2, ab math.Vector2, lenSq float64, p math.Point2, tEndPad, tol float64) bool {
	t := pa.VectorTo(p).Dot(ab) / lenSq
	if t <= tEndPad || t >= 1-tEndPad {
		return false
	}
	return pa.TranslateBy(ab.Scale(t)).DistanceTo(p) <= tol
}

// paddedEdgeBox is the edge's AABB grown by pad on every side — the box the vertex grid is queried
// over, sized so no vertex the narrow phase could accept falls outside it.
func paddedEdgeBox(pa, pb math.Point2, pad float64) (x0, y0, x1, y1 float64) {
	return min(float64(pa.X), float64(pb.X)) - pad, min(float64(pa.Y), float64(pb.Y)) - pad,
		max(float64(pa.X), float64(pb.X)) + pad, max(float64(pa.Y), float64(pb.Y)) + pad
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
