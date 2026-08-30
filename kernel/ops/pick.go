// SPDX-License-Identifier: GPL-2.0-only

package ops

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

// rayCastMesh returns the nearest positive hit distance of a ray against a mesh's
// triangles.
// RayCastMesh returns the nearest forward hit distance of the ray (origin, dir) against the
// mesh's triangles, and whether any was hit. It lets callers ray-test against a mesh they
// already hold (e.g. the hidden-line engine occlusion-tests projected edge points against a
// once-tessellated body) without re-tessellating per ray.
func RayCastMesh(m *Mesh, origin math.Point3, dir math.Vector3) (float64, bool) {
	return rayCastMesh(m, origin, dir)
}

func rayCastMesh(m *Mesh, origin math.Point3, dir math.Vector3) (float64, bool) {
	best := stdmath.Inf(1)
	hit := false
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a := m.Positions[m.Indices[i]]
		b := m.Positions[m.Indices[i+1]]
		c := m.Positions[m.Indices[i+2]]
		if t, ok := rayTriangleDist(origin, dir, a, b, c); ok && t < best {
			best, hit = t, true
		}
	}
	return best, hit
}

// rayTriangleDist returns the positive distance along the ray to triangle abc, via
// Möller–Trumbore, or ok=false if there is no forward hit.
func rayTriangleDist(orig math.Point3, dir math.Vector3, a, b, c math.Point3) (float64, bool) {
	const eps = 1e-9
	e1 := a.VectorTo(b)
	e2 := a.VectorTo(c)
	pv := dir.Cross(e2)
	det := e1.Dot(pv)
	if det > -eps && det < eps {
		return 0, false
	}
	inv := 1 / det
	tv := a.VectorTo(orig)
	u := tv.Dot(pv) * inv
	if u < 0 || u > 1 {
		return 0, false
	}
	qv := tv.Cross(e1)
	v := dir.Dot(qv) * inv
	if v < 0 || u+v > 1 {
		return 0, false
	}
	t := e2.Dot(qv) * inv
	return t, t > eps
}
