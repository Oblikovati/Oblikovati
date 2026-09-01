// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/ops/tessellate"
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

// BodyGeometryProperties computes the volume, area, and centroid of a body by integrating its
// ANALYTIC B-rep face by face (AnalyticGeometryProperties, M48/C3 #3453). This result gates the
// boolean acceptance bracket and the identical-body signature, and "an oracle that gates a result
// must be more exact than the result it gates" — a tessellated integral is not: an inscribed N-gon
// under-measures every curved face by a systematic −π²/(3N²) (Oblikovati/Oblikovati#3485).
//
// q parameterises only the FALLBACK for a body the analytic path declines — a non-solid or empty
// body, or a face whose uv boundary cannot be reconstructed — which integrates the tessellation at
// that quality. A non-solid or empty body yields zero volume either way.
//
// Example: gp := ops.BodyGeometryProperties(cyl, ops.PropertyQuality()) // gp.Volume == πr²h
func BodyGeometryProperties(b *topo.Body, q Quality) GeometryProperties {
	if p, ok := AnalyticGeometryProperties(b); ok {
		return p
	}
	mesh, _ := tessellate.TessellateBody(b, q)
	return MeshGeometryProperties(mesh)
}

// MeshGeometryProperties is the tessellated integral: no longer the kernel's mass-properties
// oracle (M48/C3 #3454), but the named fallback BodyGeometryProperties takes for a body the
// analytic path declines, and the display statistic for a mesh that has no B-rep behind it at all
// (an imported soup). It is a pure computation over a triangle mesh, factored out so it is testable
// without a body. Each triangle (a,b,c) forms a tetrahedron with the origin;
// the signed tetra volumes sum to the enclosed volume regardless of where the origin sits
// (the parts outside the body cancel), and the volume-weighted tetra centroids give the
// body centroid. The triangles are first oriented consistently outward TOPOLOGICALLY (shared-
// edge 2-colouring, see tessellate.ConsistentOutwardFlips), so the sum is correct even when a tessellator
// does not guarantee globally consistent winding across faces, and — unlike the old shading-
// normal test — without spurious per-triangle flips at saddles/silhouette slivers
// (Oblikovati/Oblikovati#1318). It stays translation-invariant (see TestVolumeIsTranslationInvariant).
func MeshGeometryProperties(mesh *Mesh) GeometryProperties {
	flips := tessellate.ConsistentOutwardFlips(mesh)
	origin := math.P3(0, 0, 0)
	var vol, area, cx, cy, cz float64
	for ti, n := 0, mesh.TriangleCount(); ti < n; ti++ {
		a, b, c := tessellate.TriVerts(mesh, ti)
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
