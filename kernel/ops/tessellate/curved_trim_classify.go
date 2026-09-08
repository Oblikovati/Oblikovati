// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The curved-trim CLASSIFICATION (ADR-0061 stage 5, #3409). It replaced specialCurvedMeshers, an
// eleven-entry ordered try-list in which each mesher "declined" so the next could claim the face —
// the last first-fit ladder in the kernel, and the shape the ground rules forbid ("dispatch is a
// classification that selects exactly one path; never add to an ordered try-list, a first-fit ladder,
// or a load-bearing order").
//
// What replaces it is a classification: classifyCurvedTrim reads the face's trim ONCE and names the
// one kind it is, and specialCurvedMesh switches on that name. The predicates are mutually exclusive,
// so the order they are written in decides nothing — TestCurvedTrimKindsAreMutuallyExclusive evaluates
// every one of them on every corpus face and fails if two answer true for the same face. That test is
// what a ladder can never have: with a ladder, "two rungs both accept this face" is not a bug, it is
// the mechanism.
//
// The classification is TOTAL: a face no special kind claims is named too — kindChart when it carries
// the parametric trim ADR-0063 records, so the chart-driven mesher meshes the face's OWN region, and
// kindUncharted when it does not, which keeps the generic (u,v) trim path.

// curvedTrimKind names the ONE mesher a curved face's trim selects.
type curvedTrimKind int

const (
	// kindUncharted is a trim no special kind claims on a face that carries no chart. It keeps the
	// generic (u,v) trim path (iso grid, rectilinear cells, or the boundary CDT).
	kindUncharted curvedTrimKind = iota
	// kindChart is a trim no special kind claims on a face that DOES carry a chart: chartFaceMesh
	// meshes the region the face records, taking its boundary points from the shared edges.
	kindChart
	// kindConeApexFan is a cone closing to its apex — a drill point, an oblique apex cut, or an
	// apex-collapsed angular sector. A cone is developable, so a fan from the apex has exact area.
	kindConeApexFan
	// kindSphereCapFan is a sphere trim whose rim is one closed circle (with or without a meridian
	// seam running to the enclosed pole): latitude rings from the rim to that pole.
	kindSphereCapFan
	// kindSphereZoneBand is a sphere trim between two coaxial closed rims: a belt of latitude rings.
	kindSphereZoneBand
	// kindSpherePatch is an arc-bounded sphere trim: a constrained triangulation in a patch-centred
	// gnomonic or stereographic chart.
	kindSpherePatch
	// kindRuledBandLoft is a developable side bounded by two full-wrap rims with no lens hole — two
	// closed rims, or one closed rim and one notched rim. A ruled band needs no interior row.
	kindRuledBandLoft
	// kindSpiricBand is a torus band bounded by two edges that each wrap the whole tube.
	kindSpiricBand
	// kindTwoRimHoledBand is a developable side with two full-wrap rims that also carries lens holes:
	// bridge the rims at a seam, unroll, triangulate the holes into the unrolled branch.
	kindTwoRimHoledBand
	// kindWedgeBand is an open oblique-ended cylinder wedge: one zipped strip between its two end
	// chains.
	kindWedgeBand
)

// String names the kind, so a failing classification test says which two kinds collided.
func (k curvedTrimKind) String() string {
	names := [...]string{"uncharted", "chart", "cone-apex-fan", "sphere-cap-fan", "sphere-zone-band",
		"sphere-patch", "ruled-band-loft", "spiric-band", "two-rim-holed-band", "wedge-band"}
	if int(k) < 0 || int(k) >= len(names) {
		return "curvedTrimKind(unknown)"
	}
	return names[k]
}

// classifyCurvedTrim names the one kind a curved face's trim is. The predicates it reads are mutually
// exclusive (TestCurvedTrimKindsAreMutuallyExclusive), so this reads as a chain only because Go has no
// "the one true predicate" expression — reordering it changes no answer.
//
// Example: the wall of a rod a ball is set into carries one full-wrap rim plus the ball's lens window,
// so it classifies as kindTwoRimHoledBand and nothing else.
func classifyCurvedTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) curvedTrimKind {
	if isConeApexTrim(f, s, outer3D, holes3D) {
		return kindConeApexFan
	}
	switch classifySphereTrim(f, s, outer3D, holes3D, q) {
	case sphereTrimCap:
		return kindSphereCapFan
	case sphereTrimBelt:
		return kindSphereZoneBand
	case sphereTrimPatch:
		return kindSpherePatch
	}
	if isRuledTwoRimBand(f, s, q) {
		return kindRuledBandLoft
	}
	if isSpiricTubeBand(f, s, q) {
		return kindSpiricBand
	}
	if isTwoRimHoledBand(s, holes3D) {
		return kindTwoRimHoledBand
	}
	if isWedgeBandTrim(f, s, q) {
		return kindWedgeBand
	}
	return chartedKind(f)
}

// chartedKind splits the faces no special kind claims by whether they carry the parametric trim
// ADR-0063 records — the one thing that decides whether the chart-driven mesher can mesh the face's
// own region rather than the surface's whole domain.
func chartedKind(f *topo.Face) curvedTrimKind {
	if len(f.Chart()) > 0 {
		return kindChart
	}
	return kindUncharted
}

// isConeApexTrim reports whether the trim is a cone closing to its apex. A holed cone face never is:
// its inner rim is a hole, not the fan's far end.
func isConeApexTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) bool {
	_, _, _, ok := coneApexFanRim(f, s, outer3D, holes3D)
	return ok
}

// isSphereCapTrim, isSphereBeltTrim and isSpherePatchTrim read ONE inventory of the sphere trim's
// rims (classifySphereTrim), so their exclusivity is a property of that function rather than a
// coincidence between three independent tests.
func isSphereCapTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) bool {
	return classifySphereTrim(f, s, outer3D, holes3D, q) == sphereTrimCap
}

// isSphereBeltTrim reports whether the sphere trim is the belt between two coaxial closed rims.
func isSphereBeltTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) bool {
	return classifySphereTrim(f, s, outer3D, holes3D, q) == sphereTrimBelt
}

// isSpherePatchTrim reports whether the sphere trim is the arc-bounded patch a chart holds.
func isSpherePatchTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) bool {
	return classifySphereTrim(f, s, outer3D, holes3D, q) == sphereTrimPatch
}

// isRuledTwoRimBand reports whether the trim is a developable side bounded by two full-wrap rims and
// carrying no lens hole — the shape the ruled loft stitches rim-to-rim with no interior row. A band
// carrying a genuine (non-wrapping) lens hole is kindTwoRimHoledBand instead: the loft pools ALL open
// edges into one rim, so it would fold the lens into the base rim (#1591).
func isRuledTwoRimBand(f *topo.Face, s geom.Surface, q Quality) bool {
	return isDevelopableSide(s) && !faceHasLensHole(f, s, q) && isPeriodicTwoRimBand(f)
}

// isSpiricTubeBand reports whether the trim is a torus band bounded by two edges that each wrap the
// whole tube (#1375) — a torus cut through its hole.
func isSpiricTubeBand(f *topo.Face, s geom.Surface, q Quality) bool {
	_, _, ok := tubeWrappingEdges(f, s, q)
	return ok
}

// isTwoRimHoledBand reports whether the trim is a singly-periodic developable side whose hole loops are
// ONE full-wrap rim plus at least one lens window. The lens is what separates it from
// kindRuledBandLoft, which the pure rim-to-rim loft can mesh precisely because it carries none.
func isTwoRimHoledBand(s geom.Surface, holes3D [][]math.Point3) bool {
	if !isDevelopableSide(s) || IsPeriodic(s.UDomain()) == IsPeriodic(s.VDomain()) {
		return false
	}
	rims, lenses := splitWrappingHoles(s, holes3D)
	return len(rims) == 1 && len(lenses) > 0
}

// isWedgeBandTrim reports whether the trim is an open oblique-ended cylinder wedge — two end chains,
// no seam and no closed rim (A1/D4's pyramid slant fillet).
func isWedgeBandTrim(f *topo.Face, s geom.Surface, q Quality) bool {
	_, ok := wedgeBandEndChains(f, s, q)
	return ok
}
