// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	stdmath "math"

	"oblikovati/kernel/subd"
	"oblikovati/math"
)

// DefaultWeldTolerance is the grid cell (mm) on which coincident vertices are merged.
// Triangle-soup formats (STL/3MF) repeat each shared vertex per triangle with tiny
// float drift; snapping to this grid reunites them so adjacent triangles share an edge.
const DefaultWeldTolerance = 1e-6

// Weld collapses coincident vertices of a soup onto a tolerance grid and returns a
// subd.Mesh whose faces (triangles) reference the shared vertices — the form
// subd.ToBody welds into a B-rep. A watertight soup welds into a cage whose every edge
// is shared by two faces (⇒ a solid body); an open soup keeps boundary edges (⇒ a
// surface body). Degenerate triangles (two welded corners coincide) are dropped.
//
// Example:
//
//	cage := meshio.Weld(raw, meshio.DefaultWeldTolerance)
//	body := subd.ToBody(cage, "import:stl#0")
func Weld(raw RawMesh, tol float64) subd.Mesh {
	if tol <= 0 {
		tol = DefaultWeldTolerance
	}
	cage := subd.Mesh{}
	index := map[[3]int64]int{}
	for _, tri := range raw.Tris {
		face := weldFace(raw, tri, tol, &cage, index)
		if face != nil {
			cage.Faces = append(cage.Faces, face)
		}
	}
	return cage
}

// weldFace welds a triangle's three corners into cage, returning its shared-vertex loop
// or nil when the triangle collapses to a degenerate (a repeated vertex) after welding.
func weldFace(raw RawMesh, tri [3]int, tol float64, cage *subd.Mesh, index map[[3]int64]int) []int {
	face := make([]int, 0, 3)
	for _, vi := range tri {
		idx := weldVertex(raw.Verts[vi], tol, cage, index)
		if len(face) > 0 && face[len(face)-1] == idx {
			continue // adjacent corners coincide → skip the duplicate
		}
		face = append(face, idx)
	}
	if len(face) >= 3 && face[0] == face[len(face)-1] {
		face = face[:len(face)-1]
	}
	if len(face) < 3 {
		return nil
	}
	return face
}

// weldVertex returns the cage index of p, reusing a coincident vertex on the grid.
func weldVertex(p math.Point3, tol float64, cage *subd.Mesh, index map[[3]int64]int) int {
	k := [3]int64{
		int64(stdmath.Round(p.X / tol)),
		int64(stdmath.Round(p.Y / tol)),
		int64(stdmath.Round(p.Z / tol)),
	}
	if i, ok := index[k]; ok {
		return i
	}
	i := len(cage.Verts)
	cage.Verts = append(cage.Verts, p)
	index[k] = i
	return i
}
