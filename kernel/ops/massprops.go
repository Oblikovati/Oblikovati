// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati/kernel/topo"
	"oblikovati/math"
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
// body centroid. Each triangle is first oriented outward using the mesh's per-vertex
// normals, so the sum is correct even when a tessellator does not guarantee globally
// consistent winding across faces (without this the divergence sum is not
// translation-invariant — see TestVolumeIsTranslationInvariant).
func meshGeometryProperties(mesh *Mesh) GeometryProperties {
	origin := math.P3(0, 0, 0)
	var vol, area, cx, cy, cz float64
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		ia, ib, ic := mesh.Indices[t], mesh.Indices[t+1], mesh.Indices[t+2]
		a, b, c := mesh.Positions[ia], mesh.Positions[ib], mesh.Positions[ic]
		// Force outward winding: if the geometric normal opposes the (outward) shading
		// normal, swap two vertices.
		if outwardRef(mesh, ia, ib, ic).Dot(a.VectorTo(b).Cross(a.VectorTo(c))) < 0 {
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
