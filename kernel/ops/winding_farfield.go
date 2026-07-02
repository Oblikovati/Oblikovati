// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Provably exact acceleration of repeated winding-number queries against one mesh (#1607).
//
// The generalized winding number needs EVERY triangle's solid angle, so a plain BVH prune is
// WRONG — dropping far triangles changes the sum — and the Barill et al. 2018 hierarchical
// dipole approximation is uncertified: its error, however small, can flip a classification for
// a query near the w≈0.5 iso-surface, which in boolean classification is exactly a point ON
// the other solid's boundary. A certified hierarchical variant was tried and rejected: the
// only RIGOROUS per-subtree cap, |ΣΩ| ≤ area/d² (a patch of area A projected onto the unit
// sphere from distance ≥ d shrinks by ≥ 1/d²), cannot see the dipole cancellation that makes
// far patches negligible, so interior queries essentially never certify and the traversal just
// adds overhead. The exact per-triangle loop therefore stays the classifier.
//
// What IS provably exact — and cached here — is the far-field early-out: from distance d
// outside the mesh's bounding box the winding number can never exceed totalArea/(4π·d²), so a
// query far enough away is certified outside without visiting a single triangle.
type meshWindingFarField struct {
	mesh      *Mesh
	box       math.Box
	totalArea float64
}

// newMeshWindingFarField caches the mesh's bounding box and total triangle area — one O(T)
// pass, about the cost of a single winding query.
func newMeshWindingFarField(mesh *Mesh) *meshWindingFarField {
	f := &meshWindingFarField{mesh: mesh, box: math.EmptyBox()}
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		t := meshTriangle(mesh, i)
		f.box = f.box.ExtendPoint(t[0]).ExtendPoint(t[1]).ExtendPoint(t[2])
		f.totalArea += 0.5 * float64(t[0].VectorTo(t[1]).Cross(t[0].VectorTo(t[2])).Length())
	}
	return f
}

// inside classifies p with the same result as pointInMesh: the certified far-field early-out
// when it applies, the exact brute loop otherwise.
func (f *meshWindingFarField) inside(p math.Point3) bool {
	if f.certifiedOutside(p) {
		return false
	}
	return pointInMesh(f.mesh, p)
}

// windingFarFieldMaxW is the winding-number ceiling under which the far-field early-out fires:
// 2× below the 0.5 inside cutoff, so the exact value (≤ totalArea/(4π·d²) ≤ 0.25) plus any
// float64 summation error of the brute loop (≲ n·eps·|w|, astronomically below 0.25) can never
// reach the cutoff — the early-out returns exactly what the brute loop would.
const windingFarFieldMaxW = 0.25 // tol:numeric — dimensionless winding-number ceiling (certified; see above)

// certifiedOutside reports whether p is provably outside: strictly beyond the mesh box, with
// the rigorous winding cap totalArea/(4π·d²) at or below [windingFarFieldMaxW].
func (f *meshWindingFarField) certifiedOutside(p math.Point3) bool {
	d2 := pointBoxDistanceSq(p, f.box)
	return d2 > 0 && f.totalArea/d2 <= windingFarFieldMaxW*(4*stdmath.Pi)
}

// pointBoxDistanceSq is the squared Euclidean distance from p to the closed box (0 inside).
func pointBoxDistanceSq(p math.Point3, b math.Box) float64 {
	dx := axisGap(float64(p.X), float64(b.Min.X), float64(b.Max.X))
	dy := axisGap(float64(p.Y), float64(b.Min.Y), float64(b.Max.Y))
	dz := axisGap(float64(p.Z), float64(b.Min.Z), float64(b.Max.Z))
	return dx*dx + dy*dy + dz*dz
}

// axisGap is the 1D distance from v to the interval [lo, hi] (0 inside).
func axisGap(v, lo, hi float64) float64 {
	if v < lo {
		return lo - v
	}
	if v > hi {
		return v - hi
	}
	return 0
}

// windingCacheMinQueries gates the far-field cache to workloads that amortize its O(T) build
// (a count, not a length): a single query is cheaper through the brute loop alone.
const windingCacheMinQueries = 2

// insideMeshQuerier returns the point-in-mesh classifier for `queries` upcoming queries
// against one mesh: the far-field-cached classifier when the workload amortizes the cache
// build, else the plain brute loop. Both classify identically (the early-out is certified —
// see the meshWindingFarField type comment).
//
// Example: inMesh := insideMeshQuerier(mesh, len(verts)); for _, v := range verts { inMesh(v.Point()) }
func insideMeshQuerier(mesh *Mesh, queries int) func(math.Point3) bool {
	if queries < windingCacheMinQueries {
		return func(p math.Point3) bool { return pointInMesh(mesh, p) }
	}
	return newMeshWindingFarField(mesh).inside
}
