// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

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

// signedSolidAngle returns the signed solid angle (steradians) that triangle (a,b,c) subtends at p,
// via the Van Oosterom–Strackee atan2 formula (IEEE Trans. Biomed. Eng. 1983) — numerically stable
// across the full sphere, including the back hemisphere where a naive arccos loses sign/precision.
// The sign follows the triangle's winding (right-hand rule), so the sum over an outward mesh is +4π
// inside and 0 outside. A degenerate triangle, or p coincident with a corner, contributes 0.
func signedSolidAngle(p, a, b, c math.Point3) float64 {
	va, vb, vc := p.VectorTo(a), p.VectorTo(b), p.VectorTo(c)
	la, lb, lc := va.Length(), vb.Length(), vc.Length()
	if la == 0 || lb == 0 || lc == 0 {
		return 0
	}
	num := float64(va.Dot(vb.Cross(vc)))
	den := float64(la*lb*lc) + float64(va.Dot(vb))*float64(lc) +
		float64(vb.Dot(vc))*float64(la) + float64(vc.Dot(va))*float64(lb)
	return 2 * stdmath.Atan2(num, den)
}

// pointInMesh reports whether p lies inside the solid bounded by an outward-oriented mesh, by
// thresholding the generalized winding number. Hoisting the (already-tessellated) mesh in keeps a
// multi-point query — e.g. classifying every vertex of one operand against another — at O(points·tris)
// without re-tessellating per point (the cost that made allVerticesInside O(V·T) before #1317).
func pointInMesh(mesh *Mesh, p math.Point3) bool {
	return windingNumber(mesh, p) > pointInsideWindingThreshold
}
