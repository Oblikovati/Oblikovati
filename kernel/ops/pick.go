// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// RayCastFaces returns the nearest face of a body hit by a ray, with the hit
// distance — the geometric core of viewport picking (the renderer's ID-buffer is an
// optimization of the same query). Each face is tessellated and its triangles
// tested; the closest hit wins. Pure Go, so the UI's hit-test is headless-testable.
func RayCastFaces(b *topo.Body, origin math.Point3, dir math.Vector3, q Quality) (*topo.Face, float64, bool) {
	var nearest *topo.Face
	best := stdmath.Inf(1)
	for _, f := range b.Faces() {
		if t, ok := rayCastMesh(TessellateFace(f, q), origin, dir); ok && t < best {
			best, nearest = t, f
		}
	}
	return nearest, best, nearest != nil
}

// rayCastMesh returns the nearest positive hit distance of a ray against a mesh's
// triangles.
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
