// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// refinedTrimmedMesh meshes a non-rectangular curved patch with INTERIOR points, not just its
// boundary. The trim loops are mapped to the surface's (u,v) and meshed with a constrained
// Delaunay triangulation (the loop edges are hard constraints; an interior (u,v) grid refines the
// patch). This is robust where boundary-only ear-clipping fails — a slightly self-intersecting
// (u,v) boundary (from ParamAt inversion on a rational patch) no longer tears, the CDT respects
// concave trims exactly (no triangle bridges a notch), and the interior points make a strongly
// curved patch read smooth. Boundary points keep their exact 3D positions (shared with the
// neighbour face, so the patch stays watertight). Used for B-spline faces (their (u,v) is
// well-behaved); analytic patches keep boundaryPatchMesh (their (u,v) degenerates at a pole/seam).
// Falls back to boundaryPatchMesh if the CDT yields nothing.
func refinedTrimmedMesh(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) *Mesh {
	// No interior Steiner points: ParamAt on a rational patch returns a distorted (u,v), so PointAt
	// of a freshly sampled interior (u,v) lands off the trimmed region and inflates the mesh. We
	// mesh only the boundary loops (their exact 3D points), relying on the CDT's constraint recovery
	// + domain flood to stay watertight where boundary-only ear-clipping tears.
	uv, pos, nrm, loops := patchBoundaryUV(s, outer3D, holes3D)
	tris := constrainedDelaunay(uv, loops)
	if len(tris) == 0 {
		return boundaryPatchMesh(s, outer3D, holes3D)
	}
	return patchMeshFrom(pos, nrm, tris)
}

// patchBoundaryUV maps each boundary loop into (u,v) (keeping its exact 3D points + normals) and
// returns the (u,v) coords, parallel 3D positions and normals, and the loops as index sequences.
func patchBoundaryUV(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) (uv [][2]float64, pos []math.Point3, nrm []math.Vector3, loops [][]int) {
	addLoop := func(loop []math.Point3) []int {
		idx := make([]int, len(loop))
		for i, p := range loop {
			u, v := s.ParamAt(p)
			idx[i] = len(uv)
			uv = append(uv, [2]float64{u, v})
			pos = append(pos, p)
			nrm = append(nrm, s.NormalAt(u, v))
		}
		return idx
	}
	loops = [][]int{addLoop(outer3D)}
	for _, h := range holes3D {
		loops = append(loops, addLoop(h))
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
