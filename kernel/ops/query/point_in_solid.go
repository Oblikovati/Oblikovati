// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// pointInsideWindingThreshold is the generalized-winding-number cutoff for "inside". For a closed,
// outward-oriented mesh the winding number is ~1 strictly inside and ~0 strictly outside, crossing
// 0.5 on the surface; the 0.5 cutoff is the maximally robust split and tolerates small mesh cracks
// (a hairline gap perturbs the number by ≪0.5 rather than flipping a ray's parity).
const pointInsideWindingThreshold = 0.5

// windingNumber returns the generalized winding number of mesh around p (Jacobson, Kavan &
// Sorkine, "Robust Inside-Outside Segmentation using Generalized Winding Numbers", SIGGRAPH 2013):
// w(p) = (1/4π)·Σ signed_solid_angle(p; a,b,c). Unlike a single parity ray it has no grazing-edge or
// grazing-vertex degeneracy (it integrates the whole boundary, it does not sample one direction) and
// degrades gracefully on a cracked mesh. The mesh is expected outward-oriented — TessellateBody
// already runs orientFacesOutward — so w≈+1 inside the solid material and ≈0 outside.
func windingNumber(mesh *Mesh, p math.Point3) float64 {
	sum := 0.0
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		a := mesh.Positions[mesh.Indices[t]]
		b := mesh.Positions[mesh.Indices[t+1]]
		c := mesh.Positions[mesh.Indices[t+2]]
		sum += signedSolidAngle(p, a, b, c)
	}
	return sum / (4 * stdmath.Pi)
}

// signedSolidAngle is geom.SignedSolidAngle (the Van Oosterom–Strackee formula).
func signedSolidAngle(p, a, b, c math.Point3) float64 {
	return geom.SignedSolidAngle(p, a, b, c)
}

// PointInMesh reports whether p lies inside the solid bounded by an outward-oriented mesh, by
// thresholding the generalized winding number.
//
// It has NO production caller: every topological inside/outside decision now uses the analytic
// classifier brep.PointInside (M48/C3 #3422/#3423/#3426/#3427 retired the tessellated oracle). It is
// kept as an INDEPENDENT test oracle — the certification suites cross-check the analytic classifier and
// the exact boolean against this mesh winding number, so it must stay a separate implementation (a test
// that checked the analytic classifier against itself would prove nothing).
func PointInMesh(mesh *Mesh, p math.Point3) bool {
	return windingNumber(mesh, p) > pointInsideWindingThreshold
}
