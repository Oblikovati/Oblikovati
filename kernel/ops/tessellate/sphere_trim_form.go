// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Which region of a sphere a face's trim cuts out (ADR-0061 stage 5, #3409). Four sphere meshers used
// to sit as four consecutive rungs of the curved-trim ladder, each declining so the next could try; the
// three cap rungs even shared one builder (buildSphereCap) and differed only in how they read the rim.
// They are one classification now: read the trim's rim inventory ONCE and name the form.

// sphereTrimForm names the region a sphere face's trim bounds.
type sphereTrimForm int

const (
	// sphereTrimNone is not a sphere, or a sphere trim no chart can hold.
	sphereTrimNone sphereTrimForm = iota
	// sphereTrimCap is bounded by ONE closed rim, with or without a meridian seam to the enclosed pole.
	sphereTrimCap
	// sphereTrimBelt is bounded by TWO coaxial closed rims.
	sphereTrimBelt
	// sphereTrimPatch is the residual: an arc-bounded trim, which is neither a cap nor a belt and which
	// a patch-centred gnomonic or stereographic chart holds.
	sphereTrimPatch
)

// classifySphereTrim names the form of a sphere face's trim. The cap and belt forms are disjoint by
// the rim inventory itself — a cap's boundary reduces to one closed rim, a belt's to two — and the
// patch form is the explicit RESIDUAL, so exactly one form can answer for a face.
//
// Example: a ball an axle passes through keeps a belt between the two bore rims → sphereTrimBelt,
// never sphereTrimPatch, whose chart covers less than a hemisphere and cannot hold a belt that
// straddles its own equator (#2061).
func classifySphereTrim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) sphereTrimForm {
	sph, isSphere := sphereOf(s)
	if !isSphere {
		return sphereTrimNone
	}
	if _, _, _, ok := sphereCapRim(f, s, outer3D, holes3D, q); ok {
		return sphereTrimCap
	}
	if _, _, _, ok := zoneBandRims(f, sph, q); ok {
		return sphereTrimBelt
	}
	if _, ok := sphereChartOf(f, sph, outer3D, holes3D); ok {
		return sphereTrimPatch
	}
	return sphereTrimNone
}

// sphereChartOf returns the patch-centred chart that holds an arc-bounded sphere trim, or ok=false
// when the trim reaches too far round the sphere for either chart.
func sphereChartOf(f *topo.Face, sph geom.Sphere, outer3D []math.Point3, holes3D [][]math.Point3) (sphereChart, bool) {
	if len(outer3D) < 3 {
		return nil, false
	}
	return chooseSphereChart(f, sph, outer3D, holes3D)
}

// sphereCapRim returns the sphere, the rim ring and the pole axis of a trim that is a CAP — the input
// buildSphereCap sweeps latitude rings over. Three rim FORMS reach it, and each excludes the other two
// by the shape of the boundary itself, so no order among them is load-bearing:
//
//   - a bare closed rim: every boundary sample is coplanar, which a loop carrying a pole vertex or a
//     seam chain cannot be (capAxis);
//   - one full-circle rim EDGE plus a meridian seam down to an enclosed pole vertex: the loop holds
//     exactly one closed conic edge and a lone pole vertex (zoneRimAxis);
//   - a coplanar MULTI-ARC rim closed by one doubled seam edge ending at the pole: the loop's rim is
//     several edges, so it holds no single full-circle edge (recognizeSeamedCapRim).
//
// Example: a hemisphere whose equator is one circle edge → the bare form, rim = the equator samples.
func sphereCapRim(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (geom.Sphere, []math.Point3, math.Vector3, bool) {
	sph, isSphere := sphereOf(s)
	if !isSphere {
		return sph, nil, math.Vector3{}, false
	}
	if axis, ok := planarCircleRimAxis(sph, outer3D, holes3D); ok {
		return sph, outer3D, axis, true
	}
	if f == nil || len(f.Loops()) != 1 {
		return sph, nil, math.Vector3{}, false
	}
	if rim, axis, ok := zoneRimAxis(f, sph, q); ok && len(rim) >= 3 {
		return sph, rim, axis, true
	}
	rim, axis, ok := recognizeSeamedCapRim(f, sph, q)
	return sph, rim, axis, ok
}

// sphereCapFanMesh meshes a cap trim: latitude rings from its rim to the pole the rim encloses.
func sphereCapFanMesh(f *topo.Face, s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, q Quality) (*Mesh, bool) {
	sph, rim, axis, ok := sphereCapRim(f, s, outer3D, holes3D, q)
	if !ok {
		return nil, false
	}
	return buildSphereCap(sph, rim, axis, q), true
}
