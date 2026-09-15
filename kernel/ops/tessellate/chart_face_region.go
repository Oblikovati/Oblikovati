// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The REGION half of the chart-driven curved-face mesher (ADR-0061; the chart itself is ADR-0063).
//
// A trimmed face on a periodic surface is not determined by its 3D loops: a torus band's two rim
// curves bound the strip between them AND the strip the other way round the seam. The producer that
// wound the face recorded which, as closed (u,v) contours in the surface's COVERING space, and this is
// the reader of that record: the contours, the one-period window they were recorded on, and the
// even-odd membership test the mesher classifies its triangles with.
//
// Even-odd over ALL contours together, not outer-minus-holes, because a region that wraps a whole
// period arrives SPLIT at the arrangement's own seam: a ring bored by a coaxial shaft records its torus
// face as TWO disjoint positive rectangles (v ∈ [3.98, 2π] and v ∈ [0, 2.30]), which are one band
// through the v seam. Reading the first as an outer and the second as its hole inverts it.

// chartRegion is a face's carried parametric trim placed on its own branch of the covering space.
type chartRegion struct {
	contours   [][]math.Point2
	uLo, uHi   float64 // the branch window in u: one whole period when u wraps
	vLo, vHi   float64 // likewise in v
	uPer, vPer bool
}

// newChartRegion reads the face's chart onto the branch it was recorded on. ok=false for a face that
// carries no chart (nothing to read) or whose surface wraps in neither direction (ToUVLoops already
// charts an aperiodic trim, and there is no branch to choose).
//
// Example: if r, ok := newChartRegion(f, f.Geometry()); ok { inside = r.covers(u, v) }
func newChartRegion(f *topo.Face, s geom.Surface) (chartRegion, bool) {
	contours := f.Chart()
	uPer, vPer := IsPeriodic(s.UDomain()), IsPeriodic(s.VDomain())
	if len(contours) == 0 || (!uPer && !vPer) {
		return chartRegion{}, false
	}
	u0, u1, v0, v1, ok := chartBounds(contours)
	if !ok {
		return chartRegion{}, false
	}
	r := chartRegion{contours: contours, uPer: uPer, vPer: vPer}
	r.uLo, r.uHi = branchWindow(u0, u1, uPer)
	r.vLo, r.vHi = branchWindow(v0, v1, vPer)
	return r, true
}

// branchWindow is the covering-space window one axis's contours live on: a whole period from their
// lower bound where the axis wraps (the region may run all the way round it, and the chart is recorded
// on exactly one period), their own extent where it does not (a sphere's latitude, a wall's height).
func branchWindow(lo, hi float64, periodic bool) (float64, float64) {
	if periodic {
		return lo, lo + 2*stdmath.Pi
	}
	return lo, hi
}

// covers reports whether (u,v) is material: the query is carried onto the chart's own branch on each
// wrapping axis, then counted even-odd against every contour (see the file doc for why every). It is
// the ONE membership answer the mesher gets; there is no second opinion behind it.
//
// There used to be. A centroid landing ON a contour edge was counted as UNDECIDED — within
// chartContourIncidence (1e-6 of the chart's own extent) of an edge — and re-asked as a majority vote
// over three points halfway to the triangle's vertices. It was built for the merged cocylindrical
// wall's SLANTED artificial seam, whose grid-built centroids fell on it to 1e-11 and vanished from both
// branches (forty holes, 615 unpaired edges against a rim of 578).
//
// Both of that case's conditions are gone, and the retry with them (#3519). The wall's chart no longer
// slants — its seam runs (0,0) → (0,10) since the band re-cut was fixed to shift by whole turns
// (ADR-0061 stage 5 round 3) — and the face no longer reaches this mesher at all: it classifies as
// ruled-band-loft. Swept over the chart corpus (the 21 classification-corpus bodies, the merged
// cocylindrical wall and six torus/tangent-plane figure-eight pieces) at BOTH facetings, with the band
// set to 0, 1e-8, 1e-6, 1e-4, 1e-2 and 1e-1 of the extent:
//
//	band     centroids on a contour   triangles the retry rescued   corpus rows that move
//	0                             3                             0   none (the reference)
//	1e-8                         13                             0   none
//	1e-6 (shipped)               19                             0   none
//	1e-4                        751                             0   none
//	1e-2                       8749                             0   none
//	1e-1                      25508                             0   none
//
// Not one number in the corpus — free edges, body volume, per-face area — differs between DISABLING the
// retry and widening it by five decades, because the majority vote answered "outside" every single time
// it was asked. A constant with no plateau edge in either direction is not a tolerance that was tuned;
// it is a branch that decides nothing, so it is deleted rather than documented.
//
// The hazard it named is real and is now this predicate's to own: an even-odd count exactly on a
// contour answers by which side the ray was cast from. The fix for that, when a shape needs it, is an
// exact or consistently-signed predicate here (kernel ground rules: "every topological decision uses an
// exact or filtered predicate; epsilon compares metric quantities only") — not an epsilon band and a
// vote, which is what the deleted retry was.
func (r chartRegion) covers(u, v float64) bool {
	fu, fv := r.fold(u, v)
	in := false
	for _, sh := range r.shifts() {
		for _, c := range r.contours {
			if pointInUVPoly(c, [2]float64{fu + sh[0], fv + sh[1]}) {
				in = !in
			}
		}
	}
	return in
}

// fold carries a query onto the chart's branch on each wrapping axis. A bounded axis is left alone: a
// query past its ends is genuinely off the surface, not one period along.
func (r chartRegion) fold(u, v float64) (float64, float64) {
	if r.uPer {
		u = r.uLo + wrapToPeriod(u-r.uLo)
	}
	if r.vPer {
		v = r.vLo + wrapToPeriod(v-r.vLo)
	}
	return u, v
}

// windowCandidate reports whether an UNFOLDED (u,v) lies in the branch window, CLOSED at both ends: the
// filter that says which of a periodic triangle's replicas is worth classifying at all. It is
// deliberately not the de-duplication — see keepOneReplicaEach (chart_face_replica.go), which is.
func (r chartRegion) windowCandidate(u, v float64) bool {
	if r.uPer && (u < r.uLo || u > r.uHi) {
		return false
	}
	return !r.vPer || (v >= r.vLo && v <= r.vHi)
}

// shifts are the whole-period offsets the covering replicates its points over: none on a bounded axis,
// one period either side on a wrapping one, so a triangle spanning a seam finds its neighbours there.
func (r chartRegion) shifts() [][2]float64 {
	var out [][2]float64
	for _, du := range periodOffsets(r.uPer) {
		for _, dv := range periodOffsets(r.vPer) {
			out = append(out, [2]float64{du, dv})
		}
	}
	return out
}

// periodOffsets is one axis's replication offsets, over the package's one set of cover shifts
// (coverShifts, periodic_nurbs_cover.go): a bounded axis has no replica, a wrapping one has the
// period either side.
func periodOffsets(periodic bool) []float64 {
	if !periodic {
		return []float64{0}
	}
	out := make([]float64, len(coverShifts))
	for i, sh := range coverShifts {
		out[i] = sh * 2 * stdmath.Pi
	}
	return out
}

// branchOffset is the whole-period shift that brings a lifted boundary trace onto the chart's branch.
// A trace is continuous by construction, so ONE shift per axis serves the whole chain; it is chosen by
// the trace's mean, which for a chain that wraps a period sits in the middle of it.
func (r chartRegion) branchOffset(cu, cv []float64) (float64, float64) {
	du, dv := 0.0, 0.0
	if r.uPer {
		du = branchShift(meanOfParams(cu), r.uLo, r.uHi)
	}
	if r.vPer {
		dv = branchShift(meanOfParams(cv), r.vLo, r.vHi)
	}
	return du, dv
}

// meanOfParams is the arithmetic mean of a parameter trace (0 for an empty one).
func meanOfParams(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// chartBounds is the (u,v) bounding box of every contour together, over the package's one loop-bbox
// (uvBBox). ok=false for a chart that bounds no area in one of the axes, which no mesher can take a
// region from.
func chartBounds(contours [][]math.Point2) (u0, u1, v0, v1 float64, ok bool) {
	u0, v0 = stdmath.Inf(1), stdmath.Inf(1)
	u1, v1 = stdmath.Inf(-1), stdmath.Inf(-1)
	for _, c := range contours {
		cu0, cu1, cv0, cv1 := uvBBox(c)
		u0, u1 = stdmath.Min(u0, cu0), stdmath.Max(u1, cu1)
		v0, v1 = stdmath.Min(v0, cv0), stdmath.Max(v1, cv1)
	}
	return u0, u1, v0, v1, u1 > u0 && v1 > v0
}
