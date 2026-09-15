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
// What replaces it is a classification: classifyCurvedTrim reads the face's trim ONCE and returns the
// one kind it is TOGETHER WITH the recognition the arm needs, and specialCurvedMesh switches on that
// kind and hands the payload straight to the builder. Nothing is recognised twice ("decide each
// incidence once and reuse the result"), and the recognizers are mutually exclusive, so the order they
// are written in decides nothing — TestCurvedTrimKindsAreMutuallyExclusive evaluates every one of them
// on every corpus face and fails if two answer true for the same face. That test is what a ladder can
// never have: with a ladder, "two rungs both accept this face" is not a bug, it is the mechanism.
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
	// kindRuledBandLoft is a developable side bounded by two full-wrap rims — two closed rims, or one
	// closed rim and one notched rim. A ruled band needs no interior row.
	kindRuledBandLoft
	// kindWedgeBand is an open oblique-ended cylinder wedge: one zipped strip between its two end
	// chains.
	kindWedgeBand
)

// String names the kind, so a failing classification test says which two kinds collided.
func (k curvedTrimKind) String() string {
	names := [...]string{"uncharted", "chart", "cone-apex-fan", "sphere-cap-fan", "sphere-zone-band",
		"sphere-patch", "ruled-band-loft", "wedge-band"}
	if int(k) < 0 || int(k) >= len(names) {
		return "curvedTrimKind(unknown)"
	}
	return names[k]
}

// curvedTrim is the classification's verdict: the kind, and the recognition the selected mesher needs.
// Exactly one payload field is meaningful, the one its kind names; the rest are zero. Carrying the
// payload is what keeps a recognizer from running twice — once to decide and once to build.
type curvedTrim struct {
	kind  curvedTrimKind
	cone  coneApexTrim
	cap   sphereCapTrim
	belt  sphereBeltTrim
	patch spherePatchTrim
	wedge wedgeBandTrim
}

// classifyCurvedTrim names the one kind a curved face's trim is and carries its recognition. The
// recognizers it reads are mutually exclusive (TestCurvedTrimKindsAreMutuallyExclusive), so this reads
// as a chain only because Go has no "the one true recognizer" expression — reordering changes no answer.
//
// Example: the wall of a rod a ball is set into carries one full-wrap rim plus the ball's lens window,
// so it classifies as kindTwoRimHoledBand, carrying that rim and that lens, and nothing else.
func classifyCurvedTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) curvedTrim {
	if c, ok := coneApexTrimOf(f, s, outer3D, holes3D); ok {
		return curvedTrim{kind: kindConeApexFan, cone: c}
	}
	if t, ok := classifySphereTrim(f, s, outer3D, holes3D, q); ok {
		return t
	}
	if ruledTwoRimBandHolds(f, s, q) {
		return curvedTrim{kind: kindRuledBandLoft}
	}
	if w, ok := wedgeBandTrimOf(f, s, q); ok {
		return curvedTrim{kind: kindWedgeBand, wedge: w}
	}
	return curvedTrim{kind: chartedKind(f, s, outer3D, holes3D)}
}

// developsIntoOneBranch reports whether a face's 3-D loops carry into ONE branch of the surface's
// (u,v) domain. An OUTERLESS face — the closed-surface complement, whose boundary is holes only — has
// no outer loop to carry, and its region is exactly what a chart records, so it never develops.
func developsIntoOneBranch(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) bool {
	if len(outer3D) < 3 {
		return false
	}
	_, _, ok := ToUVLoops(s, outer3D, holes3D)
	return ok
}

// chartedKind splits the faces no special kind claims by whether the chart-driven mesher is the one
// that has to serve them.
//
// It asks TWO questions, and the second is the one that keeps the chart a DATUM rather than a mesher
// selection. A chart says which of two complementary regions a face on a periodic surface is — and
// that only decides anything when the face's 3-D loops cannot be developed into ONE branch of the
// (u,v) domain. Where ToUVLoops does develop them, the loops already determine the region, the
// structured grid and the boundary CDT downstream mesh it exactly and cheaply, and routing it through
// the covering buys nothing while costing the covering's price.
//
// That price is not small, and it is why the second question exists: recording charts on imported and
// rebuilt faces (#3550) sent every such face through the covering, and the OCC tessellation budget went
// 0.08 s → 6.37 s against its 2.15 s ceiling while the occtparity corpus tier stopped finishing inside
// 2400 s. Asking whether the trim DEVELOPS puts each face back on the path that can serve it, and
// leaves the covering the faces that have no other.
//
// It is not a first-fit ladder: the question is a property of the TRIM, computed once, and it selects
// exactly one path (the ground rule's "dispatch is a classification"). A face whose trim does not
// develop and which records no chart is kindUncharted, and the seam-crossing router reports what it
// falls to.

func chartedKind(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) curvedTrimKind {
	if len(f.Chart()) == 0 {
		return kindUncharted
	}
	if developsIntoOneBranch(s, outer3D, holes3D) {
		return kindUncharted
	}
	return kindChart
}
