// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// RayCastFaces returns the nearest face of a body hit by a ray, with the hit
// distance — the geometric core of viewport picking (the renderer's ID-buffer is an
// optimization of the same query). The closest hit wins. Pure Go, so the UI's
// hit-test is headless-testable. q is retained for signature stability; the hit is now
// resolved analytically (see rayCastFace) and no longer depends on the tessellation Quality.
func RayCastFaces(b *topo.Body, origin math.Point3, dir math.Vector3, q Quality) (*topo.Face, float64, bool) {
	var nearest *topo.Face
	best := stdmath.Inf(1)
	for _, f := range b.Faces() {
		if t, ok := rayCastFace(f, origin, dir); ok && t < best {
			best, nearest = t, f
		}
	}
	return nearest, best, nearest != nil
}

// rayCastFace returns the ray's forward hit parameter to a face, resolved analytically: the exact
// ray∩surface pierce (geom.RaySurfaceHits) confirmed against the face's trimmed region with the
// parameter-space classifier (brep.PointInFaceTrim). It reads no tessellation, so a pick →
// reference-key resolution lands on the exact B-rep and is identical at every display Quality
// (M48/C3). This replaced the per-frame face triangulation that starved the frame-loop dispatcher
// on a degenerate planar boundary (the ear-clipper's O(m³) big.Rat escalation) — a hard placement
// deadlock — and the curved-face pick-tessellation memo it needed.
func rayCastFace(f *topo.Face, origin math.Point3, dir math.Vector3) (float64, bool) {
	t, _, ok := analyticFaceRayHit(f, origin, dir)
	return t, ok
}
