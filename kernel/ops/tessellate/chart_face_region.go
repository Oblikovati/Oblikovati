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
	slabs      []contourSlabs // parallel to contours; a zero entry is asked the unindexed way
	uLo, uHi   float64        // the branch window in u: one whole period when u wraps
	vLo, vHi   float64        // likewise in v
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
	r.slabs = make([]contourSlabs, len(contours))
	for i, c := range contours {
		r.slabs[i], _ = newContourSlabs(c) // a contour that cannot be indexed keeps the zero entry
	}
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
// retry and widening it by five decades: over that range the majority vote never once turned a "no"
// into a "yes" on any of these bodies. A constant with no plateau edge in either direction is not a
// tolerance that was tuned, so it is deleted rather than documented.
//
// "Inert" is as much as the corpus supports and MORE than it is true, which is the honest correction
// (review 2, I1): off the corpus the retry was not inert, it was harmful. Measured at the wave base
// 4b3b1819 with nothing changed but this band, two thin-torus bodies outside the corpus tore with the
// retry on and were watertight with it off — the R=20 r=1 and R=100 r=1 tangent-plane CUT pieces at
// PropertyQuality, 340 and 360 free edges against 0 and 0. There the vote answered INSIDE and admitted
// triangles that tore the wall. Both bodies read 0 at this commit, which is this branch's own reading
// of the same two rows. So the deletion is better than "it decided nothing": on the corpus it decided
// nothing, and where it did decide it decided wrongly.
//
// The retry's PRECONDITION is in fact unreachable, which is stronger than "inert on today's corpus" and
// is why no replacement is owed. It needed a decided NO at a centroid ON a contour edge. A centroid
// cannot sit on a contour edge that is a triangulation CONSTRAINT — the boundary chains go to the CDT as
// constraints and a centroid is strictly interior to its triangle — so only an artificial SEAM can carry
// one. And covers answers TRUE on a seam: the period translate supplies the crossing that the strict `<`
// in pointInUVPoly drops on the home branch. Measured on a chartRegion whose seam slants exactly as the
// deleted comment recorded, (0,0) → (−0.1963,10), queried at the point exactly on it at v = 5:
// covers = true, onContour = true, and all three half-way votes true as well. There is no decided NO to
// retry. (Nothing referenced this branch from a test at the wave base either — `git grep` over
// 4b3b1819 finds no `*_test.go` naming triangleIsMaterial, onContour, distToUVPoly or
// chartContourIncidence — so the issue's "three certifications" were prose, not rows.)
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
		for i := range r.contours {
			if r.contourContains(i, [2]float64{fu + sh[0], fv + sh[1]}) {
				in = !in
			}
		}
	}
	return in
}

// contourContains is pointInUVPoly for contour i, through its slab index when it has one — the same
// answer, bit for bit (chart_region_slabs.go says why), without walking every edge.
func (r chartRegion) contourContains(i int, p [2]float64) bool {
	if i < len(r.slabs) && r.slabs[i].edges != nil {
		return r.slabs[i].contains(r.contours[i], p)
	}
	return pointInUVPoly(r.contours[i], p)
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
//
// It hands out one of four tables built once, rather than building a slice per call, because covers calls
// it on EVERY membership query — once per covering grid node — and clearOfChains on every node again
// (#3527). The tables are read-only; every caller range-reads them and none has ever written to one.
func (r chartRegion) shifts() [][2]float64 {
	return chartShiftTables[wrapIndex(r.uPer)][wrapIndex(r.vPer)]
}

// chartShiftTables are the four offset sets a chartRegion can have, keyed [u wraps][v wraps]. There is
// no fifth: a region's only degrees of freedom here are which axes wrap.
var chartShiftTables = [2][2][][2]float64{
	{crossPeriodOffsets(false, false), crossPeriodOffsets(false, true)},
	{crossPeriodOffsets(true, false), crossPeriodOffsets(true, true)},
}

// wrapIndex is 1 when the axis wraps, which is how chartShiftTables is keyed.
func wrapIndex(periodic bool) int {
	if periodic {
		return 1
	}
	return 0
}

// crossPeriodOffsets is the cross product of the two axes' replication offsets, in the order the covering
// has always laid its replicas in — u outer, v inner — so the shift INDEX each vertex records, and the
// canonical choice that reads it, are unchanged.
func crossPeriodOffsets(uPer, vPer bool) [][2]float64 {
	var out [][2]float64
	for _, du := range periodOffsets(uPer) {
		for _, dv := range periodOffsets(vPer) {
			out = append(out, [2]float64{du, dv})
		}
	}
	return out
}

// periodOffsets is one axis's replication offsets, over the package's one set of cover shifts
// (coverShifts, periodic_nurbs_cover.go): a bounded axis has no replica, a wrapping one has the
// period either side.
//
// The 2π is not an assumption about the surface, it is what "periodic" MEANS to a chartRegion (#3527).
// newChartRegion builds a region only for an axis IsPeriodic answers true for, and that predicate is
// `lo ≈ 0 and hi ≈ 2π` — the kernel's normalised angular parametrisation and nothing else. branchWindow
// and wrapToPeriod read the same constant for the same reason. The periodic B-spline cover, whose axis
// can have any knot range, takes its period from the domain instead (coveringPeriodicMesh's
// `period := uhi - ulo`); a chart on such a surface does not reach here at all, because IsPeriodic
// refuses it. Widening IsPeriodic means giving chartRegion the period as data, in all three places.
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

// wrapToPeriod folds an angle onto [0, 2π). It lived beside the spiric band loft until that arm was
// absorbed into the chart mesher (#3517); the covering's own fold is its only remaining reader.
func wrapToPeriod(a float64) float64 {
	a = stdmath.Mod(a, 2*stdmath.Pi)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}
