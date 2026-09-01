// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A face's boundary must not run BACK OVER ground it has already covered.
//
// THE GAP THIS CLOSES. SelfCrossingFaceLoops develops each face's boundary into its own metric chart
// and asks simpleLoop2D's predicate — segmentsCross — whether two non-adjacent edges TRANSVERSALLY
// intersect (both straddle tests strictly signed). A boundary that instead back-tracks along a
// COLLINEAR sibling, running back over the top of a stretch it already traversed, scores exactly ZERO
// under that predicate: two overlapping collinear segments never straddle each other's line, so every
// orient2d in segmentsCross is 0 and `d1*d2 < 0` is false. The loop is just as malformed — it is not a
// simple polygon and it has no well-defined interior — but nothing in the kernel saw it. Worse, the
// defect is invisible to an AREA gate as well: the two traversals of the retraced stretch contribute
// equal and opposite terms to the shoelace, so a loop can carry one and still measure exactly right.
//
// It is not hypothetical. simple/Y4's largest single face error was +100 on its host plane, whose
// shipped loop was
//
//	(100,0,80) (90,0,80) (90,0,90) (100,0,90) (100,0,75) (0,0,75) (0,0,0) (100,0,0)
//
// where (100,0,90)→(100,0,75) runs back along the closing (100,0,0)→(100,0,80) for z ∈ [75,80]. The
// self-crossing ratchet's Y4 entry (23.9109) was a DIFFERENT face; this one was invisible.
//
// ★ IT IS ASKED OF THE B-REP CURVES, NOT OF A DEVELOPMENT OF THEM (M48/C3, #3475). The predicate used
// to run in the surface's metric chart on a boundary DISCRETIZED to the caller's Quality, and it needed
// two guards to survive that: a collinearity budget, because a chart is not an isometry, and a 3D
// corroboration step, because a chart is not always even a faithful ORDERING (a loop that starts on a
// periodic seam leaps the whole period on its closing step; a sphere patch holding the chart's pole has
// no single-valued u there). Both guards are gone, with the chart and the discretization: a retrace is
// now measured where it lives, on the EDGES' OWN CURVES.
//
// THE PREDICATE, AND WHY IT IS THIS AND NOT "ANY COINCIDENT PAIR". A legitimate loop routinely carries
// coincident-looking neighbours: a straight edge subdivided into collinear pieces, or two consecutive
// edges that are parallel and meet end to end. Those cover DISJOINT ground and run the SAME way. The
// violation is the boundary covering the same ground TWICE IN OPPOSITE SENSES, which is what makes the
// covered stretch enclose zero area. So a retrace is, precisely:
//
//	two edge uses of one face loop whose curves (1) coincide — to within the face's own
//	geom.Resolution weld — over a stretch of positive arc length, and (2) traverse that stretch in
//	OPPOSITE directions in the loop's own traversal order.
//
// Condition (2) is what makes the false-positive direction structurally safe rather than
// tolerance-dependent: a subdivided straight edge fails it outright, no matter how its pieces were
// built, because they run the same way. Condition (1) keeps one floor, and only one — every pair of
// edges meeting at a shared vertex coincides over a ball of the weld radius about that vertex, so a
// stretch no longer than the model's own coincidence neighbourhood is that vertex and not a stretch
// (lengthAboveVertexNeighbourhood, with the corpus measurement that separates the two populations). A
// thin but two-dimensional sliver never reaches even that: its two antiparallel edges are a real
// distance apart, and that distance is measured on the exact curves (brep.EntityDistance), not on
// chords of them.
//
// ★ ONE NAMED EXCLUSION: THE PERIODIC SEAM. A closed face — a whole cylinder, a sphere — bounds itself
// with ONE seam edge used TWICE by its own loop, forward and back. Those two uses do cover the same
// ground in opposite senses, and they are the one configuration where that is correct rather than a
// defect, so a pair of uses of the SAME edge is excluded by identity (not by tolerance). It is also
// what the old chart predicate was really buying with its unwrap/seam guard, at the cost of skipping
// every seamed face entirely; here only the seam's own pair is skipped, and the rest of the loop is
// still checked.
//
// Like SelfCrossingFaceLoops this is a diagnostic query, NOT wired into the mesher: it is O(n²) in a
// loop's EDGE count (not in its sample count, as it was) and nothing on the pick or tessellation path
// calls it.

// retraceInteriorStations is how many interior points of a candidate stretch are re-checked before it
// is reported. It is a COUNT, not a tolerance: the stretch's endpoints are exact, and this only
// confirms that the two curves really do coincide THROUGHOUT it rather than at the sub-interval
// midpoint that proposed it — which two splines that osculate without coinciding could fake.
const retraceInteriorStations = 5

// RetracingLoop names one face loop whose boundary back-tracks along itself.
type RetracingLoop struct {
	Face    *topo.Face
	Loop    int     // index into Face.Loops()
	Overlap float64 // the ARC LENGTH the boundary covers twice, on the edge curves themselves
}

// RetracingFaceLoops returns every loop of b whose boundary runs back over ground it already covered —
// the collinear-backtracking half of "is this polygon simple", which the transversal predicate in
// SelfCrossingFaceLoops cannot see. The verdict is taken on the edges' exact curves, so it carries no
// tessellation and no facet density; the Quality argument is unused and kept only so the call sites
// that thread one through do not churn (the same reason [FillInternalVoids] ignores its own).
//
// Overlap is a LENGTH, not an area: a retrace encloses zero area by construction, so there is no
// pinched-off area to quote and quoting one would be a category error.
//
// Example: RetracingFaceLoops(y1Body, PropertyQuality()) reports the host plane whose boundary runs out
// along z = 90 to x = 100 and straight back, Overlap = 10.
func RetracingFaceLoops(b *topo.Body, _ Quality) []RetracingLoop {
	var out []RetracingLoop
	for _, f := range b.Faces() {
		res := geom.ResolutionForBox(f.RangeBox())
		for i, l := range f.Loops() {
			if ov, retraces := loopRetrace(l, res); retraces {
				out = append(out, RetracingLoop{Face: f, Loop: i, Overlap: ov})
			}
		}
	}
	return out
}

// loopRetrace returns the LONGEST stretch the loop covers twice in opposite directions, and whether
// there is one. The longest (rather than the first) is reported because that is the quantity a ratchet
// ceiling must be monotone in: it cannot be reduced by re-indexing the loop.
func loopRetrace(l *topo.Loop, res geom.Resolution) (float64, bool) {
	uses := l.EdgeUses()
	worst := 0.0
	for i, ua := range uses {
		for _, ub := range uses[i+1:] {
			if ov := useRetraceLength(ua, ub, res); ov > worst {
				worst = ov
			}
		}
	}
	return worst, worst > 0
}

// useRetraceLength is the arc length over which two edge uses of one loop cover the same ground in
// opposite traversal senses, or 0 when they do not.
func useRetraceLength(ua, ub *topo.EdgeUse, res geom.Resolution) float64 {
	ea, eb := ua.Edge(), ub.Edge()
	if ea == eb {
		return 0 // the PERIODIC SEAM: one edge used twice by its own face, forward and back
	}
	if !ea.RangeBox().Intersects(eb.RangeBox()) {
		return 0
	}
	ca, cb := ea.Geometry(), eb.Geometry()
	if brep.EntityDistance(brep.CurveSupport(ca), brep.CurveSupport(cb)) > res.Weld() {
		return 0
	}
	lo, hi, ok := coincidentSpanOn(ca, cb, res.Weld())
	if !ok || !oppositeTraversal(ua, ub, ca, cb, lo, hi) {
		return 0
	}
	return lengthAboveVertexNeighbourhood(ca, lo, hi, res)
}

// lengthAboveVertexNeighbourhood is the stretch's arc length, or 0 when the stretch is no longer than
// the model's own coincidence neighbourhood.
//
// EVERY pair of edges meeting at a shared vertex coincides over a ball of the weld radius about it, so
// without this floor the detector reports the vertex itself. Measured on the OCCT blend-parity corpus
// the two populations are NINE DECADES apart: the vertex-neighbourhood reports run 1.4e-16 to 1.9e-10,
// and the smallest real retrace is simple/W2's 0.2 (its fillet radius). res.Stitch() — 1e-6 of the
// model size, the floor the chart predicate carried as retraceMinOverlap — sits in the middle of that
// gap and is model-relative, so a µm or km copy of the same part classifies the same way (ADR-0042).
func lengthAboveVertexNeighbourhood(ca geom.Curve3, lo, hi float64, res geom.Resolution) float64 {
	span := geom.CurveLength3(ca, lo, hi)
	if span <= res.Stitch() {
		return 0
	}
	return span
}

// coincidentSpanOn returns the longest parameter interval of ca every point of which lies on cb,
// within cb's own trim. The interval's ENDS are exact rather than sampled: two curve pieces stop
// covering the same ground exactly where one of the two trims stops, so every candidate end is either
// an end of ca or an end of cb projected onto ca (spanCandidates).
func coincidentSpanOn(ca, cb geom.Curve3, weld float64) (float64, float64, bool) {
	cuts := spanCandidates(ca, cb)
	covered := make([]bool, max(len(cuts)-1, 0))
	for k := range covered {
		covered[k] = pointOfCurveLiesOn(ca, cb, (cuts[k]+cuts[k+1])/2, weld)
	}
	lo, hi, ok := longestCoveredRun(ca, cuts, covered)
	if !ok || !spanHoldsThroughout(ca, cb, lo, hi, weld) {
		return 0, 0, false
	}
	return lo, hi, true
}

// spanCandidates are the parameters of ca at which a coincident stretch can begin or end — ca's own
// ends plus cb's two ends projected onto ca — in increasing order.
func spanCandidates(ca, cb geom.Curve3) []float64 {
	lo, hi := ca.Domain()
	cuts := []float64{lo, hi}
	blo, bhi := cb.Domain()
	for _, t := range [2]float64{blo, bhi} {
		if s, _ := geom.CurveParamAtPoint3(ca, cb.PointAt(t)); s > lo && s < hi {
			cuts = append(cuts, s)
		}
	}
	sort.Float64s(cuts)
	return cuts
}

// longestCoveredRun returns the parameter interval spanned by the longest run of consecutive covered
// sub-intervals, measured as arc length on ca so the winner is a real length and not a parameter span.
func longestCoveredRun(ca geom.Curve3, cuts []float64, covered []bool) (float64, float64, bool) {
	lo, hi, best, runStart := 0.0, 0.0, 0.0, -1
	for k := 0; k <= len(covered); k++ {
		if k < len(covered) && covered[k] {
			runStart = startOfRun(runStart, k)
			continue
		}
		if runStart >= 0 {
			best, lo, hi = keepLonger(ca, cuts[runStart], cuts[k], best, lo, hi)
		}
		runStart = -1
	}
	return lo, hi, best > 0
}

// startOfRun keeps the FIRST index of a covered run: once set it survives the rest of the run.
func startOfRun(runStart, k int) int {
	if runStart < 0 {
		return k
	}
	return runStart
}

// keepLonger returns the longer of the running best interval and the candidate [from, to].
func keepLonger(ca geom.Curve3, from, to, best, lo, hi float64) (float64, float64, float64) {
	if l := geom.CurveLength3(ca, from, to); l > best {
		return l, from, to
	}
	return best, lo, hi
}

// spanHoldsThroughout re-checks interior stations of a candidate stretch, so a pair of curves that
// touch at the one midpoint that proposed the stretch (two splines osculating without coinciding) is
// declined rather than reported.
func spanHoldsThroughout(ca, cb geom.Curve3, lo, hi, weld float64) bool {
	for k := 1; k <= retraceInteriorStations; k++ {
		t := lo + (hi-lo)*float64(k)/float64(retraceInteriorStations+1)
		if !pointOfCurveLiesOn(ca, cb, t, weld) {
			return false
		}
	}
	return true
}

// pointOfCurveLiesOn reports whether ca's point at parameter t lies on cb WITHIN cb's own trim — the
// exact "is this piece of boundary also that piece of boundary" question, answered by inverting the
// point onto cb and measuring the residual there.
func pointOfCurveLiesOn(ca, cb geom.Curve3, t, weld float64) bool {
	p := ca.PointAt(t)
	s, _ := geom.CurveParamAtPoint3(cb, p)
	lo, hi := cb.Domain()
	if s < lo || s > hi {
		return false
	}
	return float64(cb.PointAt(s).DistanceTo(p)) <= weld
}

// oppositeTraversal reports whether the two uses run the shared stretch in OPPOSITE directions. That
// is the whole difference between a back-track and a straight edge merely subdivided into collinear
// pieces, and it is a SIGN, so no tolerance enters it.
func oppositeTraversal(ua, ub *topo.EdgeUse, ca, cb geom.Curve3, lo, hi float64) bool {
	first, second := lo, hi
	if ua.Reversed() {
		first, second = hi, lo
	}
	return traversalParam(ub, cb, ca.PointAt(second)) < traversalParam(ub, cb, ca.PointAt(first))
}

// traversalParam is p's parameter on the use's edge curve, negated for a reversed use so it always
// increases along the LOOP's traversal of that use.
func traversalParam(u *topo.EdgeUse, c geom.Curve3, p math.Point3) float64 {
	s, _ := geom.CurveParamAtPoint3(c, p)
	if u.Reversed() {
		return -s
	}
	return s
}
