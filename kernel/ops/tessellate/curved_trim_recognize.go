// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
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
