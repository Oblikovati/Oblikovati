// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// RayCastFaces returns the nearest face of a body hit by a ray, with the hit
// distance — the geometric core of viewport picking (the renderer's ID-buffer is an
// optimization of the same query). The closest hit wins. Pure Go, so the UI's
// hit-test is headless-testable.
func RayCastFaces(b *topo.Body, origin math.Point3, dir math.Vector3, q Quality) (*topo.Face, float64, bool) {
	var nearest *topo.Face
	best := stdmath.Inf(1)
	for _, f := range b.Faces() {
		if t, ok := rayCastFace(f, origin, dir, q); ok && t < best {
			best, nearest = t, f
		}
	}
	return nearest, best, nearest != nil
}

// rayCastFace returns the ray's forward hit distance to a face. A PLANAR face is hit-tested by
// ray–plane intersection plus a point-in-polygon test on its boundary loops — O(m), no
// triangulation — so the synchronous per-frame hover pick never triangulates a planar face. That
// matters because tessellatePlanarFace runs the exact-predicate ear-clipper, which on a degenerate
// (near-collinear) boundary escalates to O(m³) big.Rat arithmetic (seconds on a ~66-vertex face);
// re-run every frame by the viewport's hovered-plane pick, it starved the frame-loop dispatcher an
// async add-in build blocks on → a hard placement deadlock. A curved face still ray-tests its
// tessellation (a bounded UV grid — no ear-clipping).
func rayCastFace(f *topo.Face, origin math.Point3, dir math.Vector3, q Quality) (float64, bool) {
	if _, ok := f.Geometry().(geom.Plane); ok {
		return rayCastPlanarFace(f, origin, dir, q)
	}
	return rayCastMesh(TessellateFace(f, q), origin, dir)
}

// rayCastPlanarFace intersects the ray with the face's plane, then tests the hit point against the
// face's boundary loops (inside the outer loop, outside every hole) — the same coverage a
// triangulation of the face would give, since the boundary is discretized identically (loopBoundary),
// but in O(m) with no ear-clipping.
func rayCastPlanarFace(f *topo.Face, origin math.Point3, dir math.Vector3, q Quality) (float64, bool) {
	outer3D := faceOuterBoundary(f, q)
	if len(outer3D) < 3 {
		return 0, false
	}
	normal := f.Geometry().NormalAt(0, 0)
	denom := float64(dir.Dot(normal))
	if stdmath.Abs(denom) < 1e-12 {
		return 0, false // ray parallel to the face plane
	}
	t := float64(origin.VectorTo(outer3D[0]).Dot(normal)) / denom
	if t <= 0 {
		return 0, false // behind the ray origin
	}
	hit := origin.TranslateBy(dir.Scale(math.Scalar(t)))
	flat := planeProjector(normal)
	p := flat(hit)
	if !pointInLoop2D(p, project2D(outer3D, flat)) {
		return 0, false
	}
	for _, h := range faceHoleBoundaries(f, q) {
		if pointInLoop2D(p, project2D(h, flat)) {
			return 0, false // the ray passes through a hole
		}
	}
	return t, true
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
