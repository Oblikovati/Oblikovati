// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "oblikovati.org/math"

// The BOUNDARY half of the chart-driven mesher's classification (Oblikovati/Oblikovati#3518).
//
// The region comes from the chart and the mesh's boundary comes from the shared edges, and those are
// two samplings of the SAME trim curve taken at different times. The chart's is the boolean's own
// record, written once when the face was wound (ADR-0063); the shared edge's is re-taken at whatever
// quality the caller asks for. So there is always a faceting fine enough for the boundary to
// out-resolve the region, and where the curve turns sharply that gap is not a hair's width.
//
// Measured on the #1818 near-pinch crossing rods — r = 3 joined to r = 3.00004, whose merged wall
// carries two lens windows that pass 0.031 mm apart — at PropertyQuality: in the pinch neighbourhood
// (u > 6.27, |v - 6| < 0.05) the face's chart contour carries ONE point and its shared edge carries
// 63. All 56 edge points across the pinch read MATERIAL against the chart, up to 2.05e-3 in (u,v)
// outside its contour, while lying inside the lens the mesh is bounded by. The covering triangulates
// that band, the centroid test keeps it, and the face ships a skin inside its own hole: 112 of each
// lens chain's 380 segments came back bounded by TWO triangles instead of one.
//
// The chart may not be asked a question finer than its own sampling, so the answer is not a finer
// query. It is that the BOUNDARY decides the triangles it bounds. A constrained triangulation puts
// every triangle wholly on one side of every constraint, and a face's loop is wound consistently, so
// "material lies on the left of this chain" is ONE bit per chain. That bit is read off the chart
// where the chart and the boundary already agree — the rim segments the chart bounded exactly once —
// and then enforced on every segment of that chain, including the ones in the band where the chart
// is wrong. The vote is per CHAIN and not per face: rimBoundedWindowedWall's window hole is wound the
// other way from its two rims, and a single per-face bit threw that fixture's window away
// (18 unpaired edges, 9 rim segments unbound).
//
// Both directions are enforced and both were needed. Dropping the wrong-side triangle alone left the
// segments where the chart had kept ONLY the wrong-side one bounded by nothing — measured, R = 30
// with dr = 4e-4 at DefaultQuality had 5 of those. So the rule is symmetrical: of a rim segment's two
// adjacent triangles the material-side one is kept and the other is not.
//
// THE MEASUREMENT. The near-pinch corpus is the eight JOIN bodies of kernel/ops/boolean's
// TestNearPinchCutJoinWatertight (R in {3, 30}, |dr|/R in {4e-5, 6e-5, 1.6e-4, 3.2e-4}), built
// through ops.Boolean. Each carries exactly one two-rim holed band and the eight CUT bodies carry
// none, so the corpus is 8 faces at two facetings = 16 rows. Driving chartFaceMesh on them and
// reading chartRimMismatch (rim segments the mesh does not bound / unpaired edges that are no rim
// segment), against the #3520 base fff94140:
//
//	                 rim segments unbound    unpaired edges that are no rim segment    rows exactly rim-bounded
//	base                          1873                                       637                     4 of 16
//	after                            0                                        38                    10 of 16
//
// Per face the area against query.AnalyticFaceArea is within 0.0003 % at PropertyQuality, where it
// was within 0.0024 %: the band was always thin, which is why only the rim gate ever saw it.
//
// The 38 that remain are NOT this defect and are all at the covering's own seam — see
// chart_face_replica.go, which names what is left and what it blocks.

// rimSegment is one boundary segment as the covering indexes it, with the chain that laid it.
type rimSegment struct{ a, b, chain int }

// directedRimSegments maps each directed covering segment of the boundary to the chain that laid it,
// keeping only the segments that are the face's RIM.
//
// The rim is read from chainSegmentKeys — the one place a rim is keyed — so this rule and the gate
// that judges its result cannot disagree about what the boundary is. Two things fall out of that and
// both were needed. A SLIT is dropped: a segment the boundary walks twice bounds material on both
// sides, and chainSegmentKeys drops it by its odd-count rule. And the count is taken on the WELDED 3D
// segment, not on the covering index pair, because a slit's two traversals are laid as SEPARATE
// covering vertices at the same 3D points — the piston head's merged cocylindrical wall arrives as one
// loop of 56 points whose seam runs up and back. An index-pair test sees two unrelated segments there,
// keeps both, and enforces opposite sides on the two triangles that share the slit: measured, it tore
// that wall (3 unpaired edges on the face, 32 free edges on the body at DefaultQuality).
//
// The rim is read from the chains ONCE, before replication, so a segment is judged by the boundary it
// belongs to and not by how many period images of it the pad happened to carry. Counting the replicas
// instead was tried and is wrong for exactly that reason: replication then reads as slitting, the
// pinch stretch of a lens is carried at two shifts and counts even, and the rule stops applying there
// — the near-pinch corpus went from 0 unbound rim segments back to 889.
func (b *chartCover) directedRimSegments(segs []rimSegment, chains []chartChain) map[[2]int]int {
	grid := weldGrid([][]math.Point3{b.pos})
	rim := chainSegmentKeys(chains, grid)
	out := make(map[[2]int]int, len(segs))
	for _, s := range segs {
		if rim[orderedSegmentKey(b.pos[s.a], b.pos[s.b], grid)] {
			out[[2]int{s.a, s.b}] = s.chain
		}
	}
	return out
}

// directedTriangleEdges is a CCW triangle's three edges in its own traversal order; the triangle lies
// to the LEFT of each of them.
func directedTriangleEdges(t [3]int) [3][2]int {
	return [3][2]int{{t[0], t[1]}, {t[1], t[2]}, {t[2], t[0]}}
}

// directedEdgeOwner is the triangle traversing each directed edge. A constrained triangulation is a
// manifold, so a directed edge has at most one.
func directedEdgeOwner(tris [][3]int) map[[2]int]int {
	out := make(map[[2]int]int, 3*len(tris))
	for i, t := range tris {
		for _, e := range directedTriangleEdges(t) {
			out[e] = i
		}
	}
	return out
}

// rimSide is which side of its own direction each boundary chain carries material on, and whether the
// chart's verdicts decided it at all.
type rimSide struct {
	left    []bool
	decided []bool
}

// bindToTheRim makes the kept set bounded by exactly the face's own boundary: for every boundary
// segment whose side the chart decided, the triangle on the material side is kept and the one on the
// other side is not.
func (b *chartCover) bindToTheRim(tris [][3]int, keep []bool) {
	side := b.materialSideOfEachChain(tris, keep)
	owner := directedEdgeOwner(tris)
	want, drop := make([]bool, len(tris)), make([]bool, len(tris))
	for e, ci := range b.rimChain {
		if side.decided[ci] {
			b.markRimSides(tris, owner, e, side.left[ci], want, drop)
		}
	}
	for i := range keep {
		keep[i] = (keep[i] || want[i]) && !drop[i]
	}
}

// markRimSides records which of a boundary segment's two adjacent triangles the material side wants and
// which it refuses. A wanted triangle outside the branch window is left alone: its own replica inside
// the window carries that segment there.
func (b *chartCover) markRimSides(tris [][3]int, owner map[[2]int]int, e [2]int, left bool, want, drop []bool) {
	in, out := e, [2]int{e[1], e[0]}
	if !left {
		in, out = out, in
	}
	if t, ok := owner[in]; ok && b.triangleInWindow(tris[t]) {
		want[t] = true
	}
	if t, ok := owner[out]; ok {
		drop[t] = true
	}
}

// triangleInWindow reports whether a triangle's centroid lies in the chart's branch window.
func (b *chartCover) triangleInWindow(t [3]int) bool {
	u, v := b.centroid(t)
	return b.r.windowCandidate(u, v)
}

// materialSideOfEachChain reads each chain's material side off the rim segments the chart ALREADY
// bounded exactly once — where its answer and the mesh's boundary agree — by majority.
func (b *chartCover) materialSideOfEachChain(tris [][3]int, keep []bool) rimSide {
	fwd := keptDirectedEdgeUse(tris, keep)
	left, right := make([]int, b.chains), make([]int, b.chains)
	for e, ci := range b.rimChain {
		switch back := [2]int{e[1], e[0]}; {
		case fwd[e] == 1 && fwd[back] == 0:
			left[ci]++
		case fwd[back] == 1 && fwd[e] == 0:
			right[ci]++
		}
	}
	return rimSideFrom(left, right)
}

// rimSideFrom turns each chain's two vote counts into its side and whether it has one.
func rimSideFrom(left, right []int) rimSide {
	s := rimSide{left: make([]bool, len(left)), decided: make([]bool, len(left))}
	for ci := range left {
		s.left[ci] = left[ci] > right[ci]
		s.decided[ci] = left[ci] != right[ci]
	}
	return s
}

// keptDirectedEdgeUse counts how many of the KEPT triangles traverse each directed edge.
func keptDirectedEdgeUse(tris [][3]int, keep []bool) map[[2]int]int {
	out := make(map[[2]int]int, 3*len(tris))
	for i, t := range tris {
		if !keep[i] {
			continue
		}
		for _, e := range directedTriangleEdges(t) {
			out[e]++
		}
	}
	return out
}
