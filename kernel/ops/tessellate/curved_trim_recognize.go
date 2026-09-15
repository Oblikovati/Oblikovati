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
