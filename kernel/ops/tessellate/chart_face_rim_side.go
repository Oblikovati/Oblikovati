// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"
	stdmath "math"
)

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
// (u > 6.27, |v - 6| < 0.05) that lens's chart contour carries ONE point of its 256 and its shared
// edge carries 63. All 63 edge points there read MATERIAL against the chart, standing as far as
// 2.071e-3 in (u,v) from the nearest contour, while lying inside the lens the mesh is bounded by. The
// covering triangulates that band, the centroid test keeps it, and the face ships a skin inside its
// own hole: 112 of each lens chain's 380 segments came back bounded by TWO triangles instead of one.
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
//	base                          1777                                       637                     4 of 16
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
func (b *chartCover) directedRimSegments(segs []rimSegment, chains []chartChain, grid float64) map[[2]int]int {
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
	// conflict names the chain whose decisive segments did NOT agree, and is why the face is refused
	// rather than bound to a side a majority chose (#3518 review I4).
	conflict string
}

// bindToTheRim makes the kept set bounded by exactly the face's own boundary: for every boundary
// segment whose side the chart decided, the triangle on the material side is kept and the one on the
// other side is not.
func (b *chartCover) bindToTheRim(tris [][3]int, keep []bool) {
	side := b.materialSideOfEachChain(tris, keep)
	if side.conflict != "" {
		b.rimSideConflict = side.conflict
	}
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
// bounded exactly once — where its answer and the mesh's boundary agree — and requires them to AGREE.
//
// Unanimity, not a majority. A loop is wound consistently, so its decisive segments cannot honestly
// name two sides; if they do, the chart is wrong about this chain in a way no count can repair, and a
// majority would then bind the whole chain to a side confidently. The rim gate would not catch that:
// a chain bound to the WRONG side still bounds every one of its segments exactly once, and
// chartRimMismatch answers (0, 0) for that band. So a disagreement is refused by name instead.
//
// It costs nothing today: measured over the near-pinch corpus, every chain of every row is unanimous —
// right = 0 throughout, left 30 … 565 per chain, 1110 summed on the r = 3 face at PropertyQuality.
func (b *chartCover) materialSideOfEachChain(tris [][3]int, keep []bool) rimSide {
	fwd := keptDirectedEdgeUse(tris, keep)
	left, right := make([]int, b.chains), make([]int, b.chains)
	for e, ci := range b.rimChain {
		if b.segmentRunsAlongTheWindowEdge(e) {
			continue
		}
		switch back := [2]int{e[1], e[0]}; {
		case fwd[e] == 1 && fwd[back] == 0:
			left[ci]++
		case fwd[back] == 1 && fwd[e] == 0:
			right[ci]++
		}
	}
	return rimSideFrom(left, right)
}

// segmentRunsAlongTheWindowEdge reports whether a rim segment lies ON the branch window's own boundary,
// which makes it UNDECISIVE rather than a dissenting vote.
//
// The vote reads a segment's side from which of its two adjacent triangles the kept set holds. For a
// segment lying along the window's edge those two triangles are in DIFFERENT window images, and which
// of them survives is decided by the canonical replica selection (chart_face_replica.go) rather than by
// the chart. Counting it asks the wrong question and gets an arbitrary answer.
//
// ONE vote refused a face, and removing that vote means not counting 280 segments. Both numbers matter
// and the first sentence used to carry only the first. Measured on occtparity simple/J3's and
// bfuseblend/A4's host tori, DefaultQuality, through the real pipeline now that the rim rebuild carries
// their charts (#3550): each has 420 rim segments, and WITHOUT this rule 253 say left and ONE says
// right — a contradiction, which refuses the face. WITH it, 280 of the 420 are not counted and the
// remaining vote reads 126 left, 0 right.
//
// 280, not 1, because a tube-wrapping band's two rims ARE the branch window's two edges: the band runs
// u ∈ [uLo, uHi] and each rim circle sits on one of them. So this is not a rule that excuses one odd
// segment; it is the statement that a rim lying along the window edge carries no side information at
// all, and only the seam does.
//
// Unanimity stays the rule for what remains — a consistently wound loop cannot honestly name two
// sides, and a majority would bind a whole chain to a side confidently (#3518) — so the repair is to
// stop counting segments that were never decisive, not to start tolerating dissent.
// TestASegmentOnTheWindowEdgeCastsNoVote asserts both halves on a covering built for it.
func (b *chartCover) segmentRunsAlongTheWindowEdge(e [2]int) bool {
	tol := windowEdgeIncidence * stdmath.Max(b.r.uHi-b.r.uLo, b.r.vHi-b.r.vLo)
	return onSameWindowEdge(b.uu[e[0]], b.uu[e[1]], b.r.uLo, b.r.uHi, b.r.uPer, tol) ||
		onSameWindowEdge(b.vv[e[0]], b.vv[e[1]], b.r.vLo, b.r.vHi, b.r.vPer, tol)
}

// windowEdgeIncidence is how close, as a fraction of the chart's own (u,v) extent, a rim vertex has to
// be to a branch-window edge to count as ON it.
//
// It is a ROUNDING tolerance and nothing more. A vertex that sits on a window edge is put there by the
// branch shift — a whole period added to a traced parameter — so the two sides of the comparison are the
// same number arrived at by different arithmetic, and what separates them is last-place error in that
// addition, not geometry. Anything a rim vertex is doing NEAR but not ON the edge is a vertex the rule
// must not claim.
//
// The value is inherited from chartContourIncidence, which asked the same kind of question of the same
// spans until #3519 deleted it with the contour-incidence retry, and the reading does not depend on it:
// swept 1e-12, 1e-9, 1e-6 and 1e-3 on occtparity simple/J3's and bfuseblend/A4's host tori at
// DefaultQuality, all four read 280 segments excluded of 420 and a vote of 128 left, 0 right, and the
// whole ./kernel/ops/tessellate suite is green at all four. Four decades of plateau either side of the
// value, because the quantity it bounds is a few ulps of 2π.
const windowEdgeIncidence = 1e-6 // tol:parametric (relative to the chart's own extent; swept 1e-12…1e-3)

// onSameWindowEdge reports whether both of a segment's ends sit on the SAME end of one wrapping axis's
// branch window. A bounded axis has no branch and so no such edge.
func onSameWindowEdge(a, c, lo, hi float64, periodic bool, tol float64) bool {
	if !periodic {
		return false
	}
	atEnd := func(x, end float64) bool { return stdmath.Abs(x-end) <= tol }
	return (atEnd(a, lo) && atEnd(c, lo)) || (atEnd(a, hi) && atEnd(c, hi))
}

// rimSideFrom turns each chain's two counts into its side, whether it has one, and — when the two
// disagree — the refusal that names the chain and both counts.
func rimSideFrom(left, right []int) rimSide {
	s := rimSide{left: make([]bool, len(left)), decided: make([]bool, len(left))}
	for ci := range left {
		s.left[ci] = left[ci] > 0
		s.decided[ci] = (left[ci] > 0) != (right[ci] > 0)
		if left[ci] > 0 && right[ci] > 0 {
			s.conflict = fmt.Sprintf("its boundary chain %d names both sides as material (%d segments "+
				"say left, %d say right), so no side of it can be trusted", ci, left[ci], right[ci])
		}
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

// meshOrRefusal turns the covering's own state into either the triangles the face ships or the reason
// it is refused, in the order those reasons outrank each other. It exists so the refusal a chain's
// contradiction raises cannot be set and then dropped: the field and the decline are one call, and one
// test drives it (#3518 review N2).
//
// A CONTRADICTION outranks the other two. A chain that names both material sides makes every triangle
// touching it untrustworthy, so "an ear was still standing" and "nothing was kept" are symptoms of it
// rather than reasons of their own, and reporting a symptom would send the reader to the wrong place.
func (b *chartCover) meshOrRefusal(kept [][3]int, split bool) ([][3]int, string) {
	if b.rimSideConflict != "" {
		return nil, b.rimSideConflict
	}
	if !split {
		return nil, fmt.Sprintf("its %d rim-only-ear splitting rounds were spent with an ear still "+
			"standing, and an ear carries no surface point of its own", chartRimEarRounds)
	}
	if len(kept) == 0 {
		return nil, "the covering kept no triangle inside the chart's own window"
	}
	return kept, ""
}
