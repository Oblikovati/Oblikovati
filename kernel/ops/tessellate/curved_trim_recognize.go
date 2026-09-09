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
// J3 and A4 host tori record no chart (measured: chart=0, chartFaceMesh declines), and the loft meshes
// them in 340988 and 406540 triangles against the 1115132 and 1180684 the generic CDT downstream of it
// needs.
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
// The chart mesher lays its boundary CONSTRAINTS at the shared edges' own discretisation. Where two
// windows pass closer than a few of those chords, the two chord polygons no longer separate the
// corridor and the constrained triangulation loses it: measured on the #1818 near-pinch crossings
// (R = 3 and 30, |Δr| = 4e-5 … 3.2e-4 scaled), it comes back with 126–2359 unpaired edges against rims
// of 128–2304 and its region as much as 4 % out. The unroll's BENT seam is built for exactly that
// corridor (ADR-0061 stage 4), so those bands keep it.
//
// The measured corridor/chord ratio over the whole two-rim corpus is 0.05 … 2.4 on every band the chart
// mesher LOSES and 8.6 … 8.8 or infinite (a single window) on every band it takes, at all three sampled
// tolerances. Swept against the kernel's own corpus, the failure count by ratio is 0.5→8, 1→5, 2→2,
// 3→0, 4→0, 6→0, 8→0, 12→1, 20→1, 40→1: a plateau of 3 … 8, with 4 inside it. The split it produces is
// asserted in both directions by TestTheTwoRimArmKeepsOnlyWhatTheChartCannotServe.
const nearPinchCorridorChords = 4 // tol:mesh-density (multiples of the boundary's own chord)

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
