// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// SignedSolidAngle returns the signed solid angle (steradians) that triangle (a,b,c) subtends at p,
// via the Van Oosterom–Strackee atan2 formula (IEEE Trans. Biomed. Eng. 1983) — numerically stable
// across the full sphere, including the back hemisphere where a naive arccos loses sign/precision.
// The sign follows the triangle's winding (right-hand rule), so the sum over an outward-oriented
// closed boundary is +4π at an interior point and 0 outside — the building block of the generalized
// winding number (Jacobson, Kavan & Sorkine, SIGGRAPH 2013) used by both the mesh classifier
// (kernel/ops, #1317) and the planar-boolean fragment classifier (kernel/brep, #1599). A degenerate
// triangle, or p coincident with a corner, contributes 0.
//
// Example — the +Z octant triangle subtends π/2 at the origin:
//
//	w := geom.SignedSolidAngle(math.P3(0,0,0), math.P3(1,0,0), math.P3(0,1,0), math.P3(0,0,1))
func SignedSolidAngle(p, a, b, c math.Point3) float64 {
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
