// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The non-sphere recognizers of the curved-trim classification (ADR-0061 stage 5, #3409). Each returns
// the payload its arm's builder needs alongside the verdict, so the shape is recognised once and built
// from that same reading — never recognised again inside the mesher. The sphere family's recognizers
// live in sphere_trim_form.go, which has three rim forms to keep disjoint.

// coneApexTrim is the cone-apex arm's recognition: the cone, the rim the fan sweeps and whether that
// rim CLOSES (a conic cap's rim wraps; a sector's base arc does not).
type coneApexTrim struct {
	cone   geom.Cone
	rim    []math.Point3
	closed bool
}

// coneApexTrimOf recognises a cone face that closes to its apex. ok=false for every cone face that is
// not an apex topology — a frustum band, a saddle-bounded stub, a holed face whose inner rim is a
// hole, or a seamed apex face whose loop already spans the apex.
//
// Example: a drill point's single circular rim → {rim: the circle, closed: true}.
func coneApexTrimOf(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (coneApexTrim, bool) {
	cone, isCone := s.(geom.Cone)
	if !isCone || len(holes3D) != 0 {
		return coneApexTrim{}, false
	}
	if faceIsConeApexCap(f) {
		return coneApexTrim{cone: cone, rim: outer3D, closed: true}, len(outer3D) >= 3
	}
	if len(f.Loops()) != 1 {
		return coneApexTrim{}, false
	}
	rim := rimExcludingApex(outer3D, cone.Apex, geom.ResolutionForPoints(outer3D).Weld())
	// No apex vertex on the loop (a frustum/stub), or too few rim points for a fan.
	return coneApexTrim{cone: cone, rim: rim}, len(rim) != len(outer3D) && len(rim) >= 2
}

// ruledTwoRimBandHolds reports whether the trim is a developable side bounded by two full-wrap rims —
// the shape the ruled loft stitches rim-to-rim with no interior row.
//
// The lens guard belongs to the NOTCHED form alone, and deliberately so: the saddle loft pools all
// open edges into one rim, so a notched band carrying a genuine lens would fold that lens into its
// base rim (#1591) and is kindTwoRimHoledBand's instead. The two-closed-rim form has no open edge to
// pool into, has never carried the guard, and must not acquire one here — a developable side whose two
// closed edges include a lens (a drilled cone apex cap) has always been lofted and still is.
func ruledTwoRimBandHolds(f *topo.Face, s geom.Surface, q Quality) bool {
	if !isDevelopableSide(s) {
		return false
	}
	return hasTwoClosedRimsNoOpen(f) || (!faceHasLensHole(f, s, q) && hasFullCircleAndNotchedRim(f))
}

// spiricTubeTrim is the spiric arm's recognition: the torus and the two edges that each wrap its tube.
type spiricTubeTrim struct {
	torus         geom.Torus
	first, second *topo.Edge
}

// spiricTubeTrimOf recognises a torus band bounded by two tube-wrapping edges (#1375) — a torus cut
// through its hole — that records NO chart. A band that records one is the general chart-driven
// mesher's, like every other charted face.
//
// The loft is not an exact fast path: it sweeps ONE direction round the tube for the whole band, which
// describes the region only while the strip between its boundaries has a width everywhere. Measured on
// all ten spiric faces the kernel corpus builds, every one of them charted, the chart mesher accepts
// each and reads the same area to within 0.004–0.17 % on the eight that do not pinch, with a THIRD
// fewer triangles at PropertyQuality (120184 against 171008 on the widest). On the two that DO pinch —
// the figure-eight, a torus R=5 r=2 cut by y=3 tangent to its inner equator — the loft was not merely
// coarser but wrong: 310.800 mm² and 215.177 where the analytic regions are 283.100 and 111.684, its
// two halves summing to 525.98 against a torus of 394.78, because it covered the tangency twice. The
// chart mesher reads 283.075 and 111.675.
//
// What keeps the loft is the UNCHARTED band, which the chart mesher cannot serve at all: occtparity's
// J3 and A4 host tori record no chart, and the arm meshes them in 340988 and 406540 triangles.
//
// #3517 MEASURED WHAT DELETING IT WOULD COST, and left it standing. Three facts, in the order a reader
// needs them.
//
// "chart=0" is not a property of these two faces; it is a property of every face of both bodies — all
// four of J3's and all nine of A4's. They are a STEP import plus a fillet, and no importer and no rim
// rebuild writes a chart (ADR-0063 puts it on the producer that WOUND the face; brep's arrangement is
// the only producer there is). That gap is #3550 and it is far wider than these two hosts.
//
// Supplying a chart by hand does not rescue them. A4 then meshes; J3 is refused by the boundary-side
// classification because its loop walks the torus's artificial v-seam twice and continuousTrace snaps
// the second traversal onto the first's branch. And at PropertyQuality — the faceting the per-face
// oracle reads — the covering does not finish at all: 788250 points and 5660 constraint loops into the
// constrained triangulation, which did not return in 880 s. Two defects sit under that, #3548 (the
// constraint recovery's budgeted loop is O(n·T)) and #3549 (the chart cover's facet-count policy emits
// ~1e6 samples for one periodic torus face), and they hide each other.
//
// Delete the arm and the two faces do NOT reach the general pipeline: they fall to the surface's whole
// domain at 2097152 triangles and 394781.31 mm² against the band's 292951, with the degradation
// reported. Until #3548/#3549/#3550, a lower recognizer count would buy a wrong body. ADR-0061's
// "G13 stays open" section carries the measurement; tube_wrapping_band_test.go plants it.
func spiricTubeTrimOf(f *topo.Face, s geom.Surface, q Quality) (spiricTubeTrim, bool) {
	if len(f.Chart()) > 0 {
		return spiricTubeTrim{}, false // the general chart-driven mesher serves this band
	}
	t, first, second, ok := tubeWrappingEdges(f, s, q)
	if !ok {
		return spiricTubeTrim{}, false
	}
	return spiricTubeTrim{torus: t, first: first, second: second}, true
}

// twoRimHoledTrim is the unrolled-wall arm's recognition: the face's one full-wrap hole rim and the
// lens windows it carries.
type twoRimHoledTrim struct {
	rim    []math.Point3
	lenses [][]math.Point3
}

// twoRimHoledTrimOf recognises a singly-periodic developable side whose hole loops are ONE full-wrap
// rim plus at least one lens window — and which the GENERAL chart-driven mesher cannot serve. The lens
// is what separates the shape from kindRuledBandLoft, whose pure rim-to-rim loft is exact precisely
// because it carries none.
//
// The unroll is not an exact fast path: it bridges the two rims at an invented seam and triangulates the
// flattened branch, and on the rod a ball is set into it meshes the right 24.5 mm² of wall with
// triangles whose planes pass 0.5 from the axis, so the wall integrates 7.19 where 8.26 is right. Per
// FACE the chart mesher is better on every band whose windows its own sampling can separate — over the
// corpus it matches the unroll to ±0.2 % of area with 40–85 % fewer triangles and the same rim count,
// and routing those to it moves RODB∪/RODB− from 8.72 %/9.02 % to 1.44 %/1.43 %.
//
// So the arm keeps exactly two configurations, both of them CONDITIONING on the general path, not shape:
// a face that records no chart (there is no region to mesh from), and a band whose windows nearly pinch
// (below). Everything else is kindChart.
func twoRimHoledTrimOf(chart [][]math.Point2, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (twoRimHoledTrim, bool) {
	if !isDevelopableSide(s) || IsPeriodic(s.UDomain()) == IsPeriodic(s.VDomain()) {
		return twoRimHoledTrim{}, false
	}
	rims, lenses := splitWrappingHoles(s, holes3D)
	if len(rims) != 1 || len(lenses) == 0 {
		return twoRimHoledTrim{}, false
	}
	if len(chart) > 0 && !lensCorridorOutrunsTheSampling(lenses, meanChainChord(outer3D)) {
		return twoRimHoledTrim{}, false // the general chart-driven mesher serves this band
	}
	return twoRimHoledTrim{rim: rims[0], lenses: lenses}, true
}

// nearPinchCorridorChords is how many boundary chords wide the corridor between two lens windows must
// be before the covering mesher is trusted with it.
//
// THE CONSTANT NO LONGER SEPARATES WHAT IT WAS CHOSEN TO SEPARATE, and saying so is the honest part
// (#3518). It was chosen on a sweep whose failure count by ratio read 0.5→8, 1→5, 2→2, 3→0, 4→0,
// 6→0, 8→0, 12→1, 20→1, 40→1 — a plateau of 3 … 8 with 4 inside it — and on the reading that the
// chart mesher "loses the corridor". Both were re-measured over the near-pinch corpus (the eight JOIN
// bodies of TestNearPinchCutJoinWatertight, at both facetings) after the boundary-side classification
// landed (chart_face_rim_side.go), driving chartFaceMesh on each band directly:
//
//	corridor/chord   0.053 0.065 0.105 0.149 | 0.105 0.129 0.211 0.298 | 0.421 0.515 0.842 1.190 | 0.842 1.031 1.683 2.380
//	unpaired edges       0     0     0     0 |     0     8     0     8 |     4     0     4     0 |     0     8     6     0
//
// Ten of the sixteen come back bounded by EXACTLY their rim, the region is within 0.0003 % of
// query.AnalyticFaceArea at PropertyQuality, and no rim segment anywhere is left unbounded. The six
// that fail do not sort by the ratio at all: 0.129 and 1.683 fail while 0.105 and 2.380 pass. Their
// cause is the covering seam, not the corridor — chart_face_replica.go carries the measurement and
// names what it needs.
//
// The UPPER bracket the first sweep gave still stands and still brackets the value: over the whole
// two-rim corpus the ratio is 8.6 … 8.8, or infinite for a band with a single window, on every band
// the chart mesher takes, so nothing between the near-pinch rows' 2.380 and that 8.6 has to be
// separated. 4 sits inside that gap.
//
// So the gate stays, at 4, because removing it ships those six as torn faces (196–1196 free edges on
// the body, measured), and it is now a SUPERSET keep bracketed above rather than a plateau. The split
// it produces is still asserted in BOTH directions by
// TestTheTwoRimArmKeepsOnlyWhatTheChartCannotServe, which is the constant's plant. It is the last
// thing holding the unrolled arm alive.
//
// #3517 measured both arms and deleted NEITHER. This one stays because the six torn rows need exact
// covering seams (#3542). The spiric arm stays because deleting it would ship a wrong body: its two
// remaining faces record no chart (#3550) and the chart mesher cannot take one (#3548, #3549), so they
// fall to the surface's whole domain. What #3517 DID delete is the second, shadowed loft that used to
// catch them silently. ADR-0061's "G13 stays open" section carries both measurements.
const nearPinchCorridorChords = 4 // tol:mesh-density (multiples of the boundary's own chord; see above)

// lensCorridorOutrunsTheSampling reports whether two lens windows pass within nearPinchCorridorChords of
// the boundary's own chord — the corridor the covering cannot resolve. A band with a single window has
// no corridor and never does.
func lensCorridorOutrunsTheSampling(lenses [][]math.Point3, chord float64) bool {
	return closestLensApproach(lenses) < nearPinchCorridorChords*chord
}

// closestLensApproach is the smallest distance between points of two DIFFERENT lens windows (+Inf when
// there is only one).
func closestLensApproach(lenses [][]math.Point3) float64 {
	best := stdmath.Inf(1)
	for i := range lenses {
		for j := i + 1; j < len(lenses); j++ {
			best = stdmath.Min(best, closestPointPair(lenses[i], lenses[j]))
		}
	}
	return best
}

// closestPointPair is the smallest distance between a point of a and a point of b.
func closestPointPair(a, b []math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, p := range a {
		for _, q := range b {
			best = stdmath.Min(best, float64(p.DistanceTo(q)))
		}
	}
	return best
}

// wedgeBandTrim is the wedge arm's recognition: the cylinder and its two oblique end chains.
type wedgeBandTrim struct {
	cyl  geom.Cylinder
	ends [2]wedgeEndChain
}

// wedgeBandTrimOf recognises an open oblique-ended cylinder wedge — two end chains, no seam and no
// closed rim (A1/D4's pyramid slant fillet).
func wedgeBandTrimOf(f *topo.Face, s geom.Surface, q Quality) (wedgeBandTrim, bool) {
	cyl, ends, ok := wedgeBandEndChains(f, s, q)
	if !ok {
		return wedgeBandTrim{}, false
	}
	return wedgeBandTrim{cyl: cyl, ends: ends}, true
}
