// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/math"
)

// A sub-face point that lies IN a plane the other solid's boundary occupies — but on no face of it —
// cannot be classified by a volumetric query AT that point, and neither evaluator degrades gracefully
// there (#3459, the multipoint disk):
//
//   - the ray-parity classifier pierces every one of that plane's faces at t=0, so every candidate
//     direction grazes and firstCleanDirection finds none. The degeneracy is at the ray ORIGIN, so
//     re-selecting the direction — the mechanism built for a grazing crossing — cannot clear it;
//   - the winding-number fallback zeroes exactly those faces by design (faceSolidAngle's on-plane
//     rule), leaving a sum that reads "outside" for a point in the solid's interior.
//
// coplanarCover already resolves the point that is ON a face of the other solid, through the ON/ON
// table. This resolves the one that is on its PLANE and off its faces: the point is then provably not
// on the other solid's boundary, so the material to either side of the plane is the same material,
// and a probe on each side answers the question the point itself cannot. Requiring the two probes to
// AGREE is what makes it a certificate rather than a guess — a disagreement means the point is on the
// boundary after all (a cover coplanarCover missed), and the caller keeps its own verdict.
//
// The probes are offset in the ONLY direction that can escape the degenerate plane, its normal, by a
// multiple of the solid's own on-plane resolution. No output coordinate moves: this displaces a
// classification query, never geometry (SoS discipline).

// offPlaneProbeSteps places each probe clear of the on-plane band that made the query at p
// degenerate, without reaching for neighbouring geometry: the band is the classifier's own
// grazing width, so a few multiples of it is off-plane by construction and still far below any
// modelled feature.
const offPlaneProbeSteps = 8

// insideOffPlane answers membership of a point lying in a plane of the solid's boundary that no face
// of it covers, by probing a resolution-derived step to each side of that plane. ok=false when the two
// sides disagree — the point is on the boundary, and the caller decides.
func insideOffPlane(o insideOracle, p math.Point3, n math.Vector3, step float64) (in, ok bool) {
	off := n.Scale(math.Scalar(step))
	up := o.inside(p.TranslateBy(off))
	down := o.inside(p.TranslateBy(off.Scale(-1)))
	if up != down {
		return false, false
	}
	return up, true
}

// insidePlaneSafe is the membership query classifySubFace uses for a sub-face point: the plain
// volumetric query, except where coplanarCover found the point degenerate — in a plane the other
// solid's boundary occupies, within the band of a coplanar face's own boundary — which the two-sided
// probe above resolves.
func insidePlaneSafe(other insideOracle, p math.Point3, n math.Vector3, step float64, degenerate bool) bool {
	if !degenerate {
		return other.inside(p)
	}
	if in, ok := insideOffPlane(other, p, n, step); ok {
		return in
	}
	return other.inside(p)
}
