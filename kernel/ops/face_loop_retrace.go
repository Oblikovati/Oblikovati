// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A face's developed boundary must not run BACK OVER ground it has already covered.
//
// THE GAP THIS CLOSES. SelfCrossingFaceLoops develops each face's boundary into its own metric chart
// and asks simpleLoop2D's predicate — segmentsCross — whether two non-adjacent edges TRANSVERSALLY
// intersect (both straddle tests strictly signed). A boundary that instead back-tracks along a
// COLLINEAR sibling, running back over the top of a segment it already traversed, scores exactly ZERO
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
// THE PREDICATE, AND WHY IT IS THIS AND NOT "ANY COLLINEAR PAIR". A legitimate loop routinely carries
// collinear segments: a straight edge subdivided by discretization, or two consecutive edges that
// happen to be parallel and meet end to end. Those cover DISJOINT ground and run the SAME way. The
// violation is the boundary covering the same ground TWICE IN OPPOSITE SENSES, which is what makes the
// covered stretch enclose zero area. So a retrace is, precisely:
//
//	two segments of one face loop that (1) develop collinear to within retraceCollinearTol of the
//	loop's own chart diagonal, (2) share an interval of that common line of length at least
//	retraceMinOverlap of the same diagonal, (3) traverse it in OPPOSITE directions, and (4) are
//	CORROBORATED IN 3D — the shared stretch resolves to the same point on both segments' own 3D chords.
//
// Condition (3) is what makes the false-positive direction structurally safe rather than
// tolerance-dependent: a subdivided straight edge fails it outright, no matter how the discretizer
// rounds, because its pieces run the same way. Condition (2) is what keeps a genuinely thin — but
// two-dimensional — sliver face out: two antiparallel edges of a sliver of width w are only
// "collinear" if w ≤ tol, and with tol three decades below the overlap floor the flagged stretch bounds
// an area of at most tol × overlap ≈ 1e-15 of the chart, i.e. it is degenerate, not merely thin. Both
// budgets are RELATIVE to the loop's own bounding diagonal, so the predicate is scale-invariant
// (ADR-0042).
//
// Condition (4) is what a chart-only predicate CANNOT do, and it is not optional: the (u,v) inversion
// is not always a faithful development. `unwrap` measures a periodic loop's span across the OPEN chain
// only, so a loop that starts exactly on the seam passes its "< 2π" guard by ε and then leaps the whole
// period on its CLOSING step; and a fillet corner sphere whose patch has the chart's own POLE as a
// vertex has no single-valued u there at all. Measured on the corpus, both produce a chart segment
// whose length is 100+ times the 3D chord it develops — and a chart-only predicate reads that leap as a
// long collinear back-track along v = 0. Asking the 3D boundary whether it really does pass through
// those points twice rejects every one of them, because a retrace is ultimately a claim ABOUT THE
// BOUNDARY, not about its chart.
//
// Like SelfCrossingFaceLoops this is a diagnostic query, NOT wired into the mesher: it is O(n²) on up
// to ~2500 boundary points and nothing on the pick or tessellation path calls it.

// The three budgets of the predicate above. retraceCollinearTol and retraceMinOverlap are taken
// against the developed loop's own bounding diagonal; retraceWeldFraction against the shared stretch.
//
// retraceCollinearTol is set three decades ABOVE double-precision noise on a chart coordinate
// (~1e-16 relative) so a retrace whose two strands were computed by different arithmetic paths is
// still recognised, and many decades BELOW any width a real face has, so a thin face is never mistaken
// for a degenerate one. retraceMinOverlap is three decades above it again: the flagged stretch must be
// a genuine shared interval, not the endpoint two collinear neighbours legitimately share.
//
// retraceWeldFraction is the same aspect-ratio idea applied in 3D, where the chart cannot be trusted:
// two strands separated by more than a twentieth of the length they are claimed to share are not the
// same ground. It is deliberately loose because a coarsely sampled curved boundary is a POLYLINE — the
// chart's linear parameter and the 3D chord's differ by the chord's own sagitta, ~3e-3 of the chord at
// the corpus's sampling density. It needs no tuning, because the two populations are THIRTEEN DECADES
// apart: swept over the corpus, all 7 real retraces corroborate to ≤ 9.4e-12 of the length they claim
// and all 4 chart artefacts to ≥ 113 — the artefact's two strands are a hundred times further apart
// than the stretch they supposedly share is long.
const (
	retraceCollinearTol = 1e-9
	retraceMinOverlap   = 1e-6
	retraceWeldFraction = 0.05
	retraceWeldStations = 3
)

// RetracingLoop names one face loop whose developed boundary back-tracks along itself.
type RetracingLoop struct {
	Face    *topo.Face
	Loop    int
	Overlap float64 // the developed LENGTH the boundary covers twice, in the surface's own metric chart
}

// RetracingFaceLoops returns every loop of b whose boundary, developed onto its own face's surface,
// runs back over ground it already covered — the collinear-backtracking half of "is this polygon
// simple", which the transversal predicate in SelfCrossingFaceLoops cannot see. Faces with no usable
// development, and loops that wrap a periodic seam, are skipped exactly as they are there; a candidate
// that the 3D boundary does not corroborate is discarded, so a report here is always a real defect.
//
// Overlap is a LENGTH, not an area: a retrace encloses zero area by construction, so there is no
// pinched-off area to quote and quoting one would be a category error.
//
// Example: RetracingFaceLoops(y1Body, PropertyQuality()) reports the host plane whose boundary runs out
// along z = 90 to x = 100 and straight back, Overlap = 10.
func RetracingFaceLoops(b *topo.Body, q Quality) []RetracingLoop {
	var out []RetracingLoop
	for _, f := range b.Faces() {
		loops, ok := developedFaceLoops(f, q)
		rings := faceLoopRings(f, q)
		if !ok || len(rings) != len(loops) {
			continue
		}
		for i, l := range loops {
			if ov, retraces := loopRetrace(l, rings[i]); retraces {
				out = append(out, RetracingLoop{Face: f, Loop: i, Overlap: ov})
			}
		}
	}
	return out
}

// faceLoopRings returns f's discretized boundary rings in the SAME order developedFaceLoops develops
// them — outer first, then the holes — so a developed loop can be checked against its own 3D points.
func faceLoopRings(f *topo.Face, q Quality) [][]math.Point3 {
	outer := faceOuterBoundary(f, q)
	if len(outer) == 0 {
		return nil
	}
	return append([][]math.Point3{outer}, faceHoleBoundaries(f, q)...)
}

// loopRetrace returns the LONGEST stretch the loop covers twice in opposite directions, and whether
// there is one. The longest (rather than the first) is reported because that is the quantity a ratchet
// ceiling must be monotone in: it cannot be reduced by re-indexing the loop.
func loopRetrace(l developedLoop, ring []math.Point3) (float64, bool) {
	n := len(l.pts)
	if n < 3 || n != len(ring) {
		return 0, false
	}
	diag := chartDiagonal(l.pts)
	if diag <= 0 {
		return 0, false
	}
	tol, floor := retraceCollinearTol*diag, retraceMinOverlap*diag
	worst := 0.0
	for i := range n {
		for j := i + 1; j < n; j++ {
			if ov := retraceOfPair(l.pts, ring, i, j, tol, floor); ov > worst {
				worst = ov
			}
		}
	}
	return worst, worst > 0
}

// retraceOfPair is the length segments i and j of the loop retrace, or 0 when they do not — the chart
// test first, then the 3D corroboration that rejects an unfaithful development.
func retraceOfPair(pts []math.Point2, ring []math.Point3, i, j int, tol, floor float64) float64 {
	shared, ok := collinearBacktrack(chartSegAt(pts, i), chartSegAt(pts, j), tol, floor)
	if !ok || !corroboratedIn3D(ring, i, j, shared) {
		return 0
	}
	return shared.length
}

// chartOverlap is a stretch two chart segments share, expressed as a parameter sub-interval of EACH of
// them, so the claim can be re-asked of the 3D polyline they develop.
type chartOverlap struct {
	length             float64
	sLo, sHi, tLo, tHi float64 // the same two chart locations, as fractions along each segment
}

// collinearBacktrack reports the stretch over which s and t cover the SAME chart ground in OPPOSITE
// directions. It declines when they are not collinear, share less than floor, or run the same way
// along it (which is a subdivided straight edge, not a back-track). The LONGER segment is the
// reference frame so a short one never sets the direction over a long lever arm.
func collinearBacktrack(s, t chartSeg, tol, floor float64) (chartOverlap, bool) {
	ref, other, swapped := s, t, false
	if s.length() < t.length() {
		ref, other, swapped = t, s, true
	}
	ux, uy, l := ref.unit()
	if l < floor {
		return chartOverlap{}, false // even total agreement could not reach the floor
	}
	ta, offA := ref.frameOf(other.a, ux, uy)
	tb, offB := ref.frameOf(other.b, ux, uy)
	if stdmath.Abs(offA) > tol || stdmath.Abs(offB) > tol || tb >= ta {
		return chartOverlap{}, false
	}
	lo, hi := stdmath.Max(0, tb), stdmath.Min(l, ta)
	if hi-lo < floor {
		return chartOverlap{}, false
	}
	return sharedFractions(lo, hi, l, ta, tb, swapped), true
}

// sharedFractions expresses the shared chart interval [lo,hi] — measured in the reference segment's
// own frame, where the other segment runs from ta down to tb — as a parameter sub-interval of each
// segment, restoring the caller's original ordering.
func sharedFractions(lo, hi, l, ta, tb float64, swapped bool) chartOverlap {
	o := chartOverlap{
		length: hi - lo,
		sLo:    lo / l, sHi: hi / l,
		tLo: (lo - ta) / (tb - ta), tHi: (hi - ta) / (tb - ta),
	}
	if swapped {
		o.sLo, o.sHi, o.tLo, o.tHi = o.tLo, o.tHi, o.sLo, o.sHi
	}
	return o
}

// corroboratedIn3D asks the BOUNDARY, not its chart, whether it really passes through the shared
// stretch twice: each of retraceWeldStations interior stations must resolve to the same point on
// segment i's own 3D chord as on segment j's. A chart that is not a faithful development — a periodic
// loop leaping the seam on its closing step, a sphere patch holding the chart's pole — puts the two
// resolutions far apart and is rejected here instead of being reported as a defect.
func corroboratedIn3D(ring []math.Point3, i, j int, o chartOverlap) bool {
	n := len(ring)
	weld := retraceWeldFraction * o.length
	for k := 1; k <= retraceWeldStations; k++ {
		f := float64(k) / float64(retraceWeldStations+1)
		p := ring[i].Lerp(ring[(i+1)%n], o.sLo+f*(o.sHi-o.sLo))
		q := ring[j].Lerp(ring[(j+1)%n], o.tLo+f*(o.tHi-o.tLo))
		if p.DistanceTo(q) > weld {
			return false
		}
	}
	return true
}

// chartSeg is one directed segment of a developed loop, in the surface's metric chart.
type chartSeg struct{ a, b [2]float64 }

// chartSegAt is the segment leaving vertex i of a closed developed loop.
func chartSegAt(pts []math.Point2, i int) chartSeg {
	return chartSeg{a: xy(pts[i]), b: xy(pts[(i+1)%len(pts)])}
}

// length is the segment's extent in the metric chart.
func (s chartSeg) length() float64 { return stdmath.Hypot(s.b[0]-s.a[0], s.b[1]-s.a[1]) }

// unit returns the segment's direction and length; a zero-length segment gets a zero direction.
func (s chartSeg) unit() (ux, uy, l float64) {
	if l = s.length(); l == 0 {
		return 0, 0, 0
	}
	return (s.b[0] - s.a[0]) / l, (s.b[1] - s.a[1]) / l, l
}

// frameOf returns p's coordinate along the segment's own direction (ux,uy) and its signed
// perpendicular offset from the segment's supporting line.
func (s chartSeg) frameOf(p [2]float64, ux, uy float64) (along, off float64) {
	dx, dy := p[0]-s.a[0], p[1]-s.a[1]
	return dx*ux + dy*uy, dy*ux - dx*uy
}

// chartDiagonal is the bounding-box diagonal of a developed loop — the scale the collinearity and
// overlap budgets are taken relative to.
func chartDiagonal(pts []math.Point2) float64 {
	lo, hi := xy(pts[0]), xy(pts[0])
	for _, p := range pts[1:] {
		lo = [2]float64{stdmath.Min(lo[0], p.X), stdmath.Min(lo[1], p.Y)}
		hi = [2]float64{stdmath.Max(hi[0], p.X), stdmath.Max(hi[1], p.Y)}
	}
	return stdmath.Hypot(hi[0]-lo[0], hi[1]-lo[1])
}
