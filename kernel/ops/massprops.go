// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

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

// outwardRef returns the triangle's reference outward direction — the sum of its three
// vertex normals (the tessellator writes outward shading normals).
func outwardRef(mesh *Mesh, ia, ib, ic int) math.Vector3 {
	return mesh.Normals[ia].Add(mesh.Normals[ib]).Add(mesh.Normals[ic])
}
