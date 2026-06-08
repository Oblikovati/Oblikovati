// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// trimmedPatchMesh meshes a non-rectangular curved patch via a constrained Delaunay triangulation
// of its boundary loops. The 2D embedding to triangulate in is chosen by patchProjection: the
// surface's own (u,v) for a B-spline (where the trim loops are a simple polygon), or the boundary's
// best-fit plane for an analytic surface (whose (u,v) degenerates at a pole/seam). The CDT is robust
// where boundary-only ear-clipping tears (boundary segments are recovered by edge flips) and exact
// on concave trims and holes (the domain flood respects the constrained edges); it meshes the real
// trim region, not the surface's whole UV domain (which fullDomainGridMesh did — the torn full-sphere
// fan). No interior Steiner points: ParamAt's distorted (u,v) makes a freshly sampled interior
// point's PointAt land off the patch and inflate the mesh, so refinement stays boundary-only and the
// exact 3D boundary points are kept (watertight with neighbour faces). Falls back to
// boundaryPatchMesh if the CDT yields nothing.
func trimmedPatchMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) *Mesh {
	outer2D, holes2D := patchProjection(s, outer3D, holes3D)
	uv, pos, nrm, loops := patchLoops2D(s, outer3D, holes3D, outer2D, holes2D)
	tris := constrainedDelaunay(uv, loops)
	if len(tris) == 0 {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	return patchMeshFrom(pos, nrm, tris)
}

// patchLoops2D pairs each boundary loop's chosen 2D embedding (outer2D/holes2D) with its exact 3D
// point and surface normal, returning the (u,v/plane) coords, parallel positions and normals, and
// the loops as index sequences for the CDT.
func patchLoops2D(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3, outer2D []math.Point2, holes2D [][]math.Point2) (uv [][2]float64, pos []math.Point3, nrm []math.Vector3, loops [][]int) {
	loops3D := append([][]math.Point3{outer3D}, holes3D...)
	loops2D := append([][]math.Point2{outer2D}, holes2D...)
	for li, loop := range loops3D {
		idx := make([]int, len(loop))
		for i, p := range loop {
			u, v := s.ParamAt(p)
			idx[i] = len(uv)
			uv = append(uv, [2]float64{float64(loops2D[li][i].X), float64(loops2D[li][i].Y)})
			pos = append(pos, p)
			nrm = append(nrm, s.NormalAt(u, v))
		}
		loops = append(loops, idx)
	}
	return uv, pos, nrm, loops
}

// patchMeshFrom builds the 3D mesh from the CDT triangles, winding each to agree with its own
// vertex normals (consistent on a curved patch — see windingOpposesNormals).
func patchMeshFrom(pos []math.Point3, nrm []math.Vector3, tris [][3]int) *Mesh {
	m := &Mesh{}
	for i := range pos {
		m.addVertex(pos[i], nrm[i])
	}
	for _, t := range tris {
		a, b, c := t[0], t[1], t[2]
		if windingOpposesNormals(pos[a], pos[b], pos[c], nrm[a], nrm[b], nrm[c]) {
			b, c = c, b
		}
		m.addTriangle(a, b, c)
	}
	return m
}
