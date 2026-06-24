// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// GeometryProperties is the density-independent mass-properties of a closed body: the
// enclosed volume, the surface area, and the centroid (center of mass for a uniform
// body). Mass = density × Volume is left to the caller, which owns the material density.
// Lengths are in database units, so Volume is units³ and Area units².
type GeometryProperties struct {
	Volume   float64
	Area     float64
	Centroid math.Point3
}

// BodyGeometryProperties computes the volume, area, and centroid of a body from its
// tessellation at quality q. Volume and centroid use the divergence-theorem
// signed-tetrahedron sum over the triangulated surface (exact for planar faces; it
// converges with q for curved ones); area sums the triangle areas. A non-solid or empty
// body yields zero volume.
func BodyGeometryProperties(b *topo.Body, q Quality) GeometryProperties {
	mesh, _ := TessellateBody(b, q)
	return meshGeometryProperties(mesh)
}

// meshGeometryProperties is the pure computation over a triangle mesh, factored out so it
// is testable without a body. Each triangle (a,b,c) forms a tetrahedron with the origin;
// the signed tetra volumes sum to the enclosed volume regardless of where the origin sits
// (the parts outside the body cancel), and the volume-weighted tetra centroids give the
// body centroid. The triangles are first oriented consistently outward TOPOLOGICALLY (shared-
// edge 2-colouring, see consistentOutwardFlips), so the sum is correct even when a tessellator
// does not guarantee globally consistent winding across faces, and — unlike the old shading-
// normal test — without spurious per-triangle flips at saddles/silhouette slivers
// (Oblikovati/Oblikovati#1318). It stays translation-invariant (see TestVolumeIsTranslationInvariant).
func meshGeometryProperties(mesh *Mesh) GeometryProperties {
	flips := consistentOutwardFlips(mesh)
	origin := math.P3(0, 0, 0)
	var vol, area, cx, cy, cz float64
	for ti, n := 0, mesh.TriangleCount(); ti < n; ti++ {
		a, b, c := triVerts(mesh, ti)
		if flips[ti] {
			b, c = c, b
		}
		sv := float64(origin.VectorTo(a).Dot(origin.VectorTo(b).Cross(origin.VectorTo(c)))) / 6
		vol += sv
		cx += sv * float64(a.X+b.X+c.X) / 4
		cy += sv * float64(a.Y+b.Y+c.Y) / 4
		cz += sv * float64(a.Z+b.Z+c.Z) / 4
		area += float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length()) / 2
	}
	centroid := origin
	if vol != 0 {
		centroid = math.P3(math.Scalar(cx/vol), math.Scalar(cy/vol), math.Scalar(cz/vol))
	}
	if vol < 0 {
		vol = -vol // orientation may flip the sign; the enclosed volume is its magnitude
	}
	return GeometryProperties{Volume: vol, Area: area, Centroid: centroid}
}

// outwardRef returns the triangle's reference outward direction — the sum of its three
// vertex normals (the tessellator writes outward shading normals).
func outwardRef(mesh *Mesh, ia, ib, ic int) math.Vector3 {
	return mesh.Normals[ia].Add(mesh.Normals[ib]).Add(mesh.Normals[ic])
}
