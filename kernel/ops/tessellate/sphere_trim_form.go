// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Which region of a sphere a face's trim cuts out (ADR-0061 stage 5, #3409). Four sphere meshers used
// to sit as four consecutive rungs of the curved-trim ladder, each declining so the next could try, and
// three of those four shared one builder (buildSphereCap) and differed only in how they read the RIM.
//
// They are one classification now, and the cap's three rim forms are read as an INVENTORY rather than
// as a chain: all three are evaluated, the inventory reports how many held, and a rim that answers to
// two forms is not a cap at all. Nothing is decided by the order the forms are written in, and
// TestSphereCapRimFormsAreDisjoint evaluates them independently on every corpus sphere face.

// sphereCapRimForm names one of the three boundary shapes a spherical CAP presents. They are disjoint
// by the boundary's own shape, and the inventory below is what proves it rather than assuming it:
//
//   - rimFormPlanarCircle: every boundary sample is coplanar, which a loop carrying a pole vertex or a
//     seam chain cannot be;
//   - rimFormPoleSeamed: the loop holds exactly ONE full-circle rim edge plus a lone off-plane pole
//     vertex, so its samples are not coplanar and the planar form cannot read it;
//   - rimFormMultiArcSeam: the rim is SEVERAL coplanar arcs closed by one doubled seam edge ending at
//     the pole, so it has two or more rim edges where the pole-seamed form has exactly one.
//
// That last discriminator is measured, not assumed. A loop of [seam, one full circle, seam-reversed]
// satisfies BOTH the pole-seamed reading (one full-circle edge, a lone pole vertex) and the seamed
// reading (a lone doubled edge, a coplanar rim ring) — it is OCCT blend/simple J2, and the ladder hid
// the contradiction by trying the pole-seamed rung first. rimFormMultiArcSeam therefore requires what
// its name has always claimed: a rim of two or more edges.
type sphereCapRimForm int

const (
	rimFormPlanarCircle sphereCapRimForm = iota
	rimFormPoleSeamed
	rimFormMultiArcSeam
	sphereCapRimFormCount
)

// String names the rim form, so a failing disjointness test says which two collided.
func (r sphereCapRimForm) String() string {
	names := [...]string{"planar-circle-rim", "pole-seamed-rim", "multi-arc-seam-rim"}
	if int(r) < 0 || int(r) >= len(names) {
		return "sphereCapRimForm(unknown)"
	}
	return names[r]
}

// sphereCapTrim is the cap arm's recognition: the sphere, the rim ring buildSphereCap sweeps from, and
// the axis pointing at the pole that rim encloses.
type sphereCapTrim struct {
	sph  geom.Sphere
	rim  []math.Point3
	axis math.Vector3
}

// sphereBeltTrim is the belt arm's recognition: the sphere, its two coaxial rims near-to-far, and the
// frame their shared axis defines.
type sphereBeltTrim struct {
	sph       geom.Sphere
	near, far zoneRing
	frame     zoneFrame
}

// spherePatchTrim is the patch arm's recognition: the sphere and the patch-centred chart that holds it.
type spherePatchTrim struct {
	sph   geom.Sphere
	chart sphereChart
}

// classifySphereTrim names which region of a sphere the face's trim bounds, with the recognition its
// builder needs. The cap and the belt are evaluated INDEPENDENTLY and both must not hold — an
// ambiguous reading is refused rather than resolved by position, so no order here is load-bearing.
// The patch is the declared RESIDUAL and is deliberately not independent: a chart can hold a cap too,
// but a cap is not a patch, because the latitude fan is exact where the chart's CDT chords across the
// curvature (#2061: the belt straddles its own equator and no chart covers it at all).
func classifySphereTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (curvedTrim, bool) {
	sph, isSphere := sphereOf(s)
	if !isSphere {
		return curvedTrim{}, false
	}
	capTrim, isCap := sphereCapTrimOf(f, sph, outer3D, holes3D, q)
	belt, isBelt := sphereBeltTrimOf(f, sph, q)
	if isCap != isBelt {
		return oneSphereRegion(capTrim, isCap, belt), true
	}
	if isCap && isBelt {
		return curvedTrim{}, false // ambiguous: neither fan may claim it, so the residual takes it
	}
	patch, isPatch := spherePatchTrimOf(f, sph, outer3D, holes3D)
	return curvedTrim{kind: kindSpherePatch, patch: patch}, isPatch
}

// oneSphereRegion wraps whichever of the two exclusive sphere regions held.
func oneSphereRegion(capTrim sphereCapTrim, isCap bool, belt sphereBeltTrim) curvedTrim {
	if isCap {
		return curvedTrim{kind: kindSphereCapFan, cap: capTrim}
	}
	return curvedTrim{kind: kindSphereZoneBand, belt: belt}
}

// sphereCapTrimOf reads the cap rim INVENTORY — all three forms, not the first that answers — and
// accepts only when exactly one held. Two forms answering is a contradiction about what the boundary
// IS, and resolving it by position is what the ladder used to do; refusing it sends the face to the
// residual instead, where a chart still meshes it.
func sphereCapTrimOf(f *topo.Face, sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (sphereCapTrim, bool) {
	held, trim := 0, sphereCapTrim{}
	for form := range sphereCapRimFormCount {
		if got, ok := sphereCapRimOfForm(form, f, sph, outer3D, holes3D, q); ok {
			held, trim = held+1, got
		}
	}
	return trim, held == 1
}

// sphereCapRimOfForm reads ONE named rim form. Each form fully gates itself, so the caller may ask for
// them in any order and get the same inventory.
func sphereCapRimOfForm(form sphereCapRimForm, f *topo.Face, sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (sphereCapTrim, bool) {
	switch form {
	case rimFormPlanarCircle:
		return planarCircleCapRim(sph, outer3D, holes3D)
	case rimFormPoleSeamed:
		return poleSeamedCapRim(f, sph, q)
	case rimFormMultiArcSeam:
		return multiArcSeamCapRim(f, sph, q)
	}
	return sphereCapTrim{}, false
}

// planarCircleCapRim reads the BARE rim form — a boundary that is one closed planar circle and nothing
// else. A face carrying HOLES is not that form: the fan sweeps the rim straight to the pole and would
// pave right over them, so it declines and lets the belt or the patch take the face.
func planarCircleCapRim(sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3) (sphereCapTrim, bool) {
	if len(outer3D) < 3 || len(holes3D) > 0 {
		return sphereCapTrim{}, false
	}
	axis, ok := capAxis(sph, outer3D)
	return sphereCapTrim{sph: sph, rim: outer3D, axis: axis}, ok
}

// poleSeamedCapRim reads the form whose outer loop is ONE full-circle rim edge plus a meridian seam
// down to an enclosed pole vertex off the rim plane (OCCT blend/simple J2). A holed face is not a
// pole-reaching zone; the fan would pave over the hole.
func poleSeamedCapRim(f *topo.Face, sph geom.Sphere, q Quality) (sphereCapTrim, bool) {
	if f == nil || len(f.Loops()) != 1 {
		return sphereCapTrim{}, false
	}
	rim, axis, ok := zoneRimAxis(f, sph, q)
	return sphereCapTrim{sph: sph, rim: rim, axis: axis}, ok && len(rim) >= 3
}

// multiArcSeamCapRim reads the form whose outer loop is a coplanar MULTI-ARC rim closed by one doubled
// seam edge running to the pole (S6/S7's imported boss hemisphere).
func multiArcSeamCapRim(f *topo.Face, sph geom.Sphere, q Quality) (sphereCapTrim, bool) {
	if f == nil || len(f.Loops()) != 1 || rimEdgeCount(f.Loops()[0]) < 2 {
		return sphereCapTrim{}, false
	}
	rim, axis, ok := recognizeSeamedCapRim(f, sph, q)
	return sphereCapTrim{sph: sph, rim: rim, axis: axis}, ok
}

// rimEdgeCount is how many DISTINCT edges the loop uses once, i.e. its rim edges — a doubled edge is
// the seam and belongs to neither rim nor count. One is the pole-seamed form's signature, two or more
// the multi-arc form's, and that is what keeps the two readings apart.
func rimEdgeCount(l *topo.Loop) int {
	uses := map[*topo.Edge]int{}
	for _, u := range l.EdgeUses() {
		uses[u.Edge()]++
	}
	n := 0
	for _, count := range uses {
		if count == 1 {
			n++
		}
	}
	return n
}

// sphereBeltTrimOf recognises the belt between two coaxial closed rims.
func sphereBeltTrimOf(f *topo.Face, sph geom.Sphere, q Quality) (sphereBeltTrim, bool) {
	near, far, fr, ok := zoneBandRims(f, sph, q)
	if !ok {
		return sphereBeltTrim{}, false
	}
	return sphereBeltTrim{sph: sph, near: near, far: far, frame: fr}, true
}

// spherePatchTrimOf recognises the arc-bounded residual: a sphere trim a patch-centred gnomonic or
// stereographic chart can hold. ok=false when the trim reaches too far round the sphere for either.
func spherePatchTrimOf(f *topo.Face, sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3) (spherePatchTrim, bool) {
	if len(outer3D) < 3 {
		return spherePatchTrim{}, false
	}
	chart, ok := chooseSphereChart(f, sph, outer3D, holes3D)
	return spherePatchTrim{sph: sph, chart: chart}, ok
}
