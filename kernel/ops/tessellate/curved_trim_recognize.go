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
// through its hole.
func spiricTubeTrimOf(f *topo.Face, s geom.Surface, q Quality) (spiricTubeTrim, bool) {
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
// rim plus at least one lens window. The lens is what separates it from kindRuledBandLoft, whose pure
// rim-to-rim loft is exact precisely because it carries none.
//
// This arm is the one the chart mesher very nearly takes, and the measurement of why it does not is
// worth keeping (ADR-0061 stage 5). The unroll is not an exact fast path — it bridges the two rims at
// an invented seam and triangulates the flattened branch, and on the rod a ball is set into it meshes
// the right 24.5 mm² of wall with triangles whose planes pass 0.5 from the axis, so the wall
// integrates 7.19 where 8.26 is right. Per FACE the chart mesher is better: over the corpus's 19
// charted two-rim holed bands it matches the unroll to ±0.2% of area on 18 and betters the rod wall by
// 1.5%, with 40–85% fewer triangles and the same rim count, and routing them to it moves RODB∪/RODB−
// from 8.72%/9.02% to 1.44%/1.43%. Per BODY it is not: at PropertyQuality the corner junction's wall
// (#1738) comes back with 870 rim edges against its neighbours' 864, cracking the body with 6 free
// edges, and its area FALLS from 160.93 to 158.65 as the chord tolerance tightens — refinement is
// meant to raise it. Until that is fixed the wall the boolean charts stays on the unroll.
func twoRimHoledTrimOf(s geom.Surface, holes3D [][]math.Point3) (twoRimHoledTrim, bool) {
	if !isDevelopableSide(s) || IsPeriodic(s.UDomain()) == IsPeriodic(s.VDomain()) {
		return twoRimHoledTrim{}, false
	}
	rims, lenses := splitWrappingHoles(s, holes3D)
	if len(rims) != 1 || len(lenses) == 0 {
		return twoRimHoledTrim{}, false
	}
	return twoRimHoledTrim{rim: rims[0], lenses: lenses}, true
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
