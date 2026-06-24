// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// InertiaTensor is the mass-moment-of-inertia tensor of a closed body about its centroid, per unit
// density (so the caller multiplies by density to get the mass inertia). The off-diagonal products
// follow the convention Ixy = -∫xy dV. Lengths are database units, so each component is units⁵.
type InertiaTensor struct {
	Ixx, Iyy, Izz float64
	Ixy, Iyz, Izx float64
}

// BodyInertia computes the body's inertia tensor about its centroid (per unit density) from its
// tessellation at quality q. It uses the signed-tetrahedron covariance: each outward-wound triangle
// (a,b,c) forms the tetra (0,a,b,c), whose second-moment (covariance) integral is the canonical
// tetra covariance carried through the affine map [a b c]; the signed sum gives the body's
// covariance about the origin, which is reduced to the inertia tensor and shifted to the centroid.
func BodyInertia(b *topo.Body, q Quality) InertiaTensor {
	mesh, _ := TessellateBody(b, q)
	return meshInertia(mesh)
}

// canonTetraCovariance is ∫ q qᵀ dV over the reference tetra (0,e1,e2,e3) — a known constant.
var canonTetraCovariance = mat3{
	{1.0 / 60, 1.0 / 120, 1.0 / 120},
	{1.0 / 120, 1.0 / 60, 1.0 / 120},
	{1.0 / 120, 1.0 / 120, 1.0 / 60},
}

// meshInertia is the pure inertia computation over a triangle mesh, factored out for testing.
func meshInertia(mesh *Mesh) InertiaTensor {
	cov, vol, cx, cy, cz := accumulateCovariance(mesh)
	if vol < 0 { // outward winding should make vol positive; guard a flipped mesh
		vol, cov = -vol, scale3(cov, -1)
		cx, cy, cz = -cx, -cy, -cz
	}
	if vol == 0 {
		return InertiaTensor{}
	}
	d := math.P3(math.Scalar(cx/vol), math.Scalar(cy/vol), math.Scalar(cz/vol))
	return inertiaFromCovariance(cov, vol, d)
}

// accumulateCovariance sums each outward-wound triangle's tetra covariance (∫ p pᵀ dV about the
// origin) and signed volume / centroid numerator over the mesh. Triangles are oriented consistently
// outward TOPOLOGICALLY (consistentOutwardFlips), not from shading normals, so a saddle/silhouette
// facet can no longer flip its covariance sign at random (Oblikovati/Oblikovati#1318).
func accumulateCovariance(mesh *Mesh) (cov mat3, vol, cx, cy, cz float64) {
	flips := consistentOutwardFlips(mesh)
	for ti, n := 0, mesh.TriangleCount(); ti < n; ti++ {
		a, b, c := triVerts(mesh, ti)
		if flips[ti] {
			b, c = c, b
		}
		// A = [a b c] as columns; det A = a · (b × c) = 6 × signed tetra volume.
		col := mat3{
			{float64(a.X), float64(b.X), float64(c.X)},
			{float64(a.Y), float64(b.Y), float64(c.Y)},
			{float64(a.Z), float64(b.Z), float64(c.Z)},
		}
		det := det3([3][3]float64(col))
		// ∫ p pᵀ dV over this tetra = det · A · Ccanon · Aᵀ.
		contrib := mul3(mul3(col, canonTetraCovariance), transpose3(col))
		cov = add3(cov, scale3(contrib, det))
		sv := det / 6
		vol += sv
		cx += sv * float64(a.X+b.X+c.X) / 4
		cy += sv * float64(a.Y+b.Y+c.Y) / 4
		cz += sv * float64(a.Z+b.Z+c.Z) / 4
	}
	return cov, vol, cx, cy, cz
}

// inertiaFromCovariance reduces the origin covariance C to the inertia tensor about the centroid:
// I_origin = tr(C)·Id − C, then the parallel-axis shift subtracts the inertia of a point mass V at
// the centroid d.
func inertiaFromCovariance(c mat3, vol float64, d math.Point3) InertiaTensor {
	tr := c[0][0] + c[1][1] + c[2][2]
	dx, dy, dz := float64(d.X), float64(d.Y), float64(d.Z)
	d2 := dx*dx + dy*dy + dz*dz
	return InertiaTensor{
		Ixx: (tr - c[0][0]) - vol*(d2-dx*dx),
		Iyy: (tr - c[1][1]) - vol*(d2-dy*dy),
		Izz: (tr - c[2][2]) - vol*(d2-dz*dz),
		Ixy: -c[0][1] - vol*(-dx*dy),
		Iyz: -c[1][2] - vol*(-dy*dz),
		Izx: -c[2][0] - vol*(-dz*dx),
	}
}

// mat3 is a 3×3 matrix indexed [row][col]. (det3 for [3][3]float64 lives in retopo.go.)
type mat3 [3][3]float64

func transpose3(m mat3) mat3 {
	return mat3{{m[0][0], m[1][0], m[2][0]}, {m[0][1], m[1][1], m[2][1]}, {m[0][2], m[1][2], m[2][2]}}
}

func mul3(a, b mat3) mat3 {
	var r mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i][j] = a[i][0]*b[0][j] + a[i][1]*b[1][j] + a[i][2]*b[2][j]
		}
	}
	return r
}

func add3(a, b mat3) mat3 {
	var r mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i][j] = a[i][j] + b[i][j]
		}
	}
	return r
}

func scale3(a mat3, s float64) mat3 {
	var r mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i][j] = a[i][j] * s
		}
	}
	return r
}
